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
   Base objects for our API responses and internal communications
*/

//msgp:ignore DBSeries TotalTimeSeries DBSeriesList

package metrics

import (
	"bytes"
	"cadent/server/schemas"
	"cadent/server/schemas/repr"
	"cadent/server/series"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"
)

const MAX_QUANTIZE_POINTS = 100000

// OffsetInSeries This is to aid w/ the kafka (or another) partition/offset consumption
// and other metadata that gets added as needed
type OffsetInSeries struct {
	Offset    int64
	Partition int32
	Topic     string
}

/******************  a simple union of series.TimeSeries and repr.StatName *********************/
type TotalTimeSeries struct {
	Name   *repr.StatName
	Series series.TimeSeries
	Offset *OffsetInSeries
}

/******************  structs for the raw Database query (for blob series only) *********************/

//easyjson:json
type DBSeries struct {
	Id         interface{} // don't really know what the ID format for a DB is
	Uid        string
	Start      int64
	End        int64
	Ptype      uint8
	Pbytes     []byte
	Resolution uint32
	TTL        uint32
}

func (d *DBSeries) Iter() (series.TimeSeriesIter, error) {
	s_name := series.NameFromId(d.Ptype)
	return series.NewIter(s_name, d.Pbytes)
}

//easyjson:json
type DBSeriesList []*DBSeries

func (mb DBSeriesList) Start() int64 {
	t_s := int64(0)
	for idx, d := range mb {
		if idx == 0 || (d.Start < t_s && d.Start != 0) {
			t_s = d.Start
		}
	}
	return t_s
}

func (mb DBSeriesList) End() int64 {
	t_s := int64(0)
	for idx, d := range mb {
		if idx == 0 || d.End > t_s {
			t_s = d.End
		}
	}
	return t_s
}

func (mb DBSeriesList) ToRawRenderItem() (*RawRenderItem, error) {
	rawd := new(RawRenderItem)
	for _, d := range mb {
		iter, err := d.Iter()
		if err != nil {
			return nil, err
		}
		for iter.Next() {
			to, mi, mx, ls, su, ct := iter.Values()
			t := uint32(time.Unix(0, to).Unix())
			rawd.Data = append(rawd.Data, &RawDataPoint{
				Count: ct,
				Min:   mi,
				Max:   mx,
				Last:  ls,
				Sum:   su,
				Time:  t,
			})
		}
	}
	if len(rawd.Data) > 0 {
		sort.Sort(RawDataPointList(rawd.Data))
		rawd.RealStart = rawd.Data[0].Time
		rawd.Start = rawd.RealStart
		rawd.End = rawd.Data[len(rawd.Data)-1].Time
		rawd.RealEnd = rawd.End
	}

	return rawd, nil
}

// ToRawRenderItemBounds get a RawRenderItem from the timeseries list in the time bounded
// by a start and end time
// times are in UnixNano
func (mb DBSeriesList) ToRawRenderItemBounds(sTime int64, eTime int64) (*RawRenderItem, error) {
	rawd := new(RawRenderItem)
	for _, d := range mb {
		iter, err := d.Iter()
		if err != nil {
			return nil, err
		}
		for iter.Next() {
			to, mi, mx, ls, su, ct := iter.Values()
			if to < sTime || to > eTime {
				continue
			}
			t := uint32(time.Unix(0, to).Unix())
			rawd.Data = append(rawd.Data, &RawDataPoint{
				Count: ct,
				Min:   mi,
				Max:   mx,
				Last:  ls,
				Sum:   su,
				Time:  t,
			})
		}
	}
	if len(rawd.Data) > 0 {
		sort.Sort(RawDataPointList(rawd.Data))
		rawd.RealStart = rawd.Data[0].Time
		rawd.Start = rawd.RealStart
		rawd.End = rawd.Data[len(rawd.Data)-1].Time
		rawd.RealEnd = rawd.End
	}

	return rawd, nil
}

// ToRawRenderItemPool same as ToRawRenderItem put pulls RawDataPpints from a sync pool
// for temporary instances of this
func (mb DBSeriesList) ToRawRenderItemPool() (*RawRenderItem, error) {
	rawd := new(RawRenderItem)
	lastT := uint32(0)
	idx := -1
	for _, d := range mb {
		iter, err := d.Iter()
		if err != nil {
			return nil, err
		}
		for iter.Next() {
			to, mi, mx, ls, su, ct := iter.Values()
			t := uint32(time.Unix(0, to).Unix())
			pt := GetRawDataPoint()
			pt.Count = ct
			pt.Sum = su
			pt.Max = mx
			pt.Min = mi
			pt.Last = ls
			pt.Time = t

			// deal w/ same times properly
			if lastT == t && idx > -1 {
				rawd.Data[idx].Merge(pt)
			} else {
				rawd.Data = append(rawd.Data, pt)
				idx++
				lastT = t
			}

		}
	}
	if len(rawd.Data) > 0 {
		sort.Sort(RawDataPointList(rawd.Data))
		rawd.RealStart = rawd.Data[0].Time
		rawd.Start = rawd.RealStart
		rawd.End = rawd.Data[len(rawd.Data)-1].Time
		rawd.RealEnd = rawd.End
	}

	return rawd, nil
}

