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
	The Ram Only Metric Log

	Use the Chunk based Cacher, but don't write anything, just pure old RAM

*/

package metrics

import (
	sindexer "cadent/server/schemas/indexer"
	smetrics "cadent/server/schemas/metrics"
	"cadent/server/schemas/repr"
	"cadent/server/stats"
	"cadent/server/utils"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"fmt"
	"golang.org/x/net/context"
	"gopkg.in/op/go-logging.v1"
	"time"
)

const (
	// how many chunks to keep in RAM
	RAM_LOG_CHUNKS = 8

	// hold RAM_LOG_TIME_CHUNKS duration per timeseries
	RAM_LOG_TIME_CHUNKS = 15
)

/****************** Metrics Writer *********************/
type RamLogMetric struct {
	CacheBase

	cacher *CacherChunk
	log    *logging.Logger
}

func NewRamLogMetrics() *RamLogMetric {
	rm := new(RamLogMetric)
	rm.log = logging.MustGetLogger("writers.ramlog")
	rm.cacheChunks = RAM_LOG_CHUNKS
	rm.cacheTimeWindow = RAM_LOG_TIME_CHUNKS

	return rm
}

// Config setups all the options for the writer
func (rm *RamLogMetric) Config(conf *options.Options) (err error) {
	rm.options = conf

	// even if not used, it best be here for future goodies
	_, err = conf.Int64Required("resolution")
	if err != nil {
		return fmt.Errorf("resolution needed for ram writer: %v", err)
	}

	_tgs := conf.String("tags", "")
	if len(_tgs) > 0 {
		rm.staticTags = repr.SortingTagsFromString(_tgs)
	}

	rm.name = conf.String("name", "metrics:ram")
	err = RegisterMetrics(rm.Name(), rm)
	if err != nil {
		return err
	}

	rdur, err := time.ParseDuration(CASSANDRA_DEFAULT_RENDER_TIMEOUT)
	if err != nil {
		return err
	}
	rm.renderTimeout = rdur

	_cache, _ := conf.ObjectRequired("cache")
	if _cache == nil {
		return errMetricsCacheRequired
	}

	// need to cast this to the proper cache type
	var ok bool
	rm.cacher, ok = _cache.(*CacherChunk)
	if !ok {
		return ErrorMustBeChunkedCacheType
	}
	rm.CacheBase.cacher = rm.cacher // need to assign to sub object

	// chunk cache window options
	rm.cacheTimeWindow = int(conf.Int64("cache_time_window", RAM_LOG_TIME_CHUNKS))
	rm.cacheChunks = int(conf.Int64("cache_num_chunks", RAM_LOG_CHUNKS))

	return nil
}

func (rm *RamLogMetric) Driver() string {
	return "ram-log"
}

func (rm *RamLogMetric) Start() {
	rm.startstop.Start(func() {

		rm.log.Notice(
			"Starting ram-log series at %d time chunks", rm.cacher.maxTime,
		)

		rm.shutitdown = false

		rm.loggerChan = rm.cacher.GetLogChan()
		rm.slicerChan = rm.cacher.GetSliceChan()

		go rm.writeLog()
		go rm.writeSlice()

		// set cache options
		rm.cacher.maxChunks = uint32(rm.cacheChunks)
		rm.cacher.maxTime = uint32(rm.cacheTimeWindow) * 60
		rm.cacher.logTime = 1 // just dump every second as we d0n't really write anything

		rm.cacher.Start()

	})
}

func (rm *RamLogMetric) Stop() {
	rm.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()
		rm.log.Warning("Starting Shutdown of ram series writer")

		if rm.shutitdown {
			return // already did
		}
		rm.shutitdown = true

		rm.cacher.Stop()

		rm.log.Warning("Shutdown finished ... quiting ram-log cache")
		return
	})
}

// listen for the broad cast chan an flush out a log entry:: noop, just bleed the channel
func (rm *RamLogMetric) writeLog() {
	for {
		_, more := <-rm.loggerChan.Ch
		if !more {
			return
		}
	}
}

// listen for the broadcast chan an flush out the newest slice in the mix: Noop just bleed the channel
func (rm *RamLogMetric) writeSlice() {
	for {
		_, more := <-rm.slicerChan.Ch
		if !more {
			return
		}
	}
}

// Write simple proxy to the cacher
func (rm *RamLogMetric) Write(stat *repr.StatRepr) error {
	return rm.WriteWithOffset(stat, nil)
}

// WriteWithOffset the offset is not relevant for this writer
func (rm *RamLogMetric) WriteWithOffset(stat *repr.StatRepr, offset *smetrics.OffsetInSeries) error {

	if rm.shutitdown {
		return nil
	}

	stat.Name.MergeMetric2Tags(rm.staticTags)
	// only need to do this if the first resolution
	if rm.currentResolution == rm.resolutions[0][0] {
		rm.indexer.Write(*stat.Name)
	}

	return rm.cacher.AddWithOffset(stat.Name, stat, offset)

}

