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
	THe MySQL indexer.

	Both the Cassandra and MySQL indexers share the same basic table space and methods


CREATE TABLE `{segment_table}` (
  `segment` varchar(255) NOT NULL DEFAULT '',
  `pos` int NOT NULL,
  PRIMARY KEY (`pos`, `segment`)
);

CREATE TABLE `{path_table}` (
  `id` BIGINT unsigned NOT NULL AUTO_INCREMENT,
  `segment` varchar(255) NOT NUL,
  `pos` int NOT NULL,
  `uid` varchar(50) NOT NULL,
  `path` varchar(255) NOT NULL DEFAULT '',
  `length` int NOT NULL,
  `has_data` bool DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `seg_pos` (`segment`, `pos`),
  KEY `uid` (`uid`),
  KEY `length` (`length`)
);


CREATE TABLE `{id_table}` (
  `id` BIGINT unsigned NOT NULL AUTO_INCREMENT,
  `segment` varchar(255) NOT NUL,
  `pos` int NOT NULL,
  `uid` varchar(50) NOT NULL,
  `path` varchar(255) NOT NULL DEFAULT '',
  `length` int NOT NULL,
  `has_data` bool DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `seg_pos` (`segment`, `pos`),
  KEY `uid` (`uid`),
  KEY `length` (`length`)
);

CREATE TABLE `{tagTable}` (
  `id` BIGINT unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `value` varchar(255) NOT NULL,
  `is_meta` tinyint(1) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `name` (`name`),
  UNIQUE KEY `uid` (`value`, `name`, `is_meta`)
);

CREATE TABLE `{tagTable}_xref` (
  `tag_id` BIGINT unsigned,
  `uid` varchar(50) NOT NULL,
  PRIMARY KEY (`tag_id`, `uid`)
);



*/

package indexer

import (
	"cadent/server/dispatch"
	"cadent/server/schemas/indexer"
	"cadent/server/schemas/repr"
	"cadent/server/stats"
	"cadent/server/utils"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/dbs"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/net/context"
	logging "gopkg.in/op/go-logging.v1"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	MYSQL_INDEXER_QUEUE_LEN = 1024 * 1024
	MYSQL_INDEXER_WORKERS   = 8
	MYSQL_WRITES_PER_SECOND = 200
)

/****************** Interfaces *********************/
type MySQLIndexer struct {
	RamIndexer

	db        *dbs.MySQLDB
	conn      *sql.DB
	indexerId string

	write_lock sync.Mutex

	numWorkers int
	queueLen   int
	_accept    bool //shtdown notice

	dispatcher *dispatch.DispatchQueue

	writesPerSecond int // rate limit writer
	shouldWrite     bool
	writerIndex     int

	shutitdown uint32
	shutdown   chan bool
	startstop  utils.StartStop

	// tag ID cache so we don't do a bunch of unessesary inserts
	tagIdCache *TagCache
	indexCache *IndexReadCache

	log *logging.Logger
}

func NewMySQLIndexer() *MySQLIndexer {
	my := new(MySQLIndexer)
	my.log = logging.MustGetLogger("indexer.mysql")
	my.RamIndexer = *NewRamIndexer()
	return my
}

func (my *MySQLIndexer) Config(conf *options.Options) (err error) {
	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("`dsn` (user:pass@tcp(host:port)/db) is needed for mysql config")
	}

	// url parse so that the password is not in the name
	parsed, _ := url.Parse("mysql://" + dsn)
	host := parsed.Host + parsed.Path

	my.indexerId = conf.String("name", "indexer:mysql:"+host)

	err = my.RamIndexer.ConfigRam(conf)
	if err != nil {
		my.log.Critical(err.Error())
		return err
	}

	// reg ourselves before try to get conns
	my.log.Noticef("Registering indexer: %s", my.Name())
	err = RegisterIndexer(my.Name(), my)
	if err != nil {
		return err
	}

	db, err := dbs.NewDB("mysql", dsn, conf)
	if err != nil {
		return err
	}

	my.db = db.(*dbs.MySQLDB)
	my.conn = db.Connection().(*sql.DB)

	// tweak queues and worker sizes
	my.numWorkers = int(conf.Int64("write_workers", MYSQL_INDEXER_WORKERS))
	my.queueLen = int(conf.Int64("write_queue_ength", MYSQL_INDEXER_QUEUE_LEN))

	my.cache, err = getCacherSingleton(my.indexerId)
	if err != nil {
		return err
	}

	my.writesPerSecond = int(conf.Int64("writes_per_second", MYSQL_WRITES_PER_SECOND))
	my.cache.maxKeys = int(conf.Int64("cache_index_size", CACHER_METRICS_KEYS))
	my.shouldWrite = conf.Bool("should_write", true)
	i, err := conf.Int64Required("writer_index")
	if err != nil {
		return err
	}
	my.writerIndex = int(i)

	atomic.StoreUint32(&my.shutitdown, 1)
	my.shutdown = make(chan bool)

	return RegisterIndexer(my.Name(), my)
}

