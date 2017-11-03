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
   Kafka injector tester
*/

package injectors

import (
	"cadent/server/schemas/metrics"
	"cadent/server/schemas/repr"
	"cadent/server/schemas/series"
	"cadent/server/test/helper"
	"cadent/server/utils/options"
	"cadent/server/writers"
	"fmt"
	"github.com/Shopify/sarama"
	"testing"
	"time"
)

var kport string = "9092"

var enctypes []string = []string{"json", "msgpack", "protobuf"}

func NoOpLog(format string, args ...interface{}) {

}

type TestWorker struct {
	Name   string
	logger func(format string, args ...interface{})
	all_ct int
	raw_ct int
	up_ct  int
	sin_ct int
	ser_ct int
}

func (w *TestWorker) Config(options.Options) error {
	return nil
}

// interface matching
func (w *TestWorker) SetWriter(wr *writers.WriterLoop)               {}
func (w *TestWorker) GetWriter() *writers.WriterLoop                 { return nil }
func (w *TestWorker) SetSubWriter(wr *writers.WriterLoop)            {}
func (w *TestWorker) GetSubWriter() *writers.WriterLoop              { return nil }
func (w *TestWorker) Repr(metric series.KMessageBase) *repr.StatRepr { return nil }

func (w *TestWorker) DoWork(metric series.KMessageBase) (*repr.StatRepr, error) {
	w.all_ct++
	switch metric.(type) {
	case *series.KMetric:
		m := metric.(*series.KMetric)
		r := m.Repr()
		w.logger("AnyMatric")
		if m.Raw != nil {
			w.raw_ct++
			w.logger("%s", r)
		}
		if m.Single != nil {
			w.sin_ct++
			w.logger("%s", r)
		}
		if m.Unprocessed != nil {
			w.up_ct++
			w.logger("%s", r)
		}
		if m.Series != nil {
			w.ser_ct++
			w.logger("%s", r)
		}
		return nil, nil
	case *series.KRawMetric:
		w.logger("RawMetric")
		w.logger("%s", metric.(*series.KRawMetric).Repr())
		w.raw_ct++
		return nil, nil
	case *series.KUnProcessedMetric:
		w.logger("Unprocessed")
		w.logger("%s", metric.(*series.KUnProcessedMetric).Repr())
		w.up_ct++
		return nil, nil
	case *series.KSingleMetric:
		w.logger("Single")
		w.logger("%s", metric.(*series.KSingleMetric).Repr())
		w.sin_ct++
		return nil, nil
	case *series.KSeriesMetric:
		w.logger("Series")
		w.logger("metric series: " + metric.(*series.KSeriesMetric).Metric)
		w.ser_ct++
		return nil, nil
	default:
		return nil, ErrorBadMessageType
	}
}

func (w *TestWorker) DoWorkRepr(r *repr.StatRepr) error { return nil }
func (w *TestWorker) DoWorkReprOffset(r *repr.StatRepr, offset *metrics.OffsetInSeries) error {
	return nil
}

func getConsumer(enctype string, cgroup string, topic string, t func(format string, args ...interface{})) (*Kafka, error) {
	on_ip := helper.DockerIp()
	config_opts := options.Options{}
	config_opts.Set("dsn", fmt.Sprintf("%s:%s", on_ip, kport))
	config_opts.Set("topic", topic)
	config_opts.Set("consumer_group", cgroup)
	config_opts.Set("starting_offset", "newest")
	config_opts.Set("encoding", enctype)
	config_opts.Set("message_type", "any")

	kf := NewKafka("tester")
	err := kf.Config(config_opts)
	if err != nil {
		return nil, err
	}

	// set the work to the echo type
	wk := new(TestWorker)
	wk.logger = t
	kf.KafkaWorker = wk

	return kf, nil
}

