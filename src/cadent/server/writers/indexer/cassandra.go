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
	The Cassandra Index Writer/Reader

	The table should have this schema to match the repr item

		keyspace: base keyspace name (default: metric)
		path_table: base table name (default: path)
		segment_table: base table name (default: segment)
		write_consistency: "one"
		read_consistency: "one"
		port: 9042
		numcons: 5  (connection pool size)
		timeout: "30s"
		user: ""
		pass: ""
		should_write: true # if false, things will just write to the ram cache



	a "brief" schema ..

	CREATE TYPE metric.segment_pos (
    		pos int,
    		segment text
	);

	CREATE TABLE metric.segment (
   		pos int,
   		segment text,
   		PRIMARY KEY (pos, segment)
	) WITH COMPACT STORAGE AND CLUSTERING ORDER BY (segment ASC)


	CREATE TABLE metric.path (
    		segment frozen<segment_pos>,
    		path text,
    		length int,
    		has_data boolean,
 		id varchar,  # repr.StatName.UniqueIDString()
  		PRIMARY KEY ((segment, length), path, id)
	) WITH CLUSTERING ORDER BY (path ASC)

	CREATE INDEX ON metric.path (id);

	CREATE TABLE metric.tag (
    		id varchar,  # see repr.StatName.UniqueId()
    		tags list<text>  # this will be a [ "name=val", "name=val", ...] so we can do `IN "moo=goo" in tag`
    		PRIMARY KEY (id)
	);
	CREATE INDEX ON metric.tag (tags);

	# an index of tags basically to do name="{regex}" things
	# get a list of name=values and then Q the metric.tag for id lists
	CREATE TABLE metric.tag_list (
		name text
    		value text
    		PRIMARY KEY (name)
	);
	CREATE INDEX ON metric.tag_list (value);


*/

package indexer

import (
	"cadent/server/dispatch"
	stats "cadent/server/stats"
	"cadent/server/writers/dbs"
	"fmt"
	"github.com/gocql/gocql"
	logging "gopkg.in/op/go-logging.v1"

	"cadent/server/schemas/indexer"
	"cadent/server/schemas/repr"
	"cadent/server/utils"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"golang.org/x/net/context"
	"strings"
	"sync/atomic"
	"time"
)

const (
	CASSANDRA_INDEXER_QUEUE_LEN = 1024 * 1024
	CASSANDRA_INDEXER_WORKERS   = 128
	CASSANDRA_WRITES_PER_SECOND = 1000

	// for the Ram index, we only update if seen date if the last added has a delta bigger then this
	// this keeps the a nutty "update the last seen" time down inside cassandra itself and
	// save some cycles on the "add" process
	CASSANDRA_HIT_LAST_UPDATE        = time.Duration(15 * time.Minute)
	CASSANDRA_HIT_LAST_UPDATE_BUFFER = 20480
)

/****************** Writer *********************/
type CassandraIndexer struct {
	RamIndexer
	db        *dbs.CassandraDB
	conn      *gocql.Session
	indexerId string

	numWorkers int
	queueLen   int
	shutitdown uint32 //shutdown notice
	startstop  utils.StartStop

	writeQueue      chan dispatch.IJob
	dispatchQueue   chan chan dispatch.IJob
	writeDispatcher *dispatch.Dispatch

	writesPerSecond int // rate limit writer

	updateKeyExpireChan chan repr.StatName

	log *logging.Logger

	indexCache *IndexReadCache

	writerIndex int
	shouldWrite bool
	okToWrite   chan bool

	// general fix queries strings
	selectPathQ    string
	insertPathQ    string
	selectIdQ      string
	insertIdQ      string
	selectSegmentQ string
}

func NewCassandraIndexer() *CassandraIndexer {
	cass := new(CassandraIndexer)
	cass.log = logging.MustGetLogger("indexer.cassandra")
	cass.indexCache = NewIndexCache(10000)
	cass.RamIndexer = *NewRamIndexer()
	cass.shouldWrite = true
	cass.okToWrite = make(chan bool)
	atomic.SwapUint32(&cass.shutitdown, 0)
	cass.updateKeyExpireChan = make(chan repr.StatName, CASSANDRA_HIT_LAST_UPDATE_BUFFER)
	return cass
}

