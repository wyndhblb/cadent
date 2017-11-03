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
   Get Metrics Handlers
*/

package http

import (
	sapi "cadent/server/schemas/api"
	smetrics "cadent/server/schemas/metrics"
	"cadent/server/stats"
	"cadent/server/writers/indexer"
	"cadent/server/writers/metrics"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/opentracing/opentracing-go"
	"golang.org/x/net/context"
	"net/http"
	"sort"
	"time"
)

type MetricsAPI struct {
	a       *ApiLoop
	Indexer indexer.Indexer
	Metrics metrics.Metrics
}

func NewMetricsAPI(a *ApiLoop) *MetricsAPI {
	return &MetricsAPI{
		a:       a,
		Indexer: a.Indexer,
		Metrics: a.Metrics,
	}
}

func (m *MetricsAPI) AddHandlers(mux *mux.Router) {
	mux.HandleFunc("/cluster/rawrender", m.RawRenderInCluster)

	// raw graphite (skipping graphite-api)
	mux.HandleFunc("/render", m.GraphiteRender)

	// for py-cadent
	mux.HandleFunc("/metrics", m.Render)

	// cadent specific
	mux.HandleFunc("/rawrender", m.RawRender)
}

func (re *MetricsAPI) GetMetrics(ctx context.Context, args sapi.MetricQuery) ([]*smetrics.RawRenderItem, error) {
	stp := re.a.minResolution(args.Start, args.End, args.Step)
	return re.Metrics.RawRender(ctx, args.Target, args.Start, args.End, args.Tags, stp)
}

// getMetricsInClusterMode use the rpc clients to fan out and get all the metrics
// first we need to "find" things and pin them to hosts that their caches belong on
func (re *MetricsAPI) getMetricsInClusterMode(ctx context.Context, args sapi.MetricQuery) ([]*smetrics.RawRenderItem, error) {

	if !re.a.rpcClient.ClusterMode() {
		return nil, ErrNotInClusterMode
	}

	data := make([]*smetrics.RawRenderItem, 0)
	// map[host][]*items from the cluster
	dataums, err := re.a.rpcClient.Metrics(ctx, &args)
	if err != nil {
		re.a.log.Errorf("Error in find: %v", err)
	}

	if len(dataums) > 0 {
		// dedupe the list
		seenMap := make(map[string]bool)
		for _, items := range dataums {
			for _, item := range items {
				if _, ok := seenMap[item.Id]; ok {
					continue
				}
				seenMap[item.Id] = true
				data = append(data, item)
			}
		}
	}

	return data, nil
}

// getRawMetricsInClusterMode same as getMetricsInClusterMode but don't dedupe
func (re *MetricsAPI) getRawMetricsInClusterMode(ctx context.Context, args sapi.MetricQuery) (map[string][]*smetrics.RawRenderItem, error) {

	if !re.a.rpcClient.ClusterMode() {
		return nil, ErrNotInClusterMode
	}

	// map[host][]*items from the cluster
	dataums, err := re.a.rpcClient.Metrics(ctx, &args)
	if err != nil {
		re.a.log.Errorf("Error in find: %v", err)
		if len(dataums) == 0 {
			return nil, err
		}
	}

	return dataums, err
}

func (re *MetricsAPI) spanLog(span opentracing.Span, args sapi.MetricQuery) {
	span.LogKV(
		"target", args.Target,
		"tags", args.Tags,
		"start", args.Start,
		"end", args.End,
		"format", args.Format,
		"agg", args.Agg,
		"step", args.Step,
		"cluster_mode", re.a.rpcClient.ClusterMode(),
		"on_host", re.a.rpcClient.localNodeHost,
		"cluster_name", re.a.rpcClient.clusterName,
	)
}

