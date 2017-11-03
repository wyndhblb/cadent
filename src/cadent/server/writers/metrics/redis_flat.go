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
	The Redis flat stat write


	OPTIONS: For `Config`

		db="0" #  base table name (default: metrics)
		batch_count=1000 # batch this many inserts for much faster insert performance (default 1000)
		periodic_flush="1s" # regardless of if batch_count met always flush things at this interval (default 1s)

		## create a new index based on the time of the point
		# of the form "{metrics}_{resolution}s-{YYYY-MM-DD}"
		# default is "none"
		index_per_date = "none|week|month|day|hourly"


*/

package metrics

import (
	"cadent/server/broadcast"
	sindexer "cadent/server/schemas/indexer"
	smetrics "cadent/server/schemas/metrics"
	"cadent/server/schemas/repr"
	"cadent/server/stats"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/dbs"
	"fmt"
	"golang.org/x/net/context"
	logging "gopkg.in/op/go-logging.v1"
	"gopkg.in/redis.v5"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	REDIS_DEFAULT_METRIC_RENDER_WORKERS = 4
	REDIS_RENDER_TIMEOUT                = "5s"
	REDIS_FLUSH_TIME_SECONDS            = 5
)

// little object to let us have "multiple" piplines
type redisPipe struct {
	conn        *redis.Client
	clusterconn *redis.ClusterClient
	pipe        *redis.Pipeline
	idx         int
	inqueue     *int32
	lock        sync.Mutex
}

var rdBytesPool sync.Pool

// pools of preallocated byte arrays for zset keys
func rdGetBytes() []byte {
	x := esBytesPool.Get()
	if x == nil {
		buf := make([]byte, 128)
		return buf[:0]
	}
	buf := x.([]byte)
	return buf[:0]
}

func rdPutBytes(buf []byte) {
	esBytesPool.Put(buf)
}

/****************** RedisFlatMetrics *********************/
type RedisFlatMetrics struct {
	WriterBase

	db   *dbs.RedisDB
	conn *redis.Client

	writeList []*repr.StatRepr // buffer the writes so as to do "multi" inserts per query

	maxWriteSize   int32         // size of that buffer before a flush
	maxIdle        time.Duration // either maxWriteSize will trigger a write or this time passing will
	workers        int
	periodicTick   *time.Ticker
	metricAddQueue chan *repr.StatRepr // can provide some back pressure on flush writing if it takes too long
	writeLock      sync.Mutex
	renderTimeout  time.Duration

	indexPerDate string

	pipes []*redisPipe // pipeline batches

	log *logging.Logger

	shutdown *broadcast.Broadcaster
}

func NewRedisFlatMetrics() *RedisFlatMetrics {
	rd := new(RedisFlatMetrics)
	rd.log = logging.MustGetLogger("writers.redisflat")
	rd.shutdown = broadcast.New(1)
	rd.workers = 8
	return rd
}

