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
	The Cassandra - Flat Map Metric Reader/Writer

	Much like the Flat writer except we take a slightly different storage model

	Rather then a point per row, we store a MAP of points in a row per hour where the map is simply

	map<time, metric_point>

	Thus we index things with a new column (if table per resolution, resolution is dropped)

	[uid] [resolution] [YYYYMMDDHH] [map]

	To update things we simple do the nice map update in cassandra

	map = map + {time: point}

	A Schema for one to use

	CREATE TYPE metric_point (
            max double,
            min double,
            sum double,
            last double,
            count int
        );



        CREATE TABLE metric.metric (
            uid ascii,
            res int,
            slab ascii
            points map<int, frozen<metric_point>>,
            PRIMARY KEY (uid, res, slab)
        ) WITH COMPACT STORAGE
            AND CLUSTERING ORDER BY (slab ASC)
            AND compaction = {
                'class': 'DateTieredCompactionStrategy',
                'min_threshold': '12',
                'max_threshold': '32',
                'max_sstable_age_days': '0.083',
                'base_time_seconds': '50'
            }
            AND compression = {'sstable_compression': 'org.apache.cassandra.io.compress.LZ4Compressor'};

	CONFIG options::

	[graphite-cassandra-flat.accumulator.writer.metrics]
	driver="cassandra-flat-map"
	dsn="my.cassandra.com"


*/

package metrics

import (
	"cadent/server/broadcast"
	sindexer "cadent/server/schemas/indexer"
	smetrics "cadent/server/schemas/metrics"
	"cadent/server/schemas/repr"
	"cadent/server/stats"
	"cadent/server/utils"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/dbs"
	"fmt"
	"github.com/cenkalti/backoff"
	"github.com/gocql/gocql"
	"golang.org/x/net/context"
	logging "gopkg.in/op/go-logging.v1"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type CassandraFlatMapWriter struct {
	// just the writer connections for this
	db   *dbs.CassandraDB
	conn *gocql.Session

	writeQueue chan *cassandraFlatMetricJob

	cacher                *CacherSingle
	cacheOverFlowListener *broadcast.Listener // on byte overflow of cacher force a write

	shutitdown int32 // just a flag
	shutdown   *broadcast.Broadcaster
	startstop  utils.StartStop

	writesPerSecond int // allowed writes per second
	numWorkers      int
	queueLen        int
	writeLock       sync.Mutex
	log             *logging.Logger

	tablePerResolution bool
	indexPerDate       string // hourly, daily, weekly, monthly

	resolutions       [][]int
	currentResolution int

	insertQuery     string //render once
	selectTimeQuery string //render once
	getQuery        string //render once
}

func NewCassandraFlatMapWriter(conf *options.Options, resolutions [][]int) (*CassandraFlatMapWriter, error) {

	cass := new(CassandraFlatMapWriter)
	cass.log = logging.MustGetLogger("metrics.cassandraflatmap")
	cass.shutitdown = 0
	cass.shutdown = broadcast.New(1)

	gots, err := conf.StringRequired("dsn")
	if err != nil {
		return nil, fmt.Errorf("Metrics: `dsn` (server1,server2,server3) is needed for cassandra config")
	}

	tRes, err := conf.Int64Required("resolution")
	if err != nil || tRes == 0 {
		return nil, fmt.Errorf("Metrics: `resolutions` required is needed for cassandra config")
	}
	cass.currentResolution = int(tRes)

	connKey := fmt.Sprintf("%v:%v/%v/%v", gots, conf.Int64("port", 9042), conf.String("keyspace", "metric"), conf.String("metrics_table", "metrics"))
	cass.log.Notice("Connecting Metrics to Cassandra (%s)", connKey)

	cass.resolutions = resolutions

	db, err := dbs.NewDB("cassandra", connKey, conf)
	if err != nil {
		return nil, err
	}

	// need to cast for real usage
	cass.db = db.(*dbs.CassandraDB)
	cass.conn, err = cass.db.GetSession()
	if err != nil {
		return nil, err
	}

	// tweak queues and worker sizes
	cass.numWorkers = int(conf.Int64("write_workers", CASSANDRA_FLAT_METRIC_WORKERS))
	cass.queueLen = int(conf.Int64("write_queue_length", CASSANDRA_FLAT_METRIC_QUEUE_LEN))

	cass.writesPerSecond = int(conf.Int64("writes_per_second", CASSANDRA_FLAT_WRITES_PER_SECOND))
	cass.tablePerResolution = conf.Bool("table_per_resolution", CASSANDRA_FLAT_DEFAULT_TABLE_PER_RESOLUTION)

	cass.indexPerDate = conf.String("index_by_date", "hourly")

	cass.insertQuery = "UPDATE %s %s SET pts = pts + ? WHERE uid=? AND res=? AND slab=? AND ord=?"

	cass.selectTimeQuery = "SELECT pts FROM %s WHERE uid=? AND res=? AND slab IN (%s)"

	if cass.tablePerResolution {
		cass.insertQuery = "UPDATE %s %s SET pts = pts + ? WHERE uid=? AND slab=? AND ord=?"

		cass.selectTimeQuery = "SELECT pts FROM %s WHERE uid=? AND slab IN (%s)"
	}

	return cass, nil
}

func (cass *CassandraFlatMapWriter) Stop() {
	cass.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()

		cass.log.Warning("Starting Shutdown of writer")
		if atomic.LoadInt32(&cass.shutitdown) == 1 {
			return // already did
		}
		cass.cacher.Stop()

		atomic.StoreInt32(&cass.shutitdown, 1)

		if cass.writeQueue != nil {
			for i := 0; i < cass.numWorkers; i++ {
				shutdown.AddToShutdown()
			}
		}

		cass.shutdown.Close()

		did := 0

		//bleed the cass.writeQueue
		l := len(cass.writeQueue)
		cass.log.Warning("Shutting down, exhausting work queue (%d items)", l)
		if cass.writeQueue != nil && l > 0 {
			for job := range cass.writeQueue {
				if did%100 == 0 {
					cass.log.Warning("shutdown purge current queue: written %d/%d...", did, l)
				}
				cass.InsertMulti(job.Name, job.Stats)
				did++
				if did >= l {
					break
				}
			}
		}

		close(cass.writeQueue)

		mets := cass.cacher.Queue
		metsL := len(mets)
		cass.log.Warning("Shutting down, exhausting the queue (%d items) and quiting", metsL)
		// full tilt write out
		for _, queueitem := range mets {

			name, points, _ := cass.cacher.GetById(queueitem.metric)
			if points != nil {
				stats.StatsdClient.Incr(fmt.Sprintf("writer.cassandraflatmap.write.send-to-writers"), 1)
				cass.InsertMulti(name, points)
			}
			did++
		}
		cass.conn.Close()

		cass.log.Warning("shutdown purge: written %d/%d...", did, metsL)
		cass.log.Warning("Shutdown finished ... quiting cassandra writer")
	})
}

