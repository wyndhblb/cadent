/*
Copyright 2014-2017 Bo Blanton

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
   Zookeeper "Discovery" module
*/

package discovery

import (
	sapi "cadent/server/schemas/api"
	"cadent/server/utils"
	"cadent/server/utils/options"
	"cadent/server/writers/api"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"gopkg.in/op/go-logging.v1"
	"net/url"
	"os"
	"strings"
	"time"
)

const ZK_API_PREFIX = "/apihosts"
const ZK_ROOT_PREFIX = "/cadent"

var ErrNoZookeeperConn = errors.New("No Zookeeper connection estabilished")

type ZKlogger struct {
	log *logging.Logger
}

func (l ZKlogger) Printf(s string, args ...interface{}) {
	if strings.Contains(strings.ToLower(s), "failed") || strings.Contains(strings.ToLower(s), "error") {
		l.log.Errorf(s, args...)
	} else {
		l.log.Info(s, args...)
	}
}

// Discover interface
type Zookeeper struct {
	conf           options.Options
	zkHosts        []string // zk hosts
	basePath       string   // root bath
	apiHostsPrefix string
	name           string   // key path name
	conn           *zk.Conn // the actuall conn
	doRegister     bool
	starstop       utils.StartStop
	log            *logging.Logger
	zkEvt          <-chan zk.Event
}

func NewZookeeper() *Zookeeper {
	z := new(Zookeeper)
	z.log = logging.MustGetLogger("discover.zookeeper")
	return z
}

func (z *Zookeeper) Config(conf options.Options) error {
	z.conf = conf

	hst, err := conf.StringRequired("dsn")
	if err != nil {
		return err
	}

	z.doRegister = conf.Bool("register", true)

	pth := conf.String("root_path", ZK_ROOT_PREFIX)

	if !strings.HasPrefix(pth, "/") {
		pth = "/" + pth
	}
	z.basePath = pth

	z.apiHostsPrefix = conf.String("api_path", ZK_API_PREFIX)

	defHost, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("Could not dermine the Hostname for this host: %v", err)
	}
	name := conf.String("name", defHost)
	if len(name) == 0 {
		return fmt.Errorf("Could not dermine the name for this host: %v", err)
	}
	z.name = name

	z.zkHosts = strings.Split(hst, ",")

	return nil
}

// PathPrefix {root_path}/{api_path}/{name}
func (z *Zookeeper) Path(nm string) string {
	return strings.Replace(fmt.Sprintf("%s/%s/%s", z.basePath, z.apiHostsPrefix, nm), "//", "/", -1)
}

// PathPrefix {root_path}/{api_path}/{hostname}
func (z *Zookeeper) PathPrefix() string {
	return z.Path(z.name)
}

// BasePathPrefix {root_path}/{api_path}
func (z *Zookeeper) BasePathPrefix() string {
	return strings.Replace(fmt.Sprintf("%s/%s", z.basePath, z.apiHostsPrefix), "//", "/", -1)
}

// Start fire up zookeeper connections
func (z *Zookeeper) Start() error {
	var err error
	z.starstop.Start(func() {
		z.log.Noticef("Starting Zoookeeper API Discover on %s%s", z.zkHosts, z.BasePathPrefix())
		var terr error
		z.conn, z.zkEvt, terr = z.makeConnection()
		if terr != nil {
			err = terr
			return
		}
	})

	return err

}

func (z *Zookeeper) makeConnection() (*zk.Conn, <-chan zk.Event, error) {

	zkLog := ZKlogger{log: z.log}
	setLogger := func(c *zk.Conn) {
		c.SetLogger(zkLog)
	}

	conn, evt, err := zk.Connect(z.zkHosts, 20*time.Second, setLogger)
	if err != nil {
		return nil, nil, err
	}
	if evt != nil {
		go func() {
			select {
			case item, more := <-evt:
				if !more {
					return
				}
				z.log.Info("ZK event: State: %v Error: %v", item.State, item.Err)
			}
		}()
	}
	return conn, evt, nil
}

