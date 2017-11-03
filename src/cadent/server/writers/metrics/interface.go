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
   Writers/Readers of stats

   We are attempting to mimic the Graphite API json blobs throughout the process here
   such that we can hook this in directly to either graphite-api/web

   NOTE: this is not a full graphite DSL, just paths and metrics, we leave the fancy functions inside
   graphite-api/web to work their magic .. one day we'll implement the full DSL, but until then ..

   Currently just implementing /find /expand and /render (json only) for graphite-api
*/

package metrics

import (
	sindexer "cadent/server/schemas/indexer"
	"cadent/server/schemas/metrics"
	smetrics "cadent/server/schemas/metrics"
	"cadent/server/schemas/repr"
	sseries "cadent/server/schemas/series"
	"cadent/server/series"
	"cadent/server/utils"
	"cadent/server/utils/options"
	"cadent/server/writers/indexer"
	"github.com/opentracing/opentracing-go"
	"golang.org/x/net/context"
	logging "gopkg.in/op/go-logging.v1"
	"math"
	"time"
)

var log = logging.MustGetLogger("metrics")

type WritersNeeded int

const (
	AllResolutions  WritersNeeded = iota // all of them
	FirstResolution                      // just one
)

type CacheTypeNeeded int

const (
	Chunked CacheTypeNeeded = iota // time based log chunks
	Single                         // single byte sized based
)

// Writer interface ..
type Metrics interface {
	// SetTracer set the tracer object, will be used in API/get calls
	SetTracer(t opentracing.Tracer)

	// GetSpan start a tracer named span from context
	GetSpan(name string, ctx context.Context) (opentracing.Span, func())

	// the name of the driver
	Driver() string

	// Name of the writer, for the registry
	Name() string

	// SetResolutions need to able to set what our resolutions are for ease of resolution picking
	// the INT return tells the agg loop if we need to have MULTI writers
	// i.e. for items that DO NOT self rollup (DBs) we need as many writers as resolutions
	// for Whisper (or 'other' things) we only need the Lowest time for the writers
	// as the whisper file rolls up internally
	SetResolutions([][]int) int

	// GetResolutions get the resolutions
	GetResolutions() [][]int

	// SetCurrentResolution we can have a writer per resolution, so this just sets the one we are currently on
	SetCurrentResolution(int)

	// IsPrimaryWriter this writer is responsible for also sending things to the indexer
	// if there are multiple writers, we pick one to be the index sender
	IsPrimaryWriter() bool

	// ShouldWrite should this writer write anything
	ShouldWrite() bool

	// ShouldWriteIndex is true only for the lowest resolution writer
	ShouldWriteIndex() bool

	// Cache the main cache object
	Cache() Cacher

	// CachePrefix prefix of the name in the registry of cache objects
	CachePrefix() string

	// Config configure the writer
	Config(*options.Options) error

	// need an Indexer 99% of the time to deal with render
	SetIndexer(indexer.Indexer) error

	// Write add a metric to the writer (the writer can do many things with this)
	Write(*repr.StatRepr) error

	// WriteWithOffset same as Write expect add a kafka offset marker as well
	WriteWithOffset(*repr.StatRepr, *metrics.OffsetInSeries) error

	RawRender(ctx context.Context, path string, from int64, to int64, tags repr.SortingTags, resample uint32) ([]*metrics.RawRenderItem, error)

	// just get data in the write-back caches
	CacheRender(ctx context.Context, path string, from int64, to int64, tags repr.SortingTags) ([]*metrics.RawRenderItem, error)

	// InCache is the target in this nodes cache?
	InCache(path string, tags repr.SortingTags) (bool, error)

	// CacheList list of all the names in the cache
	CacheList() ([]*repr.StatName, error)

	// return the cached data as the raw binary series
	// note for now only ONE metric can be returned using this method
	CachedSeries(path string, from int64, to int64, tags repr.SortingTags) (*metrics.TotalTimeSeries, error)

	// TurnOnWriteNotify initialize the notify write channel .. note not every writer has this
	TurnOnWriteNotify() <-chan *sseries.MetricWritten

	// OnWriteNotify the channel with a series.MetricWritten message will only be active if
	// the config options `return_write_notifications` is true
	OnWriteNotify() <-chan *sseries.MetricWritten

	// Stop stop the writer and cleanup
	Stop()

	// Start start the writer and associated parts, this needs to be called before anything will happen
	Start()
}

