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

// Metrics main RPC endpoint

package server

import (
	"cadent/server/schemas/api"
	"cadent/server/schemas/repr"
	"cadent/server/schemas/series"
	"cadent/server/stats"
	"cadent/server/writers/indexer"
	"cadent/server/writers/metrics"
	"cadent/server/writers/rpc/pb"
	"errors"
	"github.com/opentracing/opentracing-go"
	context "golang.org/x/net/context"
	"time"
)

var ErrorNoData = errors.New("No data found")
var noResp = &pb.NoResponse{}

// MetricServer match the rpc interface CadentMetric
type MetricServer struct {
	Metrics metrics.Metrics
	Indexer indexer.Indexer
	Tracer  opentracing.Tracer
}

func (m *MetricServer) getSpan(name string, ctx context.Context) (opentracing.Span, func()) {
	var parentCtx opentracing.SpanContext
	parentSpan := opentracing.SpanFromContext(ctx)
	if parentSpan != nil {
		parentCtx = parentSpan.Context()
	}

	if m.Tracer == nil && parentSpan == nil {
		return parentSpan, func() {}
	}
	if m.Tracer == nil {
		return parentSpan, func() {}
	}
	span := m.Tracer.StartSpan(name, opentracing.ChildOf(parentCtx))
	span.SetTag("span.kind", "server")
	span.SetTag("component", "cadent-grpc-server")
	return span, func() { span.Finish() }
}

// GetMetrics get the series
func (m *MetricServer) GetMetrics(q *api.MetricQuery, stream pb.CadentMetric_GetMetricsServer) error {
	sp, closer := m.getSpan("GetMetrics", stream.Context())
	sp.LogKV("target", q.Target, "start", q.Start, "end", q.End, "tags", q.Tags, "step", q.Step)
	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("rpc.getmetrics.get-time-ns", time.Now())
	stats.StatsdClientSlow.Incr("rpc.getmetrics.hits", 1)

	data, err := m.Metrics.RawRender(stream.Context(), q.Target, q.Start, q.End, q.Tags, q.Step)

	if err != nil {
		stats.StatsdClientSlow.Incr("rpc.getmetrics.error", 1)
		return err
	}

	if data == nil {
		stats.StatsdClientSlow.Incr("rpc.getmetrics.nodata", 1)
		return ErrorNoData
	}

	for _, d := range data {
		if err := stream.Send(d); err != nil {
			return err
		}
	}

	stats.StatsdClientSlow.Incr("rpc.getmetrics.ok", 1)

	return nil
}

// GetMetricsInCache get the series that are in the cache only (don't hit the backend DB)
func (m *MetricServer) GetMetricsInCache(q *api.MetricQuery, stream pb.CadentMetric_GetMetricsInCacheServer) error {
	ctx := stream.Context()
	sp, closer := m.getSpan("GetMetricsInCache", ctx)
	sp.LogKV("target", q.Target, "start", q.Start, "end", q.End, "tags", q.Tags)
	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("rpc.getmetricsincache.get-time-ns", time.Now())
	stats.StatsdClientSlow.Incr("rpc.getmetricsincache.hits", 1)

	data, err := m.Metrics.CacheRender(ctx, q.Target, q.Start, q.End, q.Tags)

	if err != nil {
		stats.StatsdClientSlow.Incr("rpc.getmetricsincache.error", 1)
		return err
	}

	if data == nil {
		stats.StatsdClientSlow.Incr("rpc.getmetricsincache.nodata", 1)
		return ErrorNoData
	}

	for _, d := range data {
		if d.UsingCache {
			if d.InCache {
				if err := stream.Send(d); err != nil {
					return err
				}
			}
		} else {
			if err := stream.Send(d); err != nil {
				return err
			}
		}
	}

	stats.StatsdClientSlow.Incr("rpc.getmetricsincache.ok", 1)

	return nil
}

