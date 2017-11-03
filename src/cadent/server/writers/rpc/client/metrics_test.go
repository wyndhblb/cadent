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

// Metrics main RPC client

package client

import (
	//. "github.com/smartystreets/goconvey/convey"

	"cadent/server/schemas/api"
	sindexer "cadent/server/schemas/indexer"
	smetrics "cadent/server/schemas/metrics"
	"cadent/server/schemas/repr"
	"cadent/server/schemas/series"
	"cadent/server/test/helper"
	"cadent/server/utils/options"
	"cadent/server/writers/indexer"
	"cadent/server/writers/metrics"
	"cadent/server/writers/rpc/server"
	"fmt"
	"golang.org/x/net/context"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"
)

var myPort int = 3308 // see docker-compose.yml

var myIdx *indexer.MySQLIndexer
var myMets *metrics.MySQLFlatMetrics
var srv *server.Server
var staged bool = false
var maxMets = 1000
var numWords = 5
var someMets = []string{}

func randDotString(n int) string {
	words := []string{"server", "host", "cpu", "ram", "free", "load", "house"}
	out := []string{}
	for i := 0; i < n; i++ {
		out = append(out, words[rand.Intn(len(words))])
	}
	return strings.Join(out, ".")
}
func randStat(keylen int) *repr.StatRepr {
	rnum := rand.Float64() * 100
	return &repr.StatRepr{
		Name: &repr.StatName{
			Key:        randDotString(keylen),
			Resolution: 1,
		},
		Time:  time.Now().UnixNano(),
		Count: 1,
		Min:   rnum,
		Max:   rnum,
		Sum:   rnum,
		Last:  rnum,
	}
}

func getMySQL() (*indexer.MySQLIndexer, *metrics.MySQLFlatMetrics, error) {
	if myIdx != nil && myMets != nil {
		return myIdx, myMets, nil
	}
	//onIp := helper.DockerIp()

	config_opts := options.New()
	config_opts.Set("dsn", fmt.Sprintf("root:password@tcp(%s:%d)/cadent", "localhost", myPort))
	config_opts.Set("table", "test_mets")
	config_opts.Set("path_table", "test_path")
	config_opts.Set("segment_table", "test_seg")
	config_opts.Set("tag_table", "test_tag")
	config_opts.Set("id_table", "text_ids")
	config_opts.Set("resolution", 1)
	config_opts.Set("periodic_flush", "1ms")

	myIdx = indexer.NewMySQLIndexer()
	err := myIdx.Config(&config_opts)
	if err != nil {
		return nil, nil, err
	}

	myMets = metrics.NewMySQLFlatMetrics()
	err = myMets.Config(&config_opts)
	if err != nil {
		return nil, nil, err
	}
	myMets.SetCurrentResolution(1)
	myMets.SetResolutions([][]int{{1, 0}})

	return myIdx, myMets, nil
}

func getDocker(t *testing.T) {
	// fire up the docker test helper
	helper.DockerUp("mysql")
	ok := helper.DockerWaitUntilReady("mysql")
	if !ok {
		t.Fatalf("Could not start the docker container for mysql")
	}
}

// needed to set up all the pieces, add some metrics, etc
func stageEnv(t *testing.T) {

	if staged {
		return
	}
	getDocker(t)
	_, _, err := getMySQL()
	if err != nil {
		t.Fatalf("%v", err)
	}

	// start up the two items to get schemes in the mix
	myIdx.Start()
	myMets.Start()
	myMets.SetIndexer(myIdx)

	// add some data
	for i := 0; i < maxMets; i++ {
		st := randStat(numWords)
		someMets = append(someMets, st.Name.Key)
		myMets.Write(st)
		myIdx.WriteOne(st.Name) // force a write here to bypass the actual queuing mechanism
		time.Sleep(time.Millisecond)
	}

	startServer(t)

	staged = true

}

