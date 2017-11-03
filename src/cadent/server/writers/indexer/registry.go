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

package indexer

import (
	"errors"
	"fmt"
	"sync"
)

// ErrorAlreadyRegistered writer already registered
var ErrorAlreadyRegistered = errors.New("Indexer is already registered")

var indexerReg map[string]Indexer
var idxLock sync.RWMutex

func init() {
	indexerReg = make(map[string]Indexer)
}

func RegisterIndexer(name string, idx Indexer) error {
	idxLock.Lock()
	defer idxLock.Unlock()
	if _, ok := indexerReg[name]; ok {
		return ErrorAlreadyRegistered
	}
	indexerReg[name] = idx
	return nil
}

func GetIndexer(name string) Indexer {
	idxLock.RLock()
	defer idxLock.RUnlock()
	return indexerReg[name]
}

func NewIndexer(name string) (Indexer, error) {
	switch {
	case name == "cassandra":
		return NewCassandraIndexer(), nil
	case name == "ram":
		return NewRamIndexer(), nil
	case name == "whisper":
		return NewWhisperIndexer(), nil
	case name == "kafka":
		return NewKafkaIndexer(), nil
	case name == "mysql":
		return NewMySQLIndexer(), nil
	case name == "leveldb":
		return NewLevelDBIndexer(), nil
	case name == "elasticsearch":
		return NewElasticIndexer(), nil
	case name == "noop":
		return NewNoopIndexer(), nil
	default:
		return nil, fmt.Errorf("Invalid indexer `%s`", name)
	}
}
