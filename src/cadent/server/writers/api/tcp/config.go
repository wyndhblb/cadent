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
   Fire up an TCP server for an http interface to the

   metrics/indexer interfaces

   example config

   [graphite-proxy-map.accumulator.tcpapi]
        listen = "0.0.0.0:8083"

        use_metrics="name-of-main-writer"
        use_indexer="name-of-main-indexer"

        -OR-
            [graphite-proxy-map.accumulator.tcpapi.metrics]
            driver = "whisper"
            dsn = "/data/graphite/whisper"

            [graphite-proxy-map.accumulator.tcpapi.indexer]
            driver = "leveldb"
            dsn = "/data/graphite/idx"

*/

package tcp

import (
	"cadent/server/utils/options"
	"cadent/server/writers"
	"cadent/server/writers/indexer"
	"cadent/server/writers/metrics"
	"fmt"
)

type TCPApiMetricConfig struct {
	Driver   string          `toml:"driver" json:"driver,omitempty"`
	DSN      string          `toml:"dsn"  json:"dsn,omitempty"`
	UseCache string          `toml:"cache"  json:"cache,omitempty"`
	Options  options.Options `toml:"options"  json:"options,omitempty"`
}

type TCPApiIndexerConfig struct {
	Driver  string          `toml:"driver"  json:"driver,omitempty"`
	DSN     string          `toml:"dsn"  json:"dsn,omitempty"`
	Options options.Options `toml:"options"  json:"options,omitempty"`
}

// ApiConfig each writer for both metrics and indexers should get the http API attached to it
type TCPApiConfig struct {
	Listen         string              `toml:"listen"  json:"listen,omitempty"`
	Logfile        string              `toml:"log_file"  json:"log-file,omitempty"`
	MaxClients     int64               `toml:"max_clients"  json:"max-clients,omitempty"`
	NumWorkers     int64               `toml:"num_workers"  json:"num-workers,omitempty"`
	TLSKeyPath     string              `toml:"key"  json:"key,omitempty"`
	TLSCertPath    string              `toml:"cert"  json:"cert,omitempty"`
	UseMetrics     string              `toml:"use_metrics"  json:"use-metrics,omitempty"`
	UseIndexer     string              `toml:"use_indexer"  json:"use-indexer,omitempty"`
	MetricOptions  TCPApiMetricConfig  `toml:"metrics"  json:"metrics,omitempty"`
	IndexerOptions TCPApiIndexerConfig `toml:"indexer"  json:"indexer,omitempty"`
}

// SoloApiConfig for when you just want to run the API only (i.e. a reader only node)
type SoloTCPApiConfig struct {
	TCPApiConfig
	Seed        string  `toml:"seed" json:"seed,omitempty"` // only used in "api only" mode
	Resolutions []int64 `toml:"resolutions" json:"resolutions,omitempty"`
}

func (s *SoloTCPApiConfig) GetApiConfig() TCPApiConfig {
	return s.TCPApiConfig
}

// GetMetrics creates a new metrics object
func (re *TCPApiConfig) GetMetrics(resolution float64) (metrics.Metrics, error) {
	if len(re.UseMetrics) > 0 {
		gots := metrics.GetMetrics(re.UseMetrics)
		if gots == nil {
			return nil, fmt.Errorf("Could not find %s in the metric registry", re.UseMetrics)
		}
		return gots, nil
	}

	reader, err := metrics.NewWriterMetrics(re.MetricOptions.Driver)
	if err != nil {
		return nil, err
	}
	if re.MetricOptions.Options == nil {
		re.MetricOptions.Options = options.New()
	}
	re.MetricOptions.Options.Set("dsn", re.MetricOptions.DSN)
	re.MetricOptions.Options.Set("resolution", resolution)

	// need to match caches
	// use the defined cacher object
	if len(re.MetricOptions.UseCache) == 0 {
		return nil, writers.ErrCacheOptionRequired
	}

	// find the proper cache to use
	res := uint32(resolution)
	proper_name := fmt.Sprintf("%s:%d", re.MetricOptions.UseCache, res)
	c, err := metrics.GetCacherSingleton(proper_name, metrics.CacheNeededName(re.MetricOptions.Driver))
	if err != nil {
		return nil, err
	}
	re.MetricOptions.Options.Set("cache", c)

	err = reader.Config(&re.MetricOptions.Options)

	if err != nil {
		return nil, err
	}
	return reader, nil
}

// GetIndexer creates a new indexer object
func (re *TCPApiConfig) GetIndexer() (indexer.Indexer, error) {
	if len(re.UseIndexer) > 0 {
		gots := indexer.GetIndexer(re.UseIndexer)
		if gots == nil {
			return nil, fmt.Errorf("Could not find %s in the indexer registry", re.UseIndexer)
		}
		return gots, nil
	}

	idx, err := indexer.NewIndexer(re.IndexerOptions.Driver)
	if err != nil {
		return nil, err
	}
	if re.IndexerOptions.Options == nil {
		re.IndexerOptions.Options = options.New()
	}
	re.IndexerOptions.Options.Set("dsn", re.IndexerOptions.DSN)
	err = idx.Config(&re.IndexerOptions.Options)
	if err != nil {
		return nil, err
	}
	return idx, nil
}
