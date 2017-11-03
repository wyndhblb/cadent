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
   Here we accumulate statsd metrics and then push to a output format of whatever
*/

package accumulator

import (
	"cadent/server/schemas/repr"
	"io"
	"time"
)

/****************** Interfaces *********************/
// StatItem is the interface for generic metric. Each implementation will have it's own general format
type StatItem interface {
	// Key the metric key name
	Key() repr.StatName
	// StatTime the metric time object
	StatTime() time.Time
	// Type a string name for the item
	Type() string
	// Write dump the stat to an Writer
	Write(buffer io.Writer, fmatter FormatterItem, acc AccumulatorItem)
	// Accumulate merge a metric into itself
	Accumulate(val float64, sample float64, stattime time.Time) error
	// ZeroOut reset the internal items to whatever "0" is for that metric
	ZeroOut() error
	// Repr returns the core internal metric object
	Repr() *repr.StatRepr
}

type AccumulatorItem interface {
	// Init the accumulator with the proper formatter object
	Init(FormatterItem) error
	// Stats all the accumulated bins of metrics
	Stats() map[string]StatItem
	// Flush the bunch of stats to a Writer and return the list of things we flushed
	Flush(buf io.Writer) *flushedList
	// Flushlist returns the list of metrics we should flush
	FlushList() *flushedList
	// Name pretty print name
	Name() string
	// ProcessLine parse incoming into proper internals
	ProcessLine(line []byte) error
	// ProcessLineToRepr given a chunk of data return the core internal metric object
	ProcessLineToRepr(line []byte) (*repr.StatRepr, error)
	// ProcessRepr convert the internal metric object to internals
	ProcessRepr(stat *repr.StatRepr) error
	// Reset will clear out the internal map of objects
	Reset() error
	// Tags a static set of tags to apply to all incoming items
	Tags() *repr.SortingTags
	// SetKeepKeys (default false) will not "remove" the metrics on a flush, just set them to "zero" (ZeroOut on a StatItem)
	SetKeepKeys(bool) error
	// SetTagMode for all incoming set the tag mode to "all" or "metric2"
	SetTagMode(mode repr.TagMode) error
	// SetHashMode for all incoming set the hash mode to "fnv" of "farm"
	SetHashMode(mode repr.HashMode) error
	// SetTags set the static tag list for all incoming
	SetTags(*repr.SortingTags)
	// SetResolution set the resolution we want to bin things in our accumulation window
	SetResolution(time.Duration) error
	// GetResolution return the current resolution bin
	GetResolution() time.Duration
	// SetOptions set any depending options
	SetOptions([][]string) error
	// GetOption gets an option or a default
	GetOption(name string, defaults interface{}) interface{}
}

// FormatterItem the interface that will name, value, time, etc and format it to the proper line protocol
type FormatterItem interface {
	// ToString return the line protocol string
	ToString(name *repr.StatName, val float64, tstamp int32, stats_type string, tags *repr.SortingTags) string
	// Write the protocol to an io.Writer
	Write(buf io.Writer, name *repr.StatName, val float64, tstamp int32, statsType string, tags *repr.SortingTags)
	// Type string name
	Type() string
	// Init any string options we need
	Init(...string) error
	// SetAccumulator the accumulator we are attached to
	SetAccumulator(AccumulatorItem)
	// GetAccumulator
	GetAccumulator() AccumulatorItem
}

// This is an internal struct used for the Accumulator to get both lines and StatReprs on a Flush
type flushedList struct {
	Lines []string
	Stats []*repr.StatRepr
}

func (fl *flushedList) Add(lines []string, stat *repr.StatRepr) {
	fl.Lines = append(fl.Lines, lines...)
	fl.Stats = append(fl.Stats, stat)
}

func (fl *flushedList) AddStat(stat *repr.StatRepr) {
	fl.Stats = append(fl.Stats, stat)
}
