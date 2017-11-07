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
   Simple make of new objects
*/

package metrics

import (
	"errors"
	"fmt"
	"sync"
)

// cache configs are required for some metric writers
var errMetricsCacheRequired = errors.New("Cache object is required")

// /cached/series endpoint cannot have multi targets
var errMultiTargetsNotAllowed = errors.New("Multiple Targets are not allowed")

// somehow a nil name
var errNameIsNil = errors.New("Name object cannot be nil")

// somehow a nil series
var errSeriesIsNil = errors.New("Name object cannot be nil")

// ErrorAlreadyRegistered writer already registered
var ErrorAlreadyRegistered = errors.New("Writer is already registered")

// Metrics registry by name
var metricReg map[string]Metrics
var metrLock sync.RWMutex

func init() {
	metricReg = make(map[string]Metrics)
}

func RegisterMetrics(name string, idx Metrics) error {
	metrLock.Lock()
	defer metrLock.Unlock()
	if _, ok := metricReg[name]; ok {
		return ErrorAlreadyRegistered
	}
	metricReg[name] = idx
	return nil
}

func GetMetrics(name string) Metrics {
	metrLock.RLock()
	defer metrLock.RUnlock()
	return metricReg[name]
}

func NewWriterMetrics(name string) (Metrics, error) {
	switch {
	case name == "mysql":
		return NewMySQLMetrics(), nil
	case name == "mysql-triggered":
		return NewMySQLTriggeredMetrics(), nil
	case name == "mysql-flat":
		return NewMySQLFlatMetrics(), nil
	case name == "file":
		return NewCSVFileMetrics(), nil
	case name == "cassandra":
		return NewCassandraMetrics(), nil
	case name == "cassandra-triggered":
		return NewCassandraTriggerMetrics(), nil
	case name == "cassandra-log":
		return NewCassandraLogMetrics(), nil
	case name == "cassandra-log-triggered":
		return NewCassandraLogTriggerMetrics(), nil
	case name == "ram-log":
		return NewRamLogMetrics(), nil
	case name == "leveldb-log" || name == "leveldb-log-triggered":
		return NewLevelDBLogMetrics(), nil
	case name == "cassandra-flat":
		return NewCassandraFlatMetrics(), nil
	case name == "cassandra-flat-map":
		return NewCassandraFlatMapMetrics(), nil
	case name == "elasticsearch-flat-map":
		return NewElasticSearchFlatMapMetrics(), nil
	case name == "elasticsearch-flat" || name == "elastic-flat":
		return NewElasticSearchFlatMetrics(), nil
	case name == "whisper" || name == "carbon" || name == "graphite":
		return NewWhisperMetrics(), nil
	case name == "kafka":
		return NewKafkaMetrics(), nil
	case name == "kafka-flat":
		return NewKafkaFlatMetrics(), nil
	case name == "redis-flat" || name == "redis-flat-map":
		return NewRedisFlatMetrics(), nil
	case name == "echo":
		return NewEchoMetrics(), nil
	default:
		return nil, fmt.Errorf("Invalid metrics driver `%s`", name)
	}
}

func ResolutionsNeeded(name string) (WritersNeeded, error) {
	switch {
	case name == "mysql":
		return AllResolutions, nil
	case name == "mysql-flat":
		return AllResolutions, nil
	case name == "file":
		return AllResolutions, nil
	case name == "cassandra":
		return AllResolutions, nil
	case name == "cassandra-flat":
		return AllResolutions, nil
	case name == "cassandra-flat-map":
		return AllResolutions, nil
	case name == "elasticsearch-flat-map":
		return AllResolutions, nil
	case name == "echo":
		return AllResolutions, nil
	case name == "elasticsearch-flat" || name == "elastic-flat":
		return AllResolutions, nil
	case name == "kafka" || name == "kafka-flat" || name == "ram-log":
		return AllResolutions, nil
	case name == "whisper" || name == "carbon" || name == "ram-log-triggered" || name == "graphite" ||
		name == "cassandra-triggered" || name == "mysql-triggered" || name == "cassandra-log-triggered" ||
		name == "leveldb-log" || name == "leveldb-log-triggered":
		return FirstResolution, nil
	default:
		return AllResolutions, fmt.Errorf("Invalid metrics driver `%s`", name)
	}
}

func CacheNeeded(name string) (CacheTypeNeeded, error) {
	switch {
	case name == "cassandra-log" || name == "cassandra-log-triggered":
		return Chunked, nil
	case name == "leveldb-log" || name == "leveldb-log-triggered":
		return Chunked, nil
	case name == "ram-log" || name == "ram-log-triggered":
		return Chunked, nil
	default:
		return Single, nil
	}
}

func CacheNeededName(name string) string {
	switch {
	case name == "cassandra-log" || name == "cassandra-log-triggered":
		return "chunk"
	case name == "leveldb-log" || name == "leveldb-log-triggered":
		return "chunk"
	case name == "ram-log" || name == "ram-log-triggered":
		return "chunk"
	default:
		return "single"
	}
}
