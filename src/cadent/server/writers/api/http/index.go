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

   Http handlers for the "find me a metric" please

*/

package http

import (
	sapi "cadent/server/schemas/api"
	sindexer "cadent/server/schemas/indexer"
	"cadent/server/stats"
	"cadent/server/writers/indexer"
	"cadent/server/writers/metrics"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/opentracing/opentracing-go"
	"golang.org/x/net/context"
	"net/http"
	"strings"
	"time"
)

type FindAPI struct {
	a       *ApiLoop
	indexer indexer.Indexer
	metrics metrics.Metrics
}

func NewFindAPI(a *ApiLoop) *FindAPI {
	return &FindAPI{
		a:       a,
		indexer: a.Indexer,
		metrics: a.Metrics,
	}
}

func (f *FindAPI) AddHandlers(mux *mux.Router) {
	// these are for a "raw" graphite like finder (skipping the graphite-api)
	mux.HandleFunc("/metrics/find/", f.Find)
	mux.HandleFunc("/metrics/find", f.Find)

	// cluster things
	mux.HandleFunc("/cluster/find", f.FindInCluster)
	mux.HandleFunc("/cluster/list", f.ListInCluster)

	// cluster things but only the local finder
	mux.HandleFunc("/local/find", f.FindInLocal)
	mux.HandleFunc("/local/list", f.List)

	// for 'py-cadent'
	mux.HandleFunc("/find/incache", f.FindInCache)
	mux.HandleFunc("/find", f.Find)
	mux.HandleFunc("/paths", f.Find)
	mux.HandleFunc("/expand", f.Expand)

	mux.HandleFunc("/list", f.List)
}

