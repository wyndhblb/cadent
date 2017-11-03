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
	THe ElasticSearch indexer.


	A simple mapping to uids and tags/paths

	{
		uid: [string]
		path: [string]
		tags:[ {name:[string], value: [string], is_meta:[bool]},...]
	}

*/

package indexer

import (
	"cadent/server/broadcast"
	"cadent/server/dispatch"
	"cadent/server/schemas/indexer"
	"cadent/server/schemas/repr"
	"cadent/server/stats"
	"cadent/server/utils"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/dbs"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/net/context"
	es5 "gopkg.in/olivere/elastic.v5"
	logging "gopkg.in/op/go-logging.v1"
	"io"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// how big the backlog can get for writes
	ES_INDEXER_QUEUE_LEN = 1024 * 1024
	// number of parallel writers
	ES_INDEXER_WORKERS = 8
	// writers per second to the index
	ES_WRITES_PER_SECOND = 200
	// default max number of results that we give back
	ES_INDEXER_MAX_RESULTS = 1024
	// default batch size
	ES_DEFAULT_BATCH_SIZE = 1000
	// force a write even if not at batch size
	ES_DEFAULT_FLUSH_TIME = 5 * time.Second
)

// ESPath pool for less ram pressure under load

var esPathPool sync.Pool

func getESPath() *ESPath {
	x := esPathPool.Get()
	if x == nil {
		return new(ESPath)
	}
	return x.(*ESPath)
}

func putESPath(spl *ESPath) {
	esPathPool.Put(spl)
}

/****************** Interfaces *********************/
type ElasticIndexer struct {
	RamIndexer

	db        *dbs.ElasticSearch
	conn      *es5.Client
	indexerId string

	writeLock sync.Mutex

	numWorkers int
	queueLen   int

	dispatcher *dispatch.DispatchQueue

	writesPerSecond int // rate limit writer
	maxResults      int
	updateQueue     chan repr.StatName // update queue for last seen

	shutitdown int32
	shutdown   *broadcast.Broadcaster
	startstop  utils.StartStop

	// index of the writer for a multi node entity
	writerIndex int
	shouldWrite bool

	indexCache *IndexReadCache
	// tag ID cache so we don't do a bunch of unnecessary inserts
	tagIdCache *TagCache

	// elastic bulk inserters
	bulkQuery      *es5.BulkService
	numInBulk      int32
	maxInBulk      int32
	forceWriteTime time.Duration

	log *logging.Logger
}

func NewElasticIndexer() *ElasticIndexer {
	es := new(ElasticIndexer)
	es.log = logging.MustGetLogger("indexer.elastic")
	es.indexCache = NewIndexCache(10000)
	es.tagIdCache = NewTagCache()
	es.updateQueue = make(chan repr.StatName, 4)
	es.RamIndexer = *NewRamIndexer()
	es.shutdown = broadcast.New(1)
	atomic.StoreInt32(&es.shutitdown, 0)
	return es
}

func (es *ElasticIndexer) Config(conf *options.Options) (err error) {
	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("`dsn` host:9200/index_name is needed for elasticsearch config")
	}

	// url parse so that the password is not in the name
	parsed, _ := url.Parse(dsn)
	host := parsed.Host + parsed.Path

	i, err := conf.Int64Required("writer_index")
	if err != nil {
		return err
	}
	es.writerIndex = int(i)

	es.indexerId = conf.String("name", "indexer:elastic:"+host)
	es.log.Noticef("Registering indexer: %s", es.Name())

	// config the Ram Cacher
	err = es.RamIndexer.ConfigRam(conf)
	if err != nil {
		es.log.Critical(err.Error())
		return err
	}

	// reg ourselves before try to get conns
	err = RegisterIndexer(es.Name(), es)
	if err != nil {
		return err
	}

	db, err := dbs.NewDB("elasticsearch", dsn, conf)
	if err != nil {
		return err
	}

	es.db = db.(*dbs.ElasticSearch)
	es.conn = es.db.Client

	// tweak queues and worker sizes
	es.numWorkers = int(conf.Int64("write_workers", ES_INDEXER_WORKERS))
	es.queueLen = int(conf.Int64("write_queue_length", ES_INDEXER_QUEUE_LEN))

	es.cache, err = getCacherSingleton(es.indexerId)
	if err != nil {
		return err
	}
	es.maxResults = int(conf.Int64("max_results", ES_INDEXER_MAX_RESULTS))
	es.writesPerSecond = int(conf.Int64("writes_per_second", ES_WRITES_PER_SECOND))
	es.cache.maxKeys = int(conf.Int64("cache_index_size", CACHER_METRICS_KEYS))

	es.maxInBulk = int32(conf.Int64("batch_count", ES_DEFAULT_BATCH_SIZE))
	es.bulkQuery = es.conn.Bulk()
	es.forceWriteTime = conf.Duration("periodic_flush", ES_DEFAULT_FLUSH_TIME)
	es.shouldWrite = conf.Bool("should_write", true)

	return nil

}
func (es *ElasticIndexer) Name() string { return es.indexerId }

