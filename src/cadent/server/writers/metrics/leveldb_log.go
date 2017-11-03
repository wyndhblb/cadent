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
	The LevelDB Metric Log

	Use the Chunk based Cacher
	Log and Write to a levelDB set of Dbs

*/

package metrics

import (
	"bytes"
	"cadent/server/broadcast"
	"cadent/server/schemas/indexer"
	smetrics "cadent/server/schemas/metrics"
	"cadent/server/schemas/repr"
	"cadent/server/series"
	"cadent/server/stats"
	"cadent/server/utils"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/dbs"
	"encoding/json"
	"fmt"
	"github.com/DataDog/zstd"
	"github.com/syndtr/goleveldb/leveldb"
	leveldb_util "github.com/syndtr/goleveldb/leveldb/util"
	"golang.org/x/net/context"
	"gopkg.in/op/go-logging.v1"
	"sort"
	"strconv"
	"strings"
	"time"
)

/****************** Metrics Writer *********************/
type LevelDBLogMetric struct {
	CacheBase

	log *logging.Logger

	cacher     *CacherChunk
	loggerChan *broadcast.Listener // log writes
	slicerChan *broadcast.Listener // current slice writes

	cacheTimeWindow int // chunk time before flush
	cacheChunks     int // number of chunks in ram

	expireTick time.Duration

	// DBs
	db *dbs.LevelDB

	okToWrite chan bool // temp channel to block as we may need to pull in the cache first

}

func NewLevelDBLogMetrics() *LevelDBLogMetric {
	lb := new(LevelDBLogMetric)
	lb.log = logging.MustGetLogger("writers.leveldblog")
	lb.okToWrite = make(chan bool)
	lb.expireTick = time.Duration(time.Minute * 30) // every 30 min run an purge of "ttl'ed" data
	return lb
}

// Config setups all the options for the writer
func (lb *LevelDBLogMetric) Config(conf *options.Options) (err error) {
	lb.options = conf

	// even if not used, it best be here for future goodies
	_, err = conf.Int64Required("resolution")
	if err != nil {
		return fmt.Errorf("resolution needed for ram writer: %v", err)
	}

	_tgs := conf.String("tags", "")
	if len(_tgs) > 0 {
		lb.staticTags = repr.SortingTagsFromString(_tgs)
	}

	lb.name = conf.String("name", "metrics:leveldb")
	err = RegisterMetrics(lb.Name(), lb)
	if err != nil {
		return err
	}

	rdur, err := time.ParseDuration(CASSANDRA_DEFAULT_RENDER_TIMEOUT)
	if err != nil {
		return err
	}
	lb.renderTimeout = rdur

	_cache, _ := conf.ObjectRequired("cache")
	if _cache == nil {
		return errMetricsCacheRequired
	}

	// need to cast this to the proper cache type
	var ok bool
	lb.cacher, ok = _cache.(*CacherChunk)
	if !ok {
		return ErrorMustBeChunkedCacheType
	}
	lb.CacheBase.cacher = lb.cacher // need to assign to sub object

	// chunk cache window options
	lb.cacheTimeWindow = int(conf.Int64("cache_time_window", int64(CACHE_LOG_TIME_CHUNKS/60)))
	lb.cacheChunks = int(conf.Int64("cache_num_chunks", CACHE_LOG_CHUNKS))
	lb.cacheLogFlush = int(conf.Int64("cache_log_flush_time", int64(CACHE_LOG_FLUSH)))

	// if this option is "false" then we never write actuall metric blobs to the DB
	// instead we operate as normal, and use only the things as a memory chunk cache.
	// This option is useful for replication, where we want to have a hot standby
	// that has all it's read caches full, and still maintains its log state
	// (so yes the Log items will still get written, and purged accordingly to
	// flushes, just no series or rollups occur)
	lb.shouldWrite = conf.Bool("cache_only", true)

	// for the overflow cached items::
	// these caches can be shared for a given writer set, and the caches may provide the data for
	// multiple writers, we need to specify that ONE of the writers is the "main" one otherwise
	// the Metrics Write function will add the points over again, which is not a good thing
	// when the accumulator flushes things to the multi writers
	// The Writer needs to know it's "not" the primary writer and thus will not "add" points to the
	// cache .. so the cache basically gets "one" primary writer pointed (first come first serve)
	lb.isPrimary = lb.cacher.SetPrimaryWriter(lb)
	if lb.isPrimary {
		lb.log.Notice("LevelDB Log series writer is the primary writer to write back cache %s", lb.cacher.Name)
	}

	// only metrics tables here
	conf.Set("open_index_tables", false)

	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("Indexer: `dsn` (/path/to/dbs) is needed for leveldb config")
	}

	db, err := dbs.NewDB("leveldb", dsn, conf)
	if err != nil {
		return err
	}

	lb.db = db.(*dbs.LevelDB)

	return nil
}