func (rd *RedisFlatMetrics) Config(conf *options.Options) error {
	rd.options = conf

	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("`dsn` host:port is needed for redis config")
	}

	dbKey := dsn + conf.String("metric_index", "metrics")

	db, err := dbs.NewDB("redis", dbKey, conf)
	if err != nil {
		return err
	}

	rd.db = db.(*dbs.RedisDB)
	rd.conn = rd.db.SingleClient

	res, err := conf.Int64Required("resolution")
	if err != nil {
		return fmt.Errorf("Resolution needed for redis writer")
	}

	rd.name = conf.String("name", "metrics:redis:"+dsn)

	// FLAT writers have different "writers" for each resolution,  so we need to set different names for them
	// the API readers use the "first one", so if things already exist w/ the name, slap on a resolution tag
	if GetMetrics(rd.Name()) != nil {
		rd.name = conf.String("name", "metrics:redis:"+dsn) + fmt.Sprintf(":%v", res)
	}
	err = RegisterMetrics(rd.Name(), rd)
	if err != nil {
		rd.log.Critical(err.Error())
		return err
	}

	//need to hide the usr/pw from things
	p, _ := url.Parse(dsn)
	cacheKey := fmt.Sprintf("redisflat:cache:%s/%s:%v", p.Host, conf.String("table", "metrics"), res)
	_cache, err := GetCacherSingleton(cacheKey, "single")
	if _cache == nil {
		return errMetricsCacheRequired
	}

	if err != nil {
		return err
	}

	scacher, ok := _cache.(*CacherSingle)
	if !ok {
		return ErrorMustBeSingleCacheType
	}
	rd.cacher = scacher

	rd.maxWriteSize = int32(conf.Int64("batch_count", 1000))
	rd.maxIdle = conf.Duration("periodic_flush", time.Duration(REDIS_FLUSH_TIME_SECONDS*time.Second))
	_tgs := conf.String("tags", "")
	if len(_tgs) > 0 {
		rd.staticTags = repr.SortingTagsFromString(_tgs)
	}

	rdur, err := time.ParseDuration(REDIS_RENDER_TIMEOUT)
	if err != nil {
		return err
	}
	rd.renderTimeout = rdur

	// hourly, daily, weekly, monthly, none
	rd.indexPerDate = conf.String("index_by_date", "hourly")
	rd.workers = int(conf.Int64("workers", 8))

	rd.metricAddQueue = make(chan *repr.StatRepr, rd.maxWriteSize)
	rd.periodicTick = time.NewTicker(rd.maxIdle)

	rd.pipes = make([]*redisPipe, rd.workers)
	for i := 0; i < rd.workers; i++ {
		rd.pipes[i] = new(redisPipe)
		switch rd.db.Mode() {
		case "cluster":
			rd.pipes[i].clusterconn = rd.db.NewCluster()
			rd.pipes[i].pipe = rd.pipes[i].conn.Pipeline()
		default:
			rd.pipes[i].conn = rd.db.NewSingle()
			rd.pipes[i].pipe = rd.pipes[i].conn.Pipeline()
		}
		rd.pipes[i].inqueue = new(int32)
		rd.pipes[i].idx = i
	}
	return nil
}

func (rd *RedisFlatMetrics) Driver() string {
	return "redis-flat"
}

func (rd *RedisFlatMetrics) Stop() {
	rd.startstop.Stop(func() {
		rd.cacher.Stop()

		scache := rd.cacher.(*CacherSingle)

		mets := scache.Queue
		metsL := len(mets)
		rd.log.Warning("RedisFlat: Shutting down, exhausting the queue (%d items) and quiting", metsL)
		// full tilt write out
		did := 0
		for i, queueitem := range mets {
			if did%100 == 0 {
				rd.log.Notice("RedisFlat: shutdown purge: written %d/%d...", did, metsL)
			}
			name, points, _ := rd.cacher.GetById(queueitem.metric)
			bulkR := rd.workers % i
			if points != nil {
				stats.StatsdClient.Incr(fmt.Sprintf("writer.whisper.write.send-to-writers"), 1)
				rd.addToBulkMulti(rd.pipes[bulkR], name, points)
			}
			did++
		}

		// need to close this many
		for i := 0; i < rd.workers; i++ {
			shutdown.AddToShutdown()
		}
		rd.shutdown.Close()
	})
	return
}

func (rd *RedisFlatMetrics) Start() {
	rd.startstop.Start(func() {
		rd.cacher.Start()
		for i := 0; i < rd.workers; i++ {
			go rd.periodFlush(rd.pipes[i])
			go rd.pushStat(rd.pipes[i])
		}
		go rd.sendToWriters()

	})
}

func (rd *RedisFlatMetrics) periodFlush(pipe *redisPipe) {
	shuts := rd.shutdown.Listen()

	for {
		select {
		case <-rd.periodicTick.C:
			rd.flush(pipe)
		case <-shuts.Ch:
			rd.log.Notice("Got shutdown, doing final flush for %s...", rd.Name())
			//rd.cacher.Stop() // no cache here
			rd.flush(pipe)

			rd.log.Notice("Done final flush for %s", rd.Name())
			shutdown.ReleaseFromShutdown()
			rd.log.Notice("Done shutdown for %s", rd.Name())
			return
		}
	}
}