// ShouldWrite based on the `should_write` option, if false, just populate the ram caches
func (es *ElasticIndexer) ShouldWrite() bool { return es.shouldWrite }

func (es *ElasticIndexer) Start() {
	es.startstop.Start(func() {
		es.log.Notice("starting elastic indexer: %s/%s", es.db.PathTable(), es.db.TagTable())
		err := NewElasticSearchSchema(es.conn, es.db.SegmentTable(), es.db.PathTable(), es.db.TagTable()).AddIndexTables()
		if err != nil {
			panic(err)
		}

		es.RamIndexer.Start()

		retries := 2
		es.dispatcher = dispatch.NewDispatchQueue(es.numWorkers, es.queueLen, retries)
		es.dispatcher.Start()

		go es.periodicFlush()

		go es.sendToWriters() // the dispatcher
		// several consumers
		for i := 0; i < 4; i++ {
			go es.runUpdateQueue()
		}
	})
}

func (es *ElasticIndexer) Stop() {
	es.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()

		es.shutdown.Close()
		atomic.StoreInt32(&es.shutitdown, 1)

		es.log.Notice("Shutting down elasticsearh indexer: %s/%s", es.db.PathTable(), es.db.TagTable())
		es.RamIndexer.Stop()
		es.log.Notice("Finished Shutdown elasticsear indexer: %s/%s", es.db.PathTable(), es.db.TagTable())
	})
}

// this fills up the tag cache on startup
func (es *ElasticIndexer) fillTagCache() {
	//ToDo

}

// pop from the cache and send to actual writers
func (es *ElasticIndexer) sendToWriters() error {
	// this may not be the "greatest" ratelimiter of all time,
	// as "high frequency tickers" can be costly .. but should the workers get backedup
	// it will block on the write_queue stage

	if es.writesPerSecond <= 0 {
		es.log.Notice("Starting indexer writer: No rate limiting enabled")
		for {
			if atomic.LoadInt32(&es.shutitdown) == 1 {
				return nil
			}
			skey := es.cache.Pop()
			switch skey.IsBlank() {
			case true:
				time.Sleep(time.Second)
			default:
				stats.StatsdClient.Incr(fmt.Sprintf("indexer.elastic.write.send-to-writers"), 1)
				es.dispatcher.Add(&elasticIndexerJob{ES: es, Stat: skey})
			}
		}
	} else {
		sleep_t := float64(time.Second) * (time.Second.Seconds() / float64(es.writesPerSecond))
		es.log.Notice("Starting indexer writer: limiter every %f nanoseconds (%d writes per second)", sleep_t, es.writesPerSecond)
		dur := time.Duration(int(sleep_t))
		for {
			if atomic.LoadInt32(&es.shutitdown) == 1 {
				return nil
			}
			skey := es.cache.Pop()
			switch skey.IsBlank() {
			case true:
				time.Sleep(time.Second)
			default:
				stats.StatsdClient.Incr(fmt.Sprintf("indexer.elastic.write.send-to-writers"), 1)
				es.dispatcher.Add(&elasticIndexerJob{ES: es, Stat: skey})
				time.Sleep(dur)
			}
		}
	}
}

func (es *ElasticIndexer) runUpdateQueue() {

	shuts := es.shutdown.Listen()
	for {
		select {
		case name := <-es.updateQueue:
			es.writeUpdate(name)
		case <-shuts.Ch:
			shuts.Close()
			return
		}
	}
}

