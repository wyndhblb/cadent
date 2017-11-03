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
	THe Kafka Metrics "blob" metrics writer

	this only emits on the "overflow" from the cacher and a big binary blobl of data


*/

package metrics

import (
	"cadent/server/broadcast"
	"cadent/server/schemas"
	"cadent/server/schemas/metrics"
	"cadent/server/schemas/repr"
	"cadent/server/schemas/series"
	tseries "cadent/server/series"
	"cadent/server/stats"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/dbs"
	"fmt"
	"github.com/Shopify/sarama"
	"golang.org/x/net/context"
	logging "gopkg.in/op/go-logging.v1"
	"sync"
	"time"
)

/****************** Interfaces *********************/
type KafkaMetrics struct {
	CacheBase

	db   *dbs.KafkaDB
	conn sarama.AsyncProducer

	batches int // number of stats to "batch" per message (default 0)
	log     *logging.Logger

	resolution uint32

	enctype series.SendEncoding

	cacheOverFlow *broadcast.Listener // on byte overflow of cacher force a write

}

func NewKafkaMetrics() *KafkaMetrics {
	kf := new(KafkaMetrics)
	kf.batches = 0
	kf.log = logging.MustGetLogger("writers.kafka.metrics")

	kf.shutitdown = false
	kf.isPrimary = false
	return kf
}

func (kf *KafkaMetrics) Config(conf *options.Options) error {
	kf.options = conf
	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("`dsn` (kafkahost1,kafkahost2) is needed for kafka config")
	}

	db, err := dbs.NewDB("kafka", dsn, conf)
	if err != nil {
		return err
	}

	resolution, err := conf.Float64Required("resolution")
	if err != nil {
		return fmt.Errorf("Resolution needed for kafka blob writer")
	} else {
		kf.resolution = uint32(resolution)
	}

	kf.db = db.(*dbs.KafkaDB)
	kf.conn = db.Connection().(sarama.AsyncProducer)

	g_tag := conf.String("tags", "")
	if len(g_tag) > 0 {
		kf.staticTags = repr.SortingTagsFromString(g_tag)
	}

	enct := conf.String("encoding", "json")
	if len(enct) > 0 {
		kf.enctype = series.SendEncodingFromString(enct)
	}

	_cache, _ := conf.ObjectRequired("cache")
	if _cache == nil {
		return errMetricsCacheRequired
	}
	kf.cacher = _cache.(Cacher)

	kf.cacherPrefix = kf.cacher.GetPrefix()

	// for the overflow cached items::
	// these caches can be shared for a given writer set, and the caches may provide the data for
	// multiple writers, we need to specify that ONE of the writers is the "main" one otherwise
	// the Metrics Write function will add the points over again, which is not a good thing
	// when the accumulator flushes things to the multi writers
	// The Writer needs to know it's "not" the primary writer and thus will not "add" points to the
	// cache .. so the cache basically gets "one" primary writer pointed (first come first serve)
	kf.isPrimary = kf.cacher.SetPrimaryWriter(kf)
	if kf.isPrimary {
		kf.log.Notice("Kafka series writer is the primary writer to write back cache %s", kf.cacher.GetName())
	}
	return nil
}

func (kf *KafkaMetrics) Driver() string {
	return "kafka"
}

func (kf *KafkaMetrics) Start() {
	kf.startstop.Start(func() {
		kf.log.Notice("Starting Kafka writer for %s at %d bytes per series", kf.db.DataTopic(), kf.cacher.GetMaxBytesPerMetric())
		kf.cacher.SetOverFlowMethod("chan") // force chan

		kf.cacher.Start()
		// only register this if we are really going to consume it
		kf.cacheOverFlow = kf.cacher.GetOverFlowChan()
		go kf.overFlowWrite()
		go kf.onError()
		go kf.onSuccess()
	})
}