// Name
func (my *MySQLIndexer) Name() string { return my.indexerId }

// ShouldWrite based on the `should_write` option, if false, just populate the ram caches
func (my *MySQLIndexer) ShouldWrite() bool { return my.shouldWrite }

func (my *MySQLIndexer) Start() {
	my.startstop.Start(func() {
		my.log.Notice("starting mysql indexer: %s/%s/%s", my.db.PathTable(), my.db.SegmentTable(), my.db.TagTable())
		err := NewMySQLIndexSchema(my.conn, my.db.SegmentTable(), my.db.PathTable(), my.db.TagTable(), my.db.IdTable()).AddIndexTables()
		if err != nil {
			panic(err)
		}
		my.RamIndexer.Start()
		retries := 2
		my.dispatcher = dispatch.NewDispatchQueue(my.numWorkers, my.queueLen, retries)
		my.dispatcher.Start()
		my.cache.Start() //start cacher
		my.shutitdown = 0

		go my.sendToWriters() // the dispatcher
	})
}

func (my *MySQLIndexer) Stop() {
	my.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()

		my.log.Notice("shutting down mysql indexer: %s/%s/%s", my.db.PathTable(), my.db.SegmentTable(), my.db.TagTable())
		my.cache.Stop()
		my.RamIndexer.Stop()

		my.shutitdown = 1
	})
}

// pop from the cache and send to actual writers
func (my *MySQLIndexer) sendToWriters() error {
	// this may not be the "greatest" ratelimiter of all time,
	// as "high frequency tickers" can be costly .. but should the workers get backedup
	// it will block on the write_queue stage

	if my.writesPerSecond <= 0 {
		my.log.Notice("Starting indexer writer: No rate limiting enabled")
		for {
			if my.shutitdown == 1 {
				return nil
			}
			skey := my.cache.Pop()
			switch skey.IsBlank() {
			case true:
				time.Sleep(time.Second)
			default:
				stats.StatsdClient.Incr(fmt.Sprintf("indexer.mysql.write.send-to-writers"), 1)
				my.dispatcher.Add(&mysqlIndexerJob{Msql: my, Stat: skey})
			}
		}
	} else {
		sleepT := float64(time.Second) * (time.Second.Seconds() / float64(my.writesPerSecond))
		my.log.Notice("Starting indexer writer: limiter every %f nanoseconds (%d writes per second)", sleepT, my.writesPerSecond)
		dur := time.Duration(int(sleepT))
		for {
			if my.shutitdown == 1 {
				return nil
			}
			skey := my.cache.Pop()
			switch skey.IsBlank() {
			case true:
				time.Sleep(time.Second)
			default:
				stats.StatsdClient.Incr(fmt.Sprintf("indexer.mysql.write.send-to-writers"), 1)
				my.dispatcher.Add(&mysqlIndexerJob{Msql: my, Stat: skey})
				time.Sleep(dur)
			}
		}
	}
}

// this writes an "update" to the id table, we don't do this every write, just periodically
// otherwise we'd kill cassandra w/ a bunch of silly updates
func (my *MySQLIndexer) writeUpdate(inname repr.StatName) error {
	// only if we need to update it
	//slap in the id -> path table
	tx, err := my.conn.Begin()
	if err != nil {
		my.log.Error("Transaction failure", err)
		stats.StatsdClientSlow.Incr("indexer.mysql.update-failures", 1)
		return err
	}

	uid := inname.UniqueIdString()
	Q := fmt.Sprintf(`INSERT INTO %s (id, path, writer_index, tags, metatags, lastseen) VALUES (?,?,?,?,?,NOW()) ON DUPLICATE KEY UPDATE lastseen=NOW()`, my.db.IdTable())

	_, err = tx.Exec(Q,
		uid, inname.Key, my.writerIndex, inname.Tags.StringList(), inname.MetaTags.StringList(),
	)
	if err != nil {
		my.log.Error("Id/path commit failed: %v", err)
		return err
	}
	err = tx.Commit()
	if err != nil {
		my.log.Error("Id/path commit failed: %v", err)
		return err
	}
	if err != nil {
		my.log.Error("Could not insert id path %v (%s) (%s): %v", uid, inname.Key, Q, err)
		stats.StatsdClientSlow.Incr("indexer.mysql.update-failures", 1)
	} else {
		stats.StatsdClientSlow.Incr("indexer.mysql.update-writes", 1)
	}
	return err
}

// keep an index of the stat keys and their fragments so we can look up
func (my *MySQLIndexer) Write(skey repr.StatName) error {
	go func() {
		if my.RamIndexer.NeedsWrite(skey, CASSANDRA_HIT_LAST_UPDATE) {
			my.RamIndexer.Write(skey) //add to ram index
			if my.ShouldWrite() {
				my.writeUpdate(skey)
			}
		}
	}()
	if my.ShouldWrite() {
		return my.cache.Add(skey)
	}
	return nil
}

