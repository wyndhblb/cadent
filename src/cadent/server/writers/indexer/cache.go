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
	The "cacher"

	Designed to behave like "carbon cache" which allows a few things to happen

	1) ability to "buffer/ratelimit" writes so that we don't overwhelm and writing backend
	2) Query things that are not yet written
	3) allow to reject incoming should things get too far behind (the only real recourse for stats influx overload)

	settings

	[acc.agg.writer.metrics]
	driver="blaa"
	dsn="blaa"
	[acc.agg.writer.metrics.options]
		...
		cache_metric_size=102400  # number of metric strings to keep
		cache_points_size=1024 # number of points per metric to cache above to keep before we drop (this * cache_metric_size * 32 * 128 bytes == your better have that ram)
		...
*/

package indexer

import (
	"cadent/server/stats"
	"fmt"

	"cadent/server/schemas/repr"
	"cadent/server/utils"
	"cadent/server/utils/shutdown"
	logging "gopkg.in/op/go-logging.v1"
	"sync"
	"time"
)

const (
	CACHER_METRICS_KEYS = 1024 * 1024 * 10
)

// the singleton
var _CACHER_SINGLETON map[string]*Cacher
var _cacher_mutex sync.Mutex

func getCacherSingleton(nm string) (*Cacher, error) {
	_cacher_mutex.Lock()
	defer _cacher_mutex.Unlock()

	if val, ok := _CACHER_SINGLETON[nm]; ok {
		return val, nil
	}

	cacher := NewCacher()
	_CACHER_SINGLETON[nm] = cacher
	return cacher, nil
}

// special onload init
func init() {
	_CACHER_SINGLETON = make(map[string]*Cacher)
}

// The "cache" item for points
type Cacher struct {
	mu                  sync.Mutex
	maxKeys             int // max num of keys to keep before we have to drop
	maxPoints           int // max num of points per key to keep before we have to drop
	curSize             int64
	numCurKeys          int
	lowFruitRate        float64   // % of the time we reverse the max sortings to persist low volume stats
	shutdown            chan bool // when received stop allowing adds and updates
	_accept             bool      // flag to stop
	log                 *logging.Logger
	AlreadyWrittenCache map[repr.StatId]bool
	Cache               map[repr.StatId]repr.StatName
	starstop            utils.StartStop
}

func NewCacher() *Cacher {
	wc := new(Cacher)
	wc.maxKeys = CACHER_METRICS_KEYS
	wc.curSize = 0
	wc.log = logging.MustGetLogger("cacher.indexer")
	wc.AlreadyWrittenCache = make(map[repr.StatId]bool)
	wc.Cache = make(map[repr.StatId]repr.StatName)
	wc.shutdown = make(chan bool, 2)
	wc._accept = true
	return wc
}

func (wc *Cacher) Start() {
	wc.starstop.Start(func() {
		go wc.statsTick()
	})
}

func (wc *Cacher) Stop() {
	wc.starstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown() //release me as i'm done
		wc.shutdown <- true
	})
}

func (wc *Cacher) statsTick() {
	ticker := time.NewTicker(2 * time.Second)
	for {
		select {
		case <-wc.shutdown:

			wc._accept = false
			ticker.Stop()
			wc.log.Warning("Index Cache shutdown .. stopping accepts")

			return

		case <-ticker.C:
			stats.StatsdClientSlow.Gauge("cacher.indexer.bytes", wc.curSize)
			stats.StatsdClientSlow.Gauge("cacher.indexer.keys", int64(len(wc.Cache)))
			stats.StatsdClientSlow.Gauge("cacher.indexer.indexed", int64(len(wc.AlreadyWrittenCache)))
			wc.log.Debug("Cacher Indexer: Keys: %d :: Bytes:: %d", len(wc.Cache), wc.curSize)
		}
	}
}

func (wc *Cacher) Add(metric repr.StatName) error {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	//wc.log.Critical("STAT: %s, %v", metric, stat)

	if !wc._accept {
		//wc.log.Error("Shutting down, will not add any more items to the queue")
		return nil
	}

	if len(wc.Cache) > wc.maxKeys {
		wc.log.Error("Indexer Key Cache is too large .. over %d keys, have to drop this one", wc.maxKeys)
		stats.StatsdClientSlow.Incr("cacher.indexer.overflow", 1)
		return fmt.Errorf("Cacher: too many keys, dropping metric")
	}

	/** ye old debuggin'
	if strings.Contains(metric, "flushesposts") {
		wc.log.Critical("ADDING: %s Time: %d, Val: %f", metric, time, value)
	}
	*/
	uid := metric.UniqueIdAllTags()
	if _, ok := wc.AlreadyWrittenCache[uid]; ok {
		stats.StatsdClient.Incr("cacher.indexer.already-written", 1)
		return nil
	}

	if _, ok := wc.Cache[uid]; ok {
		stats.StatsdClient.Incr("cacher.indexer.already-added", 1)
	} else {
		wc.Cache[uid] = metric
		wc.curSize += int64(metric.ByteSize())
		stats.StatsdClient.Incr("cacher.indexer.add", 1)
	}

	return nil
}

func (wc *Cacher) Get(metric repr.StatName) repr.StatName {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	gots, ok := wc.Cache[metric.UniqueIdAllTags()]
	if !ok {
		return repr.StatName{}
	}
	return gots
}

func (wc *Cacher) GetNextMetric() repr.StatName {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	for k, g := range wc.Cache {
		wc.AlreadyWrittenCache[k] = true
		wc.curSize -= int64(g.ByteSize())
		delete(wc.Cache, k)
		return g
	}
	return repr.StatName{}
}

func (wc *Cacher) Pop() repr.StatName {
	return wc.GetNextMetric()
}

// add a metrics/point list back on the queue as it either "failed" or was ratelimited
func (wc *Cacher) AddBack(metric repr.StatName) {
	wc.mu.Lock()
	delete(wc.AlreadyWrittenCache, metric.UniqueIdAllTags())
	wc.mu.Unlock()
	wc.Add(metric)
}
