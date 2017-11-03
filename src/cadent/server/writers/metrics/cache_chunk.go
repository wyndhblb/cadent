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
	The cache chunk

	maintains "N" caches where each chunk represents a time slice (rather then size based)

	this is used in the *-log writers

	The "current" chunk also maintains another "last CACHE_LOG_FLUSH (10s)" state, this state
	is then flushed to the LogWriter for writing of the entire blob

	writers that use this, need two channel responders

	1. chunk channel: the cacher will push metrics in the chunk
	 that needs to be written as "normal timeseries"

	2. log channel: will push "current" log to dump to the log tables/files

	The writer is responsible for removing the log chunks once the series have all been written

	NOTE:
		this is NOT data-race safe, it is assumed ONE writer object will be adding items to the cache
		(several writers can get the message broadcast however)
		there is some internal locking to get certain book keeping in order, but EXPECT `go test -race`
		to toss a lot of warnings

		We have to do it this way to avoid waiting in internal chan locks or semaphore hell
		as we need to add LOTS of metrics in short order


*/

package metrics

import (
	"cadent/server/broadcast"
	"cadent/server/schemas/metrics"
	"cadent/server/schemas/repr"
	"cadent/server/series"
	"cadent/server/stats"
	"cadent/server/utils/shutdown"
	"errors"
	"fmt"
	logging "gopkg.in/op/go-logging.v1"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// how many chunks to keep in RAM
	CACHE_LOG_CHUNKS = 6

	// hold CACHE_LOG_TIME_CHUNKS duration per timeseries
	CACHE_LOG_TIME_CHUNKS = uint32(10 * 60)

	// every CACHE_LOG_FLUSH duration, drop the current set of points into the log table
	CACHE_LOG_FLUSH = uint32(10)
)

var ErrZeroTime = errors.New("Time for metric is 0 cannot insert")

// CacheChunk singletons as the readers may need to use this too

// the singleton
var _CHUNK_CACHER_SINGLETON map[string]*CacherChunk
var _chunk_cacher_mutex sync.RWMutex

func GetCacherChunkSingleton(nm string) (*CacherChunk, error) {
	_chunk_cacher_mutex.Lock()
	defer _chunk_cacher_mutex.Unlock()

	if val, ok := _CHUNK_CACHER_SINGLETON[nm]; ok {
		return val, nil
	}

	cacher := NewCacherChunk()
	_CHUNK_CACHER_SINGLETON[nm] = cacher
	cacher.SetName(nm)
	return cacher, nil
}

// just GET by name if it exists
func GetCacherChunkByName(nm string) *CacherChunk {
	_chunk_cacher_mutex.RLock()
	defer _chunk_cacher_mutex.RUnlock()

	if val, ok := _CHUNK_CACHER_SINGLETON[nm]; ok {
		return val
	}
	return nil
}

// special onload init
func init() {
	_CHUNK_CACHER_SINGLETON = make(map[string]*CacherChunk)
}

type cacheChunkItem struct {
	sync.RWMutex
	ts         map[repr.StatId]*CacheItem // list of time series in this chunk
	seriesType string
	sequence   int64 // sequence that match the Log portion
	started    int64
	timeWindow int64
	written    bool
	curCount   int64
}

func NewCacheChunkItem(timeW int64, sequenceNum int64, seriesType string) *cacheChunkItem {
	c := new(cacheChunkItem)
	c.Lock()
	c.timeWindow = timeW
	c.sequence = sequenceNum
	c.seriesType = seriesType
	c.started = time.Now().UnixNano()
	c.ts = make(map[repr.StatId]*CacheItem)
	c.curCount = 0
	c.written = false
	c.Unlock()
	return c
}

func (c *cacheChunkItem) Copy() *cacheChunkItem {

	out := new(cacheChunkItem)
	out.curCount = c.curCount
	out.written = c.written
	out.timeWindow = c.timeWindow
	out.sequence = c.sequence
	out.seriesType = c.seriesType
	out.started = c.started

	out.ts = make(map[repr.StatId]*CacheItem)

	c.RLock()
	for k, v := range c.ts {
		out.ts[k] = v
	}
	c.RUnlock()
	return out
}

