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
   Kafka injector

   A Kafka consumer for "writing" metrics

   On a given topic we consume the message and send it to the writers.
   Rather then mark any offsets, we emit another message on the same topic that indicates a series was
   written.  Such that on startup/restart/etc, we start consuming, backfill all the series that are in ram,
   if we get a message that is that series was written, we simply mark that cached objected as done, and start over
   in this way we can recover from an "outage" of sorts, depending on the topics time span this startup process
   can take some time.
*/

package injectors

import (
	"cadent/server/schemas/series"
	"cadent/server/stats"
	"cadent/server/utils"
	"cadent/server/utils/options"
	"cadent/server/utils/shutdown"
	"cadent/server/writers/dbs"
	"errors"
	"fmt"
	"github.com/Shopify/sarama"
	"gopkg.in/op/go-logging.v1"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// on a restart, we run through the topic starting back this far (roughly)
	// this should be 2x the "max time in cache" for the series writers so that we catch all the
	// missing bits
	DEFAULT_KAFKA_MAX_TIME_BACK = 2 * time.Duration(time.Hour)
	DEFAULT_KAFKA_ENCODING      = "msgpack"
)

var ErrorBadOffsetName = errors.New("starting_offset should be `oldest` or `newest`")
var ErrorBadMessageType = errors.New("Message type not supported by this kafka worker")
var ErrorWriterNotDefined = errors.New("The writer is not set.")

// Kafka the main consumer of goodies
type Kafka struct {
	InjectorBase

	Name string // name of the injector

	Topic             string              // topic to consume
	CommitTopic       string              // topic to commit offsets to
	ConsumerGroup     string              // consumer group name (default to "cadent-" + Name)
	Brokers           string              // List of brokers
	StartOffset       string              // Starting offset for new consumers (default = newest)
	EncodingType      series.SendEncoding // msgpack, json, protobuf (default = msgpack)
	MessageType       series.MessageType  // "unprocessed", "raw", "single", "any" (default = any)
	KafkaWorker       Worker
	ConsumePartitions string // a list of numbers comma separated 1,2,4,5...
	OffsetDbPath      string // path to the local level DB index for offset marks

	ClusterConfig     *sarama.Config
	consumer          sarama.Consumer
	client            sarama.Client
	partitions        map[string][]int32
	consumePartitions map[string][]int32

	restartBackSlurpTime time.Duration // on a restart (if kafka 10.1) try to go back this much time in the parition rather then the entire range

	log       *logging.Logger
	startstop utils.StartStop

	IsReady bool

	// if this is false, we don't explicitly mark offsets
	// instead use the KafkaMarkWritten object to mark things as "written" w/ a message instead
	// false is the default
	MarkOffset  bool
	consumeStop chan bool
	doneWg      sync.WaitGroup

	markDoneChan     chan *sarama.ConsumerMessage
	markWriteMessage *KafkaMarkWritten
	kafkaPartitions  kafkaPartitionConsumers
	localMark        *localKafkaOffset
}

// New a new kafka consumer object
func NewKafka(name string) *Kafka {
	kf := new(Kafka)

	kf.Name = name
	kf.MarkOffset = false
	kf.partitions = make(map[string][]int32, 0)
	kf.consumePartitions = make(map[string][]int32, 0)
	kf.markWriteMessage = NewMarkWritten()
	kf.consumeStop = make(chan bool, 1)
	kf.markDoneChan = make(chan *sarama.ConsumerMessage, 128)
	kf.restartBackSlurpTime = DEFAULT_KAFKA_MAX_TIME_BACK

	kf.log = logging.MustGetLogger("kafka.injestor")

	return kf
}