// this writes an "update" to the id table, we don't do this every write, just periodically
// otherwise we'd kill elastic w/ a bunch of silly updates
func (es *ElasticIndexer) writeUpdate(inname repr.StatName) error {

	if es.RamIndexer.NeedsWrite(inname, CASSANDRA_HIT_LAST_UPDATE) {
		has := es.RamIndexer.KeyExists(inname.Key)
		// this seems to have some bad performance
		es.RamIndexer.Write(inname) //add to ram index
		if !has {
			return nil
		}
	} else {
		return nil
	}

	// quick bail
	if !es.ShouldWrite() {
		return nil
	}

	// only if we need to update it
	//slap in the id -> path table
	tg := new(ESPathLastUpdate)
	tg.Uid = inname.UniqueIdString()
	tg.Widx = es.writerIndex
	tg.LastUpdate = time.Now()

	_, err := es.conn.Update().
		Index(es.db.PathTable()).
		Type(es.db.PathType).
		Id(tg.Uid).
		Doc(tg).
		Do(context.Background())

	// not found is ok, just means it's new and the main indexer will update it
	if err != nil && !es5.IsNotFound(err) {
		es.log.Error("Could not insert id path %v (%s) : %v", inname.UniqueIdString(), inname.Key, err)
		stats.StatsdClientSlow.Incr("indexer.elasticsearch.update-failures", 1)
	} else {
		stats.StatsdClientSlow.Incr("indexer.elasticsearch.update-writes", 1)
	}
	return err
}

// keep an index of the stat keys and their fragments so we can look up
func (es *ElasticIndexer) Write(skey repr.StatName) error {
	es.updateQueue <- skey
	if es.ShouldWrite() {
		return es.cache.Add(skey)
	}
	return nil
}

func (es *ElasticIndexer) WriteTags(inname *repr.StatName, doMain bool, doMeta bool) error {

	haveMeta := !inname.MetaTags.IsEmpty()
	haveTgs := !inname.Tags.IsEmpty()
	if !haveTgs && !haveMeta || (!doMain && !doMeta) {
		return nil
	}

	if haveTgs && doMain {
		for _, t := range inname.Tags {
			_, got := es.inTagCache(t.Name, t.Value)
			if got {
				continue
			}
			tg := new(ESTag)
			tg.Name = t.Name
			tg.Value = t.Value
			tg.IsMeta = false
			tgSum := md5.New()
			tgSum.Write([]byte(fmt.Sprintf("%s:%s:%v", t.Name, t.Value, false)))
			ret, err := es.conn.Index().
				Index(es.db.TagTable()).
				Type(es.db.TagType).
				Id(hex.EncodeToString(tgSum.Sum(nil))).
				BodyJson(tg).
				Do(context.Background())
			if err != nil {
				es.log.Error("Could not insert tag %v (%v) :: %v", t.Name, t.Value, err)
				continue
			}
			if len(ret.Id) > 0 {
				es.tagIdCache.Add(t.Name, t.Value, false, ret.Id)
			}
		}
	}
	if haveMeta && doMeta {
		for _, t := range inname.Tags {
			_, got := es.inTagCache(t.Name, t.Value)
			if got {
				continue
			}
			tg := new(ESTag)
			tg.Name = t.Name
			tg.Value = t.Value
			tg.IsMeta = true
			tgSum := md5.New()
			tgSum.Write([]byte(fmt.Sprintf("%s:%s:%v", t.Name, t.Value, true)))
			_, err := es.conn.Index().
				Index(es.db.TagTable()).
				Type(es.db.TagType).
				Id(hex.EncodeToString(tgSum.Sum(nil))).
				BodyJson(tg).
				Do(context.Background())
			if err != nil {
				es.log.Error("Could not insert tag %v (%v) :: %v", t.Name, t.Value, err)
			}
		}
	}
	return nil
}

func (es *ElasticIndexer) periodicFlush() {
	tick := time.NewTicker(es.forceWriteTime)
	for {
		<-tick.C
		if atomic.LoadInt32(&es.numInBulk) > 0 {
			es.flush()
		}
	}
}

func (es *ElasticIndexer) flush() {
	// do if only more then needed
	l := atomic.LoadInt32(&es.numInBulk)
	if l > 0 {
		_, err := es.bulkQuery.Do(context.Background())
		es.log.Debug("Writing %d index items for %s", l, es.Name())
		if err != nil {
			stats.StatsdClientSlow.Incr("indexer.elastic.path-failures", 1)
		} else {
			stats.StatsdClientSlow.Incr("indexer.elastic.path-writes", 1)
		}
		atomic.StoreInt32(&es.numInBulk, 0)
	}
}