func startServer(t *testing.T) {
	if srv != nil {
		return
	}

	srv, _ = server.New()
	srv.Listen = "0.0.0.0:0"
	srv.Indexer = myIdx
	srv.Metrics = myMets
	srv.Start()
	t.Log("New gRPC SERVER on: ", srv.ListenAddr)

}

func Test_______Stage(t *testing.T) {
	stageEnv(t)

	got, err := myIdx.Find("*", nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) == 0 {
		t.Fatal("No data found for find, fail :(")
	}

	got, err = myIdx.Find(someMets[0], nil)
	t.Log(got, someMets[0])
	if err != nil {
		t.Fatal(err)
	}

	if len(got) == 0 {
		t.Fatal("No data found for find, fail :(")
	}

}

func Test_______NewServer(t *testing.T) {
	stageEnv(t)
}

func Test_______PutRawMetric(t *testing.T) {
	stageEnv(t)

	st := randStat(numWords)
	m := &series.RawMetric{
		Metric:   st.Name.Key,
		Tags:     st.Name.Tags,
		MetaTags: st.Name.MetaTags,
		Value:    st.Sum,
		Time:     st.Time,
	}

	cli, err := New(srv.ListenAddr.String(), nil, "", nil)
	if err != nil {
		t.Fatal(err)
	}

	err = cli.PutRawMetric(m)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_______PutStatRepr(t *testing.T) {
	stageEnv(t)

	st := randStat(numWords)

	cli, err := New(srv.ListenAddr.String(), nil, "", nil)
	if err != nil {
		t.Fatal(err)
	}

	err = cli.PutStatRepr(st)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_______FindMetrics(t *testing.T) {
	stageEnv(t)

	q := api.IndexQuery{
		Query: someMets[0],
	}

	cli, err := New(srv.ListenAddr.String(), nil, "", nil)
	if err != nil {
		t.Fatal(err)
	}

	found, err := cli.FindMetrics(&q)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Index Find: ", len(found))
}

func Test_______FindMetricsChan(t *testing.T) {
	stageEnv(t)

	spl := strings.Split(someMets[0], ".")
	spl[len(spl)-1] = "*"
	m := strings.Join(spl, ".")

	q := api.IndexQuery{
		Query: m,
	}

	cli, err := New(srv.ListenAddr.String(), nil, "", nil)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan interface{})
	rcv := make(chan *sindexer.MetricFindItem)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		for {
			select {
			case met := <-rcv:
				t.Log("FIND FOUND: ", met.Path)
			case <-done:
				wg.Done()
				return
			}

		}
	}()
	ctx := context.Background()
	err = cli.FindMetricsChan(ctx, &q, rcv, done)
	wg.Wait()
	if err != nil {
		t.Fatal(err)
	}

}

func Test_______GetMetrics(t *testing.T) {
	stageEnv(t)

	end := time.Now().Unix()
	start := end - 360

	q := api.MetricQuery{
		Target: someMets[0],
		Start:  start,
		End:    end,
	}

	cli, err := New(srv.ListenAddr.String(), nil, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	found, err := cli.GetMetrics(ctx, &q)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("FOUND: ", len(found))
}

func Test_______GetMetricsChan(t *testing.T) {
	stageEnv(t)

	end := time.Now().Unix()
	start := end - 360

	spl := strings.Split(someMets[0], ".")
	spl[len(spl)-1] = "*"
	m := strings.Join(spl, ".")

	q := api.MetricQuery{
		Target: m,
		Start:  start,
		End:    end,
	}

	cli, err := New(srv.ListenAddr.String(), nil, "", nil)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan interface{})
	rcv := make(chan *smetrics.RawRenderItem)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		for {
			select {
			case met := <-rcv:
				t.Log("FOUND: ", met.Metric, len(met.Data))
			case <-done:
				wg.Done()
				return
			}

		}
	}()
	ctx := context.Background()
	err = cli.GetMetricsChan(ctx, &q, rcv, done)
	wg.Wait()
	if err != nil {
		t.Fatal(err)
	}

}
