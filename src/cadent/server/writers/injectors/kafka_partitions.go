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
   Kafka a helper class that maintains topic+partition consumption information
*/

package injectors

import (
	"cadent/server/schemas/metrics"
	"cadent/server/schemas/series"
	"cadent/server/stats"
	"encoding/binary"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/syndtr/goleveldb/leveldb"
	leveldb_filter "github.com/syndtr/goleveldb/leveldb/filter"
	leveldb_opt "github.com/syndtr/goleveldb/leveldb/opt"
	leveldb_util "github.com/syndtr/goleveldb/leveldb/util"
	"gopkg.in/op/go-logging.v1"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// default location for th offset index
const KAFKA_OFFSET_LOCATION = "/opt/cadent/data/kafka"

// read cache size
const KAFKA_OFFSET_READ_CACHE_SIZE = 8

// basic size in MB of a given chunk of levelDB files
const KAFKA_OFFSET_LEVELDB_FILE_SIZE = 20

// the key that indicates the last offset time we wrote
const KAFKA_OFFSET_LAST_MARK = "kafka.lastmark"

// each marked offset gets this "prefix + topic + partition + the uid" -> series.MSG_WRITTEN
const KAFKA_OFFSET_OFFSET_PREFIX = "kafka.offset."

// helper for the localDB version of marking offsets
type localKafkaOffset struct {

	// we also store a local "offset" marker DB which will be used (if it's not empty)
	// otherwise we assume a new node and need to run the full commit topic
	writtenChan  chan *series.KWrittenMessage
	stopChan     chan bool
	messageType  series.MessageType
	encodingType series.SendEncoding
	timeRestart  time.Duration
	levelDBPath  string
	db           *leveldb.DB
	log          *logging.Logger
}

func NewLocalOffset(kf *Kafka) *localKafkaOffset {

	// do nothing if no kf.OffsetDbPath
	if len(kf.OffsetDbPath) == 0 || kf.OffsetDbPath == "none" {
		return nil
	}

	n := new(localKafkaOffset)
	n.levelDBPath = kf.OffsetDbPath
	n.timeRestart = kf.restartBackSlurpTime
	n.encodingType = kf.EncodingType
	n.messageType = kf.MessageType
	n.stopChan = make(chan bool)
	n.writtenChan = make(chan *series.KWrittenMessage, 8)
	n.log = kf.log

	return n
}

func (lo *localKafkaOffset) MarkChan() chan *series.KWrittenMessage {
	return lo.writtenChan
}

func (lo *localKafkaOffset) Init() error {
	// fire up the levelDB
	dbOpts := new(leveldb_opt.Options)
	dbOpts.Filter = leveldb_filter.NewBloomFilter(10)

	dbOpts.BlockCacheCapacity = KAFKA_OFFSET_READ_CACHE_SIZE * leveldb_opt.MiB
	dbOpts.CompactionTableSize = KAFKA_OFFSET_LEVELDB_FILE_SIZE * leveldb_opt.MiB

	info, err := os.Stat(lo.levelDBPath)
	if err == nil && info.IsDir() {

	} else {
		err = os.MkdirAll(lo.levelDBPath, 0755)
		if err != nil {
			return err
		}
	}
	lo.db, err = leveldb.OpenFile(lo.levelDBPath, dbOpts)
	if err != nil {
		return err
	}
	return nil
}

func (lo *localKafkaOffset) Stop() error {
	close(lo.stopChan)
	return nil
}

// check that the levelDB has a mark for the last offsets committed
// if there is no lastMark key, we have to consume the entire commit topic first
func (lo *localKafkaOffset) getLatestOffsets(topic string, partition int32) (int64, error) {

	tpc := topic
	part := partition

	gots, err := lo.db.Get([]byte(KAFKA_OFFSET_LAST_MARK), nil)
	if err != nil {
		lo.log.Errorf("Error reading latest mark: %v", err)
		return 0, err
	}
	// no key move along
	if gots == nil || len(gots) == 0 {
		return 0, nil
	}

	// make sure the the last mark is w/i our timeback window

	tStamp, read := binary.Varint(gots)
	if read <= 0 {
		lo.log.Errorf("Error reading latest mark not enough bytes")
		return 0, err
	}

	onT := time.Unix(0, tStamp).UTC()
	backTick := time.Now().UTC().Add(-1 * lo.timeRestart)
	if onT.Before(backTick) {
		lo.log.Warningf("Local offset mark DB is out of date: Last update: %s", onT)
		return 0, nil
	}

	foundCT := int64(0)
	searchPrefix := fmt.Sprintf("%s%s.%d.", KAFKA_OFFSET_OFFSET_PREFIX, tpc, part)

	iter := lo.db.NewIterator(leveldb_util.BytesPrefix([]byte(searchPrefix)), nil)
	for iter.Next() {

		// this best be a "written" message
		newObj := series.KMetricObjectFromType(series.MSG_WRITTEN)
		newObj.SetSendEncoding(lo.encodingType)
		err := newObj.Decode(iter.Value())

		if err != nil {
			lo.log.Errorf("Could not process levelDB message: %v", err)
			series.PutPool(newObj) // put it back in the syc pool
			continue
		}

		conv, ok := newObj.(*series.KWrittenMessage)
		if !ok {
			lo.log.Errorf("level DB Message is not a KMetricWritten object: %v", err)
			series.PutPool(newObj) // put it back in the syc pool
			continue
		}
		if conv.Offset > 0 {
			// only deal with things in our time back window
			of := &offsetMarker{
				Id:        conv.Uid,
				Offset:    conv.Offset,
				Partition: conv.Partition,
				Topic:     conv.Topic,
				WriteTime: conv.WriteTime,
			}
			kafkaOffsetMap.AddIfBigger(of)
			foundCT++
			lo.log.Debugf("Found offset for %s on %s.%d @ %d", conv.Uid, conv.Topic, conv.Partition, conv.Offset)

		} else {
			lo.log.Warning("Message does not have an offset: %s parition: %s topid: %s", conv.Metric, conv.Partition, conv.Topic)
		}
	}
	iter.Release()
	return foundCT, nil
}

// this consumes the marked offsets sent from KafkaMarkWritten to write into levelDB lands
func (lo *localKafkaOffset) processMarkWrote(topic string, partition int32) {
	tpc := topic
	part := partition
	keyPerf := fmt.Sprintf("%s%s.%d.", KAFKA_OFFSET_OFFSET_PREFIX, tpc, part)
	for {
		select {
		case <-lo.stopChan:
			return
		case mw := <-lo.writtenChan:
			key := keyPerf + mw.Uid
			lo.log.Debugf("Marking local offset for %s @ offset %d", key, mw.Offset)
			val, err := mw.Encode()
			if err != nil {
				lo.log.Errorf("Could not encode written message: %v", err)
				continue
			}
			err = lo.db.Put([]byte(key), val, nil)
			if err != nil {
				lo.log.Errorf("Could not write written message: %v", err)
				continue
			}
			buf := make([]byte, 16)
			wrote := binary.PutVarint(buf, time.Now().UnixNano())
			// mark the latest
			lo.db.Put([]byte(KAFKA_OFFSET_LAST_MARK), buf[0:wrote], nil)
		}
	}
}

// each consumed partition gets one of these managers to deal w/ offset marking and consuming
type kafkaPartition struct {
	partition          int32
	topic              string
	commitTopic        string
	offset             string
	useOffset          int64
	messageType        series.MessageType
	encodingType       series.SendEncoding
	timeRestart        time.Duration
	resetConsumerCount int

	worker                  Worker
	client                  sarama.Client
	consumer                sarama.Consumer
	partitionConsumer       sarama.PartitionConsumer
	rollupPartitionConsumer sarama.PartitionConsumer
	injector                Injector

	consumeStop chan bool
	doneWg      sync.WaitGroup
	log         *logging.Logger

	localMark *localKafkaOffset
}

func newFromPartitionFromKafka(kf *Kafka) *kafkaPartition {
	n := new(kafkaPartition)
	n.offset = kf.StartOffset
	n.commitTopic = kf.CommitTopic
	n.timeRestart = kf.restartBackSlurpTime
	n.worker = kf.KafkaWorker
	n.log = kf.log
	n.consumeStop = kf.consumeStop
	n.client = kf.client
	n.consumer = kf.consumer
	n.encodingType = kf.EncodingType
	n.messageType = kf.MessageType
	n.injector = kf
	n.localMark = kf.localMark
	n.resetConsumerCount = 10

	return n
}

func (kf *kafkaPartition) startConsume() error {

	var err error
	// get the proper offset .. kafka 10.1 has a "time" index we can use, and we should
	kf.useOffset = sarama.OffsetNewest
	skipCommitLog := true
	switch kf.offset {
	case "oldest":
		kf.useOffset = sarama.OffsetOldest
		skipCommitLog = false
	case "reset":
		kf.useOffset = sarama.OffsetOldest
		skipCommitLog = true
	case "time":
		skipCommitLog = false
		kf.useOffset, err = kf.client.GetOffset(
			kf.topic,
			kf.partition,
			time.Now().Add(-1*kf.timeRestart).UnixNano()/int64(time.Millisecond),
		)
		// if out of range, just set to newest
		if err != nil {
			kf.log.Warningf("Offset failure: %v", err)
		}
		if err == sarama.ErrOffsetOutOfRange {
			kf.log.Warningf("Offset of out range for %v: setting to oldest", kf.timeRestart)
			kf.useOffset = sarama.OffsetOldest
			err = nil
		}
	}
	if kf.useOffset == sarama.OffsetOldest || kf.useOffset == sarama.OffsetNewest {
		kf.useOffset, err = kf.client.GetOffset(
			kf.topic,
			kf.partition,
			kf.useOffset,
		)

		// if out of range, just set to newest
		if err != nil {
			kf.log.Warningf("Offset failure: %v, defaulting to the consumer groups last known position", err)
		}
	}

	if err != nil {
		return err
	}

	if !skipCommitLog {
		var gotLocal int64
		// first try the local index
		if kf.localMark != nil {
			gotLocal, err = kf.localMark.getLatestOffsets(kf.topic, kf.partition)
			if err != nil {
				kf.log.Error("Local get of offsets failed reverting back to commit topic: %v", err)
			}
			if gotLocal > 0 {
				kf.log.Noticef("Found %d local mark offsets", gotLocal)
			}
		}

		if gotLocal <= 0 {
			// fill up the commit log first
			_, err = kf.getLatestOffsetsCommitted()
			if err != nil {
				return err
			}
		}
	} else {
		kf.log.Noticef("Directive was to start at oldest offset, we will not be attempting to figure out the offsets")
	}

	minOffset := kafkaOffsetMap.PartitionMinOffset(kf.topic, kf.partition)
	if minOffset > 0 {
		kf.useOffset = minOffset
		kf.log.Notice("Found minimum offset to start on %s:%d at %d", kf.topic, kf.partition, minOffset)
	}

	// need to check that the min offset is still available
	lastOffset, err := kf.client.GetOffset(kf.topic, kf.partition, sarama.OffsetOldest)
	if err != nil {
		kf.log.Errorf("getting %s:%d oldest offset failed have revert to newest: %v", kf.topic, kf.partition, err)
	}

	if lastOffset > kf.useOffset {
		kf.useOffset = lastOffset
	}

	kf.partitionConsumer, err = kf.consumer.ConsumePartition(kf.topic, kf.partition, kf.useOffset)

	// can start at the newest one here, as, this is both the writer of to this topic and consumer
	// kf.rollupPartitionConsumer, err = kf.consumer.ConsumePartition(kf.commitTopic, kf.partition, sarama.OffsetNewest)

	if err != nil {
		kf.log.Errorf("Start Consume error: %v", err)
		return err
	}

	go kf.onConsumePartition()
	return nil
}

// before we start consuming the main data pipes, we grab all the committed message from the maxtime back
// and prefill the kf.currentOffsets object
func (kf *kafkaPartition) getLatestOffsetsCommitted() (int64, error) {
	var err error
	var commitOffset int64 = sarama.OffsetOldest
	backTime := int64(0)
	if kf.offset == "time" {
		backTime = time.Now().UTC().Add(-1 * kf.timeRestart).UnixNano()
		commitOffset, err = kf.client.GetOffset(
			kf.topic,
			kf.partition,
			time.Now().UTC().Add(-1*kf.timeRestart).UnixNano()/int64(time.Millisecond),
		)

		// if out of range, just set to oldest
		if err != nil {
			kf.log.Warningf("Offset failure: %v", err)
		}
		if err == sarama.ErrOffsetOutOfRange {
			kf.log.Warningf("Offset of out range for %v: setting to oldest", kf.timeRestart)
			commitOffset = sarama.OffsetOldest
			err = nil
		} else if err != nil {
			commitOffset = sarama.OffsetOldest
		}
	}
	// need to get the real offset
	if commitOffset == 0 || commitOffset == sarama.OffsetOldest {
		commitOffset, err = kf.client.GetOffset(
			kf.topic,
			kf.partition,
			sarama.OffsetOldest,
		)

		// if out of range, just set to newest
		if err != nil {
			kf.log.Warningf("Offset failure: %v, defaulting to the consumer groups last known position", err)
		}
	}

	pC, err := kf.consumer.ConsumePartition(kf.commitTopic, kf.partition, commitOffset)
	// start at the oldest if we get the outside of range error
	if err != nil && strings.Contains(err.Error(), "The requested offset is outside the range") {
		commitOffset = sarama.OffsetOldest
		pC, err = kf.consumer.ConsumePartition(kf.commitTopic, kf.partition, sarama.OffsetOldest)
	}
	if err != nil {
		kf.log.Errorf("Error in Commit topic consumer: %v", err)
		return 0, err
	}

	dataChan := pC.Messages()
	errChan := pC.Errors()

	kf.log.Noticef("Starting Commit Consumer for topic: %s, partition: %d, offset: %d", kf.commitTopic, kf.partition, commitOffset)
	tick := time.NewTicker(time.Second)

	// need to get the last in case logs things are remove
	lastOffset, err := kf.client.GetOffset(kf.commitTopic, kf.partition, sarama.OffsetOldest)
	if err != nil {
		kf.log.Errorf("Getting oldest offset failed, have to set to Newest otherwise we will hang: %v", err)
		lastOffset, err = kf.client.GetOffset(kf.commitTopic, kf.partition, sarama.OffsetNewest)
		if err != nil {
			kf.log.Errorf("Getting newest offset failed, i think things are messed: %v", err)
		}
	}
	minOff := int64(-1)

	for {
		select {
		case <-kf.consumeStop:
			kf.log.Notice("Consumer stopped for topic: %s, partition: %d, offset: %d", kf.commitTopic, kf.partition, commitOffset)
			return minOff, nil
		case <-tick.C:
			latest, err := kf.client.GetOffset(kf.commitTopic, kf.partition, sarama.OffsetNewest)
			if err != nil {
				kf.log.Errorf("Failed to get latest offset: %v", err)
			} else {
				delta := latest - lastOffset
				kf.log.Notice("At delta %d for %s:%d (on: %d newest: %d)", delta, kf.commitTopic, kf.partition, lastOffset, latest)
				// at the end and so we stop consuming
				if delta <= 1 {
					debug.FreeOSMemory() // free ram as this can be expensive
					kf.log.Infof("Consumed commit-log of %s:%d, can proceed with data consumption", kf.commitTopic, kf.partition)
					pC.Close()
					return minOff, nil
				}
			}
		case errs, more := <-errChan:
			if !more {
				return minOff, nil
			}
			kf.log.Errorf("Errors consuming: %v", errs)
		case msg, more := <-dataChan:

			if !more {
				kf.log.Notice("Consumer stopped")
				return minOff, nil
			}
			lastOffset = msg.Offset
			// this best be a "written" message
			newObj := series.KMetricObjectFromType(series.MSG_WRITTEN)
			newObj.SetSendEncoding(kf.encodingType)
			err := newObj.Decode(msg.Value)

			if err != nil {
				kf.log.Errorf("Could not process incoming message: %v", err)
				series.PutPool(newObj) // put it back in the syc pool
				continue
			}

			conv, ok := newObj.(*series.KWrittenMessage)
			if !ok {
				kf.log.Errorf("Message is not a KMetricWritten object: %v", err)
				series.PutPool(newObj) // put it back in the syc pool
				continue
			}
			if conv.Offset > 0 {
				// only deal with things in our time back window
				of := &offsetMarker{
					Id:        conv.Uid,
					Offset:    conv.Offset,
					Partition: conv.Partition,
					Topic:     conv.Topic,
					WriteTime: conv.WriteTime,
				}
				if backTime == 0 {
					kafkaOffsetMap.AddIfBigger(of)
				} else if conv.WriteTime >= backTime {
					kafkaOffsetMap.AddIfBigger(of)
				}

			} else {
				kf.log.Warning("Message does not have an offset: %s parition: %s topid: %s", conv.Metric, conv.Partition, conv.Topic)
			}
		}
	}
}

// main data consumer
func (kf *kafkaPartition) onConsumePartition() {
	kf.doneWg.Add(1)
	defer kf.doneWg.Done()

	pConsumer := kf.partitionConsumer
	topic := kf.topic
	partition := kf.partition
	offset := kf.useOffset

	dataChan := pConsumer.Messages()
	errChan := pConsumer.Errors()

	kf.log.Noticef("Starting Consumer for topic: %s, partition: %d, offset: %d", topic, partition, offset)
	tickRate := int64(10)
	tick := time.NewTicker(time.Second * time.Duration(tickRate))
	lastOffset := int64(0)

	// separate go routine for the stats ticker
	go func() {
		// if this > resetConsumerCount then we assume something borked upstream
		// and thus we "reset" the consumer
		numZeros := 0
		r := int64(10)
		lastCurrentOffset := int64(0) //consume Rate
		lastNewestOffset := int64(0)  //produce Rate
		deltaStat := fmt.Sprintf("injector.kafka.consume.offset.%s.%d.delta", topic, partition)
		rateStat := fmt.Sprintf("injector.kafka.consume.offset.%s.%d.rate", topic, partition)
		prodRateStat := fmt.Sprintf("injector.kafka.consume.offset.%s.%d.produce-rate", topic, partition)
		for {
			select {
			case <-kf.consumeStop:
				return
			case <-tick.C:
				latest, err := kf.client.GetOffset(topic, partition, sarama.OffsetNewest)
				if err != nil {
					kf.log.Errorf("Failed to get latest offset: %v", err)
				} else {
					curOff := atomic.LoadInt64(&lastOffset)
					delta := latest - curOff
					rate := curOff - lastCurrentOffset
					prodRate := latest - lastNewestOffset
					if lastCurrentOffset != 0 {
						kf.log.Info("Topic: %s, partition: %d, offset: %d, latest: %d, delta: %d, consume rate: %d/s produce rate: %d/s", topic, partition, lastOffset, latest, delta, rate/r, prodRate/r)
						stats.StatsdClientSlow.Gauge(deltaStat, delta)
						stats.StatsdClientSlow.Gauge(rateStat, rate/tickRate)
						stats.StatsdClientSlow.Gauge(prodRateStat, prodRate/tickRate)
					}
					lastCurrentOffset = curOff
					lastNewestOffset = latest
					if delta == 0 {
						numZeros++
					} else {
						numZeros = 0
					}
					if numZeros > kf.resetConsumerCount {
						kf.log.Critical("This partitions `%s:%d` has stopped consuming, we have no idea why (might be due to a rolled log before we finished consuming it)", topic, partition)
					}
				}
			}
		}
	}()

	// sometimes we get a ErrOffsetOutOfRange err, which 99% of the time means
	// that the "offset" asked for at the begining of the main loop is now too old (due to
	// mainly kafka log rotations before we've finished consuming the log),
	// as a result we need to restart the main loop again at new oldest offset
	mainLoop := func() int {
		for {
			select {
			case <-kf.consumeStop:
				kf.log.Notice("Consumer stopped for topic: %s, partition: %d, offset: %d", topic, partition, offset)
				return 0
			case errs, more := <-errChan:
				if !more {
					return 0
				}
				kf.log.Errorf("Errors consuming: %v", errs)
				if errs.Err == sarama.ErrOffsetOutOfRange {
					return int(sarama.ErrOffsetOutOfRange)
				}

			case msg, more := <-dataChan:

				if !more {
					kf.log.Notice("Consumer stopped")
					return 0
				}
				atomic.StoreInt64(&lastOffset, msg.Offset)

				//kf.log.Debug("Got message from %s: partition: %d, offset: %d", msg.Topic, msg.Partition, msg.Offset)
				newObj := series.KMetricObjectFromType(kf.messageType)
				newObj.SetSendEncoding(kf.encodingType)
				err := newObj.Decode(msg.Value)

				// we have commit offsets for decode fails otherwise we'll never move
				if err != nil {
					kf.log.Errorf("Could not process incoming message: %v", err)
					series.PutPool(newObj) // put it back in the syc pool
					continue
				}

				//check the last commits
				r := kf.worker.Repr(newObj)
				if r == nil {
					kf.log.Warning("Could not process incoming message: Not a metric message")
					continue
				}
				uid := r.Name.UniqueIdString()
				// too old, move on
				if kafkaOffsetMap.IsLower(uid, msg.Offset) {
					stats.StatsdClient.Incr("injector.kafka.consume.skipped", 1)
					continue
				}

				newOff := new(metrics.OffsetInSeries)
				newOff.Offset = lastOffset
				newOff.Partition = msg.Partition
				newOff.Topic = msg.Topic

				//override the min TTL (if defined) and different the one incoming
				if len(kf.injector.Resolutions()) > 0 && uint32(kf.injector.Resolutions()[0][1]) != r.Name.Ttl {
					r.Name.Ttl = uint32(kf.injector.Resolutions()[0][1])
				}

				err = kf.worker.DoWorkReprOffset(r, newOff)
				if err != nil {
					kf.log.Errorf("Error working on message: %v", err)
					series.PutPool(newObj) // put it back in the syc pool
					stats.StatsdClientSlow.Incr("injector.kafka.consume.errors", 1)
					continue
				}
				series.PutPool(newObj) // put it back in the syc pool
			}

		}
	}
	retry := mainLoop()
	for retry == int(sarama.ErrOffsetOutOfRange) {
		// reset the consumers to the oldest
		var err error
		kf.log.Warningf("Reseting %s/%d to the oldest offset, as we are out of range", topic, partition)
		kf.partitionConsumer, err = kf.consumer.ConsumePartition(kf.topic, kf.partition, sarama.OffsetOldest)
		if err != nil {
			kf.log.Errorf("Got error on reseting partition: %v", err)
			continue
		}
		pConsumer = kf.partitionConsumer
		dataChan = pConsumer.Messages()
		errChan = pConsumer.Errors()
		retry = mainLoop()
	}
}

// onConsumeRollupPartition listen to the commit channel, and then do rollups based on the times given.
// TODO
func (kf *kafkaPartition) onConsumeRollupPartition() {
	kf.doneWg.Add(1)
	defer kf.doneWg.Done()

	pConsumer := kf.rollupPartitionConsumer
	topic := kf.commitTopic
	partition := kf.partition
	offset := kf.useOffset

	dataChan := pConsumer.Messages()
	errChan := pConsumer.Errors()

	kf.log.Noticef("Rollup: Starting Consumer for topic: %s, partition: %d, offset: %d", topic, partition, offset)
	tick := time.NewTicker(time.Second * 10)
	lastOffset := int64(0)

	// separate go routine for the stats ticker
	go func() {
		for {
			select {
			case <-kf.consumeStop:
				return
			case <-tick.C:
				latest, err := kf.client.GetOffset(topic, partition, sarama.OffsetNewest)
				if err != nil {
					kf.log.Errorf("Failed to get latest offset: %v", err)
				} else {
					delta := latest - lastOffset
					kf.log.Info("Last consumed topic: %s, partition: %d, offset: %d, latest: %d delta: %d", topic, partition, lastOffset, latest, delta)
					stats.StatsdClient.Gauge(fmt.Sprintf("injector.kafka.consume.offset.%s.%d.delta", topic, partition), delta)
				}

			}
		}
	}()

	for {
		select {
		case <-kf.consumeStop:
			kf.log.Notice("Rollup: Consumer stopped for topic: %s, partition: %d, offset: %d", topic, partition, offset)
			return
		case errs, more := <-errChan:
			if !more {
				return
			}
			kf.log.Errorf("Errors consuming: %v", errs)
		case msg, more := <-dataChan:

			if !more {
				kf.log.Notice("Consumer stopped")
				return
			}
			lastOffset = msg.Offset

			//kf.log.Debug("Got message from %s: partition: %d, offset: %d", msg.Topic, msg.Partition, msg.Offset)
			newObj := series.KMetricObjectFromType(series.MSG_WRITTEN)
			newObj.SetSendEncoding(kf.encodingType)
			err := newObj.Decode(msg.Value)

			// we have commit offsets for decode fails otherwise we'll never move
			if err != nil {
				kf.log.Errorf("Could not process incoming message: %v", err)
				series.PutPool(newObj) // put it back in the syc pool
				continue
			}

			//check the last commits
			// TODO: actually do the rollup
			_, ok := newObj.(*series.KWrittenMessage)
			if !ok {
				kf.log.Errorf("Could not process incoming message: not a MetricWritten message: %v", err)
				continue
			}

			series.PutPool(newObj) // put it back in the syc pool
		}
	}
}

func (kp *kafkaPartition) LastIdOffset(id string) int64 {
	gt := kafkaOffsetMap.Get(id)
	if gt != nil {
		return gt.Offset
	}
	return 0
}

func (kp *kafkaPartition) LastOffset(id string) *offsetMarker {
	gt := kafkaOffsetMap.Get(id)
	if gt != nil {
		return gt
	}
	return nil
}

// maintain a list of the consumers so we can find out where a given id lives
type kafkaPartitionConsumers []*kafkaPartition
