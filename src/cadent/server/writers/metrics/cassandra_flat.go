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
	The Cassandra - Flat Metric Reader/Writer

	The table should have this schema to match the repr item
	The same as the writer items

	By "flat" we mean we don't store "chunks" of data in a blob, but a row, by row per time

	this one is much nicer for other system that need to access the data, but is more "space heavy"
	and not nearly as efficient as the data blob representation

	this one ALSO LETS YOU UPDATE/MERGE data points in case times come in the past for older data points (which can
	be a performance penalty)

	the blob version does NOT allow upserts

	A Schema for one to use

	CREATE TYPE metric_point (
            max double,
            min double,
            sum double,
            last double,
            count int
        );

        CREATE TYPE metric_id_res (
            id ascii,
            res int
        );

        CREATE TABLE metric.metric (
            mid frozen<metric_id_res>,
            time bigint,
            point frozen<metric_point>,
            PRIMARY KEY (id, time)
        ) WITH COMPACT STORAGE
            AND CLUSTERING ORDER BY (mpath ASC, time ASC)
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
	driver="cassandra-flat"
	dsn="my.cassandra.com"
	[graphite-cassandra-flat.accumulator.writer.smetrics.options]
		keyspace="metric"
		metric_table="metric"
		path_table="path"
		segment_table="segment"
		write_consistency="one"
		read_consistency="one"
		port=9042
		cache_metric_size=102400  # the "internal carbon-like-cache" size (ram is your friend)
		cache_byte_size=1024 # number of bytes
		cache_low_fruit_rate=0.25 # every 1/4 of the time write "low count" metrics to at least persist them
		writesPerSecond=5000 # allowed insert queries per second

		numcons=5  # cassandra connection pool size
		timeout="30s" # query timeout
		user: ""
		pass: ""
		write_workers=32  # dispatch workers to write
		write_queueLength=102400  # buffered queue size before we start blocking

		# NOPE: batch_count: batch this many inserts for much faster insert performance (default 1000)
		# NOPE: periodic_flush: regardless of if batch_count met always flush things at this interval (default 1s)

*/

package metrics

import (
	"cadent/server/broadcast"
	"cadent/server/dispatch"
	sindexer "cadent/server/schemas/indexer"
	smetrics "cadent/server/schemas/metrics"
	"cadent/server/schemas/repr"
	"cadent/server/stats"
	"cadent/server/utils"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/dbs"
	"errors"
	"fmt"
	"github.com/gocql/gocql"
	"golang.org/x/net/context"
	logging "gopkg.in/op/go-logging.v1"
	"strconv"
	"sync"
	"time"
)

const (
	CASSANDRA_FLAT_METRIC_WORKERS               = 32
	CASSANDRA_FLAT_METRIC_QUEUE_LEN             = 1024 * 100
	CASSANDRA_FLAT_WRITES_PER_SECOND            = 5000
	CASSANDRA_FLAT_WRITE_UPSERT                 = true
	CASSANDRA_FLAT_RENDER_TIMEOUT               = "5s" // 5 second time out on any render
	CASSANDRA_FLAT_DEFAULT_TABLE_PER_RESOLUTION = false
)

var errNotimplemented = errors.New("Method not implemented")

/** Being Cassandra we need some mappings to match the schemas **/

/**
	CREATE TYPE metric_point (
        max double,
        min double,
        sum double,
        last double,
        count int
    );
*/

type CassMetricPoint struct {
	Max   float64
	Min   float64
	Sum   float64
	Last  float64
	Count int
}

/*
	CREATE TYPE metric_id_res (
        id ascii,
        resolution int
    );
*/

type CassMetricID struct {
	Id         string
	Resolution int
}

/*
 CREATE TABLE metric (
        id frozen<metric_id>,
        time bigint,
        point frozen<metric_point>
 )
*/
type CassMetric struct {
	Id         CassMetricID
	Time       int64
	Resolution CassMetricPoint
}

