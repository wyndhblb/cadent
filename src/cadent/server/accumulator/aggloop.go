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
   The accumulators first pass is to do the first flush time for incoming stats
   after that it's past to this object where it manages the multiple "keeper" loops
   for various time snapshots (i.e. 10s, 1m, 10m, etc)

   The Main Accumulator's job is simply to pass things to this object
   which can then use the Writers to dump data out as nessesary .. if there is no "writer" defined
   this object is not used as otherwise it does nothing useful except take up ram

   Based on the flusher times for a writer, it will prefix things by the string rep

   for instance if things are files the output will be "/path/to/file_[10s|60s|600s]"
   for Databases, the tables will be assumed "basetable_[10s|60s|600s]"

*/

package accumulator

import (
	broadcast "cadent/server/broadcast"
	dispatch "cadent/server/dispatch"
	"cadent/server/schemas/repr"
	stats "cadent/server/stats"
	"cadent/server/utils"
	writers "cadent/server/writers"
	httpapi "cadent/server/writers/api/http"
	tcpapi "cadent/server/writers/api/tcp"
	metrics "cadent/server/writers/metrics"
	"fmt"
	logging "gopkg.in/op/go-logging.v1"
	"sync"
	"time"
)

const (
	AGGLOOP_DEFAULT_QUEUE_LENGTH = 1024
	AGGLOOP_DEFAULT_WORKERS      = 32
)

/** dispatcher job **/
type StatJob struct {
	Aggregators *repr.MultiAggregator
	Stat        *repr.StatRepr
}

func (j StatJob) IncRetry() int {
	return 0
}

func (j StatJob) OnRetry() int {
	return 0
}

func (t StatJob) DoWork() error {
	t.Aggregators.Add(t.Stat)
	return nil
}

// if in a multi writer world, need to hold these for the arguments to the startWriteLooper
type multiWriter struct {
	ws    []*writers.WriterLoop
	mu    *sync.Mutex
	flush time.Duration
	ttl   time.Duration
}

type AggregateLoop struct {

	// these are assigned from the config file in the PreReg config file
	FlushTimes []time.Duration `json:"flush_time"`
	TTLTimes   []time.Duration `json:"ttl_times"`

	Aggregators *repr.MultiAggregator
	Name        string

	Shutdown   *broadcast.Broadcaster
	startstop  utils.StartStop
	shutitdown bool
	InputChan  chan repr.StatRepr

	// write to a DB of some kind on flush sense there can be multiple writers
	// use the flush duration as a key
	OutWriters map[int64]*multiWriter

	// if ByPass is set in the accumulator, StatReprs get directly injected
	// into the writers, skipping any accumulation (be warned that this can by very taxing on the writers)
	ByPass         bool `json:"bypass"`
	DirectToWriter chan *repr.StatRepr

	OutReader    *httpapi.ApiLoop
	OutTCPReader *tcpapi.TCPLoop

	// dispathers
	statDispatcher *dispatch.DispatchQueue

	// if true will set the flusher to basically started at "now" time otherwise it will use time % duration
	// use case:
	// statsd flushes to a "none-writer" should be more or less randomized to keep everything
	// from hammering the pre-writer listeners ever tick
	// things that are writers (especially graphite style) should be flushed on
	// time % duration intervals
	// defaults to false
	flush_random_ticker bool

	log *logging.Logger
}

func NewAggregateLoop(flushtimes []time.Duration, ttls []time.Duration, name string) (*AggregateLoop, error) {

	agg := &AggregateLoop{
		Name:                name,
		FlushTimes:          flushtimes,
		TTLTimes:            ttls,
		Shutdown:            broadcast.New(1),
		OutWriters:          make(map[int64]*multiWriter),
		Aggregators:         repr.NewMulti(flushtimes),
		ByPass:              false,
		DirectToWriter:      make(chan *repr.StatRepr, AGGLOOP_DEFAULT_QUEUE_LENGTH),
		InputChan:           make(chan repr.StatRepr, AGGLOOP_DEFAULT_QUEUE_LENGTH),
		flush_random_ticker: false,
	}

	agg.log = logging.MustGetLogger("aggloop." + name)

	return agg, nil
}

