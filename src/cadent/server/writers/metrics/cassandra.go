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
	The Cassandra Metric Blob Reader/Writer

	This cassandra blob writer takes one of the "series" blobs

	These blobs are stored in ram until "max_blob_chunk_size" is reached (default of 16kb)

	8kb for the "gob" blob type is about 130 metrics (at a flush window of 10s, about .3 hours
	Blobs are flushed every "max_time_in_ram" (default 1 hour)

	json are much larger (2-3x), but are more generic to other system backends (i.e. if you need
	to read from the DB from things other then cadent)

	protobuf are about the same size as the Gob ones, but since it's not a go-specific item, has more portability

	gorilla is a highly compressed format, but only supports forward in time points which for writing is
	probably a good thing

	Since these things are stored in Ram for a while, the size of the blobs can become important

	Unlike the "flat" cassandra where there are zillion writes happening, this one trades off the writes
	for ram.  So we don't need to use the crazy worker queue mechanism as adding metrics
	to the ram pools is fast enough, a separate slow processor periodically marches through the
	cache pool and flushes those items that need flushing to cassandra in a more serial fashion


	[graphite-cassandra.accumulator.writer.metrics]
	driver="cassandra"
	dsn="my.cassandra.com"
	cache="my-series-cache"
	[graphite-cassandra.accumulator.writer.smetrics.options]
		keyspace="metric"
		metric_table="metric"
		path_table="path"
		segment_table="segment"
		write_consistency="one"
		read_consistency="one"
		port=9042
		numcons=5  # cassandra connection pool size
		timeout="30s" # query timeout
		user: ""
		pass: ""
		write_workers=32  # dispatch workers to write
		write_queueLength=102400  # buffered queue size before we start blocking


	Schema

	CREATE TYPE metric_id (
    		uid varchar,   # repr.StatName.UniqueIDString()
    		res int  # resolution
	);

	CREATE TABLE metric (
    		id frozen<metric_id>,
    		stime bigint,
    		etime bigint,
    		points blob,
    		PRIMARY KEY (id, stime, etime)
	) WITH CLUSTER ORDERING stime DESC



*/

package metrics

import (
	"cadent/server/stats"
	"cadent/server/writers/dbs"
	"fmt"
	"github.com/gocql/gocql"
	logging "gopkg.in/op/go-logging.v1"

	"cadent/server/broadcast"
	"cadent/server/dispatch"
	sindexer "cadent/server/schemas/indexer"
	smetrics "cadent/server/schemas/metrics"
	"cadent/server/schemas/repr"
	sseries "cadent/server/schemas/series"
	"cadent/server/series"
	"cadent/server/utils"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/indexer"
	"github.com/cenkalti/backoff"
	"golang.org/x/net/context"
	"runtime/debug"
	"sync"

	"time"
)

const (
	CASSANDRA_DEFAULT_RENDER_TIMEOUT        = "5s"
	CASSANDRA_DEFAULT_ROLLUP_TYPE           = "cached"
	CASSANDRA_DEFAULT_METRIC_WORKERS        = 16
	CASSANDRA_DEFAULT_METRIC_QUEUE_LEN      = 1024 * 100
	CASSANDRA_DEFAULT_METRIC_RETRIES        = 2
	CASSANDRA_DEFAULT_METRIC_RENDER_WORKERS = 4
	CASSANDRA_DEFAULT_TABLE_PER_RESOLUTION  = false
)

/*** set up "one" real writer (per dsn) .. need just a single cassandra DB connection for all the time resolutions */

type CassandraWriter struct {
	// the writer connections for this
	db         *dbs.CassandraDB
	conn       *gocql.Session
	insertConn *gocql.Session // different session for inserts/rollups as they can eat alot of our sockets

	// shutdowners
	shutitdown         bool
	shutdown           chan bool
	tablePerResolution bool

	insQ string //insert query
	log  *logging.Logger
}

func NewCassandraWriter(conf *options.Options) (*CassandraWriter, error) {
	cass := new(CassandraWriter)
	cass.log = logging.MustGetLogger("metrics.cassandra")
	cass.shutdown = make(chan bool)
	cass.shutitdown = false

	gots, err := conf.StringRequired("dsn")
	if err != nil {
		return nil, fmt.Errorf("Metrics: `dsn` (server1,server2,server3) is needed for cassandra config")
	}

	connKey := fmt.Sprintf("%v:%v/%v/%v", gots, conf.Int64("port", 9042), conf.String("keyspace", "metric"), conf.String("metrics_table", "metric"))
	nm := conf.String("name", connKey)

	cass.log.Notice("Connecting Metrics to Name: %s Cassandra: %s", nm, connKey)
	db, err := dbs.NewDB("cassandra", nm, conf)
	if err != nil {
		return nil, err
	}
	// need to cast for real usage
	cass.db = db.(*dbs.CassandraDB)
	cass.conn, err = cass.db.GetSession()
	if err != nil {
		return nil, err
	}
	cass.insertConn, err = cass.db.GetSession()
	if err != nil {
		return nil, err
	}

	cass.tablePerResolution = conf.Bool("table_per_resolution", CASSANDRA_DEFAULT_TABLE_PER_RESOLUTION)

	cass.insQ = "INSERT INTO %s (mid, etime, stime, ptype, points) VALUES  ({id: ?, res: ?}, ?, ?, ?, ?)"
	if cass.tablePerResolution {
		cass.insQ = "INSERT INTO %s (id, etime, stime, ptype, points) VALUES  (?, ?, ?, ?, ?)"
	}
	return cass, nil
}

func (cass *CassandraWriter) onePerTableInsert(name *repr.StatName, timeseries series.TimeSeries, resolution uint32) (n int, err error) {
	tName := cass.db.MetricTable() + fmt.Sprintf("_%d", resolution) + "s"
	DO_Q := fmt.Sprintf(cass.insQ, tName)
	if name.Ttl > 0 {
		DO_Q += fmt.Sprintf(" USING TTL %d", name.Ttl)
	}

	blob, err := timeseries.MarshalBinary()
	if err != nil {
		return 0, err
	}

	// if no bytes don't add
	if len(blob) == 0 {
		return 0, nil
	}
	err = cass.insertConn.Query(
		DO_Q,
		name.UniqueIdString(),
		timeseries.LastTime(),
		timeseries.StartTime(),
		series.IdFromName(timeseries.Name()),
		blob,
	).Exec()

	return 1, err
}

func (cass *CassandraWriter) oneTableInsert(name *repr.StatName, timeseries series.TimeSeries, resolution uint32) (n int, err error) {

	DO_Q := fmt.Sprintf(cass.insQ, cass.db.MetricTable())
	if name.Ttl > 0 {
		DO_Q += fmt.Sprintf(" USING TTL %d", name.Ttl)
	}

	blob, err := timeseries.MarshalBinary()
	if err != nil {
		return 0, err
	}

	// if no bytes don't add
	if len(blob) == 0 {
		return 0, nil
	}
	err = cass.insertConn.Query(
		DO_Q,
		name.UniqueIdString(),
		int64(name.Resolution),
		timeseries.LastTime(),
		timeseries.StartTime(),
		series.IdFromName(timeseries.Name()),
		blob,
	).Exec()

	return 1, err
}

func (cass *CassandraWriter) InsertSeries(
	ts *smetrics.TotalTimeSeries,
	resolution uint32,
	writeNotify chan *sseries.MetricWritten,
) (n int, err error) {

	defer func() {
		if r := recover(); r != nil {
			cass.log.Critical("Cassandra Failure (panic) %v ::", r)
			err = fmt.Errorf("The recover error: %v", r)
		}
	}()

	usingTs := ts
	//panic(fmt.Sprintf("InsertSeries: %v :: %v", cass.tablePerResolution, cass.insQ))
	if usingTs.Name == nil {
		return 0, errNameIsNil
	}
	if usingTs.Series == nil {
		return 0, errSeriesIsNil
	}
	defer stats.StatsdNanoTimeFunc(fmt.Sprintf("writer.cassandra.insert.metric-time-ns"), time.Now())

	l := usingTs.Series.Count()
	if l == 0 {
		return 0, nil
	}

	onOffset := usingTs.Offset
	name := usingTs.Name

	retryNotify := func(err error, after time.Duration) {
		stats.StatsdClientSlow.Incr("writer.cassandra.insert.metric-retry", 1)
		cass.log.Error("Update failed will retry :: %v : after %v", err, after)
	}
	retryFunc := func() error {
		switch cass.tablePerResolution {
		case true:
			n, err = cass.onePerTableInsert(usingTs.Name, usingTs.Series, resolution)
		default:
			n, err = cass.oneTableInsert(usingTs.Name, usingTs.Series, resolution)
		}
		if err != nil {
			cass.log.Error("Cassandra Driver: Metric insert failed, %v", err)
			stats.StatsdClientSlow.Incr("writer.cassandra.insert.metric-failures", 1)
			return err
		} else {
			stats.StatsdClientSlow.Incr("writer.cassandra.insert.writes", 1)
			stats.StatsdClientSlow.GaugeAvg("writer.cassandra.insert.metrics-per-writes", int64(l))

			if writeNotify != nil {
				item := new(sseries.MetricWritten)
				item.Id = uint64(name.UniqueId())
				item.Uid = name.UniqueIdString()
				item.WriteTime = time.Now().UTC().UnixNano()
				item.Tags = name.Tags
				item.MetaTags = name.MetaTags
				item.Metric = name.Key
				item.StartTime = usingTs.Series.StartTime()
				item.EndTime = usingTs.Series.LastTime()
				item.Resolution = resolution
				item.Ttl = name.Ttl

				if onOffset != nil {
					item.Offset = onOffset.Offset
					item.Partition = onOffset.Partition
					item.Topic = onOffset.Topic
				}
				writeNotify <- item
			}
		}
		return nil
	}

	err = backoff.RetryNotify(retryFunc, backoff.NewExponentialBackOff(), retryNotify)

	return l, err
}

func (cass *CassandraWriter) Stop() {
	if cass.shutitdown {
		return
	}
	cass.shutitdown = true
}

/************************************************************************/
/**********  Standard Worker Dispatcher JOB   ***************************/
/************************************************************************/
// insert job queue workers

type cassandraWriter interface {
	doInsert(*smetrics.TotalTimeSeries) error
}

type cassandraBlobMetricJob struct {
	Cass  cassandraWriter
	Ts    *smetrics.TotalTimeSeries // where the point list live
	retry int
}

func (j cassandraBlobMetricJob) IncRetry() int {
	j.retry++
	return j.retry
}
func (j cassandraBlobMetricJob) OnRetry() int {
	return j.retry
}

func (j cassandraBlobMetricJob) DoWork() error {
	err := j.Cass.doInsert(j.Ts)
	return err
}

/****************** Metrics Writer *********************/
type CassandraMetric struct {
	CacheBase

	driver string
	writer *CassandraWriter

	cacheOverFlow *broadcast.Listener // on byte overflow of cacher force a write

	// if the rolluptype == cached, then we this just uses the internal RAM caches
	// otherwise if "trigger" we only have the lowest res cache, and trigger rollups on write
	rollupType string
	rollup     *RollupMetric
	doRollup   bool

	renderWg      sync.WaitGroup
	renderMu      sync.Mutex
	renderTimeout time.Duration

	// dispatch writer worker queue
	numWorkers      int
	queueLen        int
	dispatchRetries int
	dispatcher      *dispatch.DispatchQueue

	tablePerResolution bool   // if this is "true" then rather then "one big table" we assume different tables for each rollup
	selQ               string //select query

	shutdown      chan bool
	shutDownPurge bool // if true, will attempt to blast write all the things.  if using a queue (kafka) this is not nessesary
}

func NewCassandraMetrics() *CassandraMetric {
	cass := new(CassandraMetric)
	cass.driver = "cassandra"
	cass.isPrimary = false
	return cass
}

func NewCassandraTriggerMetrics() *CassandraMetric {
	cass := new(CassandraMetric)
	cass.driver = "cassandra-triggered"
	cass.rollupType = "triggered"
	cass.isPrimary = false
	return cass
}

func (cass *CassandraMetric) Config(conf *options.Options) (err error) {

	cass.options = conf

	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("Metrics: `dsn` (server1,server2,server3) is needed for cassandra config")
	}

	// only need one real "writer DB" here as we are writing to the same metrics table
	gots, err := NewCassandraWriter(conf)
	if err != nil {
		return err
	}

	cass.writer = gots
	cass.name = conf.String("name", "metrics:cassandra:"+dsn)
	// reg ourselves before try to get conns
	cass.writer.log.Noticef("Registering metrics: %s", cass.Name())
	err = RegisterMetrics(cass.Name(), cass)
	if err != nil {
		return err
	}

	// even if not used, it best be here for future goodies
	_, err = conf.Int64Required("resolution")
	if err != nil {
		return fmt.Errorf("resolution needed for cassandra writer: %v", err)
	}

	_tgs := conf.String("tags", "")
	if len(_tgs) > 0 {
		cass.staticTags = repr.SortingTagsFromString(_tgs)
	}

	// tweak queus and worker sizes
	cass.numWorkers = int(conf.Int64("write_workers", CASSANDRA_DEFAULT_METRIC_WORKERS))
	cass.queueLen = int(conf.Int64("write_queue_length", CASSANDRA_DEFAULT_METRIC_QUEUE_LEN))
	cass.dispatchRetries = int(conf.Int64("write_queue_retries", CASSANDRA_DEFAULT_METRIC_RETRIES))
	cass.tablePerResolution = conf.Bool("table_per_resolution", false)
	cass.shutDownPurge = conf.Bool("shutdown_purge", true)

	rdur, err := time.ParseDuration(CASSANDRA_DEFAULT_RENDER_TIMEOUT)
	if err != nil {
		return err
	}
	cass.renderTimeout = rdur

	// rolluptype
	if cass.rollupType == "" {
		cass.rollupType = conf.String("rollup_type", CASSANDRA_DEFAULT_ROLLUP_TYPE)
	}

	_cache, _ := conf.ObjectRequired("cache")
	if _cache == nil {
		return errMetricsCacheRequired
	}

	// make sure it's of the correct type
	_, ok := _cache.(*CacherSingle)
	if !ok {
		return fmt.Errorf("cacher for cassandra needs to be of 'single' type")
	}

	cass.cacher = _cache.(Cacher)
	cass.cacherPrefix = cass.cacher.GetPrefix()

	// for the overflow cached items::
	// these caches can be shared for a given writer set, and the caches may provide the data for
	// multiple writers, we need to specify that ONE of the writers is the "main" one otherwise
	// the Metrics Write function will add the points over again, which is not a good thing
	// when the accumulator flushes things to the multi writers
	// The Writer needs to know it's "not" the primary writer and thus will not "add" points to the
	// cache .. so the cache basically gets "one" primary writer pointed (first come first serve)
	cass.isPrimary = cass.cacher.SetPrimaryWriter(cass)
	if cass.isPrimary {
		cass.writer.log.Notice("Cassandra series writer is the primary writer to write back cache %s", cass.cacher.GetName())
	}

	if cass.rollupType == "triggered" {
		cass.driver = "cassandra-triggered" // reset the name
		cass.rollup = NewRollupMetric(cass, cass.cacher.GetMaxBytesPerMetric())
	}

	// can be just a "hotstandby" cache only until we force it to start writing
	cass.shouldWrite = conf.Bool("cache_only", true)

	if cass.tablePerResolution {
		cass.selQ = "SELECT ptype, points FROM %s WHERE id=? AND etime >= ? AND etime <= ?"
	} else {
		cass.selQ = "SELECT ptype, points FROM %s WHERE mid={id: ?, res: ?} AND etime >= ? AND etime <= ?"
	}

	cass.writeNotifySubscribe = conf.Bool("return_write_notifications", false)
	if cass.writeNotifySubscribe {
		cass.writeNotify = make(chan *sseries.MetricWritten)
	}

	return nil
}

