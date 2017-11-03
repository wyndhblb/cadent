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
	The "single cacher"

	By single we mean just "one list" of metrics and their timeseries

	Metrics come in and eventually go out.

	1) ability to "buffer/ratelimit" writes so that we don't overwhelm and writing backend
	2) Query things that are not yet written
	3) allow to reject incoming should things get too far behind (the only real recourse for stats influx overload)

	settings

	[acc.agg.writer.metrics]
	driver="blaa"
	dsn="blaa"
	[[acc.agg.writer.metrics.caches]]
		name="protobuf-test"
		metric_size=102400  # number of metric strings to keep
		byte_size=8192 # number of bytes to keep in caches per metric
		series_type="protobuf" # gob, protobuf, json, gorilla
		tag_mode="metrics2"
		historical_chunks=4 # number of previously written chunks to cache for reading speed


	We use the default "gob" series as
		a) it's pretty small, and
		b) we can "sort it" by time (i.e. the format is NOT time-ordered)
		(note: gorilla would be ideal for space, but REQUIRES time ordered inputs)

	Overflow overflow_method ::
		2 overflow modes allowed, an overflow is when there are too make bytes in a series
		based on the cache_byte_size option.

		DROP ::
		`overflow_method="drop"` : just drop the points as we can do no more until the current blob is written

		if the writers are known not to be able to keep up fast enough, this is really the only thing we can do
		otherwise we will run out of ram, and lock on the "chan" method stopping the world basically.

		CHAN ::
		`overflow_method="chan"` : send the current about to expire timeseries + name pair to a channel

		 for the `chan` method to function correctly, it must be set "external" to this object
		 (i.e. the writers/whatever need to give this object the channel to send the overflows)

		 this is useful for writers that use want to write entire "blobs" of timeseries rather then
		 write single points of data.  Basically we want to write the entire cache_byte_size blob in one
		 write action and not the single points.


*/

package metrics

import (
	"cadent/server/series"
	"cadent/server/stats"
	"fmt"
	"math/rand"
	"sort"

	"cadent/server/broadcast"
	smetrics "cadent/server/schemas/metrics"
	"cadent/server/schemas/repr"
	"cadent/server/utils/shutdown"
	"errors"
	logging "gopkg.in/op/go-logging.v1"
	"sync"
	"time"
)

const (
	CACHER_NUMBER_BYTES      = 8192
	CACHER_SERIES_TYPE       = "protobuf"
	CACHER_METRICS_KEYS      = 1024000
	CACHER_DEFAULT_OVERFLOW  = "drop"
	CACHER_CHUNKS_IN_HISTORY = 4

	// for "series" writers using the overflow chan method, we can force a
	// "write" if either nothing has been added in this time (really slow series)
	// OR It's been in RAM this long
	// this helps avoid too much data loss if things crash.
	CACHER_DEFAULT_OVERFLOW_DURATION = 3600

	// max length for the broadcast channel
	CACHER_DEFAULT_BROADCAST_LEN = 256
)

/*
To save disk writes, the way carbon-cache does things is that it maintains an internal timeseries
map[metric]timeseries
once a "worker" gets to the writing of that queue it dumps all the points at once not
one write per point.  The trick is that since graphite "queries" the cache it can backfill all the
not-written points into the return that are still waiting to be written, just from some initial testing
at a rate of 100k points in a 10s window it can take up to 5-15 min for the metrics to actually be written

welcome to diskIO pain, we will try to do the same thing here

a "write" operation will simply add things to this map of points and let a single writer write
the "render" step will then attempt to backfill those points

*/

// struct to pull the next thing to "write"
type CacheQueueItem struct {
	metric repr.StatId
	count  int // number of stats in it
	bytes  int // byte number
}

func (wqi *CacheQueueItem) ToString() string {
	return fmt.Sprintf("Metric Id: %v Count: %d", wqi.metric, wqi.count)
}

// sync.Pool to reduce the "new" CacheQueueItem allocs as we do this a lot

var cacheQueueItemPool sync.Pool

func getCacheQueueItem() *CacheQueueItem {
	x := cacheQueueItemPool.Get()
	if x == nil {
		return new(CacheQueueItem)
	}
	return x.(*CacheQueueItem)
}

func putCacheQueueItem(spl *CacheQueueItem) {
	cacheQueueItemPool.Put(spl)
}

// we pop the "most" populated item
type CacheQueue []*CacheQueueItem

