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
   This is the "standalone" API server, which will interface to the main
   Data store as well as use a "seed" or "discovery" to grab any members in a cluster of writers

   //TODO: Still a work in progress .. need to build in some "get from cache" mechanisms

   metrics/indexer interfaces

   example config
   [api]
        base_path = "/graphite/"
        listen = "0.0.0.0:8084"
        seed = "http://localhost:8083/graphite"
        #and / OR (it will get the resolutions from the seed
        resolutions = [int, int, int]

            [api.metrics]
            driver = "whisper"
            dsn = "/data/graphite/whisper"

            # this is the read cache that will keep the latest goods in ram
            read_cache_max_items=102400
            read_cache_max_bytes_per_metric=8192

            [api.indexer]
            driver = "leveldb"
            dsn = "/data/graphite/idx"

   ---- or in discovery mode ----
       [api]
        base_path = "/graphite/"
        listen = "0.0.0.0:8084"

        #and / OR (it will get the resolutions from the seed
        resolutions = [int, int, int]

	    [api.discover]
            driver="zookeeper"
            dsn="127.0.0.1:2181"

            [api.metrics]
            driver = "whisper"
            dsn = "/data/graphite/whisper"

            # this is the read cache that will keep the latest goods in ram
            read_cache_max_items=102400
            read_cache_max_bytes_per_metric=8192

            [api.indexer]
            driver = "leveldb"
            dsn = "/data/graphite/idx"
*/

package solo

import (
	apiinfo "cadent/server/writers/api"
	api "cadent/server/writers/api/http"
	"cadent/server/writers/indexer"
	"cadent/server/writers/metrics"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"gopkg.in/op/go-logging.v1"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DEFAULT_HTTP_TIMEOUT = 10 * time.Second

type SoloApiLoop struct {
	Conf api.ApiConfig
	Api  *api.ApiLoop

	Metrics metrics.Metrics
	Indexer indexer.Indexer

	Members          []string
	MemberInfo       []*apiinfo.InfoData
	MaxMinResolution [][]int // weird name, but its the "max" each resolution from each member

	shutdown chan bool
	log      *logging.Logger

	ReadCache *metrics.ReadCache

	started bool
}

func NewSoloApiLoop() *SoloApiLoop {
	s := new(SoloApiLoop)
	s.log = logging.MustGetLogger("api.solo")
	return s
}

/*** using gossip mode to get node info ***/
// periodically check the seed info for any member info to update the list
func (re *SoloApiLoop) getMembersGossip() {

	tick := time.NewTicker(time.Minute)
	for {
		<-tick.C
		if len(re.Members) == 0 {
			continue
		}

		// loop through current members and which ever actually responds "wins"
		for _, m := range re.Members {
			_url := m + "/info"
			if strings.HasSuffix(m, "/") {
				_url = m + "info"
			}
			u, _ := url.Parse(_url) // need to add the info target
			_, err := re.GetSeedData(u)
			if err != nil {
				re.log.Critical("Unable to get member data at %s", m)
				continue
			}
			break
		}
	}

}

func (re *SoloApiLoop) getUrl(u *url.URL, timeout time.Duration) (*http.Response, error) {

	transport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: timeout,
		}).Dial,
		TLSHandshakeTimeout: timeout,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true}, // ignore "bad" certs
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	return client.Get(u.String())
}

// ComposeMembers from the member list, ask each member for it's API info
// we ASSUME that each member is setup the same old way the seed node is
// if not, we bail on error, as well that should not be the case
// this is not in use if the Discovery module is used
func (re *SoloApiLoop) ComposeMembers(seed *url.URL, membs []apiinfo.ApiMemberInfo) error {

	re.log.Debug("Incoming Members: %v", membs)
	newInfo := make([]*apiinfo.InfoData, 0)
	newMems := make([]string, 0)
	for _, n := range membs {
		re.log.Notice("Found cluster member at %s", n.GossipAddr)

		nUrl, err := url.Parse(n.HttpApiUrl)
		if err != nil {
			return err
		}
		infoD, err := re.GetInfoData(nUrl)
		if err != nil {
			re.log.Critical("Failed getting member at %s info: %v", nUrl, err)
			continue
		}
		newInfo = append(newInfo, infoD)
		if len(infoD.Api.Host) != 0 {
			api_host := fmt.Sprintf("%s://%s:%s/%s", infoD.Api.Scheme, infoD.Api.Host, infoD.Api.Port, infoD.Api.BasePath)
			newMems = append(newMems, api_host)
			re.log.Notice("Found Cluster API member at %s", api_host)

		}
	}
	re.MemberInfo = newInfo
	re.Members = newMems
	return nil
}

// GetApiMembers returns a list of URLs for members that have an API mode attached
func (re *SoloApiLoop) GetApiMembers() []string {
	if re.Api.DiscoverList != nil {
		out := make([]string, 0)
		hosts := re.Api.DiscoverList.ApiHosts()
		if hosts == nil {
			return out
		}
		for _, h := range hosts.Hosts {
			if h.IsApi && h.IsWriter {
				out = append(out, h.AdvertiseUrl)
			}
		}
		re.Members = out
	}
	return re.Members
}

