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
	The Cassandra Metric Log

	This is an attempt to mend the issue w/ the "series" metric writers.  That issue is what happens when things
	crash, restart, etc.  Any of the series in Ram get lost.  So this is a slightly different table
	where per "now" timestamp we take the recent metric arrivals, and add them in one big blob of zstd compressed
	json.

	On start up, the series writer then simply reads back in this log, and maintains its RAM profile
	for querying and eventual writing into the time-series tables

	This fundamentally changes "overflow" writing into a time chunk based method.

	the ram series cache is then flushed every "chunk" (configurable, default of 10m) and we keep "N" chunks (configurable,
	default of 6) for each metric in Ram for fast queries.  Once we "hit #7" we flush out the oldest one to cassandra
	Once that entire chunk is flushed out, we drop that "sequence" we are on (a sequence is just the counter
	of each chunk we've completed)

	Thus if we crash, we still have the most (yes some data will be lost depending on the crash
	scenario, but we will have much more then if we had just the RAM series).

	Each writer needs to have its own log table, (if there are 3 cadents writing/reading from the same DB
	each one will be at a different checkpoint) so there MUST be an "index" for them otherwise on crash
	and restart (or even node loss and restore) it needs to know where to pick up from.


	CREATE TABLE metric_log_{cadentindex} (
    		stamp bigint,
    		sequence bigint,
    		points blob,
    		PRIMARY KEY (stamp, sequence)
	) WITH COMPACT STORAGE AND CLUSTER ORDERING stamp DESC;



*/

package metrics

import (
	"bytes"
	"cadent/server/dispatch"
	smetrics "cadent/server/schemas/metrics"
	"cadent/server/schemas/repr"
	"cadent/server/stats"
	"cadent/server/utils"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"encoding/json"
	"fmt"
	"github.com/DataDog/zstd"
	"github.com/cenkalti/backoff"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// max bytes for the rollup triggers
	CASSANDRA_LOG_ROLLLUP_MAXBYTES = 2048

	// this is queue or blast, in queue mode, things are added to
	// a queue uniquely by name, and things are "gently" rolled up
	// pretty much "all the time" as rollups for LOTS of keys is time consuming
	// and may not be done in blast mode before the next flush happens
	CASSANDRA_LOG_ROLLLUP_MODE = "queue"

	// need some lower default values here then the basic one as we are doing lots
	// of rollups potentially, and we should have the time do do this over a longer period
	// so a bit of queue blocking is not that bad
	CASSANDRA_LOG_DEFAULT_METRIC_WORKERS   = 4
	CASSANDRA_LOG_DEFAULT_METRIC_QUEUE_LEN = 128
	CASSANDRA_LOG_DEFAULT_METRIC_RETRIES   = 2

	CASSANDRA_LOG_TTL = int64(24 * time.Hour) // expire stuff after a day
)

/****************** Metrics Writer *********************/
type CassandraLogMetric struct {
	CassandraMetric

	cacher *CacherChunk

	okToWrite   chan bool // temp channel to block as we may need to pull in the cache first
	writerIndex int
}

func NewCassandraLogMetrics() *CassandraLogMetric {
	cass := new(CassandraLogMetric)
	cass.driver = "cassandra-log"
	cass.isPrimary = false
	cass.rollupType = "cached"
	cass.okToWrite = make(chan bool)
	return cass
}

func NewCassandraLogTriggerMetrics() *CassandraLogMetric {
	cass := new(CassandraLogMetric)
	cass.driver = "cassandra-log-triggered"
	cass.isPrimary = false
	cass.rollupType = "triggered"
	cass.okToWrite = make(chan bool)
	return cass
}

// Config setups all the options for the writer
func (cass *CassandraLogMetric) Config(conf *options.Options) (err error) {
	cass.options = conf

	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("Metrics: `dsn` (server1,server2,server3) is needed for cassandra config")
	}

	cass.name = conf.String("name", "metrics:cassandra-log:"+dsn)
	// reg ourselves before try to get conns
	err = RegisterMetrics(cass.Name(), cass)
	if err != nil {
		return err
	}

	gots, err := NewCassandraWriter(conf)
	if err != nil {
		return err
	}

	cass.writer = gots
	cass.writer.log.Noticef("Registering metrics: %s", cass.Name())

	// even if not used, it best be here for future goodies
	_, err = conf.Int64Required("resolution")
	if err != nil {
		return fmt.Errorf("resolution needed for cassandra writer: %v", err)
	}

	// need writer index for the log table
	i, err := conf.Int64Required("writer_index")
	if err != nil {
		return fmt.Errorf("writer_index is required for cassandra-log writer: %v", err)
	}
	cass.writerIndex = int(i)

	_tgs := conf.String("tags", "")
	if len(_tgs) > 0 {
		cass.staticTags = repr.SortingTagsFromString(_tgs)
	}

	// tweak queus and worker sizes
	cass.numWorkers = int(conf.Int64("write_workers", CASSANDRA_LOG_DEFAULT_METRIC_WORKERS))
	cass.queueLen = int(conf.Int64("write_queue_length", CASSANDRA_LOG_DEFAULT_METRIC_QUEUE_LEN))
	cass.dispatchRetries = int(conf.Int64("write_queue_retries", CASSANDRA_LOG_DEFAULT_METRIC_RETRIES))
	cass.tablePerResolution = conf.Bool("table_per_resolution", false)

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

	// need to cast this to the proper cache type
	var ok bool
	cass.cacher, ok = _cache.(*CacherChunk)
	if !ok {
		return ErrorMustBeChunkedCacheType
	}
	cass.CacheBase.cacher = cass.cacher // need to assign to sub object

	// chunk cache window options
	cass.cacheTimeWindow = int(conf.Int64("cache_time_window", int64(CACHE_LOG_TIME_CHUNKS/60)))
	cass.cacheChunks = int(conf.Int64("cache_num_chunks", CACHE_LOG_CHUNKS))
	cass.cacheLogFlush = int(conf.Int64("cache_log_flush_time", int64(CACHE_LOG_FLUSH)))

	// if this option is "false" then we never write actuall metric blobs to the DB
	// instead we operate as normal, and use only the things as a memory chunk cache.
	// This option is useful for replication, where we want to have a hot standby
	// that has all it's read caches full, and still maintains its log state
	// (so yes the Log items will still get written, and purged accordingly to
	// flushes, just no series or rollups occur)
	cass.shouldWrite = conf.Bool("cache_only", true)

	// for the overflow cached items::
	// these caches can be shared for a given writer set, and the caches may provide the data for
	// multiple writers, we need to specify that ONE of the writers is the "main" one otherwise
	// the Metrics Write function will add the points over again, which is not a good thing
	// when the accumulator flushes things to the multi writers
	// The Writer needs to know it's "not" the primary writer and thus will not "add" points to the
	// cache .. so the cache basically gets "one" primary writer pointed (first come first serve)
	cass.isPrimary = cass.cacher.SetPrimaryWriter(cass)
	if cass.isPrimary {
		cass.writer.log.Notice("Cassandra series writer is the primary writer to write back cache %s", cass.cacher.Name)
	}

	if cass.rollupType == "triggered" {
		cass.driver = "cassandra-log-triggered" // reset the name
		rollupSize := conf.Int64("rollup_byte_size", CASSANDRA_LOG_ROLLLUP_MAXBYTES)
		cass.rollup = NewRollupMetric(cass, int(rollupSize))
	}

	if cass.tablePerResolution {
		cass.selQ = "SELECT ptype, points FROM %s WHERE id=? AND etime >= ? AND etime <= ?"
	} else {
		cass.selQ = "SELECT ptype, points FROM %s WHERE mid={id: ?, res: ?} AND etime >= ? AND etime <= ?"
	}

	return nil
}