func (v CacheQueue) Len() int           { return len(v) }
func (v CacheQueue) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (v CacheQueue) Less(i, j int) bool { return v[i].count < v[j].count }

// we pop the "most" biggest item
type CacheQueueBytes []*CacheQueueItem

func (v CacheQueueBytes) Len() int           { return len(v) }
func (v CacheQueueBytes) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (v CacheQueueBytes) Less(i, j int) bool { return v[i].bytes < v[j].bytes }

//errors
var errWriteCacheTooManyMetrics = errors.New("Cacher: too many keys, dropping metric")
var errWriteCacheTooManyPoints = errors.New("Cacher: too many points in cache, dropping metric")

// helper function to convert timeSeries to repr.StatReprSlice
func getStatsStream(name *repr.StatName, ts series.TimeSeries) (repr.StatReprSlice, error) {
	it, err := ts.Iter()
	if err != nil {
		return nil, err
	}
	ouStats := make(repr.StatReprSlice, 0)

	lastT := int64(-1)
	idx := -1
	for it.Next() {
		st := it.ReprValue()
		if st == nil {
			continue
		}
		st.Name = name
		// deal w/ same time entries
		if lastT == st.Time && idx > -1 {
			ouStats[idx].Merge(st)
		} else {
			ouStats = append(ouStats, st)
			lastT = st.Time
			idx++
		}

	}
	if it.Error() != nil {
		return ouStats, it.Error()
	}
	if ouStats != nil {
		sort.Sort(ouStats)
	}
	return ouStats, nil
}

type CacheItem struct {
	Name    *repr.StatName
	Series  series.TimeSeries
	Offset  *smetrics.OffsetInSeries
	Started uint32 //keep track of how long we've been in here
}

func (c *CacheItem) Copy() *CacheItem {
	o := new(CacheItem)
	o.Name = c.Name
	o.Series = c.Series
	o.Offset = c.Offset
	o.Started = c.Started
	return o
}

// GetAsRawRenderItem convert the time series to a rendering item
func (ci *CacheItem) GetAsRawRenderItem() (*smetrics.RawRenderItem, error) {

	if ci.Name == nil {
		return nil, nil
	}
	if ci.Series == nil {
		return nil, nil
	}

	data, err := getStatsStream(ci.Name, ci.Series)
	if err != nil || data == nil || len(data) == 0 {
		return nil, err
	}
	rawd := new(smetrics.RawRenderItem)
	rawd.Start = uint32(data[0].ToTime().Unix())
	rawd.End = uint32(data[len(data)-1].ToTime().Unix())
	rawd.RealEnd = rawd.End
	rawd.RealStart = rawd.Start
	rawd.AggFunc = uint32(ci.Name.AggType())

	firstT := uint32(0)
	stepT := uint32(0)

	rawd.Data = make([]*smetrics.RawDataPoint, len(data), len(data))
	for idx, pt := range data {
		onT := uint32(pt.ToTime().Unix())
		rawd.Data[idx] = &smetrics.RawDataPoint{
			Time:  onT,
			Count: pt.Count,
			Min:   float64(pt.Min),
			Max:   float64(pt.Max),
			Last:  float64(pt.Last),
			Sum:   float64(pt.Sum),
		}
		if firstT == 0 {
			firstT = onT
		}
		if stepT == 0 {
			stepT = onT - firstT
		}
	}
	rawd.Step = stepT
	return rawd, nil
}

// CacheHistory this keeps older cache chunks around in ram so we can query things past the typical flush times
// from cache.  Based on the metrics influx, series type and overflow "size" this cache can be big.
type CacheHistory struct {
	Chunks       map[repr.StatId][]*CacheItem `json:"-"`
	chLock       sync.RWMutex
	log          *logging.Logger
	statsdPrefix string
	maxHistory   int
}

// NewCacheHistory object (maxHistory chunks is set to the `CACHER_CHUNKS_IN_HISTORY`)
func NewCacheHistory() *CacheHistory {
	ch := new(CacheHistory)
	ch.Chunks = make(map[repr.StatId][]*CacheItem)
	ch.maxHistory = CACHER_CHUNKS_IN_HISTORY
	return ch
}