// prefixName name based on the incoming time
// for redis it is {metricPrefix}:{resolution}:{date}:{uid}
func (rd *RedisFlatMetrics) prefixName(res uint32, inTime time.Time) string {
	dstr := ""
	switch rd.indexPerDate {
	case "hourly":
		dstr = inTime.Format("2006010215")
	case "daily":
		dstr = inTime.Format("20060102")
	case "weekly":
		ynum, wnum := inTime.ISOWeek()
		dstr = fmt.Sprintf("%04d%02d", ynum, wnum)
	case "monthly":
		dstr = inTime.Format("200601")
	}
	return fmt.Sprintf("%s:%d:%s:", rd.db.Tablename(), res, dstr)
}

// from the start times get the list of indexes we need to query
func (rd *RedisFlatMetrics) prefixRange(res uint32, sTime time.Time, eTime time.Time) []string {

	outStr := []string{}
	onT := sTime
	switch rd.indexPerDate {
	case "hourly":
		eTime = eTime.Add(time.Hour) // need to include the end
		for onT.Before(eTime) {
			outStr = append(outStr, fmt.Sprintf("%s:%d:%s:", rd.db.Tablename(), res, onT.Format("2006010215")))
			onT = onT.Add(time.Hour)
		}
		return outStr
	case "daily":
		eTime = eTime.AddDate(0, 0, 1) // need to include the end
		for onT.Before(eTime) {
			outStr = append(outStr, fmt.Sprintf("%s:%d:%s:", rd.db.Tablename(), res, onT.Format("20060102")))
			onT = onT.AddDate(0, 0, 1)
		}
		return outStr
	case "weekly":
		eTime = eTime.AddDate(0, 0, 7) // need to include the end
		for onT.Before(eTime) {
			ynum, wnum := onT.ISOWeek()
			outStr = append(outStr, fmt.Sprintf("%s:%d:%04d%02d:", rd.db.Tablename(), res, ynum, wnum))
			onT = onT.AddDate(0, 0, 7)
		}
		return outStr
	case "monthly":
		eTime = eTime.AddDate(0, 1, 0) // need to include the end
		for onT.Before(eTime) {
			outStr = append(outStr, fmt.Sprintf("%s:%d:%s:", rd.db.Tablename(), res, onT.Format("200601")))
			onT = onT.AddDate(0, 1, 0)
		}
		return outStr
	}
	return []string{fmt.Sprintf("%s:%d:", rd.db.Tablename(), res)}
}

// pop from the cache and send to actual writers
func (rd *RedisFlatMetrics) sendToWriters() error {
	// NOTE: if the writes per second is really high, this is NOT a good idea (using tickers)
	// that said, if you actually have the disk capacity to do 10k/writes per second ..  well damn
	// the extra CPU burn here may not kill you
	shuts := rd.shutdown.Listen()
	pId := 0
	for {
		select {
		case <-shuts.Ch:
			rd.log.Warning("Redis shutdown received, stopping write loop")
			return nil
		default:
			if pId >= rd.workers {
				pId = 0
			}
			metric, points := rd.cacher.(*CacherSingle).Pop()
			if metric == nil || points == nil || len(points) == 0 {
				time.Sleep(time.Second) // snooze a bit to go in a bunch of nutty loops when empty
				continue
			}
			rd.addToBulkMulti(rd.pipes[pId], metric, points)
			pId++
		}
	}
}

func (rd *RedisFlatMetrics) addToBulkMulti(pipe *redisPipe, name *repr.StatName, points repr.StatReprSlice) {
	for _, p := range points {
		p.Name = name
		rd.addToBulk(pipe, p)
	}
}

// For redis we store things as a Zset (sorted set), where the "sort" is just the timestamp
// and the data is a "squuezed" mush of the {time}:count:sum:min:max:last
// We store things in a batch to allow for pipelining in redis