// GetResolutions returns a list of URLs for members that have an API mode attached
func (re *SoloApiLoop) GetResolutions() [][]int {
	if re.Api.DiscoverList != nil {
		out := make([][]int, 0)
		hosts := re.Api.DiscoverList.ApiHosts()
		if hosts == nil {
			return out
		}

		for _, h := range hosts.Hosts {
			if h.IsApi {
				res := h.Resolutions
				if len(res) == 0 {
					continue
				}
				if len(out) == 0 {
					for _, r := range res {
						out = append(out, []int{int(r.Resolution), int(r.Ttl)})
					}
					continue
				}
				for idx, o := range out {
					// too many
					if len(res) < idx {
						break
					}
					if o[0] < int(res[0].Resolution) {
						out[idx][0] = int(res[0].Resolution)
					}
				}
			}
		}
		re.MaxMinResolution = out
	}
	return re.MaxMinResolution
}

// GetInfoData grab the /info from the various members lists
func (re *SoloApiLoop) GetInfoData(url *url.URL) (info *apiinfo.InfoData, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Error: Recovered: %v", r)
		}
	}()

	r, err := re.getUrl(url, DEFAULT_HTTP_TIMEOUT)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	info = new(apiinfo.InfoData)
	err = json.NewDecoder(r.Body).Decode(info)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (re *SoloApiLoop) GetSeedData(seed *url.URL) (*apiinfo.InfoData, error) {

	re.log.Notice("Attempting to get cluster info from %s", seed)
	info, err := re.GetInfoData(seed)
	if err != nil {
		return nil, err
	}
	err = re.ComposeMembers(seed, info.Members)
	return info, err

}

func (re *SoloApiLoop) Config(conf SoloApiConfig) (err error) {
	rl := new(api.ApiLoop)
	re.Api = rl

	// set up the discoverer lister
	re.Api.DiscoverList, err = conf.GetDiscoverList()

	// first need to grab things from any seeds (if any)
	if len(conf.Seed) == 0 && len(conf.Resolutions) == 0 && re.Api.DiscoverList == nil {
		return fmt.Errorf("`seed` or `resolutions` is required")
	}

	if err != nil {
		re.log.Critical("Error in Discover List module: %v", err)
		return err
	}

	if re.Api.DiscoverList != nil && len(conf.Seed) > 0 {
		re.log.Notice("Using Discovery module for finding nodes, no need for seed")
	}

	if re.Api.DiscoverList == nil && len(conf.Seed) > 0 {
		parsed, err := url.Parse(conf.Seed)
		if err != nil {
			return err
		}
		sData, err := re.GetSeedData(parsed)
		if err != nil {
			return err
		}

		if len(sData.Resolutions) == 0 {
			return fmt.Errorf("Unable to determine resolutions from seed %s", conf.Seed)
		}
		apiconf := conf.GetApiConfig()
		apiconf.ApiMetricOptions.UseCache = "dummy" // need a fake cacher
		err = rl.Config(apiconf, float64(sData.Resolutions[0][0]))
		if err != nil {
			return err
		}
		rl.SetResolutions(sData.Resolutions)
		re.Api = rl
		return nil
	} else if re.Api.DiscoverList != nil {
		// need to force a pull
		_, err := re.Api.DiscoverList.ForcePullHosts()
		if err != nil {
			return err
		}
		re.GetApiMembers()
		res := re.GetResolutions()
		re.log.Noticef("Discovered %d hosts", len(re.Members))
		if len(res) == 0 {
			return fmt.Errorf("Unable to determine resolutions from discovery %s", conf.Seed)
		}

		apiconf := conf.GetApiConfig()
		apiconf.ApiMetricOptions.UseCache = "dummy" // need a fake cacher

		re.log.Noticef("Using min resolution as %ds", re.MaxMinResolution[0][0])
		err = rl.Config(apiconf, float64(re.MaxMinResolution[0][0]))
		if err != nil {
			return err
		}

		rl.SetResolutions(res)

	} else {
		apiconf := conf.GetApiConfig()
		apiconf.ApiMetricOptions.UseCache = "dummy" // need a fake cacher

		err = rl.Config(apiconf, float64(conf.Resolutions[0]))
		if err != nil {
			return err
		}
		forRes := [][]int{}
		for _, r := range conf.Resolutions {
			forRes = append(forRes, []int{int(r), 0})
		}
		re.MaxMinResolution = forRes
		rl.SetResolutions(forRes)
	}

	return nil
}

func (re *SoloApiLoop) Stop() {
	re.Api.Stop()
}

func (re *SoloApiLoop) Start() {

	if re.Api.DiscoverList != nil {
		go re.Api.DiscoverList.Start()
	} else {
		go re.getMembersGossip()
	}

	re.Api.Start()
}
