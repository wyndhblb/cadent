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

	triggered "rollups"

	Since Blob series for "long flush windows" (like 60+seconds) can take a _very_ long time
	to flush if the blob size is in the kb range, basically they will take for ever to be persisted
	and we don't want to persist "just a few points" in the storage, so for the "smallest" flush window
	any time we do an overflow write we trigger a "rollup" event which will

	This also means we only need to store in the cache just the smallest time which will save lots of
	ram if you have lots of flush windows.

	The process is as follows

	1. the min Resolution writer triggers a "rollup" once it writes data
	2. Based on the other resolutions grab all the data from the min resolution
	   that fits in the resolution window (start - resolution, end + resolution)
	3. Resample that list to the large resolution
	4. if there is data in the range already, just update the row, otherwise insert a new one

	find the current UniqueID already persisted for the longer flush windows
	If found: "MERGE" the new data into it (if the blob size is still under the window)
		"START" a new blob if too big
	If not found:
		"Start" a new blob


	There are 2 modes here, depending on the "volume" of the incoming.

	the first mode: "blast": is just write until there is nothing left in the dispatch queue
	this can be problematic if there are a lot of things to rollup and things are using the
	cache-log mechanism.  You can start to overlap flushes, which basically means death (as rollups
	can take a a lot of horse power and ram to compute for that many).
	This mode takes the series that was just written and appends it to the current items in the DB.

	The second mode: "queue": is a slower unique queue based mechanism, where by on a writer insert, the
	uid, stime, etime (of the recent chunk that was just written) gets added to a list, if the uid
	is still in the list (from a previous flush) it does not get re-added as it will be computed in a bit.
	In this way rollups are basically a continuous process eating away at that list to avoid the above issue
	It should also be less taxing on the system.  In this mode, rather then using the TS that was just written
	it fetches the latest set TS that are >= etime and appends those to the new items.

*/

package metrics

import (
	"cadent/server/schemas/metrics"
	"cadent/server/utils/shutdown"
	"github.com/cenkalti/backoff"
	_ "github.com/go-sql-driver/mysql"
	logging "gopkg.in/op/go-logging.v1"

	"cadent/server/dispatch"
	"cadent/server/schemas/repr"
	"cadent/server/series"
	"cadent/server/stats"
	"cadent/server/utils"
	"errors"
	"math"
	"sync"
	"time"
)

const (
	ROLLUP_QUEUE_LENGTH = 16384
	ROLLUP_NUM_WORKERS  = 8
	ROLLUP_NUM_RETRIES  = 2

	ROLLUP_MODE              = "blast"
	ROLLUP_NUM_QUEUE_WORKERS = 4
)

const (
	blastMode uint8 = iota
	queueMode
)

/************************************************************************/
/**********  Standard Worker Dispatcher JOB   ***************************/
/************************************************************************/
// rollup job queue workers
type rollupJob struct {
	Rollup *RollupMetric
	Ts     *metrics.TotalTimeSeries
	Retry  int
}

func (j *rollupJob) IncRetry() int {
	j.Retry++
	return j.Retry
}
func (j *rollupJob) OnRetry() int {
	return j.Retry
}

func (j *rollupJob) DoWork() error {
	err := j.Rollup.DoRollup(j.Ts)
	return err
}

/************************************************************************/
/**********  "queue" for queue mode   ***************************/
/************************************************************************/
type rollupQueueItem struct {
	uid        string
	stime      int64
	etime      int64
	statName   *repr.StatName
	seriesType string
	highRes    bool
}

type rollupQueue struct {
	sync.RWMutex
	inQueue map[string]bool
	queue   []*rollupQueueItem
}

