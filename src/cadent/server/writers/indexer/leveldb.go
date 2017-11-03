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

LevelDB key/value store for local saving for indexes on the stat key space
and how it maps to files on disk.

This is much like how Prometheus does things, stores a index in levelDB
and single files for each series.

We have "3" dbs
{segments} -> {position} mappings
{segments} -> {id}:{path} mappings
{tags} -> {id}:{path} mappings
{id} -> {path} mappings

If no tags are used (i.e. graphite like) then the tags will basically be empty

We follow a data pattern a lot like the Cassandra one, as we attempting to get a similar glob matcher
when tags enter (not really done yet) the tag thing will be important

LevelDB is a "key sorted" DB, so we take advantage of the "startswith" (python parlance)
style to walk through the key space and do the typical regex/glob like matches on things

The nice thing about the "." style for things is that we know the desired length of a given metric
so we can do

the segment DB

things like `find.startswith({length}:{longest.non.regex.part}) -> id:path`
and do the regex on the iterator

the segment DB

to do the "expand" parts (basically searching) we have another database
that is {pos}:{segment} -> path

paths have "data" anything else does not and is just a segment

The "path" DB


*/

package indexer

import (
	"bytes"
	"cadent/server/dispatch"
	"cadent/server/schemas/indexer"
	"cadent/server/schemas/repr"
	"cadent/server/stats"
	"cadent/server/utils"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/dbs"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	leveldb_iter "github.com/syndtr/goleveldb/leveldb/iterator"
	leveldb_util "github.com/syndtr/goleveldb/leveldb/util"
	"golang.org/x/net/context"
	logging "gopkg.in/op/go-logging.v1"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	LEVELDB_WRITES_PER_SECOND = 1000
	LEVELDB_INDEXER_QUEUE_LEN = 1024 * 1024
	LEVELDB_INDEXER_WORKERS   = 8
)

/*

Segment index

level DB keys

This one is for "finds"
i.e.  cadent.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99 -> cadent.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99
- SEG:{length}:{segment} -> {id}:{path}

This is used for "expands"
"has data" is when segment+1 == path
i.e cadent.zipperwork.local.writer -> cadent.zipperwork.local.writer.cassandra
- POS:{pos}:{segment}:{has_data} -> {id}:{segment+1}:{has_data} used for expand

for reverse lookups basically for deletion purposes
Second set :: PATH:{path} -> {length}:{segment}


*/

type LevelDBSegment struct {
	Pos         int
	Length      int
	Segment     string
	NextSegment string
	Path        string
	Id          string
}

// given a path get all the various segment datas
func ParsePath(stat_key string, id string) (segments []LevelDBSegment) {

	s_parts := strings.Split(stat_key, ".")
	p_len := len(s_parts)

	cur_part := ""
	next_part := ""

	segments = make([]LevelDBSegment, p_len)

	if p_len > 0 {
		next_part = s_parts[0]
	}

	// for each "segment" add in the path to do a segment to path(s) lookup
	/*
		for key consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99

		Segment -> NextSegment

		consthash -> consthash.zipperwork
		consthash.zipperwork -> consthash.zipperwork.local
		consthash.zipperwork.local -> consthash.zipperwork.local.writer
		consthash.zipperwork.local.writer -> consthash.zipperwork.local.writer.cassandra
		consthash.zipperwork.local.writer.cassandra -> consthash.zipperwork.local.writer.cassandra.write
		consthash.zipperwork.local.writer.cassandra.write -> consthash.zipperwork.local.writer.cassandra.write.metric-time-ns
		consthash.zipperwork.local.writer.cassandra.write.metric-time-ns -> consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99

		Segment -> Path
		consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99 -> consthash.zipperwork.local.writer.cassandra.write.metric-time-ns.upper_99
	*/
	for idx, part := range s_parts {
		if len(cur_part) > 1 {
			cur_part += "."
		}
		cur_part += part
		if idx < p_len && idx > 0 {
			next_part += "."
			next_part += part
		}

		segments[idx] = LevelDBSegment{
			Segment:     cur_part,
			NextSegment: next_part,
			Path:        stat_key,
			Length:      p_len - 1,
			Pos:         idx, // starts at 0
			Id:          id,
		}
	}

	return segments
}