// a basic clone of the cassandra indexer
func (es *ElasticIndexer) WriteOne(inname *repr.StatName) error {
	defer stats.StatsdSlowNanoTimeFunc(fmt.Sprintf("indexer.elastic.write.path-time-ns"), time.Now())
	stats.StatsdClientSlow.Incr("indexer.elastic.noncached-writes-path", 1)

	skey := inname.Key
	uniqueID := inname.UniqueIdString()

	// we are going to assume that if the path is already in the system, we've indexed it and therefore
	// do not need to do the super loop (which is very expensive)
	got_already, err := es.conn.Get().Index(es.db.PathTable()).Type(es.db.PathType).Id(uniqueID).Do(context.Background())
	if err == nil && got_already != nil && got_already.Found {
		return es.WriteTags(inname, false, true)
	}

	pth := NewParsedPath(skey, uniqueID)

	// now to upsert them all (inserts in cass are upserts)
	for idx, seg := range pth.Segments {

		// now for each "partial path" add in the fact that it's not a "data" node
		// for each "segment" add in the path to do a segment to path(s) lookup
		// the skey one obviously has data
		// if key is consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99
		/* insert

		consthash -> consthash.zipperwork
		consthash.zipperwork -> consthash.zipperwork.local
		consthash.zipperwork.local -> consthash.zipperwork.local.writer
		consthash.zipperwork.local.writer -> consthash.zipperwork.local.writer.cassandra
		consthash.zipperwork.local.writer.cassandra -> consthash.zipperwork.local.writer.cassandra.write
		consthash.zipperwork.local.writer.cassandra.write -> consthash.zipperwork.local.writer.cassandra.write.metric-time-ns

		as data-less nodes

		*/
		esObj := new(ESPath) //getEsPath()
		//defer putESPath(esObj)
		esObj.Segment = seg.Segment
		esObj.Widx = es.writerIndex
		esObj.Pos = seg.Pos
		esObj.HasData = false
		esObj.Length = seg.Pos + 1
		esObj.LastUpdate = time.Now() // last time we've seen this

		esSeg := new(ESSegment)
		esSeg.Segment = seg.Segment
		esSeg.Pos = seg.Pos

		// we set the "_id" to the md5 of the path to avoid dupes
		es.bulkQuery.Add(es5.NewBulkIndexRequest().
			Index(es.db.SegmentTable()).
			Type(es.db.SegmentType).
			Id(esObj.Segment).
			Doc(*esSeg))

		atomic.AddInt32(&es.numInBulk, 1)
		//es.log.Critical("INS: %d %s %d %s %v", idx, seg.Segment, seg.Pos, skey, esObj)
		if skey != seg.Segment {
			esObj.Path = seg.Segment + "." + pth.Parts[idx+1]

			// we set the "_id" to the md5 of the path to avoid dupes
			pthMd := md5.New()
			pthMd.Write([]byte(esObj.Path))
			es.bulkQuery.Add(es5.NewBulkIndexRequest().
				Index(es.db.PathTable()).
				Type(es.db.PathType).
				Id(hex.EncodeToString(pthMd.Sum(nil))).
				Doc(*esObj))

			atomic.AddInt32(&es.numInBulk, 1)

		} else {
			// full path object
			esObj.Path = skey
			esObj.Uid = uniqueID
			esObj.Tags = make([]ESTag, 0)
			esObj.HasData = true

			if !inname.Tags.IsEmpty() {
				for _, tg := range inname.Tags {
					esObj.Tags = append(esObj.Tags, ESTag{
						Name:   tg.Name,
						Value:  tg.Value,
						IsMeta: false,
					})
				}
			}
			if !inname.MetaTags.IsEmpty() {
				for _, tg := range inname.MetaTags {
					esObj.Tags = append(esObj.Tags, ESTag{
						Name:   tg.Name,
						Value:  tg.Value,
						IsMeta: true,
					})
				}
			}
			es.bulkQuery.Add(es5.NewBulkIndexRequest().
				Index(es.db.PathTable()).
				Type(es.db.PathType).
				Id(uniqueID).
				Doc(*esObj))
			atomic.AddInt32(&es.numInBulk, 1)

		}
	}

	// do if only more then needed
	if atomic.LoadInt32(&es.numInBulk) > es.maxInBulk {
		es.flush()
	}

	err = es.WriteTags(inname, true, true)
	if err != nil {
		es.log.Error("Could not write tag index", err)
		return err
	}

	return err
}

func (es *ElasticIndexer) Delete(name *repr.StatName) error {

	es.RamIndexer.Delete(name)

	uid := name.UniqueIdString()
	_, err := es.conn.Delete().Index(es.db.PathTable()).Type(es.db.PathType).Id(uid).Do(context.Background())
	if err != nil {
		es.log.Error("Elastic Driver: Delete Path failed, %v", err)
		return err
	}

	return nil
}

/**** READER ***/