func (c *cacheChunkItem) AllSeries() map[repr.StatId]*CacheItem {
	return c.ts
}

// Len number of metrics in the cache item
func (c *cacheChunkItem) Len() int {
	if c == nil {
		return 0
	}
	c.RLock()
	defer c.RUnlock()
	return len(c.ts)
}

// Count number of points
func (c *cacheChunkItem) Count() int {
	return int(atomic.LoadInt64(&c.curCount))
}

func (c *cacheChunkItem) Add(name *repr.StatName, stat *repr.StatRepr) error {
	return c.AddWithOffset(name, stat, nil)
}

func (c *cacheChunkItem) AddWithOffset(name *repr.StatName, stat *repr.StatRepr, offset *metrics.OffsetInSeries) error {
	var err error
	if stat.Time == 0 {
		return ErrZeroTime
	}

	c.Lock()
	defer c.Unlock()

	uid := name.UniqueId()
	if _, ok := c.ts[uid]; ok {
		err = c.ts[uid].Series.AddStat(stat)
		if err != nil {
			return err
		}
		c.ts[uid].Offset = offset // update the offset to the current one
		atomic.AddInt64(&c.curCount, 1)
		return nil
	}

	tp, err := series.NewTimeSeries(c.seriesType, stat.Time, nil)
	if err != nil {
		return err
	}
	err = tp.AddStat(stat)
	if err != nil {
		return err
	}
	c.ts[uid] = &CacheItem{Series: tp, Name: name, Started: uint32(time.Now().Unix()), Offset: offset}
	atomic.AddInt64(&c.curCount, 1)
	return nil
}

// Get a stat from the name
func (c *cacheChunkItem) Get(name *repr.StatName) (repr.StatReprSlice, error) {
	c.RLock()
	defer c.RUnlock()
	if gots, ok := c.ts[name.UniqueId()]; ok {
		return c.getStatsStream(name, gots.Series)
	}
	return nil, nil
}