func (cass *CassandraIndexer) Config(conf *options.Options) (err error) {
	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("Indexer: `dsn` (server1,server2,server3) is needed for cassandra config")
	}

	cass.indexerId = conf.String("name", fmt.Sprintf(
		"%v:%v/%v|%v|%v|%v",
		dsn,
		conf.Int64("port", 9042),
		conf.String("keyspace", "metric"),
		conf.String("path_table", "path"),
		conf.String("segment_table", "segment"),
		conf.String("id_table", "ids"),
	))

	i, err := conf.Int64Required("writer_index")
	if err != nil {
		return err
	}
	cass.writerIndex = int(i)

	cass.log.Noticef("Registering indexer: %s", cass.Name())

	// config the Ram Cacher
	err = cass.RamIndexer.ConfigRam(conf)
	if err != nil {
		cass.log.Critical(err.Error())
		return err
	}

	// reg ourselves
	err = RegisterIndexer(cass.Name(), cass)
	if err != nil {
		return err
	}

	cass.log.Notice("Connecting Indexer to Cassandra %s", cass.indexerId)
	db, err := dbs.NewDB("cassandra", cass.indexerId, conf)
	if err != nil {
		return err
	}
	cass.db = db.(*dbs.CassandraDB)
	cass.conn, err = cass.db.GetSession()
	if err != nil {
		return err
	}

	// tweak queues and worker sizes
	cass.numWorkers = int(conf.Int64("write_workers", CASSANDRA_INDEXER_WORKERS))
	cass.queueLen = int(conf.Int64("write_queue_length", CASSANDRA_INDEXER_QUEUE_LEN))

	c_key := "indexer:cassandra:" + cass.indexerId
	cass.cache, err = getCacherSingleton(c_key)
	if err != nil {
		return err
	}
	cass.cache.maxKeys = int(conf.Int64("cache_index_size", CACHER_METRICS_KEYS))
	cass.writesPerSecond = int(conf.Int64("writes_per_second", CASSANDRA_WRITES_PER_SECOND))

	cass.shouldWrite = conf.Bool("should_write", true)

	// set the buffer if we have one
	updateBuffer := int(conf.Int64("update_tick_buffer", CASSANDRA_HIT_LAST_UPDATE_BUFFER))
	if updateBuffer != CASSANDRA_HIT_LAST_UPDATE_BUFFER {
		cass.updateKeyExpireChan = make(chan repr.StatName, updateBuffer)
	}

	return nil
}

func (cass *CassandraIndexer) Start() {
	cass.startstop.Start(func() {
		cass.log.Notice("starting up cassandra indexer: %s", cass.Name())
		cass.log.Notice("Adding index tables ...")

		schems := NewCassandraIndexerSchema(
			cass.conn, cass.db.Keyspace(), cass.db.PathTable(), cass.db.SegmentTable(), cass.db.IdTable(),
		)
		err := schems.AddIndexerTables()
		if err != nil {
			panic(err)
		}

		workers := cass.numWorkers
		cass.writeQueue = make(chan dispatch.IJob, cass.queueLen)
		cass.dispatchQueue = make(chan chan dispatch.IJob, workers)
		cass.writeDispatcher = dispatch.NewDispatch(workers, cass.dispatchQueue, cass.writeQueue)
		cass.writeDispatcher.SetRetries(2)
		cass.writeDispatcher.Run()

		cass.RamIndexer.Start() //start cacher

		// backfill things
		go cass.backfill()

		// two workers
		go cass.processNameUpdate()
		go cass.processNameUpdate()

		// make the nice query strings
		cass.selectPathQ = fmt.Sprintf(
			"SELECT id,path,length,has_data FROM %s WHERE segment={pos: ?, segment: ?} ",
			cass.db.PathTable(),
		)

		cass.selectIdQ = fmt.Sprintf(
			"SELECT id,widx,path,tags,metatags,lastseen FROM %s WHERE id=? AND widx=? ",
			cass.db.IdTable(),
		)
		// this is a cassandra "upsert" basically
		cass.insertIdQ = fmt.Sprintf(
			"UPDATE %s SET path=?, tags=?, metatags=?, lastseen=?, widx=? WHERE id=?",
			cass.db.IdTable(),
		)

		cass.insertPathQ = fmt.Sprintf(
			"INSERT INTO %s (segment, path, id, length, has_data) VALUES  ({pos: ?, segment: ?}, ?, ?, ?, ?)",
			cass.db.PathTable(),
		)

		cass.selectSegmentQ = fmt.Sprintf(
			"SELECT segment FROM %s WHERE pos=? AND segment=?",
			cass.db.SegmentTable(),
		)

		// we are initialize
		close(cass.okToWrite)
		cass.okToWrite = nil

		go cass.sendToWriters() // the dispatcher
	})
}

