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
	THe MySQL stat write for "binary" blobs of time series

     CREATE TABLE `{table}{prefix}` (
      `id` BIGINT unsigned NOT NULL AUTO_INCREMENT,
      `uid` varchar(50) CHARACTER SET ascii NOT NULL,
      `path` varchar(255) NOT NULL DEFAULT '',
      `ptype` TINYINT NOT NULL,
      `points` blob,
      `stime` BIGINT unsigned NOT NULL,
      `etime` BIGINT unsigned NOT NULL,
      PRIMARY KEY (`id`),
      KEY `uid` (`uid`, `etime`),
      KEY `path` (`path`)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8;

	Prefixes are `_{resolution}s` (i.e. "_" + (uint32 resolution) + "s")

	OPTIONS: For `Config`

	table="metrics"

	# series (and cache) encoding types
	series_encoding="gorilla"

	# the "internal carbon-like-cache" size (ram is your friend)
	# if there are more then this many metric keys in the system, newer ones will be DROPPED
	cache_metric_size=102400

	# number of points per metric to cache before we write them
	# note you'll need AT MOST cache_byte_size * cache_metric_size * 2*8 bytes of RAM
	# this is also the "chunk" size stored in Mysql
	cache_byte_size=8192

	# we write the blob after this much time even if it has not reached the byte limit
	# one hour default
	cache_longest_time=3600s

	# Rollup Type
	# this can be either `cache` or `triggered`
	# cache = means we keep each resolution in Ram and flush when approriate
	# triggered = just keep the lowest resoltuion in ram, upon a write, trigger the other resolutions to get
	# rolledup and written
	# the Cached version can take a lot of RAM (NumResolutions * Metrics * BlobSize) but is better for queries
	# that over the "nowish" timescale as things are in RAM
	rollupType="cached"

*/

package metrics

import (
	"cadent/server/broadcast"
	"cadent/server/dispatch"
	sindexer "cadent/server/schemas/indexer"
	smetrics "cadent/server/schemas/metrics"
	"cadent/server/schemas/repr"
	"cadent/server/series"
	"cadent/server/stats"
	"cadent/server/utils"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/dbs"
	"database/sql"
	"errors"
	"fmt"
	"github.com/cenkalti/backoff"
	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/net/context"
	logging "gopkg.in/op/go-logging.v1"
	"sync"
	"time"
)

const (
	MYSQL_RENDER_TIMEOUT                = "5s" // 5 second time out on any render
	MYSQL_DEFAULT_ROLLUP_TYPE           = "cached"
	MYSQL_DEFAULT_METRIC_WORKERS        = 16
	MYSQL_DEFAULT_METRIC_QUEUE_LEN      = 1024 * 100
	MYSQL_DEFAULT_METRIC_RETRIES        = 2
	MYSQL_DEFAULT_METRIC_RENDER_WORKERS = 4
	MYSQL_DEFAULT_EXPIRE_TICK           = "5m"
)

// common errors to avoid GC pressure
var ErrorNotADataNode = errors.New("Render: Not a data node")
var ErrorTimeTooSmall = errors.New("Render: time too narrow")

/************************************************************************/
/**********  Standard Worker Dispatcher JOB   ***************************/
/************************************************************************/
// insert job queue workers
type mysqlBlobMetricJob struct {
	My *MySQLMetrics
	Ts *smetrics.TotalTimeSeries // where the point list live
}

func (j mysqlBlobMetricJob) DoWork() error {
	err := j.My.doInsert(j.Ts)
	return err
}

/****************** Interfaces *********************/
type MySQLMetrics struct {
	CacheBase

	driver string
	db     *dbs.MySQLDB
	conn   *sql.DB

	renderTimeout time.Duration

	// run an periodic "expire" (aka delete) metrics from the tables
	// based on the TTLs given and the "endtime" in the DB
	runExpire  bool
	expireTick time.Duration

	cacheOverFlow *broadcast.Listener // on byte overflow of cacher force a write

	// if the rolluptype == cached, then we this just uses the internal RAM caches
	// otherwise if "trigger" we only have the lowest res cache, and trigger rollups on write
	rollupType string
	rollup     *RollupMetric
	doRollup   bool

	// dispatcher worker Queue
	num_workers      int
	queue_len        int
	dispatch_retries int
	dispatcher       *dispatch.DispatchQueue

	log *logging.Logger
}

func NewMySQLMetrics() *MySQLMetrics {
	my := new(MySQLMetrics)
	my.driver = "mysql"
	my.isPrimary = false
	my.log = logging.MustGetLogger("writers.mysql")
	return my
}

func NewMySQLTriggeredMetrics() *MySQLMetrics {
	my := new(MySQLMetrics)
	my.driver = "mysql-triggered"
	my.rollupType = "triggered"
	my.isPrimary = false
	my.log = logging.MustGetLogger("writers.mysql")
	return my
}

func (my *MySQLMetrics) Config(conf *options.Options) error {
	my.options = conf

	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("`dsn` (user:pass@tcp(host:port)/db) is needed for mysql config")
	}

	resolution, err := conf.Float64Required("resolution")
	if err != nil {
		return fmt.Errorf("Resolution needed for mysql writer")
	}

	// newDB is a "cached" item to let us pass the connections around
	db_key := dsn + fmt.Sprintf("%f", resolution) + conf.String("table", "metrics")
	db, err := dbs.NewDB("mysql", db_key, conf)
	if err != nil {
		return err
	}

	my.name = conf.String("name", "metrics:mysql:"+dsn)
	err = RegisterMetrics(my.Name(), my)
	if err != nil {
		return err
	}

	my.db = db.(*dbs.MySQLDB)
	my.conn = db.Connection().(*sql.DB)

	_tgs := conf.String("tags", "")
	if len(_tgs) > 0 {
		my.staticTags = repr.SortingTagsFromString(_tgs)
	}

	// rolluptype
	if my.rollupType == "" {
		my.rollupType = conf.String("rollupType", MYSQL_DEFAULT_ROLLUP_TYPE)
	}

	// tweak queues and worker sizes
	my.num_workers = int(conf.Int64("write_workers", MYSQL_DEFAULT_METRIC_WORKERS))
	my.queue_len = int(conf.Int64("write_queue_length", MYSQL_DEFAULT_METRIC_QUEUE_LEN))
	my.dispatch_retries = int(conf.Int64("write_queue_retries", MYSQL_DEFAULT_METRIC_RETRIES))
	my.runExpire = conf.Bool("expire_on_ttl", true)

	my.expireTick, err = time.ParseDuration(MYSQL_DEFAULT_EXPIRE_TICK)
	if err != nil {
		return err
	}

	_cache, err := conf.ObjectRequired("cache")
	if err != nil {
		return errMetricsCacheRequired
	}
	my.cacher = _cache.(Cacher)

	//force single cacher
	_, ok := my.cacher.(*CacherSingle)
	if !ok {
		return ErrorMustBeSingleCacheType
	}

	my.cacherPrefix = my.cacher.GetPrefix()

	// for the overflow cached items::
	// these caches can be shared for a given writer set, and the caches may provide the data for
	// multiple writers, we need to specify that ONE of the writers is the "main" one otherwise
	// the Metrics Write function will add the points over again, which is not a good thing
	// when the accumulator flushes things to the multi wrtiers
	// The Writer needs to know it's "not" the primary writer and thus will not "add" points to the
	// cache .. so the cache basically gets "one" primary writer pointed (first come first serve)
	my.isPrimary = my.cacher.SetPrimaryWriter(my)
	if my.isPrimary {
		my.log.Notice("Mysql series writer is the primary writer to write back cache for %s", my.cacher.GetName())
	}

	if my.rollupType == "triggered" {
		my.driver = "mysql-triggered"
		my.rollup = NewRollupMetric(my, my.cacher.GetMaxBytesPerMetric())
	}

	rdur, err := time.ParseDuration(MYSQL_RENDER_TIMEOUT)
	if err != nil {
		return err
	}
	my.renderTimeout = rdur

	return nil
}