// little periodic gather of stats in the cache
func (ch *CacheHistory) stats() {

	tick := time.NewTicker(time.Second * 5)
	for {
		<-tick.C

		ch.chLock.RLock()

		numKeys := len(ch.Chunks)
		numMetrics := 0
		numBytes := 0
		numItems := 0
		for _, items := range ch.Chunks {
			numItems += len(items)
			for _, item := range items {
				numMetrics += item.Series.Count()
				numBytes += item.Series.Len()
			}
		}
		ch.chLock.RUnlock()
		aveChunks := float64(numItems) / float64(numKeys)
		stats.StatsdClientSlow.FGauge(ch.statsdPrefix+"history.chunks", aveChunks)
		stats.StatsdClientSlow.Gauge(ch.statsdPrefix+"history.keys", int64(numKeys))
		stats.StatsdClientSlow.Gauge(ch.statsdPrefix+"history.metrics", int64(numMetrics))
		stats.StatsdClientSlow.Gauge(ch.statsdPrefix+"history.bytes", int64(numBytes))

		ch.log.Info("Cache History: Avg Chunks: %v Keys: %d Metrics: %d Bytes: %d", aveChunks, numKeys, numMetrics, numBytes)

	}
}

// add a CacheItem to the history
func (ch *CacheHistory) AddChunk(chunk *CacheItem) error {
	if ch.maxHistory <= 0 {
		return nil
	}
	if chunk.Series.Count() == 0 {
		return nil
	}
	cp := chunk.Copy()
	id := cp.Name.UniqueId()

	ch.chLock.Lock()
	defer ch.chLock.Unlock()
	if got, ok := ch.Chunks[id]; ok {
		// pop the newest, push to back
		if len(got) >= ch.maxHistory {
			ch.Chunks[id] = ch.Chunks[id][1:len(got)]
			ch.Chunks[id] = append(ch.Chunks[id], cp)
		} else {
			ch.Chunks[id] = append(ch.Chunks[id], cp)
		}
	} else {
		ch.Chunks[id] = make([]*CacheItem, 1)
		ch.Chunks[id][0] = cp
	}
	return nil
}

// GetAsRawRenderItem stats as the nominal output
func (ch *CacheHistory) GetAsRawRenderItem(name *repr.StatName) (rawd *smetrics.RawRenderItem, err error) {
	if ch == nil || len(ch.Chunks) == 0 {
		return nil, nil
	}
	if ch.maxHistory <= 0 {
		return nil, nil
	}
	ch.chLock.RLock()
	defer ch.chLock.RUnlock()

	chunks, ok := ch.Chunks[name.UniqueId()]
	//nothing here
	if !ok {
		return rawd, nil
	}

	// check each chunk
	for _, chunk := range chunks {
		mets, err := chunk.GetAsRawRenderItem()
		if mets == nil {
			continue
		}

		if err != nil {
			ch.log.Error("Failed in Historical Cache Chunks GetAsRawRenderItem: %v", err)
			continue
		}
		if rawd == nil {
			rawd = mets
		} else {
			rawd.Merge(mets)
		}
	}

	return rawd, nil
}

// The "cache" item for points
type CacherSingle struct {
	CacherBase

	log *logging.Logger
	// bookkeeping object to keep track of sizes/bytes/times for each thing in the cache
	Queue CacheQueue `json:"-"`
	// current cache that's written to
	Cache map[repr.StatId]*CacheItem `json:"-"`
	// History chunks of older already written cached items
	History *CacheHistory `json:"-"`
	_accept bool
}

func NewSingleCacher() *CacherSingle {
	wc := new(CacherSingle)
	wc.mu = new(sync.RWMutex)
	wc.qmu = new(sync.Mutex)
	wc.maxKeys = CACHER_METRICS_KEYS
	wc.maxBytes = CACHER_NUMBER_BYTES
	wc.seriesType = CACHER_SERIES_TYPE
	wc.maxTimeInCache = CACHER_DEFAULT_OVERFLOW_DURATION
	wc.overFlowMethod = CACHER_DEFAULT_OVERFLOW
	wc.cacheChunks = CACHER_CHUNKS_IN_HISTORY - 1

	wc.overFlowBroadcast = nil
	wc.PrimaryWriter = nil

	wc.curSize = 0
	wc.log = logging.MustGetLogger("cacher.metrics")
	wc.statsdPrefix = "cacher.metrics."
	wc.Cache = make(map[repr.StatId]*CacheItem)
	wc.shutdown = broadcast.New(0)
	wc._accept = true
	wc.lowFruitRate = 0.25
	wc.started = false
	wc.inited = false

	wc.History = NewCacheHistory()
	wc.History.log = wc.log
	wc.History.maxHistory = wc.CacherBase.cacheChunks
	wc.History.statsdPrefix = wc.statsdPrefix

	wc.overFlowBroadcast = broadcast.New(CACHER_DEFAULT_BROADCAST_LEN)
	return wc
}