func (agg *AggregateLoop) getResolutionArray() [][]int {
	// get the [time, ttl] array for use in the Metrics Writers
	var res [][]int
	for idx, dur := range agg.FlushTimes {
		res = append(res, []int{
			int(dur.Seconds()),
			int(agg.TTLTimes[idx].Seconds()),
		})
	}
	return res
}

// config the HTTP interface if desired
func (agg *AggregateLoop) SetReader(conf httpapi.ApiConfig) error {
	rl := new(httpapi.ApiLoop)

	// set the resolution bits
	res := agg.getResolutionArray()

	// grab the first resolution as that's the one the main "reader" will be on
	err := rl.Config(conf, float64(res[0][0]))
	if err != nil {
		return err
	}

	rl.SetResolutions(res)
	rl.SetBasePath(conf.BasePath)
	agg.OutReader = rl
	return nil

}

// config the HTTP interface if desired
func (agg *AggregateLoop) SetTCPReader(conf tcpapi.TCPApiConfig) error {
	rl := new(tcpapi.TCPLoop)

	// set the resolution bits
	res := agg.getResolutionArray()

	// grab the first resolution as that's the one the main "reader" will be on
	err := rl.Config(conf, float64(res[0][0]))
	if err != nil {
		return err
	}

	rl.SetResolutions(res)
	agg.OutTCPReader = rl
	return nil
}

// set the metrics and index writers types.  Based on the writer type
// the number of actuall aggregator loops needed may change
// for instance RRD file DBs typically "self rollup" so there's no need
// to deal with the aggregation of longer times, but DBs (cassandra) cannot do that
// automatically, so we set things appropriately
func (agg *AggregateLoop) SetWriter(conf writers.WriterConfig, mainorsub string) (err error) {
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
		agg.log.Critical("Error setting indexer: %s", err)
		return err
	}

	numWriters, _ := confMets.ResolutionsNeeded()

	// need a writer for each timer loop
	for i, dur := range agg.FlushTimes {
		wr, err := writers.New()
		if conf.MetricQueueLength > 0 {
			wr.MetricQLen = conf.MetricQueueLength
		}
		if conf.IndexerQueueLength > 0 {
			wr.IndexerQLen = conf.IndexerQueueLength
		}
		if err != nil {
			agg.log.Critical("Writer error:: %s", err)
			return err
		}
		mets, err := confMets.NewMetrics(dur, conf.Caches)
		if err != nil {
			return err
		}

		agg.log.Notice("Setting up writer for resolution %s", dur.String())
		wr.SetName(dur.String())

		mets.SetIndexer(idx)
		mets.SetResolutions(agg.getResolutionArray())
		mets.SetCurrentResolution(int(dur.Seconds()))
		wr.SetMetrics(mets)
		wr.SetIndexer(idx)

		durNS := dur.Nanoseconds()
		if _, ok := agg.OutWriters[durNS]; !ok {
			agg.OutWriters[durNS] = &multiWriter{
				ws:    []*writers.WriterLoop{wr},
				mu:    new(sync.Mutex),
				flush: dur,
				ttl:   agg.TTLTimes[i],
			}
		} else {
			agg.OutWriters[durNS].ws = append(agg.OutWriters[durNS].ws, wr)
		}

		agg.log.Notice("Set Aggregator writer @ %s", dur.String())
		if numWriters == metrics.FirstResolution {
			agg.log.Notice("Only one writer needed for this writer driver")
			break
		}

	}
	agg.log.Notice("Set %d Aggregator writers", len(agg.OutWriters))
	return nil
}