func (cass *CassandraMetric) Driver() string {
	return cass.driver
}

func (cass *CassandraMetric) Start() {
	cass.startstop.Start(func() {
		/**** dispatcher queue ***/
		cass.writer.log.Notice(
			"Starting cassandra series writer for %s at %d bytes per series Write Mode: %v",
			cass.writer.db.MetricTable(), cass.cacher.GetMaxBytesPerMetric(), cass.writer.db.Cluster().Consistency,
		)
		cass.writer.log.Notice("Adding metric tables ...")

		schems := NewCassandraMetricsSchema(
			cass.writer.conn,
			cass.writer.db.Keyspace(),
			cass.writer.db.MetricTable(),
			cass.resolutions,
			"blob",
			cass.tablePerResolution,
			cass.writer.db.GetCassandraVersion(),
		)
		err := schems.AddMetricsTable()
		if err != nil {
			panic(err)
		}
		cass.cacher.SetOverFlowMethod("chan") // need to force this issue
		cass.cacher.Start()

		// register the overflower
		cass.cacheOverFlow = cass.cacher.GetOverFlowChan()

		cass.shutitdown = false
		go cass.overFlowWrite()

		// if the resolutions list is just "one" there is no triggered rollups
		if len(cass.resolutions) == 1 {
			cass.rollupType = "cached"
		}
		cass.writer.log.Notice("Rollup Type: %s on resolution: %d (min resolution: %d)", cass.rollupType, cass.currentResolution, cass.resolutions[0][0])
		cass.doRollup = cass.rollupType == "triggered" && cass.currentResolution == cass.resolutions[0][0]
		// start the rollupper if needed
		if cass.doRollup {
			cass.writer.log.Notice("Starting rollup machine")
			// all but the lowest one
			cass.rollup.blobMaxBytes = int(cass.options.Int64("rollup_byte_size", int64(cass.cacher.GetMaxBytesPerMetric())))
			cass.rollup.numRetries = int(cass.options.Int64("rollup_retries", ROLLUP_NUM_RETRIES))
			cass.rollup.numWorkers = int(cass.options.Int64("rollup_workers", ROLLUP_NUM_WORKERS))
			cass.rollup.queueLength = int(cass.options.Int64("rollup_queue_length", ROLLUP_QUEUE_LENGTH))
			cass.rollup.consumeQueueWorkers = int(cass.options.Int64("rollup_queue_workers", ROLLUP_NUM_QUEUE_WORKERS))

			// queue or blast
			cass.rollup.SetRunMode(cass.options.String("rollup_method", CASSANDRA_LOG_ROLLLUP_MODE))
			cass.rollup.SetResolutions(cass.resolutions[1:])
			cass.rollup.SetMinResolution(cass.resolutions[0][0])
			go cass.rollup.Start()
		}

		//start up the dispatcher
		cass.dispatcher = dispatch.NewDispatchQueue(
			cass.numWorkers,
			cass.queueLen,
			cass.dispatchRetries,
		)
		cass.dispatcher.Start()
	})
}