func (my *MySQLMetrics) Driver() string {
	return my.driver
}

func (my *MySQLMetrics) Start() {
	my.startstop.Start(func() {

		// now we make sure the metrics schemas are added
		err := NewMySQLMetricsSchema(my.conn, my.db.RootMetricsTableName(), my.resolutions, "blob").AddMetricsTable()
		if err != nil {
			panic(err)
		}

		my.log.Notice("Starting mysql writer for %s at %d bytes per series", my.db.Tablename(), my.cacher.GetMaxBytesPerMetric())

		my.cacher.SetOverFlowMethod("chan")
		my.cacher.Start()

		// only register this when we start as we really want to consume
		my.cacheOverFlow = my.cacher.GetOverFlowChan()
		go my.overFlowWrite()

		// if the resolutions list is just "one" there is no triggered rollups
		if len(my.resolutions) == 1 {
			my.rollupType = "cached"
		}
		my.log.Notice("Rollup Type: %s on resolution: %d (min resolution: %d)", my.rollupType, my.currentResolution, my.resolutions[0][0])

		my.doRollup = my.rollupType == "triggered" && my.currentResolution == my.resolutions[0][0]
		// start the rolluper if needed
		if my.doRollup {
			my.log.Notice("Starting rollup machine")
			// all but the lowest one
			my.rollup.blobMaxBytes = int(my.options.Int64("rollup_byte_size", CASSANDRA_LOG_ROLLLUP_MAXBYTES))
			my.rollup.numRetries = int(my.options.Int64("rollup_retries", ROLLUP_NUM_RETRIES))
			my.rollup.numWorkers = int(my.options.Int64("rollup_workers", ROLLUP_NUM_WORKERS))
			my.rollup.queueLength = int(my.options.Int64("rollup_queue_length", ROLLUP_QUEUE_LENGTH))
			my.rollup.consumeQueueWorkers = int(my.options.Int64("rollup_queue_workers", ROLLUP_NUM_QUEUE_WORKERS))

			// queue or blast
			my.rollup.SetRunMode(my.options.String("rollup_method", CASSANDRA_LOG_ROLLLUP_MODE))
			my.rollup.SetResolutions(my.resolutions[1:])
			my.rollup.SetMinResolution(my.resolutions[0][0])

			go my.rollup.Start()
		}

		my.dispatcher = dispatch.NewDispatchQueue(
			my.num_workers,
			my.queue_len,
			my.dispatch_retries,
		)
		my.dispatcher.Start()

		if my.runExpire {
			go my.RunPeriodicExpire()
		}
	})
}