func (cass *CassandraIndexer) Stop() {
	cass.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()

		if atomic.SwapUint32(&cass.shutitdown, 1) == 1 {
			return // already did
		}
		cass.log.Notice("shutting down cassandra indexer: %s", cass.Name())

		cass.RamIndexer.Stop()
		if cass.writeQueue != nil {
			cass.writeDispatcher.Shutdown()
		}

		close(cass.updateKeyExpireChan)
	})
}

// ShouldWrite based on the `should_write` option, if false, just populate the ram caches
func (cass *CassandraIndexer) ShouldWrite() bool {
	return cass.shouldWrite
}

func (cass *CassandraIndexer) Name() string {
	return cass.indexerId
}

// given the id table, backfill the ram index
func (cass *CassandraIndexer) backfill() error {

	cass.log.Notice("Backfilling Index into ram ...")
	sess, err := cass.db.GetSession()
	if err != nil {
		cass.log.Error(err.Error())
		return err
	}

	// yep full table scan, but nessesary
	Q := fmt.Sprintf("SELECT id, widx, path, tags, metatags, lastseen FROM %s", cass.db.IdTable())
	iter := sess.Query(Q).PageSize(2000).Iter()

	var path string
	var tags []string
	var metatags []string
	var id string
	var lastseen int
	var widx int
	did := 0
	for iter.Scan(&id, &widx, &path, &tags, &metatags, &lastseen) {
		if widx != cass.writerIndex {
			continue
		}
		sName := repr.StatName{
			Key:      path,
			Tags:     *repr.SortingTagsFromArray(tags),
			MetaTags: *repr.SortingTagsFromArray(tags),
		}
		cass.RamIndexer.WriteLastSeen(sName, lastseen)
		did++
		if did%1000 == 0 {
			cass.log.Notice("Backfilled %d paths...", did)
		}
	}
	cass.log.Notice("DONE Backfilled %d paths", did)

	if err := iter.Close(); err != nil {
		cass.log.Error(err.Error())
		return err
	}
	return nil

}

// this writes an "update" to the id table, we don't do this every write, just periodically
// otherwise we'd kill cassandra w/ a bunch of silly updates
func (cass *CassandraIndexer) writeUpdate(inname repr.StatName) error {
	// only if we need to update it
	//slap in the id -> path table
	uid := inname.UniqueIdString()
	err := cass.conn.Query(cass.insertIdQ,
		inname.Key, inname.Tags.StringList(), inname.MetaTags.StringList(), time.Now().Unix(), cass.writerIndex, uid,
	).Exec()

	if err != nil {
		cass.log.Error("Could not insert id path %v (%s) (%s): %v", uid, inname.Key, cass.insertIdQ, err)
		stats.StatsdClientSlow.Incr("indexer.cassandra.update-failures", 1)
	} else {
		stats.StatsdClientSlow.Incr("indexer.cassandra.update-writes", 1)
	}
	return err
}

