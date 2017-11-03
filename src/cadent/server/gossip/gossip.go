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
   Simple gossip protocol to find the other Cadent's out there
*/

package gossip

import (
	"bytes"
	"cadent/server/broadcast"
	"encoding/json"
	"fmt"
	"github.com/DataDog/zstd"
	"github.com/cenkalti/backoff"
	"github.com/hashicorp/memberlist"
	logging "gopkg.in/op/go-logging.v1"
	"strings"
	"sync"
	"time"
)

var glog = logging.MustGetLogger("gossip")

//singleton
var CadentMembers *Members
var enabled bool

func Get() *Members {
	return CadentMembers
}

func Enabled() bool {
	return enabled
}

// MemberInfo a Member's metadata .. note there is a 512byte limit on this
// so it needs to be compressed in flight
type MemberInfo struct {
	GossipAddr  string  `json:"gossip-addr"` //gossip addr
	GRPCaddr    string  `json:"grpc-addr"`   // grpc address
	HttpApiUrl  string  `json:"http-addr"`   // http address
	Resolutions [][]int `json:"resolutions"` // resolutions served
	Name        string  `json:"name"`        // name
	Cluster     string  `json:"cluster"`     // cluster name: (when there are multiple api hosts on a single cadent)
	IsWriter    bool    `json:"write"`       // is this cadent writing
	IsReader    bool    `json:"read"`        // is this cadent a reader
	IsApi       bool    `json:"api"`         // is the http api running
	IsTcpApi    bool    `json:"tcpapi"`      // is the tcp api running
	IsGrpc      bool    `json:"grcp"`        // is the grpc api running
}

// Member a member in the gossip ring
type Member struct {
	Name       string                `json:"name"`
	Info       map[string]MemberInfo `json:"info"`    // can have multiple API targets for a given cadent, so we map things to cluster names
	LastUpdate time.Time             `json:"updated"` // last time we pinged
}

// Members main gossip controller
type Members struct {
	List      *memberlist.Memberlist
	Name      string
	Mode      string
	Port      int
	Bind      string
	Advertise string
	Seed      string
	Conf      *memberlist.Config
	Nodes     map[string]*Member
	Meta      map[string]MemberInfo
	LocalInfo *Member

	NodeUpdated *broadcast.Broadcaster // on new, updated, rm'ed fire out an event to whom ever is listening
	NodeLeft    *broadcast.Broadcaster // on new, updated, rm'ed fire out an event to whom ever is listening
	nLock       sync.RWMutex
	joined      bool
}

// Start fire up a new gossiper
func Start(mode string, port int, name string, bind string, advertise string) (ms *Members, err error) {
	li := new(Members)
	li.Name = name
	li.Mode = mode
	li.Port = port
	li.Bind = bind
	li.NodeUpdated = broadcast.New(1)
	li.NodeLeft = broadcast.New(1)

	li.Nodes = make(map[string]*Member)
	li.Advertise = advertise
	li.Conf = li.Config()
	li.Conf.Events = li
	li.Conf.Delegate = li

	enabled = true
	membs, err := memberlist.Create(li.Conf)
	if err != nil {
		glog.Errorf("error in memberlist: %v -- %v", err, li.Conf)
		return nil, err
	}
	li.List = membs

	if CadentMembers == nil {
		CadentMembers = li
	}
	return li, err
}

func Stop() {
	if CadentMembers != nil {
		glog.Warning("Stopping gossip and leaving")
		CadentMembers.List.Leave(time.Second)
		CadentMembers.NodeUpdated.Close()
		CadentMembers.NodeLeft.Close()
	}
}

func (m *Members) Config() *memberlist.Config {
	var conf *memberlist.Config
	switch m.Mode {
	case "local":
		conf = memberlist.DefaultLocalConfig()
	case "lan":
		conf = memberlist.DefaultLANConfig()
	case "wan":
		conf = memberlist.DefaultWANConfig()
	default:
		panic("Gossip mode can be only local, lan, or wan")
	}

	if len(m.Bind) > 0 {
		conf.BindAddr = m.Bind
	} else {
		conf.BindAddr = "0.0.0.0"
	}

	if len(m.Advertise) > 0 {
		conf.AdvertiseAddr = m.Advertise
	}

	conf.AdvertisePort = m.Port
	conf.BindPort = m.Port

	if m.Name != "" {
		conf.Name = m.Name
	} else {
		m.Name = conf.Name
	}

	if len(m.Advertise) == 0 {
		conf.AdvertiseAddr = m.Name
	}
	return conf
}

// Joined has the node joined the cluster
func (m *Members) Joined() bool {
	return m.joined
}

