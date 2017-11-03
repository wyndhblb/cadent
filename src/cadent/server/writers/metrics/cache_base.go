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
	"base" chunked cache object for log based writers
*/

package metrics

import (
	"cadent/server/broadcast"
	sindexer "cadent/server/schemas/indexer"
	smetrics "cadent/server/schemas/metrics"
	"cadent/server/schemas/repr"
	"cadent/server/stats"
	"cadent/server/utils"
	"fmt"
	"golang.org/x/net/context"
	"gopkg.in/op/go-logging.v1"
	"strings"
	"time"
)

/****************** Metrics Writer *********************/
type CacheBase struct {
	WriterBase

	log *logging.Logger

	loggerChan *broadcast.Listener // log writes
	slicerChan *broadcast.Listener // current slice writes

	cacheTimeWindow int // chunk time before flush
	cacheChunks     int // number of chunks in ram
	cacheLogFlush   int // flush the current log ever this seconds
}

/************************ READERS ****************/

func (lb *CacheBase) GetFromWriteCache(metric *sindexer.MetricFindItem, start uint32, end uint32, resolution uint32) (*smetrics.RawRenderItem, error) {

	// grab data from the write inflight cache
	// need to pick the "proper" cache
	cacheDb := fmt.Sprintf("%s:%d", lb.cacherPrefix, resolution)
	useRes := resolution

	userCache := GetCacherByName(cacheDb)
	if userCache == nil {
		userCache = lb.cacher
	}
	statName := metric.StatName()
	inflight, err := userCache.GetAsRawRenderItem(statName)

	if err != nil {
		return nil, err
	}
	if inflight == nil {
		return nil, nil
	}
	inflight.Metric = metric.Path
	inflight.Id = metric.UniqueId
	inflight.Step = useRes
	inflight.Start = start
	inflight.End = end
	inflight.Tags = metric.Tags
	inflight.MetaTags = metric.MetaTags
	inflight.AggFunc = statName.AggType()
	inflight.InCache = true
	inflight.UsingCache = true

	return inflight, nil
}

// InCache Is the metric in our cache
func (lb *CacheBase) InCache(path string, tags repr.SortingTags) (bool, error) {
	// only need the lowest res cache as that will have the metric or not
	cacheDb := fmt.Sprintf("%s:%d", lb.cacherPrefix, lb.resolutions[0][0])
	userCache := GetCacherByName(cacheDb)
	if userCache == nil {
		userCache = lb.cacher
	}
	statName := &repr.StatName{
		Key:  path,
		Tags: tags,
	}
	return userCache.Exists(statName)
}

// CacheList list of all metrics in our caches (can be a big one)
func (lb *CacheBase) CacheList() ([]*repr.StatName, error) {
	// only need the lowest res cache as that will have the metric or not
	cacheDb := fmt.Sprintf("%s:%d", lb.cacherPrefix, lb.resolutions[0][0])
	userCache := GetCacherByName(cacheDb)
	if userCache == nil {
		userCache = lb.cacher
	}
	if userCache == nil {
		return nil, nil
	}
	return userCache.CacheList()
}

func (lb *CacheBase) CacheRender(ctx context.Context, path string, start int64, end int64, tags repr.SortingTags) (rawd []*smetrics.RawRenderItem, err error) {
	sp, closer := lb.GetSpan("Find", ctx)
	sp.LogKV("driver", "CacheRender", "path", path, "from", start, "to", end)
	defer closer()

	defer stats.StatsdSlowNanoTimeFunc("reader.cachelog.cacherender.get-time-ns", time.Now())

	//figure out the best res
	resolution := lb.GetResolution(start, end)

	start = TruncateTimeTo(start, int(resolution))
	end = TruncateTimeTo(end, int(resolution))

	paths := strings.Split(path, ",")
	var metrics []*sindexer.MetricFindItem

	renderWg := utils.GetWaitGroup()
	defer utils.PutWaitGroup(renderWg)

	for _, pth := range paths {
		mets, err := lb.indexer.Find(ctx, pth, tags)
		if err != nil {
			continue
		}
		metrics = append(metrics, mets...)
	}

	rawd = make([]*smetrics.RawRenderItem, len(metrics), len(metrics))

	// ye old fan out technique
	renderOne := func(metric *sindexer.MetricFindItem, idx int) {
		defer renderWg.Done()
		// resolution is not known so set it to the min allowed
		_ri, err := lb.GetFromWriteCache(metric, uint32(start), uint32(end), 1)

		if err != nil {
			lb.log.Error("Read Error for %s (%s->%s) : %v", path, start, end, err)
			return
		}
		rawd[idx] = _ri
		return
	}

	for idx, metric := range metrics {
		renderWg.Add(1)
		go renderOne(metric, idx)
	}
	renderWg.Wait()
	return rawd, nil
}

func (lb *CacheBase) CachedSeries(path string, start int64, end int64, tags repr.SortingTags) (series *smetrics.TotalTimeSeries, err error) {

	defer stats.StatsdSlowNanoTimeFunc("reader.cachelog.cacheseries.get-time-ns", time.Now())

	paths := strings.Split(path, ",")
	if len(paths) > 1 {
		return series, errMultiTargetsNotAllowed
	}

	metric := &repr.StatName{Key: path}
	metric.MergeMetric2Tags(&tags)
	metric.MergeMetric2Tags(lb.staticTags)

	resolution := lb.GetResolution(start, end)
	cacheDb := fmt.Sprintf("%s:%v", lb.cacherPrefix, resolution)
	userCache := GetCacherByName(cacheDb)
	if userCache == nil {
		userCache = lb.cacher
	}

	name, inflight, err := userCache.GetCurrentSeries(metric)
	if err != nil {
		return nil, err
	}

	if inflight == nil {
		// try the the path as unique ID
		gotsInt := metric.StringToUniqueId(path)
		if gotsInt != 0 {
			name, inflight, err = userCache.GetCurrentSeriesById(gotsInt)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, nil
		}
	}

	return &smetrics.TotalTimeSeries{Name: name, Series: inflight}, nil
}