func (kf *KafkaMetrics) Stop() {
	kf.log.Warning("Stopping Kafka writer for (%s)", kf.cacher.GetName())
	kf.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()

		if kf.shutitdown {
			return // already did
		}
		kf.shutitdown = true
		kf.cacher.Stop()

		// need to cast this
		scache := kf.cacher.(*CacherSingle)

		mets := scache.Cache
		mets_l := len(mets)
		kf.log.Warning("Shutting down %s and exhausting the queue (%d items) and quiting", kf.cacher.GetName(), mets_l)

		// full tilt write out
		procs := 16
		go_do := make(chan metrics.TotalTimeSeries, procs)
		wg := sync.WaitGroup{}

		goInsert := func() {
			for {
				select {
				case s, more := <-go_do:
					if !more {
						return
					}
					stats.StatsdClient.Incr(fmt.Sprintf("writer.cache.shutdown.send-to-writers"), 1)
					kf.PushSeries(s.Name, s.Series)
					wg.Done()
				}
			}
		}
		for i := 0; i < procs; i++ {
			go goInsert()
		}

		did := 0
		for _, queueitem := range mets {
			wg.Add(1)
			if did%100 == 0 {
				kf.log.Warning("shutdown purge: written %d/%d...", did, mets_l)
			}
			if queueitem.Series != nil {
				go_do <- metrics.TotalTimeSeries{Name: queueitem.Name, Series: queueitem.Series.Copy()}
			}
			did++
		}
		wg.Wait()
		close(go_do)
		kf.log.Warning("shutdown purge: written %d/%d...", did, mets_l)

		kf.log.Warning("Shutdown finished ... quiting kafka writer")
		return
	})
}

func (kf *KafkaMetrics) onError() {

	for {
		err, more := <-kf.conn.Errors()
		if !more {
			return
		}
		stats.StatsdClientSlow.Incr("writer.kafka.metrics.writes.error", 1)
		kf.log.Errorf("%s", err)
	}
}

func (kf *KafkaMetrics) onSuccess() {

	for {
		_, more := <-kf.conn.Successes()
		if !more {
			return
		}
		stats.StatsdClientSlow.Incr("writer.kafka.metrics.writes.success", 1)
	}
}

// listen to the overflow chan from the cache and attempt to write "now"
func (kf *KafkaMetrics) overFlowWrite() {
	defer func() {
		if r := recover(); r != nil {
			kf.log.Critical("Kafka Failure (panic) %v", r)
		}
	}()

	for {
		statitem, more := <-kf.cacheOverFlow.Ch
		if !more {
			return
		}
		if statitem != nil {
			ts := statitem.(*metrics.TotalTimeSeries)
			kf.PushSeries(ts.Name, ts.Series)
		} else {
			kf.log.Errorf("%s", schemas.ErrMetricIsNil)
		}
	}
}
func (kf *KafkaMetrics) Write(stat *repr.StatRepr) error {
	return kf.WriteWithOffset(stat, nil)
}

func (kf *KafkaMetrics) WriteWithOffset(stat *repr.StatRepr, offset *metrics.OffsetInSeries) error {
	// merge the tags in
	stat.Name.MergeMetric2Tags(kf.staticTags)
	kf.indexer.Write(*stat.Name) // to the indexer
	// not primary writer .. move along
	if !kf.isPrimary {
		return nil
	}
	kf.cacher.AddWithOffset(stat.Name, stat, offset)
	return nil
}

func (kf *KafkaMetrics) PushSeries(name *repr.StatName, points tseries.TimeSeries) error {
	if name == nil {
		return errNameIsNil
	}
	if points == nil {
		return errSeriesIsNil
	}

	pts, err := points.MarshalBinary()
	if err != nil {
		return err
	}

	sMetric := new(series.SeriesMetric)
	sMetric.Metric = name.Key
	sMetric.Time = time.Now().UnixNano()
	sMetric.Data = pts
	sMetric.Encoding = points.Name()
	sMetric.Resolution = name.Resolution
	sMetric.Ttl = name.Ttl
	sMetric.Tags = name.SortedTags().Tags()
	sMetric.MetaTags = name.SortedMetaTags().Tags()

	obj := &series.KMetric{
		AnyMetric: series.AnyMetric{
			Series: sMetric,
		},
	}
	obj.SetSendEncoding(kf.enctype)

	kf.conn.Input() <- &sarama.ProducerMessage{
		Topic: kf.db.DataTopic(),
		Key:   sarama.StringEncoder(name.UniqueIdString()),
		Value: obj,
	}

	stats.StatsdClientSlow.Incr("writer.kafka.metrics.writes", 1)
	return nil
}

/**** READER ***/

// needed to match interface, but we obviously cannot do this

func (kf *KafkaMetrics) RawRender(ctx context.Context, path string, from int64, to int64, tags repr.SortingTags, resample uint32) ([]*metrics.RawRenderItem, error) {
	return []*metrics.RawRenderItem{}, ErrWillNotBeimplemented
}