func (lb *LevelDBLogMetric) Driver() string {
	return "leveldb-log"
}

func (lb *LevelDBLogMetric) Start() {
	lb.startstop.Start(func() {

		lb.log.Notice(
			"Starting leveldb-log series at %d time chunks", lb.cacher.maxTime,
		)

		lb.shutitdown = false

		lb.loggerChan = lb.cacher.GetLogChan()
		lb.slicerChan = lb.cacher.GetSliceChan()

		go lb.writeLog()
		go lb.writeSlice()
		go lb.runPeriodicExpire()

		// set cache options
		lb.cacher.maxChunks = uint32(lb.cacheChunks)
		lb.cacher.maxTime = uint32(lb.cacheTimeWindow) * 60
		lb.cacher.logTime = uint32(lb.cacheLogFlush)

		lb.cacher.Start()

		// backfill
		lb.getSequences()

		// off to the races
		close(lb.okToWrite)
		lb.okToWrite = nil
	})
}

func (lb *LevelDBLogMetric) Stop() {
	lb.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()
		lb.log.Warning("Starting Shutdown of cassandra series writer")

		if lb.shutitdown {
			return // already did
		}
		lb.shutitdown = true

		lb.cacher.Stop()

		// we only need to write any metrics in the
		cItems := lb.cacher.CurrentLog()
		lb.log.Warning("Shutting down %s and writing final log (%d metrics)", lb.cacher.Name, len(cItems))

		time.Sleep(2 * time.Second) // little pause to let any writes complete
		lb.db.Close()
		lb.log.Warning("Shutdown finished ... quiting leveldb log series writer")
		return
	})
}

