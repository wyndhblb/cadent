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

package indexer

import (
	//. "github.com/smartystreets/goconvey/convey"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

var testMetricList = []string{
    "stats.timers.kafka-broker.kafka.network.RequestMetrics.StopReplica-LocalTimeMs.p75.count",
    "servers.tester-dev.haproxy.service.frontend.comp_rsp",
    "stats.timers.kafka-broker.kafka.network.RequestMetrics.OffsetFetch-RequestQueueTimeMs.p98.upper",
    "stats.timers.kafka-broker.kafka.server.BrokerTopicMetrics.analytic-prod-20-FailedProduceRequestsPerSec.m1_rate.mean",
    "stats.timers.kafka-broker.kafka.network.RequestMetrics.request.UpdateMetadata.ResponseSendTimeMs.p95.std",
    "stats.timers.kafka-broker.kafka.server.BrokerTopicMetrics.topic.stripe-webhooks-integ-5.MessagesInPerSec.m15_rate.upper",
    "servers.elastic-dev.elasticsearch.indices.foods_x010.search.fetch_total",
    "stats.app-dev.execution.blocks.event_producer.86581b6dc6955b8dd5de55d3324fa961.kafka.sent",
    "servers.db-cassandra.org.apache.cassandra.metrics.keyspace.RangeLatency.metric.m5_rate",
    "stats.timers.kafka-broker.kafka.server.BrokerTopicMetrics.topic.recipe-indexer.FailedProduceRequestsPerSec.mean_rate.mean_95",
    "stats.timers.kafka-broker.kafka.server.BrokerTopicMetrics.exercises-FailedFetchRequestsPerSec.mean_rate.upper",
    "servers.stage-dev.mysql.Com_lock_tables_for_backup",
    "stats.timers.kafka-broker.kafka.network.RequestMetrics.Fetch-Consumer-ResponseSendTimeMs.p98.count_99",
    "stats.timers.kafka-broker.kafka.server.BrokerTopicMetrics.foods-FailedFetchRequestsPerSec.m5_rate.upper_95",
    "servers.metrics-cassandra.org.apache.cassandra.metrics.ThreadPools.TotalBlockedTasks.internal.Repair#105.count",
    "stats.timers.app-svc-dev.in-flight-time.upper_95",
    "stats.timers.kafka-broker.kafka.server.BrokerTopicMetrics.notes-LogBytesAppendedPerSec.m15_rate.upper_90",
    "stats.timers.kafka-broker.kafka.server.BrokerTopicMetrics.food-search-20-FailedFetchRequestsPerSec.mean_rate.upper_99",
    "stats.timers.kafka-broker.kafka.server.BrokerTopicMetrics.topic.ClientEvent.BytesRejectedPerSec.m5_rate.sum_90",
    "stats.svc-test.bad-line",
    "stats.timers.kafka-broker.kafka.network.RequestMetrics.Fetch-Consumer-ResponseQueueTimeMs.p95.count_90",
    "stats.timers.kafka-broker.kafka.server.BrokerTopicMetrics.user-indexer-BytesInPerSec.mean_rate.count_99",
    "stats.timers.app-api.response_time_ms.std",
    "stats.timers.kafka-broker.kafka.server.BrokerTopicMetrics.topic.mfp-meal-event.BytesRejectedPerSec.m1_rate.mean",
    "stats.timers.app-api.get-zone.mean",
    "stats.timers.kafka-broker.kafka.network.RequestMetrics.request.Produce.ResponseSendTimeMs.stddev.sum_95",
    "stats.timers.kafka-broker.kafka.network.RequestMetrics.Metadata-RequestQueueTimeMs.p50.lower",
    "stats.timers.app-api.requests.fmr.response_time_ms.count",
    "stats.timers.kafka-broker.kafka.server.BrokerTopicMetrics.cognitivecoaching-interaction-event-BytesOutPerSec.mean_rate.mean_99",
    "stats.timers.app-api.GET.ms_in_proxy.upper_90",
    "stats.timers.kafka-broker.kafka.server.BrokerTopicMetrics.food-search-FailedProduceRequestsPerSec.m5_rate.sum_95",
    "stats.timers.app-api.GET.response_time_ms.count_95",
    "servers.app-dev.memory.VmallocChunk",
    "stats.timers.kafka-broker.kafka.server.BrokerTopicMetrics.analytic-integ-p20-LogBytesAppendedPerSec.mean_rate.lower",
    "stats.timers.kafka-broker.kafka.network.RequestMetrics.Fetch-Consumer-TotalTimeMs.p99.upper_99",
    "stats.timers.notification-aggregator.http-client.registry-1-schema-mbus-integ-mfpaws-com-3061.response_time_ms.mean",
    "stats.timers.consumer.app.notes.in-flight-time.upper_95",
    "stats.timers.kafka-broker.kafka.server.BrokerTopicMetrics.topic.restaurants-offline-integ.BytesOutPerSec.mean_rate.count_95",
    "stats.timers.kafka-broker.kafka.server.BrokerTopicMetrics.stripe-webhooks-integ-5-FailedFetchRequestsPerSec.m1_rate.median",
    "servers.service-dev.network.eth0.tx_multicast",
    "stats.timers.kafka-broker.kafka.server.BrokerTopicMetrics.restaurants-offline-MessagesInPerSec.mean_rate.count_ps",
    "stats.timers.kafka-broker.kafka.network.RequestMetrics.request.Fetch.TotalTimeMs.p999.sum",
    "stats.timers.kafka-broker.kafka.network.RequestMetrics.OffsetCommit-RemoteTimeMs.min.count_99",
    "servers.db-elastic.elasticsearch.thread_pool.suggest.threads",
    "servers.db-cassandra.org.apache.cassandra.metrics.keyspace.RangeLatency.system_traces.p999",
    "stats.timers.integ-app-service-venue.menu_item_matcher.db.bulk-menu-items-matches.response_time_ms.count_ps",
    "stats.timers.kafka-broker.kafka.server.BrokerTopicMetrics.topic.mfp-images.BytesInPerSec.m15_rate.upper_99",
    "servers.app-server.iostat.xvda.read_requests_merged_per_second",
    "stats.timers.kafka-broker.kafka.log.LogFlushStats.LogFlushRateAndTimeMs.p999.upper_99",
    "stats.timers.kafka-broker.kafka.server.BrokerTopicMetrics.uacf-client-event-MessagesInPerSec.mean_rate.mean_99",
    "stats.timers.kafka-broker.kafka.server.BrokerTopicMetrics.topic.ClientEvent.BytesRejectedPerSec.m15_rate.upper_90",
    "stats.timers.kafka-broker.kafka.server.BrokerTopicMetrics.recipe-indexer-FailedProduceRequestsPerSec.m1_rate.upper",
    "stats.timers.kafka-broker.kafka.network.RequestMetrics.StopReplica-RemoteTimeMs.p75.sum",
    "stats.timers.kafka-broker.kafka.network.RequestMetrics.request.ConsumerMetadata.TotalTimeMs.p999.sum_90",
    "stats.timers.kafka-broker.kafka.network.RequestMetrics.request.ConsumerMetadata.RequestQueueTimeMs.p95.lower",
    "stats.timers.kafka-broker.kafka.server.BrokerTopicMetrics.topic.diary.FailedFetchRequestsPerSec.mean_rate.upper_90",
    "servers.broker-4-kafka-mbus-integ.iostat.xvde.writes_milliseconds",
    "stats.timers.consumer.apps.delete.time.lower",
}

func testMetricIndexSetup() (*MetricIndex, []string) {
	tx, _ := NewMetricIndex("/tmp/test")

	uid := 100000000
	uidStr := strconv.FormatUint(uint64(uid), 36)
	uidList := []string{}
	for _, tgList := range testMetricList {
		tx.Add(uidStr, tgList)
		uidList = append(uidList, uidStr)
		uid++
		uidStr = strconv.FormatUint(uint64(uid), 36)

	}

	return tx, uidList
}

func testMetricIndexSetupForBench(mets int, index bool) (*MetricIndex, []string, []string, []string) {
	tx, _ := NewMetricIndex("/tmp/test")

	added := 0
	uid := 100000000
	uidStr := strconv.FormatUint(uint64(uid), 36)
	uidList := []string{}
	findList := []string{}
	addedStr := []string{}
	for added < mets {

		for _, tgList := range testMetricList {

			// randomly shuffle things to get lots of names
			spl := strings.Split(tgList, ".")
			perms := rand.Perm(len(spl))
			str := ""
			for _, i := range perms {
				str += spl[i] + "."
			}
			str = str[:len(str)-2]

			// take the first 4 to make the regex list
			fstr := ""
			for _, i := range perms[:3] {
				fstr += spl[i] + "."
			}
			fstr += "*"
			if index {
				tx.Add(uidStr, str)
			}
			uidList = append(uidList, uidStr)
			findList = append(findList, fstr)
			addedStr = append(addedStr, str)
			uid++
			uidStr = strconv.FormatUint(uint64(uid), 36)
			added++
			if added > mets {
				return tx, addedStr, findList, uidList
			}
		}
	}

	return tx, addedStr, findList, uidList
}

func Test_MetricRamIndex__Add(t *testing.T) {

	tx, _ := testMetricIndexSetup()

	for _, met := range testMetricList {
		if _, ok := tx.nameIndex[met]; !ok {
			t.Fatalf("Metric %s not in name index", met)
		}
	}
}

func Test_MetricRamIndex__FindIter(t *testing.T) {
	tx, _ := testMetricIndexSetup()

	iter := tx.FindIter("", nil)
	ct := 0
	for iter.Next() {
		if len(iter.Value()) > 0 {
			ct++
		}
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		t.Fatal(err)
	}
	if ct != len(testMetricList) {
		t.Fatalf("Not all metrics indexed wanted %d, got %d", len(testMetricList), ct)
	}
}

func Test_MetricRamIndex__FindRoot(t *testing.T) {
	tx, _ := testMetricIndexSetup()

	items, err := tx.FindRoot()
	if err != nil {
		t.Fatalf("FindRoot returned err: %v", err)
	}
	ct := 0
	for _, it := range items {
		if it.Expandable != 1 {
			t.Fatalf("Sould be expandable")
		}
		if it.Path == "stats" {
			ct++
		}
		if it.Path == "servers" {
			ct++
		}
		fmt.Println("Root: ", it.Path)
	}
	if ct != 2 {
		t.Fatalf("Not all root metrics indexed wanted 2, got %d", ct)
	}
}

func Test_MetricRamIndex__Find(t *testing.T) {
	tx, _ := testMetricIndexSetup()

	fi := testMetricList[0]

	items, err := tx.Find(fi, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("Did not find the one we wanted")
	}
	fmt.Println("Found Exact:", items[0].Path, items[0].UniqueId)

	// regexer
	for _, fi = range []string{"stats.timers.*", "stats.t[i]?mers.*", "{stats|servers}.timers.*"} {

		items, err = tx.Find(fi, false)

		if err != nil {
			t.Fatal(err)
		}

		needMap := make(map[string]bool)
		//the should get
		for _, i := range testMetricList {
			want := strings.Split(i, ".")

			if len(want) > 3 {
				if strings.HasPrefix(i, "stats.timers.") {
					needMap[strings.Join(want[:3], ".")] = true
				}
			}
		}

		ct := 0
		for _, it := range items {
			ct++
			fmt.Println("Found: ", it.Path)
		}
		if ct != len(needMap) {
			t.Logf("Not all root metrics indexed wanted %d, got %d", len(needMap), ct)
		}
	}
}

func Test_MetricRamIndex__Delete(t *testing.T) {
	tx, _ := testMetricIndexSetup()

	fi := testMetricList[0]

	items, err := tx.Find(fi, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("Did not find the one we wanted")
	}
	fmt.Println("Found Exact:", items[0].Path, items[0].UniqueId)

	err = tx.Delete(fi)
	if err != nil {
		t.Fatal(err)
	}
	items, err = tx.Find(fi, false)
	if err != nil {
		t.Fatal("find err:", err)
	}
	if len(items) != 0 {
		t.Fatal("Delete failed")
	}
}

func Test_MetricRamIndex__DeleteByUid(t *testing.T) {
	tx, uidList := testMetricIndexSetup()

	fi := uidList[0]

	items, err := tx.FindByUid(fi)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("Did not find the one we wanted")
	}
	fmt.Println("Found Exact:", items[0].Path, items[0].UniqueId)

	err = tx.DeleteByUid(fi)
	if err != nil {
		t.Fatal(err)
	}
	items, err = tx.Find(fi, false)
	if err != nil {
		t.Fatal("err:", err)
	}
	if len(items) != 0 {
		t.Fatal("Delete failed")
	}
}