func (cass *CassandraMetric) Stop() {
	cass.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()
		cass.writer.log.Warning("Starting Shutdown of cassandra series writer")

		if cass.shutitdown {
			return // already did
		}
		cass.shutitdown = true

		cass.cacher.Stop()

		// full tilt write out
		if cass.shutDownPurge {
			// need to cast this out
			scache := cass.cacher.(*CacherSingle)

			mets := scache.Cache
			mets_l := len(mets)
			cass.writer.log.Warning("Shutting down %s and exhausting the queue (%d items) and quiting", cass.cacher.GetName(), mets_l)
			procs := 16
			goDo := make(chan smetrics.TotalTimeSeries, procs)
			wg := sync.WaitGroup{}

			goInsert := func() {
				for {
					select {
					case s, more := <-goDo:
						if !more {
							return
						}

						// in ByPass mode there is no res so we "fake" the minimum
						if s.Name.Resolution <= 0 {
							s.Name.Resolution = uint32(cass.MinResolution())
						}

						stats.StatsdClient.Incr(fmt.Sprintf("writer.cache.shutdown.send-to-writers"), 1)
						cass.writer.InsertSeries(&s, s.Name.Resolution, cass.writeNotify)
						if cass.doRollup {
							cass.rollup.DoRollup(&s)
						}
						wg.Done()

					}
				}
			}
			for i := 0; i < procs; i++ {
				go goInsert()
			}
			did := 0
			for _, queueitem := range mets {
				wg.Add(1)
				if did%100 == 0 {
					cass.writer.log.Warning("shutdown purge: written %d/%d...", did, mets_l)
				}
				if queueitem.Series != nil {
					goDo <- smetrics.TotalTimeSeries{Name: queueitem.Name, Series: queueitem.Series.Copy()}
				}
				did++
			}
			wg.Wait()
			close(goDo)
			cass.writer.log.Warning("shutdown purge: written %d/%d...", did, mets_l)
		}

		// stop the notifier
		if cass.writeNotify != nil {
			close(cass.writeNotify)
		}

		if cass.doRollup {
			cass.rollup.Stop()
		}
		if cass.dispatcher != nil {
			cass.dispatcher.Stop()
		}

		cass.writer.conn.Close()
		cass.writer.insertConn.Close()

		cass.writer.log.Warning("Shutdown finished ... quiting cassandra series writer")
		return
	})
}

