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
	The ElasticSearch flat stat write


	OPTIONS: For `Config`

		metric_index="metrics" #  base table name (default: metrics)
		batch_count=1000 # batch this many inserts for much faster insert performance (default 1000)
		periodic_flush="1s" # regardless of if batch_count met always flush things at this interval (default 1s)

		## create a new index based on the time of the point
		# of the form "{metrics}_{resolution}s-{YYYY-MM-DD}"
		# default is "none"
		index_per_date = "none|week|month|day"


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
	"encoding/json"
	"fmt"
	"golang.org/x/net/context"
	es5 "gopkg.in/olivere/elastic.v5"
	logging "gopkg.in/op/go-logging.v1"
	"io"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	ELASTIC_DEFAULT_METRIC_RENDER_WORKERS = 4
	ELASTIC_RENDER_TIMEOUT                = "5s"
	ELASTIC_FLUSH_TIME_SECONDS            = 5
	ELASTIC_DEFAULT_PAGE_SIZE             = 4096
	ELASTIC_DEFAULT_MAX_TASKS             = 100
)

// this is mimic the BulkInsert interface and save lots of GC things
// elastic.BulkableRequest interface
type elasticBulkInsert struct {
	fmt.Stringer
	str []string
}

func (e elasticBulkInsert) Source() ([]string, error) {
	return e.str, nil
}

var esBytesPool sync.Pool

// pools of preallocated byte arrays
func esGetBytes() []byte {
	x := esBytesPool.Get()
	if x == nil {
		buf := make([]byte, 512)
		return buf[:0]
	}
	buf := x.([]byte)
	return buf[:0]
}

func esPutBytes(buf []byte) {
	esBytesPool.Put(buf)
}

// ESmetric pool

var esMetricPool sync.Pool

func getESMetric() *ESMetric {
	x := esMetricPool.Get()
	if x == nil {
		return new(ESMetric)
	}
	return x.(*ESMetric)
}

func putESMetric(spl *ESMetric) {
	esMetricPool.Put(spl)
}

/****************** Interfaces *********************/
type ElasticSearchFlatMetrics struct {
	WriterBase

	db   *dbs.ElasticSearch
	conn *es5.Client

	writeList []*repr.StatRepr // buffer the writes so as to do "multi" inserts per query
	inBulkLen int32

	maxWriteSize   int32         // size of that buffer before a flush
	maxIdle        time.Duration // either maxWriteSize will trigger a write or this time passing will
	periodicTick   *time.Ticker
	metricAddQueue chan *repr.StatRepr // can provide some back pressure on flush writing if it takes too long
	writeLock      sync.Mutex
	renderTimeout  time.Duration

	indexPerDate string
	maxTasks     int
	bulkRequest  *es5.BulkService

	log *logging.Logger

	shutdown *broadcast.Broadcaster
}

func NewElasticSearchFlatMetrics() *ElasticSearchFlatMetrics {
	es := new(ElasticSearchFlatMetrics)
	es.log = logging.MustGetLogger("writers.elasticflat")
	es.shutdown = broadcast.New(1)
	return es
}

func (es *ElasticSearchFlatMetrics) Config(conf *options.Options) error {
	es.options = conf

	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("`dsn` host:9200/index_name is needed for elasticsearch config")
	}

	dbKey := dsn + conf.String("metric_index", "metrics")

	db, err := dbs.NewDB("elasticsearch", dbKey, conf)
	if err != nil {
		return err
	}

	es.db = db.(*dbs.ElasticSearch)
	es.conn = es.db.Client

	res, err := conf.Int64Required("resolution")
	if err != nil {
		return fmt.Errorf("Resolution needed for elasticsearch writer")
	}

	es.name = conf.String("name", "metrics:elasticsearch:"+dsn)

	// FLAT writers have different "writers" for each resolution,  so we need to set different names for them
	// the API readers use the "first one", so if things already exist w/ the name, slap on a resolution tag
	if GetMetrics(es.Name()) != nil {
		es.name = conf.String("name", "metrics:elasticsearch:"+dsn) + fmt.Sprintf(":%v", res)
	}
	err = RegisterMetrics(es.Name(), es)
	if err != nil {
		es.log.Critical(err.Error())
		return err
	}

	//need to hide the usr/pw from things
	p, _ := url.Parse(dsn)
	cacheKey := fmt.Sprintf("elasticflat:cache:%s/%s:%v", p.Host, conf.String("table", "metrics"), res)
	es.cacher, err = GetCacherSingleton(cacheKey, "single")

	if err != nil {
		return err
	}

	es.maxWriteSize = int32(conf.Int64("batch_count", 5000))
	es.maxIdle = conf.Duration("periodic_flush", time.Duration(ELASTIC_FLUSH_TIME_SECONDS*time.Second))
	_tgs := conf.String("tags", "")
	if len(_tgs) > 0 {
		es.staticTags = repr.SortingTagsFromString(_tgs)
	}

	rdur, err := time.ParseDuration(ELASTIC_RENDER_TIMEOUT)
	if err != nil {
		return err
	}
	es.renderTimeout = rdur

	// hourly, daily, weekly, monthly, none
	es.indexPerDate = conf.String("index_by_date", "daily")

	// pause if there are "too many" pending
	es.maxTasks = int(conf.Int64("max_pending_tasks", ELASTIC_DEFAULT_MAX_TASKS))

	es.metricAddQueue = make(chan *repr.StatRepr, es.maxWriteSize)

	es.periodicTick = time.NewTicker(es.maxIdle)
	es.bulkRequest = es.conn.Bulk().Refresh("false")

	return nil
}