func (wc *CacherSingle) SetCacheChunks(m int) {
	wc.cacheChunks = m
	wc.History.maxHistory = m
}

func (wc *CacherSingle) GetCacheChunks() int {
	return wc.cacheChunks
}

func (wc *CacherSingle) Len() int {
	if wc.mu == nil {
		return 0 // means we've not "started"
	}
	wc.mu.RLock()
	defer wc.mu.RUnlock()
	return len(wc.Cache)
}

func (wc *CacherSingle) Start() {
	wc.startstop.Start(func() {
		wc.started = true
		wc.log.Notice("Starting Metric Cache sorter tick (%d max metrics, %d max bytes per metric) [%s]", wc.maxKeys, wc.maxBytes, wc.Name)
		wc.log.Notice("Starting Metric Cache Encoding: %s", wc.seriesType)
		go wc.startUpdateTick()

		// if the overFlowMethod == "chan", then we need to force fire things
		// into the overflow chan if they've been in the hold too long
		go wc.startCacheExpiredTick()

		wc.History.statsdPrefix = wc.statsdPrefix
		go wc.History.stats()
	})
}

func (wc *CacherSingle) Stop() {
	wc.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()

		if wc.started {
			wc.shutdown.Close()
			wc.overFlowBroadcast.Close()
		}
	})
}

func (wc *CacherSingle) DumpPoints(pts []*repr.StatRepr) {
	for idx, pt := range pts {
		wc.log.Notice("TimerSeries: %d Time: %d Sum: %f", idx, pt.Time, pt.Sum)
	}
}

// do this every 5 min and write out any things that need to be \
// flush as they been in the cache too long
func (wc *CacherSingle) startCacheExpiredTick() {

	// only can do this if overflow is chan
	if wc.overFlowMethod != "chan" || wc.overFlowBroadcast == nil {
		return
	}
	wc.log.Notice("Starting Metric Timer for cache expiration write (max time: %ds)", wc.maxTimeInCache)

	tick := time.NewTicker(2 * time.Minute)
	shuts := wc.shutdown.Listen()
	// only push this many per flush as to not overwhelm the backends
	maxPerRun := 1024

	pushExpired := func() {
		tNow := uint32(time.Now().Unix())
		did := 0
		wc.mu.Lock()
		defer wc.mu.Unlock()

		for uniqueId, item := range wc.Cache {
			if (tNow - item.Started) > wc.maxTimeInCache {
				wc.log.Debug(
					"Pushing metric to writer as it's been in ram too long: (%d) (%ds)",
					uniqueId,
					wc.maxTimeInCache,
				)
				// add it to the history
				wc.History.AddChunk(wc.Cache[uniqueId])

				wc.overFlowBroadcast.Send(&smetrics.TotalTimeSeries{
					Name: item.Name, Series: item.Series, Offset: item.Offset,
				})

				wc.curSize -= int64(item.Series.Len()) // shrink the bytes
				delete(wc.Cache, uniqueId)

				stats.StatsdClientSlow.Incr(wc.statsdPrefix+"write.expired", 1)
				did++
				if did > maxPerRun {
					return
				}
			}
		}
	}

	for {
		select {
		case <-tick.C:
			pushExpired()
		case <-shuts.Ch:
			tick.Stop()
			wc._accept = false
			wc.log.Warning("Cache shutdown .. stopping expire tick")
			return
		}
	}
}

// do this only once a second as it can be expensive
func (wc *CacherSingle) startUpdateTick() {

	tick := time.NewTicker(2 * time.Second)
	shuts := wc.shutdown.Listen()
	for {
		select {
		case <-tick.C:
			wc.updateQueue()
		case <-shuts.Ch:
			tick.Stop()
			wc._accept = false
			wc.log.Warning("Cache shutdown .. stopping accepts")

			return
		}
	}
}