// ToGraphiteRender take a rawrender and make it a graphite api json format
// meant for PyCadent hooked into the graphite-api backend storage item
func (re *MetricsAPI) ToGraphiteRender(raw_data []*smetrics.RawRenderItem) *smetrics.WhisperRenderItem {
	whis := new(smetrics.WhisperRenderItem)
	whis.Series = make(map[string]*smetrics.WhisperDataItem)

	if raw_data == nil {
		return nil
	}
	for _, data := range raw_data {
		if data == nil {
			continue
		}
		d_points := make([]*smetrics.DataPoint, data.Len(), data.Len())
		whis.End = data.End
		whis.Start = data.Start
		whis.From = data.Start
		whis.To = data.End
		whis.Step = data.Step
		whis.RealEnd = data.RealEnd
		whis.RealStart = data.RealStart

		whisData := &smetrics.WhisperDataItem{
			Target:     data.Metric,
			InCache:    data.InCache,
			UsingCache: data.UsingCache,
			Uid:        data.Id,
		}

		for idx, d := range data.Data {
			v := d.AggValue(data.AggFunc)
			d_points[idx] = &smetrics.DataPoint{Time: d.Time, Value: v}
		}
		whisData.Data = d_points
		whis.Series[data.Metric] = whisData
	}
	return whis
}

func (re *MetricsAPI) ToGraphiteApiRender(rawData []*smetrics.RawRenderItem) smetrics.GraphiteApiItems {
	graphite := make(smetrics.GraphiteApiItems, 0)

	if rawData == nil {
		return nil
	}

	for _, data := range rawData {
		if data == nil {
			continue
		}
		g_item := new(smetrics.GraphiteApiItem)
		d_points := make([]*smetrics.DataPoint, data.Len(), data.Len())
		g_item.Target = data.Metric
		g_item.InCache = data.InCache
		g_item.UsingCache = data.UsingCache

		for idx, d := range data.Data {
			v := d.AggValue(data.AggFunc)
			d_points[idx] = &smetrics.DataPoint{Time: d.Time, Value: v}
		}
		g_item.Datapoints = d_points
		graphite = append(graphite, g_item)
	}
	sort.Sort(graphite)
	return graphite
}