func Benchmark_MetricIndex__IndexTime(b *testing.B) {
	b.ReportAllocs()
	_, keys, _, uids := testMetricIndexSetupForBench(10000, false)
	b.ResetTimer()
	idxed := 0
	start := time.Now()

	for i := 0; i < b.N; i++ {
		tx, _ := NewMetricIndex("/tmp/test")
		for i, t := range keys {
			tx.Add(uids[i], t)
			idxed++
		}
	}
	endT := time.Now()

	b.Logf("Did %d Indexes :: %f per second ", idxed, float64(time.Second)*float64(idxed)/float64(endT.UnixNano()-start.UnixNano()))

}

func Benchmark_MetricIndex__FindMulti(b *testing.B) {
	procs := []int{1, 2, 4, 8}
	mEles := []int{1000, 10000, 100000}

	for _, proc := range procs {
		for _, maxEle := range mEles {
			b.Run(fmt.Sprintf("Find_Procs_%d_eles_%d", proc, maxEle), func(b *testing.B) {
				tx, _, fi, _ := testMetricIndexSetupForBench(maxEle, true)

				findF := make(chan string, proc)
				var wg sync.WaitGroup

				doFind := func() {
					for {
						tg, more := <-findF
						if !more {
							return
						}
						tx.Find(tg, false)
						wg.Done()
					}
				}

				for i := 0; i < proc; i++ {
					go doFind()
				}
				start := time.Now()
				finds := 0

				b.ResetTimer()
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					for _, finder := range fi {
						wg.Add(1)
						findF <- finder
						finds++
					}
					wg.Wait()
				}
				endT := time.Now()
				b.Logf("Did %d Finds :: %f per second for %d elements at %d procs", finds, float64(time.Second)*float64(finds)/float64(endT.UnixNano()-start.UnixNano()), maxEle, proc)
				close(findF)
			})
		}
	}
}