func (cass *CassandraMetric) SetIndexer(idx indexer.Indexer) error {
	cass.indexer = idx
	return nil
}

func (cass *CassandraMetric) TurnOnWriteNotify() <-chan *sseries.MetricWritten {
	if cass.writeNotify != nil {
		return cass.writeNotify
	}
	cass.writeNotify = make(chan *sseries.MetricWritten)
	return cass.writeNotify
}

func (cass *CassandraMetric) doInsert(ts *smetrics.TotalTimeSeries) error {
	// in ByPass mode there is no res so we "fake" the minimum
	if ts.Name.Resolution <= 0 {
		ts.Name.Resolution = uint32(cass.MinResolution())
	}
	_, err := cass.writer.InsertSeries(ts, ts.Name.Resolution, cass.writeNotify)
	if err == nil && cass.doRollup && ts.Name.Resolution == uint32(cass.MinResolution()) {
		cass.rollup.Add(ts)
	} else if err != nil {
		cass.writer.log.Errorf("Failed to add series to DB: %s (%s) %v", ts.Name.Key, ts.Name.UniqueIdString(), err)
	}
	return err
}

// listen to the overflow chan from the cache and attempt to write "now"
func (cass *CassandraMetric) overFlowWrite() {
	for {
		statitem, more := <-cass.cacheOverFlow.Ch
		if !more {
			return
		}

		// skip writes if not a writer
		if !cass.ShouldWrite() {
			continue
		}

		if statitem == nil {
			cass.log.Warning("Got a nil stat from the overflow channel ... ")
			debug.PrintStack()
			continue
		}

		ts, ok := statitem.(*smetrics.TotalTimeSeries)
		if !ok {
			cass.log.Error("Got a non TotalTimeSeries stat from the overflow channel ... ")
			debug.PrintStack()
			continue
		}

		cass.doInsert(ts) // doInsert does the rollups
	}
}

