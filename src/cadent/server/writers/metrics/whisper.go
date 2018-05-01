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
	The whisper file writer

	NOTE:: the whisper file ITSELF takes care of rollups, NOT, so unlike the DB/CSV ones, we DO NOT
	need to bother with the flushes for anything BUT the lowest time

	[acc.agg.writer.metrics]
	driver="whisper"
	dsn="/root/metrics/path"
	[acc.agg.writer.smetrics.options]
		xFilesFactor=0.3
		write_workers=8
		write_queue_length=102400
		writes_per_second=1000 # allowed physical writes per second

*/

package metrics

import (
	"cadent/server/dispatch"
	"cadent/server/stats"
	"cadent/server/writers/indexer"
	"fmt"

	"cadent/server/broadcast"
	sindexer "cadent/server/schemas/indexer"
	smetrics "cadent/server/schemas/metrics"
	"cadent/server/schemas/repr"
	"cadent/server/utils"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"errors"
	whisper "github.com/robyoung/go-whisper"
	"golang.org/x/net/context"
	logging "gopkg.in/op/go-logging.v1"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	WHISPER_METRIC_WORKERS    = 8
	WHISPER_METRIC_QUEUE_LEN  = 1024 * 100
	WHISPER_WRITES_PER_SECOND = 1024
)

var errWhisperNotaDataNode = errors.New("Whisper: RawRenderOne: Not a data node")
var errWhisperNoDataInTimes = errors.New("Whisper: No data between these times")

// the singleton (per DSN) as we really only want one "controller" of writes not many lest we explode the
// IOwait to death

var _WHISP_WRITER_SINGLETON map[string]*WhisperWriter
var _whisp_set_mutex sync.Mutex

// special initer
func init() {
	_WHISP_WRITER_SINGLETON = make(map[string]*WhisperWriter)
}

func _get_whisp_signelton(conf *options.Options) (*WhisperWriter, error) {
	_whisp_set_mutex.Lock()
	defer _whisp_set_mutex.Unlock()
	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return nil, fmt.Errorf("Metrics: `dsn` /root/path/to/files is needed for whisper config")
	}

	if val, ok := _WHISP_WRITER_SINGLETON[dsn]; ok {
		return val, nil
	}

	writer, err := NewWhisperWriter(conf)
	if err != nil {
		return nil, err
	}
	_WHISP_WRITER_SINGLETON[dsn] = writer
	return writer, nil
}

type WhisperWriter struct {
	WriterBase

	basePath     string
	xFilesFactor float32

	retentions whisper.Retention
	indexer    indexer.Indexer

	cacheOverFlow *broadcast.Listener // on byte overflow of cacher force a write

	shutdown chan bool // when triggered, we skip the rate limiter and go full out till the queue is done

	writeQueue      chan dispatch.IJob
	dispatchQueue   chan chan dispatch.IJob
	writeDispatcher *dispatch.Dispatch
	numWorkers      int
	queueLen        int
	writesPerSecond int // number of actuall writes we will do per second assuming "INF" writes allowed

	write_lock sync.Mutex
	log        *logging.Logger
}

func NewWhisperWriter(conf *options.Options) (*WhisperWriter, error) {
	ws := new(WhisperWriter)
	ws.options = conf
	ws.log = logging.MustGetLogger("metrics.whisper")
	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return nil, fmt.Errorf("`dsn` /root/path/of/data is needed for whisper config")
	}

	ws.basePath = dsn

	// remove trialing "/"
	if strings.HasSuffix(dsn, "/") {
		ws.basePath = dsn[0 : len(dsn)-1]
	}
	ws.xFilesFactor = float32(conf.Float64("xFilesFactor", 0.3))
	ws.numWorkers = int(conf.Int64("write_workers", WHISPER_METRIC_WORKERS))
	ws.queueLen = int(conf.Int64("write_queue_length", WHISPER_METRIC_QUEUE_LEN))
	ws.writesPerSecond = int(conf.Int64("writes_per_second", WHISPER_WRITES_PER_SECOND))

	info, err := os.Stat(ws.basePath)
	if err != nil {
		return nil, fmt.Errorf("Whisper Metrics: could not find base path %s", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("Whisper Metrics: base path %s is not a directory", ws.basePath)
	}

	_cache, err := conf.ObjectRequired("cache")
	if err != nil {
		return nil, errMetricsCacheRequired
	}
	ws.cacher = _cache.(Cacher)
	_, ok := ws.cacher.(*CacherSingle)
	if !ok {
		return nil, ErrorMustBeSingleCacheType
	}

	ws.shutdown = make(chan bool, 5)
	ws.shutitdown = false

	return ws, nil
}