func (c *cacheChunkItem) getStatsStream(name *repr.StatName, ts series.TimeSeries) (repr.StatReprSlice, error) {
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
			ouStats[idx].MergeInPlace(st)

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

func (wc *cacheChunkItem) GetAsRawRenderItem(name *repr.StatName) (*metrics.RawRenderItem, error) {

	name, data, err := wc.GetById(name.UniqueId())

	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	rawd := new(metrics.RawRenderItem)
	rawd.Start = uint32(data[0].ToTime().Unix())
	rawd.End = uint32(data[len(data)-1].ToTime().Unix())
	rawd.RealEnd = rawd.End
	rawd.RealStart = rawd.Start
	rawd.AggFunc = uint32(name.AggType())

	fT := uint32(0)
	stepT := uint32(0)

	rawd.Data = make([]*metrics.RawDataPoint, len(data), len(data))
	for idx, pt := range data {
		on_t := uint32(pt.ToTime().Unix())
		rawd.Data[idx] = &metrics.RawDataPoint{
			Time:  on_t,
			Count: pt.Count,
			Min:   float64(pt.Min),
			Max:   float64(pt.Max),
			Last:  float64(pt.Last),
			Sum:   float64(pt.Sum),
		}
		if fT == 0 {
			fT = on_t
		}
		if stepT == 0 {
			stepT = on_t - fT
		}
	}
	rawd.Step = stepT
	return rawd, nil
}

func (wc *cacheChunkItem) GetById(metric_id repr.StatId) (*repr.StatName, repr.StatReprSlice, error) {
	if wc == nil {
		return nil, nil, nil
	}
	wc.RLock()
	defer wc.RUnlock()

	if gots, ok := wc.ts[metric_id]; ok {
		tseries, err := wc.getStatsStream(gots.Name, gots.Series)
		return gots.Name, tseries, err
	}
	return nil, nil, nil
}

func (wc *cacheChunkItem) GetSeries(name *repr.StatName) (*repr.StatName, series.TimeSeries, error) {
	if wc == nil {
		return nil, nil, nil
	}
	wc.RLock()
	defer wc.RUnlock()

	if gots, ok := wc.ts[name.UniqueId()]; ok {
		return gots.Name, gots.Series, nil
	}
	return nil, nil, nil
}

func (wc *cacheChunkItem) GetSeriesById(metric_id repr.StatId) (*repr.StatName, series.TimeSeries, error) {
	if wc == nil {
		return nil, nil, nil
	}
	wc.RLock()
	defer wc.RUnlock()

	if gots, ok := wc.ts[metric_id]; ok {
		return gots.Name, gots.Series, nil
	}
	return nil, nil, nil
}

func (c *cacheChunkItem) ToString() string {
	return fmt.Sprintf("Chunk Cache Started: %v Count: %d", c.started, len(c.ts))
}

func (c *cacheChunkItem) NeedFlush() bool {
	return time.Now().UnixNano()-c.timeWindow >= c.started
}

/********************* Chunk Cacher Log object *****************************/
type CacheChunkLog struct {
	Slice      map[repr.StatId][]*repr.StatRepr // the "current" list of things to write to the log
	SequenceId int64
}

/********************* Chunk Cacher Slice object *****************************/
type CacheChunkSlice struct {
	Slice      *cacheChunkItem // the "current" list of things to write to the log
	SequenceId int64
}

/********************* Chunk Cacher objects *****************************/

type adderMessage struct {
	name *repr.StatName
	stat *repr.StatRepr
	uid  repr.StatId
}

type CacherChunk struct {
	CacherBase

	chmu *sync.RWMutex
	slmu *sync.RWMutex

	chunks        []*cacheChunkItem                // the N chunks
	curChunk      *cacheChunkItem                  // curnet chunk we are on
	curSlice      map[repr.StatId][]*repr.StatRepr // the "current" list of things to write to the log
	maxChunks     uint32
	maxTime       uint32
	logTime       uint32
	curSequenceId int64 // what is the current log sequence we are on
	statCount     int
	metricCount   int
	log           *logging.Logger

	addSliceChan chan *adderMessage
	addChunkChan chan *adderMessage

	// broadcaster for logging writer
	logBroadcast   *broadcast.Broadcaster // pass to the log elements
	sliceBroadcast *broadcast.Broadcaster // pass to the Writer element

}

func NewCacherChunk() *CacherChunk {
	wc := new(CacherChunk)
	wc.mu = new(sync.RWMutex)
	wc.chmu = new(sync.RWMutex)
	wc.slmu = new(sync.RWMutex)

	wc.maxChunks = CACHE_LOG_CHUNKS
	wc.maxTime = CACHE_LOG_TIME_CHUNKS
	wc.logTime = CACHE_LOG_FLUSH
	wc.seriesType = CACHER_SERIES_TYPE

	wc.log = logging.MustGetLogger("metrics.chunkcacher")
	wc.statsdPrefix = "chunkcacher.metrics."
	wc.curSlice = make(map[repr.StatId][]*repr.StatRepr)
	wc.chunks = make([]*cacheChunkItem, CACHE_LOG_CHUNKS)
	wc.logBroadcast = broadcast.New(1)
	wc.sliceBroadcast = broadcast.New(1)
	wc.shutdown = broadcast.New(10)
	wc.addChunkChan = make(chan *adderMessage, 32)
	wc.addSliceChan = make(chan *adderMessage, 32)

	// general chan comms
	go wc.addStatToChunk()
	go wc.addStatToSlice()

	return wc
}

// CurrentLog returns the current chunk of stats that need to get put into the log
func (wc *CacherChunk) CurrentLog() map[repr.StatId][]*repr.StatRepr {
	return wc.curSlice
}

// CurrentLog returns the current chunk of stats that need to get put into the log
func (wc *CacherChunk) CurrenSequencId() int64 {
	return wc.getCurSequenceId()
}

func (wc *CacherChunk) SetMaxChunks(m uint32) {
	wc.maxChunks = m
}

// SetChunkTimeWinow the max time each chunk can be contain for default of 10min
func (wc *CacherChunk) SetChunkTimeWindow(m uint32) {
	wc.maxTime = m
}

// SetChunkTimeWinow the max time each chunk can be contain for default of 10min
func (wc *CacherChunk) SetLogTimeWindow(m uint32) {
	wc.logTime = m
}

func (wc *CacherChunk) Len() int {
	wc.chmu.RLock()
	l := wc.curChunk.Len()
	wc.chmu.RUnlock()
	for _, c := range wc.chunks {
		l += c.Len()
	}
	return l
}

func (wc *CacherChunk) SliceLen() int {
	wc.slmu.RLock()
	defer wc.slmu.RUnlock()
	return len(wc.curSlice)
}

func (wc *CacherChunk) ChunkLen() int {
	wc.chmu.RLock()
	defer wc.chmu.RUnlock()
	return wc.curChunk.Len()
}

func (wc *CacherChunk) GetLogChan() *broadcast.Listener {
	return wc.logBroadcast.Listen()
}

func (wc *CacherChunk) GetSliceChan() *broadcast.Listener {
	return wc.sliceBroadcast.Listen()
}

// a little statsd emitter for cache stats
func (wc *CacherChunk) emitStats() {
	tick := time.NewTicker(time.Duration(int64(wc.logTime) / 2 * int64(time.Second)))
	for {
		<-tick.C

		cl := wc.ChunkLen()
		stats.StatsdClientSlow.Gauge("cacherlog.chunk.metrics", int64(cl))
		sl := wc.SliceLen()
		stats.StatsdClientSlow.Gauge("cacherlog.slice.metrics", int64(sl))
	}
}

// Start the cacher
func (wc *CacherChunk) Start() {
	wc.startstop.Start(func() {
		wc.log.Notice("Starting Metric Chunk Cache sorter tick (%d max chunks, %d second Time Window) [%s]", wc.maxChunks, wc.maxTime, wc.Name)
		wc.log.Notice("Starting Metric Chunk Cache Encoding: %s", wc.seriesType)
		wc.log.Notice("Starting Metric Chunk Cache Log Runner every: %d seconds", wc.logTime)
		go wc.runLogDump()
		go wc.cycleChunks()
		go wc.emitStats()
	})
}

// Stop the cacher
func (wc *CacherChunk) Stop() {
	wc.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()
		wc.shutdown.Close()
	})
}