// Start the writer, needs to be called before anything will run/work
func (cass *CassandraFlatMapWriter) Start() {
	/**** dispatcher queue ***/
	cass.startstop.Start(func() {

		// schemas
		cass.log.Notice(
			"Starting cassandra-flat-map writer for %s: Write mode: %v resolution: %d",
			cass.db.MetricTable(), cass.db.Cluster().Consistency, cass.currentResolution,
		)

		cass.log.Notice("Adding metric tables resolutions: %v...", cass.resolutions)

		schems := NewCassandraMetricsSchema(
			cass.conn,
			cass.db.Keyspace(),
			cass.db.MetricTable(),
			cass.resolutions,
			"flatset",
			cass.tablePerResolution,
			cass.db.GetCassandraVersion(),
		)

		err := schems.AddMetricsTable()

		if err != nil {
			panic(err)
		}

		cass.writeQueue = make(chan *cassandraFlatMetricJob, cass.numWorkers)
		for i := 0; i < cass.numWorkers; i++ {
			go cass.consumeWriteQueue(i)
		}

		cass.cacher.Start()
		go cass.sendToWriters() // the dispatcher
	})
}

func (cass *CassandraFlatMapWriter) consumeWriteQueue(wnum int) {
	shuts := cass.shutdown.Listen()
	retryNotify := func(err error, after time.Duration) {
		stats.StatsdClientSlow.Incr("writer.cassandraflatmap.update.metric-retry", 1)
		cass.log.Error("Write failed will retry :: %v : after %v", err, after)
	}
	w := wnum
	cass.log.Notice("Starting write worker #%d resolution %d", w, cass.currentResolution)
	for {
		select {
		case <-shuts.Ch:
			cass.log.Warningf("Shutting down write worker #%d resolution %d", w, cass.currentResolution)
			shutdown.ReleaseFromShutdown()
			return
		case job, more := <-cass.writeQueue:
			if !more {
				return
			}

			retryFunc := func() error {
				_, err := cass.InsertMulti(job.Name, job.Stats)
				return err
			}

			backoff.RetryNotify(retryFunc, backoff.NewExponentialBackOff(), retryNotify)
		}
	}
}