func (q *rollupQueue) push(ts *metrics.TotalTimeSeries) {
	q.Lock()
	defer q.Unlock()

	uidst := ts.Name.UniqueIdString()
	// already there skip
	if _, ok := q.inQueue[uidst]; ok {
		return
	}
	q.queue = append(q.queue, &rollupQueueItem{
		uid:        uidst,
		stime:      ts.Series.StartTime(),
		etime:      ts.Series.LastTime(),
		statName:   ts.Name,
		seriesType: ts.Series.Name(),
		highRes:    ts.Series.HighResolution(),
	})
	q.inQueue[uidst] = true

}

func (q *rollupQueue) pop() *rollupQueueItem {
	q.Lock()
	defer q.Unlock()

	if len(q.queue) == 0 {
		return nil
	}
	var current *rollupQueueItem
	current, q.queue = q.queue[0], q.queue[1:]
	delete(q.inQueue, current.uid)

	return current
}

/************************************************************************/
/**********  Rollup Driver ***************************/
/************************************************************************/
type RollupMetric struct {
	writer        DBMetrics
	resolutions   [][]int
	minResolution int

	runMode uint8

	dispatcher          *dispatch.DispatchQueue
	queue               *rollupQueue
	consumeQueue        chan *rollupQueueItem
	consumeQueueStop    chan bool
	consumeQueueWorkers int

	seriesEncoding string
	blobMaxBytes   int

	startstop utils.StartStop

	numWorkers  int
	queueLength int
	numRetries  int

	log *logging.Logger
}

func NewRollupMetric(writer DBMetrics, maxBytes int) *RollupMetric {
	rl := new(RollupMetric)
	rl.writer = writer
	rl.blobMaxBytes = maxBytes
	rl.consumeQueueWorkers = ROLLUP_NUM_QUEUE_WORKERS
	// in it queue bits
	rl.queue = new(rollupQueue)
	rl.queue.inQueue = make(map[string]bool)
	rl.queue.queue = make([]*rollupQueueItem, 0)

	rl.consumeQueue = make(chan *rollupQueueItem, 1)
	rl.consumeQueueStop = make(chan bool, 1)
	rl.log = logging.MustGetLogger("writers.rollup")
	rl.numWorkers = ROLLUP_NUM_WORKERS
	rl.queueLength = ROLLUP_QUEUE_LENGTH
	rl.numRetries = ROLLUP_NUM_RETRIES
	return rl
}

// SetRunMode set
func (rl *RollupMetric) SetRunMode(md string) {
	switch md {
	case "queue", "list", "slow":
		rl.runMode = queueMode
	default:
		rl.runMode = blastMode
	}
}

// the resolution/ttl pairs we wish to rollup .. you should not include
// the resolution that is the foundation for the rollup
// if the resolutions are [ [ 5, 100], [ 60, 720 ]] just include the [[60, 720]]
func (rl *RollupMetric) SetResolutions(res [][]int) {
	rl.resolutions = res
}

func (rl *RollupMetric) SetMinResolution(res int) {
	rl.minResolution = res
}

// pop things off the queue list forever
func (rl *RollupMetric) popQueue() {
	for {
		select {
		case <-rl.consumeQueueStop:
			close(rl.consumeQueue)
			close(rl.consumeQueueStop)
			return
		default:
			item := rl.queue.pop()

			// nothing there, take a nap
			if item == nil {
				time.Sleep(1 * time.Second)
				continue
			}
			rl.log.Debugf("Poped %s (%s) from queue, rolling up ...", item.statName.Key, item.statName.UniqueIdString())
			rl.consumeQueue <- item
		}
	}
}