func (cass *CassandraIndexer) WriteOne(inname repr.StatName) error {
	defer stats.StatsdSlowNanoTimeFunc(fmt.Sprintf("indexer.cassandra.write.path-time-ns"), time.Now())
	stats.StatsdClientSlow.Incr("indexer.cassandra.noncached-writes-path", 1)
	onName := inname

	skey := onName.Key

	// we are going to assume that if the path is already in the system, we've indexed it and therefore
	// do not need to do the super loop (which is very expensive)

	uid := onName.UniqueIdString()
	pth := NewParsedPath(skey, uid)

	// if we've made it here the write cache index backend has not been hit
	cass.writeUpdate(onName)

	/* Skip this for now
	SelQ := fmt.Sprintf(
		"SELECT path, length, has_data FROM %s WHERE segment={pos: ?, segment: ?}",
		cass.db.PathTable(),
	)
	var _pth string
	var _len int
	var _dd bool
	gerr := cass.conn.Query(SelQ, pth.Len-1, skey).Scan(&_pth, &_len, &_dd)

	// got it
	if gerr == nil {
		if _pth == skey && _dd && _len == pth.Len-1 {
			return nil
		}
	}*/

	lastPath := pth.Last()
	// now to upsert them all (inserts in cass are upserts)
	segQ := fmt.Sprintf(
		"INSERT INTO %s (pos, segment) VALUES  (?, ?) ",
		cass.db.SegmentTable(),
	)

	for idx, seg := range pth.Segments {
		err := cass.conn.Query(segQ, seg.Pos, seg.Segment).Exec()

		if err != nil {
			cass.log.Error("Could not insert segment %v (%s) : %v", seg, skey, err)
			stats.StatsdClientSlow.Incr("indexer.cassandra.segment-failures", 1)
		} else {
			stats.StatsdClientSlow.Incr("indexer.cassandra.segment-writes", 1)
		}

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

		consthash.zipperwork.local.writer.cassandra.write.metric-time-ns -> consthash.zipperwork.local.writer.cassandra.write.metric-time-ns

		as "data'ed" nodes

		*/

		if skey != seg.Segment && idx < pth.Len-1 {
			err = cass.conn.Query(
				cass.insertPathQ,
				seg.Pos, seg.Segment, seg.Segment+"."+pth.Parts[idx+1], "", seg.Pos+1, false,
			).Exec()
		} else {
			//the "raw data" path
			err = cass.conn.Query(
				cass.insertPathQ,
				seg.Pos, seg.Segment, skey, uid, pth.Len-1, true,
			).Exec()
		}

		//cass.log.Critical("DATA:: Seg INS: %s PATH: %s Len: %d", seg.Segment, skey, p_len-1)

		if err != nil {
			cass.log.Error("Could not insert path %v (%v) :: %v", lastPath, uid, err)
			stats.StatsdClientSlow.Incr("indexer.cassandra.path-failures", 1)
		} else {
			stats.StatsdClientSlow.Incr("indexer.cassandra.path-writes", 1)
		}
	}
	return nil
}

// pop from the cache and send to actual writers
func (cass *CassandraIndexer) sendToWriters() error {
	// this may not be the "greatest" ratelimiter of all time,
	// as "high frequency tickers" can be costly .. but should the workers get backedup
	// it will block on the writeQueue stage

	if cass.writesPerSecond <= 0 {
		cass.log.Notice("Starting indexer writer: No rate limiting enabled")
		for {
			if cass.shutitdown == 1 {
				return nil
			}
			skey := cass.cache.Pop()
			switch skey.IsBlank() {
			case true:
				time.Sleep(time.Second)
			default:
				stats.StatsdClient.Incr(fmt.Sprintf("indexer.cassandra.write.send-to-writers"), 1)
				cass.writeQueue <- &cassandraIndexerJob{Cass: cass, Stat: skey}
			}
		}
	} else {
		sleep_t := float64(time.Second) * (time.Second.Seconds() / float64(cass.writesPerSecond))
		cass.log.Notice("Starting indexer writer: limiter every %f nanoseconds (%d writes per second)", sleep_t, cass.writesPerSecond)
		dur := time.Duration(int(sleep_t))
		for {
			if cass.shutitdown == 1 {
				return nil
			}
			skey := cass.cache.Pop()
			switch skey.IsBlank() {
			case true:
				time.Sleep(time.Second)
			default:
				stats.StatsdClient.Incr(fmt.Sprintf("indexer.cassandra.write.send-to-writers"), 1)
				cass.writeQueue <- &cassandraIndexerJob{Cass: cass, Stat: skey}
				time.Sleep(dur)
			}
		}
	}
}

