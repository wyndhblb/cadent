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

// Metrics main RPC client

package client

import (
	"cadent/server/schemas/api"
	"cadent/server/schemas/indexer"
	"cadent/server/schemas/metrics"
	"cadent/server/schemas/repr"
	"cadent/server/schemas/series"
	"cadent/server/writers/rpc/pb"
	"cadent/server/writers/rpc/server"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/op/go-logging.v1"
	"io"
	"time"
)

const MAX_BACKOFF_DELAY = 10 // num seconds max backoff

// MetricClient for gRPC
type MetricClient struct {
	cl     pb.CadentMetricClient
	log    *logging.Logger
	Tracer opentracing.Tracer
}

// New MetricClient
func New(serverAddr string, certFile *string, serverHost string, tracer opentracing.Tracer) (*MetricClient, error) {
	m := new(MetricClient)

	m.log = logging.MustGetLogger("client.grpc")

	var opts []grpc.DialOption
	if certFile != nil {
		var creds credentials.TransportCredentials
		var err error
		_, err = credentials.NewClientTLSFromFile(*certFile, serverHost)
		if err != nil {
			return nil, err
		}
		creds = credentials.NewClientTLSFromCert(nil, serverHost)
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	opts = append(opts, grpc.WithUserAgent("Cadent gPRC client/1.0"))
	opts = append(opts, grpc.WithBackoffMaxDelay(time.Duration(MAX_BACKOFF_DELAY*time.Second)))

	// opentracing attachment
	if tracer != nil {
		opts = append(opts,
			grpc.WithUnaryInterceptor(otgrpc.OpenTracingClientInterceptor(tracer)),
		)
	}

	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		return nil, err
	}

	m.cl = pb.NewCadentMetricClient(conn)
	return m, nil
}

func (m *MetricClient) getSpan(name string, ctx context.Context) (opentracing.Span, func()) {
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
	span.SetTag("span.kind", "client")
	span.SetTag("component", "cadent-grpc-client")

	return span, func() { span.Finish() }
}

// PutStatRepr push a metric into the base writer
func (m *MetricClient) PutStatRepr(ctx context.Context, s *repr.StatRepr) error {
	_, closer := m.getSpan("PutStatRepr", ctx)
	defer closer()

	_, err := m.cl.PutStatRepr(ctx, s)
	if err != nil {
		m.log.Errorf("Failure in PutStatRepr %v %v", m.cl, err)
		return err
	}

	return nil
}

// PutRawMetric push a raw metric into the base writer
func (m *MetricClient) PutRawMetric(ctx context.Context, s *series.RawMetric) error {
	_, closer := m.getSpan("PutRawMetric", ctx)
	defer closer()

	_, err := m.cl.PutRawMetric(ctx, s)
	if err != nil {
		m.log.Errorf("Failure in PutRawMetric %v %v", m.cl, err)
		return err
	}

	return nil
}

// GetMetrics get metrics based on an api query
func (m *MetricClient) GetMetrics(ctx context.Context, q *api.MetricQuery) ([]*metrics.RawRenderItem, error) {
	_, closer := m.getSpan("GetMetrics", ctx)
	defer closer()

	stream, err := m.cl.GetMetrics(ctx, q)
	if err != nil {
		m.log.Errorf("Failure in GetMetircs %v %v", m.cl, err)
		return nil, err
	}
	mets := make([]*metrics.RawRenderItem, 0)
	for {
		met, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			m.log.Errorf("Failure in GetMetircs %v %v", m.cl, err)
			return mets, err
		}
		mets = append(mets, met)
	}
	return mets, nil
}

// GetMetricsChan get metrics based on an api query but "stream" them back into a channel
// good for large queries over a large metric space as we can use the protobuf stream api
func (m *MetricClient) GetMetricsChan(ctx context.Context, q *api.MetricQuery, out chan<- *metrics.RawRenderItem, finished chan<- interface{}) error {
	_, closer := m.getSpan("GetMetricsChan", ctx)
	defer closer()

	stream, err := m.cl.GetMetrics(ctx, q)
	if err != nil {
		m.log.Errorf("Failure in GetMetricsChan %v %v", m.cl, err)
		return err
	}
	for {
		met, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			m.log.Errorf("Failure in GetMetricsChan %v %v", m.cl, err)
			return err
		}
		out <- met
	}
	finished <- true
	return nil
}

// GetMetricsInCache get metrics based from the server that are in the cache
func (m *MetricClient) GetMetricsInCache(ctx context.Context, q *api.MetricQuery) ([]*metrics.RawRenderItem, error) {
	_, closer := m.getSpan("GetMetricsInCache", ctx)
	defer closer()

	stream, err := m.cl.GetMetricsInCache(ctx, q)
	if err != nil {
		m.log.Errorf("Failure in GetMetricsInCache %v %v", m.cl, err)
		return nil, err
	}
	mets := make([]*metrics.RawRenderItem, 0)
	for {
		met, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			m.log.Errorf("Failure in GetMetricsInCache %v %v", m.cl, err)
			return mets, err
		}
		mets = append(mets, met)
	}
	return mets, nil
}

