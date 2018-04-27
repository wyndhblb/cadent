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
	Ram indexes form the basis for all the other indexes,

	It uses the metric_ram and tag_ram modules to maintain internal states

	local_index_dir is the location for the local levelDB index files.
	These are "wiped out" on each startup

	Note this is not really "ram" but a levelDB backed store as the ram requirements can be very large
	otherwise
*/

package indexer

import (
	"bytes"
	"cadent/server/schemas/indexer"
	"cadent/server/schemas/repr"
	"cadent/server/stats"
	"cadent/server/utils"
	"cadent/server/utils/options"
	"golang.org/x/net/context"
	"gopkg.in/op/go-logging.v1"
	"sync/atomic"
	"time"
)

const RAM_INDEXER_EXPIRE_METRICS_TIME = time.Duration(48 * time.Hour)

/****************** Interfaces *********************/
type RamIndexer struct {
	TracerIndexer

	indexerId string

	numWorkers int
	queueLen   int
	_accept    bool //shtdown notice

	cache *Cacher // simple cache to rate limit and buffer writes

	startstop utils.StartStop

	cullMetricDelta time.Duration
	TagIndex        *TagIndex
	MetricIndex     *MetricIndex

	// since this can form a basis for other indexer
	// we need to know weather or not to use the find functions here or the DB directly
	hasData int32

	log *logging.Logger
}

func NewRamIndexer() *RamIndexer {
	my := new(RamIndexer)
	my.log = logging.MustGetLogger("indexer.ram")
	my.TagIndex = NewTagIndex()
	my.log = logging.MustGetLogger("indexer.ram")
	my.cullMetricDelta = RAM_INDEXER_EXPIRE_METRICS_TIME
	my.hasData = 0
	return my
}

func (r *RamIndexer) ConfigRam(conf *options.Options) (err error) {
	r.MetricIndex, err = NewMetricIndex(conf.String("local_index_dir", ""))
	r.cache, err = getCacherSingleton(r.indexerId)
	if err != nil {
		return err
	}
	return err
}

// Config the indexer
func (r *RamIndexer) Config(conf *options.Options) (err error) {

	if r.MetricIndex == nil {
		err = r.ConfigRam(conf)
		if err != nil {
			return err
		}
	}

	r.indexerId = conf.String("name", "indexer:ram")
	// reg ourselves before try to get conns
	r.log.Noticef("Registering indexer: %s", r.Name())
	err = RegisterIndexer(r.Name(), r)
	if err != nil {
		return err
	}

	r.cache, err = getCacherSingleton(r.indexerId)
	if err != nil {
		return err
	}
	return nil
}

// HasData do we have data to serve
func (r *RamIndexer) HasData() bool {
	return atomic.LoadInt32(&r.hasData) == 1
}

// ShouldWrite
func (r *RamIndexer) ShouldWrite() bool {
	return true
}

// Name of the indexer
func (r *RamIndexer) Name() string { return r.indexerId }

// Stop the ram indexer
func (r *RamIndexer) Stop() {
	r.startstop.Stop(func() {
		r.cache.Stop()
	})
}

// Start up the indexer
func (r *RamIndexer) Start() {
	r.startstop.Start(func() {
		r.log.Notice("starting ram indexer")

		r.cache.Start() //start cacher
		go r.periodicCull()
	})
}

func (r *RamIndexer) periodicCull() {
	tick := time.NewTicker(time.Hour)
	for {
		<-tick.C
		r.log.Notice("Starting ram index cull...")
		items := r.MetricIndex.CullExpired(RAM_INDEXER_EXPIRE_METRICS_TIME)
		r.log.Notice("Culled %d items from the ram index as they were too old", int64(len(items)))
		stats.StatsdClientSlow.Gauge("indexer.ram.cull.items", int64(len(items)))
	}
}

func (r *RamIndexer) Delete(name *repr.StatName) error {
	err := r.MetricIndex.Delete(name.Key)
	if err != nil {
		return err
	}
	return r.TagIndex.DeleteByUid(name.UniqueIdString())
}

// Write write a name into the ram/tag indexes
func (r *RamIndexer) Write(metric repr.StatName) error {

	err := r.MetricIndex.Add(metric.UniqueIdString(), metric.Key)
	if err != nil {
		return err
	}

	// gots
	atomic.StoreInt32(&r.hasData, 1)

	if !metric.Tags.IsEmpty() {
		err = r.TagIndex.Add(metric.UniqueIdString(), metric.Tags)
		if err != nil {
			return err
		}
	}
	if !metric.MetaTags.IsEmpty() {
		err = r.TagIndex.Add(metric.UniqueIdString(), metric.MetaTags)
		if err != nil {
			return err
		}
	}
	return nil
}

// KeyExists is the key there
func (r *RamIndexer) KeyExists(metric string) bool {
	return len(r.MetricIndex.Contains(metric)) > 0
}

