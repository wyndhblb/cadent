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
   Here we accumulate graphite metrics and then push to a output format of whatever
   basically an internal graphite accumulator server
*/

package accumulator

import (
	"cadent/server/schemas/repr"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

/****************** RUNNERS *********************/
const GRAPHITE_ACC_NAME = "graphite_accumlator"
const GRAHPITE_ACC_MIN_LEN = 3
const GRAPHITE_ACC_MIN_FLAG = math.MinInt64

/** counter/gauge type **/

type GraphiteBaseStatItem struct {
	InKey      repr.StatName
	Values     repr.AggFloat64
	InType     string
	ReduceFunc repr.AGG_FUNC
	Time       time.Time
	Resolution time.Duration

	Min   float64
	Max   float64
	Sum   float64
	Last  float64
	Count int64

	mu sync.Mutex
}

func (s *GraphiteBaseStatItem) Repr() *repr.StatRepr {
	return &repr.StatRepr{
		Time:  s.Time.UnixNano(),
		Name:  &s.InKey,
		Min:   repr.CheckFloat(s.Min),
		Max:   repr.CheckFloat(s.Max),
		Count: s.Count,
		Sum:   repr.CheckFloat(s.Sum),
		Last:  repr.CheckFloat(s.Last),
	}
}
func (s *GraphiteBaseStatItem) StatTime() time.Time { return s.Time }
func (s *GraphiteBaseStatItem) Type() string        { return s.InType }
func (s *GraphiteBaseStatItem) Key() repr.StatName  { return s.InKey }

func (s *GraphiteBaseStatItem) ZeroOut() error {
	// reset the values
	s.Time = time.Time{}
	s.Values = repr.AggFloat64{}
	s.Min = GRAPHITE_ACC_MIN_FLAG
	s.Max = GRAPHITE_ACC_MIN_FLAG
	s.Sum = 0.0
	s.Count = 0
	s.Last = GRAPHITE_ACC_MIN_FLAG
	return nil
}

func (s *GraphiteBaseStatItem) Write(buffer io.Writer, fmatter FormatterItem, acc AccumulatorItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	val := s.ReduceFunc(s.Values)
	fmatter.Write(
		buffer,
		&s.InKey,
		val,
		int32(s.StatTime().Unix()),
		"c",
		acc.Tags(),
	)

}

// merge this item w/ another stat repr
func (s *GraphiteBaseStatItem) Merge(stat *repr.StatRepr) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Min == GRAPHITE_ACC_MIN_FLAG || s.Min > float64(stat.Min) {
		s.Min = float64(stat.Min)
	}
	if s.Max == GRAPHITE_ACC_MIN_FLAG || s.Max < float64(stat.Max) {
		s.Max = float64(stat.Max)
	}

	s.Count += stat.Count
	s.Sum += float64(stat.Sum)
	if s.Time.Before(stat.ToTime()) || s.Last == GRAPHITE_ACC_MIN_FLAG {
		s.Last = float64(stat.Last)
	}
	return nil
}