// Stop remove ourselves from the registry and quit
func (z *Zookeeper) Stop() {
	z.starstop.Stop(func() {
		z.log.Noticef("Stopping/Deregistering Zoookeeper API Discover on %s%s", z.zkHosts, z.BasePathPrefix())
		if z.conn != nil {
			z.DeRegister()
			z.conn.Close()
		}
	})
}

// Register add ourselves from the registry
func (z *Zookeeper) Register(info *api.InfoData) error {

	if !z.doRegister {
		z.log.Noticef("Register is set to false, not register node in discovery")
		return nil
	}

	if z.conn == nil {
		return ErrNoZookeeperConn
	}

	// need to set the root path if not there
	ext, _, err := z.conn.Exists(z.basePath)
	if err != nil {
		z.log.Critical("Error exists path: %s: %v", z.basePath, err)
		return err
	}
	nFlag := int32(0)
	if !ext {
		d, err := z.conn.Create(z.basePath, []byte(""), nFlag, zk.WorldACL(zk.PermAll))
		fmt.Println(ext, err, d)
		if err != nil {
			z.log.Critical("Error creating path: %s: %v", z.basePath, err)
			return err
		}
	}

	// need to set the root path if not there
	ext, _, err = z.conn.Exists(z.BasePathPrefix())
	if err != nil {
		z.log.Critical("Error exists path: %s: %v", z.BasePathPrefix(), err)
		return err
	}
	if !ext {
		d, err := z.conn.Create(z.BasePathPrefix(), []byte(""), nFlag, zk.WorldACL(zk.PermAll))
		fmt.Println(ext, err, d)
		if err != nil {
			z.log.Critical("Error creating path: %s: %v", z.BasePathPrefix(), err)
			return err
		}
	}

	flags := int32(zk.FlagEphemeral) // Ephemeral node, want it dead if we die
	acl := zk.WorldACL(zk.PermAll)

	bits, err := json.Marshal(info)
	if err != nil {
		z.log.Critical("Error creating ephemeral key (json encode error): %s: %v", z.PathPrefix(), err)
		return err
	}

	_, err = z.conn.Create(z.PathPrefix(), bits, flags, acl)
	if err != nil {
		z.log.Critical("Error creating ephemeral: %s: %v", z.PathPrefix(), err)
	}
	return err
}

// DeRegister remove ourselves from the registry
func (z *Zookeeper) DeRegister() error {
	if !z.doRegister {
		return nil
	}
	if z.conn == nil {
		return ErrNoZookeeperConn
	}

	err := z.conn.Delete(z.PathPrefix(), 0)
	return err
}

/******************************************************/
/** lister object **/

// ZookeeperLister Lister interface, grab all active hosts
type ZookeeperLister struct {
	Zookeeper
	currentList *sapi.DiscoverHosts
	evtC        chan zk.Event
}

func NewZookeeperLister() *ZookeeperLister {
	z := new(ZookeeperLister)
	z.Zookeeper = *NewZookeeper()
	return z
}

// Start start the watch on the path prefix, and grab any initial data
func (z *ZookeeperLister) Start() (err error) {

	z.starstop.Start(func() {
		if z.conn == nil {
			z.conn, z.zkEvt, err = z.makeConnection()

			if err != nil {
				return
			}
		}
		err = z.setWatch()
	})
	return err
}

// Stop disconnect and quit
func (z *ZookeeperLister) Stop() {
	z.starstop.Stop(func() {
		if z.conn != nil {
			z.conn.Close()
			z.conn = nil
		}
	})
}

func (z *ZookeeperLister) ForcePullHosts() (data *sapi.DiscoverHosts, err error) {
	if z.conn == nil {
		z.conn, z.zkEvt, err = z.makeConnection()
		if err != nil {
			z.log.Errorf("Could not get ZK connections: %v", err)
			return nil, err
		}
	}

	err = z.doPullHosts()
	if err != nil {
		return nil, err
	}
	return z.currentList, nil
}

func (z *ZookeeperLister) doPullHosts() error {

	if z.conn == nil {
		return ErrNoZookeeperConn
	}

	path := z.BasePathPrefix()

	exists, _, err := z.conn.Exists(path)
	if err != nil {
		return err
	}
	if !exists {
		return nil // nothing to do
	}
	children, _, err := z.conn.Children(path)
	if err != nil {
		return err
	}
	return z.grabHostData(children)
}