/*** set up "one" real writer (per dsn) .. and writer queue .. no
  no need to get multiqueues/channel/etc of these per resolution
  as we are all sharing the same DB pool and should write things in as they come in
  multiple writer pools tend to lead to bad lock contention behavior on input channels
  and output channels as well as the cassandra writer (gocql) itself.
  Having a "single" real writer for all resolutions saves all of that.

  this, and the "non-channel" Queue in "writer.go", comes from a lot of performance testing and tuning
  while not the "go'est" way of doing things.  It works with many 100s of thousands of metrics being flushed
  one a single machine.

  We don't need to do this for the "indexer" portion of the cassandra writer, as there is only "one" instance
  of that per DSN and it also maintains it's own "hot" cache check, which after 1-3 flushes will fill up and
  basically never write anymore

*/

/************************************************************************/
/**********  Standard Worker Dispatcher JOB   ***************************/
/************************************************************************/

type cassJobIface interface {
	DoWork() error
}

type cassWriter interface {
	InsertMulti(*repr.StatName, repr.StatReprSlice) (int, error)
}

// insert job queue workers
type cassandraFlatMetricJob struct {
	Cass  cassWriter
	Name  *repr.StatName
	Stats repr.StatReprSlice // where the point list live
	Retry int
}

func (j *cassandraFlatMetricJob) DoWork() error {
	_, err := j.Cass.InsertMulti(j.Name, j.Stats)
	return err
}

// CassandraFlatWriter Writes a "row" for each point metric point to a cassandra backend
type CassandraFlatWriter struct {
	// just the writer connections for this
	db   *dbs.CassandraDB
	conn *gocql.Session

	dispatcher *dispatch.DispatchQueue

	cacher                *CacherSingle
	cacheOverFlowListener *broadcast.Listener // on byte overflow of cacher force a write

	shutitdown bool // just a flag
	startstop  utils.StartStop

	writesPerSecond int // allowed writes per second
	numWorkers      int
	queueLen        int
	maxWriteSize    int           // size of that buffer before a flush
	maxIdle         time.Duration // either maxWriteSize will trigger a write or this time passing will
	writeLock       sync.Mutex
	log             *logging.Logger

	// upsert (true) or select -> merge -> update (false)
	// either squish metrics that have the same time windowe as a previious insert
	// or try to "update" the data point if exists
	// note upsert is WAY faster and should handle most of the cases
	insertMode         bool
	tablePerResolution bool

	resolutions       [][]int
	currentResolution int

	insertQuery     string //render once
	selectTimeQuery string //render once
	getQuery        string //render once
}

func NewCassandraFlatWriter(conf *options.Options, resolutions [][]int) (*CassandraFlatWriter, error) {

	cass := new(CassandraFlatWriter)
	cass.log = logging.MustGetLogger("metrics.cassandraflat")
	cass.shutitdown = false

	gots, err := conf.StringRequired("dsn")
	if err != nil {
		return nil, fmt.Errorf("Metrics: `dsn` (server1,server2,server3) is needed for cassandra config")
	}

	_, err = conf.Int64Required("resolution")
	if err != nil {
		return nil, fmt.Errorf("Metrics: `resolutions` required is needed for cassandra config")
	}

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

	cass.maxWriteSize = int(conf.Int64("batch_count", 50))

	if gocql.BatchSizeMaximum < cass.maxWriteSize {
		cass.log.Warning("Cassandra Driver: Setting batch size to %d, as it's the largest allowed", gocql.BatchSizeMaximum)
		cass.maxWriteSize = gocql.BatchSizeMaximum
	}

	cass.maxIdle = conf.Duration("periodic_flush", time.Duration(time.Second))

	// tweak queus and worker sizes
	cass.numWorkers = int(conf.Int64("write_workers", CASSANDRA_FLAT_METRIC_WORKERS))
	cass.queueLen = int(conf.Int64("write_queue_length", CASSANDRA_FLAT_METRIC_QUEUE_LEN))
	cass.writesPerSecond = int(conf.Int64("writes_per_second", CASSANDRA_FLAT_WRITES_PER_SECOND))
	cass.insertMode = conf.Bool("write_upsert", CASSANDRA_FLAT_WRITE_UPSERT)
	cass.tablePerResolution = conf.Bool("table_per_resolution", CASSANDRA_FLAT_DEFAULT_TABLE_PER_RESOLUTION)

	cass.insertQuery = "INSERT INTO %s (mid, t, pt) VALUES  ({id: ?, res: ?}, ?, {s: ?, mi: ?, mx: ?, l: ?, c: ?})"

	cass.selectTimeQuery = "SELECT pt.mx, pt.mi, pt.s, pt.l, pt.c, t FROM %s WHERE mid={id: ?, res: ?} AND t <= ? and t >= ?"

	cass.getQuery = "SELECT pt.mx, pt.mi, pt.s, pt.l, pt.c, t FROM %s WHERE mid={id: ?, res: ?} and t = ?"

	if cass.tablePerResolution {
		cass.insertQuery = "INSERT INTO %s (id, t, pt) VALUES  (?, ?, {s: ?, mi: ?, mx: ?, l: ?, c: ?})"

		cass.selectTimeQuery = "SELECT pt.mx, pt.mi, pt.s, pt.l, pt.c, time FROM %s WHERE id=? AND t <= ? and t >= ?"

		cass.getQuery = "SELECT pt.mx, pt.mi, pt.s, pt.l, pt.c, t FROM %s WHERE id=? and t = ?"

	}
	return cass, nil
}