func (wc *CacherChunk) CurrentLogChunk() *CacheChunkLog {
	// need to clone it to avoid overwriting
	tmp := make(map[repr.StatId][]*repr.StatRepr)
	wc.slmu.RLock()
	for k, v := range wc.curSlice {
		tmp[k] = v
	}
	wc.slmu.RUnlock()
	return &CacheChunkLog{
		Slice:      tmp,
		SequenceId: wc.getCurSequenceId(),
	}
}

// every logtime dump to logs chan
func (wc *CacherChunk) runLogDump() {
	tick := time.NewTicker(time.Duration(int64(wc.logTime) * int64(time.Second)))
	shuts := wc.shutdown.Listen()
	for {
		select {
		case <-tick.C:
			sl := wc.SliceLen()
			wc.log.Info("Flushing log to writer for %d metrics for %s sequence %d", sl, wc.Name, wc.getCurSequenceId())
			stats.StatsdClientSlow.Incr("cacherlog.log.flush.metrics.count", int64(sl))
			// need to clone it to avoid overwriting
			cLog := wc.CurrentLogChunk()
			wc.logBroadcast.Send(cLog)
			wc.slmu.Lock()
			wc.curSlice = make(map[repr.StatId][]*repr.StatRepr)
			wc.slmu.Unlock()

		case <-shuts.Ch:
			wc.log.Notice("Shutdown of log dump, doing final log write")
			cLog := wc.CurrentLogChunk()
			wc.logBroadcast.Send(cLog)
			shuts.Close()
			return
		}
	}
}

func (wc *CacherChunk) cycleInternalChunks() *CacheChunkSlice {
	wc.chmu.Lock()
	defer wc.chmu.Unlock()

	if wc.curChunk == nil {
		atomic.AddInt64(&wc.curSequenceId, 1)
		wc.curChunk = NewCacheChunkItem(time.Now().UnixNano(), wc.getCurSequenceId(), wc.seriesType)
		return nil

	}

	wc.chunks = wc.chunks[1:len(wc.chunks)]
	wc.chunks = append(wc.chunks, wc.curChunk)

	// need to copy the data
	outchunk := wc.curChunk.Copy()
	outSlice := &CacheChunkSlice{
		Slice:      outchunk,
		SequenceId: wc.curChunk.sequence,
	}

	atomic.AddInt64(&wc.curSequenceId, 1)

	wc.curChunk = NewCacheChunkItem(time.Now().UnixNano(), wc.getCurSequenceId(), wc.seriesType)
	return outSlice
}