// ToRawRenderItemBoundsPool same as ToRawRenderItemBounds put pulls RawDataPpints from a sync pool
// for temporary instances of this and trims anything out not in the time bounds
// times are in UnixNano
func (mb DBSeriesList) ToRawRenderItemBoundsPool(sTime int64, eTime int64) (*RawRenderItem, error) {
	rawd := new(RawRenderItem)
	lastT := uint32(0)
	idx := -1
	for _, d := range mb {
		iter, err := d.Iter()
		if err != nil {
			return nil, err
		}
		for iter.Next() {
			to, mi, mx, ls, su, ct := iter.Values()
			if to <= 0 || to < sTime || to > eTime {
				continue
			}
			t := uint32(time.Unix(0, to).Unix())
			pt := GetRawDataPoint()
			pt.Count = ct
			pt.Sum = su
			pt.Max = mx
			pt.Min = mi
			pt.Last = ls
			pt.Time = t

			// deal w/ same times properly
			if lastT == t && idx > -1 {
				rawd.Data[idx].Merge(pt)
			} else {
				rawd.Data = append(rawd.Data, pt)
				idx++
				lastT = t
			}

		}
	}
	if len(rawd.Data) > 0 {
		sort.Sort(RawDataPointList(rawd.Data))
		rawd.RealStart = rawd.Data[0].Time
		rawd.Start = rawd.RealStart
		rawd.End = rawd.Data[len(rawd.Data)-1].Time
		rawd.RealEnd = rawd.End
	}

	return rawd, nil
}

/****************** Output structs for the graphite API*********************/

func NewDataPoint(time uint32, val float64) *DataPoint {
	d := &DataPoint{Time: time, Value: val}
	return d
}

func (d DataPoint) MarshalJSON() ([]byte, error) {
	if math.IsNaN(d.Value) {
		return []byte(fmt.Sprintf("[null, %d]", d.Time)), nil
	}

	return []byte(fmt.Sprintf("[%f, %d]", d.Value, d.Time)), nil
}

func (d DataPoint) SetValue(val *float64) {
	d.Value = *val
}

//easyjson:json
type RenderItems []*RenderItem

/****************** WhisperRenderItem API outs *********************/

// MarshalJSON custom json-er for sorting the series by keys before the marshal, as graphite/grafana
// colors series by their order, and mixing up the order mixes the colors each refresh
func (w *WhisperRenderItem) MarshalJSON() ([]byte, error) {

	var bs []byte

	bs = append(bs, `{"real_start":`...)
	bs = strconv.AppendUint(bs, uint64(w.RealStart), 10)
	bs = append(bs, `,"real_end":`...)
	bs = strconv.AppendUint(bs, uint64(w.RealEnd), 10)
	bs = append(bs, `,"start":`...)
	bs = strconv.AppendUint(bs, uint64(w.Start), 10)
	bs = append(bs, `,"end":`...)
	bs = strconv.AppendUint(bs, uint64(w.End), 10)
	bs = append(bs, `,"from":`...)
	bs = strconv.AppendUint(bs, uint64(w.From), 10)
	bs = append(bs, `,"to":`...)
	bs = strconv.AppendUint(bs, uint64(w.To), 10)
	bs = append(bs, `,"step":`...)
	bs = strconv.AppendUint(bs, uint64(w.Step), 10)
	bs = append(bs, `,`...)

	// grab names and sort them
	tNames := make([]string, len(w.Series))
	idx := 0
	for n := range w.Series {
		tNames[idx] = n
		idx++
	}
	sort.Strings(tNames)
	l := len(tNames)
	for i, n := range tNames {
		bbs, err := json.Marshal(w.Series[n])
		if err != nil {
			fmt.Println("Json Error: ", err)
			continue
		}

		bs = strconv.AppendQuoteToASCII(bs, n)
		bs = append(bs, repr.COLON_SEPARATOR_BYTE...)
		bs = append(bs, bbs...)

		if i < l-1 {
			bs = append(bs, ',')
		}
	}
	bs = append(bs, '}')

	return bs, nil
}

//easyjson:json
type GraphiteApiItems []*GraphiteApiItem

// Sorting for the target outputs
func (p GraphiteApiItems) Len() int { return len(p) }
func (p GraphiteApiItems) Less(i, j int) bool {
	return p[i].Target < p[j].Target
}
func (p GraphiteApiItems) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

