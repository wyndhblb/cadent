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
   Fire up an HTTP server for an http interface to the

   metrics/indexer interfaces

   example config
   [graphite-proxy-map.accumulator.api]
        base_path = "/graphite/"
       	listen = "0.0.0.0:8083"
        enable_gzip = true # default is FALSE here for GC issues on high volume
        key = "/path/to/key
        cert = "/path/to/cert"

        zipkin_url = "http://path.to.zipkin:9411/api/v1/spans"
	zipkin_batch_size = 5

        use_metrics="name-of-main-writer"
        use_indexer="name-of-main-indexer"

*/

package http

import (
	"cadent/server/utils/options"
	"cadent/server/writers"
	"cadent/server/writers/api/discovery"
	"cadent/server/writers/indexer"
	"cadent/server/writers/metrics"
	"fmt"
	"github.com/lightstep/lightstep-tracer-go"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-client-go/transport/zipkin"
	"io"
	"time"
)

type ApiMetricConfig struct {
	Driver   string          `toml:"driver" json:"driver,omitempty" yaml:"driver"`
	DSN      string          `toml:"dsn"  json:"dsn,omitempty" yaml:"dsn"`
	UseCache string          `toml:"cache"  json:"cache,omitempty" yaml:"cache"`
	Options  options.Options `toml:"options"  json:"options,omitempty" yaml:"options"`
}

type ApiIndexerConfig struct {
	Driver  string          `toml:"driver"  json:"driver,omitempty" yaml:"driver"`
	DSN     string          `toml:"dsn"  json:"dsn,omitempty" yaml:"dsn"`
	Options options.Options `toml:"options"  json:"options,omitempty" yaml:"options"`
}

type ApiTracingConfig struct {
	ZipkinURL        string  `toml:"zipkin_url"  json:"zipkin-url,omitempty"  yaml:"zipkin_url"`
	JaegerHost       string  `toml:"jaeger_host"  json:"jaeger-host,omitempty"  yaml:"jaeger_host"`
	LightStepKey     string  `toml:"lightstep_key" json:"lightstep-key,omitempty"  yaml:"lightstep_key"`
	BatchSize        int     `toml:"batch_size"  json:"batch-size,omitempty"  yaml:"batch_size"`
	SamplesPerSecond float64 `toml:"samples_per_second"  json:"samples-per-second,omitempty"  yaml:"samples_per_second"`
	Name             string  `toml:"name"  json:"name,omitempty"  yaml:"name"`
}

// ApiConfig each writer for both metrics and indexers should get the http API attached to it
type ApiConfig struct {
	Listen                     string                   `toml:"listen"  json:"listen,omitempty" yaml:"listen"`
	GPRCListen                 string                   `toml:"grpc_listen" json:"grpc_listen,omitempty" yaml:"grpc_listen"`
	Logfile                    string                   `toml:"log_file"  json:"log-file,omitempty" yaml:"log_file"`
	AdvertiseName              string                   `toml:"advertise_name"  json:"advertise-name,omitempty" yaml:"advertise_name"`
	AdvertiseUrl               string                   `toml:"advertise_url"  json:"advertise-url,omitempty" yaml:"advertise_url"`
	BasePath                   string                   `toml:"base_path"  json:"base-path,omitempty" yaml:"base_path"`
	EnableGzip                 bool                     `toml:"enable_gzip"  json:"enable-gzip,omitempty" yaml:"enable_gzip"`
	CadentHosts                []string                 `toml:"cadent_hosts"  json:"cadent-hosts,omitempty" yaml:"cadent_hosts"`
	ClusterHost                string                   `toml:"advertise_host"  json:"advertise-host,omitempty" yaml:"advertise_host"`
	ClusterName                string                   `toml:"cluster_name"  json:"cluster-name,omitempty" yaml:"cluster_name"`
	TLSKeyPath                 string                   `toml:"key"  json:"key,omitempty" yaml:"key"`
	TLSCertPath                string                   `toml:"cert"  json:"cert,omitempty" yaml:"cert"`
	DiscoveryOptions           discovery.DiscoverConfig `toml:"discover"  json:"discover,omitempty" yaml:"discover"`
	ApiMetricOptions           ApiMetricConfig          `toml:"metrics"  json:"metrics,omitempty" yaml:"metrics"`
	ApiIndexerOptions          ApiIndexerConfig         `toml:"indexer"  json:"indexer,omitempty" yaml:"indexer"`
	MaxReadCacheBytes          int                      `toml:"read_cache_total_bytes"  json:"read-cache-total-bytes,omitempty" yaml:"read_cache_total_bytes"`
	MaxReadCacheBytesPerMetric int                      `toml:"read_cache_max_bytes_per_metric"  json:"read-cache-max-bytes-per-metric,omitempty" yaml:"read_cache_max_bytes_per_metric"`
	Tags                       string                   `toml:"tags"  json:"tags,omitempty" yaml:"tags"`
	UseMetrics                 string                   `toml:"use_metrics"  json:"use-metrics,omitempty" yaml:"use_metrics"`
	UseIndexer                 string                   `toml:"use_indexer"  json:"use-indexer,omitempty" yaml:"use_indexer"`
	TracingOptions             ApiTracingConfig         `toml:"tracing" json:"tracing,omitempty" yaml:"tracing"`
}