func (ls *LevelDBSegment) SegmentKey(segment string, len int) []byte {
	return []byte(fmt.Sprintf("SEG:%d:%s", len, segment))
}

func (ls *LevelDBSegment) SegmentData() ([]byte, []byte) {
	return []byte(ls.SegmentKey(ls.Segment, ls.Length)), []byte(fmt.Sprintf("%s|%s", ls.Id, ls.Path))
}

func (ls *LevelDBSegment) IdKey(id string) []byte {
	return []byte(id)
}

func (ls *LevelDBSegment) IdData() ([]byte, []byte) {
	return []byte(ls.IdKey(ls.Id)), []byte(ls.Path)
}

func (ls *LevelDBSegment) PathKey(path string) []byte {
	return []byte(path)
}

func (ls *LevelDBSegment) PathData() ([]byte, []byte) {
	return ls.PathKey(ls.Path), []byte(ls.Id)
}

func (ls *LevelDBSegment) ReverseSegmentData() ([]byte, []byte) {
	return []byte(ls.PathKey(ls.Path)), []byte(fmt.Sprintf("%d:%s", ls.Pos, ls.Segment))
}

func (ls *LevelDBSegment) PosSegmentKey(path string, pos int) []byte {
	has_data := "0"
	if ls.NextSegment == ls.Path {
		has_data = "1"
	}
	return []byte(fmt.Sprintf("POS:%d:%s:%s", pos, path, has_data))
}

// POS:{pos}:{segment} -> {segment+1}:{has_data}
func (ls *LevelDBSegment) PosSegmentData() ([]byte, []byte) {
	has_data := "0"
	use_id := "" //paths w/ no data have no id
	if ls.NextSegment == ls.Path {
		has_data = "1"
		use_id = ls.Id
	}
	return []byte(ls.PosSegmentKey(ls.Segment, ls.Pos)),
		[]byte(fmt.Sprintf("%s:%s:%s", use_id, ls.NextSegment, has_data))

}

// if the full path and segment are the same ..
func (ls *LevelDBSegment) HasData() bool {
	return ls.Segment == ls.Path
}
func (ls *LevelDBSegment) InsertAllIntoBatch(batch *leveldb.Batch) {
	if ls.HasData() {
		k, v := ls.SegmentData()
		batch.Put(k, v)
		k1, v1 := ls.ReverseSegmentData()
		batch.Put(k1, v1)
	}
	k, v := ls.PosSegmentData()
	batch.Put(k, v)
}

func (ls *LevelDBSegment) InsertAll(segdb *leveldb.DB, pathdb *leveldb.DB, uiddb *leveldb.DB) (err error) {

	batch := new(leveldb.Batch)
	ls.InsertAllIntoBatch(batch)
	err = segdb.Write(batch, nil)
	if err != nil {
		return err
	}

	idbatch := new(leveldb.Batch)
	k1, v1 := ls.IdData()
	idbatch.Put(k1, v1)
	err = uiddb.Write(idbatch, nil)
	if err != nil {
		return err
	}

	pathbatch := new(leveldb.Batch)
	k2, v2 := ls.PathData()
	pathbatch.Put(k2, v2)
	err = pathdb.Write(pathbatch, nil)

	return
}