// listen to the overflow chan from the cache and attempt to write "now"
func (cass *CassandraFlatMapWriter) overFlowWrite() {
	shuts := cass.shutdown.Listen()
	for {
		select {
		case <-shuts.Ch:
			return
		case item, more := <-cass.cacheOverFlowListener.Ch:

			// bail
			if !more {
				return
			}
			statitem := item.(*smetrics.TotalTimeSeries)
			// need to make a list of points from the series
			iter, err := statitem.Series.Iter()
			if err != nil {
				cass.log.Error("error in overflow writer %v", err)
				continue
			}
			pts := make(repr.StatReprSlice, 0)
			for iter.Next() {
				pts = append(pts, iter.ReprValue())
			}
			if iter.Error() != nil {
				cass.log.Error("error in overflow iterator %v", iter.Error())
			}
			cass.log.Debug("Cache overflow force write for %s you may want to do something about that", statitem.Name.Key)
			cass.InsertMulti(statitem.Name, pts)
		}
	}
}

// slabName name based on the incoming time
func (cass *CassandraFlatMapWriter) slabName(inTime time.Time) string {
	switch cass.indexPerDate {
	case "daily":
		return inTime.Format("20060102")
	case "weekly":
		ynum, wnum := inTime.ISOWeek()
		return fmt.Sprintf("%04d%02d", ynum, wnum)
	case "monthly":
		return inTime.Format("200601")
	default:
		return inTime.Format("2006010215")
	}
}

// slabRange from the start times get the list of indexes we need to query
func (cass *CassandraFlatMapWriter) slabRange(sTime time.Time, eTime time.Time) []string {

	outStr := []string{}
	onT := sTime
	switch cass.indexPerDate {
	case "daily":
		eTime = eTime.AddDate(0, 0, 1) // need to include the end
		for onT.Before(eTime) {
			outStr = append(outStr, onT.Format("20060102"))
			onT = onT.AddDate(0, 0, 1)
		}
		return outStr
	case "weekly":
		eTime = eTime.AddDate(0, 0, 7) // need to include the end
		for onT.Before(eTime) {
			ynum, wnum := onT.ISOWeek()
			outStr = append(outStr, fmt.Sprintf("%04d%02d", ynum, wnum))
			onT = onT.AddDate(0, 0, 7)
		}
		return outStr
	case "monthly":
		eTime = eTime.AddDate(0, 1, 0) // need to include the end
		for onT.Before(eTime) {
			outStr = append(outStr, onT.Format("200601"))
			onT = onT.AddDate(0, 1, 0)
		}
		return outStr
	default:
		eTime = eTime.Add(time.Hour) // need to include the end
		for onT.Before(eTime) {
			outStr = append(outStr, onT.Format("2006010215"))
			onT = onT.Add(time.Hour)
		}
		return outStr
	}
}

