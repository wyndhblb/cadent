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
   For a given set of api hosts, we have a little cluster object that can do the fan outs and grpc initialization

   If enabled (by setting the `cadent_hosts =["host1:port1", "host2:port2"]` in the  api config) then
   for 3 function Find, RawRender, and Render this will do a fan out to all the hosts in that list.

   For RawRender and Render, we first query "findincache" to get the proper host -> to -> metrics place for things

   since there can be "multiple" api backends each w/ a different cluster, we need to be able to separate clusters
   for each ApiLoop

*/

package http

import (
	sapi "cadent/server/schemas/api"
	sindexer "cadent/server/schemas/indexer"
	smetrics "cadent/server/schemas/metrics"
	grpclient "cadent/server/writers/rpc/client"
	"golang.org/x/net/context"

	"cadent/server/gossip"
	"cadent/server/utils"
	"cadent/server/utils/shared"
	"cadent/server/writers/indexer"
	"cadent/server/writers/metrics"
	"errors"
	"fmt"
	"github.com/opentracing/opentracing-go"
	"gopkg.in/op/go-logging.v1"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

var ErrNotInClusterMode = errors.New("Not in cluster mode")

type rpcClients struct {
	localNodeHost string
	localNodeIp   string
	cLock         sync.RWMutex
	clients       map[string]*grpclient.MetricClient
	log           *logging.Logger
	clusterMode   bool
	clusterName   string
	localMetrics  metrics.Metrics
	localIndexer  indexer.Indexer
	localApi      *ApiLoop
	tracer        opentracing.Tracer
}

type rpcClientMap struct {
	sync.RWMutex
	rpc map[string]*rpcClients // cluster name -> clients
}

var RpcClients *rpcClientMap

func init() {
	RpcClients = new(rpcClientMap)
	RpcClients.rpc = make(map[string]*rpcClients)
}

func (rp *rpcClientMap) Init(hosts []*url.URL, api *ApiLoop, tracer opentracing.Tracer) (*rpcClients, error) {

	cName := api.Conf.ClusterName

	rp.Lock()
	defer rp.Unlock()

	if g, ok := rp.rpc[cName]; ok && len(g.clients) > 0 {
		return g, nil
	}

	nClient := new(rpcClients)
	nClient.log = logging.MustGetLogger(fmt.Sprintf("rpc.clients.%s", cName))
	nClient.clients = make(map[string]*grpclient.MetricClient)
	nClient.clusterMode = false
	nClient.clusterName = cName
	nClient.tracer = tracer

	nClient.localApi = api

	// get our hostname
	onHost, err := os.Hostname()
	if err != nil {
		nClient.log.Errorf("Could not get the local hostname: %v", err)
		return nil, err
	}
	nClient.localNodeHost = onHost

	// get the current IP
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		nClient.log.Error("Could not get the local interfaces: %v", err)
		return nil, err
	}

	var ip string
	// have to just pick the first non-loopback as we don't have much else to go on
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip = ipnet.IP.String()
				break
			}
		}
	}

	nClient.localNodeIp = ip

	// make some clients for the static list of hosts if added
	if len(hosts) > 0 {
		for _, h := range hosts {
			nClient.AddGrpcHost(h.String())
		}
		nClient.clusterMode = true
	}

	if gossip.Enabled() {
		go nClient.listenGossip()
		go func() {
			// wait a bit of time for gossip to settle and get some items
			tick := time.NewTicker(time.Second * 5)
			defer tick.Stop()
			<-tick.C

			for {
				<-tick.C
				// set our gossip metadata .. may need to wait a bit until things have joined
				info := nClient.localApi.GetInfoData()
				isApi, _ := shared.Get("is_api").(bool)
				isTcpApi, _ := shared.Get("is_tcpapi").(bool)
				isReader, _ := shared.Get("is_reader").(bool)
				isWriter, _ := shared.Get("is_writer").(bool)
				isRcp, _ := shared.Get("is_grpc").(bool)
				meta := gossip.MemberInfo{
					GRPCaddr:    info.GRPCHost,
					HttpApiUrl:  info.AdvertiseUrl,
					Resolutions: info.Resolutions,
					Name:        info.AdvertiseName,
					Cluster:     nClient.clusterName,
					IsApi:       isApi,
					IsTcpApi:    isTcpApi,
					IsReader:    isReader,
					IsWriter:    isWriter,
					IsGrpc:      isRcp,
				}
				if gossip.Get() == nil {
					continue
				}
				if gossip.Get().Joined() {
					gossip.Get().SetMeta(meta)
					return
				}
			}
		}()
	}
	rp.rpc[cName] = nClient
	return nClient, nil
}

// Clients map of host -> rpc clients
func (rp *rpcClients) Clients() map[string]*grpclient.MetricClient {
	return rp.clients
}