func (my *MySQLMetrics) Stop() {
	my.log.Warning("Stopping Mysql writer for (%s)", my.cacher.GetName())
	my.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()

		if my.shutitdown {
			return // already did
		}
		my.shutitdown = true

		my.cacher.Stop()

		//have to cast thie
		scache := my.cacher.(*CacherSingle)

		mets := scache.Cache
		mets_l := len(mets)
		my.log.Warning("Shutting down %s and exhausting the queue (%d items) and quiting", my.cacher.GetName(), mets_l)

		// full tilt write out
		procs := 16
		go_do := make(chan smetrics.TotalTimeSeries, procs)
		wg := sync.WaitGroup{}

		goInsert := func() {
			for {
				select {
				case s, more := <-go_do:
					if !more {
						return
					}
					stats.StatsdClient.Incr(fmt.Sprintf("writer.cache.shutdown.send-to-writers"), 1)
					_, err := my.InsertSeries(s.Name, s.Series)
					if my.doRollup {
						my.rollup.DoRollup(&s)
					}
					if err != nil {
						my.log.Errorf("Insert error: %v", err)
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
				my.log.Warning("shutdown purge: written %d/%d...", did, mets_l)
			}
			if queueitem.Series != nil {
				go_do <- smetrics.TotalTimeSeries{Name: queueitem.Name, Series: queueitem.Series.Copy()}
			}
			did++
		}
		wg.Wait()

		close(go_do)
		my.log.Warning("shutdown purge: written %d/%d...", did, mets_l)

		if my.dispatcher != nil {
			my.dispatcher.Stop()
		}

		my.log.Warning("Shutdown finished ... quiting mysql blob writer")
		return
	})
}

func (my *MySQLMetrics) doInsert(ts *smetrics.TotalTimeSeries) (err error) {
	stats.StatsdClientSlow.Incr("writer.mysql.consume.add", 1)

	if my.doRollup {
		my.rollup.Add(ts)
	}
	_, err = my.InsertSeries(ts.Name, ts.Series)
	if err != nil {
		my.log.Errorf("Failed to add series to DB: %s (%s) %v", ts.Name.Key, ts.Name.UniqueIdString(), err)
	}
	return err
}