func (my *MySQLIndexer) WriteTags(inname *repr.StatName, do_main bool, do_meta bool) error {

	have_meta := !inname.MetaTags.IsEmpty()
	have_tgs := !inname.Tags.IsEmpty()
	if !have_tgs && !have_meta || (!do_main && !do_meta) {
		return nil
	}

	// be warned .. found this out the hard way .. the transaction lock interfears w/ this lock
	// so it must go first
	tx, err := my.conn.Begin()
	if err != nil {
		my.log.Error("Failure in Index Write getting transations: %v", tx)
		return err
	}

	unique_ID := inname.UniqueIdString()

	// Tag Time
	tagQ := fmt.Sprintf(
		"INSERT INTO %s (name, value, is_meta) VALUES (?, ?, ?)",
		my.db.TagTable(),
	)

	var tagIds []string
	if have_tgs && do_main {

		for _, tag := range inname.Tags.Tags() {
			cId, cMeta := my.inTagCache(tag.Name, tag.Value)
			if len(cId) > 0 && cMeta == false {
				tagIds = append(tagIds, cId)
			} else {
				res, err := tx.Exec(tagQ, tag.Name, tag.Value, false)
				if err != nil {
					if strings.Contains(fmt.Sprintf("%s", err), "Duplicate entry") {
						id, err := my.FindTagId(tag.Name, tag.Value, false)
						if err == nil {
							tagIds = append(tagIds, id)
							continue
						}
					}

					my.log.Error("Could not write tag %v: %v", tag, err)
					continue
				}
				id, err := res.LastInsertId()
				if err != nil {
					my.log.Error("Could not get insert tag ID %v: %v", tag, err)
					continue
				}
				idStr := fmt.Sprintf("%d", id)
				tagIds = append(tagIds, idStr)
				my.tagIdCache.Add(tag.Name, tag.Value, false, idStr)
			}
		}
	}

	if have_meta && do_meta {
		for _, tag := range inname.MetaTags.Tags() {
			cId, cMeta := my.inTagCache(tag.Name, tag.Value)
			if len(cId) > 0 && cMeta == true {
				tagIds = append(tagIds, cId)
			} else {
				res, err := tx.Exec(tagQ, tag.Name, tag.Value, true)
				if err != nil {
					if strings.Contains(fmt.Sprintf("%s", err), "Duplicate entry") {
						id, err := my.FindTagId(tag.Name, tag.Value, true)
						if err == nil {
							tagIds = append(tagIds, id)
							continue
						}
					}
					my.log.Error("Could not write tag %v: %v", tag, err)
					continue
				}
				id, err := res.LastInsertId()
				if err != nil {
					my.log.Error("Could not get insert tag ID %v: %v", tag, err)
					continue
				}
				idStr := fmt.Sprintf("%d", id)
				tagIds = append(tagIds, idStr)
				my.tagIdCache.Add(tag.Name, tag.Value, true, idStr)

			}
		}
	}

	if len(tagIds) > 0 {
		tagQxr := fmt.Sprintf(
			"INSERT IGNORE INTO %s (tag_id, uid) VALUES ",
			my.db.TagTableXref(),
		)
		var val_q []string
		var q_bits []interface{}
		for _, tg := range tagIds {
			val_q = append(val_q, "(?,?)")
			q_bits = append(q_bits, []interface{}{tg, unique_ID}...)
		}
		_, err := tx.Exec(tagQxr+strings.Join(val_q, ","), q_bits...)
		if err != nil {
			my.log.Error("Could not get insert tag UID-ID: %v", err)
		}
	}
	err = tx.Commit()
	if err != nil {
		my.log.Error("Tag commit failed: %v", err)
		return err
	}
	// and we are done
	return nil
}