// Write simple proxy to the cacher
func (cass *CassandraMetric) Write(stat *repr.StatRepr) error {
	return cass.WriteWithOffset(stat, nil)
}

// WriteWithOffset simple proxy to the cacher
func (cass *CassandraMetric) WriteWithOffset(stat *repr.StatRepr, offset *smetrics.OffsetInSeries) error {
	if cass.shutitdown {
		return nil
	}
	stat.Name.MergeMetric2Tags(cass.staticTags)

	// only need to do this if the first resolution
	if cass.ShouldWriteIndex() {
		cass.indexer.Write(*stat.Name)
	}

	// not primary writer .. move along
	if !cass.isPrimary {
		return nil
	}

	if cass.rollupType == "triggered" {
		if cass.currentResolution == cass.resolutions[0][0] {
			return cass.cacher.AddWithOffset(stat.Name, stat, offset)
		}
	} else {
		return cass.cacher.AddWithOffset(stat.Name, stat, offset)
	}
	return nil
}

/************************ READERS ****************/
// GetFromReadCache (note not really in use)
func (cass *CassandraMetric) GetFromReadCache(metric string, start int64, end int64) (rawd *smetrics.RawRenderItem, got bool) {

	// check read cache
	rCache := GetReadCache()
	if rCache == nil {
		stats.StatsdClient.Incr("reader.cassandra.render.cache.miss", 1)
		return rawd, false
	}

	tStart := time.Unix(int64(start), 0)
	t_end := time.Unix(int64(end), 0)
	cachedStats, _, _ := rCache.Get(metric, tStart, t_end)
	var d_points []*smetrics.RawDataPoint
	step := uint32(0)

	// the ReadCache will only have the "sum" point in the mix as that's
	// the designated cached point
	if cachedStats != nil && len(cachedStats) > 0 {
		stats.StatsdClient.Incr("reader.cassandra.render.cache.hits", 1)

		tempT := uint32(0)
		for _, stat := range cachedStats {
			t := uint32(stat.ToTime().Unix())
			d_points = append(d_points, &smetrics.RawDataPoint{
				Count: 1,
				Sum:   stat.Sum,
				Time:  t,
			})
			if tempT == 0 {
				tempT = t
			}
			if step == 0 {
				step = t - tempT
			}
		}
		rawd.AggFunc = repr.GuessReprValueFromKey(metric)
		rawd.RealEnd = d_points[len(d_points)-1].Time
		rawd.RealStart = d_points[0].Time
		rawd.Start = rawd.RealStart
		rawd.End = rawd.RealEnd + step
		rawd.Metric = metric
		rawd.Step = step
		rawd.Data = d_points
		return rawd, len(d_points) > 0
	} else {
		stats.StatsdClient.Incr("reader.cassandra.render.cache.miss", 1)
	}

	return rawd, false
}