// Config set topics, consumer groups, offsets, etc
func (kf *Kafka) Config(conf options.Options) (err error) {
	kf.Topic = conf.String("topic", "cadent")
	kf.CommitTopic = conf.String("commit_topic", "cadent-commit")
	kf.ConsumerGroup = conf.String("consumer_group", "cadent-"+kf.Name)
	kf.MarkOffset = conf.Bool("mark_offsets", false)

	kf.EncodingType = series.SendEncodingFromString(conf.String("encoding", DEFAULT_KAFKA_ENCODING))
	kf.MessageType = series.MetricTypeFromString(conf.String("message_type", "any"))
	kf.StartOffset = conf.String("starting_offset", "time")
	kf.ConsumePartitions = conf.String("consume_partitions", "all")
	kf.Brokers, err = conf.StringRequired("dsn")

	kf.OffsetDbPath = conf.String("local_offset_db", "")

	if err != nil {
		return err
	}

	//worker based on message type of course
	switch kf.MessageType {
	case series.MSG_RAW:
		kf.KafkaWorker = new(RawMetricWorker)
	case series.MSG_UNPROCESSED:
		kf.KafkaWorker = new(UnProcessedMetricWorker)
	case series.MSG_SINGLE:
		kf.KafkaWorker = new(SingleMetricWorker)
	case series.MSG_ANY:
		kf.KafkaWorker = new(AnyMetricWorker)
	default:
		kf.log.Errorf("Invalid message type %v", kf.MessageType)
		return ErrorBadMessageType
	}

	kf.ClusterConfig = sarama.NewConfig()

	kf.ClusterConfig.ClientID = kf.ConsumerGroup
	kf.ClusterConfig.Net.MaxOpenRequests = 16
	kf.ClusterConfig.Net.KeepAlive = time.Second * time.Duration(10)
	kf.ClusterConfig.ChannelBufferSize = int(conf.Int64("channel_buffer_size", 4096*100))
	kf.ClusterConfig.Consumer.Return.Errors = true
	kf.ClusterConfig.Consumer.Fetch.Min = int32(conf.Int64("consumer_fetch_min", 1))
	kf.ClusterConfig.Consumer.Fetch.Default = int32(conf.Int64("consumer_fetch_default", 4096*1000))
	kf.ClusterConfig.Consumer.MaxWaitTime = conf.Duration("consumer_max_wait_time", time.Second*time.Duration(1))
	kf.ClusterConfig.Consumer.MaxProcessingTime = conf.Duration("consumer_max_processing_time", time.Second*time.Duration(1))
	kf.ClusterConfig.Net.MaxOpenRequests = int(conf.Int64("max_open_requests", 128))
	kf.ClusterConfig.Version = sarama.V0_10_0_0

	kf.restartBackSlurpTime = conf.Duration("restart_back_time", DEFAULT_KAFKA_MAX_TIME_BACK)

	switch kf.StartOffset {
	case "oldest", "reset":
		kf.ClusterConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	case "newest":
		kf.ClusterConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	case "time":
		break
	default:
		kf.log.Errorf("Invalid start offset %s", kf.StartOffset)
		return ErrorBadOffsetName
	}

	err = kf.ClusterConfig.Validate()
	if err != nil {
		kf.log.Errorf("Invalid kafka config %v", err)
		return err
	}

	kf.client, err = sarama.NewClient(strings.Split(kf.Brokers, ","), kf.ClusterConfig)
	if err != nil {
		kf.log.Errorf("kafka config failed: %v", err)
		return err
	}

	kf.consumer, err = sarama.NewConsumerFromClient(kf.client)
	if err != nil {
		return err
	}

	err = kf.markWriteMessage.Config(conf)

	if err != nil {
		kf.log.Errorf("kafka config failed: %v", err)
		return err
	}
	kf.markWriteMessage.mainConsumer = kf

	//local offset manager
	kf.localMark = NewLocalOffset(kf)

	return nil
}

