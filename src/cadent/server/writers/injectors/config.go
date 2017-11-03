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
   Injectors

   configuration elements looks very similar to the accumulator config (server/accumulator/config.go)
*/

package injectors

import (
	"cadent/server/utils/options"
	"cadent/server/utils/shared"
	"cadent/server/writers"
	httpapi "cadent/server/writers/api/http"
	tcpapi "cadent/server/writers/api/tcp"
	"fmt"
	"github.com/wyndhblb/go-utils/parsetime"
	"math"
	"strings"
	"time"
)

//ten years
const DEFAULT_TTL = 60 * 60 * 24 * 365 * 10 * time.Second

type InjectorConfig struct {
	Name      string               `toml:"name" json:"name,omitempty"`
	Driver    string               `toml:"driver" json:"driver,omitempty"`
	DSN       string               `toml:"dsn"  json:"dsn,omitempty"`
	Options   options.Options      `toml:"options"  json:"options,omitempty"`
	Writer    writers.WriterConfig `toml:"writer"  json:"writer"`
	Reader    httpapi.ApiConfig    `toml:"api"  json:"api"`       // http server for reading
	TCPReader tcpapi.TCPApiConfig  `toml:"tcpapi"  json:"tcpapi"` // tcp server for reading

	// Aggregate bins
	Times []string `toml:"times"  json:"times"`

	durations       []time.Duration
	ttls            []time.Duration
	resolutionArray [][]int
}

func (cf *InjectorConfig) ParseDurations() error {
	cf.durations = []time.Duration{}
	cf.ttls = []time.Duration{}

	if len(cf.Times) == 0 {
		cf.durations = []time.Duration{1 * time.Second}
		cf.ttls = []time.Duration{time.Duration(DEFAULT_TTL)}
		cf.resolutionArray = [][]int{{int(cf.durations[0].Seconds()), int(cf.ttls[0].Seconds())}}

	} else {
		lastKeeper := time.Duration(0)
		cf.resolutionArray = make([][]int, 0)
		var err error
		for _, st := range cf.Times {

			// can be "time:ttl" or just "time"
			spl := strings.Split(st, ":")
			dd := spl[0]
			var ttlDur time.Duration
			if len(spl) > 2 {
				return fmt.Errorf("Times can be `duration` or `duration:TTL` only")
			} else if len(spl) == 2 {
				ttl := spl[1]
				ttlDur, err = parsetime.ParseDuration(ttl)
				if err != nil {
					return err
				}
				cf.ttls = append(cf.ttls, ttlDur)
			} else {
				cf.ttls = append(cf.ttls, time.Duration(DEFAULT_TTL))
			}

			tDur, err := parsetime.ParseDuration(dd)
			if err != nil {
				return err
			}
			// need to be properly time ordered and MULTIPLES of each other
			if lastKeeper.Seconds() > 0 {
				if tDur.Seconds() < lastKeeper.Seconds() {
					return fmt.Errorf("Times need to be in smallest to longest order")
				}
				if math.Mod(float64(tDur.Seconds()), float64(lastKeeper.Seconds())) != 0.0 {
					return fmt.Errorf("Times need to be in multiples of themselves")
				}
			}
			lastKeeper = tDur
			cf.durations = append(cf.durations, tDur)
			cf.resolutionArray = append(cf.resolutionArray, []int{int(tDur.Seconds()), int(ttlDur.Seconds())})
		}
	}
	if len(cf.durations) <= 0 {
		return fmt.Errorf("Accumulator: No Times/Durations given, cannot proceed")
	}
	return nil
}

// New Injector from the config
func (c *InjectorConfig) New() (Injector, error) {

	if len(c.Options) == 0 {
		c.Options = options.New()
	}
	c.Options.Set("dsn", c.DSN)

	switch c.Driver {
	case "kafka":
		err := c.ParseDurations()
		if err != nil {
			return nil, err
		}
		kf := NewKafka(c.Name)
		err = kf.Config(c.Options)
		if err != nil {
			return nil, err
		}
		err = c.SetWritersAndReaders(kf)
		return kf, err
	default:
		return nil, fmt.Errorf("Invalid injector driver %s", c.Driver)
	}

}

// SetWritersAndReaders fom the config set the injector http apis, tcp apis, and writers
func (cf *InjectorConfig) SetWritersAndReaders(inj Injector) error {

	inj.SetResolutions(cf.resolutionArray)
	inj.SetDurations(cf.durations)

	// set up the writer it needs to be done AFTER the Resolution times are verified
	if cf.Writer.Metrics.Driver != "" {
		err := inj.SetWriter(cf.Writer)
		if err != nil {
			return err
		}

		shared.Set("is_writer", true)
	}

	// set up sub writers after Main Writers are added
	if cf.Writer.SubMetrics.Driver != "" {

		if cf.Writer.SubIndexer.Driver == "" {
			return fmt.Errorf("You need an Indexer config set for a Sub Metric writer")
		}
		err := inj.SetSubWriter(cf.Writer)
		if err != nil {
			return err
		}
		// some data for the shared land
		shared.Set("is_writer", true)

	}

	if cf.Reader.Listen != "" {
		err := inj.SetReader(cf.Reader)
		if err != nil {
			return err
		}
		// some data for the shared land
		shared.Set("is_reader", true)
		shared.Set("is_api", true)
	}

	if cf.TCPReader.Listen != "" {
		err := inj.SetTCPReader(cf.TCPReader)
		if err != nil {
			return err
		}
		// some data for the shared land
		shared.Set("is_tcpapi", true)
	}

	if cf.Reader.GPRCListen != "" {
		shared.Set("is_grpc", true)
	}
	return nil
}
