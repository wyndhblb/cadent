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
   The accumulator aggrigator, based on a config

    THe "line" flow (for a given item) goes something like this

    Listen Ingest
    	-> PreReg (for rejection/backend selection)
    	-> Accumulate -> (Flush to OutputQueue of ToBackend)
    		-> BackendInputQueue
    		-> PreReg again (new keys and new backends, i.e. statsd -> graphite)
    			(note: skipping accumulate again if that backend PreReg has an accumulator
    			 i.e. the splitItem has AccumulatedParsed set)
    		-> Backend ConstHash
    		-> Out

    There must be a "splitter.Splitter" for each FormatterItem otherwise there's no way

    If there is an "external writer" you can set the Backend to 'black_hole' and the line "ends"
    there on Flush and the line should hopefully get "written" somewhere like your
    Favorite DB
*/

package accumulator

import (
	"bytes"
	"cadent/server/schemas/repr"
	splitter "cadent/server/splitter"
	stats "cadent/server/stats"
	"cadent/server/utils"
	writers "cadent/server/writers"
	"fmt"
	logging "gopkg.in/op/go-logging.v1"
	"io"
	"sync"
	"time"
)

const BLACK_HOLE_BACKEND = "BLACKHOLE"

/**** main accumulator object */

type Accumulator struct {

	// these are assigned from the config file in the PreReg config file
	ToBackend         string          `json:"backend" yaml:"backend"`
	FromBackend       string          `json:"from-backend" yaml:"from-backend"`
	FormatterName     string          `json:"formatter" yaml:"formatter"`
	AccumulatorName   string          `json:"accumulator" yaml:"accumulator"`
	Name              string          `json:"name" yaml:"name"`
	KeepKeys          bool            `json:"keep-keys" yaml:"keep-keys"` // if true, will not "remove" the keys post flush, just set them to 0
	AccumulateTime    time.Duration   `json:"accumulate-timer" yaml:"accumulate-timer"`
	FlushTimes        []time.Duration `json:"flush-time" yaml:"flush-time"`
	TTLTimes          []time.Duration `json:"ttl-times" yaml:"ttl-times"`
	RandomTickerStart bool            `json:"random-ticker-start" yaml:"random-ticker-start"`
	TagMode           repr.TagMode    `json:"tag-mode" yaml:"tag-mode"`                 // see repr.TAG_MODE
	HashMode          repr.HashMode   `json:"hash-mode" yaml:"hash-mode"`               // see repr.HASH_MODE
	ByPass            bool            `json:"bypass" yaml:"bypass"`                     // if true, we skip the accumulator, and toss into writers (if any), or direct to backend
	ShowLineErrors    bool            `json:"show_line_errors" yaml:"show_line_errors"` // on bad line show an error or not

	Accumulate AccumulatorItem `json:"-" yaml:"-"`
	Formatter  FormatterItem   `json:"-" yaml:"-"`

	InSplitter  splitter.Splitter `json:"-" yaml:"-"`
	OutSplitter splitter.Splitter `json:"-" yaml:"-"`

	mu          sync.Mutex
	timer       *time.Ticker
	LineQueue   chan []byte             `json:"-" yaml:"-"`
	OutputQueue chan splitter.SplitItem `json:"-" yaml:"-"`
	Shutdown    chan bool               `json:"-" yaml:"-"`
	shutitdown  bool

	Aggregators *AggregateLoop `json:"-" yaml:"-"` // writers hook into the main agg flushing loops

	log *logging.Logger
}