// a basic clone of the cassandra indexer
func (my *MySQLIndexer) WriteOne(inname *repr.StatName) error {
	defer stats.StatsdSlowNanoTimeFunc(fmt.Sprintf("indexer.mysql.write.path-time-ns"), time.Now())
	stats.StatsdClientSlow.Incr("indexer.mysql.noncached-writes-path", 1)

	skey := inname.Key
	unique_ID := inname.UniqueIdString()
	pth := NewParsedPath(skey, unique_ID)

	// we are going to assume that if the path is already in the system, we've indexed it and therefore
	// do not need to do the super loop (which is very expensive)
	SelQ := fmt.Sprintf(
		"SELECT uid FROM %s WHERE has_data=? AND segment=? AND path=? LIMIT 1",
		my.db.PathTable(),
	)

	if rows, gerr := my.conn.Query(SelQ, true, skey, skey); gerr == nil {
		// already indexed
		var uid string
		defer rows.Close()
		for rows.Next() {
			if err := rows.Scan(&uid); err == nil && uid == unique_ID {
				return my.WriteTags(inname, false, true)
			}
		}
	}

	// begin the big transation
	tx, err := my.conn.Begin()
	if err != nil {
		my.log.Error("Failure in Index Write getting transations: %v", tx)
		return err
	}

	last_path := pth.Last()
	// now to upsert them all (inserts in cass are upserts)
	for idx, seg := range pth.Segments {
		Q := fmt.Sprintf(
			"INSERT IGNORE INTO %s (pos, segment) VALUES  (?, ?) ",
			my.db.SegmentTable(),
		)
		_, err := tx.Exec(Q, seg.Pos, seg.Segment)

		if err != nil {
			my.log.Error("Could not insert segment %v", seg)
			stats.StatsdClientSlow.Incr("indexer.mysql.segment-failures", 1)
		} else {
			stats.StatsdClientSlow.Incr("indexer.mysql.segment-writes", 1)
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

		*/
		Q = fmt.Sprintf(
			"INSERT IGNORE INTO %s (segment, pos, path, uid, length, has_data) VALUES  (?, ?, ?, ?, ?, ?)",
			my.db.PathTable(),
		)

		if skey != seg.Segment && idx < pth.Len-1 {
			_, err = tx.Exec(Q,
				seg.Segment, seg.Pos, seg.Segment+"."+pth.Parts[idx+1], "", seg.Pos+1, false,
			)
		} else {
			// now add the has data path as well
			_, err = tx.Exec(Q,
				seg.Segment, seg.Pos, skey, unique_ID, pth.Len-1, true,
			)
		}

		if err != nil {
			my.log.Error("Could not insert path %v (%v) :: %v", last_path, unique_ID, err)
			stats.StatsdClientSlow.Incr("indexer.mysql.path-failures", 1)
		} else {
			stats.StatsdClientSlow.Incr("indexer.mysql.path-writes", 1)
		}
	}
	if err != nil {
		my.log.Error("Could not write index", err)
	}

	err = my.WriteTags(inname, true, true)
	if err != nil {
		my.log.Error("Could not write tag index", err)
		return err
	}
	err = tx.Commit()

	return err
}

func (my *MySQLIndexer) Delete(name *repr.StatName) error {

	my.RamIndexer.Delete(name)

	pthQ := fmt.Sprintf(
		"DELETE FROM %s WHERE uid=? AND path=? AND length=?",
		my.db.PathTable(),
	)

	tagQ := fmt.Sprintf(
		"DELETE FROM %s WHERE uid=?",
		my.db.TagTableXref(),
	)

	uid := name.UniqueIdString()
	pvals := []interface{}{uid, name.Key, len(strings.Split(name.Key, "."))}

	tx, err := my.conn.Begin()
	if err != nil {
		my.log.Error("Mysql Driver: Delete Path failed, %v", err)
		return err
	}

	//format all vals at once
	_, err = tx.Exec(pthQ, pvals...)
	if err != nil {
		my.log.Error("Mysql Driver: Delete Path failed, %v", err)
		return err
	}

	//format all vals at once
	_, err = tx.Exec(tagQ, uid)
	if err != nil {
		my.log.Error("Mysql Driver: Delete Tags failed, %v", err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

/**** READER ***/

func (my *MySQLIndexer) ExpandNonRegex(metric string) (indexer.MetricExpandItem, error) {
	paths := strings.Split(metric, ".")
	mLen := len(paths)
	cass_Q := fmt.Sprintf(
		"SELECT segment FROM %s WHERE pos=? AND segment=?",
		my.db.SegmentTable(),
	)

	var onPth string
	var me indexer.MetricExpandItem

	rows, err := my.conn.Query(cass_Q, mLen-1, metric)
	if err != nil {
		return me, err
	}

	for rows.Next() {
		// just grab the "n+1" length ones
		err = rows.Scan(&onPth)
		if err != nil {
			return me, err
		}
		me.Results = append(me.Results, onPth)
	}
	if err := rows.Close(); err != nil {
		return me, err
	}
	return me, nil
}

// Expand simply pulls out any regexes into full form
func (my *MySQLIndexer) Expand(metric string) (indexer.MetricExpandItem, error) {

	defer stats.StatsdSlowNanoTimeFunc("indexer.mysql.expand.get-time-ns", time.Now())

	needs_regex := needRegex(metric)
	//cass.log.Debug("REGSS: %v, %s", needs_regex, metric)

	if !needs_regex {
		return my.ExpandNonRegex(metric)
	}
	paths := strings.Split(metric, ".")
	mLen := len(paths)

	var me indexer.MetricExpandItem

	the_reg, err := regifyKey(metric)

	if err != nil {
		return me, err
	}
	cassQ := fmt.Sprintf(
		"SELECT segment FROM %s WHERE pos=?",
		my.db.SegmentTable(),
	)

	rows, err := my.conn.Query(cassQ, mLen-1)
	if err != nil {
		return me, err
	}
	var seg string
	for rows.Next() {
		// just grab the "n+1" length ones
		err = rows.Scan(&seg)
		if err != nil {
			return me, err
		}
		//cass.log.Debug("SEG: %s", seg)

		if !the_reg.MatchString(seg) {
			continue
		}
		me.Results = append(me.Results, seg)
	}
	if err := rows.Close(); err != nil {
		return me, err
	}
	return me, nil

}

/******************* TAG METHODS **********************************/
func (my *MySQLIndexer) inTagCache(name string, value string) (tagId string, ismeta bool) {

	got := my.tagIdCache.Get(name, value, false)

	if len(got) > 0 {
		return got, false
	}
	got = my.tagIdCache.Get(name, value, true)
	if len(got) > 0 {
		return got, true
	}

	return "", false
}

func (my *MySQLIndexer) FindTagId(name string, value string, ismeta bool) (string, error) {

	// see if in the writer tag cache
	cId, cMeta := my.inTagCache(name, value)
	if ismeta == cMeta && len(cId) > 0 {
		return cId, nil
	}

	selTagQ := fmt.Sprintf(
		"SELECT id FROM %s WHERE name=? AND value=? AND is_meta=?",
		my.db.TagTable(),
	)

	rows, err := my.conn.Query(selTagQ, name, value, ismeta)
	if err != nil {
		my.log.Error("Error Getting Tags: %s : %v", selTagQ, err)
		return "", err
	}
	defer rows.Close()
	var id int64
	// only get one entry
	rows.Next()
	err = rows.Scan(&id)
	if err != nil {
		my.log.Error("Error Getting Tags Iterator: %s : %v", selTagQ, err)
		return "", err
	}
	idStr := fmt.Sprintf("%d", id)
	my.tagIdCache.Add(name, value, ismeta, idStr)

	return idStr, nil
}

func (my *MySQLIndexer) GetTagsByUid(unique_id string) (tags repr.SortingTags, metatags repr.SortingTags, err error) {

	tagTable := my.db.TagTable()
	tagXrTable := my.db.TagTableXref()

	tgsQ := fmt.Sprintf(
		"SELECT name,value,is_meta FROM %s WHERE id IN ( SELECT tag_id FROM %s WHERE uid = ? )",
		tagTable, tagXrTable,
	)

	var name string
	var value string
	var isMeta bool
	rows, err := my.conn.Query(tgsQ, unique_id)
	if err != nil {
		my.log.Error("Error Getting Tags: %s : %v", tgsQ, err)
		return
	}
	defer rows.Close()
	oTags := &repr.SortingTags{}
	mTags := &repr.SortingTags{}
	for rows.Next() {
		err = rows.Scan(&name, &value, &isMeta)
		if err != nil {
			my.log.Error("Error Getting Tags Iterator: %s : %v", tgsQ, err)
			continue
		}
		if isMeta {
			mTags.Set(name, value)
		} else {
			oTags.Set(name, value)
		}
	}
	return *oTags, *mTags, err
}

func (my *MySQLIndexer) GetTagsByName(name string, page int) (tags indexer.MetricTagItems, err error) {
	sel_tagQ := fmt.Sprintf(
		"SELECT id, name, value, is_meta FROM %s WHERE name=? LIMIT ?, ?",
		my.db.TagTable(),
	)
	use_name := name

	if needRegex(name) {
		sel_tagQ = fmt.Sprintf(
			"SELECT id, name, value, is_meta FROM %s WHERE name REGEXP ? LIMIT ?, ?",
			my.db.TagTable(),
		)
		use_name = regifyMysqlKeyString(name)
	}

	rows, err := my.conn.Query(sel_tagQ, use_name, page*MAX_PER_PAGE, MAX_PER_PAGE)
	if err != nil {
		my.log.Error("Error Getting Tags: %s : %v", sel_tagQ, err)
		return tags, err
	}
	var tname string
	var id int64
	var isMeta bool
	var value string
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&id, &tname, &value, &isMeta)
		if err != nil {
			my.log.Error("Error Getting Tags Iterator: %s : %v", sel_tagQ, err)
			return
		}
		idStr := fmt.Sprintf("%d", id)
		tags = append(tags, &indexer.MetricTagItem{Name: tname, Value: value, Id: idStr, IsMeta: isMeta})
		my.tagIdCache.Add(tname, value, isMeta, idStr)
	}
	return
}

func (my *MySQLIndexer) GetTagsByNameValue(name string, value string, page int) (tags indexer.MetricTagItems, err error) {
	selTagQ := fmt.Sprintf(
		"SELECT id, name, value, is_meta FROM %s WHERE name=? AND value=? LIMIT ?, ?",
		my.db.TagTable(),
	)
	useName := name
	useVal := value

	if needRegex(name) {
		return tags, fmt.Errorf("only the `value` can be a regex for tags by name and value")
	}

	if needRegex(value) {
		selTagQ = fmt.Sprintf(
			"SELECT id, name, value, is_meta FROM %s WHERE name=? AND value REGEXP ? LIMIT ?, ?",
			my.db.TagTable(),
		)
		useVal = regifyMysqlKeyString(value)
	} else {
		// see if in cache
		cId, cMeta := my.inTagCache(name, value)
		if len(cId) > 0 {
			tags = append(tags, &indexer.MetricTagItem{Name: name, Value: value, Id: cId, IsMeta: cMeta})
			return tags, nil
		}

	}
	//my.log.Errorf("Q: %v | %s, %s", sel_tagQ, use_name, use_val)
	rows, err := my.conn.Query(selTagQ, useName, useVal, page*MAX_PER_PAGE, MAX_PER_PAGE)
	if err != nil {
		my.log.Error("Error Getting Tags: %s : %v", selTagQ, err)
		return tags, err
	}
	var tname string
	var id int64
	var isMeta bool
	var tvalue string
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&id, &tname, &tvalue, &isMeta)
		if err != nil {
			my.log.Error("Error Getting Tags Iterator: %s : %v", selTagQ, err)
			return
		}
		idStr := fmt.Sprintf("%d", id)
		tags = append(tags, &indexer.MetricTagItem{Name: tname, Value: tvalue, Id: idStr, IsMeta: isMeta})

		// add to caches
		my.tagIdCache.Add(tname, tvalue, isMeta, idStr)

	}
	return
}