// for those writers that are "blob" writers, we need them to match this interface
// so that we can do resolution rollups
type DBMetrics interface {

	// Driver the name of the driver
	Driver() string

	// GetLatestFromDB gets the latest point(s) written
	GetLatestFromDB(name *repr.StatName, resolution uint32) (metrics.DBSeriesList, error)

	// GetRangeFromDB get the series that fit in a window
	GetRangeFromDB(name *repr.StatName, start uint32, end uint32, resolution uint32) (metrics.DBSeriesList, error)

	// UpdateDBSeries update a db row
	UpdateDBSeries(dbs *metrics.DBSeries, ts series.TimeSeries) error

	// InsertDBSeries add a new row
	InsertDBSeries(name *repr.StatName, timeseries series.TimeSeries, resolution uint32) (int, error)
}

/**** a "common" object we need prepped for almost every "query" we do ****/
type commonQueryGet struct {
	metric        *sindexer.MetricFindItem
	rawd          *smetrics.RawRenderItem
	realStart     uint32
	realEnd       uint32
	uStart        uint32
	uEnd          uint32
	start         int64
	end           int64
	dbResolution  uint32
	outResolution uint32
	doResample    bool
}

// WriterBase is the "parent" object for all writers
type WriterBase struct {
	TracerMetrics

	indexer           indexer.Indexer
	resolutions       [][]int
	currentResolution int
	staticTags        *repr.SortingTags
	options           *options.Options // incoming opts
	name              string           // assigned name

	// this is for Render where we may have several caches, but only "one"
	// cacher get picked for the default render (things share the cache from writers
	// and the api render, but not all the caches, so we need to be able to get the
	// the cache singleton keys
	// `cache:series:seriesMaxMetrics:seriesEncoding:seriesMaxBytes:maxTimeInCache`
	cacherPrefix string
	cacher       Cacher
	isPrimary    bool // is this the primary writer to the cache?
	shouldWrite  bool // default true, if false, the writer will simply act as a cache and do no DB writing

	renderTimeout time.Duration // a timeout on queries

	shutitdown bool
	startstop  utils.StartStop

	writeNotifySubscribe bool
	writeNotify          chan *sseries.MetricWritten
}

// Name the name assigned to this writer
func (wc *WriterBase) Name() string {
	return wc.name
}

// SetResolutions should be of the form
// [BinTime, TTL]
// we select the BinTime based on the TTL
func (wb *WriterBase) SetResolutions(res [][]int) int {
	wb.resolutions = res
	return len(res) // need as many writers as bins
}

// GetResolutions return the [ [BinTime, TTL] ... ] items
func (wb *WriterBase) GetResolutions() [][]int {
	return wb.resolutions
}

// MinResolution the smallest resolution in the set
func (wb *WriterBase) MinResolution() int {
	return wb.resolutions[0][0]
}

// SetCurrentResolution set the current resolution a writer is treating
func (wb *WriterBase) SetCurrentResolution(res int) {
	wb.currentResolution = res
}

// SetIndexer sets the indexer for the metrics writer, all index writes pass through the metrics writer first
func (wb *WriterBase) SetIndexer(idx indexer.Indexer) error {
	wb.indexer = idx
	return nil
}

// IsPrimaryWriter writers that use triggered rollups use the same cache backend, but we only want
// one writer acctually writing, which will then trigger the sub resolutions to get written
func (wc *WriterBase) IsPrimaryWriter() bool {
	return wc.isPrimary
}