func (es *ElasticSearchFlatMetrics) Driver() string {
	return "elasticsearch-flat"
}

func (es *ElasticSearchFlatMetrics) Stop() {
	es.startstop.Stop(func() {
		shutdown.AddToShutdown()
		es.shutdown.Close()
	})
	return
}

func (es *ElasticSearchFlatMetrics) Start() {
	es.startstop.Start(func() {
		// now we make sure the metrics schemas are added
		err := NewElasticMetricsSchema(es.conn, es.db.Tablename(), es.resolutions, "flat").AddMetricsTable()
		if err != nil {
			panic(err)
		}
		//es.cacher.Start() // no cache here
		go es.periodFlush()
		go es.pushStat()
	})
}

func (es *ElasticSearchFlatMetrics) periodFlush() {
	shuts := es.shutdown.Listen()

	for {
		select {
		case <-es.periodicTick.C:
			es.flush()
		case <-shuts.Ch:
			es.log.Notice("Got shutdown, doing final flush for %s...", es.Name())
			//es.cacher.Stop() // no cache here
			es.flush()
			es.log.Notice("Done final flush for %s", es.Name())
			shutdown.ReleaseFromShutdown()
			es.log.Notice("Done shutdown for %s", es.Name())
			return
		}
	}
}

// index name based on the incoming time
func (es *ElasticSearchFlatMetrics) indexName(res uint32, inTime time.Time) string {
	dstr := ""
	switch es.indexPerDate {
	case "hourly":
		dstr = "_" + inTime.Format("2006.01.02.15")
	case "daily":
		dstr = "_" + inTime.Format("2006.01.02")
	case "weekly":
		ynum, wnum := inTime.ISOWeek()
		dstr = fmt.Sprintf("_%d.%d", ynum, wnum)
	case "monthly":
		dstr = "_" + inTime.Format("2006.01")
	}
	return fmt.Sprintf("%s_%ds%s", es.db.Tablename(), res, dstr)
}

// from the start times get the list of indexes we need to query
func (es *ElasticSearchFlatMetrics) indexRange(res uint32, sTime time.Time, eTime time.Time) []string {

	outStr := []string{}
	onT := sTime
	switch es.indexPerDate {
	case "hourly":
		eTime = eTime.Add(time.Hour) // need to include the end
		for onT.Before(eTime) {
			outStr = append(outStr, fmt.Sprintf("%s_%ds_%s", es.db.Tablename(), res, onT.Format("2006.01.02.15")))
			onT = onT.Add(time.Hour)
		}
		return outStr
	case "daily":
		eTime = eTime.AddDate(0, 0, 1) // need to include the end
		for onT.Before(eTime) {
			outStr = append(outStr, fmt.Sprintf("%s_%ds_%s", es.db.Tablename(), res, onT.Format("2006.01.02")))
			onT = onT.AddDate(0, 0, 1)
		}
		return outStr
	case "weekly":
		eTime = eTime.AddDate(0, 0, 7) // need to include the end
		for onT.Before(eTime) {
			ynum, wnum := onT.ISOWeek()
			outStr = append(outStr, fmt.Sprintf("%s_%ds_%d.%d", es.db.Tablename(), res, ynum, wnum))
			onT = onT.AddDate(0, 0, 7)
		}
		return outStr
	case "monthly":
		eTime = eTime.AddDate(0, 1, 0) // need to include the end
		for onT.Before(eTime) {
			outStr = append(outStr, fmt.Sprintf("%s_%ds_%s", es.db.Tablename(), res, onT.Format("2006.01")))
			onT = onT.AddDate(0, 1, 0)
		}
		return outStr
	}
	return []string{fmt.Sprintf("%s_%ds", es.db.Tablename(), res)}
}