// on startup yank the current sequences we may have missed due to crash/restart
func (lb *LevelDBLogMetric) getSequences() {

	lb.log.Notice("Backfilling internal caches from log")

	logDb, err := lb.db.MetricLogDB()
	if err != nil {
		lb.log.Critical("Failed to get metric log DB: %v", err)
		return
	}

	prefix := leveldb_util.BytesPrefix([]byte(fmt.Sprintf("metric-log:")))

	iter := logDb.NewIterator(prefix, nil)
	defer iter.Release()

	sequences := make(utils.Int64List, 0)
	seqMap := make(map[int64]utils.Int64List, 0)
	// need to get the sorted list of sequence IDs
	// metric-log:sequeceid:time
	entries := 0
	for iter.Next() {
		keySpl := strings.Split(string(iter.Key()), ":")
		if len(keySpl) < 3 {
			continue
		}
		inter, err := strconv.ParseInt(keySpl[1], 10, 64)
		if err != nil {
			lb.log.Errorf("sequence ID is Not an int: %v", err)
			continue
		}
		timer, err := strconv.ParseInt(keySpl[2], 10, 64)
		if err != nil {
			lb.log.Errorf("time is Not an int: %v", err)
			continue
		}
		if _, ok := seqMap[inter]; ok {
			seqMap[inter] = append(seqMap[inter], timer)
		} else {
			seqMap[inter] = make(utils.Int64List, 0)
			seqMap[inter] = append(seqMap[inter], timer)
			sequences = append(sequences, inter)
		}
		entries++
	}
	iter.Release()

	if len(sequences) == 0 {
		lb.log.Notice("No log seqeunces found...")
		return
	}
	// set the current sequence number 'higher"
	sort.Sort(sequences)

	lb.log.Notice("Reading back %d sequence entries", entries)
	added := 0

	// can only read in the last SEQ id, if there are others, it means a nasty crash and/or
	// startup crash happened, and those other sequences are too old and will not be functional anymore
	// and also out of time order
	seq := sequences[len(sequences)-1]
	lb.cacher.setCurSequenceId(seq)
	lb.cacher.StartChunk(seq)
	buf := bytes.NewBuffer(nil)

	for _, curSeq := range sequences {

		ts := seqMap[seq]
		sort.Sort(ts)

		for _, timer := range ts {
			lb.log.Notice("Backfllling sequence/time from log %d:%d", curSeq, timer)

			key := []byte(fmt.Sprintf("metric-log:%d:%d", curSeq, timer))

			gots, err := logDb.Get(key, nil)
			if err != nil {
				lb.log.Errorf("Could not get sequence/time in log %d:%d :: %v", curSeq, timer, err)
				continue
			}
			// unzstd the punk
			buf.Reset()
			buf.Write(gots)
			decomp := zstd.NewReader(buf)

			var data []byte
			_, err = decomp.Read(data)
			if err != nil {
				lb.log.Errorf("Error reading compressed slice log data: %v", err)
				decomp.Close()
				continue
			}

			slice := make(map[repr.StatId][]*repr.StatRepr, 0)
			dec := json.NewDecoder(decomp)
			err = dec.Decode(&slice)

			if err != nil {
				lb.log.Errorf("Error reading slice log data: %v", err)
				decomp.Close()
				continue
			}

			// now we need to add the points to the chunks
			// this can mean our first chunk is longer then the "slice time" but that should be ok
			for _, item := range slice {
				sort.Sort(repr.StatReprSlice(item))
				for _, stat := range item {
					lb.cacher.BackFill(stat.Name, stat, seq)
					added++
				}
			}
			decomp.Close()
		}

		// have to purge these otherwise will just sit there forever
		if curSeq != seq {
			prefix := leveldb_util.BytesPrefix([]byte(fmt.Sprintf("metric-log:%d:", curSeq)))
			iter := logDb.NewIterator(prefix, nil)
			for iter.Next() {
				logDb.Delete(iter.Key(), nil)
			}
			iter.Release()
			continue
		}

	}

	lb.log.Notice("Backfilled %d data points from the log.  Current Sequence is %d", added, seq)
}

// listen for the broad cast chan an flush out a log entry
func (lb *LevelDBLogMetric) writeLog() {
	for {
		clog, more := <-lb.loggerChan.Ch
		if !more {
			return
		}
		if clog == nil {
			continue
		}
		tn := time.Now()

		toLog, ok := clog.(*CacheChunkLog)
		if !ok {
			stats.StatsdClientSlow.Incr("writer.leveldblog.log.writes.errors", 1)
			lb.log.Critical("Not a CacheChunkLog type cannot write log")
			continue
		}

		logDb, err := lb.db.MetricLogDB()
		if err != nil {
			stats.StatsdClientSlow.Incr("writer.leveldblog.log.writes.errors", 1)
			lb.log.Critical("Failed to get metric log DB: %v", err)
			continue
		}

		// zstd the punk
		buf := bytes.NewBuffer(nil)
		compressed := zstd.NewWriter(buf)
		enc := json.NewEncoder(compressed)
		err = enc.Encode(toLog.Slice)
		compressed.Close()

		if err != nil {
			stats.StatsdClientSlow.Incr("writer.leveldblog.log.writes.errors", 1)
			lb.log.Critical("Json encode error: cannot write log: %v", err)
			continue
		}

		key := fmt.Sprintf("metric-log:%d:%d", toLog.SequenceId, time.Now().UnixNano())

		err = logDb.Put([]byte(key), buf.Bytes(), nil)

		if err != nil {
			stats.StatsdClientSlow.Incr("writer.leveldblog.log.writes.errors", 1)
			lb.log.Critical("LeveDB commit error: cannot write log: %v", err)
			continue
		}

		if err != nil {
			stats.StatsdClientSlow.Incr("writer.leveldblog.log.writes.errors", 1)
			lb.log.Critical("Cassandra writer error: cannot write log: %v", err)
			continue
		}

		stats.StatsdClientSlow.Incr("writer.leveldblog.log.writes.success", 1)
		stats.StatsdSlowNanoTimeFunc("writer.leveldblog.log.write-time-ns", tn)

	}
}