func (wc *CacherSingle) updateQueue() {
	if !wc._accept {
		return
	}
	f_len := 0
	idx := 0
	wc.mu.RLock()
	newQueue := make(CacheQueue, len(wc.Cache))

	for _, putback := range wc.Queue {
		putCacheQueueItem(putback)
	}

	for key, values := range wc.Cache {
		num_points := values.Series.Count()
		cQItem := getCacheQueueItem()
		cQItem.metric = key
		cQItem.count = num_points
		cQItem.bytes = values.Series.Len()
		newQueue[idx] = cQItem
		idx++
		f_len += num_points
	}
	wc.mu.RUnlock()

	wc.numCurPoint = f_len
	m_len := len(newQueue)
	wc.numCurKeys = m_len
	sort.Sort(newQueue)

	wc.log.Debug("Cacher: %s Sort : Metrics: %v :: Points: %v :: Bytes:: %d", wc.Name, m_len, f_len, wc.curSize)

	stats.StatsdClientSlow.Gauge(wc.statsdPrefix+"metrics", int64(m_len))
	stats.StatsdClientSlow.Gauge(wc.statsdPrefix+"points", int64(f_len))
	stats.StatsdClientSlow.Gauge(wc.statsdPrefix+"bytes", wc.curSize)

	// now for a bit of randomness, where we "reverse" the order on occasion to get the
	// not-updated often and thus hardly written to try to persist some slow stats
	// do this 1/4 of the time, so that we don't end up with having to shutdown in order to all things written
	r := rand.Float64()
	if r < wc.lowFruitRate {
		for i, j := 0, len(newQueue)-1; i < j; i, j = i+1, j-1 {
			newQueue[i], newQueue[j] = newQueue[j], newQueue[i]
		}
	}

	wc.qmu.Lock()
	defer wc.qmu.Unlock()
	wc.Queue = nil
	wc.Queue = newQueue
}

// add metric then update the sort queue
// use this for more "direct" writing for very small caches
func (wc *CacherSingle) AddAndUpdate(metric *repr.StatName, stat *repr.StatRepr) (err error) {
	err = wc.Add(metric, stat)
	wc.updateQueue()
	return err
}

func (wc *CacherSingle) Add(name *repr.StatName, stat *repr.StatRepr) error {
	return wc.AddWithOffset(name, stat, nil)
}

func (wc *CacherSingle) AddWithOffset(name *repr.StatName, stat *repr.StatRepr, offset *smetrics.OffsetInSeries) error {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	if !wc._accept {
		//wc.log.Error("Shutting down, will not add any more items to the queue")
		return nil
	}

	if len(wc.Cache) > wc.maxKeys {
		wc.log.Error("Key Cache is too large .. over %d metrics keys, have to drop this one", wc.maxKeys)
		stats.StatsdClientSlow.Incr(wc.statsdPrefix+"overflow", 1)
		return errWriteCacheTooManyMetrics
	}

	/** ye old debuggin'
	if strings.Contains(metric, "flushesposts") {
		wc.log.Critical("ADDING: %s Time: %d, Val: %f", metric, time, value)
	}
	*/
	tNow := uint32(time.Now().Unix())
	uniqueId := name.UniqueId()

	if gots, ok := wc.Cache[uniqueId]; ok && gots != nil {
		curSize := gots.Series.Len()
		if curSize > wc.maxBytes {

			// if the overflow method is chan, and there is valid overFLowChan, we "pop" the item from
			// the cache and send it to the chan (note we're already "locked" here)
			if wc.overFlowBroadcast != nil && wc.overFlowMethod == "chan" {
				// add it to the history
				wc.History.AddChunk(gots)
				wc.curSize -= int64(curSize) // shrink the bytes
				if !wc.overFlowBroadcast.IsClosed() {
					nTs := new(smetrics.TotalTimeSeries)
					nTs.Series = gots.Series
					nTs.Offset = gots.Offset
					nTs.Name = gots.Name
					wc.overFlowBroadcast.Send(nTs)
					stats.StatsdClientSlow.Incr(wc.statsdPrefix+"write.overflow", 1)
				}
				delete(wc.Cache, uniqueId)

				// break out of this loop and add a new guy
				goto NEWSTAT
			}

			wc.log.Error("Too Many Bytes for %v (max bytes: %d current metrics: %v)... have to drop this one", uniqueId, wc.maxBytes, gots.Series.Count())
			stats.StatsdClientSlow.Incr(wc.statsdPrefix+"points.overflow", 1)
			return errWriteCacheTooManyPoints
		}
		err := wc.Cache[uniqueId].Series.AddStat(stat)
		if err != nil {
			wc.log.Errorf("Error writing point to series: %s", err)
			return err
		}

		// add a new offset if present
		if offset != nil {
			if wc.Cache[uniqueId].Offset == nil {
				wc.Cache[uniqueId].Offset = offset
			} else {
				if wc.Cache[uniqueId].Offset.Offset < offset.Offset {
					wc.Cache[uniqueId].Offset.Offset = offset.Offset
				}
			}
		}

		nowLen := wc.Cache[uniqueId].Series.Len()
		wc.curSize += int64(nowLen - curSize)
		stats.StatsdClient.GaugeAvg(wc.statsdPrefix+"add.ave-points-per-metric", int64(gots.Series.Count()))
		return nil
	}

NEWSTAT:
	tp, err := series.NewTimeSeries(wc.seriesType, stat.Time, nil)
	if err != nil {
		return err
	}

	err = tp.AddStat(stat)
	if err != nil {
		wc.log.Errorf("Error writing point to series: %s", err)
		return err
	}

	wc.Cache[uniqueId] = new(CacheItem)

	wc.Cache[uniqueId].Series = tp
	wc.Cache[uniqueId].Name = name
	wc.Cache[uniqueId].Started = tNow
	wc.Cache[uniqueId].Offset = offset

	wc.curSize += int64(tp.Len())
	stats.StatsdClient.GaugeAvg(wc.statsdPrefix+"add.ave-points-per-metric", 1)

	return nil
}