func (ws *WhisperWriter) getRetentions() (whisper.Retentions, error) {

	retentions := make(whisper.Retentions, 0)
	for _, item := range ws.resolutions {
		timebin := item[0]
		points := item[1]
		retent, err := whisper.ParseRetentionDef(fmt.Sprintf("%ds:%ds", timebin, points))
		if err != nil {
			return retentions, err
		}
		retentions = append(retentions, retent)
	}
	return retentions, nil
}

func (ws *WhisperWriter) getAggregateType(key string) whisper.AggregationMethod {
	agg_type := repr.GuessReprValueFromKey(key)
	switch agg_type {
	case repr.SUM:
		return whisper.Sum
	case repr.MAX:
		return whisper.Max
	case repr.MIN:
		return whisper.Min
	case repr.LAST:
		return whisper.Last
	default:
		return whisper.Average
	}
}

func (ws *WhisperWriter) getxFileFactor(key string) float32 {
	agg_type := repr.GuessReprValueFromKey(key)
	switch agg_type {
	case repr.SUM:
		return 0.0
	case repr.MAX:
		return 0.1
	case repr.MIN:
		return 0.1
	case repr.LAST:
		return 0.1
	default:
		return ws.xFilesFactor
	}
}

func (ws *WhisperWriter) getFilePath(metric string) string {
	if strings.Contains(metric, "/") {
		return metric
	}
	return filepath.Join(ws.basePath, strings.Replace(metric, ".", "/", -1)) + ".wsp"
}

func (ws *WhisperWriter) getFile(metric string) (*whisper.Whisper, error) {
	path := ws.getFilePath(metric)
	_, err := os.Stat(path)
	if err != nil {
		retentions, err := ws.getRetentions()
		if err != nil {
			return nil, err
		}
		agg := ws.getAggregateType(metric)

		// need to make the dirs
		if err = os.MkdirAll(filepath.Dir(path), os.ModeDir|os.ModePerm); err != nil {
			ws.log.Error("Could not make directory: %s", err)
			return nil, err
		}
		stats.StatsdClientSlow.Incr("writer.whisper.metric-creates", 1)
		return whisper.Create(path, retentions, agg, ws.xFilesFactor)
	}

	// need a recover in case files is corrupt and thus start over
	defer func() {
		if r := recover(); r != nil {
			ws.log.Critical("Whisper Failure (panic) %v :: need to remove file", r)
			ws.DeleteByName(metric)
		}
	}()

	return whisper.Open(path)
}

// shutdown the writer, purging all cache to files as fast as we can
func (ws *WhisperWriter) Stop() {
	ws.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()

		ws.log.Warning("Whisper: Shutting down")

		if ws.shutitdown {
			return // already did
		}

		ws.shutitdown = true
		if ws.writeDispatcher != nil {
			ws.writeDispatcher.Shutdown()
		}
		ws.shutdown <- true

		ws.cacher.Stop()

		if ws.cacher == nil {
			ws.log.Warning("Whisper: Shutdown finished, nothing in queue to write")
			return
		}

		//need to cast to get internals
		scache := ws.cacher.(*CacherSingle)

		mets := scache.Queue
		mets_l := len(mets)
		ws.log.Warning("Whisper: Shutting down, exhausting the queue (%d items) and quiting", mets_l)
		// full tilt write out
		did := 0
		for _, queueitem := range mets {
			if did%100 == 0 {
				ws.log.Warning("Whisper: shutdown purge: written %d/%d...", did, mets_l)
			}
			name, points, _ := ws.cacher.GetById(queueitem.metric)
			if points != nil {
				stats.StatsdClient.Incr(fmt.Sprintf("writer.whisper.write.send-to-writers"), 1)
				ws.InsertMulti(name, points)
			}
			did++
		}

		r_cache := GetReadCache()
		if r_cache != nil {
			ws.log.Warning("Whisper: shutdown read cache")
			r_cache.Stop()
		}

		ws.log.Warning("Whisper: shutdown purge: written %d/%d...", did, mets_l)
		ws.log.Warning("Whisper: Shutdown finished ... quiting whisper writer")
	})
}