// GraphiteRender if using another system (aka grafana, that expects the graphite-api delivered format)
func (re *MetricsAPI) GraphiteRender(w http.ResponseWriter, r *http.Request) {

	span, ctx := re.a.GetSpan("GraphiteRender", r)
	defer re.a.SpanStartEnd(span)()

	defer stats.StatsdSlowNanoTimeFunc("reader.http.graphite-render.get-time-ns", time.Now())
	stats.StatsdClientSlow.Incr("reader.http.graphite-render.hits", 1)

	args, err := ParseMetricQuery(r)
	if err != nil {
		span.LogKV("Invalid Args", args.String())
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
		return
	}

	re.spanLog(span, args)

	var data smetrics.RawRenderItems
	if re.a.rpcClient.ClusterMode() {
		data, err = re.getMetricsInClusterMode(ctx, args)
	} else {
		data, err = re.GetMetrics(ctx, args)
	}

	if err != nil {
		stats.StatsdClientSlow.Incr("reader.http.graphite-render.errors", 1)
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	if data == nil || len(data) == 0 {
		stats.StatsdClientSlow.Incr("reader.http.graphite-render.nodata", 1)
		re.a.AddSpanLog(span, "nodata", args.String())
		re.a.OutError(w, "No data found", http.StatusNoContent)
		return
	}

	for idx := range data {
		if data[idx] == nil {
			continue
		}

		// graphite needs the "nils" and expects a "full list" to match the step + start/end
		data[idx].Start = uint32(args.Start)
		data[idx].End = uint32(args.End)
		data[idx].Quantize()
	}

	renderData := re.ToGraphiteApiRender(data)

	re.a.AddToCache(data)
	stats.StatsdClientSlow.Incr("reader.http.graphite-render.ok", 1)

	re.a.OutOk(w, renderData, args.Format)
	return
}

func (re *MetricsAPI) Render(w http.ResponseWriter, r *http.Request) {

	defer stats.StatsdSlowNanoTimeFunc("reader.http.render.get-time-ns", time.Now())
	stats.StatsdClientSlow.Incr("reader.http.render.hits", 1)

	span, ctx := re.a.GetSpan("Render", r)
	defer re.a.SpanStartEnd(span)()

	args, err := ParseMetricQuery(r)
	if err != nil {
		re.a.AddSpanError(span, err)
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
		return
	}
	re.spanLog(span, args)

	var data smetrics.RawRenderItems
	if re.a.rpcClient.ClusterMode() {
		data, err = re.getMetricsInClusterMode(ctx, args)
	} else {
		data, err = re.GetMetrics(ctx, args)
	}

	if err != nil {
		stats.StatsdClientSlow.Incr("reader.http.render.errors", 1)
		re.a.AddSpanError(span, err)
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}
	if data == nil {
		stats.StatsdClientSlow.Incr("reader.http.render.nodata", 1)
		re.a.AddSpanLog(span, "nodata", args.String())
		re.a.OutError(w, "No data found", http.StatusNoContent)
		return
	}

	for idx := range data {
		// graphite needs the "nils" and expects a "full list" to match the step + start/end
		data[idx].Start = uint32(args.Start)
		data[idx].End = uint32(args.End)
		data[idx].Quantize()
	}
	render_data := re.ToGraphiteRender(data)

	re.a.AddToCache(data)

	stats.StatsdClientSlow.Incr("reader.http.render.ok", 1)
	re.a.OutOk(w, render_data, args.Format)
	return
}

func (re *MetricsAPI) RawRender(w http.ResponseWriter, r *http.Request) {

	defer stats.StatsdSlowNanoTimeFunc("reader.http.rawrender.get-time-ns", time.Now())
	stats.StatsdClientSlow.Incr("reader.http.rawrender.hits", 1)

	span, ctx := re.a.GetSpan("RawRender", r)
	defer re.a.SpanStartEnd(span)()

	args, err := ParseMetricQuery(r)
	if err != nil {
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
	}
	re.spanLog(span, args)

	var data smetrics.RawRenderItems
	if re.a.rpcClient.ClusterMode() {
		data, err = re.getMetricsInClusterMode(ctx, args)
	} else {
		data, err = re.GetMetrics(ctx, args)
	}

	if err != nil {
		stats.StatsdClientSlow.Incr("reader.http.rawrender.error", 1)
		re.a.AddSpanError(span, err)
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}

	if data == nil {
		stats.StatsdClientSlow.Incr("reader.http.rawrender.nodata", 1)
		re.a.AddSpanLog(span, "nodata", args.String())
		re.a.OutError(w, "No data found", http.StatusNoContent)
		return
	}

	// adds the gotten metrics to the internal read cache
	re.a.AddToCache(data)
	stats.StatsdClientSlow.Incr("reader.http.rawrender.ok", 1)

	re.a.OutOk(w, smetrics.RawRenderItems(data), args.Format)
	return
}

// RawRenderInCluster in cluster mode, do the rendering
func (re *MetricsAPI) RawRenderInCluster(w http.ResponseWriter, r *http.Request) {

	defer stats.StatsdSlowNanoTimeFunc("reader.http.cluster.rawrender.get-time-ns", time.Now())
	stats.StatsdClientSlow.Incr("reader.http.cluster.rawrender.hits", 1)

	span, ctx := re.a.GetSpan("RawRenderInCluster", r)
	defer re.a.SpanStartEnd(span)()

	args, err := ParseMetricQuery(r)
	if err != nil {
		re.a.AddSpanError(span, err)
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
	}
	re.spanLog(span, args)

	data, err := re.getRawMetricsInClusterMode(ctx, args)

	if err != nil {
		stats.StatsdClientSlow.Incr("reader.http.cluster.rawrender.error", 1)
		re.a.AddSpanError(span, err)
		re.a.OutError(w, fmt.Sprintf("%v", err), http.StatusServiceUnavailable)
		return
	}

	if data == nil {
		stats.StatsdClientSlow.Incr("reader.http.cluster.rawrender.nodata", 1)
		re.a.AddSpanLog(span, "nodata", args.String())
		re.a.OutError(w, "No data found", http.StatusNoContent)
		return
	}

	// adds the gotten metrics to the internal read cache
	stats.StatsdClientSlow.Incr("reader.http.cluster.rawrender.ok", 1)

	re.a.OutOk(w, data, args.Format)
	return
}