func (my *MySQLIndexer) GetUidsByTags(key string, tags repr.SortingTags, page int) (uids []string, err error) {

	tags_idsQ := fmt.Sprintf(
		"SELECT DISTINCT id FROM %s WHERE name=? AND value=?",
		my.db.TagTable(),
	)
	var q_params []interface{}

	sel_tagQ := fmt.Sprintf(
		"SELECT uid FROM %s WHERE tag_id = ? ",
		my.db.TagTableXref(),
	)

	nested_q := fmt.Sprintf(
		" (SELECT uid FROM %s WHERE tag_id = ?) ",
		my.db.TagTableXref(),
	)

	// add the key query if there
	key_q := ""
	if len(key) > 0 {
		key_q = fmt.Sprintf(
			"SELECT DISTINCT uid FROM %s WHERE segment=? AND has_data=1 AND uid IN ",
			my.db.PathTable(),
		)
		q_params = append(q_params, key)
	}

	for idx, ontag := range tags {
		rows, err := my.conn.Query(tags_idsQ, ontag.Name, ontag.Value)
		if err != nil {
			my.log.Error("Error Getting Uid: %v", err)
			return uids, err
		}
		defer rows.Close()
		var tid int64
		for rows.Next() {
			err = rows.Scan(&tid)
			if err != nil {
				my.log.Error("Error Getting Tags Id: %s : %v", tags_idsQ, tid)
				return uids, err
			}
			q_params = append(q_params, tid)
			if idx > 0 {
				sel_tagQ += " AND uid IN (" + nested_q + ")"
			}
		}
	}

	if len(q_params) == 0 {
		return
	}

	q_params = append(q_params, page*MAX_PER_PAGE)
	q_params = append(q_params, MAX_PER_PAGE)

	if len(key_q) > 0 {
		sel_tagQ = key_q + " (" + sel_tagQ + ")"
	}
	sel_tagQ += " LIMIT ?, ? "

	rows, err := my.conn.Query(sel_tagQ, q_params...)
	if err != nil {
		my.log.Error("Error Getting Uid: %v", err)
		return uids, err
	}
	defer rows.Close()

	var uid string
	for rows.Next() {
		err = rows.Scan(&uid)
		if err != nil {
			my.log.Error("Error Getting Tags Uid: %s : %v", sel_tagQ, err)
			return
		}
		uids = append(uids, uid)
	}
	return

}