// WriteLastSeen add a metric to the index only if the timestamp is w/i our cull period
func (r *RamIndexer) WriteLastSeen(metric repr.StatName, lastseen int) error {
	t := time.Unix(int64(lastseen), 0)
	// skip if expired
	expT := time.Now().Add(-r.cullMetricDelta)
	if t.Before(expT) {
		return nil
	}

	err := r.Write(metric)
	if err != nil {
		return err
	}
	r.MetricIndex.SetLastSeen(metric.UniqueIdString(), t)

	return nil
}

// NeedsWrite if the last write and the
func (r *RamIndexer) NeedsWrite(metric repr.StatName, dur time.Duration) bool {

	t := r.MetricIndex.GetLastSeen(metric.UniqueIdString())
	if t.IsZero() {
		return true
	}

	return t.Before(time.Now().Add(-dur))
}

// GetLastSeen last time the metric was "indexed"
func (r *RamIndexer) GetLastSeen(metric repr.StatName) time.Time {
	return r.MetricIndex.GetLastSeen(metric.UniqueIdString())
}

func (r *RamIndexer) List(hasData bool, page int) (mt indexer.MetricFindItems, err error) {

	cur_ct := 0
	want_start := page * MAX_PER_PAGE
	stop_count := (page + 1) * MAX_PER_PAGE

	r.MetricIndex.nmLock.RLock()
	defer r.MetricIndex.nmLock.RUnlock()

	for k, uid := range r.MetricIndex.nameIndex {
		cur_ct++
		if cur_ct < want_start {
			continue
		}
		if cur_ct > stop_count {
			break
		}
		on_path := k

		ms := new(indexer.MetricFindItem)
		ms.Text = on_path
		ms.Id = on_path
		ms.Path = on_path
		ms.UniqueId = uid

		ms.Expandable = 0
		ms.Leaf = 1
		ms.AllowChildren = 0

		ms.Tags, err = r.TagIndex.FindTagsByUid(uid)
		if err != nil {
			r.log.Errorf(err.Error())
		}

		mt = append(mt, ms)
	}

	return mt, nil
}

// Find a metric by path and tags
// TODO tag search Intersection
func (r *RamIndexer) Find(ctx context.Context, metric string, tags repr.SortingTags) (indexer.MetricFindItems, error) {
	return r.MetricIndex.Find(ctx, metric, false)
}

// FindInCache since this is the ram cache, alias to Find
func (r *RamIndexer) FindInCache(ctx context.Context, metric string, tags repr.SortingTags) (indexer.MetricFindItems, error) {
	sp, closer := r.GetSpan("FindRamCache", ctx)
	sp.LogKV("driver", "RamIndexer", "metric", metric, "tags", tags)
	defer closer()

	return r.Find(ctx, metric, tags)
}

// Expand given a metric path, return all first level children
func (r *RamIndexer) Expand(metric string) (me indexer.MetricExpandItem, err error) {
	iter, reged, _, err := r.MetricIndex.FindLowestPrefix(metric)

	if iter == nil || err != nil {
		return
	}

	// we simply troll the {len}:{prefix} world
	for iter.Next() {
		val := bytes.Split(iter.Key(), repr.COLON_SEPARATOR_BYTE)
		if len(val) <= 1 {
			continue
		}
		if reged != nil {
			if reged.Match(val[1]) {
				me.Results = append(me.Results, string(val[1]))

			}
		} else {
			me.Results = append(me.Results, string(val[1]))

		}
	}
	iter.Release()
	err = iter.Error()
	return
}

// GetTagsByUid find all tags associated with this UID
func (r *RamIndexer) GetTagsByUid(uniqueId string) (tags repr.SortingTags, metatags repr.SortingTags, err error) {
	tags, err = r.TagIndex.FindTagsByUid(uniqueId)
	return
}

// GetTagsByName find all tags with a given name value (value can be a regex)
func (r *RamIndexer) GetTagsByName(name string, page int) (tags indexer.MetricTagItems, err error) {
	return r.TagIndex.GetTagsByName(name)
}

// GetTagsByNameValue get all the tags with a given name and
func (r *RamIndexer) GetTagsByNameValue(name string, value string, page int) (tags indexer.MetricTagItems, err error) {
	return r.TagIndex.GetTagsByNameValue(name, value)
}

// GetUidsByTags get all the uids with the
// TODO key search Intersection
func (r *RamIndexer) GetUidsByTags(key string, tags repr.SortingTags, page int) (uids []string, err error) {
	return r.TagIndex.FindUidByTags(tags)
}

// culling, given that stats can disappear, we use the "last hit" list in the metric_ram item to find them
// and purge them if they have not been seen/added in the configured time
func (r *RamIndexer) cullMetrics(cTime time.Duration) {

	toPurge := []string{}
	if cTime > 0 {
		cTime = -cTime
	}
	tPurge := time.Now().Add(cTime)

	r.MetricIndex.lsLock.RLock()
	for uid, delta := range r.MetricIndex.lastHit {
		if delta.Before(tPurge) {
			toPurge = append(toPurge, uid)
		}
	}
	r.MetricIndex.lsLock.RUnlock()

	for _, uid := range toPurge {
		r.MetricIndex.DeleteByUid(uid)
		r.TagIndex.DeleteByUid(uid)
	}
}