// listen for the broadcast chan an flush out the newest slice in the mix: Noop just bleed the channel
func (lb *LevelDBLogMetric) writeSlice() {
	for {
		clog, more := <-lb.slicerChan.Ch
		if !more {
			return
		}

		tn := time.Now()

		if !lb.ShouldWrite() {
			lb.log.Notice("ShouldWrite is false, assuming cache only replica, not writing items")
		}

		toLog, ok := clog.(*CacheChunkSlice)
		if !ok {
			stats.StatsdClientSlow.Incr("writer.leveldblog.slice.writes.errors", 1)
			lb.log.Critical("Not a CacheChunkSlice type cannot write log")
			continue
		}

		hadErrors := int64(0)
		hadHappy := int64(0)
		if lb.ShouldWrite() {
			for _, item := range toLog.Slice.AllSeries() {
				err := lb.doInsert(&smetrics.TotalTimeSeries{Series: item.Series, Name: item.Name})
				if err != nil {
					lb.log.Error("Series Insert Error: %v", err)
					hadErrors++
				} else {
					hadHappy++
				}
			}
		} else {
			lb.log.Notice("ShouldWrite is false, assuming cache only replica, not writing items")
		}

		// we still do this for the cache only options
		// now purge out the sequence number as we've written our data
		key := fmt.Sprintf("metric-log:%d:", toLog.SequenceId)
		logDb, err := lb.db.MetricLogDB()

		if err != nil {
			stats.StatsdClientSlow.Incr("writer.leveldblog.log.purge.errors", 1)
			lb.log.Critical("LevelDB writer error: cannot purge sequence %d: %v", toLog.SequenceId, err)
		} else {

			iter := logDb.NewIterator(leveldb_util.BytesPrefix([]byte(key)), nil)
			for iter.Next() {
				logDb.Delete(iter.Key(), nil)
			}

			lb.log.Info(
				"Purged log sequence %d resolution %d",
				toLog.SequenceId,
				lb.currentResolution,
			)
		}
		stats.StatsdClientSlow.Incr("writer.leveldblog.slice.writes.errors", hadErrors)
		stats.StatsdClientSlow.Incr("writer.leveldblog.slice.writes.success", hadHappy)
		stats.StatsdNanoTimeFunc("writer.leveldblog.slice.write-time-ns", tn)
	}
}

// Write simple proxy to the cacher (does nothing with the offset)
func (lb *LevelDBLogMetric) WriteWithOffset(stat *repr.StatRepr, offset *smetrics.OffsetInSeries) error {
	return lb.Write(stat)
}

func (lb *LevelDBLogMetric) Write(stat *repr.StatRepr) error {

	// block until ready
	if lb.okToWrite != nil {
		<-lb.okToWrite
	}

	if lb.shutitdown {
		return nil
	}

	stat.Name.MergeMetric2Tags(lb.staticTags)
	// can only need to do this if the first resolution
	if lb.currentResolution == lb.resolutions[0][0] {
		lb.indexer.Write(*stat.Name)
	}

	// not primary writer .. move along
	if !lb.isPrimary {
		return nil
	}

	return lb.cacher.Add(stat.Name, stat)

}