func (z *ZookeeperLister) grabHostData(children []string) error {

	nlist := new(sapi.DiscoverHosts)
	for _, c := range children {
		pth := z.Path(c)
		data, _, err := z.conn.Get(pth)
		if err != nil {
			z.log.Errorf("Could not get data for node %s: %v", c, err)
			continue
		}

		var tInfo api.InfoData
		err = json.Unmarshal(data, &tInfo)
		if err != nil {
			z.log.Errorf("Json decode error: Could not get data for node %s: %v", c, err)
			continue
		}

		u := fmt.Sprintf("%s://%s:%s%s", tInfo.Api.Scheme, tInfo.Api.Host, tInfo.Api.Port, tInfo.Api.BasePath)
		_, err = url.Parse(u)
		if err != nil {
			z.log.Errorf("Inalid URL: decode error: Could not get data for node %s: %v", c, err)
			continue
		}
		isapi := false
		istcp := false
		iswrite := false
		isread := false
		ishash := false
		isgrpc := false

		if shared, ok := tInfo.SharedData.(map[string]interface{}); ok {

			if g, ok := shared["is_api"]; ok {
				isapi = g.(bool)
			}
			if g, ok := shared["is_tcpapi"]; ok {
				istcp = g.(bool)
			}
			if g, ok := shared["is_writer"]; ok {
				iswrite = g.(bool)
			}
			if g, ok := shared["is_reader"]; ok {
				isread = g.(bool)
			}
			if g, ok := shared["is_hasher"]; ok {
				ishash = g.(bool)
			}
			if g, ok := shared["is_grpc"]; ok {
				isgrpc = g.(bool)
			}
		}

		res := make([]*sapi.Resolution, 0)
		for _, r := range tInfo.Resolutions {
			res = append(res, &sapi.Resolution{Resolution: uint32(r[0]), Ttl: uint32(r[1])})
		}

		l := &sapi.DiscoverHost{
			AdvertiseName: tInfo.AdvertiseName,
			AdvertiseUrl:  tInfo.AdvertiseUrl,
			Grpchost:      tInfo.GRPCHost,
			Host:          tInfo.Api.Host,
			HostApiUrl:    u,
			Resolutions:   res,
			StartTime:     tInfo.Time,
			IsApi:         isapi,
			IsTCPapi:      istcp,
			IsWriter:      iswrite,
			IsReader:      isread,
			IsHasher:      ishash,
			IsgRPC:        isgrpc,
			Tags:          tInfo.Tags,
		}
		nlist.Hosts = append(nlist.Hosts, l)
	}
	z.log.Info("Got %d registered api nodes", len(nlist.Hosts))
	z.currentList = nlist
	return nil
}

func (z *ZookeeperLister) ApiHosts() *sapi.DiscoverHosts {
	return z.currentList
}

func (z *ZookeeperLister) setWatch() error {

	if z.conn == nil {
		return ErrNoZookeeperConn
	}

	path := z.BasePathPrefix()

	// do it the first time
	z.doPullHosts()

	// do it every min just in case of weird socket things
	go func() {
		t := time.NewTicker(time.Minute)
		var err error

		for {
			<-t.C
			if z.conn == nil {
				z.conn, z.zkEvt, err = z.makeConnection()
				if err != nil {
					z.log.Errorf("Could not get ZK connections: %v", err)
					continue
				}
			}
			z.doPullHosts()
		}

	}()

	go func() {
		for {
			//[]string, *Stat, <-chan Event, error
			current, _, evtC, err := z.conn.ChildrenW(path)
			z.log.Info("Got children update")
			if err != nil {
				z.log.Errorf("Watch error: %v", err)
				time.Sleep(10 * time.Second) // pause some
			}
			z.log.Debugf("Got watch change: %+v", current)
			z.grabHostData(current)

			n := <-evtC
			if n.Err != nil {
				z.log.Errorf("Event Watch error: %v", n.Err)
				time.Sleep(10 * time.Second) // pause some
				continue
			}

		}
	}()
	return nil
}