// get the topic -> partitions available mappings
func (kf *Kafka) getPartitions() (map[string][]int32, error) {
	partitionCount := 0
	var partitions []int32
	topics := strings.Split(kf.Topic, ",")
	out := make(map[string][]int32)

	var err error
	kf.log.Debugf("Getting partitions for topics: %s", topics)

	for i, topic := range topics {
		kf.log.Debugf("Getting partitions for topic: %s", topic)
		out[topic] = make([]int32, 0)

		partitions, err = kf.client.Partitions(topic)
		if err != nil {
			return nil, fmt.Errorf("Could not get partitions for topic %s: %s", topic, err)
		}
		if len(partitions) == 0 {
			return nil, fmt.Errorf("No partitions for topic %s", topic)
		}
		kf.log.Debugf("Found partitions: %v for topic: %s", partitions, topic)

		// if consuming multiple topics, since there's just one partition "assignment" the
		// number of partitions need to match each other across topics
		if i > 0 {
			if len(partitions) != partitionCount {
				return nil, fmt.Errorf("Configured topics have different partition counts, this is not supported")
			}
			continue
		}
		partitionCount = len(partitions)
		out[topic] = partitions
	}
	return out, nil
}

// check the commit topic is of the same partition count (call this after getPartitions)
func (kf *Kafka) getCommitPartitions() ([]int32, error) {
	var partitions []int32

	var err error
	kf.log.Debugf("Checking commit topic: Getting partitions for topics: %s", kf.CommitTopic)

	partitions, err = kf.client.Partitions(kf.CommitTopic)
	if err != nil {
		return nil, fmt.Errorf("Could not get partitions for topic %s: %s", kf.CommitTopic, err)
	}
	if len(partitions) == 0 {
		return nil, fmt.Errorf("No partitions for topic %s", kf.CommitTopic)
	}
	kf.log.Debugf("Found partitions: %v for topic: %s", partitions, kf.CommitTopic)

	return partitions, nil

}

func (kf *Kafka) setPartitions() error {

	havePartitions, err := kf.getPartitions()
	if err != nil {
		kf.log.Errorf("Getting kafka paritions failed: %v", err)
		return err
	}
	kf.log.Noticef("We have these kafka paritions available: %v", havePartitions)

	commitParts, err := kf.getCommitPartitions()
	if err != nil {
		kf.log.Errorf("Getting kafka commit paritions failed: %v", err)
		return err
	}

	// check the commit topic have the same number of partitions
	for _, parts := range havePartitions {
		if len(parts) != len(commitParts) {
			err = fmt.Errorf("Commit topic paritions and consuming topics must have the same number of partitions (have: %d want %d)", len(commitParts), len(parts))
			kf.log.Errorf("Getting kafka commit paritions failed: %v", err)
			return err
		}
	}

	subParts := []int32{}
	if kf.ConsumePartitions == "*" || kf.ConsumePartitions == "all" {
		kf.consumePartitions = havePartitions
	} else {
		parts := strings.Split(kf.ConsumePartitions, ",")
		for _, part := range parts {
			i, err := strconv.Atoi(part)
			if err != nil {
				kf.log.Errorf(
					"Could not parse partition %v. If forcing parititions, please make it a comma separated int string (1,2,3..)",
					part,
				)
			}
			subParts = append(subParts, int32(i))
		}

		kf.log.Noticef("Seems you wish these %v partitions to consume .. let's see if they exist", subParts)

		inList := func(list []int32, want int32) bool {
			for _, i := range list {
				if want == i {
					return true
				}
			}
			return false
		}

		// make sure the partitions are real that we want to consume
		noHave := []int32{}
		for _, realParts := range havePartitions {
			for _, want := range subParts {
				if !inList(realParts, want) {
					noHave = append(noHave, want)
				}
			}
		}

		if len(noHave) > 0 {
			return fmt.Errorf("Your desired list of partitions is %v, however these don't exist: %v", subParts, noHave)
		}

		topics := strings.Split(kf.Topic, ",")
		// set our main map of topic -> [partitions]
		for _, tpk := range topics {
			kf.consumePartitions[tpk] = subParts
		}
	}
	kf.log.Noticef("We're going to attempt to consume these topic/partitions: %v", kf.consumePartitions)

	return nil
}