func (ws *WhisperWriter) Start() {

	/**** start the acctuall disk writer dispatcher queue ***/
	ws.startstop.Start(func() {
		workers := ws.numWorkers
		ws.writeQueue = make(chan dispatch.IJob, ws.queueLen)
		ws.dispatchQueue = make(chan chan dispatch.IJob, workers)
		ws.writeDispatcher = dispatch.NewDispatch(workers, ws.dispatchQueue, ws.writeQueue)
		ws.writeDispatcher.SetRetries(2)
		ws.writeDispatcher.Run()

		ws.cacher.Start()
		go ws.sendToWriters() // fire up queue puller

		// set th overflow chan, and start the listener for that channel
		if ws.cacher.GetOverFlowMethod() == "chan" {
			ws.cacheOverFlow = ws.cacher.GetOverFlowChan()
			go ws.overFlowWrite()
		}
	})

}

// listen to the overflow chan from the cache and attempt to write "now"
func (ws *WhisperWriter) overFlowWrite() {
	for item := range ws.cacheOverFlow.Ch {
		// bail
		if ws.shutitdown {
			return
		}
		statitem := item.(*smetrics.TotalTimeSeries)
		// need to make a list of points from the series
		iter, err := statitem.Series.Iter()
		if err != nil {
			ws.log.Error("error in overflow writer %v", err)
			continue
		}
		pts := make(repr.StatReprSlice, 0)
		for iter.Next() {
			pts = append(pts, iter.ReprValue())
		}
		if iter.Error() != nil {
			ws.log.Error("error in overflow iterator %v", iter.Error())
		}
		ws.log.Debug("Cache overflow force write for %s you may want to do something about that", statitem.Name.Key)
		ws.InsertMulti(statitem.Name, pts)
	}
}

func (ws *WhisperWriter) InsertMulti(metric *repr.StatName, points repr.StatReprSlice) (npts int, err error) {

	whis, err := ws.getFile(metric.Key)
	if err != nil {
		ws.log.Error("Whisper:InsertMetric write error: %s", err)
		stats.StatsdClientSlow.Incr("writer.whisper.update-metric-errors", 1)
		return 0, err
	}

	// convert "points" to a whisper object
	whisp_points := make([]*whisper.TimeSeriesPoint, 0)
	agg_func := repr.GuessReprValueFromKey(metric.Key)
	for _, pt := range points {
		whisp_points = append(whisp_points,
			&whisper.TimeSeriesPoint{
				Time: int(pt.ToTime().Unix()), Value: pt.AggValue(agg_func),
			},
		)
	}

	// need a recover in case files is corrupt and thus start over
	defer func() {
		if r := recover(); r != nil {
			ws.log.Critical("Whisper Failure (panic) '%v' :: Corrupt :: need to remove file: %s", r, metric.Key)
			ws.DeleteByName(metric.Key)
			err = fmt.Errorf("%v", r)
		}
	}()

	if points != nil && len(whisp_points) > 0 {
		stats.StatsdClientSlow.GaugeAvg("writer.whisper.points-per-update", int64(len(points)))
		whis.UpdateMany(whisp_points)
	}
	stats.StatsdClientSlow.Incr("writer.whisper.update-many-writes", 1)

	whis.Close()
	return len(points), err
}

// insert the metrics from the cache
func (ws *WhisperWriter) InsertNext() (int, error) {
	defer stats.StatsdSlowNanoTimeFunc("writer.whisper.update-time-ns", time.Now())

	metric, points := ws.cacher.(*CacherSingle).Pop()
	if metric == nil || points == nil || len(points) == 0 {
		return 0, nil
	}
	return ws.InsertMulti(metric, points)
}

// the "simple" insert one metric at a time model
func (ws *WhisperWriter) InsertOne(stat repr.StatRepr) (int, error) {

	whis, err := ws.getFile(stat.Name.Key)
	if err != nil {
		ws.log.Error("Whisper write error: %s", err)
		stats.StatsdClientSlow.Incr("writer.whisper.metric-errors", 1)
		return 0, err
	}

	err = whis.Update(float64(stat.Sum), int(stat.ToTime().Unix()))
	if err != nil {
		stats.StatsdClientSlow.Incr("writer.whisper.metric-writes", 1)
	}
	whis.Close()
	return 1, err
}

