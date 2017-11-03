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
  Indexer Reader/Writer to match the GraphiteAPI .. but contains other things that may be useful for other API
  things, base objects in Protobuf file
*/

package indexer

import (
	"cadent/server/schemas/repr"
)

// MetricFindItems slice of MetricFindItem
type MetricFindItems []*MetricFindItem

// MetricTagItems slice of MetricTagItem
type MetricTagItems []*MetricTagItem

// MetricListItems slice of strings
type MetricListItems []string

// TagNameList slice of strings
type TagNameList []string

// UidList slice of uids
type UidList []string

// SelectValue attempt to pick the "correct" metric based on the stats name
func (m *MetricFindItem) SelectValue() uint32 {
	if m.Leaf == 0 {
		return repr.SUM // not data
	}
	// stat wins
	tgs := &repr.SortingTags{}
	tgs.SetTags(m.Tags)
	tg := tgs.Stat()
	if tg != "" {
		return repr.AggTypeFromTag(tg)
	}
	return repr.GuessReprValueFromKey(m.Id)
}
func (m *MetricFindItem) StatName() *repr.StatName {
	return &repr.StatName{Key: m.Path, Tags: m.Tags, MetaTags: m.MetaTags}
}