// the basic whisper metric json blob for find

/****************** Output structs for the internal API*********************/

func (d *RawDataPoint) floatToJson(buf *bytes.Buffer, name string, end string, val float64) {
	if math.IsNaN(val) {
		fmt.Fprintf(buf, name+":null"+end)
		return
	}
	fmt.Fprintf(buf, name+":%f"+end, val)
}

func (d *RawDataPoint) floatToJsonByte(bs []byte, name string, end string, val float64) []byte {
	bs = strconv.AppendQuoteToASCII(bs, name)
	bs = append(bs, `:`...)
	if math.IsNaN(val) {
		return append(bs, `null`...)
	}
	bs = strconv.AppendFloat(bs, val, 'f', -1, 64)
	return append(bs, end...)
}

// MarshalJSON a more optimized less GC heavy json marshaller
func (d RawDataPoint) MarshalJSON() ([]byte, error) {
	var bs []byte
	bs = append(bs, `{"time":`...)
	bs = strconv.AppendUint(bs, uint64(d.Time), 10)
	bs = append(bs, `,`...)
	bs = d.floatToJsonByte(bs, `sum`, ",", d.Sum)
	bs = d.floatToJsonByte(bs, `min`, ",", d.Min)
	bs = d.floatToJsonByte(bs, `max`, ",", d.Max)
	bs = d.floatToJsonByte(bs, `last`, ",", d.Last)
	bs = append(bs, `"count":`...)
	bs = strconv.AppendInt(bs, d.Count, 10)
	bs = append(bs, '}')
	return bs, nil
}

func NullRawDataPoint(time uint32) *RawDataPoint {
	out := new(RawDataPoint)
	out.Count = 0
	out.Time = time
	out.Sum = math.NaN()
	out.Last = math.NaN()
	out.Min = math.NaN()
	out.Max = math.NaN()
	return out
}

func (r *RawDataPoint) IsNull() bool {
	return math.IsNaN(r.Sum) && math.IsNaN(r.Last) && math.IsNaN(r.Min) && math.IsNaN(r.Max)
}

func (r *RawDataPoint) MakeNull(t0 uint32) {
	r.Count = 0
	r.Time = t0
	r.Sum = math.NaN()
	r.Last = math.NaN()
	r.Min = math.NaN()
	r.Max = math.NaN()
}

func (r *RawDataPoint) AggValue(aggfunc uint32) float64 {

	// if the count is 1 there is only but one real value
	if r.Count == 1 {
		return r.Sum
	}

	switch aggfunc {
	case repr.SUM:
		return r.Sum
	case repr.MIN:
		return r.Min
	case repr.MAX:
		return r.Max
	case repr.LAST:
		return r.Last
	case repr.COUNT:
		return float64(r.Count)
	default:
		if r.Count > 0 {
			return r.Sum / float64(r.Count)
		}
		return math.NaN()
	}
}

// merge two data points into one .. this is a nice merge that will add counts, etc to the pieces
// the "time" is then the greater of the two
func (r *RawDataPoint) Merge(d *RawDataPoint) {
	if d.IsNull() {
		return // just skip as nothing to merge
	}
	if math.IsNaN(r.Max) || r.Max < d.Max {
		r.Max = d.Max
	}
	if math.IsNaN(r.Min) || r.Min > d.Min {
		r.Min = d.Min
	}

	if math.IsNaN(r.Sum) && !math.IsNaN(d.Sum) {
		r.Sum = d.Sum
	} else if !math.IsNaN(r.Sum) && !math.IsNaN(d.Sum) {
		r.Sum += d.Sum
	}

	if d.Time != 0 && d.Time > r.Time && !math.IsNaN(d.Last) {
		r.Last = d.Last
	}
	if math.IsNaN(r.Last) && !math.IsNaN(d.Last) {
		r.Last = d.Last
	}

	if r.Count == 0 && d.Count > 0 {
		r.Count = d.Count
	} else if r.Count > 0 && d.Count > 0 {
		r.Count += d.Count
	}

	if r.Time < d.Time {
		r.Time = d.Time
	}

}

type RawDataPointList []*RawDataPoint

func (v RawDataPointList) Len() int           { return len(v) }
func (v RawDataPointList) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (v RawDataPointList) Less(i, j int) bool { return v[i].Time < v[j].Time }

func NewRawRenderItemFromSeriesIter(iter series.TimeSeriesIter) (*RawRenderItem, error) {
	rawd := new(RawRenderItem)

	for iter.Next() {
		to, mi, mx, ls, su, ct := iter.Values()
		t := uint32(time.Unix(0, to).Unix())

		rawd.Data = append(rawd.Data, &RawDataPoint{
			Count: ct,
			Sum:   su,
			Max:   mx,
			Min:   mi,
			Last:  ls,
			Time:  t,
		})
	}
	if len(rawd.Data) > 0 {
		sort.Sort(RawDataPointList(rawd.Data))

		rawd.RealStart = rawd.Data[0].Time
		rawd.Start = rawd.RealStart

		rawd.End = rawd.Data[len(rawd.Data)-1].Time
		rawd.RealEnd = rawd.End
	}

	return rawd, nil
}