// "write" just adds to the whispercache
func (ws *WhisperWriter) Write(stat *repr.StatRepr) error {

	if ws.shutitdown {
		return nil
	}

	// add to the writeback cache only
	ws.cacher.Add(stat.Name, stat)
	aggFunc := repr.GuessReprValueFromKey(stat.Name.Key)
	// and now add it to the read cache iff it's been activated
	rCache := GetReadCache()
	if rCache != nil {
		// add only the "value" we need count==1 makes some timeseries much smaller in cache
		pVal := stat.AggValue(aggFunc)
		tStat := &repr.StatRepr{
			Time:  stat.Time,
			Sum:   repr.CheckFloat(pVal),
			Count: 1,
			Name:  stat.Name,
		}
		rCache.InsertQueue <- tStat
	}
	//ws.writeQueue <- whisperMetricsJob{Stat: stat, Whis: ws}
	return nil
}

// pop from the cache and send to actual writers
func (ws *WhisperWriter) sendToWriters() error {
	// NOTE: if the writes per second is really high, this is NOT a good idea (using tickers)
	// that said, if you actually have the disk capacity to do 10k/writes per second ..  well damn
	// the extra CPU burn here may not kill you
	sleep_t := float64(time.Second) * (time.Second.Seconds() / float64(ws.writesPerSecond))
	ws.log.Notice("Starting Write limiter every %f nanoseconds (%d writes per second)", sleep_t, ws.writesPerSecond)
	ticker := time.NewTicker(time.Duration(int(sleep_t)))
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if ws.writeQueue != nil {
				stats.StatsdClient.Incr("writer.whisper.write.send-to-writers", 1)
				ws.writeQueue <- &whisperMetricsJob{Whis: ws}
			}
		case <-ws.shutdown:
			ws.log.Warning("Whisper shutdown received, stopping write loop")
			return nil
		}
	}
}

// the only recourse to handle "corrupt" files is to nuke it and start again
// otherwise panics all around for no good reason
func (ws *WhisperWriter) DeleteByName(name string) (err error) {
	path := ws.getFilePath(name)
	err = os.Remove(path)
	if err != nil {
		return err
	}
	if ws.indexer == nil {
		return nil
	}
	return ws.indexer.Delete(&repr.StatName{Key: name})
}

/****************** Main Writer Interfaces *********************/
type WhisperMetrics struct {
	WriterBase

	writer *WhisperWriter

	log *logging.Logger
}

func NewWhisperMetrics() *WhisperMetrics {
	ws := new(WhisperMetrics)
	ws.log = logging.MustGetLogger("smetrics.whisper")
	return ws
}

func (ws *WhisperMetrics) Driver() string {
	return "whisper"
}

func (ws *WhisperMetrics) Start() {
	ws.writer.Start()
}
func (ws *WhisperMetrics) Stop() {
	ws.writer.Stop()
}

func (ws *WhisperMetrics) Config(conf *options.Options) error {
	gots, err := _get_whisp_signelton(conf)
	if err != nil {
		return err
	}
	ws.writer = gots

	ws.name = conf.String("name", "metrics:whisper:"+ws.writer.basePath)
	return RegisterMetrics(ws.Name(), ws)
}

func (ws *WhisperMetrics) SetIndexer(idx indexer.Indexer) error {
	ws.indexer = idx
	ws.writer.indexer = idx // needed for "delete" actions
	return nil
}

// Resoltuions should be of the form
// [BinTime, TTL]
// we select the BinTime based on the TTL
func (ws *WhisperMetrics) SetResolutions(res [][]int) int {
	ws.resolutions = res
	ws.writer.resolutions = res
	return 1 // ONLY ONE writer needed whisper self-rolls up
}

func (ws *WhisperMetrics) SetCurrentResolution(res int) {
	ws.currentResolution = res
}

// WriteWithOffset the offset is not used here
func (ws *WhisperMetrics) WriteWithOffset(stat *repr.StatRepr, offset *smetrics.OffsetInSeries) error {
	return ws.Write(stat)
}

func (ws *WhisperMetrics) Write(stat *repr.StatRepr) error {
	ws.indexer.Write(*stat.Name) // write an index for the key
	err := ws.writer.Write(stat)

	return err
}

/**** READER ***/

