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

package main

// a little "blaster of stats" send to kafka
// we use the basic series type in schema

import (
	"cadent/server/schemas/repr"
	"cadent/server/schemas/series"
	"cadent/server/utils/options"
	"cadent/server/writers/dbs"
	"fmt"
	"github.com/Shopify/sarama"
	"log"
	"os"
	"time"
)

func GetKafkaProducer(broker, topic string) sarama.AsyncProducer {
	opts := options.New()
	opts.Set("compression", "none")
	opts.Set("dsn", broker)
	opts.Set("metric_topic", topic)
	opts.Set("batch_count", 4096)
	opts.Set("flush_time", "10s")
	fmt.Println("Producing to ", topic)
	k, e := dbs.NewDB("kafka", broker, &opts)
	if e != nil {
		panic(e)
	}

	return k.Connection().(sarama.AsyncProducer)
}

func UnProcessedMakeKafkaMessage(l int) *series.KMetric {
	max := 1000000
	div := 10
	taglen := 6
	tgs := []*repr.Tag{}
	if doTags {
		tgs = sprinterTagList(taglen)
	}

	item := &series.KMetric{
		AnyMetric: series.AnyMetric{
			Unprocessed: &series.UnProcessedMetric{
				Metric: MakeKey(),
				Time:   time.Now().UnixNano(),
				Sum:    randFloat(max, div),
				Last:   randFloat(max, div),
				Count:  randInt(10),
				Max:    randFloat(max, div),
				Min:    randFloat(max, div),
				Tags:   tgs,
			},
		},
	}

	return item
}

func KafkaBlast(server string, topic string, rate string, buffer int, enc string) {

	conn := GetKafkaProducer(server, topic)
	sleeper, err := time.ParseDuration(rate)
	if err != nil {
		log.Printf("Error in Rate: %s", err)
		os.Exit(1)
	}
	ienc := series.SendEncodingFromString(enc)

	for {
		stat := UnProcessedMakeKafkaMessage(numWords)
		stat.SetSendEncoding(ienc)
		conn.Input() <- &sarama.ProducerMessage{
			Topic: topic,
			Key:   sarama.StringEncoder(stat.Unprocessed.Metric), // hash on unique id
			Value: stat,
		}
		sentLines++
		time.Sleep(sleeper)
	}
}

/**/