// GetDiscover creates a new discover registry object
func (re *ApiConfig) GetDiscover() (discovery.Discover, error) {
	if len(re.DiscoveryOptions.DSN) == 0 {
		return nil, nil
	}
	d, err := re.DiscoveryOptions.New()
	if err != nil {
		return nil, err
	}

	re.DiscoveryOptions.Options.Set("dsn", re.DiscoveryOptions.DSN)
	err = d.Config(re.DiscoveryOptions.Options)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// GetDiscover creates a new discover registry object
func (re *ApiConfig) GetDiscoverList() (discovery.Lister, error) {
	if len(re.DiscoveryOptions.DSN) == 0 {
		return nil, nil
	}
	d, err := re.DiscoveryOptions.NewLister()
	if err != nil {
		return nil, err
	}

	re.DiscoveryOptions.Options.Set("dsn", re.DiscoveryOptions.DSN)
	err = d.Config(re.DiscoveryOptions.Options)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// GetMetrics creates a new metrics object
func (re *ApiConfig) GetMetrics(resolution float64) (metrics.Metrics, error) {

	if len(re.UseMetrics) > 0 {
		gots := metrics.GetMetrics(re.UseMetrics)
		if gots == nil {
			return nil, fmt.Errorf("Could not find %s in the metric registry", re.UseMetrics)
		}
		return gots, nil
	}

	reader, err := metrics.NewWriterMetrics(re.ApiMetricOptions.Driver)
	if err != nil {
		return nil, err
	}
	if re.ApiMetricOptions.Options == nil {
		re.ApiMetricOptions.Options = options.New()
	}
	re.ApiMetricOptions.Options.Set("dsn", re.ApiMetricOptions.DSN)
	re.ApiMetricOptions.Options.Set("resolution", resolution)

	// need to match caches
	// use the defined cacher object
	if len(re.ApiMetricOptions.UseCache) == 0 {
		return nil, writers.ErrCacheOptionRequired
	}

	// find the proper cache to use
	res := uint32(resolution)
	proper_name := fmt.Sprintf("%s:%d", re.ApiMetricOptions.UseCache, res)
	c, err := metrics.GetCacherSingleton(proper_name, metrics.CacheNeededName(re.ApiMetricOptions.Driver))
	if err != nil {
		return nil, err
	}
	re.ApiMetricOptions.Options.Set("cache", c)

	err = reader.Config(&re.ApiMetricOptions.Options)

	if err != nil {
		return nil, err
	}
	return reader, nil
}

// GetIndexer creates a new indexer object
func (re *ApiConfig) GetIndexer() (indexer.Indexer, error) {
	if len(re.UseIndexer) > 0 {
		gots := indexer.GetIndexer(re.UseIndexer)
		if gots == nil {
			return nil, fmt.Errorf("Could not find %s in the indexer registry", re.UseIndexer)
		}
		return gots, nil
	}

	idx, err := indexer.NewIndexer(re.ApiIndexerOptions.Driver)
	if err != nil {
		return nil, err
	}
	if re.ApiIndexerOptions.Options == nil {
		re.ApiIndexerOptions.Options = options.New()
	}
	re.ApiIndexerOptions.Options.Set("dsn", re.ApiIndexerOptions.DSN)
	err = idx.Config(&re.ApiIndexerOptions.Options)
	if err != nil {
		return nil, err
	}
	return idx, nil
}

func (re *ApiConfig) GetTracer() (tracer opentracing.Tracer, closer io.Closer, err error) {

	// cannot have both lightstep and/or zipkin jaeger
	if len(re.TracingOptions.ZipkinURL) > 0 && len(re.TracingOptions.LightStepKey) > 0 {
		return nil, nil, fmt.Errorf("Cannot have both a `lightstep_key` and/or `zipkin_url` and/or `jaeger_host`")
	}
	// cannot have both lightstep and/or zipkin jaeger
	if len(re.TracingOptions.JaegerHost) > 0 && len(re.TracingOptions.LightStepKey) > 0 {
		return nil, nil, fmt.Errorf("Cannot have both a `lightstep_key` and/or `zipkin_url` and/or `jaeger_host`")
	}
	// cannot have both lightstep and/or zipkin jaeger
	if len(re.TracingOptions.JaegerHost) > 0 && len(re.TracingOptions.ZipkinURL) > 0 {
		return nil, nil, fmt.Errorf("Cannot have both a `lightstep_key` and/or `zipkin_url` and/or `jaeger_host`")
	}

	if len(re.TracingOptions.ZipkinURL) == 0 && len(re.TracingOptions.LightStepKey) == 0 && len(re.TracingOptions.JaegerHost) == 0 {
		return opentracing.NoopTracer{}, nil, nil
	}

	nm := re.TracingOptions.Name
	if len(nm) == 0 {
		nm = "cadent-api-server"
	}

	if len(re.TracingOptions.ZipkinURL) > 0 {

		batch := re.TracingOptions.BatchSize
		if batch <= 0 {
			batch = 5
		}

		// Jaeger tracer can be initialized with a transport that will
		// report tracing Spans to a Zipkin backend
		transport, err := zipkin.NewHTTPTransport(
			re.TracingOptions.ZipkinURL,
			zipkin.HTTPBatchSize(batch),
			zipkin.HTTPLogger(jaeger.StdLogger),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("Cannot initialize HTTP transport for tracing: %v", err)
		}

		// 0 means "sample all"
		samples := re.TracingOptions.SamplesPerSecond
		var sampler jaeger.Sampler
		if samples <= 0 {
			sampler = jaeger.NewConstSampler(true)
		} else {
			sampler = jaeger.NewRateLimitingSampler(samples)
		}

		// create Jaeger tracer
		tracer, closer = jaeger.NewTracer(
			nm,
			sampler,
			jaeger.NewRemoteReporter(transport),
		)
	}

	if len(re.TracingOptions.LightStepKey) > 0 {
		tracer = lightstep.NewTracer(lightstep.Options{
			AccessToken: re.TracingOptions.LightStepKey,
		})
	}

	if len(re.TracingOptions.JaegerHost) > 0 {
		batch := re.TracingOptions.BatchSize
		if batch <= 0 {
			batch = 5
		}

		// 0 means "sample all"
		samples := re.TracingOptions.SamplesPerSecond
		var sampler *jaegercfg.SamplerConfig
		if samples <= 0 {
			sampler = &jaegercfg.SamplerConfig{
				Type:  "const",
				Param: 1.0,
			}
		} else {
			sampler = &jaegercfg.SamplerConfig{
				Type:  "rateLimiting",
				Param: samples,
			}
		}

		cfg := jaegercfg.Configuration{
			Sampler: sampler,
			Reporter: &jaegercfg.ReporterConfig{
				LogSpans:            true,
				BufferFlushInterval: 1 * time.Second,
				LocalAgentHostPort:  re.TracingOptions.JaegerHost,
			},
		}

		// create Jaeger tracer
		tracer, closer, err = cfg.New(nm)
		if err != nil {
			return nil, nil, fmt.Errorf("Cannot initialize Jaeger transport for tracing: %v", err)
		}
	}

	opentracing.SetGlobalTracer(tracer)

	return tracer, closer, nil

}