func (cass *CassandraLogMetric) Start() {
	cass.startstop.Start(func() {

		cass.okToWrite = make(chan bool)

		cass.writer.log.Notice(
			"Starting cassandra-log series writer for %s at %d time chunks: Writer Index: %d Write mode: %v",
			cass.writer.db.MetricTable(), cass.cacher.maxTime, cass.writerIndex, cass.writer.db.Cluster().Consistency,
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

		// table names is {base}_{windex}_{res}s
		schems.LogTable = cass.writer.db.LogTableBase()
		schems.WriteIndex = cass.writerIndex
		schems.Resolution = cass.currentResolution
		err := schems.AddMetricsTable()

		if err != nil {
			panic(err)
		}
		cass.writer.log.Notice("Adding metric log tables")
		schems.Resolution = cass.currentResolution // need to reset this if table per res is true
		err = schems.AddMetricsLogTable()
		if err != nil {
			panic(err)
		}

		cass.shutitdown = false

		cass.loggerChan = cass.cacher.GetLogChan()
		cass.slicerChan = cass.cacher.GetSliceChan()

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
			cass.rollup.blobMaxBytes = int(cass.options.Int64("rollup_byte_size", CASSANDRA_LOG_ROLLLUP_MAXBYTES))
			cass.rollup.numRetries = int(cass.options.Int64("rollup_retries", ROLLUP_NUM_RETRIES))
			cass.rollup.numWorkers = int(cass.options.Int64("rollup_workers", ROLLUP_NUM_WORKERS))
			cass.rollup.queueLength = int(cass.options.Int64("rollup_queue_length", ROLLUP_QUEUE_LENGTH))
			cass.rollup.consumeQueueWorkers = int(cass.options.Int64("rollup_queue_workers", ROLLUP_NUM_QUEUE_WORKERS))

			// queue or blast
			cass.rollup.SetRunMode(cass.options.String("rollup_method", CASSANDRA_LOG_ROLLLUP_MODE))
			cass.rollup.SetMinResolution(cass.resolutions[0][0])
			cass.rollup.SetResolutions(cass.resolutions[1:])
			go cass.rollup.Start()
		}

		//start up the dispatcher
		cass.dispatcher = dispatch.NewDispatchQueue(
			cass.numWorkers,
			cass.queueLen,
			cass.dispatchRetries,
		)
		cass.dispatcher.Start()

		go cass.writeLog()
		go cass.writeSlice()

		// set cache options
		cass.cacher.maxChunks = uint32(cass.cacheChunks)
		cass.cacher.maxTime = uint32(cass.cacheTimeWindow) * 60
		cass.cacher.logTime = uint32(cass.cacheLogFlush)

		//slurp the old logs if any sequence reads will force a write for each distinct
		// sequence ID so the rest of the manifold needs to be set up
		cass.getSequences()

		// now we can write
		close(cass.okToWrite)
		cass.okToWrite = nil

		cass.cacher.Start()

	})
}

