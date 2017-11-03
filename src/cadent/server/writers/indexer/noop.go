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
  simple "no op" indexer that does nothing
*/

package indexer

import (
	"cadent/server/schemas/indexer"
	"cadent/server/schemas/repr"
	"cadent/server/utils/options"
	"golang.org/x/net/context"
)

type NoopIndexer struct {
	TracerIndexer
}

func NewNoopIndexer() *NoopIndexer {
	return new(NoopIndexer)
}

func (my *NoopIndexer) Config(conf *options.Options) error {
	return nil
}
func (my *NoopIndexer) Name() string                     { return "noop-indexer" }
func (my *NoopIndexer) Stop()                            {}
func (my *NoopIndexer) Start()                           {}
func (my *NoopIndexer) Delete(name *repr.StatName) error { return nil }

func (my *NoopIndexer) Write(metric repr.StatName) error {
	return nil
}

// ShouldWrite always true, as a "write" for noop is nothing
func (my *NoopIndexer) ShouldWrite() bool {
	return true
}

func (my *NoopIndexer) List(has_data bool, page int) (indexer.MetricFindItems, error) {
	return indexer.MetricFindItems{}, ErrWillNotBeimplemented
}
func (my *NoopIndexer) Find(ctx context.Context, metric string, tags repr.SortingTags) (indexer.MetricFindItems, error) {
	return indexer.MetricFindItems{}, ErrWillNotBeimplemented
}
func (my *NoopIndexer) FindInCache(ctx context.Context, metric string, tags repr.SortingTags) (indexer.MetricFindItems, error) {
	return indexer.MetricFindItems{}, ErrWillNotBeimplemented
}
func (my *NoopIndexer) Expand(metric string) (indexer.MetricExpandItem, error) {
	return indexer.MetricExpandItem{}, ErrWillNotBeimplemented
}

func (my *NoopIndexer) GetTagsByUid(unique_id string) (tags repr.SortingTags, metatags repr.SortingTags, err error) {
	return tags, metatags, ErrWillNotBeimplemented
}

func (my *NoopIndexer) GetTagsByName(name string, page int) (tags indexer.MetricTagItems, err error) {
	return tags, ErrWillNotBeimplemented
}

func (my *NoopIndexer) GetTagsByNameValue(name string, value string, page int) (tags indexer.MetricTagItems, err error) {
	return tags, ErrWillNotBeimplemented
}

func (my *NoopIndexer) GetUidsByTags(key string, tags repr.SortingTags, page int) (uids []string, err error) {
	return uids, ErrWillNotBeimplemented
}