// SetJoined set the node to "joined" this is used for seed nodes as
// otherwise it will never get set...
func (m *Members) SetJoined(j bool) {
	m.joined = j
}

// PollJoin we actual want the "join" to continue endlessly, in case seed node drops out
func (m *Members) PollJoin(seed string) {
	tick := time.NewTicker(time.Duration(30) * time.Second)
	for {
		<-tick.C
		if !m.joined {
			m.JoinRetry(seed)
		}
	}
}

func (m *Members) JoinRetry(seed string) error {
	s := seed
	// a permanent retry in case of errors
	retryNotify := func(err error, after time.Duration) {
		glog.Errorf("Gossip join failed to seed %s will retry :: %v :: will rety after %s", s, err, after)
	}
	retryFunc := func() error {
		return m.Join(s)
	}
	back := backoff.NewExponentialBackOff()
	back.MaxElapsedTime = time.Duration(24*5) * time.Hour // 5 days to keep re-pinging
	back.MaxInterval = time.Duration(30) * time.Second
	return backoff.RetryNotify(retryFunc, back, retryNotify)
}

func (m *Members) Join(seed string) error {
	_, err := m.List.Join(strings.Split(seed, ","))
	if err != nil {
		glog.Notice("Failed to join cluster: seed %s: %v", seed, err)
	} else {
		m.joined = true
		m.Seed = seed
		if m.List == nil {
			membs, err := memberlist.Create(m.Conf)
			if err != nil {
				return err
			}
			m.List = membs
		}
	}
	return err
}

// NodeMap copy out the current nodes
func (m *Members) NodeMap() map[string]Member {
	mOut := make(map[string]Member)
	m.nLock.RLock()
	defer m.nLock.RUnlock()
	for k, mem := range m.Nodes {
		mOut[k] = *mem
	}
	return mOut
}

// Members
func (m *Members) Members() []*memberlist.Node {
	mems := m.List.Members()
	mList := make(map[string]bool)
	mOut := make([]*memberlist.Node, 0)
	for _, mem := range mems {
		nm := fmt.Sprintf("%s:%d:%s", mem.Addr, mem.Port, mem.Name)
		if _, ok := mList[nm]; !ok {
			mOut = append(mOut, mem)
			mList[nm] = true
		}
	}
	glog.Debugf("Members %v", mOut)
	return mOut
}

// SetMeta update the nodes metadata
func (m *Members) SetMeta(meta MemberInfo) {

	if m.Meta == nil {
		m.Meta = make(map[string]MemberInfo)
	}

	nm := fmt.Sprintf("%s:%d", m.Conf.AdvertiseAddr, m.Conf.AdvertisePort)
	meta.GossipAddr = nm
	clusterKey := fmt.Sprintf("%s:%d", m.Conf.AdvertiseAddr, m.Conf.AdvertisePort)

	m.nLock.Lock()

	cKey := meta.Cluster
	m.Meta[cKey] = meta

	if _, ok := m.Nodes[clusterKey]; ok {
		m.Nodes[clusterKey].Info = m.Meta
		m.Nodes[clusterKey].LastUpdate = time.Now()
	} else {
		m.Nodes[clusterKey] = &Member{
			Name:       nm,
			Info:       m.Meta,
			LastUpdate: time.Now(),
		}
	}
	m.LocalInfo = m.Nodes[clusterKey]

	glog.Notice("Gossip: Setting Metadata For Cluster: `%v`", meta.Cluster)
	m.nLock.Unlock()
	if m.List != nil {
		go func() {
			err := m.List.UpdateNode(10 * time.Second)
			if err != nil {
				glog.Errorf("SetMeta: Error in update: %v", err)
			} else {
				glog.Notice("SetMeta: New metadata pushed")
			}
		}()
	}
}

func (m *Members) decodeMeta(meta []byte) (*Member, error) {
	info := new(Member)
	if len(meta) == 0 || string(meta) == "null" {
		return info, nil
	}

	buf := bytes.NewBuffer(nil)
	r := zstd.NewReader(buf)
	_, err := r.Read(meta)
	if err != nil {
		glog.Errorf("Error decoding zstd: %v: %v", err, string(meta))
		return nil, err
	}
	r.Close()
	btys := buf.Bytes()
	err = json.Unmarshal(btys, info)

	//err := json.Unmarshal(meta, info)
	if err != nil {
		glog.Errorf("decodeMeta: Error decoding Json: %v", err)
		return nil, err
	}
	return info, nil
}

