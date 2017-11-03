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
	The Metric repr.StatRepr List
	Simple array of the objects, no fancy compression
*/

package series

import (
	"cadent/server/schemas/repr"
	"encoding/json"
	"sync"
)

const (
	SIMPLE_REPR_SERIES_TAG = "sarr" // just a flag to note we are using this one at the beginning of each blob
	REPR_NAME              = "repr"
)

// this can only handle "future pushing times" not random times
type ReprTimeSeries struct {
	mu *sync.Mutex

	T0      int64
	curTime int64
	Stats   repr.StatReprSlice
}

func NewReprTimeSeries(t0 int64, options *Options) *ReprTimeSeries {
	ret := &ReprTimeSeries{
		T0:    t0,
		mu:    new(sync.Mutex),
		Stats: make(repr.StatReprSlice, 0),
	}
	return ret
}

func (s *ReprTimeSeries) Name() string {
	return REPR_NAME
}

func (s *ReprTimeSeries) HighResolution() bool {
	return true
}
func (s *ReprTimeSeries) Count() int {
	return len(s.Stats)
}

func (s *ReprTimeSeries) UnmarshalBinary(data []byte) error {
	err := json.Unmarshal(data, &s.Stats)
	return err
}

func (s *ReprTimeSeries) MarshalBinary() ([]byte, error) {
	b, err := json.Marshal(s.Stats)
	return b, err
}

// this does not "finish" the series
func (s *ReprTimeSeries) Bytes() []byte {
	d, _ := s.MarshalBinary()
	return d
}

func (s *ReprTimeSeries) Len() int {
	b, _ := s.MarshalBinary()
	return len(b)
}

func (s *ReprTimeSeries) Iter() (iter TimeSeriesIter, err error) {
	s.mu.Lock()
	d := make(repr.StatReprSlice, len(s.Stats))
	copy(d, s.Stats)
	s.mu.Unlock()
	iter, err = NewReprIter(d)
	return iter, err
}

func (s *ReprTimeSeries) StartTime() int64 {
	return s.T0
}

func (s *ReprTimeSeries) LastTime() int64 {
	return s.curTime
}

func (s *ReprTimeSeries) Copy() TimeSeries {

	g := *s
	g.mu = new(sync.Mutex)
	s.mu.Lock()
	defer s.mu.Unlock()

	g.Stats = make(repr.StatReprSlice, len(s.Stats))
	copy(g.Stats, s.Stats)

	return &g
}

// the t is the "time we want to add
func (s *ReprTimeSeries) AddPoint(t int64, min float64, max float64, last float64, sum float64, count int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Stats = append(s.Stats, &repr.StatRepr{
		Time:  t,
		Min:   repr.CheckFloat(min),
		Max:   repr.CheckFloat(max),
		Last:  repr.CheckFloat(last),
		Sum:   repr.CheckFloat(sum),
		Count: count,
	})
	if t > s.curTime {
		s.curTime = t
	}
	if t < s.T0 {
		s.T0 = t
	}
	return nil
}

func (s *ReprTimeSeries) AddStat(stat *repr.StatRepr) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Stats = append(s.Stats, stat.Copy())

	if stat.Time > s.curTime {
		s.curTime = stat.Time
	}
	if stat.Time < s.T0 {
		s.T0 = stat.Time
	}
	return nil
}

// Iter lets you iterate over a series.  It is not concurrency-safe.
// but you should give it a "copy" of any byte array
type ReprIter struct {
	Stats   repr.StatReprSlice
	curIdx  int
	statLen int
	curStat *repr.StatRepr

	curTime int64
	min     float64
	max     float64
	last    float64
	sum     float64
	count   int64

	finished bool
	err      error
}

func NewReprIter(stats repr.StatReprSlice) (*ReprIter, error) {
	it := &ReprIter{
		Stats:   stats,
		curIdx:  0,
		statLen: len(stats),
	}
	return it, nil
}

func NewReprIterFromBytes(data []byte) (iter TimeSeriesIter, err error) {
	stats := new(repr.StatReprSlice)
	err = json.Unmarshal(data, stats)
	if err != nil {
		return nil, err
	}
	return NewReprIter(*stats)
}

func (it *ReprIter) Next() bool {
	if it.finished || it.curIdx >= it.statLen {
		return false
	}
	it.curStat = it.Stats[it.curIdx]
	it.curIdx++
	return true
}

func (it *ReprIter) Values() (int64, float64, float64, float64, float64, int64) {
	return it.curStat.Time,
		it.curStat.Min,
		it.curStat.Max,
		it.curStat.Last,
		it.curStat.Sum,
		it.curStat.Count
}

func (it *ReprIter) ReprValue() *repr.StatRepr {
	return &repr.StatRepr{
		Time:  it.curStat.Time,
		Min:   it.curStat.Min,
		Max:   it.curStat.Max,
		Last:  it.curStat.Last,
		Sum:   it.curStat.Sum,
		Count: it.curStat.Count,
	}
}

func (it *ReprIter) Error() error {
	return it.err
}
