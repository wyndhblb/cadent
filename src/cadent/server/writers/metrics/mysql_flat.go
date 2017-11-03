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
	THe MySQL flat stat write

     CREATE TABLE `{metrics-table}{resolutionprefix}` (
      `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
      `uid` varchar(50) CHARACTER SET ascii NOT NULL ,
      `path` varchar(255) NOT NULL DEFAULT '',
      `sum` float NOT NULL,
      `min` float NOT NULL,
      `max` float NOT NULL,
      `first` float NOT NULL,
      `last` float NOT NULL,
      `count` float NOT NULL,
      `time` datetime(6) NOT NULL,
      PRIMARY KEY (`id`),
      KEY `uid` (`uid`),
      KEY `path` (`path`),
      KEY `time` (`time`)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8;

	OPTIONS: For `Config`

		table: base table name (default: metrics)
		batch_count: batch this many inserts for much faster insert performance (default 1000)
		periodic_flush: regardless of if batch_count met always flush things at this interval (default 1s)

*/

package metrics

import (
	sindexer "cadent/server/schemas/indexer"
	smetrics "cadent/server/schemas/metrics"
	"cadent/server/schemas/repr"
	"cadent/server/stats"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/dbs"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/net/context"
	logging "gopkg.in/op/go-logging.v1"
	"net/url"
	"sync"
	"time"
)

/****************** Interfaces *********************/
type MySQLFlatMetrics struct {
	WriterBase

	db   *dbs.MySQLDB
	conn *sql.DB

	writeList     []*repr.StatRepr // buffer the writes so as to do "multi" inserts per query
	maxWriteSize  int              // size of that buffer before a flush
	maxIdle       time.Duration    // either maxWriteSize will trigger a write or this time passing will
	writeLock     sync.Mutex
	renderTimeout time.Duration

	log *logging.Logger

	shutdown chan bool
}

func NewMySQLFlatMetrics() *MySQLFlatMetrics {
	my := new(MySQLFlatMetrics)
	my.log = logging.MustGetLogger("writers.mysqlflat")
	return my
}

func (my *MySQLFlatMetrics) Config(conf *options.Options) error {
	my.options = conf

	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("`dsn` (user:pass@tcp(host:port)/db) is needed for mysql config")
	}

	dbKey := dsn + conf.String("table", "metrics")
	db, err := dbs.NewDB("mysql", dbKey, conf)
	if err != nil {
		return err
	}

	res, err := conf.Int64Required("resolution")
	if err != nil {
		return fmt.Errorf("Resolution needed for mysql writer")
	}

	my.name = conf.String("name", "metrics:mysqlflat:"+dsn)

	// FLAT writers have different "writers" for each resolution,  so we need to set different names for them
	// the API readers use the "first one", so if things already exist w/ the name, slap on a resolution tag
	if GetMetrics(my.Name()) != nil {
		my.name = conf.String("name", "metrics:mysqlflat:"+dsn) + fmt.Sprintf(":%v", res)
	}
	err = RegisterMetrics(my.Name(), my)
	if err != nil {
		my.log.Critical(err.Error())
		return err
	}

	my.db = db.(*dbs.MySQLDB)
	my.conn = db.Connection().(*sql.DB)

	//need to hide the usr/pw from things
	p, _ := url.Parse("mysql://" + dsn)
	cacheKey := fmt.Sprintf("mysqlflat:cache:%s/%s:%v", p.Host, conf.String("table", "metrics"), res)
	my.cacher, err = GetCacherSingleton(cacheKey, "single")

	if err != nil {
		return err
	}

	my.maxWriteSize = int(conf.Int64("batch_count", 1000))
	my.maxIdle = conf.Duration("periodic_flush", time.Duration(time.Second))
	_tgs := conf.String("tags", "")
	if len(_tgs) > 0 {
		my.staticTags = repr.SortingTagsFromString(_tgs)
	}

	my.shutitdown = false
	my.shutdown = make(chan bool)

	rdur, err := time.ParseDuration(MYSQL_RENDER_TIMEOUT)
	if err != nil {
		return err
	}
	my.renderTimeout = rdur

	return nil
}

func (my *MySQLFlatMetrics) Driver() string {
	return "mysql-flat"
}

func (my *MySQLFlatMetrics) Stop() {
	my.startstop.Stop(func() {
		if my.shutitdown {
			return
		}
		my.shutitdown = true
		shutdown.AddToShutdown()
		my.shutdown <- true
	})
	return
}

func (my *MySQLFlatMetrics) Start() {
	my.startstop.Start(func() {
		// now we make sure the metrics schemas are added
		err := NewMySQLMetricsSchema(my.conn, my.db.RootMetricsTableName(), my.resolutions, "flat").AddMetricsTable()
		if err != nil {
			panic(err)
		}
		my.cacher.Start()
		go my.periodFlush()
	})
}

func (my *MySQLFlatMetrics) periodFlush() {
	for {
		select {
		case <-my.shutdown:
			my.flush()
			shutdown.ReleaseFromShutdown()
			return
		default:
			time.Sleep(my.maxIdle)
			my.flush()
		}
	}
}

func (my *MySQLFlatMetrics) flush() (int, error) {
	my.writeLock.Lock()
	defer my.writeLock.Unlock()

	l := len(my.writeList)
	if l == 0 {
		return 0, nil
	}

	Q := fmt.Sprintf(
		"INSERT IGNORE INTO %s_%ds (uid, path, sum, min, max, last, count, time) VALUES ",
		my.db.RootMetricsTableName(), my.currentResolution,
	)

	vals := []interface{}{}

	for _, stat := range my.writeList {
		Q += "(?,?,?,?,?,?,?,?), "
		vals = append(
			vals, stat.Name.UniqueIdString(), stat.Name.Key, stat.Sum, stat.Min, stat.Max, stat.Last, stat.Count, stat.ToTime().UnixNano(),
		)
	}

	//trim the last ", "
	Q = Q[0 : len(Q)-2]

	//prepare the statement
	stmt, err := my.conn.Prepare(Q)
	if err != nil {
		my.log.Error("Mysql Driver: Metric prepare failed, %v", err)
		return 0, err
	}
	defer stmt.Close()

	//format all vals at once
	_, err = stmt.Exec(vals...)
	if err != nil {
		my.log.Error("Mysql Driver: Metric insert failed, %v", err)
		return 0, err
	}

	my.writeList = my.writeList[:0]
	return l, nil
}

// Write simple proxy to the
func (my *MySQLFlatMetrics) Write(stat *repr.StatRepr) error {
	return my.WriteWithOffset(stat, nil)
}

// WriteWithOffset for this writer the offset is not relevant
func (my *MySQLFlatMetrics) WriteWithOffset(stat *repr.StatRepr, offset *smetrics.OffsetInSeries) error {
	stat.Name.MergeMetric2Tags(my.staticTags)

	if len(my.writeList) > my.maxWriteSize {
		_, err := my.flush()
		if err != nil {
			return err
		}
	}
	// only the lowest res needs to write the index
	if my.ShouldWriteIndex() {
		my.indexer.Write(*stat.Name) // to the indexer
	}

	// Flush can cause double locking
	my.writeLock.Lock()
	my.writeList = append(my.writeList, stat)
	my.writeLock.Unlock()
	return nil
}

/**** READER ***/

func (my *MySQLFlatMetrics) RawRenderOne(ctx context.Context, metric *sindexer.MetricFindItem, start int64, end int64, resample uint32) (*smetrics.RawRenderItem, error) {
	sp, closer := my.GetSpan("RawRenderOne", ctx)
	sp.LogKV("driver", "MySQLFlatMetrics", "metric", metric.StatName().Key, "uid", metric.UniqueId)
	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("reader.mysqlflat.renderraw.get-time-ns", time.Now())

	rawd := new(smetrics.RawRenderItem)

	if metric.Leaf == 0 { //data only
		return rawd, fmt.Errorf("RawRenderOne: Not a data node")
	}

	//figure out the best res
	resolution := my.GetResolution(start, end)
	out_resolution := resolution

	//obey the bigger
	if resample > resolution {
		out_resolution = resample
	}

	start = TruncateTimeTo(start, int(resolution))
	end = TruncateTimeTo(end, int(resolution))

	b_len := uint32(end-start) / resolution //just to be safe
	if b_len <= 0 {
		return rawd, fmt.Errorf("RawRenderOne: time too narrow")
	}

	// time in cassandra is in NanoSeconds so we need to pad the times from seconds -> nanos
	nano := int64(time.Second)
	nanoEnd := end * nano
	nanoStart := start * nano

	firstT := uint32(start)
	lastT := uint32(end)

	// try the write inflight cache as nothing is written yet
	statName := metric.StatName()
	inflightRenderitem, err := my.cacher.GetAsRawRenderItem(statName)

	// need at LEAST 2 points to get the proper step size
	if inflightRenderitem != nil && err == nil {
		// move the times to the "requested" ones and quantize the list
		inflightRenderitem.Metric = metric.Id
		inflightRenderitem.Tags = metric.Tags
		inflightRenderitem.MetaTags = metric.MetaTags
		inflightRenderitem.Id = metric.UniqueId
		inflightRenderitem.AggFunc = statName.AggType()
		if inflightRenderitem.Start < uint32(start) {
			inflightRenderitem.RealEnd = uint32(end)
			inflightRenderitem.RealStart = uint32(start)
			inflightRenderitem.Start = inflightRenderitem.RealStart
			inflightRenderitem.End = inflightRenderitem.RealEnd
			return inflightRenderitem, err
		}
	}

	// grab ze data. (note data is already sorted by time asc va the cassandra schema)
	Q := fmt.Sprintf("SELECT max, min, sum, last, count, time FROM %s_%ds WHERE uid=? AND time BETWEEN ? AND ? ORDER BY time ASC",
		my.db.RootMetricsTableName(), resolution,
	)

	iter, err := my.conn.Query(Q, metric.UniqueId, nanoStart, nanoEnd)

	if err != nil {
		return rawd, err
	}

	var t, count int64
	var min, max, sum, last float64

	mKey := metric.Id

	tStart := uint32(start)
	curPt := smetrics.NullRawDataPoint(tStart)
	// sorting order for the table is time ASC (i.e. firstT == first entry)
	// on resamples (if >0 ) we simply merge points until we hit the time steps
	do_resample := resample > 0 && resample > resolution

	for iter.Next() {
		err = iter.Scan(&max, &min, &sum, &last, &count, &t)
		if err != nil {
			my.log.Errorf("error reading: %v", err)
			continue
		}
		t := uint32(time.Unix(0, t).Unix())
		if do_resample {
			if t >= tStart+resample {
				tStart += resample
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
		my.log.Error("RawRender: Failure closing iterator: %v", err)
	}

	if len(rawd.Data) > 0 && rawd.Data[0].Time > 0 {
		firstT = rawd.Data[0].Time
	}

	//cass.log.Critical("METR: %s Start: %d END: %d LEN: %d GotLen: %d", metric.Id, firstT, lastT, len(d_points), ct)

	rawd.RealEnd = uint32(lastT)
	rawd.RealStart = uint32(firstT)
	rawd.Start = uint32(start)
	rawd.End = uint32(end)
	rawd.Step = out_resolution
	rawd.Metric = mKey
	rawd.Tags = metric.Tags
	rawd.MetaTags = metric.MetaTags
	rawd.Id = metric.UniqueId
	rawd.AggFunc = statName.AggType()

	// grab the "current inflight" from the cache and merge into the main array
	if inflightRenderitem != nil && len(inflightRenderitem.Data) > 1 {
		//merge with any inflight bits (inflight has higher precedence over the file)
		inflightRenderitem.MergeWithResample(rawd, out_resolution)
		return inflightRenderitem, nil
	}

	return rawd, nil
}

func (my *MySQLFlatMetrics) RawRender(ctx context.Context, path string, from int64, to int64, tags repr.SortingTags, resample uint32) ([]*smetrics.RawRenderItem, error) {
	sp, closer := my.GetSpan("RawRender", ctx)
	sp.LogKV("driver", "MySQLFlatMetrics", "path", path, "from", from, "to", to)
	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("reader.mysqlflat.rawrender.get-time-ns", time.Now())

	paths := SplitNamesByComma(path)
	var metrics []*sindexer.MetricFindItem

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
		_ri, err := my.RawRenderOne(ctx, met, from, to, resample)

		if err != nil {
			stats.StatsdClientSlow.Incr("reader.mysqlflat.rawrender.errors", 1)
			my.log.Error("Read Error for %s (%d->%d) : %v", path, from, to, err)
		}
		return _ri
	}

	// ye old fan out technique but not "too many" as to kill the server
	jobWorker := func(jober int, taskqueue <-chan *sindexer.MetricFindItem, resultqueue chan<- *smetrics.RawRenderItem) {
		resultsChan := make(chan *smetrics.RawRenderItem, 1)
		for met := range taskqueue {
			go func() { resultsChan <- renderOne(met) }()
			select {
			case <-time.After(my.renderTimeout):
				stats.StatsdClientSlow.Incr("reader.mysqlflat.rawrender.timeouts", 1)
				my.log.Errorf("Render Timeout for %s (%d->%d)", path, from, to)
				resultqueue <- nil
			case res := <-resultsChan:
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
	stats.StatsdClientSlow.Incr("reader.mysqlflat.rawrender.metrics-per-request", int64(len(metrics)))

	return rawd, nil
}

func (my *MySQLFlatMetrics) CacheRender(ctx context.Context, path string, from int64, to int64, tags repr.SortingTags) ([]*smetrics.RawRenderItem, error) {
	return nil, ErrWillNotBeimplemented
}
func (my *MySQLFlatMetrics) CachedSeries(path string, from int64, to int64, tags repr.SortingTags) (*smetrics.TotalTimeSeries, error) {
	return nil, ErrWillNotBeimplemented
}
