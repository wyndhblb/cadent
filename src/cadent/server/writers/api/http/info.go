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
  Simple endpoint for getting "info" about this here server

  to be used for discovery of nodes around the land .. i.e. we give a server a "seed" server name
  it can then ping it and it can return back the list of other cadents out there

  this is NOT meant for a gossip like protocol, just for other services, there will
  eventually be an internal gossip protocol that wil drive the info from this endpoint

  i.e. one can use it if setting of an API frontend to discover all the cacher writer nodes out there
  (those writer/cacher nodes gossip to each other)

*/

package http

import (
	sapi "cadent/server/schemas/api"
	"cadent/server/schemas/repr"
	"cadent/server/stats"
	"cadent/server/writers/api/discovery"
	"cadent/server/writers/indexer"
	"cadent/server/writers/metrics"
	"github.com/gorilla/mux"
	"net/http"
	"strings"
)

type InfoAPI struct {
	a            *ApiLoop
	Indexer      indexer.Indexer
	Metrics      metrics.Metrics
	Discover     discovery.Discover
	DiscoverList discovery.Lister
}

func NewInfoAPI(a *ApiLoop) *InfoAPI {
	return &InfoAPI{
		a:            a,
		Indexer:      a.Indexer,
		Metrics:      a.Metrics,
		Discover:     a.Discover,
		DiscoverList: a.DiscoverList,
	}
}

func (c *InfoAPI) AddHandlers(mux *mux.Router) {
	mux.HandleFunc("/info", c.GetInfo)
	mux.HandleFunc("/discover/list", c.GetDiscover)
	mux.HandleFunc("/discover/find", c.GetDiscoverFind)
}

// GetInfo get this servers information, configs, etc
func (c *InfoAPI) GetInfo(w http.ResponseWriter, r *http.Request) {

	stats.StatsdClientSlow.Incr("reader.http.info.hit", 1)

	c.a.GetInfoData()

	stats.StatsdClientSlow.Incr("reader.http.info.ok", 1)
	switch FormatFromHeaders(r) {
	case "yaml":
		c.a.OutYaml(w, c.a.info)
	default:
		c.a.OutJson(w, c.a.info)
	}
}

// GetDiscover get the big old list of servers
func (c *InfoAPI) GetDiscover(w http.ResponseWriter, r *http.Request) {

	stats.StatsdClientSlow.Incr("reader.http.discover.hit", 1)

	if c.DiscoverList == nil {
		c.a.OutError(w, "Discover Listing not available (not set up)", http.StatusNotAcceptable)
		return
	}
	data := c.DiscoverList.ApiHosts()

	f := FormatFromHeaders(r)

	stats.StatsdClientSlow.Incr("reader.http.discover.ok", 1)
	c.a.OutOk(w, data, f)
}

// GetDiscoverFind "find" a set of hosts that match tags/host in the query
func (c *InfoAPI) GetDiscoverFind(w http.ResponseWriter, r *http.Request) {

	stats.StatsdClientSlow.Incr("reader.http.discoverfind.hit", 1)

	if c.DiscoverList == nil {
		c.a.OutError(w, "Discover Listing not available (not set up)", http.StatusNotAcceptable)
		return
	}
	data := c.DiscoverList.ApiHosts()

	q, err := ParseDiscoverQuery(r)
	if err != nil {
		c.a.OutError(w, err.Error(), http.StatusNotAcceptable)
		return
	}

	outList := new(sapi.DiscoverHosts)
	for _, d := range data.Hosts {

		if len(q.Host) > 0 && strings.ToLower(q.Host) != strings.ToLower(d.Host) && strings.ToLower(q.Host) != strings.ToLower(d.AdvertiseName) {
			continue
		}

		if len(q.Tags) == 0 {
			outList.Hosts = append(outList.Hosts, d)
			continue
		} else {
			if repr.FromTagList(d.Tags).HasAllTags(repr.SortingTags(q.Tags)) {
				outList.Hosts = append(outList.Hosts, d)
			}
		}
	}

	stats.StatsdClientSlow.Incr("reader.http.discoverfind.ok", 1)
	c.a.OutOk(w, outList, q.Format)
}