func (cass *CassandraIndexer) processNameUpdate() {
	for skey := range cass.updateKeyExpireChan {
		if cass.RamIndexer.NeedsWrite(skey, CASSANDRA_HIT_LAST_UPDATE) {
			cass.RamIndexer.Write(skey) //add to ram index
			if cass.ShouldWrite() {
				cass.writeUpdate(skey)
			}
		}
	}
}

// Write to the index
func (cass *CassandraIndexer) Write(skey repr.StatName) error {
	// block until ready
	if cass.okToWrite != nil {
		<-cass.okToWrite
	}

	cass.updateKeyExpireChan <- skey

	if cass.ShouldWrite() {
		return cass.cache.Add(skey)
	}
	return nil
}

/** reader methods **/

func (cass *CassandraIndexer) ExpandNonRegex(metric string) (indexer.MetricExpandItem, error) {
	paths := strings.Split(metric, ".")
	m_len := len(paths)

	iter := cass.conn.Query(cass.selectSegmentQ, m_len-1, metric).Iter()

	var on_pth string

	var me indexer.MetricExpandItem
	// just grab the "n+1" length ones
	for iter.Scan(&on_pth) {
		me.Results = append(me.Results, on_pth)
	}
	if err := iter.Close(); err != nil {
		cass.log.Error(err.Error())
		return me, err
	}
	return me, nil
}

// Expand simply pulls out any regexes into full form
func (cass *CassandraIndexer) Expand(metric string) (indexer.MetricExpandItem, error) {

	defer stats.StatsdSlowNanoTimeFunc("indexer.cassandra.expand.get-time-ns", time.Now())

	// use the ram index if we can
	if cass.RamIndexer.HasData() {
		return cass.RamIndexer.Expand(metric)
	}

	needs_regex := needRegex(metric)
	//cass.log.Debug("REGSS: %v, %s", needs_regex, metric)

	if !needs_regex {
		return cass.ExpandNonRegex(metric)
	}
	paths := strings.Split(metric, ".")
	m_len := len(paths)

	var me indexer.MetricExpandItem

	the_reg, err := regifyKey(metric)

	if err != nil {
		return me, err
	}
	cass_Q := fmt.Sprintf(
		"SELECT segment FROM %s WHERE pos=?",
		cass.db.SegmentTable(),
	)
	//cass.log.Debug("POSPOSPOS: %s", m_len-1)

	iter := cass.conn.Query(cass_Q,
		m_len-1,
	).Iter()

	var seg string
	// just grab the "n+1" length ones
	for iter.Scan(&seg) {
		//cass.log.Debug("SEG: %s", seg)

		if !the_reg.MatchString(seg) {
			continue
		}
		me.Results = append(me.Results, seg)
	}
	if err := iter.Close(); err != nil {
		cass.log.Error(err.Error())
		return me, err
	}
	return me, nil

}

func (cass *CassandraIndexer) List(hasData bool, page int) (indexer.MetricFindItems, error) {
	defer stats.StatsdSlowNanoTimeFunc("indexer.cassandra.list.get-time-ns", time.Now())

	// use the Ram index if we can
	if cass.RamIndexer.HasData() {
		return cass.RamIndexer.List(hasData, page)
	}

	// since there are and regex like things in the strings, we
	// need to get the all "items" from where the regex starts then hone down

	// grab all the paths that match this length if there is no regex needed
	// these are the "data/leaf" nodes

	//cassandra does "auto pagination" but we really do not want to send back all the rows
	// so we instead need to walk the iterator
	cassQ := fmt.Sprintf(
		"SELECT id,path,tags,metatags FROM %s LIMIT 2147483600",
		cass.db.IdTable(),
	)
	iter := cass.conn.Query(cassQ).PageSize(MAX_PER_PAGE).Iter()

	var mt indexer.MetricFindItems
	var onPth string
	var tags []string
	var metatags []string
	var id string

	cur_page := 0
	// just grab the "n+1" length ones
	for iter.Scan(&id, &onPth, &tags, &metatags) {
		ms := new(indexer.MetricFindItem)

		if iter.WillSwitchPage() {
			cur_page += 1
		}
		if cur_page < page {
			continue
		}

		spl := strings.Split(onPth, ".")

		ms.Text = spl[len(spl)-1]
		ms.Id = onPth
		ms.Path = onPth

		ms.Expandable = 0
		ms.Leaf = 1
		ms.AllowChildren = 0
		ms.UniqueId = id
		ms.Tags = *repr.SortingTagsFromArray(tags)
		ms.MetaTags = *repr.SortingTagsFromArray(metatags)

		mt = append(mt, ms)
	}

	if err := iter.Close(); err != nil {
		cass.log.Error(err.Error())
		return mt, err
	}

	return mt, nil
}