func (s *GraphiteBaseStatItem) Accumulate(val float64, sample float64, stattime time.Time) error {
	if math.IsInf(val, 0) || math.IsNaN(val) {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Values = append(s.Values, val)
	if s.Min == GRAPHITE_ACC_MIN_FLAG || s.Min > val {
		s.Min = val
	}
	if s.Max == GRAPHITE_ACC_MIN_FLAG || s.Max < val {
		s.Max = val
	}

	s.Count += 1
	s.Sum += val
	s.Last = val
	return nil
}

/******************************/
/** statsd accumulator **/
/******************************/

type GraphiteAccumulate struct {
	GraphiteStats map[string]StatItem
	OutFormat     FormatterItem
	InTags        *repr.SortingTags
	InKeepKeys    bool
	Resolution    time.Duration
	TagMode       repr.TagMode  // see repr.TAG_MODE
	HashMode      repr.HashMode // see repr.HASH_MODE

	mu sync.RWMutex
}

func NewGraphiteAccumulate() (*GraphiteAccumulate, error) {
	return new(GraphiteAccumulate), nil
}

// based on the resolution we need to aggregate around a
// "key+time bucket" mix.  to figure out the time bucket
// we simply use the resolution -- time % resolution
func (s *GraphiteAccumulate) ResolutionTime(t time.Time) time.Time {
	return t.Truncate(s.Resolution)
}

func (s *GraphiteAccumulate) MapKey(name string, t time.Time) string {
	return fmt.Sprintf("%s-%d", name, s.ResolutionTime(t).UnixNano())
}

func (s *GraphiteAccumulate) SetTagMode(mode repr.TagMode) error {
	s.TagMode = mode
	return nil
}

func (s *GraphiteAccumulate) SetHashMode(mode repr.HashMode) error {
	s.HashMode = mode
	return nil
}

func (s *GraphiteAccumulate) SetResolution(dur time.Duration) error {
	s.Resolution = dur
	return nil
}

func (s *GraphiteAccumulate) GetResolution() time.Duration {
	return s.Resolution
}

func (s *GraphiteAccumulate) SetOptions(ops [][]string) error {
	return nil
}

func (s *GraphiteAccumulate) GetOption(opt string, defaults interface{}) interface{} {
	return defaults
}

func (s *GraphiteAccumulate) Tags() *repr.SortingTags {
	return s.InTags
}

func (s *GraphiteAccumulate) SetTags(tags *repr.SortingTags) {
	s.InTags = tags
}

func (s *GraphiteAccumulate) SetKeepKeys(k bool) error {
	s.InKeepKeys = k
	return nil
}

func (s *GraphiteAccumulate) Init(fmatter FormatterItem) error {
	s.OutFormat = fmatter
	fmatter.SetAccumulator(s)
	s.GraphiteStats = make(map[string]StatItem)
	s.SetOptions([][]string{})
	return nil
}

func (s *GraphiteAccumulate) Stats() map[string]StatItem {
	return s.GraphiteStats
}

func (a *GraphiteAccumulate) Name() (name string) { return GRAPHITE_ACC_NAME }

func (a *GraphiteAccumulate) Reset() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	// keep or reset
	if a.InKeepKeys {
		for idx := range a.GraphiteStats {
			a.GraphiteStats[idx].ZeroOut()
		}
	} else {
		a.GraphiteStats = nil
		a.GraphiteStats = make(map[string]StatItem)
	}

	return nil
}

func (a *GraphiteAccumulate) Flush(buf io.Writer) *flushedList {
	fl := new(flushedList)

	a.mu.RLock()
	for _, stats := range a.GraphiteStats {
		stats.Write(buf, a.OutFormat, a)
		fl.AddStat(stats.Repr())
	}
	a.mu.RUnlock()
	a.Reset()
	return fl
}

func (a *GraphiteAccumulate) FlushList() *flushedList {
	fl := new(flushedList)

	a.mu.RLock()
	for _, stats := range a.GraphiteStats {
		fl.AddStat(stats.Repr())
	}
	a.mu.RUnlock()
	a.Reset()
	return fl
}

// ProcessRepr process a "repr" metric as if it was injected as a Graphite like item
// basically this is a post parsed string line
func (a *GraphiteAccumulate) ProcessRepr(stat *repr.StatRepr) error {

	tgs := stat.Name.SortedTags()

	// obey tag mode
	meta := &repr.SortingTags{}
	switch a.TagMode {
	case repr.TAG_ALLTAGS:
		tgs.Merge(&stat.Name.MetaTags)
		stat.Name.MetaTags = repr.SortingTags{}
	default:
		tgs, meta = repr.SplitIntoMetric2Tags(tgs, &stat.Name.MetaTags)
		stat.Name.Tags = *tgs
		stat.Name.MetaTags = *meta
	}
	sort.Sort(tgs)
	stat_key := a.MapKey(stat.Name.Key+tgs.ToStringSep(repr.DOT_SEPARATOR, repr.DOT_SEPARATOR), stat.ToTime())
	// now the accumlator
	a.mu.RLock()
	gots, ok := a.GraphiteStats[stat_key]
	a.mu.RUnlock()

	if !ok {

		stat.Name.TagMode = a.TagMode

		gots = &GraphiteBaseStatItem{
			InType:     "graphite",
			Time:       a.ResolutionTime(stat.ToTime()),
			InKey:      *stat.Name,
			Count:      0,
			Min:        float64(stat.Min),
			Max:        float64(stat.Max),
			Last:       float64(stat.Last),
			ReduceFunc: stat.Name.AggFunc(),
		}
	}

	// now for some trickery.  If the count is 1 then we assume not
	// a "pre-accumulated" repr, but basically a metric/value and need to properly accumulate
	if stat.Count == 1 {
		gots.Accumulate(float64(stat.Sum), 1.0, stat.ToTime())
	} else {
		gots.(*GraphiteBaseStatItem).Merge(stat)
	}

	// add it if not there
	if !ok {
		a.mu.Lock()
		a.GraphiteStats[stat_key] = gots
		a.mu.Unlock()
	}

	return nil
}