// we can use the batcher effectively for single metric multi point writes as they share the
// the same token  note that due to the "map + stuff" CQL we DO NOT want this query prepared
// it will really quickly mush the prepared caches.
func (cass *CassandraFlatMapWriter) InsertMulti(name *repr.StatName, points repr.StatReprSlice) (int, error) {

	defer stats.StatsdNanoTimeFunc("writer.cassandraflatmap.batch.metric-time-ns", time.Now())

	l := len(points)
	if l == 0 {
		return 0, nil
	}

	tName := cass.db.MetricTable()
	if cass.tablePerResolution {
		tName = tName + "_" + strconv.Itoa(int(name.Resolution)) + "s"
	}

	var DOQ string

	// make a slab -> points list
	slabList := make(map[string][]*cqlSetPoint)
	defer func() {
		for _, pts := range slabList {
			putSetItem(pts)
		}
	}()

	for _, stat := range points {
		slab := cass.slabName(stat.ToTime())
		if _, ok := slabList[slab]; ok {
			slabList[slab] = append(slabList[slab], getSubSetItem(stat))
		} else {
			slabList[slab] = getSetItemStat(stat)
		}
	}

	if name.Ttl > 0 {
		DOQ = fmt.Sprintf(cass.insertQuery, tName, " USING TTL "+strconv.Itoa(int(name.Ttl)))
	} else {
		DOQ = fmt.Sprintf(cass.insertQuery, tName, "")
	}

	var did int
	var err error
	for slab, pts := range slabList {
		if cass.tablePerResolution {
			err = cass.conn.Query(DOQ,
				pts,
				name.UniqueIdString(),
				slab,
				slab,
			).Exec()

		} else {
			err = cass.conn.Query(DOQ,
				pts,
				name.UniqueIdString(),
				name.Resolution,
				slab,
				slab,
			).Exec()
		}

		// bail quick
		if err == gocql.ErrNoConnections {
			cass.log.Error("Cassandra Driver: Cassandra seems to be gone: Batch Metric insert failed, %v", err)
			return 0, err
		}
		if err != nil {
			cass.log.Error("Cassandra Driver:Batch Metric insert failed, %v", err)
			stats.StatsdClientSlow.Incr("writer.cassandraflatmap.batch.metric-failures", 1)
		} else {
			did += 1
		}
	}

	stats.StatsdClientSlow.Incr("writer.cassandraflatmap.batch.writes", 1)
	stats.StatsdClientSlow.GaugeAvg("writer.cassandraflatmap.batch.metrics-per-writes", int64(did))

	return did, err
}

func (cass *CassandraFlatMapWriter) InsertOne(name *repr.StatName, stat *repr.StatRepr) (int, error) {

	defer stats.StatsdNanoTimeFunc("writer.cassandraflatmap.write.metric-time-ns", time.Now())

	var err error

	tName := cass.db.MetricTable()
	if cass.tablePerResolution {
		tName = tName + "_" + strconv.Itoa(int(name.Resolution)) + "s"
	}

	var ttlStr string
	slab := cass.slabName(stat.ToTime())

	if stat.Name.Ttl > 0 {
		ttlStr = " USING TTL " + strconv.Itoa(int(stat.Name.Ttl))

	}
	DOQ := fmt.Sprintf(cass.insertQuery, tName, ttlStr)
	mper := getSetItemStat(stat)

	if cass.tablePerResolution {
		err = cass.conn.Query(DOQ,
			mper,
			name.UniqueIdString(),
			slab,
			slab,
		).Exec()
	} else {
		err = cass.conn.Query(DOQ,
			mper,
			name.UniqueIdString(),
			stat.Name.Resolution,
			slab,
			slab,
		).Exec()
	}
	putSetItem(mper)
	//cass.log.Critical("METRICS WRITE %d: %v", ttl, stat)
	if err != nil {
		cass.log.Error("Cassandra Driver: insert failed, %v", err)
		stats.StatsdClientSlow.Incr("writer.cassandraflatmap.metric-failures", 1)

		return 0, err
	}
	stats.StatsdClientSlow.Incr("writer.cassandraflatmap.metric-writes", 1)

	return 1, nil
}

// pop from the cache and send to actual writers
func (cass *CassandraFlatMapWriter) sendToWriters() {

	shuts := cass.shutdown.Listen()
	for {
		select {
		case <-shuts.Ch:
			return
		default:
			name, points := cass.cacher.Pop()
			switch points {
			case nil:
				time.Sleep(time.Second)
			default:
				stats.StatsdClient.Incr(fmt.Sprintf("writer.cassandraflatmap.write.send-to-writers"), 1)
				cass.writeQueue <- &cassandraFlatMetricJob{Cass: cass, Stats: points, Name: name}
				time.Sleep(time.Millisecond)
			}
		}
	}
}

func (cass *CassandraFlatMapWriter) WriteWithOffset(stat *repr.StatRepr, offset *smetrics.OffsetInSeries) error {

	if atomic.LoadInt32(&cass.shutitdown) == 0 {
		cass.cacher.AddWithOffset(stat.Name, stat, offset)
	}
	return nil
}

func (cass *CassandraFlatMapWriter) Write(stat *repr.StatRepr) error {
	return cass.WriteWithOffset(stat, nil)
}

