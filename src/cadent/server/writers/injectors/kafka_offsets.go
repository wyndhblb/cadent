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
   Kafka a helper class that aids w/ the written max offsets as we write things

   maintains a [metricUID] -> {offset, topic, parition} map
*/

package injectors

import (
	"sync"
)

// an object that keeps track of which topic/partition/offset we are on for a given
// "id"
type offsetMarker struct {
	Id        string
	Topic     string
	Partition int32
	Offset    int64
	WriteTime int64
}

type offsetMap struct {
	offsets map[string]*offsetMarker
	ofLock  sync.RWMutex
}

// AddIfBigger add the mark only if the offset is bigger
func (o *offsetMap) AddIfBigger(omark *offsetMarker) {
	o.ofLock.Lock()
	defer o.ofLock.Unlock()
	if item, ok := o.offsets[omark.Id]; ok {
		if omark.Offset > item.Offset {
			o.offsets[omark.Id].Offset = omark.Offset
			return
		}
	}
	o.offsets[omark.Id] = omark
}

// Get an offset mark
func (o *offsetMap) Get(id string) *offsetMarker {
	o.ofLock.RLock()
	defer o.ofLock.RUnlock()
	if gt, ok := o.offsets[id]; ok {
		return gt
	}
	return nil
}

// IsLower is the id/offset combo "lower" then the cached value
func (o *offsetMap) IsLower(id string, inoffset int64) bool {
	o.ofLock.RLock()
	defer o.ofLock.RUnlock()
	if gt, ok := o.offsets[id]; ok {
		return gt.Offset > inoffset
	}
	return false
}

// UpdateOffsetIfBigger update the offset value if bigger
func (o *offsetMap) UpdateOffsetIfBigger(id string, inoffset int64) bool {
	o.ofLock.RLock()
	defer o.ofLock.RUnlock()

	if _, ok := o.offsets[id]; ok {
		if inoffset > o.offsets[id].Offset {
			o.offsets[id].Offset = inoffset
			return true
		}
		return false
	}
	return false
}

// PartitionMinOffset call this lightly can be a bit expensive, meant as an initialization step
func (o *offsetMap) PartitionMinOffset(topic string, partition int32) int64 {
	o.ofLock.RLock()
	var minO int64 = -2
	for _, of := range o.offsets {
		if of.Partition == partition && of.Topic == topic {
			if (minO == -2 || minO > of.Offset) && of.Offset > 0 {
				minO = of.Offset
			}
		}
	}
	o.ofLock.RUnlock()
	return minO
}

// initialize this big-old-singleton
var kafkaOffsetMap *offsetMap

func init() {
	kafkaOffsetMap = new(offsetMap)
	kafkaOffsetMap.offsets = make(map[string]*offsetMarker)
}