// SetLocalHost set the name of our local hosting
func (rp *rpcClients) SetLocalHost(inhost string) {
	rp.localNodeHost = inhost
}

// SetLocalMetrics set the metrics engine for our local node
func (rp *rpcClients) SetLocalMetrics(mets metrics.Metrics) {
	rp.localMetrics = mets
}

// SetLocalIndexer set the indexer engine for our local node
func (rp *rpcClients) SetLocalIndexer(idx indexer.Indexer) {
	rp.localIndexer = idx
}

// IsLocal is the node in question our local node
func (rp *rpcClients) IsLocal(inhost string) bool {
	return inhost == rp.localNodeIp || inhost == rp.localNodeHost
}

// ClusterMode do we have anything in the cluster
func (rp *rpcClients) ClusterMode() bool {
	return rp.clusterMode
}

// listenGossip hook into the gossip lands, on updates add/change things to our Cluster
func (rp *rpcClients) listenGossip() {
	if !gossip.Enabled() {
		return
	}

	updates := gossip.Get().NodeUpdated.Listen()
	left := gossip.Get().NodeLeft.Listen()
	defer updates.Close()
	defer left.Close()

	for {
		select {
		case data, more := <-updates.Ch:

			if !more {
				return
			}
			member, ok := data.(gossip.Member)
			if !ok {
				rp.log.Warning("Did not get a gossip.Member from the gossip channel")
				continue
			}

			// match cluster names
			for cluster, info := range member.Info {
				if cluster != rp.clusterName {
					continue
				}
				if !info.IsGrpc {
					rp.log.Notice("Host %s is not a gRPC client will not add", cluster)
					continue
				}
				if len(info.GRPCaddr) == 0 {
					rp.log.Warning("Did not get a gossip.GRPCaddr is blank cannot add to cluster")
					continue
				}

				rp.AddGrpcHost(info.GRPCaddr)
			}

			rp.clusterMode = true
		case data, more := <-left.Ch:
			if !more {
				return
			}

			// a node leaves via the gossip addr
			member, ok := data.(gossip.Member)
			if !ok {
				rp.log.Warning("Did not get a gossip.Member from the gossip channel")
				continue
			}
			for cluster, info := range member.Info {
				if cluster != rp.clusterName {
					continue
				}

				if !info.IsGrpc {
					rp.log.Notice("Host %s is not a gRPC client will not remove", cluster)
					continue
				}

				if len(info.GRPCaddr) == 0 || !info.IsGrpc {
					rp.log.Warning("Did not get a gossip.GRPCaddr is blank cannot add to cluster")
					continue
				}

				rp.DelGrpcHost(info.GRPCaddr)
			}
		}
	}
}

// AddGrpcHost add a host to the cluster, will return true if added and false if not (false also means already added)
func (rp *rpcClients) AddGrpcHost(host string) bool {

	rp.cLock.Lock()
	defer rp.cLock.Unlock()

	if _, ok := rp.clients[host]; ok {
		rp.log.Debugf("grpc host: %s is already added", host)
		return false
	}
	gclient, err := grpclient.New(host, nil, "", rp.tracer)

	// let things "fail" here, but w/ a warning, as the host just may be down
	if err != nil {
		rp.log.Errorf("Error connecting to host: %s: %v", host, err)
	}
	rp.clients[host] = gclient
	rp.log.Notice("grpc host: %s added to cluster: total nodes %d", host, len(rp.clients))
	return true
}

// DelGrpcHost remove a host to the cluster, will return true if removed and false if not (false also means already removed)
func (rp *rpcClients) DelGrpcHost(host string) bool {

	rp.cLock.Lock()
	defer rp.cLock.Unlock()

	if _, ok := rp.clients[host]; ok {
		delete(rp.clients, host)
		rp.log.Debugf("grpc host: %s not removed", host)
		return true
	}
	rp.log.Debugf("grpc host: %s not added", host)
	return false

}

// FindInCache find the elements in all the clients that have cached the items
func (rp *rpcClients) FindInCache(ctx context.Context, mq *sapi.IndexQuery) (map[string][]*sindexer.MetricFindItem, error) {

	wg := utils.GetWaitGroup()
	dLock := utils.GetRWMutex()

	errlist := make([]error, 0)
	dataums := make(map[string][]*sindexer.MetricFindItem)
	dQ := mq

	rp.cLock.RLock()
	defer rp.cLock.RUnlock()

	for u, c := range rp.clients {
		wg.Add(1)
		onH := u
		onC := c

		if rp.IsLocal(onH) && rp.localIndexer != nil {
			go func() {
				data, err := rp.localIndexer.FindInCache(ctx, dQ.Query, dQ.Tags)
				if err != nil {
					rp.log.Errorf("FindInCache error for host %s: %v", onH, err)
					errlist = append(errlist, err)
				}
				dLock.Lock()
				dataums[onH] = data
				dLock.Unlock()
				wg.Done()
			}()
		} else {
			go func() {
				data, err := onC.FindMetricsInCache(ctx, dQ)
				if err != nil {
					rp.log.Errorf("FindInCache error for host %s: %v", onH, err)
					errlist = append(errlist, err)
				}
				dLock.Lock()
				dataums[onH] = data
				dLock.Unlock()
				wg.Done()
			}()
		}
	}

	wg.Wait()
	utils.PutWaitGroup(wg)
	utils.PutRWMutex(dLock)
	if len(errlist) > 0 {
		return dataums, errlist[0]
	}
	return dataums, nil
}

