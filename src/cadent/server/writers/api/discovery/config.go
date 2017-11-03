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
   Config bits for the Discover module


   example config
   [graphite-proxy-map.accumulator.api]
        base_path = "/graphite/"
       	listen = "0.0.0.0:8083"
        enable_gzip = true # default is FALSE here for GC issues on high volume
        key = "/path/to/key
        cert = "/path/to/cert"

	[graphite-proxy-map.accumulator.api.discover]
	driver="zookeeper"
	dsn="host:2181,host:2181..."

	[graphite-proxy-map.accumulator.api.discover.options]
	    root_path="/cadent"
	    api_path="/api"

        [graphite-proxy-map.accumulator.api.metrics]
            driver = "whisper"
            dsn = "/data/graphite/whisper"

            # this is the read cache that will keep the latest goods in ram
            read_cache_max_items=102400
            read_cache_max_bytes_per_metric=8192

        [graphite-proxy-map.accumulator.api.indexer]
            driver = "leveldb"
            dsn = "/data/graphite/idx"
*/

package discovery

import (
	"cadent/server/utils/options"
)

type DiscoverConfig struct {
	Driver  string          `toml:"driver" json:"driver,omitempty" yaml:"driver"`
	DSN     string          `toml:"dsn"  json:"dsn,omitempty"  yaml:"dsn"`
	Name    string          `toml:"name" json:"name,omitempty"  yaml:"name"`
	Options options.Options `toml:"options"  json:"options,omitempty" yaml:"options"`
}

// New discover object
func (re *DiscoverConfig) New() (Discover, error) {
	d, err := NewDiscover(re.Driver)
	if err != nil {
		return nil, err
	}
	if re.Options == nil {
		re.Options = options.New()
	}
	re.Options.Set("dsn", re.DSN)
	if len(re.Name) > 0 {
		re.Options.Set("name", re.Name)
	}

	err = d.Config(re.Options)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// New discover object
func (re *DiscoverConfig) NewLister() (Lister, error) {
	d, err := NewDiscoverList(re.Driver)
	if err != nil {
		return nil, err
	}
	if re.Options == nil {
		re.Options = options.New()
	}
	re.Options.Set("dsn", re.DSN)
	err = d.Config(re.Options)
	if err != nil {
		return nil, err
	}
	return d, nil
}
