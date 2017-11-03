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

   These bypass the normal "line" protocols and consume from "something else"

   and feed them directly into the writer paths

   For now we have a "kafka" injector that consumes the "produced" cadent writer items
*/

package injectors

import (
	"cadent/server/utils/options"
	"cadent/server/writers"
	httpapi "cadent/server/writers/api/http"
	tcpapi "cadent/server/writers/api/tcp"

	"fmt"
	"time"
)

// Injector interface
type Injector interface {

	// SetResolutions
	SetResolutions(res [][]int)

	// Resolutions
	Resolutions() [][]int

	// SetDurations
	SetDurations([]time.Duration)

	// Durations
	Durations() []time.Duration

	// Config the object
	Config(options.Options) error

	// Start the object (basically an go routines that need to get fired up)
	Start() error

	// Stop the objects internal loops and go routines
	Stop() error

	// SetWriter set the writer for a message to go in
	SetWriter(writer writers.WriterConfig) error
	GetWriter() *writers.WriterLoop

	// SetSubWriter set a secondary
	SetSubWriter(writer writers.WriterConfig) error
	GetSubWriter() *writers.WriterLoop

	// SetReader set the http reader
	SetReader(reader httpapi.ApiConfig) error
	GetReader() *httpapi.ApiLoop

	// SetTCPReader set the tcp api
	SetTCPReader(reader tcpapi.TCPApiConfig) error
	GetTCPReader() *tcpapi.TCPLoop
}

// InjectorBase the base object that all injectors should use
type InjectorBase struct {
	resolutions [][]int
	durations   []time.Duration
	httpapi     *httpapi.ApiLoop
	tcpapi      *tcpapi.TCPLoop
	writer      *writers.WriterLoop
	subWriter   *writers.WriterLoop
}

// SetResolutions should be of the form
// [BinTime, TTL]
// we select the BinTime based on the TTL
func (inj *InjectorBase) SetResolutions(res [][]int) {
	inj.resolutions = res
}

// Resolutions
func (inj *InjectorBase) Resolutions() [][]int {
	return inj.resolutions
}

// SetResolutions should be of the form
// [BinTime, TTL]
// we select the BinTime based on the TTL
func (inj *InjectorBase) SetDurations(res []time.Duration) {
	inj.durations = res
}

// Durations
func (inj *InjectorBase) Durations() []time.Duration {
	return inj.durations
}

// SetReader config the HTTP interface if desired
func (inj *InjectorBase) SetReader(conf httpapi.ApiConfig) error {
	rl := new(httpapi.ApiLoop)

	// set the resolution bits
	res := inj.Resolutions()

	// grab the first resolution as that's the one the main "reader" will be on
	err := rl.Config(conf, float64(res[0][0]))
	if err != nil {
		return err
	}

	rl.SetResolutions(res)
	rl.SetBasePath(conf.BasePath)
	inj.httpapi = rl
	return nil

}

// SetTCPReader config the TCP api interface if desired
func (inj *InjectorBase) SetTCPReader(conf tcpapi.TCPApiConfig) error {
	rl := new(tcpapi.TCPLoop)

	// set the resolution bits
	res := inj.Resolutions()

	// grab the first resolution as that's the one the main "reader" will be on
	err := rl.Config(conf, float64(res[0][0]))
	if err != nil {
		return err
	}

	rl.SetResolutions(res)
	inj.tcpapi = rl
	return nil
}

// SetWriter set the metrics and index writers types.
// for injectors only "one" resolution is allowed as we're bypassing the accumulators
// so for other resolutions, one should use the triggered rollups writers
func (inj *InjectorBase) SetWriter(conf writers.WriterConfig) (err error) {
	return inj.setWriter(conf, "main")
}

// SetSubWriter sets up the secondary writer
func (inj *InjectorBase) SetSubWriter(conf writers.WriterConfig) (err error) {
	return inj.setWriter(conf, "sub")
}

func (inj *InjectorBase) setWriter(conf writers.WriterConfig, mainorsub string) (err error) {

	var confIdx writers.WriterIndexerConfig
	var confMets writers.WriterMetricConfig

	switch mainorsub {
	case "sub":
		confIdx = conf.SubIndexer
		confMets = conf.SubMetrics
	default:
		confIdx = conf.Indexer
		confMets = conf.Metrics
	}

	// need only one indexer
	idx, err := confIdx.NewIndexer()
	if err != nil {
		return err
	}

	// the injectors only get "one" writer so we pick the first one
	minDur := inj.durations[0]
	minRes := inj.resolutions[0][0]
	wr, err := writers.New()
	if err != nil {
		return err
	}
	mets, err := confMets.NewMetrics(minDur, conf.Caches)
	if err != nil {
		return err
	}

	wr.SetName(fmt.Sprintf("%ds", minRes))

	mets.SetIndexer(idx)
	mets.SetResolutions(inj.Resolutions())
	mets.SetCurrentResolution(minRes)

	wr.SetMetrics(mets)
	wr.SetIndexer(idx)
	switch mainorsub {
	case "sub":
		inj.subWriter = wr
	default:
		inj.writer = wr
	}

	return nil
}

// GetWriter
func (inj *InjectorBase) GetWriter() *writers.WriterLoop {
	return inj.writer
}

// GetSubWriter
func (inj *InjectorBase) GetSubWriter() *writers.WriterLoop {
	return inj.subWriter
}

// GetReader
func (inj *InjectorBase) GetReader() *httpapi.ApiLoop {
	return inj.httpapi
}

// SetTCPReader
func (inj *InjectorBase) GetTCPReader() *tcpapi.TCPLoop {
	return inj.tcpapi
}
