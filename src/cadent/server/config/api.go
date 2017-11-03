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

/** API only config elements **/

package config

import (
	"cadent/server/utils/tomlenv"
	"cadent/server/writers/api/solo"
)

type ApiOnlyConfig struct {
	BaseConfig

	Api solo.SoloApiConfig `toml:"api" json:"api,omitempty" yaml:"api"`

	runnapi *solo.SoloApiLoop
}

func ParseApiConfigFile(filename string) (cfg *ApiOnlyConfig, err error) {
	cfg = new(ApiOnlyConfig)
	if _, err := tomlenv.DecodeFile(filename, cfg); err != nil {
		log.Critical("Error decoding config file: %s", err)
		return nil, err
	}
	return cfg, nil
}

func (c *ApiOnlyConfig) Start() error {
	err := c.BaseStart()
	if err != nil {
		return err
	}

	c.runnapi = solo.NewSoloApiLoop()
	err = c.runnapi.Config(c.Api)
	if err != nil {
		return err
	}
	c.runnapi.Start()

	return nil
}