func (cass *CassandraIndexer) FindByUid(metric string) (indexer.MetricFindItems, error) {
	defer stats.StatsdSlowNanoTimeFunc("indexer.cassandra.findbyuid.get-time-ns", time.Now())

	// should only be one
	cassQ := cass.selectIdQ

	iter := cass.conn.Query(cassQ, metric).Iter()

	var mt indexer.MetricFindItems
	var path string
	var tags []string
	var metatags []string
	var id string
	var lastseen int

	for iter.Scan(&id, &path, &tags, &metatags, &lastseen) {
		ms := new(indexer.MetricFindItem)

		paths := strings.Split(path, ".")
		ms.Text = paths[len(paths)-1]
		ms.Id = metric
		ms.Path = metric
		ms.Expandable = 0
		ms.Leaf = 1
		ms.AllowChildren = 0
		ms.UniqueId = id

		// grab ze tags
		ms.Tags = *repr.SortingTagsFromArray(tags)
		ms.MetaTags = *repr.SortingTagsFromArray(metatags)
		mt = append(mt, ms)

		iter.Close()
		cass.indexCache.Add(metric, repr.SortingTags{}, &mt)
		return mt, nil
	}

	if err := iter.Close(); err != nil {
		cass.log.Error(err.Error())
		return mt, err
	}

	// set it
	cass.indexCache.Add(metric, repr.SortingTags{}, &mt)

	return mt, nil

}

// basic find for non-regex items
func (cass *CassandraIndexer) FindNonRegex(ctx context.Context, metric string, tags repr.SortingTags, exact bool) (indexer.MetricFindItems, error) {

	defer stats.StatsdSlowNanoTimeFunc("indexer.cassandra.findnoregex.get-time-ns", time.Now())
	// use the ram index if we can
	if cass.RamIndexer.HasData() {
		return cass.RamIndexer.Find(ctx, metric, tags)
	}
	// check cache
	items := cass.indexCache.Get(metric, tags)
	if items != nil {
		stats.StatsdClientSlow.Incr("indexer.cassandra.findnoregex.cached", 1)
		return *items, nil
	}

	// since there are and regex like things in the strings, we
	// need to get the all "items" from where the regex starts then hone down

	paths := strings.Split(metric, ".")
	mLen := len(paths)

	// grab all the paths that match this length if there is no regex needed
	// these are the "data/leaf" nodes
	cass_Q := fmt.Sprintf(
		"SELECT id,path,length,has_data FROM %s WHERE segment={pos: ?, segment: ?} ",
		cass.db.PathTable(),
	)

	if exact {
		cass_Q += " LIMIT 1"
	}

	iter := cass.conn.Query(cass_Q, mLen-1, metric).Iter()

	var mt indexer.MetricFindItems
	var onPth string
	var pthLen int
	var id string
	var hasData bool

	for iter.Scan(&id, &onPth, &pthLen, &hasData) {
		if pthLen > mLen {
			continue
		}
		ms := new(indexer.MetricFindItem)

		// we are looking to see if this is a "data" node or "expandable"
		if exact {
			ms.Text = paths[len(paths)-1]
			ms.Id = metric
			ms.Path = metric
			if onPth == metric && hasData {
				ms.Expandable = 0
				ms.Leaf = 1
				ms.AllowChildren = 0
				ms.UniqueId = id

				// grab ze tags
				ms.Tags, ms.MetaTags, _ = cass.GetTagsByUid(id)
				mt = append(mt, ms)
			} else {
				ms.Expandable = 1
				ms.Leaf = 0
				ms.AllowChildren = 1
				mt = append(mt, ms)
			}
			iter.Close()
			cass.indexCache.Add(metric, tags, &mt)
			return mt, nil
		}

		//cass.log.Critical("NON REG:::::PATH %s LEN %d m_len: %d", on_pth, pth_len, m_len)
		spl := strings.Split(onPth, ".")

		ms.Text = spl[len(spl)-1]
		ms.Id = onPth
		ms.Path = onPth

		if hasData {
			ms.Expandable = 0
			ms.Leaf = 1
			ms.AllowChildren = 0
			ms.UniqueId = id
		} else {
			ms.Expandable = 1
			ms.Leaf = 0
			ms.AllowChildren = 1
		}

		mt = append(mt, ms)
	}

	if err := iter.Close(); err != nil {
		cass.log.Error(err.Error())
		return mt, err
	}

	// set it
	cass.indexCache.Add(metric, tags, &mt)

	return mt, nil
}