func (ls *LevelDBSegment) DeletePath(segdb *leveldb.DB) (err error) {
	// first see if the path is there

	v_byte := []byte(ls.Path)
	pos_byte := []byte(ls.Path + ":1") // rm has_data nodes only
	path_key := ls.PathKey(ls.Path)

	val, err := segdb.Get(path_key, nil)
	if err != nil {
		return err
	}
	if len(val) == 0 {
		return fmt.Errorf("Path is not present")
	}

	// return val is {length}:{segment}
	// log.Printf("DEL: GotPath: Path: %s :: data: %s", path_key, val)

	// grab all the segments
	segs := ParsePath(string(ls.Path), ls.Id)
	l_segs := len(segs)
	errs := make([]error, 0)
	// remove all things that point to this path
	for idx, seg := range segs {
		// only the "last" segment has a value, the sub lines are "POS:..."
		if l_segs == idx+1 {
			// remove SEG:len:...
			seg_key := seg.SegmentKey(seg.Segment, seg.Length)
			// log.Printf("To DEL: Segment: %s", seg_key)
			v, err := segdb.Get(seg_key, nil)
			// log.Printf("To DEL: Segment: Error %v", err)
			if err != nil {
				errs = append(errs, err)
			} else if bytes.EqualFold(v, v_byte) {
				//EqualFold as these are strings at their core
				// log.Printf("Deleting Segment: %s", v_byte)
				segdb.Delete(seg_key, nil)
			}
		}

		// remove the POS:len:... ones as well
		pos_key := seg.PosSegmentKey(seg.Segment, seg.Pos)
		v, err := segdb.Get(pos_key, nil)
		// log.Printf("To DEL: Pos: %s: Error %v", pos_key, err)
		if err != nil {
			errs = append(errs, err)
		} else if bytes.EqualFold(v, pos_byte) { // remove the path only if it is a "data" node ({path}:1)
			// log.Printf("Deleting Pos: %s", pos_byte)
			segdb.Delete(pos_key, nil)
		}
	}
	if len(errs) == 0 {
		// log.Printf("Deleting Path: %s", path_key)
		segdb.Delete(path_key, nil)
	} else {
		return fmt.Errorf("Multiple errors trying to remove %s : %v", ls.Path, errs)
	}

	return nil
}

/*
 Tag index

 TODO: not yet in action yo
*/
type LevelDBTag struct {
	Key    string
	Value  string
	Path   string
	Length int
}

// the singleton
var _LEVELDB_CACHER_SINGLETON map[string]*Cacher
var _leveldb_cacher_mutex sync.Mutex

func _leveldb_get_cacher_signelton(nm string) (*Cacher, error) {
	_leveldb_cacher_mutex.Lock()
	defer _leveldb_cacher_mutex.Unlock()

	if val, ok := _LEVELDB_CACHER_SINGLETON[nm]; ok {
		return val, nil
	}

	cacher := NewCacher()
	_LEVELDB_CACHER_SINGLETON[nm] = cacher
	return cacher, nil
}

// special onload init
func init() {
	_LEVELDB_CACHER_SINGLETON = make(map[string]*Cacher)
}

/****************** Interfaces *********************/
type LevelDBIndexer struct {
	TracerIndexer

	db              *dbs.LevelDB
	indexerId       string
	cache           *Cacher // simple cache to rate limit and buffer writes
	writesPerSecond int     // rate limit writer

	writeQueue      chan dispatch.IJob
	dispatchQueue   chan chan dispatch.IJob
	writeDispatcher *dispatch.Dispatch

	numWorkers int
	queueLen   int
	_accept    bool //shtdown notice
	startstop  utils.StartStop

	log *logging.Logger
}

func NewLevelDBIndexer() *LevelDBIndexer {
	lb := new(LevelDBIndexer)
	lb._accept = true
	lb.log = logging.MustGetLogger("indexer.leveldb")
	return lb
}

func (lb *LevelDBIndexer) Config(conf *options.Options) (err error) {
	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("Indexer: `dsn` (/path/to/dbs) is needed for leveldb config")
	}

	// reg ourselves before try to get conns
	lb.indexerId = conf.String("name", "leveldb:"+dsn)
	lb.log.Noticef("Registering indexer: %s", lb.Name())
	err = RegisterIndexer(lb.Name(), lb)
	if err != nil {
		return err
	}

	db, err := dbs.NewDB("leveldb", dsn, conf)
	if err != nil {
		return err
	}

	lb.db = db.(*dbs.LevelDB)
	if err != nil {
		return err
	}

	lb.cache, err = _leveldb_get_cacher_signelton(dsn)
	if err != nil {
		return err
	}

	lb.cache.maxKeys = int(conf.Int64("cache_index_size", CACHER_METRICS_KEYS))
	lb.writesPerSecond = int(conf.Int64("writes_per_second", LEVELDB_WRITES_PER_SECOND))
	lb.numWorkers = int(conf.Int64("write_workers", LEVELDB_INDEXER_WORKERS))
	lb.queueLen = int(conf.Int64("write_queue_length", LEVELDB_INDEXER_QUEUE_LEN))
	return nil
}