// listen to the overflow chan from the cache and attempt to write "now"
func (my *MySQLMetrics) overFlowWrite() {
	for {
		statitem, more := <-my.cacheOverFlow.Ch

		// bail
		if !more {
			return
		}
		stats.StatsdClientSlow.Incr("writer.mysql.queue.add", 1)
		my.dispatcher.Add(
			&mysqlBlobMetricJob{
				My: my,
				Ts: statitem.(*smetrics.TotalTimeSeries),
			},
		)
	}
}

func (my *MySQLMetrics) RunPeriodicExpire() {

	ticker := time.NewTicker(my.expireTick)
	base_Q := "DELETE FROM %s_%ds WHERE etime < ?"
	RM_factor := 5
	for {
		<-ticker.C
		// we remove 5*ttl as we may need to do rollups based on the deleted data
		for _, res := range my.resolutions {
			// ttls are in seconds
			ttl := res[1]
			if ttl <= 0 {
				continue
			}

			tName := my.db.RootMetricsTableName()
			Q := fmt.Sprintf(base_Q, tName, res[0])

			exp_time := (time.Now().Unix() - int64(RM_factor*ttl)) * int64(time.Second)

			my.log.Info("Running expire for all metrics older then %d in the %ds resolution", exp_time, res[0])
			result, err := my.conn.Exec(Q, exp_time)

			if err != nil {
				my.log.Errorf("Failed to delete metrics older then %d in %ds resolution", exp_time, res[0])
			} else {
				r_eff, _ := result.RowsAffected()
				stats.StatsdClientSlow.Incr(fmt.Sprintf("writer.mysql.expire.%ds.rows.deleted", res[0]), r_eff)
				my.log.Noticef("Removed %d metrics older then %d in the %ds resolution", r_eff, exp_time, res[0])

			}
		}
	}

}

func (my *MySQLMetrics) InsertSeries(name *repr.StatName, timeseries series.TimeSeries) (int, error) {
	return my.InsertDBSeries(name, timeseries, name.Resolution)
}

// Write simple proxy to the
func (my *MySQLMetrics) Write(stat *repr.StatRepr) error {
	return my.WriteWithOffset(stat, nil)
}

// WriteWithOffset push a metric into the cache pile with the kafka offset
func (my *MySQLMetrics) WriteWithOffset(stat *repr.StatRepr, offset *smetrics.OffsetInSeries) error {

	stat.Name.MergeMetric2Tags(my.staticTags)

	// only need to do this if the first resolution
	if my.ShouldWriteIndex() {
		my.indexer.Write(*stat.Name)
	}

	// not primary writer .. move along
	if !my.isPrimary {
		return nil
	}

	// and now add it to the readcache iff it's been activated
	r_cache := GetReadCache()
	if my.currentResolution == my.resolutions[0][0] {
		if r_cache != nil {
			r_cache.InsertQueue <- stat
		}
	}

	if my.rollupType == "triggered" {
		if my.currentResolution == my.resolutions[0][0] {
			my.cacher.AddWithOffset(stat.Name, stat, offset)
		}
	} else {
		my.cacher.AddWithOffset(stat.Name, stat, offset)
	}
	return nil
}

/************** READING ********************/