// Start fire things up
func (kf *Kafka) Start() error {
	var err error
	kf.startstop.Start(func() {

		if kf.GetWriter() != nil {
			kf.log.Notice("Starting writer for kafka injector: %s", kf.GetWriter().GetName())

			// turn on the notifier
			gotChan := kf.GetWriter().Metrics().TurnOnWriteNotify()
			if gotChan != nil {
				kf.markWriteMessage.markWrittenChan = gotChan
			}

			kf.GetWriter().Start()
			kf.KafkaWorker.SetWriter(kf.GetWriter())
		}
		if kf.GetSubWriter() != nil {
			kf.log.Notice("Starting sub writer for kafka injector: %s", kf.GetSubWriter().GetName())
			kf.GetSubWriter().Start()
			kf.KafkaWorker.SetSubWriter(kf.GetSubWriter())
		}
		if kf.GetReader() != nil {
			kf.log.Notice("Starting api for kafka injector")
			go kf.GetReader().Start()
		}
		if kf.GetTCPReader() != nil {
			kf.log.Notice("Starting tcp api for kafka injector")
			go kf.GetTCPReader().Start()
		}

		err = kf.setPartitions()
		if err != nil {
			kf.log.Errorf("Partition check failed: %v", err)
			return
		}

		if len(kf.consumePartitions) == 0 {
			err = fmt.Errorf("No paritions to consume ...")
			return
		}

		kf.log.Notice("Starting Kafka consumers: %v", kf.consumePartitions)

		// init the localManager
		if kf.localMark != nil {
			err = kf.localMark.Init()
			if err != nil {
				kf.log.Errorf("Failed to init the local offset mark: %v", err)
				return
			}
		}

		for tpk, parts := range kf.consumePartitions {
			curTpk := tpk
			for _, partition := range parts {
				curPart := partition
				consumer := newFromPartitionFromKafka(kf)
				consumer.topic = curTpk
				consumer.partition = curPart
				err = consumer.startConsume()
				if err != nil {
					return
				}
				kf.kafkaPartitions = append(kf.kafkaPartitions, consumer)

				if kf.localMark != nil {
					// start up the local marks for this topic/partition
					go kf.localMark.processMarkWrote(curTpk, curPart)
				} else {
					kf.log.Notice("No Local Mark path defined, skipping local mark DB")
				}
			}
		}

		// fire up the commit marker after the partitions have all gotten up and ticking
		if kf.markWriteMessage.markWrittenChan != nil {
			kf.markWriteMessage.Start()
		}
	})
	return err
}

// Stop shutdown the land
func (kf *Kafka) Stop() error {
	var err error
	kf.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()
		kf.log.Notice("Kafka shutting down")

		err = kf.consumer.Close()
		if err != nil {
			kf.log.Errorf("Kafka stop fail: %s", err)
		}
		close(kf.consumeStop)

		kf.log.Notice("Kafka stopped")

		if kf.GetWriter() != nil {
			kf.GetWriter().Stop()
		}
		if kf.GetSubWriter() != nil {
			kf.GetSubWriter().Stop()
		}
		if kf.GetReader() != nil {
			kf.GetReader().Stop()
		}
		if kf.GetTCPReader() != nil {
			kf.GetTCPReader().Stop()
		}

		if kf.localMark != nil {
			kf.localMark.Stop()
		}
	})
	return err
}

// KafkaMarkWritten the helper that deals with the "metric written" producers
type KafkaMarkWritten struct {
	db           *dbs.KafkaDB
	conn         sarama.AsyncProducer
	mainConsumer *Kafka // the main consumer of things

	markWrittenChan <-chan *series.MetricWritten
	topic           string
	brokers         string
	enctype         series.SendEncoding

	doneWg    sync.WaitGroup
	markStop  chan bool
	log       *logging.Logger
	startstop utils.StartStop
}