// Expand simply pulls out any regexes into full form
func (es *ElasticIndexer) Expand(metric string) (indexer.MetricExpandItem, error) {

	defer stats.StatsdSlowNanoTimeFunc("indexer.elastic.expand.get-time-ns", time.Now())

	m_len := len(strings.Split(metric, ".")) - 1
	baseQ := es.conn.Search().Index(es.db.SegmentTable()).Type(es.db.SegmentType)

	need_reg := needRegex(metric)
	andFilter := es5.NewBoolQuery()
	if need_reg {
		andFilter = andFilter.Must(es5.NewTermQuery("pos", m_len))
		andFilter = andFilter.Must(es5.NewRegexpQuery("segment", regifyKeyString(metric)))
		agg := es5.NewTermsAggregation().Field("segment")
		baseQ = baseQ.Aggregation("seg_count", agg)
	} else {
		andFilter = andFilter.Must(es5.NewTermQuery("pos", m_len))
		andFilter = andFilter.Must(es5.NewTermQuery("segment", metric))
		baseQ = baseQ.From(0).Size(1)
	}
	var me indexer.MetricExpandItem

	es_items, err := baseQ.Query(andFilter).Sort("segment", true).Do(context.Background())

	if err != nil {
		return me, err
	}

	// if we get a grouped item, we know we're on segment not data lands
	terms, ok := es_items.Aggregations.Terms("seg_count")
	if ok && len(terms.Buckets) < len(es_items.Hits.Hits) {
		for _, h := range terms.Buckets {
			str := h.Key.(string)
			me.Results = append(me.Results, str)

		}
	} else {

		for _, h := range es_items.Hits.Hits {
			// just grab the "n+1" length ones
			var item ESSegment
			err := json.Unmarshal(*h.Source, &item)
			if err != nil {
				es.log.Error("Elastic Driver: json error, %v", err)
				continue
			}
			me.Results = append(me.Results, item.Segment)
		}
	}

	if err != nil {
		return me, err
	}

	return me, nil

}

/******************* TAG METHODS **********************************/
func (es *ElasticIndexer) inTagCache(name string, value string) (string, bool) {

	got := es.tagIdCache.Get(name, value, false)

	if got != "" {
		return got, false
	}
	got = es.tagIdCache.Get(name, value, true)
	if got != "" {
		return got, true
	}

	return "", false
}

func (es *ElasticIndexer) FindTagId(name string, value string, ismeta bool) (string, error) {

	// see if in the writer tag cache
	c_id, cMeta := es.inTagCache(name, value)
	if ismeta == cMeta && c_id == "" {
		return c_id, nil
	}
	andFilter := es5.NewBoolQuery()
	andFilter = andFilter.Must(es5.NewTermQuery("name", name))
	andFilter = andFilter.Must(es5.NewTermQuery("value", value))
	andFilter = andFilter.Must(es5.NewTermQuery("is_meta", ismeta))

	items, err := es.conn.Search().Index(es.db.PathTable()).Type(es.db.PathType).Query(andFilter).From(0).Size(1).Do(context.Background())
	if err != nil {
		es.log.Error("Elastic Driver: Tag find error, %v", err)
		return "", err

	}
	// just the first one
	for _, h := range items.Hits.Hits {
		return h.Id, nil
	}
	return "", err
}

func (es *ElasticIndexer) GetTagsByUid(unique_id string) (tags repr.SortingTags, metatags repr.SortingTags, err error) {

	baseQ := es.conn.Search().Index(es.db.PathTable()).Type(es.db.PathType)
	items, err := baseQ.Query(es5.NewTermQuery("uid", unique_id)).Do(context.Background())
	if err != nil {
		return tags, metatags, err
	}

	for _, h := range items.Hits.Hits {
		tg := getESPath()
		defer putESPath(tg)
		err := json.Unmarshal(*h.Source, tg)

		if err != nil {
			es.log.Error("Error Getting Tags Iterator : %v", err)
			continue
		}
		tags, metatags := tg.ToSortedTags()
		return *tags, *metatags, nil
	}

	return tags, metatags, err
}

