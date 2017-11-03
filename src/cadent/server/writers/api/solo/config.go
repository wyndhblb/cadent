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
   Config parts for Solo APi mode

   example config
   [api]
        base_path = "/graphite/"
        listen = "0.0.0.0:8084"
        seed = "http://localhost:8083/graphite"
        #and / OR (it will get the resolutions from the seed
        resolutions = [int, int, int]

            [api.metrics]
            driver = "whisper"
            dsn = "/data/graphite/whisper"

            # this is the read cache that will keep the latest goods in ram
            read_cache_max_items=102400
            read_cache_max_bytes_per_metric=8192

            [api.indexer]
            driver = "leveldb"
            dsn = "/data/graphite/idx"

    or in discovery mode
       [api]
        base_path = "/graphite/"
        listen = "0.0.0.0:8084"

        #and / OR (it will get the resolutions from the seed
        resolutions = [int, int, int]

	    [api.discover]
            driver="zookeeper"
            dsn="127.0.0.1:2181"

            [api.metrics]
            driver = "whisper"
            dsn = "/data/graphite/whisper"

            # this is the read cache that will keep the latest goods in ram
            read_cache_max_items=102400
            read_cache_max_bytes_per_metric=8192

            [api.indexer]
            driver = "leveldb"
            dsn = "/data/graphite/idx"
*/

package solo

import (
	api "cadent/server/writers/api/http"
)

// SoloApiConfig for when you just want to run the API only (i.e. a reader only node)
type SoloApiConfig struct {
	api.ApiConfig
	Seed        string  `toml:"seed" json:"seed,omitempty" yaml:"seed"` // if not in discover mode, use the /info endpoint from a seed node
	Resolutions []int64 `toml:"resolutions" json:"resolutions,omitempty" yaml:"resolutions,omitempty"`
}

func (s *SoloApiConfig) GetApiConfig() api.ApiConfig {
	return s.ApiConfig
}
