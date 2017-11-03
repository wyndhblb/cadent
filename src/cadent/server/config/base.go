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
)

type BaseConfig struct {
	System  SystemConfig  `toml:"system" json:"system,omitempty" yaml:"system"`
	Logger  LogConfig     `toml:"log" json:"log,omitempty" yaml:"log"`
	Profile ProfileConfig `toml:"profile" json:"profile,omitempty" yaml:"profile"`
	Statsd  StatsdConfig  `toml:"statsd" json:"statsd,omitempty" yaml:"statsd"`
	Health  HealthConfig  `toml:"health" json:"health,omitempty" yaml:"health"`
	Gossip  GossipConfig  `toml:"gossip" json:"gossip,omitempty" yaml:"gossip"`
}

func ParseConfigFile(filename string) (cfg *BaseConfig, err error) {
	cfg = new(BaseConfig)
	if _, err := tomlenv.DecodeFile(filename, cfg); err != nil {
		log.Critical("Error decoding config file: %s", err)
		return nil, err
	}
	return cfg, nil
}

func (c *BaseConfig) BaseStart() error {
	c.Logger.Start()
	c.System.Start()
	c.Statsd.Start()
	c.Profile.Start()
	c.Gossip.Start()
	return nil
}