// ShouldWrite if false, the actual writing of metrics will not occur, but the cache will
// be used only, good for "cache slave/replicas" this can ONLY be false if
// the Writer object is using the Log-Series based caches, otherwise caches will eat all ram.
// This option is useful for replication, where we want to have a hot standby
// that has all it's read caches full, and still maintains its log state
// (so yes the Log items will still get written, and purged accordingly to
// flushes, just no series or rollups occur)
func (wc *WriterBase) ShouldWrite() bool {

	_, ok := wc.Cache().(*CacherChunk)
	if ok {
		return wc.shouldWrite
	}
	return true
}

// ShouldWriteIndex only write to the indexer if the current writer is the lowest resolution one
func (wc *WriterBase) ShouldWriteIndex() bool {
	return wc.currentResolution == wc.resolutions[0][0] || len(wc.resolutions) == 1
}

// Cache the cache item for the writer
func (wc *WriterBase) Cache() Cacher {
	return wc.cacher
}

// CachePrefix a name for the current cacher to allow easy lookup and for metrics emission
func (wc *WriterBase) CachePrefix() string {
	return wc.cacherPrefix
}

// OnWriteNotify the channel with a series.MetricWritten message will only be active if
// the config options `return_write_notifications` is true
func (wc *WriterBase) OnWriteNotify() <-chan *sseries.MetricWritten {
	return wc.writeNotify
}

// TurnOnWriteNotify will default to nil unless a subclass has this functionality
func (wc *WriterBase) TurnOnWriteNotify() <-chan *sseries.MetricWritten {
	return nil
}

// GetResolution based on the from/to in seconds get the best resolution
// from and to should be SECONDS not nano-seconds
// from and to needs to be > then the TTL as well
func (wc *WriterBase) GetResolution(from int64, to int64) uint32 {
	redux := 3
	diff := int(math.Abs(float64(to-from))) / redux // still get a lower res if w/i 3x the diff
	n := int(time.Now().Unix())
	backF := n - int(from)
	backT := n - int(to)
	for _, res := range wc.resolutions {
		if diff <= res[1] && backF <= res[1] && backT <= res[1] {
			return uint32(res[0])
		}
	}
	return uint32(wc.resolutions[len(wc.resolutions)-1][0])
}

// InCache default is false
func (wc *WriterBase) InCache(path string, tags repr.SortingTags) (bool, error) {
	return false, nil
}

// CacheList list of things in the cache
func (wc *WriterBase) CacheList() ([]*repr.StatName, error) {
	return nil, ErrNotYetimplemented
}

// CacheRender render data only from the cache
func (cass *WriterBase) CacheRender(ctx context.Context, path string, from int64, to int64, tags repr.SortingTags) ([]*metrics.RawRenderItem, error) {
	return nil, ErrNotYetimplemented
}

// CachedSeries get the time series from the cache
func (cass *WriterBase) CachedSeries(path string, from int64, to int64, tags repr.SortingTags) (*metrics.TotalTimeSeries, error) {
	return nil, ErrNotYetimplemented
}

// a helper to aid with queries
func (cass *WriterBase) newGetPrep(metric *sindexer.MetricFindItem, start int64, end int64, resample uint32) *commonQueryGet {
	rawd := new(smetrics.RawRenderItem)

	out := new(commonQueryGet)
	out.metric = metric

	//figure out the best res
	out.dbResolution = cass.GetResolution(start, end)
	out.outResolution = out.dbResolution

	//obey the bigger
	if resample > out.dbResolution {
		out.outResolution = resample
	}

	out.doResample = resample > 0 && resample > out.dbResolution

	start = TruncateTimeTo(start, int(out.dbResolution))
	end = TruncateTimeTo(end, int(out.dbResolution))

	out.uStart = uint32(start)
	out.uEnd = uint32(end)

	statName := metric.StatName()

	rawd.Step = out.outResolution
	rawd.Metric = metric.Path
	rawd.Id = metric.UniqueId
	rawd.RealEnd = out.uEnd
	rawd.RealStart = out.uStart
	rawd.Start = rawd.RealStart
	rawd.End = rawd.RealEnd
	rawd.Tags = metric.Tags
	rawd.MetaTags = metric.MetaTags
	rawd.AggFunc = statName.AggType()
	out.rawd = rawd
	return out
}