// CassandraFlatMapMetric cassandra flat map main metric writer
type CassandraFlatMapMetric struct {
	WriterBase

	writer *CassandraFlatMapWriter

	renderTimeout time.Duration
}

func NewCassandraFlatMapMetrics() *CassandraFlatMapMetric {
	return new(CassandraFlatMapMetric)
}

func (cass *CassandraFlatMapMetric) Driver() string {
	return "cassandra-flat-map"
}

func (cass *CassandraFlatMapMetric) Start() {
	cass.writer.resolutions = cass.GetResolutions()
	cass.writer.currentResolution = cass.currentResolution
	cass.writer.Start()
}

func (cass *CassandraFlatMapMetric) Stop() {
	cass.writer.Stop()
}

func (cass *CassandraFlatMapMetric) Config(conf *options.Options) (err error) {
	cass.options = conf

	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("Metrics: `dsn` (server1,server2,server3) is needed for cassandra-flat config")
	}
	res, err := conf.Int64Required("resolution")
	if err != nil {
		return fmt.Errorf("Resolution needed for cassandra-flat writer")
	}

	cass.name = conf.String("name", "metrics:cassandraflatmap:"+dsn)

	// FLAT writers have different "writers" for each resolution,  so we need to set different names for them
	// the API readers use the "first one", so if things already exist w/ the name, slap on a resolution tag
	if GetMetrics(cass.Name()) != nil {
		cass.name = conf.String("name", "metrics:cassandraflatmap:"+dsn) + fmt.Sprintf(":%v", res)
	}
	err = RegisterMetrics(cass.Name(), cass)
	if err != nil {
		return err
	}

	gots, err := NewCassandraFlatMapWriter(conf, cass.GetResolutions())
	if err != nil {
		return err
	}
	gots.currentResolution = cass.currentResolution

	cass.writer = gots
	cass.options = conf

	cacheKey := fmt.Sprintf("cassandraflatmap:cache:%s:%v", conf.String("dsn", ""), res)
	_cache, err := GetCacherSingleton(cacheKey, "single")
	if _cache == nil {
		return errMetricsCacheRequired
	}
	scacher, ok := _cache.(*CacherSingle)
	if !ok {
		return ErrorMustBeSingleCacheType
	}
	gots.cacher = scacher

	if err != nil {
		return err
	}

	rdur, err := time.ParseDuration(CASSANDRA_FLAT_RENDER_TIMEOUT)
	if err != nil {
		return err
	}
	cass.renderTimeout = rdur

	g_tag := conf.String("tags", "")
	if len(g_tag) > 0 {
		cass.staticTags = repr.SortingTagsFromString(g_tag)
	}

	// prevent a reader from squshing this cacher
	if !gots.cacher.started && !gots.cacher.inited {
		gots.cacher.inited = true
		// set the cacher bits
		gots.cacher.maxKeys = int(conf.Int64("cache_metric_size", CACHER_METRICS_KEYS))
		gots.cacher.maxBytes = int(conf.Int64("cache_byte_size", CACHER_NUMBER_BYTES))
		gots.cacher.lowFruitRate = conf.Float64("cache_low_fruit_rate", 0.25)
		gots.cacher.seriesType = conf.String("cache_series_type", CACHER_SERIES_TYPE)
		gots.cacher.overFlowMethod = conf.String("cache_overflow_method", CACHER_DEFAULT_OVERFLOW)

		if gots.cacher.overFlowMethod == "chan" {
			gots.cacheOverFlowListener = gots.cacher.GetOverFlowChan()
			go gots.overFlowWrite()
		}
	}

	return nil
}

func (cass *CassandraFlatMapMetric) Write(stat *repr.StatRepr) error {
	return cass.WriteWithOffset(stat, nil)
}

// simple proxy to the cacher
func (cass *CassandraFlatMapMetric) WriteWithOffset(stat *repr.StatRepr, offset *smetrics.OffsetInSeries) error {
	// write the index from the cache as indexing can be slooowwww
	// keep note of this, when things are not yet "warm" (the indexer should
	// keep tabs on what it's already indexed for speed sake,
	// the push "push" of stats will cause things to get pretty slow for a while
	stat.Name.MergeMetric2Tags(cass.staticTags)

	// only the lowest res needs to write the index
	if cass.ShouldWriteIndex() {
		cass.indexer.Write(*stat.Name)
	}
	return cass.writer.Write(stat)
}