func (wc *CacherSingle) Get(name *repr.StatName) (repr.StatReprSlice, error) {
	stats.StatsdClientSlow.Incr(wc.statsdPrefix+"read.cache-gets", 1)

	wc.mu.RLock()
	defer wc.mu.RUnlock()

	if gots, ok := wc.Cache[name.UniqueId()]; ok {
		stats.StatsdClientSlow.Incr(wc.statsdPrefix+"read.cache-gets.values", 1)
		return getStatsStream(name, gots.Series)
	}
	stats.StatsdClientSlow.Incr(wc.statsdPrefix+"read.cache-gets.empty", 1)
	return nil, nil
}

func (wc *CacherSingle) Exists(name *repr.StatName) (bool, error) {
	if _, ok := wc.Cache[name.UniqueId()]; ok {
		return true, nil
	}
	return false, nil
}

func (wc *CacherSingle) CacheList() ([]*repr.StatName, error) {
	wc.mu.RLock()
	defer wc.mu.RUnlock()
	rList := make([]*repr.StatName, 0)
	for _, s := range wc.Cache {
		if s == nil || s.Name == nil {
			continue
		}
		rList = append(rList, s.Name)
	}
	return rList, nil
}

func (wc *CacherSingle) GetAsRawRenderItem(name *repr.StatName) (*smetrics.RawRenderItem, error) {
	stats.StatsdClientSlow.Incr("cacher.read.cache-gets", 1)
	if wc == nil || wc.mu == nil {
		return nil, nil
	}
	// check the history
	rawd, err := wc.History.GetAsRawRenderItem(name)

	if err != nil {
		wc.log.Errorf("Error getting from history: %v", err)
	}

	wc.mu.RLock()
	defer wc.mu.RUnlock()
	if gots, ok := wc.Cache[name.UniqueId()]; ok {
		stats.StatsdClientSlow.Incr(wc.statsdPrefix+"read.cache-gets.values", 1)
		auxR, err := gots.GetAsRawRenderItem()
		if err != nil {
			wc.log.Errorf("Error getting from cache: %v", err)
		}
		if err == nil && auxR != nil && len(auxR.Data) > 0 {
			if rawd != nil && len(rawd.Data) > 0 {
				auxR.Merge(rawd)
			}
			return auxR, nil
		}
	}
	return rawd, nil
}

func (wc *CacherSingle) GetById(metricId repr.StatId) (*repr.StatName, repr.StatReprSlice, error) {
	stats.StatsdClientSlow.Incr("cacher.read.cache-gets", 1)
	if wc == nil || wc.mu == nil {
		return nil, nil, nil
	}
	wc.mu.RLock()
	defer wc.mu.RUnlock()

	if gots, ok := wc.Cache[metricId]; ok {
		stats.StatsdClientSlow.Incr(wc.statsdPrefix+"read.cache-gets.values", 1)
		tseries, err := getStatsStream(gots.Name, gots.Series)
		return gots.Name, tseries, err
	}
	stats.StatsdClientSlow.Incr(wc.statsdPrefix+"read.cache-gets.empty", 1)
	return nil, nil, nil
}