func NewRawRenderItemFromSeries(ts *TotalTimeSeries) (*RawRenderItem, error) {
	s_iter, err := ts.Series.Iter()
	if err != nil {
		return nil, err
	}
	rawd, err := NewRawRenderItemFromSeriesIter(s_iter)
	if err != nil {
		return nil, err
	}

	rawd.Id = ts.Name.UniqueIdString()
	rawd.MetaTags = ts.Name.MetaTags
	rawd.Tags = ts.Name.Tags
	rawd.Metric = ts.Name.Key
	rawd.Step = ts.Name.Resolution

	return rawd, nil
}

func (r *RawRenderItem) Len() int {
	return len(r.Data)
}

// Copy
func (r *RawRenderItem) Copy() *RawRenderItem {
	n := new(RawRenderItem)
	n.AggFunc = r.AggFunc
	n.End = r.End
	n.Id = r.Id
	n.Step = r.Step
	n.Start = r.Start
	n.RealEnd = r.RealEnd
	n.RealStart = r.RealStart
	n.Tags = r.Tags
	n.MetaTags = r.MetaTags
	n.Metric = r.Metric
	n.InCache = r.InCache
	n.UsingCache = r.UsingCache
	n.Data = make([]*RawDataPoint, len(r.Data))
	copy(n.Data, r.Data)
	return n
}

func (r *RawRenderItem) PrintPoints() {
	fmt.Printf("RawRenderItem: %s (%s) Start: %d End: %d, Points: %d\n", r.Metric, r.Id, r.Start, r.End, r.Len())
	for idx, d := range r.Data {
		fmt.Printf("%d: %d %f %f %f %f %d\n", idx, d.Time, d.Min, d.Max, d.Last, d.Sum, d.Count)
	}
}

// is True if the start and end times are contained in this data blob
func (r *RawRenderItem) DataInRange(start uint32, end uint32) bool {
	return r.RealStart >= start && end <= r.RealEnd
}

func (r *RawRenderItem) ToDataPoint() []DataPoint {
	dpts := make([]DataPoint, len(r.Data), len(r.Data))
	for idx, d := range r.Data {
		dpts[idx].Time = d.Time
		use_v := d.AggValue(r.AggFunc)
		dpts[idx].SetValue(&use_v)
	}
	return dpts
}

// somethings like read caches, are hot data we most of the time
// only need to make sure the "start" is in range as the end may not have happened yet
func (r *RawRenderItem) StartInRange(start uint32) bool {
	return start >= r.RealStart
}

// TruncateTo trim the data array so that it fit's w/i the start and end times
// it is inclusive of the start and end times
func (r *RawRenderItem) TruncateTo(start uint32, end uint32) int {
	idxStart := -1
	idxEnd := len(r.Data)
	for i, d := range r.Data {
		if d.Time >= start && d.Time <= end {
			if idxStart == -1 {
				idxStart = i
			}
			idxEnd = i
		}
	}
	if idxStart != -1 {
		r.Data = r.Data[idxStart:idxEnd]
	}
	if len(r.Data) > 0 {
		r.RealEnd = r.Data[len(r.Data)-1].Time
		r.RealStart = r.Data[0].Time
	}
	return len(r.Data)
}

// PutDataInPool puts the Data list (RawDataPoint) back into the pool
// when the objects is "done"
func (r *RawRenderItem) PutDataIntoPool() {
	for _, item := range r.Data {
		PutRawDataPoint(item)
	}
}