func (lp *LevelDBIndexer) Name() string {
	return lp.indexerId
}

func (lp *LevelDBIndexer) Stop() {
	lp.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()
		if !lp._accept {
			return
		}
		lp.log.Notice("Shutting down LevelDB indexer: %s", lp.Name())
		lp._accept = false
		lp.cache.Stop()
		if lp.writeQueue != nil {
			lp.writeDispatcher.Shutdown()
		}
		lp.db.Close()
	})
}

// ShouldWrite always true
func (lp *LevelDBIndexer) ShouldWrite() bool {
	return true
}

func (lp *LevelDBIndexer) Start() {
	lp.startstop.Start(func() {
		lp.log.Notice("Starting LevelDB indexer: %s", lp.Name())
		workers := lp.numWorkers
		lp.writeQueue = make(chan dispatch.IJob, lp.queueLen)
		lp.dispatchQueue = make(chan chan dispatch.IJob, workers)
		lp.writeDispatcher = dispatch.NewDispatch(workers, lp.dispatchQueue, lp.writeQueue)
		lp.writeDispatcher.SetRetries(2)
		lp.writeDispatcher.Run()
		lp.cache.Start()
		go lp.sendToWriters() // the dispatcher
	})
}

// pop from the cache and send to actual writers
func (lp *LevelDBIndexer) sendToWriters() error {
	// this may not be the "greatest" ratelimiter of all time,
	// as "high frequency tickers" can be costly .. but should the workers get backedup
	// it will block on the writeQueue stage

	if lp.writesPerSecond <= 0 {
		lp.log.Notice("Starting indexer writer: No rate limiting enabled")
		for {
			if !lp._accept {
				return nil
			}
			skey := lp.cache.Pop()
			switch skey.IsBlank() {
			case true:
				time.Sleep(time.Second)
			default:
				stats.StatsdClient.Incr(fmt.Sprintf("indexer.leveldb.write.send-to-writers"), 1)
				lp.writeQueue <- &LevelDBIndexerJob{LD: lp, Name: skey}
			}
		}
	} else {
		sleep_t := float64(time.Second) * (time.Second.Seconds() / float64(lp.writesPerSecond))
		lp.log.Notice("Starting indexer writer: limiter every %f nanoseconds (%d writes per second)", sleep_t, lp.writesPerSecond)
		dur := time.Duration(int(sleep_t))
		for {
			if !lp._accept {
				return nil
			}
			skey := lp.cache.Pop()
			switch skey.IsBlank() {
			case true:
				time.Sleep(time.Second)
			default:
				stats.StatsdClient.Incr(fmt.Sprintf("indexer.leveldb.write.send-to-writers"), 1)
				lp.writeQueue <- &LevelDBIndexerJob{LD: lp, Name: skey}
				time.Sleep(dur)
			}
		}
	}
}

// keep an index of the stat keys and their fragments so we can look up
func (lp *LevelDBIndexer) Write(skey repr.StatName) error {
	return lp.cache.Add(skey)
}

func (lb *LevelDBIndexer) WriteOne(name repr.StatName) error {
	defer stats.StatsdSlowNanoTimeFunc(fmt.Sprintf("indexer.leveldb.write.path-time-ns"), time.Now())
	stats.StatsdClientSlow.Incr("indexer.leveldb.noncached-writes-path", 1)

	skey := name.Key
	segments := ParsePath(skey, name.UniqueIdString())

	for _, seg := range segments {
		err := seg.InsertAll(lb.db.SegmentConn(), lb.db.PathConn(), lb.db.UidConn())

		if err != nil {
			lb.log.Error("Could not insert segment %v", seg)
			stats.StatsdClientSlow.Incr("indexer.leveldb.segment-failures", 1)
		} else {
			stats.StatsdClientSlow.Incr("indexer.leveldb.segment-writes", 1)
		}
	}
	return nil
}