// we do the raw string things as the GC pressure for this is huge
func (es *ElasticSearchFlatMetrics) addToBulk(stat *repr.StatRepr) {

	uid := stat.Name.UniqueIdString()
	sTime := stat.ToTime()

	//{"index":{"_id":"27az2f4erv6to-1483585630000000000","_index":"metrics_flat_5s_2017-01-04-19","_type":"metric"}}
	tpl := esGetBytes()
	defer esPutBytes(tpl)
	tpl = append(tpl, []byte(`{"index":{"_id":"`)...)

	//id := fmt.Sprintf("%s-%d", uid, stat.Time)
	tpl = append(tpl, []byte(uid)...)
	tpl = append(tpl, '-')
	tpl = strconv.AppendInt(tpl, stat.Time, 10)
	tpl = append(tpl, '"')

	tpl = append(tpl, []byte(`,"_index":`)...)
	tpl = strconv.AppendQuote(tpl, es.indexName(stat.Name.Resolution, sTime))
	tpl = append(tpl, []byte(`,"_type":`)...)
	tpl = strconv.AppendQuote(tpl, es.db.MetricType)
	tpl = append(tpl, []byte("}}\n")...)

	if stat.Count == 1 {

		/*{"uid":"hymascd53qqj","path":"stats.test.house.mean","time":"2017-01-04T18:22:40-08:00","sum":59020,"count":1}*/

		tpl = append(tpl, []byte(`{"uid":`)...)
		tpl = strconv.AppendQuote(tpl, uid)
		tpl = append(tpl, []byte(`,"path":`)...)
		tpl = strconv.AppendQuote(tpl, stat.Name.Key)
		tpl = append(tpl, []byte(`,"time":`)...)
		tpl = strconv.AppendQuote(tpl, sTime.Format("2006-01-02T15:04:05-07:00"))
		tpl = append(tpl, []byte(`,"sum":`)...)
		tpl = strconv.AppendFloat(tpl, stat.Sum, 'f', -1, 64)
		tpl = append(tpl, []byte(`,"count":`)...)
		tpl = strconv.AppendInt(tpl, stat.Count, 10)

	} else {
		//{"uid":"1zf7w7s68c9a2","path":"stats.test.cow.badline.test.house","time":"2017-01-04T18:58:15-08:00","min":105241,"max":109324,"sum":322713,"last":105241,"count":3}
		tpl = append(tpl, []byte(`{"uid":`)...)
		tpl = strconv.AppendQuote(tpl, uid)
		tpl = append(tpl, []byte(`,"path":`)...)
		tpl = strconv.AppendQuote(tpl, stat.Name.Key)
		tpl = append(tpl, []byte(`,"time":`)...)
		tpl = strconv.AppendQuote(tpl, sTime.Format("2006-01-02T15:04:05-07:00"))
		tpl = append(tpl, []byte(`,"min":`)...)
		tpl = strconv.AppendFloat(tpl, stat.Min, 'f', -1, 64)
		tpl = append(tpl, []byte(`,"max":`)...)
		tpl = strconv.AppendFloat(tpl, stat.Max, 'f', -1, 64)
		tpl = append(tpl, []byte(`,"sum":`)...)
		tpl = strconv.AppendFloat(tpl, stat.Sum, 'f', -1, 64)
		tpl = append(tpl, []byte(`,"last":`)...)
		tpl = strconv.AppendFloat(tpl, stat.Last, 'f', -1, 64)
		tpl = append(tpl, []byte(`,"count":`)...)
		tpl = strconv.AppendInt(tpl, stat.Count, 10)

		/*
			esM := new(ESMetric)
			esM.Min = stat.Min
			esM.Max = stat.Max
			esM.Sum = stat.Sum
			esM.Last = stat.Last
			esM.Count = stat.Count
			esM.Path = stat.Name.Key
			esM.Uid = uid
			// time we need to convert to millis as that's the biggest "date" ES will support
			esM.Time = stat.ToTime()

			for _, t := range stat.Name.Tags {
				esM.Tags = append(esM.Tags, indexer.ESTag{
					Name:   t.Name,
					Value:  t.Value,
					IsMeta: false,
				})
			}
			for _, t := range stat.Name.MetaTags {
				esM.Tags = append(esM.Tags, indexer.ESTag{
					Name:   t.Name,
					Value:  t.Value,
					IsMeta: true,
				})
			}
			es.bulkRequest.Add(es5.NewBulkIndexRequest().
				Index(es.indexName(stat.Name.Resolution, esM.Time)).
				Type(es.db.MetricType).
				Id(id).
				Doc(*esM))
		*/

	}
	haveTags := !stat.Name.Tags.IsEmpty() || !stat.Name.MetaTags.IsEmpty()
	if haveTags {
		tpl = append(tpl, []byte(`"{"tags":[`)...)
		if !stat.Name.Tags.IsEmpty() {
			for i, t := range stat.Name.Tags {
				if i > 0 {
					tpl = append(tpl, ',')
				}
				tpl = append(tpl, []byte(`{"name":`)...)
				tpl = strconv.AppendQuote(tpl, t.Name)
				tpl = append(tpl, []byte(`,"value":`)...)
				tpl = strconv.AppendQuote(tpl, t.Value)
				tpl = append(tpl, []byte(`,"is_meta":false}`)...)
			}
		}
		if !stat.Name.MetaTags.IsEmpty() {
			for i, t := range stat.Name.Tags {
				if i > 0 {
					tpl = append(tpl, ',')
				}
				tpl = append(tpl, []byte(`{"name":`)...)
				tpl = strconv.AppendQuote(tpl, t.Name)
				tpl = append(tpl, []byte(`,"value":`)...)
				tpl = strconv.AppendQuote(tpl, t.Value)
				tpl = append(tpl, []byte(`,"is_meta":true}`)...)
			}
		}
		tpl = append(tpl, []byte(`"]}`)...)
	}
	// end data
	tpl = append(tpl, '}')

	es.bulkRequest.Add(elasticBulkInsert{str: []string{string(tpl)}})

	atomic.AddInt32(&es.inBulkLen, 1)

}