func NewAccumlator(inputtype string, outputtype string, keepkeys bool, name string) (*Accumulator, error) {

	fmter, err := NewFormatterItem(outputtype)
	if err != nil {
		return nil, err
	}

	acc, err := NewAccumulatorItem(inputtype)
	if err != nil {
		return nil, err
	}
	fmter.Init()
	acc.Init(fmter)
	acc.SetKeepKeys(keepkeys)
	acc.SetResolution(time.Duration(time.Second))

	ac := &Accumulator{
		Accumulate:        acc,
		Formatter:         fmter,
		AccumulatorName:   inputtype,
		FormatterName:     outputtype,
		KeepKeys:          keepkeys,
		Name:              name,
		FlushTimes:        []time.Duration{time.Duration(time.Second)},
		AccumulateTime:    time.Duration(time.Second),
		LineQueue:         make(chan []byte, 10000),
		shutitdown:        false,
		TagMode:           repr.TAG_METRICS2,
		HashMode:          repr.HASH_FNV,
		timer:             nil,
		ByPass:            false,
		RandomTickerStart: false,
	}

	ac.log = logging.MustGetLogger("accumulator." + name)

	// determine the splitter from the formatter item
	nul_conf := make(map[string]interface{})
	ispl, err := splitter.NewSplitterItem(inputtype, nul_conf)
	if err != nil {
		return nil, err
	}
	ac.InSplitter = ispl

	ospl, err := splitter.NewSplitterItem(outputtype, nul_conf)
	if err != nil {
		return nil, err
	}
	ac.OutSplitter = ospl

	return ac, nil
}

// create the overlord aggregator
func (acc *Accumulator) SetAggregateLoop(conf writers.WriterConfig) (agg *AggregateLoop, err error) {
	acc.Aggregators, err = NewAggregateLoop(acc.FlushTimes, acc.TTLTimes, acc.Name)
	acc.Aggregators.flush_random_ticker = acc.RandomTickerStart

	// Aggregators need to know of bypassing too
	acc.Aggregators.ByPass = acc.ByPass

	if err != nil {
		return nil, err
	}
	err = acc.Aggregators.SetWriter(conf, "main")
	if err != nil {
		acc.log.Critical("Error setting main writer: %s", err)
		return nil, err
	}
	return acc.Aggregators, nil
}

func (acc *Accumulator) SetSubAggregateLoop(conf writers.WriterConfig) (agg *AggregateLoop, err error) {
	// the aggs need to be set first
	if acc.Aggregators == nil {
		return nil, fmt.Errorf("To have 'sub' writers, you first need a main writer.")
	}
	if err != nil {
		return nil, err
	}
	err = acc.Aggregators.SetWriter(conf, "sub")
	if err != nil {
		acc.log.Critical("Error setting sub writer: %s", err)
		return nil, err
	}
	return acc.Aggregators, nil
}

func (acc *Accumulator) SetOutputQueue(qu chan splitter.SplitItem) {
	acc.OutputQueue = qu
}

func (acc *Accumulator) ProcessSplitItem(sp splitter.SplitItem) error {
	return acc.ProcessLine(sp.Line())
}

func (acc *Accumulator) ProcessLine(sp []byte) error {
	stats.StatsdClient.Incr("accumulator.lines.incoming", 1)
	if !acc.shutitdown {
		acc.LineQueue <- sp
	}
	return nil
}

// this is a helper function to get things to "start" on nicely "rounded"
// ticker intervals .. i.e. if  duration is 5 seconds .. start on t % 5
func (acc *Accumulator) delayRoundedTicker(duration time.Duration) *time.Ticker {
	//time.Sleep(time.Duration(durSec - (time.Now().UnixNano() % durSec)))
	time.Sleep(time.Now().Truncate(duration).Add(duration).Sub(time.Now()))
	return time.NewTicker(duration)
}