// a crude "resampling" of incoming data .. if the incoming is non-uniform in time
// and we need to smear the resulting set into a uniform vector
//
// If moving from a "large" time step to a "smaller" one you WILL GET NULL values for
// time slots that do not match anything .. we cannot (will not) interpolate data like that
// as it's 99% statistically "wrong"
// this will "quantize" things the Start/End set times, not the RealStart/End
//
func (r *RawRenderItem) ResampleAndQuantize(step uint32) {

	cur_len := uint32(len(r.Data))
	// nothing we can do here
	if cur_len <= 1 {
		return
	}

	// figure out the lengths for the new vector (note the difference in this
	// start time from the Quantize .. we need to go below things
	start := r.Start
	left := r.Start % step
	if left != 0 {
		start = r.Start - step + left
	}

	endTime := (r.End - step) - ((r.End - step) % step)

	if endTime < start {
		// basically the resampling makes "one" data point so we merge all the
		// incoming points into this one
		endTime = start + step
		data := make([]*RawDataPoint, 1)
		data[0] = NullRawDataPoint(start)
		for _, d := range r.Data {
			data[0].Merge(d)
		}
		r.Start = start
		r.RealStart = start
		r.Step = step
		r.End = endTime
		r.RealEnd = endTime
		r.Data = data
		return
	}

	// data length of the new data array
	data := make([]*RawDataPoint, (endTime-start)/step+1)

	//log.Error("RESAMPLE\n\n")
	// 'i' iterates Original Data.
	// 'o' iterates OutGoing Data.
	// t is the current time we need to merge
	for t, i, o := start, uint32(0), -1; t <= endTime; t += step {
		o++

		// start at null
		if data[o] == nil || !data[o].IsNull() {
			data[o] = NullRawDataPoint(t)
		}

		// past any valid points
		if i >= cur_len {
			data[o] = NullRawDataPoint(t)
			continue
		}
		// need to collect all the points that fit into the step
		// remembering that the "first" bin may include points "behind" it (by a step) from the original list
		// list
		p := r.Data[i]
		//log.Errorf("Resample: %d: cur T: %d to T: %d -- DataP: %v", i, t, (t + step), r.Data[i])

		// if the point is w/i [start-step, start)
		// then these belong to the current Bin
		if p.Time < start && p.Time >= (t-step) && p.Time < t {
			if data[o].IsNull() {
				data[o] = p
			} else {
				data[o].Merge(p)
			}
			data[o].Time = t
			i++
			p = r.Data[i]
		}

		// if the points is w/i [t, t+step)
		// grab all the points in the step range and "merge"
		if p.Time >= t && p.Time < (t+step) {
			//log.Errorf("Start Merge FP: cur T: %d to T: %d -- DataP: %v :: [t, t+step): %v [t-step, t): %v ", t, (t + step), p, p.Time >= t && p.Time < (t+step), p.Time >= (t-step) && p.Time < t)
			// start at current point and merge up
			if data[o].IsNull() {
				data[o] = p
			} else {
				data[o].Merge(p)
			}

			//log.Errorf("Start Merge FP: cur T: %d to T: %d -- DataP: %v", t, (t + step), p)
			for {
				i++
				if i >= cur_len {
					break
				}
				np := r.Data[i]
				if np.Time >= (t + step) {
					break
				}
				//log.Errorf("Merging: cur T: %d to T: %d -- DataP: %v", t, (t + step), np)
				data[o].Merge(np)
			}
			data[o].Time = t
		}
	}

	r.Start = start
	r.RealStart = start
	r.Step = step
	r.End = endTime
	r.RealEnd = endTime
	r.Data = data
}

// reample but "skip" and nils
func (r *RawRenderItem) Resample(step uint32) error {
	// based on the start, stop and step.  Fill in the gaps in missing
	// slots (w/ nulls) as graphite does not like "missing times" (expects things to have a constant
	// length over the entire interval)
	// You should Put in an "End" time of "ReadData + Step" to avoid loosing the last point as things
	// are  [Start, End) not [Start, End]

	if step <= 0 {
		return schemas.ErrQuantizeStepTooSmall
	}

	// make sure the start/ends are nicely divisible by the Step
	start := r.RealStart

	left := start % step
	if left != 0 {
		start = start + step - left
	}

	end := r.RealEnd

	endTime := (end - 1) - ((end - 1) % step) + step

	// make sure in time order
	sort.Sort(RawDataPointList(r.Data))

	// 'i' iterates Original Data.
	// 'o' iterates Incoming Data.
	// 'n' iterates New Data.
	// t is the current time we need to fill/merge
	var t uint32
	var i, n int

	iLen := len(r.Data)
	data := make([]*RawDataPoint, iLen)
	copy(data, r.Data)
	dp := NullRawDataPoint(start)
	r.Data = r.Data[:0]
	for t, i, n = start, 0, -1; t <= endTime; t += step {
		// loop through the orig data until we hit a time > then the current one
		for ; i < iLen; i++ {
			if data[i] == nil {
				continue
			}
			if data[i].Time < (t - step) {
				continue
			}
			if data[i].Time > t {
				break
			}
			dp.Merge(data[i])
		}

		if dp != nil && !dp.IsNull() {
			dp.Time = t
			r.Data = append(r.Data, dp)
			dp = NullRawDataPoint(t)
			n++
		}
	}

	r.Start = start
	r.RealStart = start
	r.Step = step
	r.End = endTime
	r.RealEnd = endTime
	return nil

}

func (r *RawRenderItem) Quantize() error {
	return r.QuantizeToStep(r.Step)
}

