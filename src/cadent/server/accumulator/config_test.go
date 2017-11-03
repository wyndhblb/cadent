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

package accumulator

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func TestConfigAccumualtorAccumulator(t *testing.T) {
	// Only pass t into top-level Convey calls

	//profiler
	//go http.ListenAndServe(":6065", nil)
	Convey("Config toml accumulator parser", t, func() {

		Convey("Config toml should parse to a good accumulator", func() {
			conf_test := `
	backend = "graphite-out"
	input_format = "statsd"
	output_format = "graphite"
	accumulate_flush = "1s"
	times = ["5s", "1m", "10m"]
	tags = "foo=bar,loo=moo"

	[writer]
	driver="file"
	dsn="/tmp/none"

	`
			acc_c, err := ParseConfigString(conf_test)
			Convey("Error should be nil", func() {
				So(err, ShouldEqual, nil)
			})
			Convey("Error acc should not be nil", func() {
				So(acc_c, ShouldNotEqual, nil)
			})

			//t.Logf("%v", acc_c)

			Convey("Should have 3 timers", func() {
				So(len(acc_c.FlushTimes), ShouldEqual, 3)
			})

			Convey("First Flush time should be 5 seconds", func() {
				So(acc_c.FlushTimes[0], ShouldEqual, time.Duration(5*time.Second))
			})
			Convey("Should have 2 tags", func() {
				So(acc_c.Accumulate.Tags().Len(), ShouldEqual, 2)
			})

		})

		Convey("Config toml keeper should not have proper mutiples", func() {
			conf_test := `
	backend = "graphite-out"
	input_format = "statsd"
	output_format = "graphite"
	times = ["5s", "31s", "10m"]
	`
			_, err := ParseConfigString(conf_test)
			So(err, ShouldNotEqual, nil)
		})

		Convey("Config toml times should not have proper order", func() {
			conf_test := `
	backend = "graphite-out"
	input_format = "statsd"
	output_format = "graphite"
	times = ["1m", "5s", "10m"]
	`
			_, err := ParseConfigString(conf_test)
			So(err, ShouldNotEqual, nil)
		})

		Convey("Config toml keeper should fail on TTL", func() {
			conf_test := `
	backend = "graphite-out"
	input_format = "statsd"
	output_format = "graphite"
	times = ["1m:7asd", "5s:30d", "10m:1y"]
	`
			_, err := ParseConfigString(conf_test)
			So(err, ShouldNotEqual, nil)
		})

		Convey("Config toml keeper should parse correctly", func() {
			conf_test := `
	backend = "graphite-out"
	input_format = "statsd"
	output_format = "graphite"
	times = ["5s:168h", "1m:720h", "10m:17520h"]
	`
			_, err := ParseConfigString(conf_test)
			So(err, ShouldEqual, nil)
		})

		Convey("Config toml should parse", func() {
			conf_test := `
	kasdjasd = {};
	backend = 8
	`
			_, err := ParseConfigString(conf_test)
			So(err, ShouldNotEqual, nil)
		})

		Convey("Config toml should parse to a fail on bad toml", func() {
			conf_test := `
	backend = "graphite-out"
	moo_input_format = "moo"
	output_format = "graphite"
	`
			_, err := ParseConfigString(conf_test)
			So(err, ShouldNotEqual, nil)
		})

		Convey("Config toml should parse to a fail on input_format", func() {
			conf_test := `
	backend = "graphite-out"
	input_format = "moo"
	output_format = "graphite"
	keep_keys = true
	`
			_, err := ParseConfigString(conf_test)
			So(err, ShouldNotEqual, nil)
		})

		Convey("Config toml should parse to a fail on flush time", func() {
			conf_test := `
	backend = "graphite-out"
	input_format = "statsd"
	output_format = "graphite"
	keep_keys = true
	times = ["5ii"]
	`
			_, err := ParseConfigString(conf_test)
			So(err, ShouldNotEqual, nil)
		})

		Convey("Config toml should parse to a fail on No Backend", func() {
			conf_test := `
	input_format = "statsd"
	output_format = "graphite"
	keep_keys = true
	`
			_, err := ParseConfigString(conf_test)
			So(err, ShouldNotEqual, nil)
		})

		Convey("Config toml should parse to a fail on output_format", func() {
			conf_test := `
	input_format = "statsd"
	output_format = "moo"
	options = [
            ["legacyNamespace", "true"],
            ["prefixGauge", "gauges"],
            ["prefixTimer", "timers"],
            ["prefixCounter", "counters"],
            ["globalPrefix", ""],
            ["globalSuffix", "stats"],
            ["percentThreshold", "0.75,0.90,0.95,0.99"]
    	]
	`
			_, err := ParseConfigString(conf_test)
			So(err, ShouldNotEqual, nil)
		})
	})
}