func (rd *RedisFlatMetrics) addToBulk(pipe *redisPipe, stat *repr.StatRepr) {
	pipe.lock.Lock()
	defer pipe.lock.Unlock()

	uid := stat.Name.UniqueIdString()
	sTime := stat.ToTime()

	//{metric}:{resolution}:{dategroup}:{uid}
	pName := rd.prefixName(stat.Name.Resolution, sTime)
	hName := rd.db.Tablename() + ":" + "dates"
	uidList := pName + "uids"
	var prefix string

	// cluster key is {uid} in cluster mode
	switch rd.db.Mode() {
	case "cluster":
		prefix = pName + "{" + uid + "}"
	default:
		prefix = pName + uid
	}

	bs := rdGetBytes()

	if stat.Count == 1 {
		bs = strconv.AppendInt(bs, stat.Time, 10)
		bs = append(bs, ':')
		bs = strconv.AppendFloat(bs, stat.Sum, 'f', -1, 64)
	} else {
		bs = strconv.AppendInt(bs, stat.Time, 10)
		bs = append(bs, ':')
		bs = strconv.AppendInt(bs, stat.Count, 10)
		bs = append(bs, ':')
		bs = strconv.AppendFloat(bs, stat.Sum, 'f', -1, 64)
		bs = append(bs, ':')
		bs = strconv.AppendFloat(bs, stat.Min, 'f', -1, 64)
		bs = append(bs, ':')
		bs = strconv.AppendFloat(bs, stat.Max, 'f', -1, 64)
		bs = append(bs, ':')
		bs = strconv.AppendFloat(bs, stat.Last, 'f', -1, 64)
	}
	zKey := string(bs)
	rdPutBytes(bs)

	_, err := pipe.pipe.ZAdd(prefix, redis.Z{Score: float64(stat.Time), Member: zKey}).Result()
	if err != nil {
		rd.log.Error("Redis Failure to insert: %s -> %s: %v", prefix, zKey, err)
	}
	if stat.Name.Ttl > 0 {
		_, err = pipe.pipe.Expire(prefix, time.Second*time.Duration(stat.Name.Ttl)).Result()
		if err != nil {
			rd.log.Error("Redis Expire set falure: %s -> %s: %v", prefix, zKey, err)
		}
	}
	if err == nil {
		// add to the "list" of dates we have
		pipe.pipe.SAdd(hName, pName).Result()
		pipe.pipe.SAdd(uidList, uid).Result()
	}
	atomic.AddInt32(pipe.inqueue, 1)

}

func (rd *RedisFlatMetrics) flush(pipe *redisPipe) (int, error) {
	pipe.lock.Lock()
	defer pipe.lock.Unlock()
	l := atomic.LoadInt32(pipe.inqueue)
	if l == 0 {
		return 0, nil
	}
	_, err := pipe.pipe.Exec()
	if err != nil {
		rd.log.Errorf("Failed pipline execution: %v", err)
	} else {
		rd.log.Debug("Flushed %d stats to redis pipe: %d", l, pipe.idx)
	}
	pipe.pipe = rd.conn.Pipeline()
	atomic.StoreInt32(pipe.inqueue, 0)
	return 1, nil

}

// provides some backpression on Write in case ES gets really slow
func (rd *RedisFlatMetrics) pushStat(pipe *redisPipe) {
	shuts := rd.shutdown.Listen()
	onPipe := pipe
	for {
		select {
		case stat := <-rd.metricAddQueue:
			l := atomic.LoadInt32(onPipe.inqueue)
			if l > rd.maxWriteSize {
				rd.flush(onPipe)
			}
			s := stat
			rd.addToBulk(onPipe, s)

			// only the lowest res needs to write the index
			if rd.ShouldWriteIndex() {
				rd.indexer.Write(*stat.Name) // to the indexer
			}
		case <-shuts.Ch:
			return
		}
	}
}

func (rd *RedisFlatMetrics) Write(stat *repr.StatRepr) error {
	return rd.WriteWithOffset(stat, nil)
}

func (rd *RedisFlatMetrics) WriteWithOffset(stat *repr.StatRepr, offset *smetrics.OffsetInSeries) error {

	stat.Name.MergeMetric2Tags(rd.staticTags)
	rd.cacher.AddWithOffset(stat.Name, stat, offset)
	rd.metricAddQueue <- stat
	return nil
}

/**** READER ***/