/********************* UID metric finders ***********************/

// List all paths w/ data
func (my *MySQLIndexer) List(hasData bool, page int) (indexer.MetricFindItems, error) {

	defer stats.StatsdSlowNanoTimeFunc("indexer.mysql.list.get-time-ns", time.Now())

	// grab all the paths that match this length if there is no regex needed
	// these are the "data/leaf" nodes .. alow or uid lookups too
	pathQ := fmt.Sprintf(
		"SELECT uid,path FROM %s WHERE has_data=? AND segment=path LIMIT ?,?",
		my.db.PathTable(),
	)

	var mt indexer.MetricFindItems
	var onPth string
	var id string
	rows, err := my.conn.Query(pathQ, hasData, page*MAX_PER_PAGE, MAX_PER_PAGE)
	if err != nil {
		return mt, err
	}
	for rows.Next() {
		ms := new(indexer.MetricFindItem)

		err = rows.Scan(&id, &onPth)
		if err != nil {
			return mt, err
		}

		spl := strings.Split(onPth, ".")

		ms.Text = spl[len(spl)-1]
		ms.Id = onPth
		ms.Path = onPth

		ms.Expandable = 0
		ms.Leaf = 1
		ms.AllowChildren = 0
		ms.UniqueId = id

		mt = append(mt, ms)
	}

	if err := rows.Close(); err != nil {
		return mt, err
	}

	return mt, nil
}