func (lb *LevelDBLogMetric) doInsert(ts *smetrics.TotalTimeSeries) error {

	_, err := lb.InsertSeries(ts.Name, ts.Series, ts.Name.Resolution)
	if err != nil {
		lb.log.Errorf("Failed to add series to DB: %s (%s) %v", ts.Name.Key, ts.Name.UniqueIdString(), err)
	}
	return err
}

func (lb *LevelDBLogMetric) pointsKeyName(uid string, t int64) []byte {
	buf := utils.GetBytesBuffer()
	defer utils.PutBytesBuffer(buf)
	fmt.Fprintf(buf, "points:%s:%d", uid, t)
	return buf.Bytes()
}

func (lb *LevelDBLogMetric) timesKeyName(uid string) []byte {
	buf := utils.GetBytesBuffer()
	defer utils.PutBytesBuffer(buf)
	fmt.Fprintf(buf, "times:%s", uid)
	return buf.Bytes()
}

// helper for the index bits for series
type timePtype struct {
	T int64
	P string
}

type timePtypeList []*timePtype

// Sorting for the target outputs
func (p timePtypeList) Len() int { return len(p) }
func (p timePtypeList) Less(i, j int) bool {
	return p[i].T < p[j].T
}
func (p timePtypeList) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func (lb *LevelDBLogMetric) getTimeTypeArray(bits []byte) []*timePtype {
	useTimes := make([]*timePtype, 0)
	for _, item := range bytes.Split(bits, repr.COMMA_SEPARATOR_BYTE) {

		spl := bytes.Split(item, repr.COLON_SEPARATOR_BYTE)
		if len(spl) < 2 {
			continue
		}
		onT, err := strconv.ParseInt(string(spl[0]), 10, 64)
		if err != nil {
			lb.log.Error("LevelDB Driver: Index read failed not a timestamp, %v", err)
			continue
		}
		item := &timePtype{
			T: onT,
			P: string(spl[1]),
		}
		useTimes = append(useTimes, item)
	}
	sort.Sort(timePtypeList(useTimes))

	return useTimes
}

func (lb *LevelDBLogMetric) runPeriodicExpire() {

	ticker := time.NewTicker(lb.expireTick)
	RMfactor := 3 // keep "3" resolution chunks
	for {
		<-ticker.C

		res := lb.resolutions[0][0]
		ttl := lb.resolutions[0][1]
		if ttl <= 0 {
			continue
		}
		// get all the "times:*" in each opened DB we have in the mix,
		// find times from the index of times to purge
		dbs := lb.db.OpenMetricTables()
		bPrefix := leveldb_util.BytesPrefix([]byte("times:"))
		expTime := (time.Now().Unix() - int64(RMfactor*ttl)) * int64(time.Second)
		purged := 0
		lb.log.Info("Running expire for all metrics older then %d in the %ds resolution (ttl: %d)", expTime, res, ttl)

		for _, db := range dbs {
			iter := db.NewIterator(bPrefix, nil)

			for iter.Next() {
				uT := lb.getTimeTypeArray(iter.Value())
				if len(uT) == 0 {
					continue
				}
				// times:uid
				onK := strings.Split(string(iter.Key()), ":")
				if len(uT) < 2 {
					continue
				}
				uid := onK[1]
				lPurged := 0
				for _, item := range uT {
					if item.T < expTime {
						toDel := []byte(fmt.Sprintf("points:%s:%d", uid, item.T))
						err := db.Delete(toDel, nil)
						if err != nil {
							lb.log.Errorf("Failed to delete metrics older then %d in %ds resolution", expTime, res)
							continue
						}
						lPurged++
					}
				}
				stats.StatsdClientSlow.Incr(fmt.Sprintf("writer.leveldblog.expire.%ds.rows.deleted", res), int64(lPurged))
				purged += lPurged
			}
			iter.Release()
		}
		lb.log.Noticef("Removed %d metrics older then %d in the %ds resolution", purged, expTime, res)
	}

}