// force a write of the current chunk and wipe it out
// this is used for startup cache reads
func (wc *CacherChunk) ForceChunkWrite() {
	wc.log.Info("Sending sequence %d to writers for %d metrics", wc.getCurSequenceId(), wc.SliceLen())

	c := wc.cycleInternalChunks()
	if c != nil {
		wc.sliceBroadcast.Send(c)
	}
}

// every chunk time, push the current slice into the overflow channel for writing
// update the sequence ID we're on .. pop the earliest chunk off the queue and add this
// old chunk to the queue
func (wc *CacherChunk) cycleChunks() {
	tick := time.NewTicker(time.Duration(int64(wc.maxTime) * int64(time.Second)))
	wc.log.Notice("Starting Chunk cycler: rotating every %d seconds", int64(wc.maxTime))
	shuts := wc.shutdown.Listen()
	for {
		select {
		case <-tick.C:
			wc.log.Info("Sending sequence %d to writers for %d metrics", wc.getCurSequenceId(), wc.SliceLen())
			c := wc.cycleInternalChunks()
			if c != nil && c.Slice != nil && c.Slice.Len() > 0 {
				cl := c.Slice.Len()
				stats.StatsdClientSlow.Incr("cacherlog.log.cycle.flush.series.count", int64(cl))
				if c != nil {
					wc.sliceBroadcast.Send(c)
				}
			}

		case <-shuts.Ch:
			wc.log.Notice("Shutdown of log dump, doing final log write")
			shuts.Close()
			return
		}
	}
}

func (wc *CacherChunk) getCurSequenceId() int64 {
	return atomic.LoadInt64(&wc.curSequenceId)
}

func (wc *CacherChunk) setCurSequenceId(s int64) {
	atomic.StoreInt64(&wc.curSequenceId, s)
}

func (wc *CacherChunk) addStatToChunk() error {
	for {
		item := <-wc.addChunkChan
		wc.chmu.Lock()
		if wc.curChunk == nil {
			wc.curChunk = NewCacheChunkItem(time.Now().UnixNano(), wc.getCurSequenceId(), wc.seriesType)
		}
		wc.chmu.Unlock()

		wc.chmu.RLock()
		err := wc.curChunk.Add(item.name, item.stat)
		wc.chmu.RUnlock()

		if err != nil {
			wc.log.Errorf("Error Adding stat: %v", err)
		}

	}
}

func (wc *CacherChunk) addStatToSlice() error {
	for {
		item := <-wc.addSliceChan
		uid := item.uid
		stat := item.stat
		wc.slmu.Lock()
		if _, ok := wc.curSlice[uid]; ok {
			wc.curSlice[uid] = append(wc.curSlice[uid], stat)
		} else {
			wc.curSlice[uid] = []*repr.StatRepr{stat}
		}
		wc.slmu.Unlock()

	}
}

func (wc *CacherChunk) StartChunk(sequence int64) {
	wc.chmu.Lock()
	defer wc.chmu.Unlock()
	wc.curChunk = NewCacheChunkItem(time.Now().UnixNano(), sequence, wc.seriesType)
}

// Add a metric to the cache
func (wc *CacherChunk) Add(name *repr.StatName, stat *repr.StatRepr) error {
	return wc.AddWithOffset(name, stat, nil)
}