func (es *ElasticIndexer) GetTagsByName(name string, page int) (tags indexer.MetricTagItems, err error) {

	var items *es5.SearchResult
	baseQ := es.conn.Search().Index(es.db.TagTable()).Type(es.db.TagType)

	if needRegex(name) {
		use_name := regifyKeyString(name)
		n_q := es5.NewRegexpQuery("name", use_name)
		baseQ = baseQ.Query(n_q)
	} else {

		n_q := es5.NewTermQuery("name", name)
		baseQ = baseQ.Query(n_q)
	}
	items, err = baseQ.From(page * MAX_PER_PAGE).Size(MAX_PER_PAGE).Do(context.Background())

	if err != nil {
		es.log.Error("Elastic Driver: Tag find error, %v", err)
		return tags, err
	}

	// just the first one
	for _, h := range items.Hits.Hits {

		var tg ESTag
		err = json.Unmarshal(*h.Source, &tg)

		if err != nil {
			es.log.Error("Error Getting Tags Iterator : %v", err)
			continue
		}
		tags = append(tags, &indexer.MetricTagItem{Name: tg.Name, Value: tg.Value, Id: h.Id, IsMeta: tg.IsMeta})
		es.tagIdCache.Add(tg.Name, tg.Value, tg.IsMeta, h.Id)
	}
	return
}

func (es *ElasticIndexer) GetTagsByNameValue(name string, value string, page int) (tags indexer.MetricTagItems, err error) {
	var items *es5.SearchResult

	baseQ := es.conn.Search().Index(es.db.TagTable()).Type(es.db.TagType)
	andFilter := es5.NewBoolQuery()

	if needRegex(name) {
		andFilter = andFilter.Must(es5.NewRegexpQuery("name", regifyKeyString(name)))
	} else {
		andFilter = andFilter.Must(es5.NewTermQuery("name", name))
	}
	if needRegex(value) {
		andFilter = andFilter.Must(es5.NewRegexpQuery("value", regifyKeyString(value)))
	} else {
		andFilter = andFilter.Must(es5.NewTermQuery("value", value))
	}

	items, err = baseQ.Query(andFilter).From(page * MAX_PER_PAGE).Size(MAX_PER_PAGE).Do(context.Background())
	if err != nil {
		es.log.Error("Elastic Driver: Tag find error, %v", err)
		return tags, err
	}

	for _, h := range items.Hits.Hits {

		var tg ESTag
		err = json.Unmarshal(*h.Source, &tg)

		if err != nil {
			es.log.Error("Error Getting Tags Iterator : %v", err)
			continue
		}
		tags = append(tags, &indexer.MetricTagItem{Name: tg.Name, Value: tg.Value, Id: h.Id, IsMeta: tg.IsMeta})
		es.tagIdCache.Add(tg.Name, tg.Value, tg.IsMeta, h.Id)
	}

	return
}

func (es *ElasticIndexer) GetUidsByTags(key string, tags repr.SortingTags, page int) (uids []string, err error) {
	//TODO
	return
}

/********************* UID metric finders ***********************/

// List all paths w/ data
func (es *ElasticIndexer) List(has_data bool, page int) (indexer.MetricFindItems, error) {

	defer stats.StatsdSlowNanoTimeFunc("indexer.elastic.list.get-time-ns", time.Now())

	var mt indexer.MetricFindItems

	filter := es5.NewTermQuery("has_data", true)

	scroll := es.conn.Scroll().
		Index(es.db.PathTable()).
		Type(es.db.PathType).
		Query(filter).
		Size(MAX_PER_PAGE).
		Sort("path", true)

	for {
		items, err := scroll.Do(context.Background())
		if err == io.EOF {
			break // all results retrieved
		}
		if err != nil {
			es.log.Error("Error Getting List : %v", err)
			break
		}

		for _, h := range items.Hits.Hits {
			ms := new(indexer.MetricFindItem)

			tg := getESPath()
			defer putESPath(tg)
			err := json.Unmarshal(*h.Source, tg)

			if err != nil {
				es.log.Error("Error Getting Tags Iterator : %v", err)
				continue
			}
			spl := strings.Split(tg.Path, ".")

			ms.Text = spl[len(spl)-1]
			ms.Id = tg.Path
			ms.Path = tg.Path

			ms.Expandable = 0
			ms.Leaf = 1
			ms.AllowChildren = 0
			ms.UniqueId = tg.Uid
			t, m := tg.ToSortedTags()
			ms.Tags, ms.MetaTags = *t, *m

			mt = append(mt, ms)
		}
	}

	return mt, nil
}