func getProducer() (sarama.AsyncProducer, error) {
	config := sarama.NewConfig()
	on_ip := helper.DockerIp()

	config.Producer.Retry.Max = int(3)
	config.Producer.RequiredAcks = sarama.WaitForAll
	producer, err := sarama.NewAsyncProducer([]string{on_ip + ":" + kport}, config)

	return producer, err
}

func getMetrics(len int, useencoding series.SendEncoding) []series.KMessageBase {
	msgs := make([]series.KMessageBase, len)

	for idx := range msgs {

		switch idx % 3 {
		case 0:
			msgs[idx] = &series.KMetric{
				AnyMetric: series.AnyMetric{
					Raw: &series.RawMetric{
						Metric: "moo.goo",
						Time:   time.Now().Unix(),
						Value:  100.0 * float64(idx),
						Tags:   repr.SortingTags([]*repr.Tag{{Name: "name", Value: "val"}, {Name: "name2", Value: "val2"}}),
					},
				},
			}
		case 1:
			msgs[idx] = &series.KMetric{
				AnyMetric: series.AnyMetric{
					Unprocessed: &series.UnProcessedMetric{
						Metric: "moo.goo.unp",
						Time:   time.Now().Unix(),
						Sum:    1000.0,
						Min:    1.0,
						Max:    120.0,
						Count:  2 + int64(idx),
						Tags:   repr.SortingTags([]*repr.Tag{{Name: "name", Value: "val"}, {Name: "name2", Value: "val2"}}),
					},
				},
			}

		default:

			msgs[idx] = &series.KMetric{
				AnyMetric: series.AnyMetric{
					Single: &series.SingleMetric{
						Metric: "moo.goo.sing",
						Id:     123123,
						Uid:    "asdasd",
						Time:   time.Now().Unix(),
						Sum:    100.0,
						Min:    1.0,
						Max:    10.0,
						Count:  20 + int64(idx),
						Tags:   repr.SortingTags([]*repr.Tag{{Name: "name", Value: "val"}, {Name: "name2", Value: "val2"}}),
					},
				},
			}
		}
		msgs[idx].SetSendEncoding(useencoding)
	}
	return msgs
}

func TestKafkaInjector(t *testing.T) {

	// fire up the docker test helper
	helper.DockerUp("kafka")
	ok := helper.DockerWaitUntilReady("kafka")
	if !ok {
		t.Fatalf("Could not start the docker container for kafka")
	}
	use_topic := "cadent-test"
	for _, enctype := range enctypes {
		var useencoding series.SendEncoding = series.SendEncodingFromString(enctype)

		kf, err := getConsumer(enctype, "cadent-test", use_topic, t.Logf)
		if err != nil {
			t.Fatalf("Failed to get consumer: %v", err)
		}
		err = kf.Start()
		if err != nil {
			t.Fatalf("Failed to start: %v", err)
		}

		t.Logf("consumer: started")

		// some raw messages
		NumMessages := 10
		msgs := getMetrics(NumMessages, useencoding)
		prod, err := getProducer()
		if err != nil {
			t.Fatalf("Error on producer: %v", err)
		}

		t.Logf("testing messages")

		for {
			if kf.IsReady {

				for _, msg := range msgs {
					prod.Input() <- &sarama.ProducerMessage{
						Topic: use_topic,
						Key:   sarama.StringEncoder(msg.Id()),
						Value: msg,
					}
					t.Logf("Produced: %v : %v", msg.Id(), msg)
				}
				prod.Close()
				break

			}
			time.Sleep(time.Second)
		}

		t.Logf("consumer: running")

		time.Sleep(20 * time.Second)
		t.Logf("consumer: stopping")

		err = kf.Stop()
		if err != nil {
			t.Fatalf("Error stop kafka: %v", err)
		}
		wrk := kf.KafkaWorker.(*TestWorker)
		if wrk.all_ct != NumMessages {
			t.Fatalf("Raw metric counts do not match produced %d/%d", wrk.all_ct, NumMessages)
		}

		t.Logf("consumer: stopped")
	}

}