/************************ READERS ****************/

// after the "raw" render we need to yank just the "point" we need from the data which
// will make the read-cache much smaller (will compress just the Mean value as the count is 1)
func (rm *RamLogMetric) RawDataRenderOne(ctx context.Context, metric *sindexer.MetricFindItem, start int64, end int64, resample uint32) (*smetrics.RawRenderItem, error) {
	sp, closer := rm.GetSpan("RawDataRenderOne", ctx)
	sp.LogKV("driver", "RamLogMetric", "metric", metric.StatName().Key, "uid", metric.UniqueId)
	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("reader.ramlog.rawrenderone.get-time-ns", time.Now())
	rawd := new(smetrics.RawRenderItem)

	//figure out the best res
	resolution := rm.GetResolution(start, end)
	outResolution := resolution

	//obey the bigger
	if resample > resolution {
		outResolution = resample
	}

	start = TruncateTimeTo(start, int(resolution))
	end = TruncateTimeTo(end, int(resolution))

	uStart := uint32(start)
	uEnd := uint32(end)

	statName := metric.StatName()

	rawd.Step = outResolution
	rawd.Metric = metric.Path
	rawd.Id = metric.UniqueId
	rawd.RealEnd = uEnd
	rawd.RealStart = uStart
	rawd.Start = rawd.RealStart
	rawd.End = rawd.RealEnd
	rawd.Tags = metric.Tags
	rawd.MetaTags = metric.MetaTags
	rawd.AggFunc = statName.AggType()

	if metric.Leaf == 0 {
		//data only but return a "blank" data set otherwise graphite no likey
		return rawd, ErrorNotADataNode
	}

	b_len := (uEnd - uStart) / resolution //just to be safe
	if b_len <= 0 {
		return rawd, ErrorTimeTooSmall
	}

	// the step is unknown, so set it to the min allowed
	inflight, err := rm.GetFromWriteCache(metric, uStart, uEnd, 1)

	// all we have is the cache,
	if inflight != nil && err == nil && len(inflight.Data) > 1 {
		// move the times to the "requested" ones and quantize the list
		if inflight.Step != outResolution {
			inflight.Resample(outResolution)
		}
		return inflight, err
	}

	if err != nil {
		rm.log.Error("Cassandra: Erroring getting inflight data: %v", err)
	}

	return inflight, nil
}

func (rm *RamLogMetric) RawRender(ctx context.Context, path string, start int64, end int64, tags repr.SortingTags, resample uint32) ([]*smetrics.RawRenderItem, error) {
	_, closer := rm.GetSpan("RawRender", ctx)
	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("reader.ramlog.rawrender.get-time-ns", time.Now())

	paths := SplitNamesByComma(path)
	var metrics []*sindexer.MetricFindItem

	renderWg := utils.GetWaitGroup()
	defer utils.PutWaitGroup(renderWg)

	for _, pth := range paths {
		mets, err := rm.indexer.Find(ctx, pth, tags)

		if err != nil {
			continue
		}
		metrics = append(metrics, mets...)
	}

	rawd := make([]*smetrics.RawRenderItem, 0)

	procs := CASSANDRA_DEFAULT_METRIC_RENDER_WORKERS

	jobs := make(chan *sindexer.MetricFindItem, len(metrics))
	results := make(chan *smetrics.RawRenderItem, len(metrics))

	renderOne := func(met *sindexer.MetricFindItem) *smetrics.RawRenderItem {
		_ri, err := rm.RawDataRenderOne(ctx, met, start, end, resample)

		if err != nil {
			stats.StatsdClientSlow.Incr("reader.ramlog.rawrender.errors", 1)
			rm.log.Errorf("Read Error for %s (%d->%d) : %v", path, start, end, err)
		}
		return _ri
	}

	// ye old fan out technique but not "too many" as to kill the server
	jobWorker := func(jober int, taskqueue <-chan *sindexer.MetricFindItem, resultqueue chan<- *smetrics.RawRenderItem) {
		rec_chan := make(chan *smetrics.RawRenderItem, 1)
		for met := range taskqueue {
			go func() { rec_chan <- renderOne(met) }()
			select {
			case <-time.After(rm.renderTimeout):
				stats.StatsdClientSlow.Incr("reader.ramlog.rawrender.timeouts", 1)
				rm.log.Errorf("Render Timeout for %s (%d->%d)", path, start, end)
				resultqueue <- nil
			case res := <-rec_chan:
				resultqueue <- res
			}
		}
	}

	for i := 0; i < procs; i++ {
		go jobWorker(i, jobs, results)
	}

	for _, metric := range metrics {
		jobs <- metric
	}
	close(jobs)

	for i := 0; i < len(metrics); i++ {
		res := <-results
		if res != nil {
			// this is always true for this type
			res.UsingCache = true
			rawd = append(rawd, res)
		}
	}
	close(results)
	stats.StatsdClientSlow.Incr("reader.ramlog.rawrender.metrics-per-request", int64(len(metrics)))

	return rawd, nil
}