// QuantizeToStep based on the start, stop and step.  Fill in the gaps in missing
// slots (w/ nulls) as graphite does not like "missing times" (expects things to have a constant
// length over the entire interval)
// You should Put in an "End" time of "ReadData + Step" to avoid loosing the last point as things
// are  [Start, End) not [Start, End]
func (r *RawRenderItem) QuantizeToStep(step uint32) error {

	if step <= 0 {
		return schemas.ErrQuantizeStepTooSmall
	}

	// make sure the start/ends are nicely divisible by the Step
	start := r.Start
	left := r.Start % step
	if left != 0 {
		start = r.Start + step - left
	}

	endTime := (r.End - 1) - ((r.End - 1) % step) + step

	// make sure in time order
	rData := RawDataPointList(r.Data)
	sort.Sort(rData)

	// basically all the data fits into one point
	if endTime < start {
		data := make([]*RawDataPoint, 1)
		r.Step = step
		r.Start = start
		r.End = start + step
		r.RealEnd = start + step
		r.RealStart = start
		data[0] = NullRawDataPoint(start)
		for _, d := range r.Data {
			data[0].Merge(d)

		}
		r.Data = data
		return nil
	}

	// trap too many points
	num_pts := ((endTime - start) / step) + 1
	if num_pts >= MAX_QUANTIZE_POINTS {
		return schemas.ErrQuantizeStepTooManyPoints
	}

	// data length of the new data array
	data := make([]*RawDataPoint, num_pts)
	cur_len := uint32(len(r.Data))

	// 'i' iterates Original Data.
	// 'o' iterates OutGoing Data.
	// t is the current time we need to fill/merge
	var t, i uint32
	var o int32
	for t, i, o = start, uint32(0), -1; t <= endTime; t += step {
		o += 1

		// No more data in the original list
		if i >= cur_len {
			data[o] = NullRawDataPoint(t)
			continue
		}

		p := rData[i]
		if p.Time == t {
			// perfect match
			data[o] = p
			i++
		} else if p.Time > t {
			// data is too recent, so we need to "skip" the slot and move on
			// unless it as merged already
			if data[o] == nil || data[o].Time == 0 {
				data[o] = NullRawDataPoint(t)
			}

			if p.Time >= endTime {
				data[o].Merge(p)
			}

		} else if p.Time > t-step && p.Time < t {
			// data fits in a slot,
			// but may need a merge w/ another point(s) in the parent list
			// so we advance "i"
			p.Time = t
			if data[o] != nil && data[o].Time != 0 && !data[0].IsNull() {
				data[o].Merge(p)
			} else {
				data[o] = p
			}
			i++
		} else if p.Time <= t-step {
			// point is too old. move on until we find one that's not
			// but we need to redo the above logic to put it in the proper spot (and put nulls where
			// needed (thus the -= 1 bits)
			for p.Time <= t-step && i < cur_len-1 {
				i++
				p = r.Data[i]
			}
			if p.Time <= t-step {
				i++
			}
			t -= step
			o -= 1
		}
	}

	r.Start = start
	r.RealStart = start
	r.Step = step
	r.End = endTime
	r.RealEnd = endTime
	r.Data = data
	return nil
}

// merges 2 series into the current one ..
func (r *RawRenderItem) Merge(m *RawRenderItem) error {

	if m.Start < r.Start {
		r.Start = m.Start
	}
	if m.End > r.End {
		r.End = m.End
	}

	if m.RealStart < r.RealStart {
		r.RealStart = m.RealStart
	}

	if m.RealEnd > r.RealEnd {
		r.RealEnd = m.RealEnd
	}

	// easy one we just merge the two lists and sort it (because we're too lazy to do linked
	// list things)
	r.Data = append(r.Data, m.Data...)
	sort.Sort(RawDataPointList(r.Data))

	r.Tags = repr.MergeTagLists(r.Tags, m.Tags)
	r.MetaTags = repr.MergeTagLists(r.MetaTags, m.MetaTags)
	if m.InCache {
		r.InCache = m.InCache
	}

	return nil
}

// when we "merge" the two series we also "aggregate" the overlapping
// data points
func (r *RawRenderItem) MergeAndAggregate(m *RawRenderItem) error {
	// steps sizes need to be the same
	if r.Step != m.Step {
		return schemas.ErrMergeStepSizeError
	}

	if m.Start < r.Start {
		r.Start = m.Start
	} else {
		m.Start = r.Start
	}
	if m.End > r.End {
		r.End = m.End
	} else {
		m.End = r.End
	}
	if m.RealStart < r.RealStart {
		r.RealStart = m.RealStart
	} else {
		m.RealStart = r.RealStart
	}
	if m.RealEnd > r.RealEnd {
		r.RealEnd = m.RealEnd
	} else {
		m.RealEnd = r.RealEnd
	}

	// both series should be the same size after this step
	err := r.Quantize()
	if err != nil {
		return err
	}

	err = m.Quantize()
	if err != nil {
		return err
	}

	// find the "longest" one
	cur_len := len(m.Data)
	for i := 0; i < cur_len; i++ {
		if r.Data[i] == nil || r.Data[i].IsNull() {
			r.Data[i] = m.Data[i]
		} else {
			r.Data[i].Merge(m.Data[i])
		}
	}

	r.Tags = repr.MergeTagLists(r.Tags, m.Tags)
	r.MetaTags = repr.MergeTagLists(r.MetaTags, m.MetaTags)

	if m.InCache {
		r.InCache = m.InCache
	}

	return nil
}

