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
	THe ElasticSearch write

*/

package dbs

import (
	"cadent/server/utils/options"
	"fmt"
	es5 "gopkg.in/olivere/elastic.v5"
	"gopkg.in/op/go-logging.v1"
	"net/http"
	"strings"
)

type ElasticSearchLog struct{}

func (el *ElasticSearchLog) Printf(str string, args ...interface{}) {
	fmt.Printf(str, args...)
}

/****************** Interfaces *********************/
type ElasticSearch struct {
	Client       *es5.Client
	metricIndex  string
	pathIndex    string
	segmentIndex string
	tagIndex     string

	PathType    string
	TagType     string
	SegmentType string
	MetricType  string
	log         *logging.Logger
}

func NewElasticSearch() *ElasticSearch {
	my := new(ElasticSearch)
	my.log = logging.MustGetLogger("writers.elastic")
	my.PathType = "path"
	my.TagType = "tag"
	my.SegmentType = "segment"
	my.MetricType = "metric"

	return my
}

func (my *ElasticSearch) Config(conf *options.Options) (err error) {
	dsn, err := conf.StringRequired("dsn")
	if err != nil {
		return fmt.Errorf("`dsn` http://host:port,http://host:port is needed for elatic search config")
	}

	opsFuncs := make([]es5.ClientOptionFunc, 0)
	opsFuncs = append(opsFuncs, es5.SetURL(strings.Split(dsn, ",")...))
	opsFuncs = append(opsFuncs, es5.SetMaxRetries(10))
	opsFuncs = append(opsFuncs, es5.SetSniff(conf.Bool("sniff", false)))

	// gzip is bad for all the things at large volumes
	opsFuncs = append(opsFuncs, es5.SetGzip(false))
	opsFuncs = append(opsFuncs, es5.SetHttpClient(&http.Client{Transport: &http.Transport{DisableCompression: true}}))

	user := conf.String("user", "")
	pass := conf.String("password", "")
	if user != "" {
		opsFuncs = append(opsFuncs, es5.SetBasicAuth(user, pass))
	}

	if conf.Bool("enable_traceing", false) {
		opsFuncs = append(opsFuncs, es5.SetTraceLog(new(ElasticSearchLog)))
	}

	my.Client, err = es5.NewClient(opsFuncs...)

	if err != nil {
		if conf.Bool("sniff", true) {
			my.log.Errorf("You may wish to turn off sniffing `sniff=false` if the nodes are behind a loadbalencer")
		}
		my.log.Critical("Unable to connect: %s (%s)", dsn, err)

		return err
	}
	my.metricIndex = conf.String("metric_index", "metrics")
	my.pathIndex = conf.String("path_index", "metric_path")
	my.segmentIndex = conf.String("segment_index", "metric_segment")
	my.tagIndex = conf.String("tag_index", "metric_tag")

	return nil
}

func (my *ElasticSearch) Tablename() string {
	return my.metricIndex
}

func (my *ElasticSearch) PathTable() string {
	return my.pathIndex
}

func (my *ElasticSearch) SegmentTable() string {
	return my.segmentIndex
}

func (my *ElasticSearch) TagTable() string {
	return my.tagIndex
}

func (my *ElasticSearch) Connection() DBConn {
	return my.Client
}