func (agg *AggregateLoop) startInputLooper() {
	shut := agg.Shutdown.Listen()

	if agg.statDispatcher == nil {
		workers := AGGLOOP_DEFAULT_WORKERS
		agg.statDispatcher = dispatch.NewDispatchQueue(workers, AGGLOOP_DEFAULT_QUEUE_LENGTH, 0)
		agg.statDispatcher.Name = "aggloop"
		agg.statDispatcher.Start()
	}

	for {
		select {
		case stat, more := <-agg.InputChan:
			if !more {
				return
			}
			agg.statDispatcher.Add(StatJob{Aggregators: agg.Aggregators, Stat: &stat})
		//agg.Aggregators.Add(stat)
		case <-shut.Ch:
			return
		}
	}
}

// this is a helper function to get things to "start" on nicely "rounded"
// ticker intervals .. i.e. if  duration is 5 seconds .. start on t % 5
// this is approximate of course .. there will be some error in the MS/NS range
// but is should be good in the second range
func (agg *AggregateLoop) delayRoundedTicker(duration time.Duration) *time.Ticker {
	time.Sleep(time.Now().Truncate(duration).Add(duration).Sub(time.Now()))
	return time.NewTicker(duration)
}

// start the looper for each flush time as well as the writers
func (agg *AggregateLoop) startWriteLooper(mws *multiWriter) {
	if agg.shutitdown {
		agg.log.Warning("Got shutdown signal, not starting writers")
		return
	}

	shut := agg.Shutdown.Listen()

	_dur := mws.flush
	_ttl := mws.ttl
	_ttlS := uint32(_ttl.Seconds())
	_durS := uint32(_dur.Seconds())

	// start up the writers listeners
	for _, w := range mws.ws {
		w.Start()
	}

	statsdName := fmt.Sprintf("aggregator.%s.writesloops", _dur.String())
	post := func(w *writers.WriterLoop, items repr.StatReprSlice) {
		defer stats.StatsdSlowNanoTimeFunc("aggregator.postwrite-time-ns", time.Now())

		//_mu.Lock()
		//defer _mu.Unlock()

		if w.Full() {
			agg.log.Critical(
				"Saddly the write queue is full, if we continue adding to it, the entire world dies, we have to bail this write tick (metric queue: %d, indexer queue: %d)",
				w.MetricQLen,
				w.IndexerQLen,
			)
			return
		}

		for _, stat := range items {
			//fmt.Println("FLUSH POST: ", stat.Name.Key, "time: ", stat.Time, "count:", stat.Count)
			stat.Name.Resolution = _durS
			stat.Name.Ttl = _ttlS // need to add in the TTL
			w.WriterChan() <- stat
		}
		//agg.log.Critical("CHAN WRITE: LEN: %d Items: %d", len(writer.WriterChan()), m_items)
		//agg.Aggregators.Clear(duration) // clear now before any "new" things get added
		// need to clear out the Agg
		stats.StatsdClientSlow.Incr(statsdName, 1)
		return
	}

	agg.log.Notice("Starting Aggregater Loop for %s", _dur.String())
	var ticker *time.Ticker
	// either flush at a random "duration interval" or flush at time % duration interval
	if agg.flush_random_ticker {
		agg.log.Notice("Aggregater Loop for %s at random start .. starting: %d", _dur.String(), time.Now().Unix())
		ticker = time.NewTicker(_dur)
	} else {
		agg.log.Notice("Aggregater Loop for %s starting at time %% %s .. starting %d", _dur.String(), _dur.String(), time.Now().Unix())
		ticker = agg.delayRoundedTicker(_dur)
	}
	for {
		select {
		case dd := <-ticker.C:
			items := agg.Aggregators.Get(_dur).GetListAndClear()
			iLen := len(items)
			agg.log.Debug(
				"Flushing %d stats in bin %s to writer at: %d",
				iLen,
				_dur.String(),
				dd.Unix(),
			)
			if iLen == 0 {
				agg.log.Debug(
					"No stats to send to writer in bin %s at: %d",
					_dur.String(),
					dd.Unix(),
				)
				continue
			}
			for _, w := range mws.ws {
				// to ensure time order of inputs, this cannot be a go routine, so we have to block here
				post(w, items)
			}

		case <-shut.Ch:
			ticker.Stop()
			shut.Close()
			return
		}
	}
}