// FindBase basic find for all paths
func (es *ElasticIndexer) FindBase(metric string, tags repr.SortingTags, exact bool) (indexer.MetricFindItems, error) {

	defer stats.StatsdSlowNanoTimeFunc("indexer.elastic.findbase.get-time-ns", time.Now())

	// check cache
	items := es.indexCache.Get(metric, tags)
	if items != nil {
		stats.StatsdClientSlow.Incr("indexer.elastic.findbase.cached", 1)
		return *items, nil
	}

	baseQ := es.conn.Search().Index(es.db.PathTable()).Type(es.db.PathType)
	andFilter := es5.NewBoolQuery()
	var all_tag_filter *es5.BoolQuery
	var mt indexer.MetricFindItems

	// if "tags" we need to find the tag Ids, and do the cross join
	for _, tag := range tags {
		t_ids, _ := es.GetTagsByNameValue(tag.Name, tag.Value, 0)
		if len(t_ids) > 0 {
			tag_filter := es5.NewBoolQuery()
			for _, tg := range t_ids {
				tag_filter = tag_filter.Must(es5.NewTermQuery("tags.name", tg.Name))
				tag_filter = tag_filter.Must(es5.NewTermQuery("tags.value", tg.Value))
				tag_filter = tag_filter.Must(es5.NewTermQuery("tags.is_meta", tg.IsMeta))
			}
			nest_filter := es5.NewNestedQuery("tags", tag_filter)
			if all_tag_filter == nil {
				all_tag_filter = es5.NewBoolQuery()
			}
			all_tag_filter = all_tag_filter.Must(nest_filter)
		} else {
			// cannot have results now can we
			return mt, nil
		}
	}

	need_reg := needRegex(metric)
	if len(metric) > 0 {
		if need_reg {
			andFilter = andFilter.Must(es5.NewTermQuery("pos", len(strings.Split(metric, "."))-1))
			andFilter = andFilter.Must(es5.NewRegexpQuery("segment", regifyKeyString(metric)))
			agg := es5.NewTermsAggregation().Field("segment").Size(es.maxResults)
			baseQ = baseQ.Aggregation("seg_count", agg).Size(es.maxResults)
		} else {
			andFilter = andFilter.Must(es5.NewTermQuery("pos", len(strings.Split(metric, "."))-1))
			andFilter = andFilter.Must(es5.NewTermQuery("segment", metric))
			baseQ = baseQ.From(0).Size(1)
		}
	}
	if all_tag_filter != nil {
		andFilter = andFilter.Must(all_tag_filter)
	}
	esItems, err := baseQ.
		Query(andFilter).
		Sort("segment", true).
		Do(context.Background())

	// dump query if debug mode
	if es.log.IsEnabledFor(logging.DEBUG) {
		agg := es5.NewTermsAggregation().Field("segment").Size(es.maxResults)
		ss, _ := es5.NewSearchSource().Aggregation("seg_count", agg).Query(andFilter).Sort("segment", true).Source()
		data, _ := json.Marshal(ss)
		es.log.Debug("ES Query: Index %s: (QUERY :: %s)", es.db.PathTable(), data)
	}

	if err != nil {
		agg := es5.NewTermsAggregation().Field("segment").Size(es.maxResults)
		ss, _ := es5.NewSearchSource().Aggregation("seg_count", agg).Query(andFilter).Sort("segment", true).Source()
		data, _ := json.Marshal(ss)
		es.log.Error("Query failed: Index %s: %v (QUERY :: %s)", es.db.PathTable(), err, data)
		//return mt, err
	}

	// if we get a grouped item, we know we're on segment not data lands
	terms, ok := esItems.Aggregations.Terms("seg_count")

	if ok && len(terms.Buckets) < len(esItems.Hits.Hits) {
		for _, h := range terms.Buckets {
			ms := new(indexer.MetricFindItem)

			str := h.Key.(string)
			spl := strings.Split(str, ".")

			ms.Text = spl[len(spl)-1]
			ms.Id = str
			ms.Path = str
			ms.Expandable = 1
			ms.Leaf = 0
			ms.AllowChildren = 1
			mt = append(mt, ms)
		}
	} else {

		for _, h := range esItems.Hits.Hits {
			ms := new(indexer.MetricFindItem)

			tg := getESPath()
			defer putESPath(tg)
			err := json.Unmarshal(*h.Source, tg)

			if err != nil {
				es.log.Error("Error in json: %v", err)
				continue
			}
			spl := strings.Split(tg.Segment, ".")

			ms.Text = spl[len(spl)-1]
			ms.Id = tg.Segment
			ms.Path = tg.Segment
			if tg.HasData {
				ms.Expandable = 0
				ms.Leaf = 1
				ms.AllowChildren = 0
				ms.UniqueId = tg.Uid
				t, m := tg.ToSortedTags()
				ms.Tags, ms.MetaTags = *t, *m
			} else {
				ms.Expandable = 1
				ms.Leaf = 0
				ms.AllowChildren = 1
			}
			mt = append(mt, ms)
		}
	}

	// set it
	es.indexCache.Add(metric, tags, &mt)

	return mt, nil
}