// MergeWithResample this is similar to the MergeAndAgg but it will NOT create nulls
// for missing points in a time sequence, and Resample the data at the same time
// the `d` here is assumed to have "earlier" data while the current object is the later data in time
// do not use this for up sampling (i.e. converting between a larger step and a small one)
// that data will be invalid as there is no granularity to accurately split the lists
func (r *RawRenderItem) MergeWithResample(d *RawRenderItem, step uint32) error {
	// based on the start, stop and step.  Fill in the gaps in missing
	// slots (w/ nulls) as graphite does not like "missing times" (expects things to have a constant
	// length over the entire interval)
	// You should Put in an "End" time of "ReadData + Step" to avoid loosing the last point as things
	// are  [Start, End) not [Start, End]

	if step <= 0 {
		return schemas.ErrQuantizeStepTooSmall
	}

	// make sure the start/ends are nicely divisible by the Step
	start := r.RealStart
	if d.RealStart < start {
		start = d.RealStart
	}

	left := start % step
	if left != 0 {
		start = start + step - left
	}

	end := r.RealEnd
	if d.RealEnd > r.RealEnd {
		end = d.RealEnd
	}

	endTime := (end - 1) - ((end - 1) % step) + step

	// make sure in time order
	sort.Sort(RawDataPointList(r.Data))
	sort.Sort(RawDataPointList(d.Data))

	// 'i' iterates Original Data.
	// 'o' iterates Incoming Data.
	// 'n' iterates New Data.
	// t is the current time we need to fill/merge
	var t uint32
	var i, o, n int

	iLen := len(r.Data)
	oLen := len(d.Data)

	dEndTime := uint32(0)
	//dStartTime := uint32(0)
	if oLen > 0 {
		dEndTime = d.Data[oLen-1].Time
		//dStartTime = d.Data[0].Time
	}
	rEndTime := uint32(0)
	//rStartTime := uint32(0)
	if iLen > 0 {
		rEndTime = r.Data[iLen-1].Time
		//rStartTime = r.Data[0].Time
	}

	ptsToMerge := make([]*RawDataPoint, 0)
	Rs := make([]*RawDataPoint, len(r.Data))
	copy(Rs, r.Data)
	r.Data = r.Data[:0]

	/*
		for _, d := range d.Data {
			fmt.Println("DB data: ", d.Time, d.Sum, d.Count)
		}
		for _, d := range Rs {
			fmt.Println("Cache data: ", d.Time, d.Sum, d.Count)
		}*/
	for t, i, o, n = start, 0, 0, -1; t <= endTime; t += step {
		// loop through the incoming data until we hit a time > then the current one
		// give a little step buffer in case things we go from a smaller res to a higher one

		if dEndTime > 0 && oLen > 0 {
			for ; o < oLen; o++ {
				if d.Data[o] == nil {
					continue
				}
				//fmt.Println("DB LOOP: index: ", o, " inTime:", d.Data[o].Time, "count:", d.Data[o].Count, "on Time:", t, "skip:", d.Data[o].Time > t)

				if d.Data[o].Time > t {
					break
				}
				//d.Data[o].Time = t
				ptsToMerge = append(ptsToMerge, d.Data[o])

			}
		}

		// idiot checks so that we don't scan the entire list if he times of the `d` list are
		// vastly different then the `r` list, otherwise we just scan `r` every single time, which for
		// large lists is not good performance.
		if rEndTime > 0 && iLen > 0 {
			//fmt.Println("Inflight LOOP: On Time:", t)

			// loop through the orig data until we hit a time > then the current one
			for ; i < iLen; i++ {
				if Rs[i] == nil {
					continue
				}
				//fmt.Println("Inflight LOOP: index: ", i, " inTime:", Rs[i].Time, "count:", Rs[i].Count, "on Time:", t, "skip:", Rs[i].Time > t)
				if Rs[i].Time > t {
					break
				}
				//Rs[i].Time = t
				ptsToMerge = append(ptsToMerge, Rs[i])
			}
		}

		// NOTE: we keep the original times not the "step" time here.  the original being the max(time) for a
		// range of ptsToMerge as we may need to merge this list again, and the merge will be more accurate
		// by keeping the raw times.
		l := len(ptsToMerge)
		if len(ptsToMerge) > 0 {
			//mPt := *ptsToMerge[0]
			switch l {
			case 1:

			case 2:
				ptsToMerge[0].Merge(ptsToMerge[1])
			default:
				sort.Sort(RawDataPointList(ptsToMerge))
				for _, pt := range ptsToMerge[1:] {
					ptsToMerge[0].Merge(pt)
				}
			}
			//ptsToMerge[0].Time = t
			r.Data = append(r.Data, ptsToMerge[0])
			ptsToMerge = ptsToMerge[:0]
			n++
		}
	}
	r.Start = start
	r.RealStart = start
	r.Step = step
	r.End = endTime
	r.RealEnd = endTime
	r.Tags = repr.MergeTagLists(r.Tags, d.Tags)
	r.MetaTags = repr.MergeTagLists(r.MetaTags, d.MetaTags)

	if d.InCache {
		r.InCache = d.InCache
	}
	/*for _, d := range r.Data {
		fmt.Println("Out data: ", d.Time, d.Sum, d.Count)
	}*/
	return nil

}