// List of metric paths per rpc host
func (rp *rpcClients) List(ctx context.Context, mq *sapi.IndexQuery) (map[string][]*sindexer.MetricFindItem, error) {

	wg := utils.GetWaitGroup()
	dLock := utils.GetRWMutex()

	errlist := make([]error, 0)
	dataums := make(map[string][]*sindexer.MetricFindItem)
	rp.cLock.RLock()
	defer rp.cLock.RUnlock()
	for u, c := range rp.clients {
		wg.Add(1)
		onH := u
		onC := c
		if rp.IsLocal(onH) && rp.localIndexer != nil {
			go func() {
				data, err := rp.localIndexer.List(mq.HasData, int(mq.Page))
				if err != nil {
					errlist = append(errlist, err)
				}
				dLock.Lock()
				dataums[onH] = data
				dLock.Unlock()
				wg.Done()
			}()
		} else {

			go func() {
				data, err := onC.List(ctx, mq)
				if err != nil {
					errlist = append(errlist, err)
				}
				dLock.Lock()
				dataums[onH] = data
				dLock.Unlock()
				wg.Done()
			}()
		}
	}

	wg.Wait()
	utils.PutWaitGroup(wg)
	utils.PutRWMutex(dLock)
	if len(errlist) > 0 {
		return dataums, errlist[0]
	}
	return dataums, nil
}

// Metrics of metric paths per rpc host
func (rp *rpcClients) Metrics(ctx context.Context, mq *sapi.MetricQuery) (map[string][]*smetrics.RawRenderItem, error) {

	// first find the items per host as we need to query each host where things live in cache
	idxQ := new(sapi.IndexQuery)
	idxQ.Query = mq.Target
	idxQ.Tags = mq.Tags
	items, err := rp.FindInCache(ctx, idxQ)
	dataums := make(map[string][]*smetrics.RawRenderItem)

	// bail if error and no data
	if err != nil {
		rp.log.Errorf("Error in find: %v", err)
		if len(items) == 0 {
			return dataums, err
		}
	}

	wg := utils.GetWaitGroup()
	dLock := utils.GetRWMutex()

	errlist := make([]error, 0)
	rp.cLock.RLock()
	defer rp.cLock.RUnlock()
	for host, found := range items {
		wg.Add(1)
		onH := host
		foundItems := found

		client, ok := rp.clients[onH]
		if !ok {
			rp.log.Errorf("Somehow the find used a rpc client that does not exist: %v", host)
			wg.Done()
			continue
		}

		// nothing to do
		if len(foundItems) == 0 {
			dataums[onH] = nil
			wg.Done()
			continue
		}

		// make a "large target" of the found items
		metricQuery := new(sapi.MetricQuery)
		metricQuery.InCache = mq.InCache
		metricQuery.Tags = mq.Tags
		metricQuery.Agg = mq.Agg
		metricQuery.End = mq.End
		metricQuery.Start = mq.Start
		metricQuery.Step = mq.Step
		metricQuery.MaxPoints = mq.MaxPoints

		targets := []string{}
		for _, item := range foundItems {
			targets = append(targets, item.Path)
		}
		metricQuery.Target = strings.Join(targets, ",")
		if rp.IsLocal(onH) && rp.localMetrics != nil {
			go func() {
				//path string, start int64, end int64, tags repr.SortingTags, resample uint32
				data, err := rp.localMetrics.RawRender(ctx, metricQuery.Target, metricQuery.Start, metricQuery.End, metricQuery.Tags, metricQuery.Step)
				if err != nil {
					errlist = append(errlist, err)
				}
				dLock.Lock()
				dataums[onH] = data
				dLock.Unlock()
				wg.Done()
			}()
		} else {
			go func() {
				data, err := client.GetMetrics(ctx, metricQuery)
				if err != nil {
					errlist = append(errlist, err)
				}
				dLock.Lock()
				dataums[onH] = data
				dLock.Unlock()
				wg.Done()
			}()
		}

	}

	wg.Wait()
	utils.PutWaitGroup(wg)
	utils.PutRWMutex(dLock)
	if len(errlist) > 0 {
		return dataums, errlist[0]
	}
	return dataums, nil
}