/************************ READERS ****************/

func (cass *CassandraFlatMapMetric) RawRenderOne(ctx context.Context, metric *sindexer.MetricFindItem, start int64, end int64, resample uint32) (*smetrics.RawRenderItem, error) {
	sp, closer := cass.GetSpan("RawRenderOne", ctx)
	sp.LogKV("driver", "CassandraFlatMapMetric", "metric", metric.StatName().Key, "uid", metric.UniqueId)
	defer closer()
	defer stats.StatsdSlowNanoTimeFunc("reader.cassandraflatmap.renderraw.get-time-ns", time.Now())

	preped := cass.newGetPrep(metric, start, end, resample)
	if metric.Leaf == 0 {
		//data only but return a "blank" data set otherwise graphite no likey
		return preped.rawd, ErrorNotADataNode
	}

	b_len := (preped.uEnd - preped.uStart) / preped.outResolution //just to be safe
	if b_len <= 0 {
		return preped.rawd, ErrorTimeTooSmall
	}

	// try the write inflight cache as nothing is written yet
	statName := metric.StatName()
	inflight, err := cass.writer.cacher.GetAsRawRenderItem(statName)

	// need at LEAST 2 points to get the proper step size
	if inflight != nil && err == nil {
		// move the times to the "requested" ones and quantize the list
		inflight.Metric = metric.Id
		inflight.Tags = metric.Tags
		inflight.MetaTags = metric.MetaTags
		inflight.Id = metric.UniqueId
		inflight.AggFunc = preped.rawd.AggFunc

		if inflight.Start < preped.uStart {
			inflight.RealEnd = preped.uStart
			inflight.RealStart = preped.uEnd
			inflight.Start = inflight.RealStart
			inflight.End = inflight.RealEnd
			inflight.Resample(preped.outResolution) // need resample to handle "time duplicates"
			return inflight, err
		}
		// The cache here can have more in RAM that is also on disk in cassandra,
		// so we need to alter the cassandra query to get things "not" already gotten from the cache
		// i.e. double counting on a merge operation
		if inflight.RealStart <= uint32(end) {
			end = TruncateTimeTo(int64(inflight.RealStart+preped.dbResolution), int(preped.dbResolution))
		}
	}

	slabs := cass.writer.slabRange(time.Unix(start, 0), time.Unix(end, 0))

	tName := cass.writer.db.MetricTable()
	args := []interface{}{metric.UniqueId, preped.dbResolution}
	if cass.writer.tablePerResolution {
		tName = fmt.Sprintf("%s_%ds", tName, preped.dbResolution)
		args = []interface{}{metric.UniqueId}
	}
	binders := make([]string, len(slabs))
	for i, s := range slabs {
		binders[i] = "?"
		args = append(args, s)
	}
	Q := fmt.Sprintf(cass.writer.selectTimeQuery, tName, strings.Join(binders, ","))

	// grab ze data. (note data is already sorted by time asc va the cassandra schema)
	iter := cass.writer.conn.Query(Q, args...).Iter()

	// sorting order for the table is time ASC (i.e. firstT == first entry)

	Tstart := uint32(start)
	curPt := smetrics.NullRawDataPoint(Tstart)

	// schema result sets are sorted by the slab already in ASC order
	// just need to make sure the map points are
	pts := make([]*cqlSetPoint, 0)
	var t uint32
	for iter.Scan(&pts) {

		sort.Sort(cqlSetPointTimePoint(pts))

		for _, pt := range pts {
			// cass driver stores "not so fully int64" things as truncated int32s
			// so we need to do this silly thing
			if pt.Time > 2147483647 {
				t = uint32(time.Unix(0, pt.Time).Unix())
			} else {
				t = uint32(time.Unix(pt.Time, 0).Unix())
			}

			if t < preped.uStart-preped.dbResolution {
				continue
			}
			if t > preped.uEnd+preped.dbResolution {
				continue
			}

			if preped.doResample {
				if t >= Tstart+resample {
					Tstart += resample
					preped.rawd.Data = append(preped.rawd.Data, curPt)
					curPt = &smetrics.RawDataPoint{
						Count: pt.Count,
						Sum:   pt.Sum,
						Max:   pt.Max,
						Min:   pt.Max,
						Last:  pt.Last,
						Time:  t,
					}
				} else {
					curPt.Merge(&smetrics.RawDataPoint{
						Count: pt.Count,
						Sum:   pt.Sum,
						Max:   pt.Max,
						Min:   pt.Max,
						Last:  pt.Last,
						Time:  t,
					})
				}
			} else {
				preped.rawd.Data = append(preped.rawd.Data, &smetrics.RawDataPoint{
					Count: pt.Count,
					Sum:   pt.Sum,
					Max:   pt.Max,
					Min:   pt.Max,
					Last:  pt.Last,
					Time:  t,
				})
			}
			preped.realEnd = t
		}
		pts = pts[:0]
	}

	if !curPt.IsNull() {
		preped.rawd.Data = append(preped.rawd.Data, curPt)
	}

	if err := iter.Close(); err != nil {
		cass.writer.log.Error("RawRender: Failure closing iterator: %v", err)
	}

	if len(preped.rawd.Data) > 0 && preped.rawd.Data[0].Time > 0 {
		preped.realStart = preped.rawd.Data[0].Time
	}

	preped.rawd.RealEnd = uint32(preped.realEnd)
	preped.rawd.RealStart = uint32(preped.realStart)
	preped.rawd.Start = uint32(start)
	preped.rawd.End = uint32(end)
	preped.rawd.Step = preped.outResolution

	if inflight == nil || len(inflight.Data) == 0 {
		return preped.rawd, nil
	}
	// fix to desired range
	inflight.RealEnd = preped.uStart
	inflight.RealStart = preped.uEnd

	if len(preped.rawd.Data) > 0 && len(inflight.Data) > 0 {
		inflight.MergeWithResample(preped.rawd, preped.outResolution)
		return inflight, nil
	}
	if inflight.Step != preped.outResolution {
		inflight.Resample(preped.outResolution)
	}

	return inflight, nil
}