// Add a metric to the cache
func (wc *CacherChunk) AddWithOffset(name *repr.StatName, stat *repr.StatRepr, offset *metrics.OffsetInSeries) error {
	// need to add it to curSlice as well
	uid := name.UniqueId()
	//a := &adderMessage{name: name, stat: stat, uid: uid}

	// lock both chunk and cur at the same time to avoid data added to one but not the other
	// which may lead to a little race while adding in the middle of in the cycleChunks/logSlice parts
	wc.slmu.Lock()
	defer wc.slmu.Unlock()
	wc.chmu.Lock()
	defer wc.chmu.Unlock()

	if _, ok := wc.curSlice[uid]; ok {
		wc.curSlice[uid] = append(wc.curSlice[uid], stat)
	} else {
		wc.curSlice[uid] = []*repr.StatRepr{stat}
	}
	if wc.curChunk == nil {
		wc.curChunk = NewCacheChunkItem(time.Now().UnixNano(), wc.getCurSequenceId(), wc.seriesType)
	}

	err := wc.curChunk.AddWithOffset(name, stat, offset)

	if err != nil {
		wc.log.Errorf("Error Adding stat: %v", err)
	}

	return err
}

// BackFill the chunks, but not the Main Current list, NOTE not thread safe and should be used serially
// as stats in time order is a factor
func (wc *CacherChunk) BackFill(name *repr.StatName, stat *repr.StatRepr, sequence int64) error {
	b := &adderMessage{name: name, stat: stat, uid: name.UniqueId()}
	wc.addChunkChan <- b
	return nil
}

// Get stats from the cache
func (wc *CacherChunk) Get(name *repr.StatName) (slice repr.StatReprSlice, err error) {
	stats.StatsdClientSlow.Incr(wc.statsdPrefix+"read.cache-gets", 1)

	// check each chunk
	for _, chunk := range wc.chunks {
		repers, err := chunk.Get(name)
		if repers == nil {
			continue
		}
		if err != nil {
			wc.log.Error("Failed in Cache Chunk Get: %v", err)
			continue
		}
		slice = append(slice, repers...)
	}

	//current chunk
	wc.chmu.RLock()
	repers, err := wc.curChunk.Get(name)
	wc.chmu.RUnlock()

	if err != nil {
		wc.log.Error("Failed in Cache Chunk Get: %v", err)
	}
	if repers != nil {
		slice = append(slice, repers...)
	}
	return slice, nil
}

func (wc *CacherChunk) Exists(name *repr.StatName) (bool, error) {
	wc.chmu.RLock()
	defer wc.chmu.RUnlock()
	if _, ok := wc.curChunk.ts[name.UniqueId()]; ok {
		return true, nil
	}
	return false, nil
}

func (wc *CacherChunk) CacheList() ([]*repr.StatName, error) {
	wc.chmu.RLock()
	defer wc.chmu.RUnlock()
	if wc == nil || wc.curChunk == nil {
		return nil, nil
	}
	rList := make([]*repr.StatName, 0)
	for _, s := range wc.curChunk.ts {
		if s == nil || s.Name == nil {
			continue
		}
		rList = append(rList, s.Name)
	}
	return rList, nil
}

// GetAsRawRenderItem stats as the nominal output
func (wc *CacherChunk) GetAsRawRenderItem(name *repr.StatName) (rawd *metrics.RawRenderItem, err error) {
	if wc == nil || wc.curChunk == nil {
		return nil, nil
	}
	// check each chunk
	for _, chunk := range wc.chunks {
		mets, err := chunk.GetAsRawRenderItem(name)
		if mets == nil {
			continue
		}

		if err != nil {
			wc.log.Error("Failed in Historical Cache Chunks GetAsRawRenderItem: %v", err)
			continue
		}
		if rawd == nil {
			rawd = mets
		} else {
			rawd.Merge(mets)
		}
	}
	wc.chmu.RLock()
	mets, err := wc.curChunk.GetAsRawRenderItem(name)
	wc.chmu.RUnlock()
	if err != nil {
		wc.log.Error("Failed in Current Cache Chunk GetAsRawRenderItem: %v", err)
	}

	if mets != nil {
		if rawd == nil {
			rawd = mets
		} else {
			rawd.Merge(mets)
		}
	}

	return rawd, nil

}