func (es *ElasticSearchFlatMetrics) flush() (int, error) {
	es.writeLock.Lock()
	defer es.writeLock.Unlock()

	defer func() {
		// reset
		es.bulkRequest = es.conn.Bulk().Refresh("false")
		atomic.StoreInt32(&es.inBulkLen, 0)
	}()

	l := atomic.LoadInt32(&es.inBulkLen)
	if l == 0 {
		return 0, nil
	}

	tsks, err := es.conn.TasksList().Do(context.Background())
	if err != nil {
		es.log.Errorf("Failed to get task list: %v", err)
		return 0, nil
	}

	tLen := 0
	for _, g := range tsks.Nodes {
		tLen += len(g.Tasks)
	}
	if tLen > es.maxTasks {
		es.log.Warningf("Task list is too large (%d), taking a nap", tLen)
		time.Sleep(5 * time.Second)
	}

	es.log.Debug("Writing %d metrics to ElasticSearch", l)
	tStart := time.Now()

	delta := time.Now().Sub(tStart).Seconds()
	es.log.Debug("Wrote Bulk %d metrics to ElasticSearch in %vs", l, delta)
	if delta > 1.0 {
		es.log.Warning("Bulk insert took longer then 1 second (%v).  Elastic is falling behind.", delta)
		stats.StatsdClientSlow.Incr("writer.elasticflat.bulk-timeout", 1)
	}

	erred := make([]*repr.StatRepr, 0)

	gots, err := es.bulkRequest.Do(context.Background())

	if err != nil {
		es.log.Error("Could not insert metrics %v", err)
		stats.StatsdClientSlow.Incr("writer.elasticflat.bulk-failures", 1)
		return 0, nil
	} else {
		stats.StatsdClientSlow.Incr("writer.elasticflat.bulk-writes", 1)
	}

	// need to check for those that "failed" this is mostly caused by under provisioned ES clusters
	// due to lack of queue/threads
	isOk := 0

	for _, b := range gots.Items {
		for _, bb := range b {
			if bb.Error != nil {
				stats.StatsdClientSlow.Incr("writer.elasticflat.insert-one-failures", 1)
				es.log.Error("Could not insert metrics %v id: (%v) ... putting it back into the queue", bb.Error, bb.Id)
			} else {
				isOk++
			}
		}
	}

	for _, s := range erred {
		es.addToBulk(s)
	}
	return isOk, nil
}