func (lp *LevelDBIndexer) findingIter(metric string) (iter leveldb_iter.Iterator, reg *regexp.Regexp, prefix string, err error) {
	segs := strings.Split(metric, ".")
	p_len := len(segs)

	// find the longest chunk w/o a reg and that will be the level db prefix filter
	needs_regex := needRegex(metric)

	long_chunk := ""
	use_key := metric
	use_key_len := p_len - 1
	if needs_regex {
		for _, pth := range segs {
			if needRegex(pth) {
				use_key = long_chunk
				break
			}
			if len(long_chunk) > 0 {
				long_chunk += "."
			}
			long_chunk += pth
		}
		reg, err = regifyKey(metric)
		if err != nil {
			return nil, nil, "", err
		}

	}

	// we simply troll the POS:{len}:{prefix} world
	prefix = fmt.Sprintf("POS:%d:%s", use_key_len, use_key)
	// log.Printf("USE KEY: %s", prefix)
	iter = lp.db.SegmentConn().NewIterator(leveldb_util.BytesPrefix([]byte(prefix)), nil)
	return iter, reg, prefix, nil
}

// FindInCache not using the ram indexer here yet so just an alias to find
func (lp *LevelDBIndexer) FindInCache(ctx context.Context, metric string, tags repr.SortingTags) (mt indexer.MetricFindItems, err error) {
	return lp.Find(ctx, metric, tags)
}

// basic find for non-regex items
func (lp *LevelDBIndexer) Find(ctx context.Context, metric string, tags repr.SortingTags) (mt indexer.MetricFindItems, err error) {
	sp, closer := lp.GetSpan("Find", ctx)
	sp.LogKV("driver", "LevelDBIndexer", "metric", metric, "tags", tags)
	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("indexer.leveldb.find.get-time-ns", time.Now())

	// special case for "root" == "*"
	if metric == "*" {
		return lp.FindRoot()
	}

	iter, reged, _, err := lp.findingIter(metric)

	if iter == nil || err != nil {
		return
	}
	// we simply troll the POS:{len}:{prefix} world

	for iter.Next() {
		ms := new(indexer.MetricFindItem)
		// Remember that the contents of the returned slice should not be modified, and
		// only valid until the next call to Next.
		value := iter.Value() // should be {id}:{path}:{has_data:0|1}
		value_arr := strings.Split(string(value), ":")
		has_data := value_arr[len(value_arr)-1]
		on_path := value_arr[1]
		on_path_byte := []byte(on_path)
		on_id := value_arr[0]

		spl := strings.Split(on_path, ".")

		if reged != nil {
			if reged.Match(on_path_byte) {
				ms.Text = spl[len(spl)-1]
				ms.Id = on_path
				ms.Path = on_path
				ms.UniqueId = on_id

				if has_data == "1" {
					ms.Expandable = 0
					ms.Leaf = 1
					ms.AllowChildren = 0
				} else {
					ms.Expandable = 1
					ms.Leaf = 0
					ms.AllowChildren = 1
				}

				mt = append(mt, ms)
			}

		} else {
			ms.Text = spl[len(spl)-1]
			ms.Id = on_path
			ms.Path = on_path
			ms.UniqueId = on_id

			if has_data == "1" {
				ms.Expandable = 0
				ms.Leaf = 1
				ms.AllowChildren = 0
			} else {
				ms.Expandable = 1
				ms.Leaf = 0
				ms.AllowChildren = 1
			}

			mt = append(mt, ms)
		}
	}
	iter.Release()
	err = iter.Error()
	return mt, err
}