func (cass *CassandraFlatWriter) Stop() {
	cass.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()

		cass.log.Warning("Starting Shutdown of writer")
		if cass.shutitdown {
			return // already did
		}
		cass.shutitdown = true

		cass.cacher.Stop()

		if cass.dispatcher != nil {
			cass.dispatcher.Stop()
		}

		mets := cass.cacher.Queue
		metsL := len(mets)
		cass.log.Warning("Shutting down, exhausting the queue (%d items) and quiting", metsL)
		// full tilt write out
		did := 0
		for _, queueitem := range mets {
			if did%100 == 0 {
				cass.log.Warning("shutdown purge: written %d/%d...", did, metsL)
			}
			name, points, _ := cass.cacher.GetById(queueitem.metric)
			if points != nil {
				stats.StatsdClient.Incr(fmt.Sprintf("writer.cassandraflat.write.send-to-writers"), 1)
				cass.InsertMulti(name, points)
			}
			did++
		}
		cass.conn.Close()

		cass.log.Warning("shutdown purge: written %d/%d...", did, metsL)
		cass.log.Warning("Shutdown finished ... quiting cassandra writer")
	})
}

func (cass *CassandraFlatWriter) Start() {
	/**** dispatcher queue ***/
	cass.startstop.Start(func() {

		// schemas
		cass.log.Notice(
			"Starting cassandra-flat writer for %s: Write mode: %v",
			cass.db.MetricTable(), cass.db.Cluster().Consistency,
		)

		cass.log.Notice("Adding metric tables resolutions: %v...", cass.resolutions)

		schems := NewCassandraMetricsSchema(
			cass.conn,
			cass.db.Keyspace(),
			cass.db.MetricTable(),
			cass.resolutions,
			"flat",
			cass.tablePerResolution,
			cass.db.GetCassandraVersion(),
		)

		err := schems.AddMetricsTable()

		if err != nil {
			panic(err)
		}

		workers := cass.numWorkers
		retries := 2
		cass.dispatcher = dispatch.NewDispatchQueue(workers, cass.queueLen, retries)
		cass.dispatcher.Start()

		cass.cacher.Start()
		go cass.sendToWriters() // the dispatcher
	})
}

// is not doing a straight upsert, we need to select then update
func (cass *CassandraFlatWriter) mergeWrite(stat *repr.StatRepr) *repr.StatRepr {
	if cass.insertMode { // true means upsert
		return stat
	}

	onTime := stat.ToTime().UnixNano()

	tName := cass.db.MetricTable()
	args := []interface{}{stat.Name.UniqueIdString(), stat.Name.Resolution, onTime}

	if cass.tablePerResolution {
		tName = fmt.Sprintf("%s_%ds", tName, stat.Name.Resolution)
		args = []interface{}{stat.Name.UniqueIdString(), onTime}
	}
	Q := fmt.Sprintf(cass.selectTimeQuery, tName)

	// grab ze data. (note data is already sorted by time asc va the cassandra schema)
	iter := cass.conn.Query(Q, args...).Iter()

	var t, count int64
	var min, max, sum, last float64

	for iter.Scan(&max, &min, &sum, &last, &count, &t) {
		// only one here
		n_stat := &repr.StatRepr{
			Time:  stat.Time,
			Last:  repr.CheckFloat(last),
			Count: count,
			Sum:   repr.CheckFloat(sum),
			Min:   repr.CheckFloat(min),
			Max:   repr.CheckFloat(max),
		}
		return stat.Merge(n_stat)
	}
	return stat
}