// NewMarkWritten a new KafkaMarkWritten object
func NewMarkWritten() *KafkaMarkWritten {
	kf := new(KafkaMarkWritten)
	kf.markStop = make(chan bool)
	kf.log = logging.MustGetLogger("kafka.writemark")

	return kf
}

func (kf *KafkaMarkWritten) Config(conf options.Options) (err error) {
	kf.topic = conf.String("commit_topic", "cadent-commits")
	kf.brokers, err = conf.StringRequired("dsn")
	if err != nil {
		return err
	}
	enct := conf.String("encoding", DEFAULT_KAFKA_ENCODING)
	if len(enct) > 0 {
		kf.enctype = series.SendEncodingFromString(enct)
	}

	db, err := dbs.NewDB("kafka", kf.brokers, &conf)
	if err != nil {
		return err
	}

	kf.db = db.(*dbs.KafkaDB)
	kf.conn = db.Connection().(sarama.AsyncProducer)

	return nil
}

func (kf *KafkaMarkWritten) Start() error {
	var err error
	kf.startstop.Start(func() {
		if kf.markWrittenChan != nil {
			go kf.onMessageWritten()
		} else {
			kf.log.Warning("No Mark written channel set, cannot start the mark written producer")
		}
	})
	return err
}

func (kf *KafkaMarkWritten) Stop() error {
	var err error
	kf.startstop.Stop(func() {
		shutdown.AddToShutdown()
		defer shutdown.ReleaseFromShutdown()
		kf.log.Notice("KafkaMarkWritten shutting down")

		err = kf.conn.Close()
		if err != nil {
			kf.log.Errorf("Kafka stop fail: %s", err)
			return
		}
		if kf.markWrittenChan != nil {
			kf.markStop <- true
		}
		kf.log.Notice("KafkaMarkWritten producer stopped")
	})
	return err
}

func (kf *KafkaMarkWritten) MarkWriteChan() <-chan *series.MetricWritten {
	return kf.markWrittenChan
}

func (kf *KafkaMarkWritten) onMessageWritten() {

	kf.doneWg.Add(1)
	defer kf.doneWg.Done()
	for {
		select {
		case <-kf.markStop:
			kf.log.Notice("Marking Message Written stopped")
			return
		case msg, more := <-kf.markWrittenChan:
			if !more {
				kf.log.Notice("Mark Message Written Producer stopped")
				return
			}
			// find the proper topic/partition, etc

			// attempt to guess where we really are, this is bit of a data race in that
			// by the time things get here, we could have consumed more ids from the data topic of this
			// reader and the offset we assume may be ahead of the real one this series had
			if msg.Offset == 0 {
				offMarker := kafkaOffsetMap.Get(msg.Uid)
				if offMarker != nil {
					msg.Topic = offMarker.Topic
					msg.Partition = offMarker.Partition
					msg.Offset = offMarker.Offset
				}
			}
			item := &series.KWrittenMessage{
				MetricWritten: *msg,
			}

			item.SetSendEncoding(kf.enctype)

			stats.StatsdClientSlow.Incr("injector.kafka.mark.written.writes", 1)

			kf.conn.Input() <- &sarama.ProducerMessage{
				Topic: kf.topic,
				Key:   sarama.StringEncoder(msg.Uid),
				Value: item,
			}

			// also send to the partition manager for levelDB marking
			if kf.mainConsumer.localMark != nil {
				kf.mainConsumer.localMark.MarkChan() <- item
			}
			kf.log.Debug(
				"Marked metric %s:%s as written at %d on topic:%s partition:%d offset %d",
				msg.Metric,
				msg.Uid,
				msg.WriteTime,
				msg.Topic,
				msg.Partition,
				msg.Offset,
			)
		}
	}
}