// chomp the queue for queue mode
func (rl *RollupMetric) queueLoop() {
	// a "nutty" large endtime
	etime := uint32(math.MaxUint32)
	res := uint32(rl.minResolution)

	var rErr = errors.New("rolluperr")

	retryNotify := func(err error, after time.Duration) {
		stats.StatsdClientSlow.Incr("writer.rollup.queue.retry", 1)
		rl.log.Error("Insert Fail Retry :: %v : after %v", err, after)
	}

	doRollup := func(inItem *rollupQueueItem) error {
		item := inItem
		// grab the items >= etime and make a big old RawRenderTime
		sTime := uint32(time.Unix(0, item.stime).Unix())
		dblist, err := rl.writer.GetRangeFromDB(item.statName, sTime-res, etime, res)
		rl.log.Debugf("Found %d to rollup for %s (%s)", len(dblist), item.statName.Key, item.statName.UniqueIdString())

		if err != nil {
			stats.StatsdClientSlow.Incr("writer.rollup.queue.error", 1)
			rl.log.Errorf("Error in GetRangeFromDB: %v", err)
			return rErr
		}

		// odd "bad" case where this was triggered but no data
		if len(dblist) == 0 {
			stats.StatsdClientSlow.Incr("writer.rollup.queue.error", 1)
			rl.log.Noticef("Error in GetRangeFromDB No data for: %s start: %d, res: %d", item.uid, sTime, res)
			return nil
		}

		//pull things from a pool (they'll get put back in the actuall merge step)
		oldData, err := dblist.ToRawRenderItemBoundsPool(item.stime, int64(math.MaxInt64))
		if err != nil {
			rl.log.Errorf("Rollup ToRenderItem failure: %v", err)
			return rErr
		}
		stats.StatsdClientSlow.Incr("writer.rollup.queue.comsume", 1)
		oldData.Id = item.statName.UniqueIdString()
		oldData.Step = res
		oldData.Metric = item.statName.Key
		rl.rollup(oldData, item.statName, item.seriesType, item.highRes)
		return nil
	}
	for {
		select {
		case item, more := <-rl.consumeQueue:
			if !more {
				return
			}

			backoffFunc := func() error {
				return doRollup(item)
			}

			backoff.RetryNotify(backoffFunc, backoff.NewExponentialBackOff(), retryNotify)
		}
	}
}

func (rl *RollupMetric) Start() {
	rl.startstop.Start(func() {
		rl.log.Notice(
			"Starting rollup engine writer at %d bytes per series (workers: %d, queue len: %d, retries: %d mode: %v)",
			rl.blobMaxBytes,
			rl.numWorkers,
			rl.queueLength,
			rl.numRetries,
			rl.runMode,
		)
		rl.dispatcher = dispatch.NewDispatchQueue(
			rl.numWorkers,
			rl.queueLength,
			rl.numRetries,
		)
		rl.dispatcher.Start()

		go rl.popQueue()
		for i := 0; i < rl.consumeQueueWorkers; i++ {
			go rl.queueLoop()
		}

	})
}

func (rl *RollupMetric) Stop() {
	rl.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()
		rl.log.Warning("Starting Shutdown of rollup engine")

		if rl.dispatcher != nil {
			rl.dispatcher.Stop()
		}
		rl.consumeQueueStop <- true
	})
}

func (rl *RollupMetric) Add(ts *metrics.TotalTimeSeries) {
	stats.StatsdClientSlow.Incr("writer.rollup.queue.add", 1)
	if ts.Series.Count() == 0 {
		rl.log.Warning("Some how got a rollup trigger but no data to rollup: %d: %s", ts.Name.UniqueId(), ts.Name.Key)
		return
	}
	use := *ts
	switch rl.runMode {
	case queueMode:
		rl.queue.push(&use)
	default:
		rl.dispatcher.Add(&rollupJob{Rollup: rl, Ts: &use})
	}
}

