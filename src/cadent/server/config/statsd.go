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

package config

import (
	"cadent/server/stats"
	statsd "github.com/wyndhblb/gostatsdclient"
	"strings"
	"time"
)

var memStats *stats.MemStats

type StatsdConfig struct {
	// send some stats to the land
	StatsdServer          string  `toml:"server" json:"server,omitempty" yaml:"server"`
	StatsdPrefix          string  `toml:"prefix" json:"prefix,omitempty" yaml:"prefix"`
	StatsdInterval        uint    `toml:"interval" json:"interval,omitempty" yaml:"interval"`
	StatsdSampleRate      float32 `toml:"sample_rate" json:"sample_rate,omitempty" yaml:"sample_rate"`
	StatsdTimerSampleRate float32 `toml:"timer_sample_rate" json:"timer_sample_rate,omitempty" yaml:"timer_sample_rate"`
	NoIncludeHost         bool    `toml:"do_not_include_host" json:"do_not_include_host,omitempty" yaml:"do_not_include_host"`
	UseShortHostname      bool    `toml:"use_short_hostname" json:"use_short_hostname,omitempty" yaml:"use_short_hostname"`
}

//init a statsd client from our config object
func (cfg *StatsdConfig) Start() {

	if len(cfg.StatsdServer) == 0 {
		log.Notice("Skipping Statsd setup, no server specified")
		stats.StatsdClient = new(statsd.StatsdNoop)
		stats.StatsdClientSlow = new(statsd.StatsdNoop)
		return
	}
	interval := time.Second * 2 // aggregate stats and flush every 2 seconds
	perf := cfg.StatsdPrefix + ".%HOST%."
	if cfg.NoIncludeHost {
		perf = cfg.StatsdPrefix + "."
	}

	if cfg.StatsdServer == "echo" {
		stats.StatsdClient = statsd.StatsdEcho{}
		stats.StatsdClientSlow = statsd.StatsdEcho{} // slow does not have sample rates enabled
		log.Notice("Statsd Echo Fast Client to %s, prefix %s, interval %d", cfg.StatsdServer, cfg.StatsdPrefix, cfg.StatsdInterval)
		log.Notice("Statsd Echo Slow Client to %s, prefix %s, interval %d", cfg.StatsdServer, cfg.StatsdPrefix, cfg.StatsdInterval)
		return
	}

	// add udp if not there
	if !strings.HasPrefix(cfg.StatsdServer, "udp://") && !strings.HasPrefix(cfg.StatsdServer, "tcp://") {
		cfg.StatsdServer = "udp://" + cfg.StatsdServer
	}

	hostn := "long"
	if cfg.UseShortHostname {
		hostn = "short"
	}
	statsdclient := statsd.NewStatsdClient(cfg.StatsdServer, perf, hostn)
	statsdclientslow := statsd.NewStatsdClient(cfg.StatsdServer, perf, hostn)

	if cfg.StatsdTimerSampleRate > 0 {
		statsdclient.TimerSampleRate = cfg.StatsdTimerSampleRate
	}
	if cfg.StatsdSampleRate > 0 {
		statsdclient.SampleRate = cfg.StatsdSampleRate
	}
	//statsdclient.CreateSocket()
	//StatsdClient = statsdclient
	//return StatsdClient

	// the buffer client seems broken for some reason
	if cfg.StatsdInterval > 0 {
		interval = time.Second * time.Duration(cfg.StatsdInterval)
	}
	statsder := statsd.NewStatsdBuffer("fast", interval, statsdclient)
	statsderslow := statsd.NewStatsdBuffer("slow", interval, statsdclientslow)
	statsder.RetainKeys = true //retain statsd keys to keep emitting 0's
	if cfg.StatsdTimerSampleRate > 0 {
		statsder.TimerSampleRate = cfg.StatsdTimerSampleRate
	}
	if cfg.StatsdSampleRate > 0 {
		statsder.SampleRate = cfg.StatsdSampleRate
	}

	stats.StatsdClient = statsder
	stats.StatsdClientSlow = statsderslow // slow does not have sample rates enabled
	log.Notice("Statsd Fast Client to %s, prefix %s, interval %d", cfg.StatsdServer, cfg.StatsdPrefix, cfg.StatsdInterval)
	log.Notice("Statsd Slow Client to %s, prefix %s, interval %d", cfg.StatsdServer, cfg.StatsdPrefix, cfg.StatsdInterval)

	// start the MemTicker
	memStats = new(stats.MemStats)
	memStats.Start()
	return
}
