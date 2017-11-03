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
   Some objects that define "us" as the cadent server
   used for the ../info api endpoint as well as the Discover data blob
*/

package api

import (
	"cadent/server/gossip"
	"cadent/server/schemas/repr"
	"cadent/server/utils/shared"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

type ApiInfoData struct {
	Host      string `json:"host"`
	BasePath  string `json:"path"`
	Scheme    string `json:"scheme"`
	Port      string `json:"port"`
	GrpcHost  string `json:"grpc_host,omitempty"`
	TCPHost   string `json:"tcp_host,omitempty"`
	TCPScheme string `json:"tcp_scheme,omitempty"`
	TCPPort   string `json:"tcp_port,omitempty"`
}

type ApiMemberInfo struct {
	gossip.MemberInfo
}

type InfoData struct {
	Time          int64           `json:"time"`
	MetricDriver  string          `json:"metric-driver"`
	IndexDriver   string          `json:"index-driver"`
	Resolutions   [][]int         `json:"resolutions"`
	Hostname      string          `json:"hostname"`
	AdvertiseName string          `json:"advertise-name"`
	AdvertiseUrl  string          `json:"advertise-url"`
	ClusterName   string          `json:"cluster-name"`
	GRPCHost      string          `json:"grpc-host"`
	Ip            []string        `json:"ip"`
	Members       []ApiMemberInfo `json:"members"`
	CachedMetrics int             `json:"cached-metrics"`
	RpcClients    []string        `json:"rpc-clients"`

	Api  ApiInfoData      `json:"api-server"`
	Tags repr.SortingTags `json:"tags,omitempty"`

	SharedData interface{} `json:"shared"`

	getLock sync.Mutex
}

// Get set everything we can about "us".
// Things like the MetricDriver, IndexDriver, Resolutions, ApiInfoData
// should be set inside the API itself
func (c *InfoData) Get() {
	name, err := os.Hostname()
	if err != nil {
		name = fmt.Sprintf("{Could not get Hostname: %v}", err)
		fmt.Println(err)
	}

	addrs, err := net.LookupHost(name)
	if err != nil {
		fmt.Println(err)
	}

	c.getLock.Lock()
	defer c.getLock.Unlock()

	c.Time = time.Now().UnixNano()
	c.Hostname = name
	c.Ip = addrs

	if gossip.Get() != nil {
		c.Members = []ApiMemberInfo{}
		mems := gossip.Get().NodeMap()
		if len(mems) == 0 {
			c.Members = nil
		} else {
			for _, m := range mems {
				for _, data := range m.Info {
					c.Members = append(
						c.Members,
						ApiMemberInfo{
							MemberInfo: data,
						},
					)
				}
			}
		}
	} else {
		c.Members = nil
	}

	apiServer := ApiInfoData{
		Host:   name,
		Scheme: "http",
	}

	c.Api = apiServer

	c.SharedData = shared.GetAll()
}