func (cass *CassandraFlatMapMetric) RawRender(ctx context.Context, path string, from int64, to int64, tags repr.SortingTags, resample uint32) ([]*smetrics.RawRenderItem, error) {
	sp, closer := cass.GetSpan("RawRender", ctx)
	sp.LogKV("driver", "CassandraFlatMapMetric", "path", path, "from", from, "to", to)
	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("reader.cassandraflatmap.rawrender.get-time-ns", time.Now())

	paths := SplitNamesByComma(path)
	var metrics []*sindexer.MetricFindItem

	renderWg := utils.GetWaitGroup()
	defer utils.PutWaitGroup(renderWg)

	for _, pth := range paths {
		mets, err := cass.indexer.Find(ctx, pth, tags)
		if err != nil {
			continue
		}
		metrics = append(metrics, mets...)
	}

	rawd := make([]*smetrics.RawRenderItem, 0, len(metrics))

	procs := CASSANDRA_DEFAULT_METRIC_RENDER_WORKERS

	jobs := make(chan *sindexer.MetricFindItem, len(metrics))
	results := make(chan *smetrics.RawRenderItem, len(metrics))

	renderOne := func(met *sindexer.MetricFindItem) *smetrics.RawRenderItem {
		_ri, err := cass.RawRenderOne(ctx, met, from, to, resample)

		if err != nil {
			stats.StatsdClientSlow.Incr("reader.cassandraflatmap.rawrender.errors", 1)
			cass.writer.log.Error("Read Error for %s (%d->%d) : %v", path, from, to, err)
		}
		return _ri
	}

	// ye old fan out technique but not "too many" as to kill the server
	jobWorker := func(jober int, taskqueue <-chan *sindexer.MetricFindItem, resultqueue chan<- *smetrics.RawRenderItem) {
		rec_chan := make(chan *smetrics.RawRenderItem, 1)
		for met := range taskqueue {
			go func() { rec_chan <- renderOne(met) }()
			select {
			case <-time.After(cass.renderTimeout):
				stats.StatsdClientSlow.Incr("reader.cassandraflatmap.rawrender.timeouts", 1)
				cass.writer.log.Errorf("Render Timeout for %s (%d->%d)", path, from, to)
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

	stats.StatsdClientSlow.Incr("reader.cassandraflatmap.rawrender.metrics-per-request", int64(len(metrics)))

	return rawd, nil
}