// GetMetricsInCacheChan get metrics based on an api query but "stream" them back into a channel
// good for large queries over a large metric space as we can use the protobuf stream api
func (m *MetricClient) GetMetricsInCacheChan(ctx context.Context, q *api.MetricQuery, out chan<- *metrics.RawRenderItem, finished chan<- interface{}) error {
	_, closer := m.getSpan("GetMetricsInCacheChan", ctx)
	defer closer()

	stream, err := m.cl.GetMetricsInCache(ctx, q)
	if err != nil {
		m.log.Errorf("Failure in GetMetricsInCache %v %v", m.cl, err)
		return err
	}
	for {
		met, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			m.log.Errorf("Failure in GetMetricsChan %v %v", m.cl, err)
			return err
		}
		out <- met
	}
	finished <- true
	return nil
}

// FindMetrics find metrics based on an api query
func (m *MetricClient) FindMetrics(ctx context.Context, q *api.IndexQuery) ([]*indexer.MetricFindItem, error) {
	_, closer := m.getSpan("FindMetrics", ctx)
	defer closer()

	stream, err := m.cl.Find(context.Background(), q)
	if err != nil {
		m.log.Errorf("Failure in FindMetrics %v %v", m.cl, err)
		return nil, err
	}
	mets := make([]*indexer.MetricFindItem, 0)
	for {
		met, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			m.log.Errorf("Failure in FindMetrics %v %v", m.cl, err)
			return mets, err
		}
		mets = append(mets, met)
	}
	return mets, nil
}

// FindMetricsChan find metrics based on an api query but "stream" them back into a channel
// good for large queries over a large metric space as we can use the protobuf stream api
func (m *MetricClient) FindMetricsChan(ctx context.Context, q *api.IndexQuery, out chan<- *indexer.MetricFindItem, finished chan<- interface{}) error {
	_, closer := m.getSpan("FindMetricsChan", ctx)
	defer closer()

	stream, err := m.cl.Find(context.Background(), q)
	if err != nil {
		m.log.Errorf("Failure in FindMetricsChan %v %v", m.cl, err)
		return err
	}
	for {
		met, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			m.log.Errorf("Failure in FindMetricsChan %v %v", m.cl, err)
			return err
		}
		out <- met
	}
	finished <- true
	return nil
}

// FindMetricsInCache find metrics based on an api query that are in the local caches
func (m *MetricClient) FindMetricsInCache(ctx context.Context, q *api.IndexQuery) ([]*indexer.MetricFindItem, error) {
	_, closer := m.getSpan("FindMetricsInCache", ctx)
	defer closer()

	stream, err := m.cl.FindInCache(ctx, q)
	if err != nil {
		m.log.Errorf("Failure in FindMetricsInCache %v %v", m.cl, err)
		return nil, err
	}
	mets := make([]*indexer.MetricFindItem, 0)
	for {
		met, err := stream.Recv()
		if err == io.EOF {
			break
		}
		// 404 is a-ok here
		if err == server.ErrorNoData {
			return mets, nil
		}
		if err != nil {
			m.log.Errorf("Failure in FindMetricsInCache %v %v", m.cl, err)
			return mets, err
		}
		mets = append(mets, met)
	}
	return mets, nil
}

// FindMetricsInCacheChan find metrics based on an api query but "stream" them back into a channel
// good for large queries over a large metric space as we can use the protobuf stream api
func (m *MetricClient) FindMetricsInCacheChan(ctx context.Context, q *api.IndexQuery, out chan<- *indexer.MetricFindItem, finished chan<- interface{}) error {
	_, closer := m.getSpan("FindMetricsInCacheChan", ctx)
	defer closer()

	stream, err := m.cl.FindInCache(ctx, q)
	if err != nil {
		m.log.Errorf("Failure in FindMetricsInCacheChan %v %v", m.cl, err)
		return err
	}
	for {
		met, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			m.log.Errorf("Failure in FindMetricsInCacheChan %v %v", m.cl, err)
			return err
		}
		out <- met
	}
	finished <- true
	return nil
}

// List of paths on the server
func (m *MetricClient) List(ctx context.Context, q *api.IndexQuery) (mets []*indexer.MetricFindItem, err error) {
	_, closer := m.getSpan("List", ctx)
	defer closer()

	stream, err := m.cl.List(ctx, q)
	if err != nil {
		m.log.Errorf("Failure in List %v %v", m.cl, err)
		return nil, err
	}
	for {
		met, err := stream.Recv()
		if err == io.EOF {
			break
		}
		// 404 is a-ok here
		if err == server.ErrorNoData {
			return mets, nil
		}
		if err != nil {
			m.log.Errorf("Failure in List %v %v", m.cl, err)
			return mets, err
		}
		mets = append(mets, met)
	}
	return mets, nil
}

// ListChan paths on a servier but "stream" them back into a channel
// good for large queries over a large metric space as we can use the protobuf stream api
func (m *MetricClient) ListChan(ctx context.Context, q *api.IndexQuery, out chan<- *indexer.MetricFindItem, finished chan<- interface{}) error {
	_, closer := m.getSpan("ListChan", ctx)
	defer closer()

	stream, err := m.cl.List(ctx, q)
	if err != nil {
		m.log.Errorf("Failure in ListChan %v %v", m.cl, err)
		return err
	}
	for {
		met, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			m.log.Errorf("Failure in ListChan %v %v", m.cl, err)
			return err
		}
		out <- met
	}
	finished <- true
	return nil
}