// JoinWithResample takes two lists and joins them, but does not "merge" the data points
// like MergeWithResample, resampling the list as it goes
func (r *RawRenderItem) JoinWithResample(d *RawRenderItem, step uint32) error {
	// based on the start, stop and step.  Fill in the gaps in missing
	// slots (w/ nulls) as graphite does not like "missing times" (expects things to have a constant
	// length over the entire interval)
	// You should Put in an "End" time of "ReadData + Step" to avoid loosing the last point as things
	// are  [Start, End) not [Start, End]

	if step <= 0 {
		return schemas.ErrQuantizeStepTooSmall
	}

	// make sure the start/ends are nicely divisible by the Step
	start := r.RealStart
	if d.RealStart < start {
		start = d.RealStart
	}

	left := start % step
	if left != 0 {
		start = start + step - left
	}

	end := r.RealEnd
	if d.RealEnd > r.RealEnd {
		end = d.RealEnd
	}

	endTime := (end - 1) - ((end - 1) % step) + step

	// make sure in time order
	sort.Sort(RawDataPointList(r.Data))
	sort.Sort(RawDataPointList(d.Data))

	// 'i' iterates Original Data.
	// 'o' iterates Incoming Data.
	// 'n' iterates New Data.
	// t is the current time we need to fill/merge
	var t uint32
	var i, o, n int

	i_len := len(r.Data)
	o_len := len(d.Data)
	data := make([]*RawDataPoint, 0)

	dp := GetNullRawDataPoint(start)
	for t, i, o, n = start, 0, 0, -1; t <= endTime; t += step {
		// loop through the orig data until we hit a time > then the current one
		for ; i < i_len; i++ {
			if r.Data[i] == nil {
				continue
			}
			if r.Data[i].Time <= (t - step) {
				continue
			}
			if r.Data[i].Time > t {
				break
			}
			dp = r.Data[i] // no merge
		}

		// loop through the incoming data until we hit a time > then the current one
		for ; o < o_len; o++ {
			if d.Data[o] == nil {
				continue
			}
			if d.Data[o].Time <= (t - step) {
				continue
			}
			if d.Data[o].Time > t {
				break
			}
			// not seen the time so onward
			if dp.IsNull() {
				dp = d.Data[o] // no merge
			}
		}
		if dp != nil && !dp.IsNull() {
			data = append(data, dp)
			dp = GetNullRawDataPoint(t)
			n++
		}
	}

	r.Start = start
	r.RealStart = start
	r.Step = step
	r.End = endTime
	r.RealEnd = endTime
	r.Data = data
	r.Tags = repr.MergeTagLists(r.Tags, d.Tags)
	r.MetaTags = repr.MergeTagLists(r.MetaTags, d.MetaTags)

	if d.InCache && !r.InCache {
		r.InCache = d.InCache
	}

	return nil

}

// RawRenderItems list of *RawRenderItem
//easyjson:json
type RawRenderItems []*RawRenderItem

// SyncPool of RawDataPoint for GC goodies

var rawDataPointPool sync.Pool

// GetRawDataPoint get a new RawDataPoint from the pool
func GetRawDataPoint() *RawDataPoint {
	x := rawDataPointPool.Get()
	if x == nil {
		return new(RawDataPoint)
	}
	return x.(*RawDataPoint)
}

// GetRawDataPoint get a new RawDataPoint from the pool
func GetNullRawDataPoint(t0 uint32) *RawDataPoint {
	x := rawDataPointPool.Get()
	if x == nil {
		return NullRawDataPoint(t0)
	}
	d := x.(*RawDataPoint)
	d.MakeNull(t0)
	return d

}

// PutRawDataPoint a RawDataPoint back to the pool
func PutRawDataPoint(spl *RawDataPoint) {
	rawDataPointPool.Put(spl)
}