// grab the time series from the DBs
func (cass *CassandraMetric) GetFromDatabase(metric *sindexer.MetricFindItem, resolution uint32, start int64, end int64, resample uint32) (rawd *smetrics.RawRenderItem, err error) {
	defer stats.StatsdSlowNanoTimeFunc("reader.cassandra.database.get-time-ns", time.Now())
	rawd = new(smetrics.RawRenderItem)

	tName := cass.writer.db.MetricTable()
	if cass.tablePerResolution {
		tName = fmt.Sprintf("%s_%ds", tName, resolution)
	}

	Q := fmt.Sprintf(cass.selQ, tName)

	// times need to be in Nanos, but coming as a epoch
	// time in cassandra is in NanoSeconds so we need to pad the times from seconds -> nanos
	nano := int64(time.Second)
	nanoEnd := end * nano
	nanoStart := start * nano

	args := []interface{}{metric.UniqueId, resolution, nanoStart, nanoEnd}
	if cass.tablePerResolution {
		args = []interface{}{metric.UniqueId, nanoStart, nanoEnd}
	}

	iter := cass.writer.conn.Query(Q, args...).Iter()

	// cass.writer.log.Debug("Select Q for %s: %s (%v, %v, %v, %v)", metric.Id, Q, metric.UniqueId, resolution, nanoStart, nanoEnd)

	// for each "series" we get make a list of points
	statName := metric.StatName()

	uStart := uint32(start)
	uEnd := uint32(end)
	rawd.Start = uStart
	rawd.Step = resolution
	rawd.End = uEnd
	rawd.Id = metric.UniqueId
	rawd.Metric = metric.Path
	rawd.AggFunc = statName.AggType()

	var pType uint8
	var pBytes []byte

	tStart := uint32(start)

	doResample := resample > 0 && resample > resolution
	if doResample {
		tStart, uEnd = ResampleBounds(tStart, uEnd, resample)
		rawd.Start = tStart
		rawd.End = uEnd
		rawd.Step = resample
	}

	for iter.Scan(&pType, &pBytes) {
		sName := series.NameFromId(pType)
		sIter, err := series.NewIter(sName, pBytes)
		if err != nil {
			return rawd, err
		}
		curPt := smetrics.NullRawDataPoint(tStart)

		for sIter.Next() {
			to, mi, mx, ls, su, ct := sIter.Values()

			t := uint32(time.Unix(0, to).Unix())

			// skip if not in range
			if t > uEnd {
				break
			}
			if t < uStart {
				continue
			}

			if doResample {
				if t >= tStart+resample {

					rawd.Data = append(rawd.Data, curPt)

					tStart += resample
					curPt = new(smetrics.RawDataPoint)
					curPt.Count = ct
					curPt.Sum = su
					curPt.Min = mi
					curPt.Max = mx
					curPt.Last = ls
					curPt.Time = t
				} else {
					rd := smetrics.GetRawDataPoint() // sync.Pool these tmp items
					rd.Count = ct
					rd.Max = mx
					rd.Min = mi
					rd.Last = ls
					rd.Sum = su
					rd.Time = t
					curPt.Merge(rd)
					smetrics.PutRawDataPoint(rd)
				}
			} else {
				rawd.Data = append(rawd.Data, &smetrics.RawDataPoint{
					Count: ct,
					Sum:   su,
					Max:   mx,
					Min:   mi,
					Last:  ls,
					Time:  t,
				})
			}
		}

		if !curPt.IsNull() {
			rawd.Data = append(rawd.Data, curPt)
		}
		if sIter.Error() != nil {
			return rawd, sIter.Error()
		}
	}
	if len(rawd.Data) > 0 {
		rawd.RealEnd = rawd.Data[len(rawd.Data)-1].Time
		rawd.RealStart = rawd.Data[0].Time
	}
	if err := iter.Close(); err != nil {
		cass.writer.log.Error("Database: Failure closing iterator: %s: %v", Q, err)
	}

	return rawd, nil

}

// after the "raw" render we need to yank just the "point" we need from the data which
// will make the read-cache much smaller (will compress just the Mean value as the count is 1)
func (cass *CassandraMetric) RawDataRenderOne(ctx context.Context, metric *sindexer.MetricFindItem, start int64, end int64, resample uint32) (*smetrics.RawRenderItem, error) {
	sp, closer := cass.GetSpan("RawDataRenderOne", ctx)
	sp.LogKV("driver", "CassandraMetric", "metric", metric.StatName().Key, "uid", metric.UniqueId)

	defer closer()
	defer stats.StatsdSlowNanoTimeFunc("reader.cassandra.rawrenderone.get-time-ns", time.Now())

	preped := cass.newGetPrep(metric, start, end, resample)
	if metric.Leaf == 0 {
		//data only but return a "blank" data set otherwise graphite no likey
		return preped.rawd, ErrorNotADataNode
	}

	bLen := (preped.uEnd - preped.uStart) / preped.outResolution //just to be safe
	if bLen <= 0 {
		return preped.rawd, ErrorTimeTooSmall
	}

	// the "step" is unknown so we set it to the min
	inflight, err := cass.GetFromWriteCache(metric, preped.uStart, preped.uEnd, 1)

	// need at LEAST 2 points to get the proper step size
	if inflight != nil && err == nil && len(inflight.Data) > 1 {
		// all the data we need is in the inflight
		// if all the data is in this list we don't need to go any further
		// move the times to the "requested" ones and quantize the list
		if inflight.RealStart <= preped.uStart {
			inflight.RealEnd = preped.uEnd
			inflight.RealStart = preped.uStart
			inflight.Resample(preped.outResolution) // need resample to handle "time duplicates"
			stats.StatsdClientSlow.Incr("reader.cassandra.rawrenderone.cache-only.hit", 1)
			return inflight, err
		}

		// The cache here can have more in RAM that is also on disk in cassandra,
		// so we need to alter the cassandra query to get things "not" already gotten from the cache
		// i.e. double counting on a merge operation
		if inflight.RealStart <= uint32(end) {
			end = TruncateTimeTo(int64(inflight.RealStart+preped.dbResolution), int(preped.dbResolution))
		}
	}

	if err != nil {
		cass.writer.log.Error("Cassandra: Error in getting inflight data: %v", err)
	}

	// and now for the Query otherwise
	cassData, err := cass.GetFromDatabase(metric, preped.dbResolution, start, end, preped.outResolution)
	if err != nil {
		cass.writer.log.Error("Cassandra: Error getting from DB: %v", err)
		return preped.rawd, err
	}

	if inflight == nil || len(inflight.Data) == 0 {
		cassData.TruncateTo(preped.uStart-preped.outResolution, preped.uEnd+preped.outResolution)
		return cassData, nil
	}

	// fix to desired range
	inflight.RealEnd = preped.uEnd + preped.outResolution
	inflight.RealStart = preped.uStart - preped.outResolution
	cassData.RealEnd = preped.uEnd + preped.outResolution
	cassData.RealStart = preped.uStart - preped.outResolution
	if len(cassData.Data) > 0 && len(inflight.Data) > 0 {
		inflight.MergeWithResample(cassData, preped.outResolution)
		return inflight, nil
	}
	if inflight.Step != preped.outResolution {
		inflight.Resample(preped.outResolution)
	} else {
		inflight.TruncateTo(preped.uStart-preped.outResolution, preped.uEnd+preped.outResolution)
	}
	return inflight, nil
}

