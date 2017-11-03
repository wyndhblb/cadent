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
	THe Kafka Index writer



*/

package indexer

import (
	"cadent/server/schemas/indexer"
	"cadent/server/schemas/repr"
	"cadent/server/stats"
	"cadent/server/utils"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/dbs"
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
	"golang.org/x/net/context"
	logging "gopkg.in/op/go-logging.v1"
	"strings"
	"time"
)

/** basic data type **/

type KafkaPath struct {
	Id       repr.StatId      `json:"id"`
	Uid      string           `json:"uid"`
	Type     string           `json:"type"`
	Path     string           `json:"path"`
	Segments []string         `json:"segments"`
	SentTime int64            `json:"senttime"`
	Tags     repr.SortingTags `json:"tags,omitempty"`
	MetaTags repr.SortingTags `json:"meta_tags,omitempty"`

	encoded []byte
	err     error
}

func (kp *KafkaPath) ensureEncoded() {
	if kp.encoded == nil && kp.err == nil {
		kp.encoded, kp.err = json.Marshal(kp)
	}
}

func (kp *KafkaPath) Length() int {
	kp.ensureEncoded()
	return len(kp.encoded)
}

func (kp *KafkaPath) Encode() ([]byte, error) {
	kp.ensureEncoded()
	return kp.encoded, kp.err
}

/****************** Interfaces *********************/
type KafkaIndexer struct {
	TracerIndexer

	db         *dbs.KafkaDB
	conn       sarama.AsyncProducer
	writeIndex bool // if false, we skip the index writing message as well, the stat metric itself has the key in it
	shutitdown bool
	indexerId  string
	log        *logging.Logger
	startstop  utils.StartStop
}

func NewKafkaIndexer() *KafkaIndexer {
	kf := new(KafkaIndexer)
	kf.log = logging.MustGetLogger("writers.indexer.kafka")
	kf.writeIndex = true
	kf.shutitdown = false
	kf.indexerId = fmt.Sprintf("kafak:indexer")
	return kf
}

func (kf *KafkaIndexer) Config(conf *options.Options) (err error) {
	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("`dsn` (kafkahost1,kafkahost2) is needed for kafka config")
	}

	kf.indexerId = conf.String("name", "indexer:kafka:"+dsn)

	// reg ourselves before try to get conns
	kf.log.Noticef("Registering indexer: %s", kf.Name())
	err = RegisterIndexer(kf.Name(), kf)
	if err != nil {
		return err
	}

	db, err := dbs.NewDB("kafka", dsn, conf)
	if err != nil {
		return err
	}

	kf.writeIndex = conf.Bool("write_index", true)
	kf.db = db.(*dbs.KafkaDB)
	kf.conn = db.Connection().(sarama.AsyncProducer)

	return nil
}

func (kf *KafkaIndexer) Start() {
	kf.startstop.Start(func() {
		kf.log.Notice("starting kafka indexer: %s", kf.Name())
	})
}

func (kf *KafkaIndexer) Stop() {
	kf.startstop.Stop(func() {
		kf.log.Notice("shutting down cassandra indexer: %s", kf.Name())

		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()
		kf.shutitdown = true
		time.Sleep(time.Second) // wait for any lingering writes
		if err := kf.conn.Close(); err != nil {
			kf.log.Error("Failed to shut down producer cleanly %v", err)
		}
	})
}

// ShouldWrite always true
func (kf *KafkaIndexer) ShouldWrite() bool {
	return true
}

func (kf *KafkaIndexer) Name() string {
	return kf.indexerId
}

func (kf *KafkaIndexer) Write(skey repr.StatName) error {
	// noop if not writing indexes
	if !kf.writeIndex || kf.shutitdown {
		return nil
	}

	item := &KafkaPath{
		Type:     "index",
		Id:       skey.UniqueId(),
		Uid:      skey.UniqueIdString(),
		Path:     skey.Key,
		Segments: strings.Split(skey.Key, "."),
		Tags:     skey.SortedTags().Tags(),
		MetaTags: skey.SortedMetaTags().Tags(),
		SentTime: time.Now().UnixNano(),
	}

	stats.StatsdClientSlow.Incr("writer.kafka.indexer.writes", 1)

	kf.conn.Input() <- &sarama.ProducerMessage{
		Topic: kf.db.IndexTopic(),
		Key:   sarama.StringEncoder(skey.Key), // hash on metric key
		Value: item,
	}

	return nil
}

// send a "delete message" to the mix
func (kf *KafkaIndexer) Delete(skey *repr.StatName) error {
	// noop if not writing indexes
	if !kf.writeIndex {
		return nil
	}

	item := &KafkaPath{
		Type:     "delete",
		Id:       skey.UniqueId(),
		Path:     skey.Key,
		Segments: strings.Split(skey.Key, "."),
		Tags:     skey.SortedTags().Tags(),
		MetaTags: skey.SortedMetaTags().Tags(),
		SentTime: time.Now().UnixNano(),
	}

	stats.StatsdClientSlow.Incr("writer.kafka.indexer.delete", 1)

	kf.conn.Input() <- &sarama.ProducerMessage{
		Topic: kf.db.IndexTopic(),
		Key:   sarama.StringEncoder(skey.Key), // hash on metric key
		Value: item,
	}
	return nil
}

/**** READER ***/
// just to match the interface, as there's no way to do this really
func (kf *KafkaIndexer) List(has_data bool, page int) (indexer.MetricFindItems, error) {
	return indexer.MetricFindItems{}, ErrWillNotBeimplemented
}
func (kf *KafkaIndexer) Find(ctx context.Context, metric string, tags repr.SortingTags) (indexer.MetricFindItems, error) {
	return indexer.MetricFindItems{}, ErrWillNotBeimplemented
}

func (kf *KafkaIndexer) FindInCache(ctx context.Context, metric string, tags repr.SortingTags) (indexer.MetricFindItems, error) {
	return indexer.MetricFindItems{}, ErrWillNotBeimplemented
}

func (kf *KafkaIndexer) Expand(metric string) (indexer.MetricExpandItem, error) {
	return indexer.MetricExpandItem{}, ErrWillNotBeimplemented
}

func (my *KafkaIndexer) GetTagsByUid(unique_id string) (tags repr.SortingTags, metatags repr.SortingTags, err error) {
	return tags, metatags, ErrWillNotBeimplemented
}

func (my *KafkaIndexer) GetTagsByName(name string, page int) (tags indexer.MetricTagItems, err error) {
	return tags, ErrWillNotBeimplemented
}

func (my *KafkaIndexer) GetTagsByNameValue(name string, value string, page int) (tags indexer.MetricTagItems, err error) {
	return tags, ErrWillNotBeimplemented
}

func (my *KafkaIndexer) GetUidsByTags(key string, tags repr.SortingTags, page int) (uids []string, err error) {
	return uids, ErrWillNotBeimplemented
}