// special case for "root" == "*" finder
func (cass *CassandraIndexer) FindRoot(ctx context.Context) (indexer.MetricFindItems, error) {
	defer stats.StatsdSlowNanoTimeFunc("indexer.cassandra.findroot.get-time-ns", time.Now())
	// use the Ram index if we can
	if cass.RamIndexer.HasData() {
		return cass.RamIndexer.Find(ctx, "*", nil)
	}

	cass_Q := fmt.Sprintf(
		"SELECT segment FROM %s WHERE pos=?",
		cass.db.SegmentTable(),
	)
	iter := cass.conn.Query(cass_Q,
		0,
	).Iter()

	var mt indexer.MetricFindItems
	var seg string

	for iter.Scan(&seg) {
		ms := new(indexer.MetricFindItem)
		ms.Text = seg
		ms.Id = seg
		ms.Path = seg
		ms.UniqueId = ""
		ms.Expandable = 1
		ms.Leaf = 0
		ms.AllowChildren = 1

		mt = append(mt, ms)
	}

	if err := iter.Close(); err != nil {
		cass.log.Error(err.Error())
		return mt, err
	}

	return mt, nil
}

// FindInCache return all the things in the RamIndex object
func (cass *CassandraIndexer) FindInCache(ctx context.Context, metric string, tags repr.SortingTags) (indexer.MetricFindItems, error) {
	return cass.RamIndexer.Find(ctx, metric, tags)
}

// Find to allow for multiple targets of the form `my.*.metric`
func (cass *CassandraIndexer) Find(ctx context.Context, metric string, tags repr.SortingTags) (indexer.MetricFindItems, error) {
	// the regex case is a bit more complicated as we need to grab ALL the segments of a given length.
	// see if the match the regex, and then add them to the lists since cassandra does not provide regex abilities
	// on the server side
	sp, closer := cass.GetSpan("Find", ctx)
	sp.LogKV("driver", "CassandraIndexer", "metric", metric, "tags", tags)
	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("indexer.cassandra.find.get-time-ns", time.Now())

	// special case for "root" == "*"
	// check cache
	items := cass.indexCache.Get(metric, tags)
	if items != nil {
		stats.StatsdClientSlow.Incr("indexer.cassandra.find.cached", 1)
		return *items, nil
	}

	if metric == "*" {
		return cass.FindRoot(ctx)
	}

	// use the Ram index if we can
	if cass.RamIndexer.HasData() {
		return cass.RamIndexer.Find(ctx, metric, tags)
	}

	needs_regex := needRegex(metric)

	//cass.log.Debug("HasReg: %v Metric: %s", needs_regex, metric)
	if !needs_regex {
		return cass.FindNonRegex(ctx, metric, tags, true)
	}

	paths := strings.Split(metric, ".")
	mLen := len(paths)

	// convert the "graphite regex" into something golang understands (just the "."s really)
	// need to replace things like "moo*" -> "moo.*" but careful not to do "..*"
	// the "graphite" globs of {moo,goo} we can do with (moo|goo) so convert { -> (, , -> |, } -> )
	the_reg, err := regifyKey(metric)

	if err != nil {
		return nil, err
	}
	cass_Q := fmt.Sprintf(
		"SELECT segment FROM %s WHERE pos=?",
		cass.db.SegmentTable(),
	)
	iter := cass.conn.Query(cass_Q, mLen-1).PageSize(MAX_PER_PAGE).Iter()

	var mt indexer.MetricFindItems

	var seg string
	// just grab the "n+1" length ones
	for iter.Scan(&seg) {
		//cass.log.Debug("REG:::::PATH %s :: REG: %s MATCH %v", seg, regable, the_reg.Match([]byte(seg)))

		if !the_reg.MatchString(seg) {
			continue
		}
		items, err := cass.FindNonRegex(ctx, seg, tags, true)
		if err != nil {
			cass.log.Warning("could not get segments for `%s` :: %v", seg, err)
			continue
		}
		if items != nil && len(items) > 0 {
			mt = append(mt, items...)
		}
	}
	cass.indexCache.Add(metric, tags, &mt)

	if err := iter.Close(); err != nil {
		cass.log.Error(err.Error())
		return mt, err
	}

	return mt, nil
}