// start the flusher at the time interval
// best to call this in a go routine
func (acc *Accumulator) Start() error {
	if acc.shutitdown {
		acc.log.Warning("Shutting down, will not start `%s`", acc.Name)
		return nil
	}

	// no need for this in bypass mode
	if acc.timer == nil && !acc.ByPass {
		// make sure to set the proper resolution in the
		// aggregator matters for those things that have time components on the incoming
		// graphite/carbon
		acc.Accumulate.SetResolution(acc.AccumulateTime)
		acc.Accumulate.SetHashMode(acc.HashMode)
		acc.Accumulate.SetTagMode(acc.TagMode)

		if acc.RandomTickerStart {
			acc.log.Notice(
				"Accumulator Loop for %s at random start .. starting: %d",
				acc.AccumulateTime.String(),
				time.Now().Unix(),
			)
			acc.timer = time.NewTicker(acc.AccumulateTime)
		} else {
			acc.log.Notice(
				"Aggregater Loop for %s starting at time %% %s .. starting %d",
				acc.AccumulateTime.String(),
				acc.AccumulateTime.String(),
				time.Now().Unix(),
			)
			acc.timer = acc.delayRoundedTicker(acc.AccumulateTime)
		}
	}

	if acc.LineQueue == nil {
		acc.LineQueue = make(chan []byte, 10000)
	}

	// here again as a stop can hit in the middle of the delay timer
	if acc.shutitdown {
		acc.log.Warning("Shutting down, will not start `%s`", acc.Name)
		return nil
	}

	acc.log.Notice("Starting accumulator loop for `%s`", acc.Name)

	// fire up Aggs
	if acc.Aggregators != nil {
		acc.log.Notice("Starting aggregator loop for `%s`", acc.Name)
		go acc.Aggregators.Start()
	}

	defer func() {
		close(acc.LineQueue)
	}()

	acc.Shutdown = make(chan bool, 5)

	// in ByPass mode, a very different loop is needed
	if acc.ByPass {
		for {
			select {
			case line, more := <-acc.LineQueue:
				if !more {
					return nil
				}
				r, err := acc.Accumulate.ProcessLineToRepr(line)
				if acc.ShowLineErrors && err != nil {
					acc.log.Warning("Cannot push line: %v", err)
				}
				if r == nil {
					continue
				}
				acc.Aggregators.DirectToWriter <- r
				stats.StatsdClient.Incr("accumulator.lines.processed", 1)
			case <-acc.Shutdown:
				acc.log.Warning("Shutting down final flush of accumulator `%s`", acc.Name)
				return nil
			}
		}
	} else {

		for {
			select {
			case line, more := <-acc.LineQueue:
				if !more {
					return nil
				}
				acc.Accumulate.ProcessLine(line)

				stats.StatsdClient.Incr("accumulator.lines.processed", 1)
			case dd := <-acc.timer.C:
				acc.log.Debug("Flushing accumulator %s to: %s at: %v", acc.Name, acc.ToBackend, dd.Unix())
				go func() {
					acc.FlushAndPost(dd)
				}()
			case <-acc.Shutdown:
				acc.timer.Stop()
				acc.log.Warning("Shutting down final flush of accumulator `%s`", acc.Name)
				return nil
			}
		}
	}
}

func (acc *Accumulator) Stop() {
	acc.log.Warning("Initiating shutdown of accumulator `%s`", acc.Name)
	acc.shutitdown = true
	if acc.Shutdown != nil {
		acc.Shutdown <- true
	}

	if !acc.ByPass {
		acc.FlushAndPost(time.Now())
	}

	if acc.Aggregators != nil {
		acc.Aggregators.Stop()
	}
}

// move back into Main Server loop
func (acc *Accumulator) PushLine(spl splitter.SplitItem) {
	if acc.OutputQueue != nil && acc.ToBackend != BLACK_HOLE_BACKEND {
		stats.StatsdClient.Incr("accumulator.stats.lines.outgoing", 1)
		acc.OutputQueue <- spl
	}
}

// move into Aggregator land
func (acc *Accumulator) PushStat(spl *repr.StatRepr) {
	stats.StatsdClient.Incr("accumulator.stats.repr.outgoing", 1)
	acc.Aggregators.InputChan <- *spl
}