/*
	ProcessLine
	<key> <value> <time> <tag> <tag> ...
	<tag> are not required, but will get saved into the MetaTags of the internal rep
	as it's assumed that the "key" is the unique item we want to lookup
	tags are of the form name=val
*/
func (a *GraphiteAccumulate) ProcessLine(linebytes []byte) (err error) {
	stats_arr := strings.Fields(string(linebytes))
	l := len(stats_arr)

	if l < GRAHPITE_ACC_MIN_LEN {
		return fmt.Errorf("Accumulate: Invalid Graphite line `%s`", linebytes)
	}

	//
	key := stats_arr[0]
	val := stats_arr[1]
	_intime := stats_arr[2] // should be unix timestamp
	t := time.Now()
	i, err := strconv.ParseInt(_intime, 10, 64)
	if err == nil {
		t = time.Unix(i, 0)
	}

	f_val, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return fmt.Errorf("Accumulate: Bad Value | Invalid Graphite line `%s`", linebytes)
	}

	tags := &repr.SortingTags{}
	if l > GRAHPITE_ACC_MIN_LEN {
		// gots some potential tags
		tags = repr.SortingTagsFromArray(stats_arr[3:])
	}

	// obey tag mode
	tgs, meta := &repr.SortingTags{}, &repr.SortingTags{}
	switch a.TagMode {
	case repr.TAG_ALLTAGS:
		tgs = tags
	default:
		tgs, meta = repr.SplitIntoMetric2Tags(&repr.SortingTags{}, tags)

	}
	sort.Sort(tgs)
	stat_key := a.MapKey(key+tgs.ToStringSep(repr.DOT_SEPARATOR, repr.DOT_SEPARATOR), t)
	// now the accumlator
	a.mu.RLock()
	gots, ok := a.GraphiteStats[stat_key]
	a.mu.RUnlock()

	if !ok {

		nm := repr.StatName{Key: key, Tags: *tgs, MetaTags: *meta, TagMode: a.TagMode, HashMode: a.HashMode}

		gots = &GraphiteBaseStatItem{
			InType:     "graphite",
			Time:       a.ResolutionTime(t),
			InKey:      nm,
			Count:      0,
			Min:        GRAPHITE_ACC_MIN_FLAG,
			Max:        GRAPHITE_ACC_MIN_FLAG,
			Last:       GRAPHITE_ACC_MIN_FLAG,
			ReduceFunc: repr.GuessAggFuncFromName(&nm),
		}
	}

	// needs to lock internally if needed
	gots.Accumulate(f_val, 1.0, t)

	// add it if not there
	if !ok {
		a.mu.Lock()
		a.GraphiteStats[stat_key] = gots
		a.mu.Unlock()
	}

	return nil
}

/*
	ProcessLineToRepr
	<key> <value> <time> <tag> <tag> ...
	<tag> are not required, but will get saved into the MetaTags of the internal rep
	as it's assumed that the "key" is the unique item we want to lookup
	tags are of the form name=val

	This is like ProcessLine, but it returns a repr.StatRepr from the incoming line
*/
func (a *GraphiteAccumulate) ProcessLineToRepr(linebytes []byte) (*repr.StatRepr, error) {
	stats_arr := strings.Fields(string(linebytes))
	l := len(stats_arr)

	if l < GRAHPITE_ACC_MIN_LEN {
		return nil, fmt.Errorf("Accumulate: Invalid Graphite line `%s`", linebytes)
	}

	//
	key := stats_arr[0]
	val := stats_arr[1]
	_intime := stats_arr[2] // should be unix timestamp
	t := time.Now()
	i, err := strconv.ParseInt(_intime, 10, 64)
	if err == nil {
		t = time.Unix(i, 0)
	}

	fVal, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return nil, fmt.Errorf("Accumulate: Bad Value | Invalid Graphite line `%s`", linebytes)
	}

	tags := &repr.SortingTags{}
	meta := &repr.SortingTags{}
	if l > GRAHPITE_ACC_MIN_LEN {
		// gots some potential tags
		orgtags := repr.SortingTagsFromArray(stats_arr[3:])

		// obey tag mode
		tags, meta = &repr.SortingTags{}, &repr.SortingTags{}
		if a.TagMode == repr.TAG_METRICS2 {
			tags, meta = repr.SplitIntoMetric2Tags(&repr.SortingTags{}, orgtags)
		} else {
			tags = orgtags
		}
		sort.Sort(tags)
	}
	r := new(repr.StatRepr)
	r.Name = &repr.StatName{Key: key, Tags: *tags, TagMode: a.TagMode, MetaTags: *meta, HashMode: a.HashMode}
	r.Count = 1
	r.Sum = fVal
	r.Time = t.UnixNano()
	return r, nil

}