// grab the time series from the DBs
func (my *MySQLMetrics) GetFromDatabase(metric *sindexer.MetricFindItem, resolution uint32, start int64, end int64, resample uint32) (rawd *smetrics.RawRenderItem, err error) {
	// i.e metrics_5s
	tName := my.db.RootMetricsTableName()
	rawd = new(smetrics.RawRenderItem)

	Q := fmt.Sprintf(
		"SELECT ptype, points FROM %s_%ds WHERE uid=? AND etime >= ? AND etime <= ?",
		tName, resolution,
	)

	// times need to be in Nanos, but coming as a epoch
	// time in cassandra is in NanoSeconds so we need to pad the times from seconds -> nanos
	nano := int64(time.Second)
	nanoEnd := end * nano
	nanoStart := start * nano

	vals := []interface{}{
		metric.UniqueId,
		nanoStart,
		nanoEnd,
	}

	rows, err := my.conn.Query(Q, vals...)
	//my.log.Critical("Q: %s, %v", Q, vals)

	if err != nil {
		my.log.Error("Mysql Driver: Metric select failed, %v", err)
		return rawd, err
	}
	defer rows.Close()

	// for each "series" we get make a list of points
	uStart := uint32(start)
	uEnd := uint32(end)

	statName := metric.StatName()
	rawd.Start = uStart
	rawd.End = uEnd
	rawd.Step = resolution
	rawd.Id = metric.UniqueId
	rawd.Metric = metric.Path
	rawd.AggFunc = statName.AggType()
	rawd.Tags = metric.Tags
	rawd.MetaTags = metric.MetaTags

	tStart := uint32(start)
	doResample := resample > 0 && resample > resolution
	if doResample {
		tStart, uEnd = ResampleBounds(tStart, uEnd, resample)
		rawd.Start = tStart
		rawd.End = uEnd
		rawd.Step = resample
	}
	for rows.Next() {
		var pType uint8
		var pBytes []byte
		if err := rows.Scan(&pType, &pBytes); err != nil {
			return rawd, err
		}
		s_name := series.NameFromId(pType)
		sIter, err := series.NewIter(s_name, pBytes)
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
			switch doResample {
			case true:
				if t >= tStart {
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
					rd := smetrics.GetRawDataPoint()
					rd.Count = ct
					rd.Max = mx
					rd.Min = mi
					rd.Last = ls
					rd.Sum = su
					rd.Time = t
					curPt.Merge(rd)
					smetrics.PutRawDataPoint(rd)
				}
			case false:
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
	}

	if len(rawd.Data) > 0 {
		rawd.RealEnd = rawd.Data[len(rawd.Data)-1].Time
		rawd.RealStart = rawd.Data[0].Time
	}
	if err := rows.Err(); err != nil {
		return rawd, err
	}

	return rawd, nil

}

func (my *MySQLMetrics) RawDataRenderOne(ctx context.Context, metric *sindexer.MetricFindItem, start int64, end int64, resample uint32) (*smetrics.RawRenderItem, error) {
	sp, closer := my.GetSpan("RawDataRenderOne", ctx)
	sp.LogKV("driver", "MySQLMetrics", "metric", metric.StatName().Key, "uid", metric.UniqueId)

	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("reader.mysql.renderraw.get-time-ns", time.Now())

	preped := my.newGetPrep(metric, start, end, resample)

	if metric.Leaf == 0 {
		//data only but return a "blank" data set otherwise graphite no likey
		return preped.rawd, ErrorNotADataNode
	}

	b_len := (preped.uEnd - preped.uStart) / preped.outResolution //just to be safe
	if b_len <= 0 {
		return preped.rawd, ErrorTimeTooSmall
	}

	//cache check
	// the read cache should have "all" the points from a "start" to "end" if the read cache has been activated for
	// a while.  If not, then it's a partial list (basically the read cache just started)
	// the res is not known so set it to the min available
	inflight, err := my.GetFromWriteCache(metric, preped.uStart, preped.uEnd, 1)
	// need at LEAST 2 points to get the proper step size
	if inflight != nil && err == nil && len(inflight.Data) > 1 {
		// if all the data is in this list we don't need to go any further
		if inflight.RealStart <= preped.uStart {
			inflight.RealEnd = preped.uStart
			inflight.RealStart = preped.uEnd

			inflight.Resample(preped.outResolution) // need resample to deal w/ time duplicates
			stats.StatsdClientSlow.Incr("reader.mysql.renderraw.inflight.total", 1)
			return inflight, err
		}
	}
	if err != nil {
		my.log.Errorf("Error getting inflight data: %v", err)
	}

	// and now for the mysql Query otherwise
	mysqlData, err := my.GetFromDatabase(metric, preped.dbResolution, start, end, preped.outResolution)
	if err != nil {
		my.log.Errorf("Error getting from DB: %v", err)
		return preped.rawd, err
	}

	//mysql_data.PrintPoints()
	if inflight == nil || len(inflight.Data) == 0 {
		return mysqlData, nil
	}

	// pin to desired range
	inflight.RealEnd = preped.uStart
	inflight.RealStart = preped.uEnd

	if len(mysqlData.Data) > 0 && len(inflight.Data) > 0 {
		//inflight.Quantize()
		//mysql_data.Quantize()
		inflight.MergeWithResample(mysqlData, preped.outResolution)

		/*oo_data := inflight.Data[0:]
		for idx, d := range oo_data {
			fmt.Printf("%d: %d: %f | %f | %f\n", idx, d.Time, d.Sum, mysql_data.Data[idx].Sum, inflight.Data[idx].Sum)
		}
		*/
		return inflight, nil
	}
	if inflight.Step != preped.outResolution {
		inflight.Resample(preped.outResolution)
	}
	return inflight, nil
}

func (my *MySQLMetrics) RawRender(ctx context.Context, path string, from int64, to int64, tags repr.SortingTags, resample uint32) ([]*smetrics.RawRenderItem, error) {
	sp, closer := my.GetSpan("RawRender", ctx)
	sp.LogKV("driver", "MySQLMetrics", "path", path, "from", from, "to", to)
	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("reader.mysql.rawrender.get-time-ns", time.Now())

	paths := SplitNamesByComma(path)
	var metrics []*sindexer.MetricFindItem

	render_wg := utils.GetWaitGroup()
	defer utils.PutWaitGroup(render_wg)

	for _, pth := range paths {
		mets, err := my.indexer.Find(ctx, pth, tags)
		if err != nil {
			continue
		}
		metrics = append(metrics, mets...)
	}

	rawd := make([]*smetrics.RawRenderItem, 0, len(metrics))

	procs := MYSQL_DEFAULT_METRIC_RENDER_WORKERS

	jobs := make(chan *sindexer.MetricFindItem, len(metrics))
	results := make(chan *smetrics.RawRenderItem, len(metrics))

	renderOne := func(met *sindexer.MetricFindItem) *smetrics.RawRenderItem {
		_ri, err := my.RawDataRenderOne(ctx, met, from, to, resample)

		if err != nil {
			stats.StatsdClientSlow.Incr("reader.mysql.rawrender.errors", 1)
			my.log.Error("Read Error for %s (%d->%d) : %v", path, from, to, err)
		}
		return _ri
	}

	// ye old fan out technique but not "too many" as to kill the server
	jobWorker := func(jober int, taskqueue <-chan *sindexer.MetricFindItem, resultqueue chan<- *smetrics.RawRenderItem) {
		recChan := make(chan *smetrics.RawRenderItem, 1)
		for met := range taskqueue {
			met := met // https://golang.org/doc/faq#closures_and_goroutines

			go func() { recChan <- renderOne(met) }()
			select {
			case <-time.After(my.renderTimeout):
				stats.StatsdClientSlow.Incr("reader.mysql.rawrender.timeouts", 1)
				my.log.Errorf("Render Timeout for %s (%d->%d)", path, from, to)
				resultqueue <- nil
			case res := <-recChan:
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
	stats.StatsdClientSlow.Incr("reader.mysql.rawrender.metrics-per-request", int64(len(metrics)))

	return rawd, nil
}

/*************** Match the DBMetrics Interface ***********************/

// given a name get the latest metric series
func (my *MySQLMetrics) GetLatestFromDB(name *repr.StatName, resolution uint32) (smetrics.DBSeriesList, error) {
	tName := my.db.RootMetricsTableName()
	Q := fmt.Sprintf(
		"SELECT id, uid, stime, etime, ptype, points FROM %s_%ds WHERE uid=? ORDER BY etime DESC LIMIT 1",
		tName, resolution,
	)
	rows, err := my.conn.Query(Q, name.UniqueIdString())

	if err != nil {
		my.log.Error("Mysql Driver: Metric select failed, %v", err)
		return nil, err
	}
	defer rows.Close()

	rawd := make(smetrics.DBSeriesList, 0)
	var pType uint8
	var pBytes []byte
	var uid string
	var id, start, end int64
	for rows.Next() {
		if err := rows.Scan(&id, &uid, &start, &end, &pType, &pBytes); err != nil {
			return nil, err
		}
		dataums := &smetrics.DBSeries{
			Id:         id,
			Uid:        uid,
			Start:      start,
			End:        end,
			Ptype:      pType,
			Pbytes:     pBytes,
			Resolution: resolution,
		}
		rawd = append(rawd, dataums)
	}
	if err := rows.Err(); err != nil {
		return rawd, err
	}

	return rawd, nil
}

// given a name get the latest metric series
func (my *MySQLMetrics) GetRangeFromDB(name *repr.StatName, start uint32, end uint32, resolution uint32) (smetrics.DBSeriesList, error) {
	tName := my.db.RootMetricsTableName()
	Q := fmt.Sprintf(
		"SELECT id, uid, stime, etime, ptype, points FROM %s_%ds WHERE uid=? AND etime >= ? AND etime <= ?",
		tName, resolution,
	)

	// need to convert second time to nan time
	nano := int64(time.Second)
	nano_end := int64(end) * nano
	nano_start := int64(start) * nano

	rows, err := my.conn.Query(Q, name.UniqueIdString(), nano_start, nano_end)
	//my.log.Debug("Q: %s, %s %d %d", Q, name.UniqueIdString(), nano_start, nano_end)

	if err != nil {
		my.log.Error("Mysql Driver: Metric select failed, %v", err)
		return nil, err
	}
	defer rows.Close()

	rawd := make(smetrics.DBSeriesList, 0)
	var pType uint8
	var pBytes []byte
	var uid string
	var id, t_start, t_end int64
	for rows.Next() {
		if err := rows.Scan(&id, &uid, &t_start, &t_end, &pType, &pBytes); err != nil {
			return rawd, err
		}
		dataums := &smetrics.DBSeries{
			Id:         id,
			Uid:        uid,
			Start:      t_start,
			End:        t_end,
			Ptype:      pType,
			Pbytes:     pBytes,
			Resolution: resolution,
		}
		rawd = append(rawd, dataums)
	}
	if err := rows.Err(); err != nil {
		return rawd, err
	}

	return rawd, nil
}

// update the row defined in dbs w/ the new bytes from the new Timeseries
func (my *MySQLMetrics) UpdateDBSeries(dbs *smetrics.DBSeries, ts series.TimeSeries) error {

	tName := my.db.RootMetricsTableName()
	Q := fmt.Sprintf(
		"UPDATE %s_%ds SET stime=?, etime=?, ptype=?, points=? WHERE uid=? AND id=?",
		tName, dbs.Resolution,
	)

	ptype := series.IdFromName(ts.Name())

	retryNotify := func(err error, after time.Duration) {
		stats.StatsdClientSlow.Incr("writer.mysql.update.metric-retry", 1)
		my.log.Error("Update failed will retry :: %v : after %v", err, after)
	}

	retryFunc := func() error {
		_, err := my.conn.Exec(Q, ts.StartTime(), ts.LastTime(), ptype, ts.Bytes(), dbs.Uid, dbs.Id)
		return err
	}
	backoff.RetryNotify(retryFunc, backoff.NewExponentialBackOff(), retryNotify)

	return nil
}

// update the row defined in dbs w/ the new bytes from the new Timeseries
func (my *MySQLMetrics) InsertDBSeries(name *repr.StatName, timeseries series.TimeSeries, resolution uint32) (added int, err error) {

	if name == nil {
		return 0, errNameIsNil
	}
	if timeseries == nil {
		return 0, errSeriesIsNil
	}

	defer func() {
		if r := recover(); r != nil {
			my.log.Critical("Mysql Failure (panic) %v ::", r)
			err = fmt.Errorf("%v", r)
		}
	}()

	tName := my.db.RootMetricsTableName()
	Q := fmt.Sprintf(
		"INSERT INTO %s_%ds (uid, path, ptype, points, stime, etime) VALUES ",
		tName, resolution,
	)

	pts := timeseries.Bytes()
	sTime := timeseries.StartTime()
	eTime := timeseries.LastTime()

	Q += "(?,?,?,?,?,?)"
	vals := []interface{}{
		name.UniqueIdString(),
		name.Key,
		series.IdFromName(timeseries.Name()),
		pts,
		sTime,
		eTime,
	}
	retryNotify := func(err error, after time.Duration) {
		stats.StatsdClientSlow.Incr("writer.mysql.insert.metric-retry", 1)
		my.log.Error("Insert failed will retry :: %v : after %v", err, after)
	}

	retryFunc := func() error {
		trans, err := my.conn.Begin()
		if err != nil {
			return err
		}
		defer trans.Commit()
		_, err = trans.Exec(Q, vals...)
		if err != nil {
			my.log.Errorf("Mysql Driver: Metric insert failed, %v", err)
			return err
		}
		return nil
	}
	backoff.RetryNotify(retryFunc, backoff.NewExponentialBackOff(), retryNotify)

	return 1, nil
}