func (lb *LevelDBLogMetric) InsertSeries(name *repr.StatName, timeseries series.TimeSeries, resolution uint32) (n int, err error) {

	defer func() {
		if r := recover(); r != nil {
			lb.log.Critical("LevelDB Failure (panic) %v ::", r)
			err = fmt.Errorf("The recover error: %v", r)
		}
	}()

	//panic(fmt.Sprintf("InsertSeries: %v :: %v", cass.tablePerResolution, cass.insQ))
	if name == nil {
		return 0, errNameIsNil
	}
	if timeseries == nil {
		return 0, errSeriesIsNil
	}
	defer stats.StatsdNanoTimeFunc(fmt.Sprintf("writer.leveldblog.insert.metric-time-ns"), time.Now())

	l := timeseries.Count()
	if l == 0 {
		return 0, nil
	}

	blob, err := timeseries.MarshalBinary()
	if err != nil {
		return 0, err
	}

	// if no bytes don't add
	if len(blob) == 0 {
		return 0, nil
	}

	uid := name.UniqueIdString()
	// levelDB gets a set of tables to write to
	db, err := lb.db.MetricDB(uid)
	if err != nil {
		lb.log.Error("LevelDB Driver: Failed to get DB: Metric insert failed, %v", err)
		stats.StatsdClientSlow.Incr("writer.leveldblog.insert.metric-failures", 1)
		return 0, err
	}

	key := lb.pointsKeyName(uid, timeseries.LastTime())
	err = db.Put(key, blob, nil)

	if err != nil {
		lb.log.Error("LevelDB Driver:Metric insert failed, %v", err)
		stats.StatsdClientSlow.Incr("writer.leveldblog.insert.metric-failures", 1)
		return 0, err
	}

	// add this entime:seriestime to the index key
	idxKey := lb.timesKeyName(uid)
	gots, err := db.Get(idxKey, nil)
	if err != nil && err != leveldb.ErrNotFound {
		lb.log.Error("LevelDB Driver: Index read failed, %v", err)
		stats.StatsdClientSlow.Incr("writer.leveldblog.insert.metric-failures", 1)
		return 0, err
	}

	if len(gots) > 0 {
		gots = append(gots, []byte(fmt.Sprintf(",%d:%s", timeseries.LastTime(), timeseries.Name()))...)

	} else {
		gots = []byte(fmt.Sprintf("%d:%s", timeseries.LastTime(), timeseries.Name()))
	}

	err = db.Put(idxKey, gots, nil)
	if err != nil {
		lb.log.Error("LevelDB Driver: Index insert failed, %v", err)
		stats.StatsdClientSlow.Incr("writer.leveldblog.insert.metric-failures", 1)
		return 0, err
	}

	stats.StatsdClientSlow.Incr("writer.leveldblog.writes", 1)
	stats.StatsdClientSlow.GaugeAvg("writer.leveldblog.insert.metrics-per-writes", int64(l))

	return l, nil
}

/************************ READERS ****************/