// special case for "root" == "*" finder
func (es *ElasticIndexer) FindRoot(tags repr.SortingTags) (indexer.MetricFindItems, error) {
	defer stats.StatsdSlowNanoTimeFunc("indexer.elastic.findroot.get-time-ns", time.Now())

	var mt indexer.MetricFindItems

	baseQ := es.conn.Search().Index(es.db.SegmentTable()).Type(es.db.SegmentType)
	andFilter := es5.NewBoolQuery()
	andFilter = andFilter.Must(es5.NewTermQuery("pos", 0))
	var all_tag_filter *es5.BoolQuery

	// if "tags" we need to find the tag Ids, and do the cross join
	for _, tag := range tags {
		t_ids, _ := es.GetTagsByNameValue(tag.Name, tag.Value, 0)
		if len(t_ids) > 0 {
			tag_filter := es5.NewBoolQuery()
			for _, tg := range t_ids {
				tag_filter = tag_filter.Must(es5.NewTermQuery("tags.name", tg.Name))
				tag_filter = tag_filter.Must(es5.NewTermQuery("tags.value", tg.Value))
				tag_filter = tag_filter.Must(es5.NewTermQuery("tags.is_meta", tg.IsMeta))
			}
			nest_filter := es5.NewNestedQuery("tags", tag_filter)
			if all_tag_filter == nil {
				all_tag_filter = es5.NewBoolQuery()
			}
			all_tag_filter = all_tag_filter.Must(nest_filter)
		} else {
			// cannot have results now can we
			return mt, nil
		}
	}

	if all_tag_filter != nil {
		andFilter = andFilter.Must(all_tag_filter)
	}
	es_items, err := baseQ.
		Query(andFilter).
		Sort("segment", true).
		Size(es.maxResults).
		Do(context.Background())

	if err != nil {
		return mt, err
	}
	for _, h := range es_items.Hits.Hits {

		var tg ESSegment
		ms := new(indexer.MetricFindItem)

		err := json.Unmarshal(*h.Source, &tg)

		if err != nil {
			es.log.Error("Error in json: %v", err)
			continue
		}
		ms.Text = tg.Segment
		ms.Id = tg.Segment
		ms.Path = tg.Segment
		ms.Expandable = 1
		ms.Leaf = 0
		ms.AllowChildren = 1

		mt = append(mt, ms)
	}

	return mt, nil
}

// FindInCache return the values from the RamIndex
func (es *ElasticIndexer) FindInCache(ctx context.Context, metric string, tags repr.SortingTags) (indexer.MetricFindItems, error) {
	return es.RamIndexer.Find(ctx, metric, tags)
}

// Find to allow for multiple targets
func (es *ElasticIndexer) Find(ctx context.Context, metric string, tags repr.SortingTags) (indexer.MetricFindItems, error) {
	// the regex case is a bit more complicated as we need to grab ALL the segments of a given length.
	// see if the match the regex, and then add them to the lists since cassandra does not provide regex abilities
	// on the server side
	sp, closer := es.GetSpan("Find", ctx)
	sp.LogKV("driver", "ElasticIndexer", "metric", metric, "tags", tags)
	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("indexer.elastic.find.get-time-ns", time.Now())

	// special case for "root" == "*"

	// check cache
	items := es.indexCache.Get(metric, tags)
	if items != nil {
		stats.StatsdClientSlow.Incr("indexer.elastic.find.cached", 1)
		return *items, nil
	}

	if metric == "*" {
		return es.FindRoot(tags)
	}

	mt, err := es.FindBase(metric, tags, true)
	if err != nil {
		return mt, err
	}
	// set it
	es.indexCache.Add(metric, tags, &mt)

	return mt, nil
}

/************************************************************************/
/**********  Standard Worker Dispatcher JOB   ***************************/
/************************************************************************/
// insert job queue workers
type elasticIndexerJob struct {
	ES    *ElasticIndexer
	Stat  repr.StatName
	retry int
}

func (j *elasticIndexerJob) IncRetry() int {
	j.retry++
	return j.retry
}
func (j *elasticIndexerJob) OnRetry() int {
	return j.retry
}

func (j *elasticIndexerJob) DoWork() error {
	err := j.ES.WriteOne(&j.Stat)
	if err != nil {
		j.ES.log.Error("Insert failed for Index: %v retrying ...", j.Stat)
	}
	return err
}