// provides some back pressure on Write in case ES gets really slow
func (es *ElasticSearchFlatMetrics) pushStat() {
	shuts := es.shutdown.Listen()
	for {
		select {
		case stat := <-es.metricAddQueue:
			l := atomic.LoadInt32(&es.inBulkLen)
			if l > es.maxWriteSize {
				es.flush()
			}
			// only the lowest res needs to write the index
			if es.ShouldWriteIndex() {
				es.indexer.Write(*stat.Name) // to the indexer
			}
			s := stat
			es.addToBulk(s)
		case <-shuts.Ch:
			return
		}
	}
}

// WriteWithOffset .. for this metrics write is
func (es *ElasticSearchFlatMetrics) WriteWithOffset(stat *repr.StatRepr, offset *smetrics.OffsetInSeries) error {
	stat.Name.MergeMetric2Tags(es.staticTags)
	es.metricAddQueue <- stat
	return nil
}

func (es *ElasticSearchFlatMetrics) Write(stat *repr.StatRepr) error {
	return es.WriteWithOffset(stat, nil)
}

/**** READER ***/

func (es *ElasticSearchFlatMetrics) RawRenderOne(ctx context.Context, metric *sindexer.MetricFindItem, start int64, end int64, resample uint32) (*smetrics.RawRenderItem, error) {
	sp, closer := es.GetSpan("RawRenderOne", ctx)
	sp.LogKV("driver", "ElasticSearchFlatMetrics", "metric", metric.StatName().Key, "uid", metric.UniqueId)
	defer closer()
	defer stats.StatsdSlowNanoTimeFunc("reader.elasticflat.renderraw.get-time-ns", time.Now())

	rawd := new(smetrics.RawRenderItem)

	if metric.Leaf == 0 { //data only
		return rawd, fmt.Errorf("RawRenderOne: Not a data node")
	}

	//figure out the best res
	resolution := es.GetResolution(start, end)
	outResolution := resolution

	//obey the bigger
	if resample > resolution {
		outResolution = resample
	}

	start = TruncateTimeTo(start, int(resolution))
	end = TruncateTimeTo(end, int(resolution))

	b_len := uint32(end-start) / resolution //just to be safe
	if b_len <= 0 {
		return rawd, fmt.Errorf("time too narrow")
	}

	// time in ES is max millis BUT only 13 chars for searches
	// so it's not really  "millis" but 1000ths
	milli := int64(1000)
	qtEnd := end * milli
	qtStart := start * milli

	firstT := uint32(start)
	lastT := uint32(end)

	// try the write inflight cache as nothing is written yet
	stat_name := metric.StatName()
	inflightRenderitem, err := es.cacher.GetAsRawRenderItem(stat_name)

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
	useIndexes := es.indexRange(resolution, onTime, endTime)

	andFilter := es5.NewBoolQuery()
	andFilter = andFilter.Must(es5.NewTermQuery("uid", metric.UniqueId))
	andFilter = andFilter.Must(es5.NewRangeQuery("time").From(qtStart).To(qtEnd).Format("epoch_millis"))

	// scroller
	scroll := es.conn.Scroll().
		Index(useIndexes...).
		IgnoreUnavailable(true).
		Type(es.db.MetricType).
		Query(andFilter).
		Size(ELASTIC_DEFAULT_PAGE_SIZE).
		Sort("time", true)

	// dump query if debug mode
	if es.log.IsEnabledFor(logging.DEBUG) {
		ss, _ := es5.NewSearchSource().Query(andFilter).Sort("time", true).Source()
		data, _ := json.Marshal(ss)
		es.log.Debug("Query: Index %s: %v (QUERY :: %s)", useIndexes, err, data)
	}

	if err != nil {
		ss, _ := es5.NewSearchSource().Query(andFilter).Sort("time", true).Source()
		data, _ := json.Marshal(ss)
		es.log.Error("Query failed: Index %s: %v (QUERY :: %s)", useIndexes, err, data)
		return rawd, err
	}

	mKey := metric.Id

	tStart := uint32(start)
	curPt := smetrics.NullRawDataPoint(tStart)
	onPage := 1
	for {
		results, err := scroll.Do(context.Background())
		if err == io.EOF {
			break // all results retrieved
		}
		if err != nil {
			es.log.Error("Scroll Query failed: Index %s: %v", useIndexes, err)
			break
		}

		for _, h := range results.Hits.Hits {
			// just grab the "n+1" length ones
			item := getESMetric()
			defer putESMetric(item)

			err := json.Unmarshal(*h.Source, item)
			if err != nil {
				es.log.Error("Elastic Driver: json error, %v", err)
				continue
			}
			t := uint32(item.Time.Unix())
			if doResample {
				if t >= tStart+resample {
					tStart += resample
					rawd.Data = append(rawd.Data, curPt)
					curPt = &smetrics.RawDataPoint{
						Count: item.Count,
						Sum:   item.Sum,
						Max:   item.Max,
						Min:   item.Min,
						Last:  item.Last,
						Time:  t,
					}
				} else {
					curPt.Merge(&smetrics.RawDataPoint{
						Count: item.Count,
						Sum:   item.Sum,
						Max:   item.Max,
						Min:   item.Min,
						Last:  item.Last,
						Time:  t,
					})
				}
			} else {
				rawd.Data = append(rawd.Data, &smetrics.RawDataPoint{
					Count: item.Count,
					Sum:   item.Sum,
					Max:   item.Max,
					Min:   item.Min,
					Last:  item.Last,
					Time:  t,
				})
			}
			lastT = t
		}
		onPage += 1
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

func (es *ElasticSearchFlatMetrics) RawRender(ctx context.Context, path string, from int64, to int64, tags repr.SortingTags, resample uint32) ([]*smetrics.RawRenderItem, error) {
	sp, closer := es.GetSpan("RawRender", ctx)
	sp.LogKV("driver", "ElasticSearchFlatMetrics", "path", path, "from", from, "to", to)
	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("reader.elasticflat.rawrender.get-time-ns", time.Now())

	paths := SplitNamesByComma(path)
	var metrics []*sindexer.MetricFindItem

	for _, pth := range paths {
		mets, err := es.indexer.Find(ctx, pth, tags)
		if err != nil {
			continue
		}
		metrics = append(metrics, mets...)
	}

	rawd := make([]*smetrics.RawRenderItem, 0, len(metrics))

	procs := ELASTIC_DEFAULT_METRIC_RENDER_WORKERS

	jobs := make(chan *sindexer.MetricFindItem, len(metrics))
	results := make(chan *smetrics.RawRenderItem, len(metrics))

	renderOne := func(met *sindexer.MetricFindItem) *smetrics.RawRenderItem {
		_ri, err := es.RawRenderOne(ctx, met, from, to, resample)

		if err != nil {
			stats.StatsdClientSlow.Incr("reader.elasticflat.rawrender.errors", 1)
			es.log.Error("Read Error for %s (%d->%d) : %v", path, from, to, err)
		}
		return _ri
	}

	// ye old fan out technique but not "too many" as to kill the server
	jobWorker := func(jober int, taskqueue <-chan *sindexer.MetricFindItem, resultqueue chan<- *smetrics.RawRenderItem) {
		resultsChan := make(chan *smetrics.RawRenderItem, 1)
		for met := range taskqueue {
			go func() { resultsChan <- renderOne(met) }()
			select {
			case <-time.After(es.renderTimeout):
				stats.StatsdClientSlow.Incr("reader.elasticflat.rawrender.timeouts", 1)
				es.log.Errorf("Render Timeout for %s (%d->%d)", path, from, to)
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
	stats.StatsdClientSlow.Incr("reader.elasticflat.rawrender.metrics-per-request", int64(len(metrics)))

	return rawd, nil
}

func (es *ElasticSearchFlatMetrics) CacheRender(ctx context.Context, path string, from int64, to int64, tags repr.SortingTags) ([]*smetrics.RawRenderItem, error) {
	return nil, ErrWillNotBeimplemented
}
func (es *ElasticSearchFlatMetrics) CachedSeries(path string, from int64, to int64, tags repr.SortingTags) (*smetrics.TotalTimeSeries, error) {
	return nil, ErrWillNotBeimplemented
}