// findInClusterMode
func (re *FindAPI) findInClusterMode(ctx context.Context, w http.ResponseWriter, r *http.Request, args sapi.IndexQuery) {

	if !re.a.rpcClient.ClusterMode() {
		re.a.log.Errorf("Error in cluster find: %v", ErrNotInClusterMode)
		re.a.OutError(w, ErrNotInClusterMode.Error(), http.StatusInternalServerError)
		return
	}

	// map[host][]*items from the cluster caches
	dataums, err := re.a.rpcClient.FindInCache(ctx, &args)
	if err != nil {
		stats.StatsdClientSlow.Incr("reader.http.cluster.find.errors", 1)
		re.a.log.Errorf("Error in find: %v", err)
		if len(dataums) == 0 {
			re.a.OutError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// dedupe the list
	seenMap := make(map[string]bool)
	var data sindexer.MetricFindItems
	for _, items := range dataums {
		for _, item := range items {
			if _, ok := seenMap[item.Path]; ok {
				continue
			}
			if item.UniqueId != "" {
				if _, ok := seenMap[item.UniqueId]; ok {
					continue
				}
				seenMap[item.UniqueId] = true
			}
			seenMap[item.Path] = true

			data = append(data, item)
		}
	}

	stats.StatsdClientSlow.Incr("reader.http.cluster.find.ok", 1)

	re.a.OutOk(w, data, args.Format)
}

// FindInCluster items from the cluster mode caches
func (f *FindAPI) FindInCluster(w http.ResponseWriter, r *http.Request) {

	span, ctx := f.a.GetSpan("FindInCluster", r)
	defer span.Finish()

	f.a.AddSpanEvent(span, "started")
	defer func() { f.a.AddSpanEvent(span, "ended") }()

	r.ParseForm()

	args, err := ParseFindQuery(r)
	if err != nil {
		span.SetTag("error", err)
		f.a.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
		return
	}

	if len(args.Query) == 0 {
		span.SetTag("error", "query/target param is required")
		f.a.OutError(w, "Query is required", http.StatusBadRequest)
		return
	}
	dataums, err := f.a.rpcClient.FindInCache(ctx, &args)
	if err != nil {
		stats.StatsdClientSlow.Incr("reader.http.cluster.find.errors", 1)
		span.SetTag("error", err)
		f.a.log.Errorf("Error in find: %v", err)
		if len(dataums) == 0 {
			f.a.OutError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	f.a.OutOk(w, dataums, args.Format)
}

func (f *FindAPI) spanLog(span opentracing.Span, args sapi.IndexQuery) {
	span.LogKV(
		"target", args.Query,
		"tags", args.Tags,
		"has_data", args.HasData,
		"in_cache", args.InCache,
		"cluster_mode", f.a.rpcClient.ClusterMode(),
		"on_host", f.a.rpcClient.localNodeHost,
		"cluster_name", f.a.rpcClient.clusterName,
	)
}

// Find metrics from the indexer from an input query
func (f *FindAPI) find(w http.ResponseWriter, r *http.Request, usecluster bool) {
	defer stats.StatsdSlowNanoTimeFunc("reader.http.find.get-time-ns", time.Now())
	stats.StatsdClientSlow.Incr("reader.http.find.hits", 1)

	span, ctx := f.a.GetSpan("find", r)
	defer span.Finish()

	f.a.AddSpanEvent(span, "started")
	defer func() { f.a.AddSpanEvent(span, "ended") }()

	r.ParseForm()

	args, err := ParseFindQuery(r)
	if err != nil {
		span.SetTag("error", err)
		f.a.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
		return
	}
	f.spanLog(span, args)

	if len(args.Query) == 0 {
		f.a.AddSpanError(span, fmt.Errorf("query/target param is required"))
		f.a.OutError(w, "Query is required", http.StatusBadRequest)
		return
	}

	if f.a.rpcClient.ClusterMode() && usecluster {
		f.findInClusterMode(ctx, w, r, args)
		return
	}

	data, err := f.indexer.Find(ctx, args.Query, args.Tags)
	if err != nil {
		f.a.AddSpanError(span, err)
		stats.StatsdClientSlow.Incr("reader.http.find.errors", 1)
		f.a.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	stats.StatsdClientSlow.Incr("reader.http.find.ok", 1)
	f.a.OutOk(w, data, args.Format)
	return
}

// Find metrics from the indexer from an input query
func (f *FindAPI) Find(w http.ResponseWriter, r *http.Request) {
	f.find(w, r, true)
}

// FindInLocal metrics from the indexer from an input query, but just for this host (if in a cluster mode)
func (f *FindAPI) FindInLocal(w http.ResponseWriter, r *http.Request) {
	f.find(w, r, false)
}

// FindInCache find only those things that are in the Ram index cache (if available)
func (f *FindAPI) FindInCache(w http.ResponseWriter, r *http.Request) {
	span, ctx := f.a.GetSpan("FindInCache", r)
	defer f.a.SpanStartEnd(span)()

	defer stats.StatsdSlowNanoTimeFunc("reader.http.findincache.get-time-ns", time.Now())
	stats.StatsdClientSlow.Incr("reader.http.findincache.hits", 1)
	r.ParseForm()

	args, err := ParseFindQuery(r)

	if err != nil {
		f.a.AddSpanError(span, err)
		f.a.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
	}
	f.spanLog(span, args)

	if len(args.Query) == 0 {
		f.a.AddSpanError(span, fmt.Errorf("query/target param is required"))
		f.a.OutError(w, "Query is required", http.StatusBadRequest)
		return
	}

	data, err := f.indexer.FindInCache(ctx, args.Query, args.Tags)
	if err != nil {
		f.a.AddSpanError(span, err)
		stats.StatsdClientSlow.Incr("reader.http.findincache.errors", 1)
		f.a.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	stats.StatsdClientSlow.Incr("reader.http.findincache.ok", 1)
	f.a.OutOk(w, data, args.Format)
	return
}

func (f *FindAPI) Expand(w http.ResponseWriter, r *http.Request) {
	span, _ := f.a.GetSpan("Expand", r)

	defer span.Finish()
	f.a.AddSpanEvent(span, "started")
	defer func() { f.a.AddSpanEvent(span, "ended") }()

	defer stats.StatsdSlowNanoTimeFunc("reader.http.expand.get-time-ns", time.Now())
	stats.StatsdClientSlow.Incr("reader.http.expand.hits", 1)
	r.ParseForm()
	var query string

	format := "json"

	if r.Method == "GET" {
		query = strings.TrimSpace(r.Form.Get("query"))
		format = r.Form.Get("format")
	} else {
		query = strings.TrimSpace(r.FormValue("query"))
		format = r.FormValue("format")
	}
	if len(format) == 0 {
		format = "json"
	}

	if len(query) == 0 {
		f.a.OutError(w, "Query is required", http.StatusBadRequest)
		return
	}

	data, err := f.indexer.Expand(query)
	if err != nil {
		f.a.AddSpanError(span, err)
		stats.StatsdClientSlow.Incr("reader.http.expand.errors", 1)
		f.a.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	stats.StatsdClientSlow.Incr("reader.http.expand.ok", 1)
	f.a.OutOk(w, data, format)
}

func (f *FindAPI) List(w http.ResponseWriter, r *http.Request) {
	span, _ := f.a.GetSpan("List", r)
	defer f.a.SpanStartEnd(span)()

	defer stats.StatsdSlowNanoTimeFunc("reader.http.list.get-time-ns", time.Now())
	stats.StatsdClientSlow.Incr("reader.http.list.hits", 1)

	args, err := ParseFindQuery(r)

	if err != nil {
		f.a.AddSpanError(span, err)
		f.a.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
		return
	}
	f.spanLog(span, args)

	datas, err := f.indexer.List(args.HasData, int(args.Page))
	if err != nil {
		stats.StatsdClientSlow.Incr("reader.http.list.errors", 1)
		f.a.AddSpanError(span, err)
		f.a.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	if datas == nil {
		f.a.AddSpanLog(span, "nodata", "true")
		f.a.OutError(w, "No data found", http.StatusNoContent)
		return
	}
	res := []string{}
	for _, data := range datas {
		res = append(res, data.Path)
	}
	stats.StatsdClientSlow.Incr("reader.http.list.ok", 1)
	f.a.OutOk(w, sindexer.MetricListItems(res), args.Format)
}

func (f *FindAPI) ListInCluster(w http.ResponseWriter, r *http.Request) {
	span, ctx := f.a.GetSpan("ListInCluster", r)
	defer f.a.SpanStartEnd(span)()

	defer stats.StatsdSlowNanoTimeFunc("reader.http.cluster.list.get-time-ns", time.Now())
	stats.StatsdClientSlow.Incr("reader.http.cluster.list.hits", 1)

	args, err := ParseFindQuery(r)

	if err != nil {
		f.a.AddSpanError(span, err)
		f.a.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
		return
	}
	f.spanLog(span, args)

	dataums, err := f.a.rpcClient.List(ctx, &args)
	if err != nil {
		stats.StatsdClientSlow.Incr("reader.http.cluster.list.errors", 1)
		f.a.AddSpanError(span, err)
		f.a.log.Errorf("Error in find: %v", err)
		if len(dataums) == 0 {
			f.a.OutError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	stats.StatsdClientSlow.Incr("reader.http.cluster.list.ok", 1)
	f.a.OutOk(w, dataums, args.Format)
}