/*

 DoRollup this function is the the "blast" mode:

 for each resolution we have find the latest item in the DB

 There are 4 main conditions we need to satisfy

  Old: S------------------E
  NEW:                      S--------------E
  1. NewData.Start > OldData.End --> merge and write (this should be 99% of cases, where OldE == NewS +/- resolution step)

  OLD: S------------------E
  NEW:           S-----------------E
  2. NewData.Start >= OldData.Start && NewData.End >= OldData.End -> Merge and write

  OLD:          S-----------------E
  NEW: S---------------E
  3. NewData.Start < OldData.Start && NewData.End < OldData.End && NewData.End > OldData.Start -> we have a "backwards" in time merge which

  OLD:                      S-----------E
  NEW: S---------------E
  4. NewData.End < OldData.Start -> Merge

  The flow:

  We know New.Start + New.End but not what the Old start and ends are.

  case #1 and #2 should be most of the cases, unless things are being back filled

  To start we need to get the Latest from the DB to check on conditions 1 -> 2

 	If there is no latest, then we know we are brand new and should just resample and write it out

  	Otherwise, merge the 2 streams, make a new series that are split into the proper chunk sizes.

  	If the resulting series is less then the chunk size, we simply "update" the DB row w/ the new data and end time

  	If not, we update the old row w/ the first chunk, and add new rows for each of the other chunks

  Fore cases 3 + 4 we need to ask for any data from the DB that spans the New Start and End

  OLD: S---------E S---------E ...... S---------E
  NEW     S-------------E

  The merge the series again into one

       S---------------------E

   And then update those N rows w/ the new starts/ends and data

*/
func (rl *RollupMetric) DoRollup(tseries *metrics.TotalTimeSeries) (err error) {

	// want to recover nicely, as we don't want this complex thing nuking the entire site
	defer func() {
		if r := recover(); r != nil {
			if terr, ok := r.(error); ok {
				err = terr
				return
			}
		}
	}()

	stats.StatsdClientSlow.Incr("writer.rollup.queue.comsume", 1)

	// make the series into our resampler object
	newData, err := metrics.NewRawRenderItemFromSeries(tseries)
	if err != nil {
		rl.log.Errorf("Rollup Failure: %v", err)
		return err
	}
	return rl.rollup(newData, tseries.Name, tseries.Series.Name(), tseries.Series.HighResolution())
}

func (rl *RollupMetric) writeOne(
	statName *repr.StatName,
	rawd *metrics.RawRenderItem,
	oldData metrics.DBSeriesList,
	resolution int,
	ttl int,
	seriesType string,
	highres bool,
) error {
	defer stats.StatsdSlowNanoTimeFunc("writer.rollup.write-time-ns", time.Now())

	uid := statName.UniqueIdString()
	// make the new series
	nOpts := series.NewDefaultOptions()
	nOpts.HighTimeResolution = highres
	nseries, err := series.NewTimeSeries(seriesType, int64(rawd.RealStart), nOpts)
	if err != nil {
		return err
	}

	// we may need to add multiple chunks based on the chunk size we have
	mTseries := make([]series.TimeSeries, 0)
	//rl.log.Debugf("Res: %d, WRITE: %d %d %s Data: %d", resolution, rawd.RealStart, rawd.RealEnd, metric, len(rawd.Data))
	for _, point := range rawd.Data {
		// skip it if not a real point
		if point.IsNull() {
			continue
		}
		// raw data pts time are in seconds
		useT := time.Unix(int64(point.Time), 0).UnixNano()
		err := nseries.AddPoint(useT, point.Min, point.Max, point.Last, point.Sum, point.Count)
		if err != nil {
			rl.log.Errorf("Res: %d: Add point failed: %v", resolution, err)
			continue
		}
		// need a new series
		if nseries.Len() > rl.blobMaxBytes {
			mTseries = append(mTseries, nseries)
			nseries, err = series.NewTimeSeries(seriesType, useT, nOpts)
			if err != nil {
				return err
			}
		}
	}
	if nseries.Count() > 0 {
		mTseries = append(mTseries, nseries)
	}
	newName := statName
	newName.Ttl = uint32(ttl)
	newName.Resolution = uint32(resolution)

	// We walk through the old data points and see if we need to "update" the rows
	// or add new ones
	// we effectively update the series until we're out of the old ones, and then add new ones for the
	// remaining series
	lOld := len(oldData)
	rl.log.Debugf("Res: %d: Series %s to update: %d (old series count: %d)", resolution, uid, len(mTseries), lOld)
	for idx, ts := range mTseries {
		if lOld > idx {
			rl.log.Debugf(
				"Res %d: Update Series: %s Old: %d -> %d New: %d -> %d (orig: %d -> %d points)",
				resolution,
				oldData[idx].Uid,
				oldData[idx].Start,
				oldData[idx].End,
				ts.StartTime(),
				ts.LastTime(),
				len(rawd.Data),
				ts.Count(),
			)
			oldData[idx].TTL = newName.Ttl // needed for some DBs
			err = rl.writer.UpdateDBSeries(oldData[idx], ts)
			if err != nil {
				rl.log.Errorf("rollup update err: %v", err)
				stats.StatsdClientSlow.Incr("writer.rollup.db.update.error", 1)
				continue
			}
			stats.StatsdClientSlow.Incr("writer.rollup.db.update.ok", 1)
		} else {
			rl.log.Debugf(
				"Res: %d: Insert New Series: %s (%s) %d->%d (%d points)",
				resolution,
				newName.Key,
				newName.UniqueIdString(),
				ts.StartTime(),
				ts.LastTime(),
				ts.Count(),
			)
			_, err = rl.writer.InsertDBSeries(newName, ts, uint32(resolution))
			if err != nil {
				rl.log.Errorf("Insert error: %s", err)
				stats.StatsdClientSlow.Incr("writer.rollup.db.insert.error", 1)
				continue
			}
			stats.StatsdClientSlow.Incr("writer.rollup.db.insert.ok", 1)
		}
	}

	return nil
}