func (ws *WhisperMetrics) GetFromReadCache(metric string, start int64, end int64) (rawd *smetrics.RawRenderItem, got bool) {
	rawd = new(smetrics.RawRenderItem)

	// check read cache
	r_cache := GetReadCache()
	if r_cache == nil {
		stats.StatsdClient.Incr("reader.whisper.render.cache.miss", 1)
		return rawd, false
	}

	t_start := time.Unix(int64(start), 0)
	t_end := time.Unix(int64(end), 0)
	cached_stats, _, _ := r_cache.Get(metric, t_start, t_end)
	var d_points []*smetrics.RawDataPoint
	step := uint32(0)
	if cached_stats != nil && len(cached_stats) > 0 {
		stats.StatsdClient.Incr("reader.whisper.render.cache.hits", 1)
		rawd.AggFunc = repr.GuessReprValueFromKey(metric)

		fT := uint32(0)
		for _, stat := range cached_stats {
			t := uint32(stat.ToTime().Unix())
			d_points = append(d_points, &smetrics.RawDataPoint{
				Sum:   stat.AggValue(rawd.AggFunc),
				Count: 1,
				Time:  t,
			})
			if fT == 0 {
				fT = t
			}
			if step == 0 {
				step = t - fT
			}
		}

		rawd.RealEnd = d_points[len(d_points)-1].Time
		rawd.RealStart = d_points[0].Time
		rawd.Start = rawd.RealStart
		rawd.End = rawd.RealEnd + step
		rawd.Metric = metric
		rawd.Step = step
		rawd.Data = d_points
		return rawd, len(d_points) > 0
	} else {
		stats.StatsdClient.Incr("reader.whisper.render.cache.miss", 1)
	}

	return rawd, false
}

func (ws *WhisperMetrics) RawDataRenderOne(ctx context.Context, metric *sindexer.MetricFindItem, start int64, end int64) (*smetrics.RawRenderItem, error) {
	sp, closer := ws.GetSpan("RawDataRenderOne", ctx)
	sp.LogKV("driver", "WhisperMetrics", "metric", metric.StatName().Key, "uid", metric.UniqueId)
	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("reader.whisper.renderraw.get-time-ns", time.Now())
	rawd := new(smetrics.RawRenderItem)

	if metric.Leaf == 0 {
		//data only
		return rawd, errWhisperNotaDataNode
	}

	rawd.Start = uint32(start)
	rawd.End = uint32(end)
	stat_name := metric.StatName()

	rawd.AggFunc = stat_name.AggType()
	rawd.Step = 1 // just for something in case of errors
	rawd.Metric = metric.Id
	rawd.Id = metric.UniqueId
	rawd.Tags = metric.Tags
	rawd.MetaTags = metric.MetaTags

	//cache check
	// the read cache should have "all" the points from a "start" to "end" if the read cache has been activated for
	// a while.  If not, then it's a partial list (basically the read cache just started)
	cached, got_cache := ws.GetFromReadCache(stat_name.Key, start, end)
	// we assume the "cache" is hot data (by design) so if the num points is "lacking"
	// we know we need to get to the data store (or at least the inflight) for the rest
	// add the "step" here as typically we have the "forward" step of data
	if got_cache && cached.Start <= uint32(start) {
		stats.StatsdClient.Incr("reader.whisper.render.read.cache.hit", 1)
		ws.log.Debug("Read Cache Hit: %s [%d, %d)", metric.Id, start, end)
		// need to set the start end to the ones requested before quantization to create the proper
		// spanning range
		cached.Start = uint32(start)
		cached.End = uint32(end)
		cached.AggFunc = rawd.AggFunc
		return cached, nil
	}

	// try the write inflight cache as nothing is written yet
	inflight_renderitem, err := ws.writer.cacher.GetAsRawRenderItem(stat_name)

	// need at LEAST 2 points to get the proper step size
	if inflight_renderitem != nil && err == nil {
		// move the times to the "requested" ones and quantize the list
		inflight_renderitem.Metric = metric.Id
		inflight_renderitem.Tags = metric.Tags
		inflight_renderitem.MetaTags = metric.MetaTags
		inflight_renderitem.Id = metric.UniqueId
		inflight_renderitem.AggFunc = rawd.AggFunc
		if inflight_renderitem.Start < uint32(start) {
			inflight_renderitem.RealEnd = uint32(end)
			inflight_renderitem.RealStart = uint32(start)
			inflight_renderitem.Start = inflight_renderitem.RealStart
			inflight_renderitem.End = inflight_renderitem.RealEnd
			return inflight_renderitem, err
		}
	}

	path := ws.writer.getFilePath(stat_name.Key)
	whis, err := whisper.Open(path)
	if err != nil {
		ws.writer.log.Error("Could not open file %s", path)
		return inflight_renderitem, err
	}
	defer whis.Close()

	// need a recover in case files is corrupt and thus start over
	var series *whisper.TimeSeries
	/*defer func() {
		if r := recover(); r != nil {
			ws.log.Critical("Read Whisper Failure (panic) %v :: Corrupt :: need to remove file: %s", r, path)
			ws.writer.DeleteByName(path)
			err = r.(error)
		}
	}()*/

	series, err = whis.Fetch(int(start), int(end))

	if err != nil || series == nil {
		if err == nil {
			err = errWhisperNoDataInTimes
		}
		ws.writer.log.Error("Could not get series from %s : error: %v", path, err)
		return rawd, err
	}

	points := series.Points()
	rawd.Data = make([]*smetrics.RawDataPoint, len(points))

	fT := uint32(0)
	stepT := uint32(0)

	for idx, point := range points {
		rawd.Data[idx] = &smetrics.RawDataPoint{
			Count: 1,
			Sum:   point.Value,
			Last:  point.Value,
			Min:   point.Value,
			Max:   point.Value,
			Time:  uint32(point.Time),
		}
		if fT == 0 {
			fT = uint32(point.Time)
		}
		if stepT == 0 {
			stepT = uint32(point.Time) - fT
		}
	}

	rawd.RealEnd = uint32(end)
	rawd.RealStart = uint32(start)
	rawd.Step = stepT

	// grab the "current inflight" from the cache and merge into the main array
	if inflight_renderitem != nil && len(inflight_renderitem.Data) > 1 {
		inflight_renderitem.Step = stepT // need to force this step size
		//merge with any inflight bits (inflight has higher precedence over the file)
		inflight_renderitem.MergeWithResample(rawd, stepT)
		return inflight_renderitem, nil
	}

	return rawd, nil
}