// basic find for non-regex items
func (my *MySQLIndexer) FindNonRegex(metric string, tags repr.SortingTags, exact bool) (indexer.MetricFindItems, error) {

	defer stats.StatsdSlowNanoTimeFunc("indexer.mysql.findnoregex.get-time-ns", time.Now())

	// check cache
	items := my.indexCache.Get(metric, tags)
	if items != nil {
		stats.StatsdClientSlow.Incr("indexer.mysql.findnoregex.cached", 1)
		return *items, nil
	}

	// since there are and regex like things in the strings, we
	// need to get the all "items" from where the regex starts then hone down

	paths := strings.Split(metric, ".")
	mLen := len(paths)

	// grab all the paths that match this length if there is no regex needed
	// these are the "data/leaf" nodes .. allow or uid lookups too
	pathQ := fmt.Sprintf(
		"SELECT uid,path,length,has_data FROM %s WHERE (pos=? AND segment=?) OR (uid = ? AND segment = path AND has_data=1)",
		my.db.PathTable(),
	)

	args := []interface{}{
		mLen - 1, metric, metric,
	}

	// if "tags" we need to find the tag Ids, and do the cross join
	tagIds := make([]string, 0)
	having_ct := 0
	for _, tag := range tags {
		tIds, _ := my.GetTagsByNameValue(tag.Name, tag.Value, 0)
		having_ct += 1 // need this as the value may be a regex
		for _, tg := range tIds {
			tagIds = append(tagIds, tg.Id)
		}
	}

	if len(tagIds) > 0 {
		limitQ := ""
		qend := ""
		// support non-target tag only things, tags only are "data only"
		// but we need to have a fancier query
		if len(metric) == 0 {

			pathQ = fmt.Sprintf(
				"SELECT uid,path,length,has_data FROM %s "+
					" WHERE has_data=1 AND segment = path AND "+
					" uid IN (SELECT DISTINCT uid FROM %s WHERE %s.tag_id IN ",
				my.db.PathTable(), my.db.TagTableXref(), my.db.TagTableXref(),
			)
			args = []interface{}{}
			qend = fmt.Sprintf(
				" GROUP BY %s.uid HAVING COUNT(%s.tag_id) = %d ",
				my.db.TagTableXref(), my.db.TagTableXref(), having_ct,
			)
			limitQ = " LIMIT ?,?"
		} else {

			pathQ = fmt.Sprintf(
				"SELECT uid,path,length,has_data FROM %s "+
					" WHERE ((pos=? AND segment=?) OR (uid = ? AND segment = path AND has_data=1)) AND"+
					" uid IN (SELECT DISTINCT uid FROM %s WHERE %s.tag_id IN ",
				my.db.PathTable(), my.db.TagTableXref(), my.db.TagTableXref(),
			)
			qend = fmt.Sprintf(
				" GROUP BY %s.uid HAVING COUNT(%s.tag_id) = %d ",
				my.db.TagTableXref(), my.db.TagTableXref(), having_ct,
			)
			limitQ = " LIMIT ?,?"

		}
		pathQ += "("
		for idx, tgid := range tagIds {
			pathQ += "?"
			if idx != len(tagIds)-1 {
				pathQ += ","
			}
			args = append(args, tgid)
		}
		pathQ += ")" + qend + ")"
		if len(limitQ) > 0 {
			pathQ += limitQ
			args = append(args, 0)
			args = append(args, MAX_PER_PAGE)
		}
	}

	var mt indexer.MetricFindItems
	var onPth string
	var pthLen int
	var id string
	var hasData bool

	if exact {
		pathQ += " LIMIT 1"
	}

	rows, err := my.conn.Query(pathQ, args...)
	if err != nil {
		return mt, err
	}
	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&id, &onPth, &pthLen, &hasData)
		if err != nil {
			return mt, err
		}
		// mlen 1 == uid lookup
		if mLen != 1 && pthLen > mLen {
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
				ms.Tags, ms.MetaTags, _ = my.GetTagsByUid(id)
				mt = append(mt, ms)
			} else {
				ms.Expandable = 1
				ms.Leaf = 0
				ms.AllowChildren = 1
				mt = append(mt, ms)
			}
			my.indexCache.Add(metric, tags, &mt)
			return mt, nil

		}

		spl := strings.Split(onPth, ".")

		ms.Text = spl[len(spl)-1]
		ms.Id = onPth
		ms.Path = onPth

		//if the incoming "metric" matches the segment

		if hasData {
			ms.Expandable = 0
			ms.Leaf = 1
			ms.AllowChildren = 0
			ms.UniqueId = id

			// grab ze tags
			ms.Tags, ms.MetaTags, _ = my.GetTagsByUid(id)

		} else {
			ms.Expandable = 1
			ms.Leaf = 0
			ms.AllowChildren = 1
		}

		mt = append(mt, ms)
	}

	// set it
	my.indexCache.Add(metric, tags, &mt)

	return mt, nil
}