// delete a path from the index
// TODO
func (cass *CassandraIndexer) Delete(name *repr.StatName) (err error) {
	cass.RamIndexer.Delete(name)

	mLen := len(strings.Split(name.Key, "."))
	cassQ := fmt.Sprintf(
		"DELETE FROM %s WHERE segment={pos: ?, segment: ?} AND has_data=1 AND length=? AND id=?",
		cass.db.PathTable(),
	)

	// Paths table
	err = cass.conn.Query(cassQ,
		mLen-1, name.Key, mLen-1, name.Key, name.UniqueIdString(),
	).Exec()
	if err != nil {
		cass.log.Errorf("Error removing paths: %v", err)
	}

	cassQ = fmt.Sprintf("DELETE FROM %s WHERE pos=? AND segment=?", cass.db.SegmentTable())

	// Segment table
	err = cass.conn.Query(cassQ, mLen-1, name.Key).Exec()
	if err != nil {
		cass.log.Errorf("Error removing segment: %v", err)
	}

	// IDs table
	cassQ = fmt.Sprintf("DELETE FROM %s WHERE id=?", cass.db.IdTable())
	err = cass.conn.Query(cassQ, name.UniqueIdString()).Exec()

	if err != nil {
		cass.log.Errorf("Error removing ids: %v", err)
	}

	return err
}

/*************** TAG STUBS ************************/
// TODO
func (cass *CassandraIndexer) GetTagsByUid(uniqueId string) (tags repr.SortingTags, metatags repr.SortingTags, err error) {
	return cass.RamIndexer.GetTagsByUid(uniqueId)
}

func (cass *CassandraIndexer) GetTagsByName(name string, page int) (tags indexer.MetricTagItems, err error) {
	return cass.RamIndexer.GetTagsByName(name, page)
}

func (cass *CassandraIndexer) GetTagsByNameValue(name string, value string, page int) (tags indexer.MetricTagItems, err error) {
	return cass.RamIndexer.GetTagsByNameValue(name, value, page)
}

func (cass *CassandraIndexer) GetUidsByTags(key string, tags repr.SortingTags, page int) (uids []string, err error) {
	return cass.RamIndexer.GetUidsByTags(key, tags, page)
}

/************************************************************************/
/**********  Standard Worker Dispatcher JOB   ***************************/
/************************************************************************/
// insert job queue workers
type cassandraIndexerJob struct {
	Cass  *CassandraIndexer
	Stat  repr.StatName
	retry int
}

func (j *cassandraIndexerJob) IncRetry() int {
	j.retry++
	return j.retry
}
func (j *cassandraIndexerJob) OnRetry() int {
	return j.retry
}

func (j *cassandraIndexerJob) DoWork() error {
	err := j.Cass.WriteOne(j.Stat)
	if err != nil {
		j.Cass.log.Error("Insert failed for Index: %v retrying ...", j.Stat)
	}
	return err
}