// listen to the overflow chan from the cache and attempt to write "now"
func (cass *CassandraFlatWriter) overFlowWrite() {
	for {
		select {
		case item, more := <-cass.cacheOverFlowListener.Ch:

			// bail
			if cass.shutitdown || !more {
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

// we can use the batcher effectively for single metric multi point writes as they share the
// the same token
func (cass *CassandraFlatWriter) InsertMulti(name *repr.StatName, points repr.StatReprSlice) (int, error) {

	defer stats.StatsdNanoTimeFunc("writer.cassandraflat.batch.metric-time-ns", time.Now())

	l := len(points)
	if l == 0 {
		return 0, nil
	}
	/*if l == 1 {
		return cass.InsertOne(points[0]) // should be faster then the batcher logic
	}*/

	batch := cass.conn.NewBatch(gocql.LoggedBatch)
	tName := cass.db.MetricTable()
	if cass.tablePerResolution {
		tName = tName + "_" + strconv.Itoa(int(name.Resolution)) + "s"
	}

	for _, stat := range points {
		DOQ := fmt.Sprintf(cass.insertQuery, tName)
		if stat.Name.Ttl > 0 {
			DOQ += " USING TTL " + strconv.Itoa(int(stat.Name.Ttl))
		}
		if cass.tablePerResolution {
			batch.Query(DOQ,
				name.UniqueIdString(),
				stat.ToTime().UnixNano(),
				float64(stat.Sum),
				float64(stat.Min),
				float64(stat.Max),
				float64(stat.Last),
				stat.Count,
			)

		} else {
			batch.Query(DOQ,
				name.UniqueIdString(),
				int64(stat.Name.Resolution),
				stat.ToTime().UnixNano(),
				float64(stat.Sum),
				float64(stat.Min),
				float64(stat.Max),
				float64(stat.Last),
				stat.Count,
			)
		}

	}
	err := cass.conn.ExecuteBatch(batch)
	if err != nil {
		cass.log.Error("Cassandra Driver:Batch Metric insert failed, %v", err)
		stats.StatsdClientSlow.Incr("writer.cassandraflat.batch.metric-failures", 1)
		return 0, err
	}
	stats.StatsdClientSlow.Incr("writer.cassandraflat.batch.writes", 1)
	stats.StatsdClientSlow.GaugeAvg("writer.cassandraflat.batch.metrics-per-writes", int64(l))

	return l, nil
}

func (cass *CassandraFlatWriter) InsertOne(name *repr.StatName, stat *repr.StatRepr) (int, error) {

	defer stats.StatsdNanoTimeFunc("writer.cassandraflat.write.metric-time-ns", time.Now())

	ttl := uint32(0)
	if stat.Name.Ttl > 0 {
		ttl = stat.Name.Ttl
	}
	writeStat := cass.mergeWrite(stat)

	tName := cass.db.MetricTable()
	args := []interface{}{
		name.UniqueIdString(),
		int64(stat.Name.Resolution),
		stat.ToTime().UnixNano(),
		float64(writeStat.Sum),
		float64(writeStat.Min),
		float64(writeStat.Max),
		float64(writeStat.Last),
		writeStat.Count,
	}
	if cass.tablePerResolution {
		tName = fmt.Sprintf("%s_%ds", tName, name.Resolution)
		args = []interface{}{
			name.UniqueIdString(),
			stat.ToTime().UnixNano(),
			float64(writeStat.Sum),
			float64(writeStat.Min),
			float64(writeStat.Max),
			float64(writeStat.Last),
			writeStat.Count,
		}
	}
	Q := fmt.Sprintf(cass.insertQuery, tName)

	if ttl > 0 {
		Q += " USING TTL ?"
		args = append(args, ttl)
	}

	err := cass.conn.Query(Q, args...).Exec()

	//cass.log.Critical("METRICS WRITE %d: %v", ttl, stat)
	if err != nil {
		cass.log.Error("Cassandra Driver: insert failed, %v", err)
		stats.StatsdClientSlow.Incr("writer.cassandraflat.metric-failures", 1)

		return 0, err
	}
	stats.StatsdClientSlow.Incr("writer.cassandraflat.metric-writes", 1)

	return 1, nil
}

// pop from the cache and send to actual writers
func (cass *CassandraFlatWriter) sendToWriters() error {
	// this may not be the "greatest" rate-limiter of all time,
	// as "high frequency tickers" can be costly .. but should the workers get backed-up
	// it will block on the write_queue stage

	//ye old unlimited
	if cass.writesPerSecond <= 0 {
		cass.log.Notice("Starting metric writer: No Write limiter")

		for {
			if cass.shutitdown {
				return nil
			}

			name, points := cass.cacher.Pop()
			switch points {
			case nil:
				time.Sleep(time.Second)
			default:
				stats.StatsdClient.Incr(fmt.Sprintf("writer.cassandraflat.write.send-to-writers"), 1)
				cass.dispatcher.Add(&cassandraFlatMetricJob{Cass: cass, Stats: points, Name: name})
			}
		}
	} else {

		sleep_t := float64(time.Second) * (time.Second.Seconds() / float64(cass.writesPerSecond))
		cass.log.Notice("Starting metric writer: limiter every %f nanoseconds (%d writes per second)", sleep_t, cass.writesPerSecond)
		dur := time.Duration(int(sleep_t))

		for {
			if cass.shutitdown {
				return nil
			}

			name, points := cass.cacher.Pop()

			switch points {
			case nil:
				time.Sleep(time.Second)
			default:

				stats.StatsdClient.Incr(fmt.Sprintf("writer.cassandraflat.write.send-to-writers"), 1)
				cass.dispatcher.Add(&cassandraFlatMetricJob{Cass: cass, Stats: points, Name: name})
				time.Sleep(dur)
			}

		}
	}
}

func (cass *CassandraFlatWriter) WriteWithOffset(stat *repr.StatRepr, offset *smetrics.OffsetInSeries) error {

	//cache keys needs metric + resolution
	// turning off
	if !cass.shutitdown {
		cass.cacher.AddWithOffset(stat.Name, stat, offset)
	}

	return nil

}

func (cass *CassandraFlatWriter) Write(stat *repr.StatRepr) error {
	//cache keys needs metric + resolution
	return cass.WriteWithOffset(stat, nil)
}

/****************** Metrics Writer *********************/
type CassandraFlatMetric struct {
	WriterBase

	writer *CassandraFlatWriter

	renderTimeout time.Duration
}

func NewCassandraFlatMetrics() *CassandraFlatMetric {
	return new(CassandraFlatMetric)
}

func (cass *CassandraFlatMetric) Driver() string {
	return "cassandra-flat"
}

func (cass *CassandraFlatMetric) Start() {
	cass.writer.resolutions = cass.GetResolutions()
	cass.writer.Start()
}

func (cass *CassandraFlatMetric) Stop() {
	cass.writer.Stop()
}

func (cass *CassandraFlatMetric) Config(conf *options.Options) (err error) {
	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("Metrics: `dsn` (server1,server2,server3) is needed for cassandra-flat config")
	}
	res, err := conf.Int64Required("resolution")
	if err != nil {
		return fmt.Errorf("Resolution needed for cassandra-flat writer")
	}

	cass.name = conf.String("name", "metrics:cassandraflat:"+dsn)

	// FLAT writers have different "writers" for each resolution,  so we need to set different names for them
	// the API readers use the "first one", so if things already exist w/ the name, slap on a resolution tag
	if GetMetrics(cass.Name()) != nil {
		cass.name = conf.String("name", "metrics:cassandraflat:"+dsn) + fmt.Sprintf(":%v", res)
	}
	err = RegisterMetrics(cass.Name(), cass)
	if err != nil {
		return err
	}

	gots, err := NewCassandraFlatWriter(conf, cass.GetResolutions())
	if err != nil {
		return err
	}
	gots.currentResolution = cass.currentResolution

	cass.writer = gots
	cass.options = conf

	cacheKey := fmt.Sprintf("cassandraflat:cache:%s:%v", conf.String("dsn", ""), res)
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

	gTag := conf.String("tags", "")
	if len(gTag) > 0 {
		cass.staticTags = repr.SortingTagsFromString(gTag)
	}

	// prevent a reader from squshing this cacher
	if !gots.cacher.started && !gots.cacher.inited {
		gots.cacher.inited = true
		// set the cacher bits
		gots.cacher.maxKeys = int(conf.Int64("cache_metric_size", int64(gots.cacher.maxKeys)))
		gots.cacher.maxBytes = int(conf.Int64("cache_byte_size", int64(gots.cacher.maxBytes)))
		gots.cacher.lowFruitRate = conf.Float64("cache_low_fruit_rate", gots.cacher.lowFruitRate)
		gots.cacher.seriesType = conf.String("cache_series_type", gots.cacher.seriesType)
		gots.cacher.overFlowMethod = conf.String("cache_overflow_method", gots.cacher.overFlowMethod)

		if gots.cacher.overFlowMethod == "chan" {
			gots.cacheOverFlowListener = gots.cacher.GetOverFlowChan()
			go gots.overFlowWrite()
		}

	}

	return nil
}

func (cass *CassandraFlatMetric) Write(stat *repr.StatRepr) error {
	return cass.WriteWithOffset(stat, nil)
}

// simple proxy to the cacher
func (cass *CassandraFlatMetric) WriteWithOffset(stat *repr.StatRepr, offset *smetrics.OffsetInSeries) error {
	// write the index from the cache as indexing can be slooowwww
	// keep note of this, when things are not yet "warm" (the indexer should
	// keep tabs on what it's already indexed for speed sake,
	// the push "push" of stats will cause things to get pretty slow for a while
	stat.Name.MergeMetric2Tags(cass.staticTags)

	// only the lowest res needs to write the index
	if cass.ShouldWriteIndex() {
		cass.indexer.Write(*stat.Name)
	}

	return cass.writer.WriteWithOffset(stat, offset)
}

/************************ READERS ****************/

func (cass *CassandraFlatMetric) RawRenderOne(ctx context.Context, metric *sindexer.MetricFindItem, start int64, end int64, resample uint32) (*smetrics.RawRenderItem, error) {
	sp, closer := cass.GetSpan("RawRenderOne", ctx)
	sp.LogKV("driver", "CassandraFlatMetric", "metric", metric.StatName().Key, "uid", metric.UniqueId)
	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("reader.cassandraflat.renderraw.get-time-ns", time.Now())

	rawd := new(smetrics.RawRenderItem)

	if metric.Leaf == 0 { //data only
		return rawd, fmt.Errorf("Cassandra: RawRenderOne: Not a data node")
	}

	//figure out the best res
	resolution := cass.GetResolution(start, end)
	outResolution := resolution

	//obey the bigger
	if resample > resolution {
		outResolution = resample
	}

	start = TruncateTimeTo(start, int(resolution))
	end = TruncateTimeTo(end, int(resolution))

	b_len := uint32(end-start) / outResolution //just to be safe
	if b_len <= 0 {
		return rawd, fmt.Errorf("Cassandra: RawRenderOne: time too narrow")
	}

	// time in cassandra is in NanoSeconds so we need to pad the times from seconds -> nanos
	nano := int64(time.Second)
	nanoEnd := end * nano
	nanoStart := start * nano

	firstT := uint32(start)
	lastT := uint32(end)

	// try the write inflight cache as nothing is written yet
	stat_name := metric.StatName()
	inflightRenderitem, err := cass.writer.cacher.GetAsRawRenderItem(stat_name)

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

	tName := cass.writer.db.MetricTable()
	args := []interface{}{metric.UniqueId, resolution, nanoEnd, nanoStart}
	if cass.writer.tablePerResolution {
		tName = fmt.Sprintf("%s_%ds", tName, resolution)
		args = []interface{}{metric.UniqueId, nanoEnd, nanoStart}
	}
	Q := fmt.Sprintf(cass.writer.selectTimeQuery, tName)

	// grab ze data. (note data is already sorted by time asc va the cassandra schema)
	iter := cass.writer.conn.Query(Q, args...).Iter()

	var t, count int64
	var min, max, sum, last float64

	mKey := metric.Id

	// sorting order for the table is time ASC (i.e. firstT == first entry)

	Tstart := uint32(start)
	curPt := smetrics.NullRawDataPoint(Tstart)

	// on resamples (if >0 ) we simply merge points until we hit the time steps
	do_resample := resample > 0 && resample > resolution

	for iter.Scan(&max, &min, &sum, &last, &count, &t) {
		t := uint32(time.Unix(0, t).Unix())
		if do_resample {
			if t >= Tstart+resample {
				Tstart += resample
				rawd.Data = append(rawd.Data, curPt)
				curPt = &smetrics.RawDataPoint{
					Count: count,
					Sum:   sum,
					Max:   max,
					Min:   min,
					Last:  last,
					Time:  t,
				}
			} else {
				curPt.Merge(&smetrics.RawDataPoint{
					Count: count,
					Sum:   sum,
					Max:   max,
					Min:   min,
					Last:  last,
					Time:  t,
				})
			}
		} else {
			rawd.Data = append(rawd.Data, &smetrics.RawDataPoint{
				Count: count,
				Sum:   sum,
				Max:   max,
				Min:   min,
				Last:  last,
				Time:  t,
			})
		}
		lastT = t
	}
	if !curPt.IsNull() {
		rawd.Data = append(rawd.Data, curPt)
	}

	if err := iter.Close(); err != nil {
		cass.writer.log.Error("RawRender: Failure closing iterator: %v", err)
	}

	if len(rawd.Data) > 0 && rawd.Data[0].Time > 0 {
		firstT = rawd.Data[0].Time
	}

	//cass.log.Critical("METR: %s Start: %d END: %d LEN: %d GotLen: %d", metric.Id, firstT, lastT, len(dPoints), ct)

	rawd.RealEnd = uint32(lastT)
	rawd.RealStart = uint32(firstT)
	rawd.Start = uint32(start)
	rawd.End = uint32(end)
	rawd.Step = outResolution
	rawd.Metric = mKey
	rawd.Id = metric.UniqueId
	rawd.Tags = metric.Tags
	rawd.MetaTags = metric.MetaTags
	rawd.AggFunc = stat_name.AggType()

	// grab the "current inflight" from the cache and merge into the main array
	if inflightRenderitem != nil && len(inflightRenderitem.Data) > 1 {
		//merge with any inflight bits (inflight has higher precedence over the file)
		inflightRenderitem.RealEnd = uint32(start)
		inflightRenderitem.RealStart = uint32(end)

		inflightRenderitem.MergeWithResample(rawd, outResolution)
		return inflightRenderitem, nil
	}

	return rawd, nil
}

func (cass *CassandraFlatMetric) RawRender(ctx context.Context, path string, from int64, to int64, tags repr.SortingTags, resample uint32) ([]*smetrics.RawRenderItem, error) {
	sp, closer := cass.GetSpan("RawRender", ctx)
	sp.LogKV("driver", "CassandraFlatMetric", "path", path, "from", from, "to", to)
	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("reader.cassandraflat.rawrender.get-time-ns", time.Now())

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
			stats.StatsdClientSlow.Incr("reader.cassandraflat.rawrender.errors", 1)
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
				stats.StatsdClientSlow.Incr("reader.cassandraflat.rawrender.timeouts", 1)
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

	stats.StatsdClientSlow.Incr("reader.cassandraflat.rawrender.metrics-per-request", int64(len(metrics)))

	return rawd, nil
}