func (cass *CassandraLogMetric) Stop() {
	cass.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()
		cass.writer.log.Warning("Starting Shutdown of cassandra series writer")

		if cass.shutitdown {
			return // already did
		}
		cass.shutitdown = true

		cass.cacher.Stop() // stopping forces the current log flush
		// pause to let writer write it
		time.Sleep(1 * time.Second)
		if cass.doRollup {
			cass.rollup.Stop()
		}
		if cass.dispatcher != nil {
			cass.dispatcher.Stop()
		}

		cass.writer.log.Warning("Shutdown finished ... quiting cassandra series writer")
		return
	})
}

// on startup yank the current sequences we may have missed due to crash/restart reassingment
func (cass *CassandraLogMetric) getSequences() {

	cass.writer.log.Notice("Backfilling internal caches from log")

	// due to sorting restrictions in cassandra, we need to get the sequence numbers first then sort them
	// to get things in time order
	Q := fmt.Sprintf(
		"SELECT DISTINCT seq FROM %s_%d_%ds",
		cass.writer.db.LogTableBase(), cass.writerIndex, cass.currentResolution,
	)

	// we need a much large timeout here, as this query and the large amount of data can take a "while"
	sess, err := cass.writer.db.GetSessionWithTimeout(time.Duration(5 * time.Minute))
	if err != nil {
		cass.writer.log.Errorf("Backfill COULD NOT GET Cassandra session, cannot read back logs")
		return
	}
	defer sess.Close()

	// small page size to avoid any frame errors
	seqiter := sess.Query(Q).PageSize(10).Iter()
	sequences := make(utils.Int64List, 0)
	var tSeq int64
	for seqiter.Scan(&tSeq) {
		sequences = append(sequences, tSeq)
	}
	seqiter.Close()
	sort.Sort(sequences)
	curSeq := int64(0)
	added := 0

	// now purge out the sequence number as we've written our data
	Q = fmt.Sprintf(
		"SELECT seq, ts, pts FROM %s_%d_%ds WHERE seq = ? ORDER BY ts ASC",
		cass.writer.db.LogTableBase(), cass.writerIndex, cass.currentResolution,
	)
	delQ := fmt.Sprintf(
		"DELETE FROM %s_%d_%ds WHERE seq = ?",
		cass.writer.db.LogTableBase(), cass.writerIndex, cass.currentResolution,
	)

	if len(sequences) == 0 {
		cass.writer.log.Notice("Backfill No sequence entries, nothing to backfill")
		return
	}
	cass.writer.log.Notice("Backfill have %d logs entries to backfill", len(sequences))

	// set the current sequence number 'higher"
	curSeq = sequences[len(sequences)-1]
	cass.cacher.setCurSequenceId(curSeq)
	cass.cacher.StartChunk(curSeq)

	buf := bytes.NewBuffer(nil)

	for _, s := range sequences {

		iter := sess.Query(Q, s).PageSize(2).Iter()

		var sequence int64
		var ts int64
		var pts []byte
		didChunks := 0
		for iter.Scan(&sequence, &ts, &pts) {
			didChunks++
			cass.writer.log.Notice("Backfill Parsing chunk %d into cache for sequence # %d:%d", didChunks, s, ts)

			// unzstd the punk
			buf.Reset()
			buf.Write(pts)
			decomp := zstd.NewReader(buf)

			var data []byte
			_, err := decomp.Read(data)
			if err != nil {
				cass.writer.log.Errorf("Backfill Error reading compressed slice log data: %v", err)
				decomp.Close()
				continue
			}

			slice := make(map[repr.StatId][]*repr.StatRepr, 0)
			dec := json.NewDecoder(decomp)
			err = dec.Decode(&slice)
			if err != nil {
				cass.writer.log.Errorf("Backfill Error reading slice log data: %v", err)
				decomp.Close()
				continue
			}

			// now we need to add the points to the chunks
			// this can mean our first chunk is longer then the "slice time" but that should be ok
			for _, item := range slice {
				sort.Sort(repr.StatReprSlice(item))
				for _, stat := range item {
					cass.cacher.BackFill(stat.Name, stat, curSeq)
					added++
				}
			}
			decomp.Close()
		}

		err := iter.Close()
		if err != nil {
			cass.writer.log.Errorf("Backfill Iter Error %v", err)
		}

		// need to purge any but the last sequence ID as those represent bad crashes
		// or other interupts and are out of time order so things are messed for them
		if s != curSeq {
			cass.writer.log.Notice("Removing Sequence %d from log", s)
			sess.Query(delQ, s).Exec()
			continue
		}

	}

	// set our sequence version
	cass.writer.log.Notice("Backfilled %d data points from the log.  Current Sequence is %d", added, curSeq)
}

