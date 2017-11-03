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
  Indexer Reader/Writer

  just Read/Write the StatName object

  StatNames have 4 writable indexes: key, uniqueId, tags, metatags

  The Unique ID is basically a hash of the key:sortedByName(tags)
*/

package indexer

import (
	"cadent/server/schemas/indexer"
	"cadent/server/schemas/repr"
	"cadent/server/utils/options"
	"github.com/opentracing/opentracing-go"
	"golang.org/x/net/context"
)

const MAX_PER_PAGE = 2048

// Indexer this interface is for all the various backend indexers
type Indexer interface {

	// Config
	Config(*options.Options) error

	// Name some identifier mostly used for logs and the
	Name() string

	// Write the main function that gets a name and does magic
	Write(metric repr.StatName) error

	// ShouldWrite if true we only write to the DB, otherwise, just the ram cache is written to
	ShouldWrite() bool

	// SetTracer set the tracer object, will be used in API/get calls
	SetTracer(t opentracing.Tracer)

	// GetSpan start a tracer named span from context
	GetSpan(name string, ctx context.Context) (opentracing.Span, func())

	// Find returns a graphite like json (or other) response for a find query
	// not all backends may implement this (things like kafka for instance) and will return an error
	// /metrics/find/?query=stats.counters.cadent-graphite.all-1-stats-infra-integ.mfpaws.com.*
	/*
		[
			{
			text: "accumulator",
			expandable: 1,
			leaf: 0,
			key: "stats.counters.cadent-graphite.all-1-stats-infra-integ.mfpaws.com.accumulator"
			id: "stats.counters.cadent-graphite.all-1-stats-infra-integ.mfpaws.com.accumulator",
			allowChildren: 1
			}
		]
	*/
	Find(ctx context.Context, metric string, tags repr.SortingTags) (indexer.MetricFindItems, error)

	// FindInCache If using a Ram base indexer, this will only return items in that cache,
	// not from the datastore
	FindInCache(ctx context.Context, metric string, tags repr.SortingTags) (indexer.MetricFindItems, error)

	// List list all "paths" w/ data
	List(hasData bool, page int) (indexer.MetricFindItems, error)

	// Expand another graphite like function to expand a tree
	// /metric/expand?query=stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.success.*
	/*
		{
		results: [
		"stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.success.send",
		"stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.success.sent-bytes",
		"stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.success.valid-lines",
		"stats.counters.consthash-graphite.all-1-stats-infra-integ.mfpaws.com.success.valid-lines-sent-to-workers"
		]
		}
	*/
	Expand(metric string) (indexer.MetricExpandItem, error)

	// Start fire up the indexer
	Start()

	// Stop all processing
	Stop()

	// Delete remove an item from the index
	Delete(name *repr.StatName) error

	// GetTagsByUid get tags for a Uid String
	// not all backends may implement this
	GetTagsByUid(uniqueId string) (tags repr.SortingTags, metatags repr.SortingTags, err error)

	// GetTagsByName the incoming can be a Regex of sorts on the name
	// not all backends may implement this
	GetTagsByName(name string, page int) (indexer.MetricTagItems, error)

	// GetTagsByNameValue the incoming can be a Regex of sorts on the value
	// not all backends may implement this
	GetTagsByNameValue(name string, value string, page int) (indexer.MetricTagItems, error)

	// GetUidsByTags given some tags, grab all the matching Uids
	// not all backends may implement this
	GetUidsByTags(key string, tags repr.SortingTags, page int) ([]string, error)
}