// the actuall rollup process
func (rl *RollupMetric) rollup(inNewData *metrics.RawRenderItem, statName *repr.StatName, seriesType string, highres bool) error {
	defer stats.StatsdSlowNanoTimeFunc("writer.rollup.process-time-ns", time.Now())

	newData := inNewData

	metric := statName.Key
	uid := statName.UniqueIdString()

	rl.log.Debugf("Rollup Triggered for %s (%s) in %s", metric, uid, rl.writer.Driver())

	// now march through the higher resolutions .. the resolution objects
	// is [ resolution, ttl ]
	for _, res := range rl.resolutions {
		step := res[0]
		ttl := res[1]
		tNewData := newData.Copy()
		if len(newData.Data) == 0 {
			rl.log.Warning(
				"Rollup warning: No data points to rollup: start: %d, end: %d, points: %d for %d (%s) @ res: %d",
				newData.RealStart, newData.RealEnd, len(newData.Data),
				statName.UniqueId(),
				statName.Key,
				res,
			)
			continue
		}

		// if the start|End time is 0 then there is trouble w/ the series itself
		// and attempting a rollup is next to suicide
		if newData.RealStart == 0 || newData.RealEnd == 0 {
			rl.log.Errorf(
				"Rollup failure: Incomming Series Start|End time is 0 (start: %d, end: %d, points: %d), the series is corrupt cannot rollup: %d (%s)",
				newData.RealStart, newData.RealEnd, len(newData.Data),
				statName.UniqueId(),
				statName.Key,
			)
			stats.StatsdClientSlow.Incr("writer.rollup.error", 1)
			continue
		}

		// resample the current blob into the step we want
		tNewData.Resample(uint32(step))

		err := rl.writeOne(statName, tNewData, nil, step, ttl, seriesType, highres)
		if err != nil {
			rl.log.Errorf("Rollup failure: %v", err)
			stats.StatsdClientSlow.Incr("writer.rollup.error", 1)
			continue
		}
		stats.StatsdClientSlow.Incr("writer.rollup.ok", 1)
	}

	// put back into pool
	newData.PutDataIntoPool()
	newData = nil
	return nil
}