func (cass *CassandraMetric) RawRender(ctx context.Context, path string, start int64, end int64, tags repr.SortingTags, resample uint32) ([]*smetrics.RawRenderItem, error) {
	sp, closer := cass.GetSpan("RawRender", ctx)
	sp.LogKV("driver", "CassandraMetric", "path", path, "from", start, "to", end)
	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("reader.cassandra.rawrender.get-time-ns", time.Now())

	paths := SplitNamesByComma(path)

	var metrics sindexer.MetricFindItems

	renderWg := utils.GetWaitGroup()
	defer utils.PutWaitGroup(renderWg)

	for _, pth := range paths {
		mets, err := cass.indexer.Find(ctx, pth, tags)
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
		_ri, err := cass.RawDataRenderOne(ctx, met, start, end, resample)
		if err != nil {
			stats.StatsdClientSlow.Incr("reader.cassandra.rawrender.errors", 1)
			cass.writer.log.Errorf("Read Error for %s (%d->%d) : %v", path, start, end, err)
		}
		return _ri
	}

	// ye old fan out technique but not "too many" as to kill the server
	jobWorker := func(jober int, taskqueue <-chan *sindexer.MetricFindItem, resultqueue chan<- *smetrics.RawRenderItem) {
		resultsChan := make(chan *smetrics.RawRenderItem, 1)
		for met := range taskqueue {
			met := met // https://golang.org/doc/faq#closures_and_goroutines
			go func() { resultsChan <- renderOne(met) }()
			select {
			case <-time.After(cass.renderTimeout):
				stats.StatsdClientSlow.Incr("reader.cassandra.rawrender.timeouts", 1)
				cass.writer.log.Errorf("Render Timeout for %s (%d->%d)", path, start, end)
				resultqueue <- nil
			case res := <-resultsChan:
				resultqueue <- res
			}
		}
	}

	for i := 0; i < procs; i++ {
		i := i // https://golang.org/doc/faq#closures_and_goroutines
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
	stats.StatsdClientSlow.Incr("reader.cassandra.rawrender.metrics-per-request", int64(len(metrics)))

	return rawd, nil
}

/*************** Match the DBMetrics Interface ***********************/

// given a name get the latest metric series
func (cass *CassandraMetric) GetLatestFromDB(name *repr.StatName, resolution uint32) (smetrics.DBSeriesList, error) {

	var Q string
	var selArgs []interface{}

	if cass.tablePerResolution {
		Q = fmt.Sprintf(
			"SELECT id, stime, etime, ptype, points FROM %s_%ds WHERE id=? ORDER BY etime DESC LIMIT 1",
			cass.writer.db.MetricTable(), resolution,
		)
		selArgs = []interface{}{name.UniqueIdString()}
	} else {
		Q = fmt.Sprintf(
			"SELECT mid.id, stime, etime, ptype, points FROM %s WHERE mid={id: ?, res: ?} ORDER BY etime DESC LIMIT 1",
			cass.writer.db.MetricTable(),
		)
		selArgs = []interface{}{name.UniqueIdString(), resolution}
	}

	iter := cass.writer.conn.Query(Q, selArgs...).Iter()

	defer iter.Close()

	rawd := make(smetrics.DBSeriesList, 0)
	var pType uint8
	var pBytes []byte
	var uid string
	var start, end int64

	for iter.Scan(&uid, &start, &end, &pType, &pBytes) {
		dataums := &smetrics.DBSeries{
			Uid:        uid,
			Start:      start,
			End:        end,
			Ptype:      pType,
			Pbytes:     pBytes,
			Resolution: resolution,
		}
		rawd = append(rawd, dataums)

	}
	if err := iter.Close(); err != nil {
		cass.writer.log.Error("Database: Failure closing iterator: %s: %v", Q, err)
	}

	return rawd, nil
}

// given a name get the latest metric series
func (cass *CassandraMetric) GetRangeFromDB(name *repr.StatName, start uint32, end uint32, resolution uint32) (smetrics.DBSeriesList, error) {

	tName := cass.writer.db.MetricTable()
	var Q string
	var selArgs []interface{}
	// need to convert second time to nan time
	nano := int64(time.Second)
	nanoEnd := int64(end) * nano
	nanoStart := int64(start) * nano

	if cass.tablePerResolution {
		Q = fmt.Sprintf(
			"SELECT id, stime, etime, ptype, points FROM %s_%ds WHERE id= ? AND etime >= ? AND etime <= ?",
			tName, resolution,
		)
		selArgs = []interface{}{name.UniqueIdString(), nanoStart, nanoEnd}
	} else {
		Q = fmt.Sprintf(
			"SELECT mid.id, stime, etime, ptype, points FROM %s WHERE mid={id: ?, res: ?} AND etime >= ? AND etime <= ?",
			tName,
		)
		selArgs = []interface{}{name.UniqueIdString(), resolution, nanoStart, nanoEnd}
	}

	iter := cass.writer.conn.Query(Q, selArgs...).Iter()

	rawd := make(smetrics.DBSeriesList, 0)
	var pType uint8
	var pBytes []byte
	var uid string
	var tstart, tend int64

	for iter.Scan(&uid, &tstart, &tend, &pType, &pBytes) {
		dataums := &smetrics.DBSeries{
			Uid:        uid,
			Start:      tstart,
			End:        tend,
			Ptype:      pType,
			Pbytes:     pBytes,
			Resolution: resolution,
		}
		rawd = append(rawd, dataums)

	}
	if err := iter.Close(); err != nil {
		cass.writer.log.Error("Database: Failure closing iterator: %s: %v", Q, err)
	}

	return rawd, nil
}

// update the row defined in dbs w/ the new bytes from the new Timeseries
// for cassandara we need to use the "old" start ansd end times
// as the "Uniqueid" uid, res, etime, stime
func (cass *CassandraMetric) UpdateDBSeries(dbs *smetrics.DBSeries, ts series.TimeSeries) error {

	// sadly cassandra does not allow one to update the Primary Key bits
	// so we need to "delete" then "insert" a new row
	// this may not be the "best" way to deal w/ rollups as there will be many-a-tombstone

	points, err := ts.MarshalBinary()
	if err != nil {
		return err
	}

	ptype := series.IdFromName(ts.Name())

	tName := cass.writer.db.MetricTable()
	delQ := fmt.Sprintf(
		"DELETE FROM %s WHERE mid={id: ?, res: ?} AND etime=? AND stime=? AND ptype=?",
		tName,
	)
	delArgs := []interface{}{dbs.Uid, dbs.Resolution, dbs.End, dbs.Start, dbs.Ptype}

	InsQ := fmt.Sprintf(
		"INSERT INTO %s (mid, etime, stime, ptype, points) VALUES ({id: ?, res:?}, ?, ?, ?, ?)",
		tName,
	)
	insArgs := []interface{}{dbs.Uid, dbs.Resolution, ts.LastTime(), ts.StartTime(), ptype, points}

	if cass.tablePerResolution {
		tName = fmt.Sprintf("%s_%ds", tName, dbs.Resolution)
		delQ = fmt.Sprintf(
			"DELETE FROM %s WHERE id = ? AND etime=? AND stime=? AND ptype=?",
			tName,
		)
		delArgs = []interface{}{dbs.Uid, dbs.End, dbs.Start, dbs.Ptype}

		InsQ = fmt.Sprintf(
			"INSERT INTO %s (id, etime, stime, ptype, points) VALUES (?, ?, ?, ?, ?)",
			tName,
		)

		insArgs = []interface{}{dbs.Uid, ts.LastTime(), ts.StartTime(), ptype, points}

	}

	if dbs.TTL > 0 {
		InsQ += fmt.Sprintf(" USING TTL %d", dbs.TTL)
	}

	retryFunc := func() error {
		err := cass.writer.insertConn.Query(delQ, delArgs...).Exec()
		if err != nil {
			stats.StatsdClientSlow.Incr("writer.cassandra.update.metric-fail", 1)
			return err
		}
		err = cass.writer.insertConn.Query(InsQ, insArgs...).Exec()
		if err != nil {
			stats.StatsdClientSlow.Incr("writer.cassandra.update.metric-fail", 1)
		} else {
			stats.StatsdClientSlow.Incr("writer.cassandra.update.ok", 1)
		}
		return err
	}

	retryNotify := func(err error, after time.Duration) {
		stats.StatsdClientSlow.Incr("writer.cassandra.update.metric-retry", 1)
		cass.writer.log.Error("Update failed will retry :: %v : after %v", err, after)
	}

	return backoff.RetryNotify(retryFunc, backoff.NewExponentialBackOff(), retryNotify)
}

// InsertDBSeries update the row defined in dbs w/ the new bytes from the new Timeseries
func (cass *CassandraMetric) InsertDBSeries(name *repr.StatName, timeseries series.TimeSeries, resolution uint32) (added int, err error) {
	// in ByPass mode there is no res so we "fake" the minimum
	if resolution <= 0 {
		resolution = uint32(cass.MinResolution())
	}

	// if we are "triggered" rollups and this is not the min-res, DO NOT send a notify it's already done
	// via the initial write
	wnotif := cass.writeNotify
	if cass.rollupType == "triggered" && resolution != uint32(cass.MinResolution()) {
		wnotif = nil
	}
	return cass.writer.InsertSeries(&smetrics.TotalTimeSeries{Name: name, Series: timeseries}, resolution, wnotif)
}
