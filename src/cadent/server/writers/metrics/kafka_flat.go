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
	The Kafka Flat Metrics writer
*/

package metrics

import (
	"cadent/server/schemas/metrics"
	"cadent/server/schemas/repr"
	"cadent/server/schemas/series"
	"cadent/server/stats"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/dbs"
	"fmt"
	"github.com/Shopify/sarama"
	"golang.org/x/net/context"
	logging "gopkg.in/op/go-logging.v1"
	"time"
)

/****************** Interfaces *********************/
type KafkaFlatMetrics struct {
	WriterBase

	db   *dbs.KafkaDB
	conn sarama.AsyncProducer

	enctype series.SendEncoding
	batches int // number of stats to "batch" per message (default 0)
	log     *logging.Logger
}

func NewKafkaFlatMetrics() *KafkaFlatMetrics {
	kf := new(KafkaFlatMetrics)
	kf.batches = 0
	kf.shutitdown = false
	kf.log = logging.MustGetLogger("writers.kafkaflat.metrics")
	return kf
}

func (kf *KafkaFlatMetrics) Config(conf *options.Options) error {
	kf.options = conf
	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("`dsn` (kafkahost1,kafkahost2) is needed for kafka config")
	}

	db, err := dbs.NewDB("kafka", dsn, conf)
	if err != nil {
		return err
	}

	g_tag := conf.String("tags", "")
	if len(g_tag) > 0 {
		kf.staticTags = repr.SortingTagsFromString(g_tag)
	}

	enct := conf.String("encoding", "json")
	if len(enct) > 0 {
		kf.enctype = series.SendEncodingFromString(enct)
	}

	kf.db = db.(*dbs.KafkaDB)
	kf.conn = db.Connection().(sarama.AsyncProducer)

	return nil
}

func (kf *KafkaFlatMetrics) Driver() string {
	return "kafka-flat"
}

func (kf *KafkaFlatMetrics) Start() {
	//noop
}

func (kf *KafkaFlatMetrics) Stop() {
	kf.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()
		kf.shutitdown = true
		time.Sleep(time.Second) // wait to get things written if pending
		if err := kf.conn.Close(); err != nil {
			kf.log.Error("Failed to shut down producer cleanly %v", err)
		}
	})
}

func (kf *KafkaFlatMetrics) Write(stat *repr.StatRepr) error {
	return kf.WriteWithOffset(stat, nil)
}

func (kf *KafkaFlatMetrics) WriteWithOffset(stat *repr.StatRepr, offset *metrics.OffsetInSeries) error {

	if kf.shutitdown {
		return nil
	}

	stat.Name.MergeMetric2Tags(kf.staticTags)
	kf.indexer.Write(*stat.Name) // to the indexer
	item := &series.KMetric{
		AnyMetric: series.AnyMetric{
			Single: &series.SingleMetric{
				Metric:     stat.Name.Key,
				Time:       stat.Time,
				SentTime:   time.Now().UnixNano(),
				Sum:        float64(stat.Sum),
				Last:       float64(stat.Last),
				Count:      stat.Count,
				Max:        float64(stat.Max),
				Min:        float64(stat.Min),
				Resolution: stat.Name.Resolution,
				Ttl:        stat.Name.Ttl,
				Id:         uint64(stat.Name.UniqueId()),
				Uid:        stat.Name.UniqueIdString(),
				Tags:       stat.Name.SortedTags().Tags(),
				MetaTags:   stat.Name.SortedMetaTags().Tags(),
			},
		},
	}
	//fmt.Println("OUTSTAT: name:", item.AnyMetric.Single.Metric, "time: ", item.AnyMetric.Single.Time, "count:", item.AnyMetric.Single.Count, "sum:", item.AnyMetric.Single.Sum)
	item.SetSendEncoding(kf.enctype)

	stats.StatsdClientSlow.Incr("writer.kafkaflat.metrics.writes", 1)

	kf.conn.Input() <- &sarama.ProducerMessage{
		Topic:     kf.db.DataTopic(),
		Key:       sarama.StringEncoder(stat.Name.UniqueIdString()), // hash on unique id
		Value:     item,
		Timestamp: time.Now(),
	}
	return nil
}

/**** READER ***/
// needed to match interface, but we obviously cannot do this

func (kf *KafkaFlatMetrics) RawRender(context.Context, string, int64, int64, repr.SortingTags, uint32) ([]*metrics.RawRenderItem, error) {
	return []*metrics.RawRenderItem{}, ErrWillNotBeimplemented
}
func (kf *KafkaFlatMetrics) CacheRender(ctx context.Context, path string, from int64, to int64, tags repr.SortingTags) ([]*metrics.RawRenderItem, error) {
	return nil, ErrWillNotBeimplemented
}
func (kf *KafkaFlatMetrics) CachedSeries(path string, from int64, to int64, tags repr.SortingTags) (*metrics.TotalTimeSeries, error) {
	return nil, ErrWillNotBeimplemented
}