func (ws *WhisperMetrics) RawRender(ctx context.Context, path string, from int64, to int64, tags repr.SortingTags, resample uint32) ([]*smetrics.RawRenderItem, error) {
	sp, closer := ws.GetSpan("RawRender", ctx)
	sp.LogKV("driver", "WhisperMetrics", "path", path, "from", from, "to", to)
	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("reader.whisper.rawrender.get-time-ns", time.Now())

	paths := SplitNamesByComma(path)
	var metrics []*sindexer.MetricFindItem

	render_wg := utils.GetWaitGroup()
	defer utils.PutWaitGroup(render_wg)

	for _, pth := range paths {
		mets, err := ws.indexer.Find(ctx, pth, tags)
		if err != nil {
			ws.log.Error("error in find: %v", err)
			continue
		}
		metrics = append(metrics, mets...)
	}

	rawd := make([]*smetrics.RawRenderItem, len(metrics))

	// ye old fan out technique
	render_one := func(metric *sindexer.MetricFindItem, idx int) {
		defer render_wg.Done()
		_ri, err := ws.RawDataRenderOne(ctx, metric, from, to)

		if err != nil {
			return
		}
		rawd[idx] = _ri
	}

	for idx, metric := range metrics {
		render_wg.Add(1)
		go render_one(metric, idx)
	}
	render_wg.Wait()
	stats.StatsdClientSlow.Incr("reader.whisper.rawrender.metrics-per-request", int64(len(metrics)))

	return rawd, nil
}

func (ws *WhisperMetrics) CacheRender(ctx context.Context, path string, from int64, to int64, tags repr.SortingTags) ([]*smetrics.RawRenderItem, error) {
	return nil, ErrWillNotBeimplemented
}
func (ws *WhisperMetrics) CachedSeries(path string, from int64, to int64, tags repr.SortingTags) (*smetrics.TotalTimeSeries, error) {
	return nil, ErrWillNotBeimplemented
}

/************************************************************************/
/**********  Standard Worker Dispatcher JOB   ***************************/
/************************************************************************/
// insert job queue workers
type whisperMetricsJob struct {
	Whis  *WhisperWriter
	Retry int
}

func (j *whisperMetricsJob) IncRetry() int {
	j.Retry++
	return j.Retry
}
func (j *whisperMetricsJob) OnRetry() int {
	return j.Retry
}

func (j *whisperMetricsJob) DoWork() error {
	_, err := j.Whis.InsertNext()
	if err != nil {
		j.Whis.log.Error("Insert failed for Metric: %v retrying ...", err)
	}
	return err
}
