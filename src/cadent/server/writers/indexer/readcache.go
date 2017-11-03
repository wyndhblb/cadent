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
	cache data from reads in a LRU cache w/ a TTL

*/

package indexer

import (
	"cadent/server/schemas/indexer"
	"cadent/server/schemas/repr"
	"cadent/server/stats"
	"sync"
	"time"
)

const (
	INDEX_CACHE_MAX_TIME_IN_CACHE = time.Minute * time.Duration(1) // if an item is not accesed in 60 sec purge it
)

// basic cached item treat it kinda like a round-robin array
type IndexCacheItem struct {
	Added int64
	Data  *indexer.MetricFindItems
}

func NewIndexCacheItem(data *indexer.MetricFindItems) *IndexCacheItem {

	return &IndexCacheItem{
		Data:  data,
		Added: time.Now().Unix(),
	}
}

// LRU read cache
type IndexReadCache struct {
	mu       *sync.RWMutex
	cache    map[string]*IndexCacheItem
	MaxItems int
}

func NewIndexCache(max_items int) *IndexReadCache {
	rc := &IndexReadCache{
		MaxItems: max_items,
		cache:    make(map[string]*IndexCacheItem),
		mu:       new(sync.RWMutex),
	}
	go rc.periodPurge()
	return rc
}

func (rc *IndexReadCache) periodPurge() {
	tick := time.NewTicker(time.Minute * time.Duration(60))
	for {
		<-tick.C
		max_back := time.Now().Unix() - int64(INDEX_CACHE_MAX_TIME_IN_CACHE)
		rc.mu.Lock()
		for k, v := range rc.cache {
			if v.Added < max_back {
				delete(rc.cache, k)
			}
		}
		rc.mu.Unlock()
	}
}

func (rc *IndexReadCache) Key(metric string, tags repr.SortingTags) string {
	return metric + tags.String()
}

// add a series to the cache .. this should only be called by a reader api
// or some 'pre-seed' mechanism
func (rc *IndexReadCache) Add(metric string, tags repr.SortingTags, items *indexer.MetricFindItems) bool {

	m_key := rc.Key(metric, tags)
	rc.mu.RLock()
	_, ok := rc.cache[m_key]
	rc.mu.RUnlock()
	if !ok {

		// just drop the first in the iter
		rc_item := NewIndexCacheItem(items)
		rc.mu.Lock()
		if len(rc.cache) > rc.MaxItems {
			for k := range rc.cache {
				delete(rc.cache, k)
				stats.StatsdClientSlow.Incr("indexer.readcache.evictions", 1)
				break
			}
		}
		stats.StatsdClientSlow.Incr("indexer.readcache.adds", 1)
		rc.cache[m_key] = rc_item
		rc.mu.Unlock()
		return true
	}
	return false // already activated
}

func (rc *IndexReadCache) Get(metric string, tags repr.SortingTags) *indexer.MetricFindItems {
	key := rc.Key(metric, tags)

	rc.mu.RLock()
	gots, ok := rc.cache[key]
	rc.mu.RUnlock()
	if !ok {
		return nil
	}
	maxBack := time.Now().Unix() - int64(INDEX_CACHE_MAX_TIME_IN_CACHE.Seconds())
	if gots.Added < maxBack {
		rc.mu.Lock()
		delete(rc.cache, key)
		rc.mu.Unlock()
		stats.StatsdClientSlow.Incr("indexer.readcache.evictions", 1)
		return nil
	}
	stats.StatsdClientSlow.Incr("indexer.readcache.hits", 1)
	return gots.Data
}

func (rc *IndexReadCache) Len() int {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return len(rc.cache)
}

func (rc *IndexReadCache) statsTick() {
	ticker := time.NewTicker(2 * time.Second)
	for {
		select {
		case <-ticker.C:
			stats.StatsdClientSlow.Gauge("indexer.readcache.items", int64(rc.Len()))
		}
	}
}