// start the looper to write stats as the come in rather then a flush aggregation
func (agg *AggregateLoop) startWriteByPassLooper(mws *multiWriter) {
	if agg.shutitdown {
		agg.log.Warning("Got shutdown signal, not starting writers")
		return
	}

	shut := agg.Shutdown.Listen()

	//_dur := mws.flush
	_ttl := uint32(mws.ttl.Seconds())

	// start up the writers listeners
	for _, w := range mws.ws {
		w.Start()
	}
	agg.log.Notice("Starting BYPASS Aggregater Loop (direct to writer)")

	for {
		for {
			select {
			case dd, more := <-agg.DirectToWriter:
				if !more {
					return
				}
				if dd == nil {
					continue
				}
				for _, w := range mws.ws {
					dd.Name.Ttl = _ttl
					w.WriterChan() <- dd
				}

			case <-shut.Ch:
				shut.Close()
				return
			}
		}
	}
}

// For every flus time, start a new timer loop to perform writes
func (agg *AggregateLoop) Start() error {
	agg.startstop.Start(func() {
		agg.log.Notice("Starting Aggregator Loop for `%s`", agg.Name)

		if agg.shutitdown {
			agg.log.Warning("Got shutdown signal, not starting loop")
			return
		}

		//start the input loop acceptor
		go agg.startInputLooper()

		// since we can have multiple writers, some may want only "one" agg some may want more
		// so we need to figure out the proper Aggs to keep
		num_writers := metrics.FirstResolution
		for _, mws := range agg.OutWriters {
			for _, wr := range mws.ws {
				needs, _ := metrics.ResolutionsNeeded(wr.Metrics().Driver())
				if needs == metrics.AllResolutions {
					num_writers = metrics.AllResolutions
					agg.Aggregators = repr.NewMulti(agg.FlushTimes)
					break
				}
			}
		}

		// need to "reset" the Aggregators to just be the FIRST one if num_writers == FirstResolution
		if num_writers == metrics.FirstResolution {
			agg.Aggregators = repr.NewMulti([]time.Duration{agg.FlushTimes[0]})
		}

		for _, writ := range agg.OutWriters {
			if !agg.ByPass {
				go agg.startWriteLooper(writ)
			} else {
				go agg.startWriteByPassLooper(writ)
			}
		}

		// fire up the reader if around
		if agg.OutReader != nil {
			go agg.OutReader.Start()
		}
		if agg.OutTCPReader != nil {
			go agg.OutTCPReader.Start()
		}
		return
	})
	return nil
}

func (agg *AggregateLoop) Stop() {
	agg.startstop.Stop(func() {
		if agg.shutitdown {
			return
		}
		agg.log.Warning("Initiating shutdown of aggregator for `%s`", agg.Name)

		if agg.statDispatcher != nil {
			agg.statDispatcher.Stop()
		}

		agg.Shutdown.Send(true)
		agg.shutitdown = true

		if agg.OutReader != nil {
			agg.OutReader.Stop()
		}

		if len(agg.OutWriters) > 0 {
			var wg sync.WaitGroup
			stopWriters := func(w *writers.WriterLoop) {
				agg.log.Warning("Shutting down writer `%s:%s`", agg.Name, w.GetName())
				w.Stop()
				agg.log.Warning("Finished Shutdown of writer `%s:%s`", agg.Name, w.GetName())
				wg.Done()
			}

			for _, mw := range agg.OutWriters {
				for _, w := range mw.ws {
					cp := w
					agg.log.Warning("Starting Shutdown of writer `%s:%s`", agg.Name, cp.GetName())
					go stopWriters(cp)
					wg.Add(1)
				}
			}
			wg.Wait()
		}
		agg.log.Warning("Shutdown of aggregator `%s`", agg.Name)
	})
}