// Note after running the benchmarks you should probably remove the topic and
// start over as the "tests" above assume a

func Benchmark__Kafka_Encoding_JSON(b *testing.B) {
	// fire up the docker test helper
	helper.DockerUp("kafka")
	ok := helper.DockerWaitUntilReady("kafka")
	if !ok {
		b.Fatalf("Could not start the docker container for kafka")
	}
	use_topic := "cadent-json"
	kf, err := getConsumer("json", "json-bench", use_topic, NoOpLog)
	if err != nil {
		b.Fatalf("Failed to get consumer: %v", err)
	}
	err = kf.Start()
	if err != nil {
		b.Fatalf("Failed to start: %v", err)
	}

	// some raw messages
	NumMessages := 1000
	msgs := getMetrics(NumMessages, series.ENCODE_JSON)
	prod, err := getProducer()
	if err != nil {
		b.Fatalf("%v", err)
	}
	//wait till we are good to go
	for {
		if kf.IsReady {
			break
		}
		time.Sleep(time.Second)

	}
	b.ResetTimer()
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		for _, msg := range msgs {
			prod.Input() <- &sarama.ProducerMessage{
				Topic: use_topic,
				Key:   sarama.StringEncoder(msg.Id()),
				Value: msg,
			}
		}
	}
	b.StopTimer()
	prod.Close()

	kf.Stop()
}

func Benchmark__Kafka_Encoding_MSGP(b *testing.B) {
	// fire up the docker test helper
	helper.DockerUp("kafka")
	ok := helper.DockerWaitUntilReady("kafka")
	if !ok {
		b.Fatalf("Could not start the docker container for kafka")
	}
	use_topic := "cadent-msgp"
	kf, err := getConsumer("msgpack", "msgp-bench", use_topic, NoOpLog)
	if err != nil {
		b.Fatalf("Failed to get consumer: %v", err)
	}
	err = kf.Start()
	if err != nil {
		b.Fatalf("Failed to start: %v", err)
	}

	// some raw messages
	NumMessages := 1000
	msgs := getMetrics(NumMessages, series.ENCODE_MSGP)
	prod, err := getProducer()
	if err != nil {
		b.Fatalf("%v", err)
	}
	//wait till we are good to go
	for {
		if kf.IsReady {
			break
		}
		time.Sleep(time.Second)

	}
	b.ResetTimer()
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		for _, msg := range msgs {
			prod.Input() <- &sarama.ProducerMessage{
				Topic: use_topic,
				Key:   sarama.StringEncoder(msg.Id()),
				Value: msg,
			}
		}
	}
	b.StopTimer()
	prod.Close()

	kf.Stop()
}

func Benchmark__Kafka_Encoding_ProtoBuf(b *testing.B) {
	// fire up the docker test helper
	helper.DockerUp("kafka")
	ok := helper.DockerWaitUntilReady("kafka")
	if !ok {
		b.Fatalf("Could not start the docker container for kafka")
	}
	use_topic := "cadent-proto"
	kf, err := getConsumer("protobuf", "proto-bench", use_topic, NoOpLog)
	if err != nil {
		b.Fatalf("Failed to get consumer: %v", err)
	}
	err = kf.Start()
	if err != nil {
		b.Fatalf("Failed to start: %v", err)
	}

	// some raw messages
	NumMessages := 1000
	msgs := getMetrics(NumMessages, series.ENCODE_PROTOBUF)
	prod, err := getProducer()
	if err != nil {
		b.Fatalf("%v", err)
	}
	//wait till we are good to go
	for {
		if kf.IsReady {
			break
		}
		time.Sleep(time.Second)

	}
	b.ResetTimer()
	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		for _, msg := range msgs {
			prod.Input() <- &sarama.ProducerMessage{
				Topic: use_topic,
				Key:   sarama.StringEncoder(msg.Id()),
				Value: msg,
			}
		}
	}
	b.StopTimer()
	prod.Close()

	kf.Stop()
}