func (cass *CassandraLogMetric) doWriteLog(toLog *CacheChunkLog) {
	tn := time.Now()

	// needs to be in seconds
	ttl := CASSANDRA_LOG_TTL / int64(time.Second)
	Q := fmt.Sprintf(
		"INSERT INTO %s_%d_%ds (seq, ts, pts) VALUES (?,?,?) USING TTL %d",
		cass.writer.db.LogTableBase(), cass.writerIndex, cass.currentResolution, ttl,
	)

	// Cassandra DOES NOT handle really big blobs very well, so we need to chunk the logs
	// in 20k metric chunks, otherwise we get `frame length is bigger than the maximum allowed`
	// errors.  The 20k is an empirical measure from a few test runs of how big the zstd blob gets
	// zstd works best on large blobs.

	buf := bytes.NewBuffer(nil)

	writeSlice := func(slice map[repr.StatId][]*repr.StatRepr) {

		// zstd the punk
		buf.Reset()
		compressed := zstd.NewWriterLevel(buf, zstd.BestSpeed)
		enc := json.NewEncoder(compressed)
		err := enc.Encode(slice)
		compressed.Close()

		if err != nil {
			stats.StatsdClientSlow.Incr("writer.cassandralog.log.writes.errors", 1)
			cass.writer.log.Critical("Json encode error: type cannot write log: %v", err)
			return
		}
		doQ := func() error {
			err = cass.writer.conn.Query(
				Q,
				toLog.SequenceId,
				time.Now().UnixNano(),
				buf.Bytes(),
			).Exec()
			if err != nil {
				stats.StatsdClientSlow.Incr("writer.cassandralog.log.writes.errors", 1)
				cass.writer.log.Critical("Cassandra writer error: cannot write log: %v", err)
				return err
			}
			return nil
		}
		backoff.Retry(doQ, backoff.NewExponentialBackOff())
		stats.StatsdClientSlow.Incr("writer.cassandralog.log.writes.success", 1)
		stats.StatsdSlowNanoTimeFunc("writer.cassandralog.log.write-time-ns", tn)
	}

	slice := toLog.Slice
	chunkSize := 20000 // an empirical measure really
	doSlice := make(map[repr.StatId][]*repr.StatRepr, 0)
	idx := int64(1)
	for k, items := range slice {
		doSlice[k] = items
		if len(doSlice) > chunkSize {
			writeSlice(doSlice)
			doSlice = make(map[repr.StatId][]*repr.StatRepr, 0)
			idx++
		}

	}
	if len(doSlice) > 0 {
		writeSlice(doSlice)
	}
}