// special case for "root" == "*" finder
func (lp *LevelDBIndexer) FindRoot() (mt indexer.MetricFindItems, err error) {
	defer stats.StatsdSlowNanoTimeFunc("indexer.leveldb.findroot.get-time-ns", time.Now())

	prefix := fmt.Sprintf("POS:%d:", 0)
	// log.Printf("USE KEY: %s", prefix)
	iter := lp.db.SegmentConn().NewIterator(leveldb_util.BytesPrefix([]byte(prefix)), nil)

	for iter.Next() {
		value := iter.Value() // should be {id}:{path}:{has_data:0|1}
		value_arr := strings.Split(string(value), ":")
		on_path := value_arr[1]

		ms := new(indexer.MetricFindItem)
		ms.Text = on_path
		ms.Id = on_path
		ms.Path = on_path

		ms.Expandable = 1
		ms.Leaf = 0
		ms.AllowChildren = 1

		mt = append(mt, ms)
	}

	iter.Release()
	err = iter.Error()
	return mt, err
}

func (lp *LevelDBIndexer) Expand(metric string) (me indexer.MetricExpandItem, err error) {
	iter, reged, _, err := lp.findingIter(metric)

	if err != nil {
		return me, err
	}

	for iter.Next() {
		val := iter.Value()
		if reged != nil {
			if reged.Match(val) {
				me.Results = append(me.Results, string(val))

			}
		} else {
			me.Results = append(me.Results, string(val))

		}
	}
	iter.Release()
	err = iter.Error()
	return me, err
}

func (lp *LevelDBIndexer) Delete(name *repr.StatName) error {
	segment := LevelDBSegment{Path: name.Key}
	segment.DeletePath(lp.db.SegmentConn())

	return nil
}

func (lp *LevelDBIndexer) List(has_data bool, page int) (indexer.MetricFindItems, error) {

	defer stats.StatsdSlowNanoTimeFunc("indexer.leveldb.find.get-time-ns", time.Now())

	// we simply troll the POS:{len}:{prefix} world
	// log.Printf("USE KEY: %s", prefix)
	iter := lp.db.PathConn().NewIterator(leveldb_util.BytesPrefix([]byte("")), nil)
	defer iter.Release()

	var mt indexer.MetricFindItems

	cur_ct := 0
	want_start := page * MAX_PER_PAGE
	stop_count := (page + 1) * MAX_PER_PAGE
	for iter.Next() {
		cur_ct++
		if cur_ct < want_start {
			continue
		}
		if cur_ct > stop_count {
			break
		}
		on_path := string(iter.Key())

		ms := new(indexer.MetricFindItem)
		ms.Text = on_path
		ms.Id = on_path
		ms.Path = on_path

		ms.Expandable = 0
		ms.Leaf = 1
		ms.AllowChildren = 0

		mt = append(mt, ms)
	}
	return mt, nil
}

/*************** TAG STUBS ************************/

func (my *LevelDBIndexer) GetTagsByUid(unique_id string) (tags repr.SortingTags, metatags repr.SortingTags, err error) {
	return tags, metatags, ErrNotYetimplemented
}

func (my *LevelDBIndexer) GetTagsByName(name string, page int) (tags indexer.MetricTagItems, err error) {
	return tags, ErrNotYetimplemented
}

func (my *LevelDBIndexer) GetTagsByNameValue(name string, value string, page int) (tags indexer.MetricTagItems, err error) {
	return tags, ErrNotYetimplemented
}

func (my *LevelDBIndexer) GetUidsByTags(key string, tags repr.SortingTags, page int) (uids []string, err error) {
	return uids, ErrNotYetimplemented
}

/************************************************************************/
/**********  Standard Worker Dispatcher JOB   ***************************/
/************************************************************************/
// insert job queue workers
type LevelDBIndexerJob struct {
	LD    *LevelDBIndexer
	Name  repr.StatName
	retry int
}

func (j *LevelDBIndexerJob) IncRetry() int {
	j.retry++
	return j.retry
}
func (j *LevelDBIndexerJob) OnRetry() int {
	return j.retry
}

func (j *LevelDBIndexerJob) DoWork() error {
	err := j.LD.WriteOne(j.Name)
	if err != nil {
		j.LD.log.Error("Insert failed for Index: %v retrying ...", j.Name.Key)
	}
	return err
}
