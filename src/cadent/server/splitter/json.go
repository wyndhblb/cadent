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
   a raw json splitter.  which is not really a "splitter" but just a data struct around
   an incoming json blob
*/

package splitter

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

/****************** RUNNERS *********************/
const JSON_SPLIT_NAME = "json"

var errBadJsonLine = errors.New("Invalid Json object")

/* JsonStructSplitItem a json input handler format (like that of openTSDB)
{
metric: xxx,
value: 123,
timestamp: 123123,
tags: {
 moo: goo,
 foo: bar
 }
}
*/
type JsonStructSplitItem struct {
	Metric string            `json:"metric"`
	Value  float64           `json:"value"`
	Time   int64             `json:"timestamp"`
	Tags   map[string]string `json:"tags,omitempty"`
}

// JsonSplitItem the split item interface implementation
type JsonSplitItem struct {
	inkey  []byte
	inline []byte
	intime time.Time
	// XXX NOT USED large GC pressure, so kill it infields [][]byte
	inphase  Phase
	inorigin Origin
	inoname  string
	tags     [][][]byte
}

func (g *JsonSplitItem) Key() []byte {
	return g.inkey
}

func (g *JsonSplitItem) Tags() [][][]byte {
	return g.tags
}

func (g *JsonSplitItem) HasTime() bool {
	return true
}

func (g *JsonSplitItem) Timestamp() time.Time {
	return g.intime
}

func (g *JsonSplitItem) Line() []byte {
	return g.inline
}

/* XXX NOT USED large GC pressure, so kill it
func (g *JsonSplitItem) Fields() [][]byte {
	return g.infields
}
*/

func (g *JsonSplitItem) Phase() Phase {
	return g.inphase
}

func (g *JsonSplitItem) SetPhase(n Phase) {
	g.inphase = n
}

func (g *JsonSplitItem) Origin() Origin {
	return g.inorigin
}

func (g *JsonSplitItem) SetOrigin(n Origin) {
	g.inorigin = n
}

func (g *JsonSplitItem) OriginName() string {
	return g.inoname
}

func (g *JsonSplitItem) SetOriginName(n string) {
	g.inoname = n
}

func (g *JsonSplitItem) IsValid() bool {
	return len(g.inkey) > 0
}

func (g *JsonSplitItem) String() string {
	return fmt.Sprintf("Splitter: Json: %s @ %d", string(g.Key()), g.intime.Unix())
}

// splitter

type JsonSplitter struct{}

func (g *JsonSplitter) Name() (name string) { return JSON_SPLIT_NAME }

func NewJsonSplitter(conf map[string]interface{}) (*JsonSplitter, error) {
	job := &JsonSplitter{}
	return job, nil
}

func (g *JsonSplitter) ParseJson(jItem *JsonStructSplitItem) (SplitItem, error) {
	if len(jItem.Metric) == 0 {
		return nil, errBadJsonLine
	}

	splItem := getJsonItem()
	splItem.inkey = []byte(jItem.Metric)

	// nano or second tstamps
	if jItem.Time > 2147483647 {
		splItem.intime = time.Unix(0, jItem.Time)
	} else {
		splItem.intime = time.Unix(jItem.Time, 0)
	}
	/* XXX NOT USED large GC pressure, so kill it
	spl_item.infields = append(spl_item.infields, spl_item.inkey)
	spl_item.infields = append(spl_item.infields, []byte(fmt.Sprintf("%v", j_item.Value)))
	spl_item.infields = append(spl_item.infields, []byte(fmt.Sprintf("%v", j_item.Time)))
	*/
	if len(jItem.Tags) > 0 {

		otags := make([][][]byte, 0)
		for nm, val := range jItem.Tags {
			otags = append(otags, [][]byte{[]byte(nm), []byte(val)})
		}

		splItem.tags = otags
	}

	// need to "squeeze" the air out of the incoming
	var err error
	splItem.inline, err = json.Marshal(jItem)

	//back in the pool
	putStructJsonItem(jItem)

	return splItem, err
}

// array parser
func (g *JsonSplitter) ProcessLines(line []byte) ([]SplitItem, error) {
	var jItems []JsonStructSplitItem
	err := json.Unmarshal(line, &jItems)
	if err != nil {
		return nil, err
	}

	if len(jItems) == 0 {
		return nil, errBadJsonLine
	}

	var jsonitems []SplitItem

	for _, item := range jItems {
		j, err := g.ParseJson(&item)
		if err != nil {
			return jsonitems, err
		}
		jsonitems = append(jsonitems, j)
	}
	return jsonitems, nil
}

func (g *JsonSplitter) ProcessLine(line []byte) (SplitItem, error) {
	j_item := getStructJsonItem()

	err := json.Unmarshal(line, j_item)
	if err != nil {
		return nil, err
	}

	return g.ParseJson(j_item)

}

var jsonStructItemPool sync.Pool

func getStructJsonItem() *JsonStructSplitItem {
	x := jsonStructItemPool.Get()
	if x == nil {
		return new(JsonStructSplitItem)
	}
	// need to mull it out
	x.(*JsonStructSplitItem).Metric = ""
	x.(*JsonStructSplitItem).Tags = make(map[string]string)
	x.(*JsonStructSplitItem).Time = 0
	x.(*JsonStructSplitItem).Value = 0
	return x.(*JsonStructSplitItem)
}

func putStructJsonItem(spl *JsonStructSplitItem) {
	jsonStructItemPool.Put(spl)
}

var jsonItemPool sync.Pool

func getJsonItem() *JsonSplitItem {
	x := jsonItemPool.Get()
	if x == nil {
		return new(JsonSplitItem)
	}
	return x.(*JsonSplitItem)
}

func putJsonItem(spl *JsonSplitItem) {
	jsonItemPool.Put(spl)
}