// grab the time series from the DBs
func (lb *LevelDBLogMetric) GetFromDatabase(metric *indexer.MetricFindItem, resolution uint32, start int64, end int64, resample uint32) (rawd *smetrics.RawRenderItem, err error) {
	defer stats.StatsdSlowNanoTimeFunc("reader.leveldblog.database.get-time-ns", time.Now())
	rawd = new(smetrics.RawRenderItem)

	db, err := lb.db.MetricDB(metric.UniqueId)
	if err != nil {
		return nil, err
	}

	// times need to be in Nanos, but incoming as a epoch
	// time in times index is in NanoSeconds so we need to pad the times from seconds -> nanos
	nano := int64(time.Second)
	nanoEnd := end * nano
	nanoStart := start * nano

	// our keys in the file are points:{uid}:{endtime}
	// there is also a "list" of endtimes times:{uid} -> [time:seriestype, etime:seriestype...]
	// so we need to iterate all the {uid}:* and use the times we need
	// add this endtime:seriestype to the index key
	idxKey := lb.timesKeyName(metric.UniqueId)
	gots, err := db.Get(idxKey, nil)

	if err == leveldb.ErrNotFound {
		return rawd, nil
	}

	if err != nil {
		lb.log.Error("LevelDB Driver: Index read failed, %v", err)
		stats.StatsdClientSlow.Incr("reader.leveldblog.index.failures", 1)
		return nil, err
	}
	// nothing there
	if len(gots) == 0 {
		return nil, nil
	}
	useTimes := lb.getTimeTypeArray(gots)

	if len(useTimes) == 0 {
		return rawd, nil
	}

	// for each "series" we get make a list of points
	statName := metric.StatName()

	uStart := uint32(start)
	uEnd := uint32(end)
	rawd.Start = uStart
	rawd.End = uEnd
	rawd.Step = resolution
	rawd.Id = metric.UniqueId
	rawd.Metric = metric.Path
	rawd.AggFunc = statName.AggType()

	var pBytes []byte

	tStart := uint32(start)

	// on resamples (if >0 ) we simply merge points until we hit the steps
	doResample := resample > 0 && resample > resolution
	if doResample {
		tStart, uEnd = ResampleBounds(tStart, uEnd, resample)
		rawd.Start = tStart
		rawd.End = uEnd
		rawd.Step = resample
	}

	for _, eTime := range useTimes {
		// not in range
		if !(eTime.T >= nanoStart && eTime.T <= nanoEnd) {
			continue
		}

		getKey := lb.pointsKeyName(metric.UniqueId, eTime.T)
		pBytes, err = db.Get(getKey, nil)
		if err == leveldb.ErrNotFound {
			continue
		}
		if err != nil {
			lb.log.Error("LevelDB Driver: Points read failed:, %v", err)
			continue
		}

		sIter, err := series.NewIter(eTime.P, pBytes)
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
		if sIter.Error() != nil {
			return rawd, sIter.Error()
		}
	}

	if len(rawd.Data) > 0 {
		rawd.RealEnd = rawd.Data[len(rawd.Data)-1].Time
		rawd.RealStart = rawd.Data[0].Time
	}

	return rawd, nil

}

