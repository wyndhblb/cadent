package metrics

import (
	"github.com/opentracing/opentracing-go"
	"golang.org/x/net/context"
)

// TracerMetrics base tracer object for all metrics objects
type TracerMetrics struct {
	tracer opentracing.Tracer
}

// SetTracer set the trace object
func (r *TracerMetrics) SetTracer(t opentracing.Tracer) {
	r.tracer = t
}

// GetSpan get a span from the context
func (r *TracerMetrics) GetSpan(name string, ctx context.Context) (opentracing.Span, func()) {
	var parentCtx opentracing.SpanContext
	parentSpan := opentracing.SpanFromContext(ctx)
	if parentSpan != nil {
		parentCtx = parentSpan.Context()
	}

	if r.tracer == nil && parentSpan == nil {
		return parentSpan, func() {}
	}
	if r.tracer == nil {
		return parentSpan, func() {}
	}
	span := r.tracer.StartSpan(name, opentracing.ChildOf(parentCtx))
	span.SetTag("span.kind", "server")
	span.SetTag("component", "cadent-metrics")

	return span, func() { span.Finish() }
}