func (m *Members) encodeMeta(meta *Member) ([]byte, error) {

	if meta == nil {
		return nil, nil
	}
	data, err := json.Marshal(meta)
	if err != nil {
		glog.Errorf("Error encoding json meta: %v", err)
		return nil, err
	}

	buf := bytes.NewBuffer(nil)
	c := zstd.NewWriterLevel(buf, zstd.BestCompression)
	_, err = c.Write(data)
	if err != nil {
		glog.Errorf("Error in compression meta: %v", err)
		return nil, err
	}
	c.Close()
	return buf.Bytes(), nil

	//return data, err
}

// NotifyJoin new member joining
func (m *Members) NotifyJoin(node *memberlist.Node) {
	return
	/*
		info, err := m.decodeMeta(node.Meta)
		if err != nil {
			glog.Errorf("Decode Error in NotifyJoin: %v", err)
			return
		}
		// use the the nodes addr and port as the key
		nm := fmt.Sprintf("%s:%d", node.Addr.String(), node.Port)
		m.nLock.Lock()
		m.Nodes[nm] = info
		m.NodeUpdated.Send(*info) //broadcast node updated
		m.nLock.Unlock()
		glog.Infof("NotifyJoin: new node key: %s Name: %s, Meta: %s", nm, node.Name, string(node.Meta))
	*/
}

// NotifyUpdate on update to a node, replace any metadata
func (m *Members) NotifyUpdate(node *memberlist.Node) {
	nm := fmt.Sprintf("%s:%d", node.Addr.String(), node.Port)
	glog.Infof("NotifyUpdate: updated key: %s node %s", nm, node.Name)
	return
	/*
		m.nLock.Lock()
		defer m.nLock.Unlock()

		if len(node.Meta) == 0 {
			return
		}
		nm := fmt.Sprintf("%s:%d", node.Addr.String(), node.Port)
		info, err := m.decodeMeta(node.Meta)
		if err != nil {
			glog.Errorf("Json Error in NotifyUpdate: %v", err)
			return
		}
		glog.Infof("NotifyUpdate: updated key: %s node %s: Meta: %s", nm, node.Name, info)

		// use the the nodes addr and port as the key
		m.Nodes[nm] = info
		m.NodeUpdated.Send(*info) //broadcast node updated
		glog.Infof("NotifyUpdate: new meta %v", info)
	*/
}

// NotifyLeave
func (m *Members) NotifyLeave(node *memberlist.Node) {
	nm := fmt.Sprintf("%s:%d", node.Addr, node.Port)
	m.nLock.Lock()
	defer m.nLock.Unlock()
	if n, ok := m.Nodes[nm]; ok {
		m.NodeLeft.Send(*n)
		delete(m.Nodes, nm)
	}
	glog.Noticef("NotifyLeave: Node left the cluster: %s:%d", node.Addr, node.Port)
}

// NotifyMsg :: noop
func (m *Members) NotifyMsg(buf []byte) {
	return
}

// Delegate memberlist interface

// GetBroadcasts :: noop
func (m *Members) GetBroadcasts(overhead, limit int) [][]byte {
	return nil
}

// LocalState serialize all the nodes metadata
func (m *Members) LocalState(join bool) []byte {
	m.nLock.Lock()
	info, err := json.Marshal(m.LocalInfo)
	m.nLock.Unlock()
	if err != nil {
		glog.Infof("NodeMeta: Json serialize failed %v", err)
	}
	return info
}

// NodeMeta current node's metadata
func (m *Members) NodeMeta(limit int) []byte {
	nm := fmt.Sprintf("%s:%d", m.Conf.AdvertiseAddr, m.Conf.AdvertisePort)
	m.nLock.RLock()
	info, err := m.encodeMeta(m.Nodes[nm])
	m.nLock.RUnlock()
	if err != nil {
		glog.Errorf("NodeMeta: Json serialize failed %v", err)
	}
	return info
}

// MergeRemoteState add nodes to our list if not present already
func (m *Members) MergeRemoteState(buf []byte, join bool) {

	haveNodes := new(Member)
	glog.Info("Gossip: MergeRemoteState: Merging States")

	err := json.Unmarshal(buf, &haveNodes)
	if err != nil {
		glog.Errorf("MergeRemoteState: Json serialize failed %v", err)
		return
	}
	glog.Debug("Gossip: MergeRemoteState: Merging States: Data: %s", string(buf))
	if haveNodes == nil {
		return
	}

	m.nLock.Lock()
	k := haveNodes.Name
	m.Nodes[k] = haveNodes
	m.NodeUpdated.Send(*haveNodes)
	m.nLock.Unlock()
	return
	/*for nm, node := range haveNodes {
		glog.Noticef("MergeRemoteState: Have new node: %s %s", nm, node.LastUpdate)

		m.Nodes[nm] = &node
		m.NodeUpdated.Send(node)
	}
	m.nLock.Unlock()*/
}