func (wc *CacherSingle) GetSeries(name *repr.StatName) (*repr.StatName, []series.TimeSeries, error) {
	stats.StatsdClientSlow.Incr(wc.statsdPrefix+"read.cache-series-gets", 1)

	wc.mu.RLock()
	defer wc.mu.RUnlock()

	if gots, ok := wc.Cache[name.UniqueId()]; ok {
		stats.StatsdClientSlow.Incr(wc.statsdPrefix+"read.cache-series-gets.values", 1)
		return gots.Name, []series.TimeSeries{gots.Series}, nil
	}
	stats.StatsdClientSlow.Incr(wc.statsdPrefix+"read.cache-series-gets.empty", 1)
	return nil, nil, nil
}

func (wc *CacherSingle) GetCurrentSeries(name *repr.StatName) (*repr.StatName, series.TimeSeries, error) {
	nm, ts, err := wc.GetSeries(name)
	if ts != nil && len(ts) > 0 {
		return nm, ts[0], err
	}
	return nm, nil, err
}

func (wc *CacherSingle) GetSeriesById(metricId repr.StatId) (*repr.StatName, []series.TimeSeries, error) {
	stats.StatsdClientSlow.Incr(wc.statsdPrefix+"read.cache-series-by-idgets", 1)

	wc.mu.RLock()
	defer wc.mu.RUnlock()

	if gots, ok := wc.Cache[metricId]; ok {
		stats.StatsdClientSlow.Incr(wc.statsdPrefix+"read.cache-series-by-idgets.values", 1)
		return gots.Name, []series.TimeSeries{gots.Series}, nil
	}
	stats.StatsdClientSlow.Incr(wc.statsdPrefix+"read.cache-series-by-idgets.empty", 1)
	return nil, nil, nil
}

func (wc *CacherSingle) GetCurrentSeriesById(metricId repr.StatId) (*repr.StatName, series.TimeSeries, error) {
	nm, ts, err := wc.GetSeriesById(metricId)
	if ts != nil && len(ts) > 0 {
		return nm, ts[0], err
	}
	return nm, nil, err
}

func (wc *CacherSingle) GetNextMetric() *repr.StatName {

	for {
		wc.qmu.Lock()
		size := len(wc.Queue)
		if size == 0 {
			wc.qmu.Unlock()
			break
		}
		item := wc.Queue[size-1]
		wc.Queue = wc.Queue[:size-1]
		wc.qmu.Unlock()

		wc.mu.RLock()
		v := wc.Cache[item.metric]
		wc.mu.RUnlock()

		if v == nil {
			wc.mu.Lock()
			delete(wc.Cache, item.metric)
			wc.mu.Unlock()
			continue
		}
		return v.Name
	}
	return nil
}

// just grab something from the list
func (wc *CacherSingle) GetAnyStat() (name *repr.StatName, stats repr.StatReprSlice) {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	for uid, stats := range wc.Cache {
		out, _ := getStatsStream(stats.Name, stats.Series)
		wc.curSize -= int64(stats.Series.Len()) // shrink the bytes
		delete(wc.Cache, uid)                   // need to purge if error as things are corrupted somehow
		return name, out
	}
	return nil, nil
}

func (wc *CacherSingle) Pop() (*repr.StatName, repr.StatReprSlice) {
	metric, ts := wc.PopSeries()
	if ts == nil {
		return nil, nil
	}
	out, _ := getStatsStream(metric, ts)
	return metric, out
}

func (wc *CacherSingle) PopSeries() (*repr.StatName, series.TimeSeries) {
	metric := wc.GetNextMetric()
	if metric != nil {
		wc.mu.Lock()
		defer wc.mu.Unlock()
		uniqueId := metric.UniqueId()
		if stats, exists := wc.Cache[uniqueId]; exists {
			wc.curSize -= int64(stats.Series.Len()) // shrink the bytes
			delete(wc.Cache, uniqueId)              // need to delete regardless as we have byte errors and things are corrupted

			return stats.Name, stats.Series
		}
	}
	return nil, nil
}

// add a metrics/point list back on the queue as it either "failed" or was ratelimited
func (wc *CacherSingle) AddBack(name *repr.StatName, points repr.StatReprSlice) {
	for _, pt := range points {
		wc.Add(name, pt)
	}
}