// listen for the broad cast chan an flush out a log entry
func (cass *CassandraLogMetric) writeLog() {
	for {
		clog, more := <-cass.loggerChan.Ch
		if !more {
			return
		}
		if clog == nil {
			continue
		}

		toLog, ok := clog.(*CacheChunkLog)
		if !ok {
			stats.StatsdClientSlow.Incr("writer.cassandralog.log.writes.errors", 1)

			cass.writer.log.Critical("Not a CacheChunkLog type cannot write log")
			continue
		}

		cass.doWriteLog(toLog)
	}
}

// listen for the broadcast chan an flush out the newest slice in the mix
func (cass *CassandraLogMetric) writeSlice() {
	for {
		var err error

		clog, more := <-cass.slicerChan.Ch
		if !more {
			return
		}

		if clog == nil {
			continue
		}

		tn := time.Now()

		if !cass.ShouldWrite() {
			cass.writer.log.Notice("ShouldWrite is false, assuming cache only replica, not writing items")
		}

		toLog, ok := clog.(*CacheChunkSlice)
		if !ok {
			stats.StatsdClientSlow.Incr("writer.cassandralog.slice.writes.errors", 1)
			cass.writer.log.Critical("Not a CacheChunkSlice type cannot write log")
			continue
		}

		// do CASSANDRA_LOG_DEFAULT_METRIC_WORKERS process for the first insert blast
		if cass.ShouldWrite() {
			waitG := new(sync.WaitGroup)
			tsChan := make(chan *smetrics.TotalTimeSeries, CASSANDRA_LOG_DEFAULT_METRIC_WORKERS)
			hadErrors := int64(0)
			hadHappy := int64(0)

			retryNotify := func(err error, after time.Duration) {
				stats.StatsdClientSlow.Incr("writer.cassandralog.slice.writes.retry", 1)
				cass.writer.log.Error("Insert Fail Retry :: %v : after %v", err, after)
			}

			doInsert := func() {
				for ts := range tsChan {
					backoffFunc := func() error {
						err = cass.doInsert(ts)
						if err != nil {
							cass.writer.log.Error("Series Insert Error: %v", err)
							atomic.AddInt64(&hadErrors, 1)
							return err
						} else {
							cass.writer.log.Debug(
								"Wrote Series %s (%s) [%d -> %d]",
								ts.Name.Key,
								ts.Name.UniqueIdString(),
								ts.Series.StartTime(),
								ts.Series.LastTime(),
							)

							atomic.AddInt64(&hadHappy, 1)
						}
						waitG.Done()
						return nil
					}

					err = backoff.RetryNotify(backoffFunc, backoff.NewExponentialBackOff(), retryNotify)
					if err != nil {
						cass.writer.log.Error("Insert Fail No More Retries :: %v ", err)
						waitG.Done()
					}
				}
				return
			}

			// fire up a few workers
			for i := 0; i < CASSANDRA_LOG_DEFAULT_METRIC_WORKERS; i++ {
				go doInsert()
			}

			for _, item := range toLog.Slice.AllSeries() {
				// need to "know" we really did write all the series before nuking the log
				// so no dispatching
				//cass.dispatcher.Add(&cassandraBlobMetricJob{Cass: cass, Ts: &smetrics.TotalTimeSeries{Name: item.Name, Series: item.Series}})
				waitG.Add(1)
				tsChan <- &smetrics.TotalTimeSeries{Name: item.Name, Series: item.Series}
			}

			waitG.Wait()
			close(tsChan)
			stats.StatsdClientSlow.Incr("writer.cassandralog.slice.writes.errors", hadErrors)
			stats.StatsdClientSlow.Incr("writer.cassandralog.slice.writes.success", hadHappy)

		} else {
			cass.writer.log.Notice("ShouldWrite is false, assuming cache only replica, not writing items")
		}

		// we still do this for the cache only options
		// now purge out the sequence number as we've written our data
		// note that we don't really care about the "tombstone" issue here as we will never read from
		// this table unless we crash or restart, and the tombstone performance hit is not the
		// end of the world, compaction will eventually take care of these
		Q := fmt.Sprintf(
			"DELETE FROM %s_%d_%ds WHERE seq = ?",
			cass.writer.db.LogTableBase(), cass.writerIndex, cass.currentResolution,
		)
		err = cass.writer.conn.Query(
			Q,
			toLog.SequenceId,
		).Exec()

		if err != nil {
			stats.StatsdClientSlow.Incr("writer.cassandralog.log.purge.errors", 1)
			cass.writer.log.Critical("Cassandra writer error: cannot purge sequence %d: %v", toLog.SequenceId, err)
		} else {
			cass.writer.log.Noticef(
				"Purged log sequence %d for writer %d and resolution %d",
				toLog.SequenceId,
				cass.writerIndex,
				cass.currentResolution,
			)
		}
		stats.StatsdNanoTimeFunc("writer.cassandralog.slice.write-time-ns", tn)
	}
}

// WriteWithOffset write a stat + offset to the cache
func (cass *CassandraLogMetric) WriteWithOffset(stat *repr.StatRepr, offset *smetrics.OffsetInSeries) error {

	// block until ready
	if cass.okToWrite != nil {
		<-cass.okToWrite
	}

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

// Write simple proxy to the WriteWithOffset
func (cass *CassandraLogMetric) Write(stat *repr.StatRepr) error {
	return cass.WriteWithOffset(stat, nil)
}