func (rd *RedisFlatMetrics) RawRenderOne(ctx context.Context, metric *sindexer.MetricFindItem, start int64, end int64, resample uint32) (*smetrics.RawRenderItem, error) {
	sp, closer := rd.GetSpan("RawRenderOne", ctx)
	sp.LogKV("driver", "RedisFlatMetrics", "metric", metric.StatName().Name(), "uid", metric.UniqueId)

	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("reader.redisflat.renderraw.get-time-ns", time.Now())

	rawd := new(smetrics.RawRenderItem)

	if metric.Leaf == 0 { //data only
		return rawd, fmt.Errorf("RawRenderOne: Not a data node")
	}

	//figure out the best res
	resolution := rd.GetResolution(start, end)
	outResolution := resolution

	//obey the bigger
	if resample > resolution {
		outResolution = resample
	}

	start = TruncateTimeTo(start, int(resolution))
	end = TruncateTimeTo(end, int(resolution))

	bLen := uint32(end-start) / resolution //just to be safe
	if bLen <= 0 {
		return rawd, fmt.Errorf("time too narrow")
	}

	firstT := uint32(start)
	lastT := uint32(end)

	// try the write inflight cache as nothing is written yet
	stat_name := metric.StatName()
	inflightRenderitem, err := rd.cacher.GetAsRawRenderItem(stat_name)

	// need at LEAST 2 points to get the proper step size
	if inflightRenderitem != nil && err == nil {
		// move the times to the "requested" ones and quantize the list
		inflightRenderitem.Metric = metric.Id
		inflightRenderitem.Tags = metric.Tags
		inflightRenderitem.MetaTags = metric.MetaTags
		inflightRenderitem.Id = metric.UniqueId
		inflightRenderitem.AggFunc = stat_name.AggType()
		if inflightRenderitem.Start < uint32(start) {
			inflightRenderitem.RealEnd = uint32(end)
			inflightRenderitem.RealStart = uint32(start)
			inflightRenderitem.Start = inflightRenderitem.RealStart
			inflightRenderitem.End = inflightRenderitem.RealEnd
			return inflightRenderitem, err
		}
	}

	// sorting order for the table is time ASC (i.e. firstT == first entry)
	// on resamples (if >0 ) we simply merge points until we hit the time steps
	doResample := resample > 0 && resample > resolution

	onTime := time.Unix(start, 0)
	endTime := time.Unix(end, 0)
	usePrefixes := rd.prefixRange(resolution, onTime, endTime)
	mKey := metric.Id

	tStart := uint32(start)
	curPt := smetrics.NullRawDataPoint(tStart)
	var min, max, last, sum float64
	var ct int64
	var getKey string

	for _, prefix := range usePrefixes {

		// cluster key is {uid}
		switch rd.db.Mode() {
		case "cluster":
			getKey = prefix + "{" + metric.UniqueId + "}"
		default:
			getKey = prefix + metric.UniqueId
		}

		res, err := rd.conn.ZRange(getKey, 0, -1).Result()
		if err != nil {
			rd.log.Error("Failure in Zrange get: %v", err)
			continue
		}

		for _, r := range res {
			items := strings.Split(r, ":")
			if len(items) < 2 {
				rd.log.Error("Failure in Zrange get Invalid Value: %s", r)
				continue
			}
			onT, err := strconv.ParseInt(items[0], 10, 64)
			if err != nil {
				rd.log.Error("Failure in Zrange get Invalid Time: %s", items[0])
				continue
			}

			t := uint32(time.Unix(0, onT).Unix())
			if t < uint32(start)-resample {
				continue
			}

			if t > uint32(end)+resample {
				break
			}

			//count:sum:min:max:last
			if len(items) == 6 {
				ct, _ = strconv.ParseInt(items[1], 10, 64)
				sum, _ = strconv.ParseFloat(items[2], 64)
				min, _ = strconv.ParseFloat(items[3], 64)
				max, _ = strconv.ParseFloat(items[4], 64)
				last, _ = strconv.ParseFloat(items[5], 64)
			} else {
				sum, _ = strconv.ParseFloat(items[1], 64)
				ct = 1
				min = sum
				max = sum
				last = sum
			}

			if doResample {
				if t >= tStart+resample {
					tStart += resample
					rawd.Data = append(rawd.Data, curPt)
					curPt = &smetrics.RawDataPoint{
						Count: ct,
						Sum:   sum,
						Max:   max,
						Min:   min,
						Last:  last,
						Time:  t,
					}
				} else {
					curPt.Merge(&smetrics.RawDataPoint{
						Count: ct,
						Sum:   sum,
						Max:   max,
						Min:   min,
						Last:  last,
						Time:  t,
					})
				}
			} else {
				rawd.Data = append(rawd.Data, &smetrics.RawDataPoint{
					Count: ct,
					Sum:   sum,
					Max:   max,
					Min:   min,
					Last:  last,
					Time:  t,
				})
			}
			lastT = t
		}
	}

	if !curPt.IsNull() {
		rawd.Data = append(rawd.Data, curPt)
	}
	if len(rawd.Data) > 0 && rawd.Data[0].Time > 0 {
		firstT = rawd.Data[0].Time
	}

	//cass.log.Critical("METR: %s Start: %d END: %d LEN: %d GotLen: %d", metric.Id, firstT, lastT, len(d_points), ct)

	rawd.RealEnd = uint32(lastT)
	rawd.RealStart = uint32(firstT)
	rawd.Start = uint32(start)
	rawd.End = uint32(end)
	rawd.Step = outResolution
	rawd.Metric = mKey
	rawd.Tags = metric.Tags
	rawd.MetaTags = metric.MetaTags
	rawd.Id = metric.UniqueId
	rawd.AggFunc = stat_name.AggType()

	// grab the "current inflight" from the cache and merge into the main array
	if inflightRenderitem != nil && len(inflightRenderitem.Data) > 1 {
		//merge with any inflight bits (inflight has higher precedence over the file)
		inflightRenderitem.MergeWithResample(rawd, outResolution)
		return inflightRenderitem, nil
	}

	return rawd, nil
}