func (acc *Accumulator) FlushAndPost(attime time.Time) ([]splitter.SplitItem, error) {
	defer stats.StatsdSlowNanoTimeFunc(fmt.Sprintf("accumulator.flushpost-time-ns"), time.Now())

	// a few modes here
	// 1. if th backend is "BLACKHOLE" we don't need the "acctual" lines
	//    just the split items as it's not going back to be re-consistently hashed
	// 2. if the input and output formats are the same, no need to "process"
	//    lines

	//log.Notice("Flush: %s", items)
	//return []splitter.SplitItem{}, nil
	//t := time.Now()
	outSpl := make([]splitter.SplitItem, 0)
	var items *flushedList

	// the buck stops here and into writers
	if acc.ToBackend != BLACK_HOLE_BACKEND {

		buffer := utils.GetBytesBuffer()
		defer utils.PutBytesBuffer(buffer)
		items = acc.Accumulate.Flush(buffer)
		for {
			line, err := buffer.ReadBytes(repr.NEWLINE_SEPARATOR_BYTE)
			if err == io.EOF {
				break
			}
			if err != nil {
				acc.log.Error("Buffer read error", err)
				continue
			}
			if len(line) == 0 {
				continue
			}

			spl, err := acc.OutSplitter.ProcessLine(line)

			if err != nil {
				acc.log.Error("Invalid Line post flush accumulate `%s` Err:%s", line, err)
				continue
			}
			// this tells the server backends to NOT send to the accumulator anymore
			// otherwise we'd get serious infinite channel loops
			//log.Warning("ACC posted: %v  Len %d", spl.Line(), acc.OutputQueue)
			spl.SetPhase(splitter.AccumulatedParsed)
			spl.SetOrigin(splitter.Other)
			spl.SetOriginName(acc.FromBackend) // where we are from
			outSpl = append(outSpl, spl)
			//log.Notice("sending: %s Len:%d", spl.Line(), len(acc.OutputQueue))
			acc.PushLine(spl)
			//log.Notice("SENT: %s Len:%d", spl.Line(), len(acc.OutputQueue))
		}
	} else {
		items = acc.Accumulate.FlushList()
	}

	if acc.Aggregators != nil {
		for _, stat := range items.Stats {
			if stat.Time == 0 {
				stat.Time = attime.UnixNano() // need to set this as this is the flush time
			}
			acc.PushStat(stat)
		}
		acc.log.Debug("Aggregator Flush: `%s` to `%s` Lines: %d", acc.Name, acc.Aggregators.Name, len(items.Stats))
	}
	stats.StatsdClientSlow.Incr("accumulator.flushesposts", 1)

	acc.log.Debug("Flushed accumulator `%s` to Backend: `%s` Lines: %d", acc.Name, acc.ToBackend, len(outSpl))
	items = nil // GC me
	return outSpl, nil
}

// flush out the accumulator, and "reparse" the lines
func (acc *Accumulator) Flush() ([]splitter.SplitItem, error) {
	defer stats.StatsdSlowNanoTimeFunc(fmt.Sprintf("accumulator.flush-time-ns"), time.Now())

	buffer := new(bytes.Buffer)
	acc.Accumulate.Flush(buffer)

	var out_spl []splitter.SplitItem
	for {
		line, err := buffer.ReadBytes(repr.NEWLINE_SEPARATOR_BYTE)
		if err == io.EOF {
			break
		}
		if err != nil {
			acc.log.Error("Buffer read error", err)
			continue
		}
		if len(line) == 0 {
			continue
		}
		spl, err := acc.OutSplitter.ProcessLine(line)
		if err != nil {
			acc.log.Error("Invalid Line post flush accumulate `%s` Err:%s", line, err)
			continue
		}
		// this tells the server backends to NOT send to the accumulator anymore
		// otherwise we'd get serious infinite channel loops
		spl.SetPhase(splitter.AccumulatedParsed)
		spl.SetOrigin(splitter.Other)
		out_spl = append(out_spl, spl)

	}
	stats.StatsdClientSlow.Incr("accumulator.flushes", 1)
	return out_spl, nil
}

func (acc *Accumulator) LogConfig() {
	acc.log.Debug(" - Accumulator Group: `%s`", acc.Name)
	acc.log.Debug("   - Delgateing to Backend: `%s`", acc.ToBackend)
	acc.log.Debug("   - Accumulator Output format:: `%s`", acc.Formatter.Type())
	acc.log.Debug("   - Accumulator Type:: `%s`", acc.Accumulate.Name())
	acc.log.Debug("   - Accumulator FlushTime:: `%v`", acc.FlushTimes)
	acc.log.Debug("   - Accumulator KeepKeys:: `%v`", acc.KeepKeys)
	if acc.Aggregators != nil {
		acc.log.Debug("   - Accumulator Aggregator:: `%v`", acc.Aggregators.Name)
	}
}

// just grab whats currently in the queue to be flushed
// this is so we can simply "look" into the accumulator from another source
// (i.e. our monitor)

func (acc *Accumulator) CurrentStats() *repr.ReprList {
	s_rep := new(repr.ReprList)
	stats := acc.Accumulate.Stats()
	t := time.Now()

	for idx, stat := range stats {
		rr := stat.Repr()
		rr.Name.SetKey(idx)
		rr.Time = t.UnixNano()
		rr.Name.Resolution = uint32(acc.FlushTimes[0].Seconds())
		s_rep.Add(*rr)
	}
	return s_rep
}