func (wc *CacherChunk) GetById(metric_id repr.StatId) (name *repr.StatName, slice repr.StatReprSlice, err error) {
	if wc == nil || wc.mu == nil {
		return nil, nil, nil
	}
	stats.StatsdClientSlow.Incr(wc.statsdPrefix+"cacherchunk.read.cache-getbyid", 1)
	// check each chunk
	for _, chunk := range wc.chunks {
		tname, repers, err := chunk.GetById(metric_id)
		if repers == nil {
			continue
		}
		if name == nil && tname != nil {
			name = tname
		}
		if err != nil {
			wc.log.Error("Failed in Cache Chunk GetById: %v", err)
			continue
		}
		slice = append(slice, repers...)
	}

	wc.chmu.RLock()
	//current chunk
	tname, repers, err := wc.curChunk.GetById(metric_id)
	wc.chmu.RUnlock()

	if err != nil {
		wc.log.Error("Failed in Cache Chunk GetById: %v", err)
	}
	if repers != nil {
		slice = append(slice, repers...)
	}
	if name == nil && tname != nil {
		name = tname
	}
	stats.StatsdClientSlow.Incr(wc.statsdPrefix+"read.cache-getsbyid.empty", 1)
	return name, slice, err
}

func (wc *CacherChunk) GetSeries(name *repr.StatName) (nm *repr.StatName, ts []series.TimeSeries, err error) {
	if wc == nil || wc.mu == nil {
		return nil, nil, nil
	}
	stats.StatsdClientSlow.Incr(wc.statsdPrefix+"read.cache-series-gets", 1)

	// check each chunk
	for _, chunk := range wc.chunks {
		tname, repers, err := chunk.GetSeries(name)
		if repers == nil {
			continue
		}
		if nm == nil && tname != nil {
			nm = tname
		}
		if err != nil {
			wc.log.Error("Failed in Cache Chunk GetSeries: %v", err)
			continue
		}
		ts = append(ts, repers)
	}

	//current chunk
	wc.chmu.RLock()
	tname, repers, err := wc.curChunk.GetSeries(name)
	wc.chmu.RUnlock()
	if err != nil {
		wc.log.Error("Failed in Cache Chunk GetSeries: %v", err)
	}
	if repers != nil {
		ts = append(ts, repers)
	}
	if nm == nil && tname != nil {
		nm = tname
	}

	return nm, ts, err
}

func (wc *CacherChunk) GetSeriesById(metricId repr.StatId) (nm *repr.StatName, ts []series.TimeSeries, err error) {
	stats.StatsdClientSlow.Incr(wc.statsdPrefix+"read.cache-series-by-idgets", 1)

	// check each chunk
	for _, chunk := range wc.chunks {
		tname, repers, err := chunk.GetSeriesById(metricId)
		if repers == nil {
			continue
		}
		if nm == nil && tname != nil {
			nm = tname
		}
		if err != nil {
			wc.log.Error("Failed in Cache Chunk GetSeries: %v", err)
			continue
		}
		ts = append(ts, repers)
	}

	//current chunk
	wc.chmu.RLock()
	tname, repers, err := wc.curChunk.GetSeriesById(metricId)
	wc.chmu.RUnlock()

	if err != nil {
		wc.log.Error("Failed in Cache Chunk GetSeries: %v", err)
	}
	if repers != nil {
		ts = append(ts, repers)
	}
	if nm == nil && tname != nil {
		nm = tname
	}

	return nm, ts, err
}

func (wc *CacherChunk) GetCurrentSeriesById(metric_id repr.StatId) (nm *repr.StatName, ts series.TimeSeries, err error) {
	//current chunk
	wc.chmu.RLock()
	tname, repers, err := wc.curChunk.GetSeriesById(metric_id)
	wc.chmu.RUnlock()

	if err != nil {
		wc.log.Error("Failed in Cache Chunk GetSeries: %v", err)
		return nil, nil, nil
	}
	if nm == nil && tname != nil {
		nm = tname
	}

	return nm, repers, err
}

func (wc *CacherChunk) GetCurrentSeries(name *repr.StatName) (nm *repr.StatName, ts series.TimeSeries, err error) {
	//current chunk
	wc.chmu.RLock()
	tname, repers, err := wc.curChunk.GetSeries(name)
	wc.chmu.RUnlock()

	if err != nil {
		wc.log.Error("Failed in Cache Chunk GetSeries: %v", err)
		return nil, nil, nil
	}
	if nm == nil && tname != nil {
		nm = tname
	}

	return nm, repers, err
}