// special case for "root" == "*" finder
func (my *MySQLIndexer) FindRoot() (indexer.MetricFindItems, error) {
	defer stats.StatsdSlowNanoTimeFunc("indexer.mysql.findroot.get-time-ns", time.Now())

	cass_Q := fmt.Sprintf(
		"SELECT segment FROM %s WHERE pos=?",
		my.db.SegmentTable(),
	)

	var mt indexer.MetricFindItems
	var seg string

	rows, err := my.conn.Query(cass_Q, 0)
	if err != nil {
		return mt, err
	}
	for rows.Next() {

		err = rows.Scan(&seg)
		if err != nil {
			return mt, err
		}
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

	if err := rows.Close(); err != nil {
		return mt, err

	}

	return mt, nil
}

// FindInCache find the items in the RamIndexer
func (my *MySQLIndexer) FindInCache(ctx context.Context, metric string, tags repr.SortingTags) (indexer.MetricFindItems, error) {
	return my.RamIndexer.Find(ctx, metric, tags)
}

// Find to allow for multiple targets
func (my *MySQLIndexer) Find(ctx context.Context, metric string, tags repr.SortingTags) (indexer.MetricFindItems, error) {
	// the regex case is a bit more complicated as we need to grab ALL the segments of a given length.
	// see if the match the regex, and then add them to the lists since cassandra does not provide regex abilities
	// on the server side
	sp, closer := my.GetSpan("Find", ctx)
	sp.LogKV("driver", "MySQLIndexer", "metric", metric, "tags", tags)
	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("indexer.mysql.find.get-time-ns", time.Now())

	// special case for "root" == "*"

	// check cache
	items := my.indexCache.Get(metric, tags)
	if items != nil {
		stats.StatsdClientSlow.Incr("indexer.mysql.find.cached", 1)
		return *items, nil
	}

	if metric == "*" {
		return my.FindRoot()
	}

	// if we are not a regex, then we're look for "expandable" or "a data node"
	// and so we simply need to find out if the thing is a "segment + data" or not
	needs_regex := needRegex(metric)

	if !needs_regex {
		return my.FindNonRegex(metric, tags, true)
	}

	// convert the "graphite regex" into something golang understands (just the "."s really)
	// need to replace things like "moo*" -> "moo.*" but careful not to do "..*"
	// the "graphite" globs of {moo,goo} we can do with (moo|goo) so convert { -> (, , -> |, } -> )
	the_reg, err := regifyKey(metric)

	paths := strings.Split(metric, ".")
	mLen := len(paths)

	if err != nil {
		return nil, err
	}
	cass_Q := fmt.Sprintf(
		"SELECT segment FROM %s WHERE pos=?",
		my.db.SegmentTable(),
	)

	var mt indexer.MetricFindItems
	rows, err := my.conn.Query(cass_Q, mLen-1)
	if err != nil {
		return mt, err
	}

	var seg string
	for rows.Next() {

		err = rows.Scan(&seg)
		if err != nil {
			return mt, err
		}

		if !the_reg.MatchString(seg) {
			continue
		}
		items, err := my.FindNonRegex(seg, tags, true)
		if err != nil {
			my.log.Warning("could not get segments for `%s` :: %v", seg, err)
			continue
		}
		if items != nil && len(items) > 0 {
			mt = append(mt, items...)
			items = nil
		}

	}

	if err := rows.Close(); err != nil {
		return mt, err
	}

	// set it
	my.indexCache.Add(metric, tags, &mt)

	return mt, nil
}

/************************************************************************/
/**********  Standard Worker Dispatcher JOB   ***************************/
/************************************************************************/
// insert job queue workers
type mysqlIndexerJob struct {
	Msql  *MySQLIndexer
	Stat  repr.StatName
	retry int
}

func (j *mysqlIndexerJob) IncRetry() int {
	j.retry++
	return j.retry
}
func (j *mysqlIndexerJob) OnRetry() int {
	return j.retry
}

func (j *mysqlIndexerJob) DoWork() error {
	err := j.Msql.WriteOne(&j.Stat)
	if err != nil {
		j.Msql.log.Error("Insert failed for Index: %v retrying ...", j.Stat)
	}
	return err
}