// Find metrics keys and metadata
func (m *MetricServer) Find(q *api.IndexQuery, stream pb.CadentMetric_FindServer) error {
	ctx := stream.Context()
	sp, closer := m.getSpan("Find", ctx)
	sp.LogKV("query", q.Query, "tags", q.Tags)
	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("rpc.find.get-time-ns", time.Now())
	stats.StatsdClientSlow.Incr("rpc.find.hits", 1)

	stream.Context()

	data, err := m.Indexer.Find(ctx, q.Query, q.Tags)

	if err != nil {
		stats.StatsdClientSlow.Incr("rpc.find.error", 1)
		return err
	}

	// no data is ok it will happen ALOT
	if data == nil {
		stats.StatsdClientSlow.Incr("rpc.find.nodata", 1)
		return nil
	}

	for _, d := range data {
		if err := stream.Send(d); err != nil {
			return err
		}
	}

	stats.StatsdClientSlow.Incr("rpc.find.ok", 1)

	return nil
}

// FindInCache an alias to Find at the moment may change
func (m *MetricServer) FindInCache(q *api.IndexQuery, stream pb.CadentMetric_FindInCacheServer) error {
	sp, closer := m.getSpan("FindInCache", stream.Context())
	sp.LogKV("query", q.Query, "tags", q.Tags)
	defer closer()
	return m.Find(q, stream)
}

// Find metrics keys and metadata
func (m *MetricServer) List(q *api.IndexQuery, stream pb.CadentMetric_ListServer) error {
	sp, closer := m.getSpan("List", stream.Context())
	sp.LogKV("query", q.Query, "tags", q.Tags)
	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("rpc.list.get-time-ns", time.Now())
	stats.StatsdClientSlow.Incr("rpc.list.hits", 1)

	data, err := m.Indexer.List(q.HasData, int(q.Page))

	if err != nil {
		stats.StatsdClientSlow.Incr("rpc.list.error", 1)
		return err
	}

	// no data is ok it will happen ALOT
	if data == nil {
		stats.StatsdClientSlow.Incr("rpc.list.nodata", 1)
		return nil
	}

	for _, d := range data {
		if err := stream.Send(d); err != nil {
			return err
		}
	}

	stats.StatsdClientSlow.Incr("rpc.list.ok", 1)

	return nil
}

// PutRawMetric add a raw metric into the writer
func (m *MetricServer) PutRawMetric(ctx context.Context, q *series.RawMetric) (*pb.NoResponse, error) {
	sp, closer := m.getSpan("PutRawMetric", ctx)
	sp.LogKV("metric", q.Metric, "tags", q.Tags, "metatags", q.MetaTags)
	defer closer()

	defer stats.StatsdNanoTimeFunc("rpc.putraw.get-time-ns", time.Now())
	stats.StatsdClient.Incr("rpc.putraw.hits", 1)

	stat := &repr.StatRepr{
		Name:  &repr.StatName{Key: q.Metric, Tags: q.Tags, MetaTags: q.MetaTags},
		Count: 1,
		Sum:   q.Value,
		Time:  q.Time,
	}
	err := m.Metrics.Write(stat)

	if err != nil {
		stats.StatsdClient.Incr("rpc.putraw.error", 1)
		return noResp, err
	}

	stats.StatsdClient.Incr("rpc.putraw.ok", 1)

	return noResp, nil
}

// PutStatRepr add a metric metric into the writer
func (m *MetricServer) PutStatRepr(ctx context.Context, stat *repr.StatRepr) (*pb.NoResponse, error) {
	sp, closer := m.getSpan("PutStatRepr", ctx)
	sp.LogKV(
		"metric", stat.Name.Name(),
		"tags", stat.Name.Tags,
		"metatags", stat.Name.MetaTags,
		"uid", stat.Name.UniqueIdString(),
	)
	defer closer()

	defer stats.StatsdNanoTimeFunc("rpc.putrepr.get-time-ns", time.Now())
	stats.StatsdClient.Incr("rpc.putraw.hits", 1)

	err := m.Metrics.Write(stat)

	if err != nil {
		stats.StatsdClient.Incr("rpc.putrepr.error", 1)
		return noResp, err
	}

	stats.StatsdClient.Incr("rpc.putrepr.ok", 1)

	return noResp, nil
}