func (rd *RedisFlatMetrics) RawRender(ctx context.Context, path string, from int64, to int64, tags repr.SortingTags, resample uint32) ([]*smetrics.RawRenderItem, error) {
	sp, closer := rd.GetSpan("RawRender", ctx)
	sp.LogKV("driver", "RedisFlatMetrics", "path", path, "from", from, "to", to)

	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("reader.redisflat.rawrender.get-time-ns", time.Now())

	paths := SplitNamesByComma(path)
	var metrics []*sindexer.MetricFindItem

	for _, pth := range paths {
		mets, err := rd.indexer.Find(ctx, pth, tags)
		if err != nil {
			continue
		}
		metrics = append(metrics, mets...)
	}

	rawd := make([]*smetrics.RawRenderItem, 0, len(metrics))

	procs := REDIS_DEFAULT_METRIC_RENDER_WORKERS

	jobs := make(chan *sindexer.MetricFindItem, len(metrics))
	results := make(chan *smetrics.RawRenderItem, len(metrics))

	renderOne := func(met *sindexer.MetricFindItem) *smetrics.RawRenderItem {
		_ri, err := rd.RawRenderOne(ctx, met, from, to, resample)

		if err != nil {
			stats.StatsdClientSlow.Incr("reader.redisflat.rawrender.errors", 1)
			rd.log.Error("Read Error for %s (%d->%d) : %v", path, from, to, err)
		}
		return _ri
	}

	// ye old fan out technique but not "too many" as to kill the server
	jobWorker := func(jober int, taskqueue <-chan *sindexer.MetricFindItem, resultqueue chan<- *smetrics.RawRenderItem) {
		rec_chan := make(chan *smetrics.RawRenderItem, 1)
		for met := range taskqueue {
			go func() { rec_chan <- renderOne(met) }()
			select {
			case <-time.After(rd.renderTimeout):
				stats.StatsdClientSlow.Incr("reader.redisflat.rawrender.timeouts", 1)
				rd.log.Errorf("Render Timeout for %s (%d->%d)", path, from, to)
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
			rawd = append(rawd, res)
		}
	}
	close(results)
	stats.StatsdClientSlow.Incr("reader.redisflat.rawrender.metrics-per-request", int64(len(metrics)))

	return rawd, nil
}

func (rd *RedisFlatMetrics) CacheRender(ctx context.Context, path string, from int64, to int64, tags repr.SortingTags) ([]*smetrics.RawRenderItem, error) {
	return nil, ErrWillNotBeimplemented
}
func (rd *RedisFlatMetrics) CachedSeries(path string, from int64, to int64, tags repr.SortingTags) (*smetrics.TotalTimeSeries, error) {
	return nil, ErrWillNotBeimplemented
}