// after the "raw" render we need to yank just the "point" we need from the data which
// will make the read-cache much smaller (will compress just the Mean value as the count is 1)
func (lb *LevelDBLogMetric) RawDataRenderOne(ctx context.Context, metric *indexer.MetricFindItem, start int64, end int64, resample uint32) (*smetrics.RawRenderItem, error) {
	sp, closer := lb.GetSpan("RawDataRenderOne", ctx)
	sp.LogKV("driver", "LevelDBLogMetric", "metric", metric.StatName().Key, "uid", metric.UniqueId)
	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("reader.leveldblog.rawrenderone.get-time-ns", time.Now())

	preped := lb.newGetPrep(metric, start, end, resample)

	if metric.Leaf == 0 {
		//data only but return a "blank" data set otherwise graphite no likey
		return preped.rawd, ErrorNotADataNode
	}

	bLen := (preped.uEnd - preped.uStart) / preped.outResolution //just to be safe
	if bLen <= 0 {
		return preped.rawd, ErrorTimeTooSmall
	}

	// the step is unknown so set it to the min allowed
	inflight, err := lb.GetFromWriteCache(metric, preped.uStart, preped.uEnd, 1)

	// all we have is the cache,
	if inflight != nil && err == nil && len(inflight.Data) > 1 {
		// move the times to the "requested" ones and quantize the list
		if inflight.RealStart < preped.uStart {
			inflight.Resample(preped.outResolution)
			return inflight, err
		}

		// The cache here can have more in RAM that is also on disk in cassandra,
		// so we need to alter the cassandra query to get things "not" already gotten from the cache
		// i.e. double counting on a merge operation
		if inflight.RealStart <= uint32(end) {
			end = TruncateTimeTo(int64(inflight.RealStart), int(preped.dbResolution))
		}
	}

	// and now for the Query otherwise
	dbData, err := lb.GetFromDatabase(metric, preped.dbResolution, start, end, preped.outResolution)
	if dbData == nil {
		return inflight, nil
	}

	if err != nil {
		lb.log.Error("LevelDB: Error getting from DB: %v", err)
		return preped.rawd, err
	}

	dbData.Step = preped.outResolution
	dbData.Start = preped.uStart
	dbData.End = preped.uEnd
	dbData.Tags = metric.Tags
	dbData.MetaTags = metric.MetaTags
	dbData.InCache = false

	if inflight == nil || len(inflight.Data) == 0 {
		return dbData, nil
	}

	if len(dbData.Data) > 0 && len(inflight.Data) > 0 {
		inflight.MergeWithResample(dbData, preped.outResolution)
		inflight.InCache = true
		return inflight, nil
	}
	if inflight.Step != preped.outResolution {
		inflight.Resample(preped.outResolution)
	}
	return inflight, nil
}

func (lb *LevelDBLogMetric) RawRender(ctx context.Context, path string, start int64, end int64, tags repr.SortingTags, resample uint32) ([]*smetrics.RawRenderItem, error) {
	sp, closer := lb.GetSpan("RawRender", ctx)
	sp.LogKV("driver", "LevelDBLogMetric", "path", path, "from", start, "to", end)
	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("reader.leveldblog.rawrender.get-time-ns", time.Now())

	paths := SplitNamesByComma(path)
	var metrics []*indexer.MetricFindItem

	renderWg := utils.GetWaitGroup()
	defer utils.PutWaitGroup(renderWg)

	for _, pth := range paths {
		mets, err := lb.indexer.Find(ctx, pth, tags)

		if err != nil {
			continue
		}
		metrics = append(metrics, mets...)
	}

	rawd := make([]*smetrics.RawRenderItem, 0)

	procs := CASSANDRA_DEFAULT_METRIC_RENDER_WORKERS

	jobs := make(chan *indexer.MetricFindItem, len(metrics))
	results := make(chan *smetrics.RawRenderItem, len(metrics))

	renderOne := func(met *indexer.MetricFindItem) *smetrics.RawRenderItem {
		_ri, err := lb.RawDataRenderOne(ctx, met, start, end, resample)

		if err != nil {
			stats.StatsdClientSlow.Incr("reader.leveldblog.rawrender.errors", 1)
			lb.log.Errorf("Read Error for %s (%d->%d) : %v", path, start, end, err)
		}
		return _ri
	}

	// ye old fan out technique but not "too many" as to kill the server
	jobWorker := func(jober int, taskqueue <-chan *indexer.MetricFindItem, resultqueue chan<- *smetrics.RawRenderItem) {
		resultsChan := make(chan *smetrics.RawRenderItem, 1)
		for met := range taskqueue {
			go func() { resultsChan <- renderOne(met) }()
			select {
			case <-time.After(lb.renderTimeout):
				stats.StatsdClientSlow.Incr("reader.leveldblog.rawrender.timeouts", 1)
				lb.log.Errorf("Render Timeout for %s (%d->%d)", path, start, end)
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
			// this is always true for this type
			res.UsingCache = true
			rawd = append(rawd, res)
		}
	}
	close(results)
	stats.StatsdClientSlow.Incr("reader.leveldblog.rawrender.metrics-per-request", int64(len(metrics)))

	return rawd, nil
}
