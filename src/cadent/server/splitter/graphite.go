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
   Graphite data runner, <key> <value> <time> <things> ...
   we allow there to be more <things> after value, so this is really "graphite style"
   space based line entries with the key as the first item
*/

package splitter

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

/****************** RUNNERS *********************/
const GRAPHITE_NAME = "graphite"

var ErrBadGraphiteLine = errors.New("Invalid Graphite/Space line")
var ErrBadGraphiteLineTime = errors.New("Invalid Graphite Time Field")

// fail if time field is too big or too small from the current day (a year)
const MinTimeDelta = 362 * 24 * 60 * 60

var grBytesPool sync.Pool

// pools of preallocated byte arrays
func grGetBytes() []byte {
	x := grBytesPool.Get()
	if x == nil {
		buf := make([]byte, 32)
		return buf[:0]
	}
	buf := x.([]byte)
	return buf[:0]
}

func grPutBytes(buf []byte) {
	grBytesPool.Put(buf)
}

var GRAPHITE_REPLACER *strings.Replacer
var GRAPHITE_REPLACER_BYTES = [][][]byte{
	{[]byte(".."), []byte(".")},
	{[]byte("=="), []byte("=")},
	{[]byte(","), []byte("_")},
	{[]byte("*"), []byte("_")},
	{[]byte("("), []byte("_")},
	{[]byte(")"), []byte("_")},
	{[]byte("{"), []byte("_")},
	{[]byte("}"), []byte("_")},
	{[]byte("  "), []byte(" ")},
}

var GRAPHITE_REPLACER_BYTE_MAP map[byte]byte

func init() {
	GRAPHITE_REPLACER = strings.NewReplacer(
		"..", ".",
		",", "_",
		"==", "=",
		"*", "_",
		"(", "_",
		")", "_",
		"{", "_",
		"}", "_",
		"  ", " ",
	)
	GRAPHITE_REPLACER_BYTE_MAP := make(map[byte]byte)
	GRAPHITE_REPLACER_BYTE_MAP[','] = '_'
	GRAPHITE_REPLACER_BYTE_MAP['*'] = '_'
	GRAPHITE_REPLACER_BYTE_MAP['{'] = '_'
	GRAPHITE_REPLACER_BYTE_MAP['}'] = '_'
	GRAPHITE_REPLACER_BYTE_MAP['('] = '_'
	GRAPHITE_REPLACER_BYTE_MAP[')'] = '_'
	GRAPHITE_REPLACER_BYTE_MAP[' '] = '_'
}

func graphiteByteReplace(bs []byte) []byte {
	bs = bytes.TrimSpace(bs)
	for i, b := range bs {
		if g, ok := GRAPHITE_REPLACER_BYTE_MAP[b]; ok {
			bs[i] = g
		}
	}
	bs = bytes.Replace(bs, []byte(".."), []byte("."), -1)
	bs = bytes.Replace(bs, []byte("=="), []byte("="), -1)
	return bs
}

type GraphiteSplitItem struct {
	inkey  []byte
	inline []byte
	intime time.Time
	// XXX NOT USED large GC pressure, so kill it infields [][]byte
	inphase  Phase
	inorigin Origin
	inoname  string
	tags     [][][]byte
}

func (g *GraphiteSplitItem) Key() []byte {
	return g.inkey
}

func (g *GraphiteSplitItem) Tags() [][][]byte {
	return g.tags
}

func (g *GraphiteSplitItem) HasTime() bool {
	return true
}

func (g *GraphiteSplitItem) Timestamp() time.Time {
	return g.intime
}

func (g *GraphiteSplitItem) Line() []byte {
	return g.inline
}

/* XXX NOT USED large GC pressure, so kill it
func (g *GraphiteSplitItem) Fields() [][]byte {
	return g.infields
}
*/

func (g *GraphiteSplitItem) Phase() Phase {
	return g.inphase
}

func (g *GraphiteSplitItem) SetPhase(n Phase) {
	g.inphase = n
}

func (g *GraphiteSplitItem) Origin() Origin {
	return g.inorigin
}

func (g *GraphiteSplitItem) SetOrigin(n Origin) {
	g.inorigin = n
}

func (g *GraphiteSplitItem) OriginName() string {
	return g.inoname
}

func (g *GraphiteSplitItem) SetOriginName(n string) {
	g.inoname = n
}

func (g *GraphiteSplitItem) IsValid() bool {
	return len(g.inline) > 0
}

func (g *GraphiteSplitItem) String() string {
	return fmt.Sprintf("Splitter: Graphite: %s @ %s", g.inkey, g.intime)
}

type GraphiteSplitter struct {
	keyIndex      int
	timeIndex     int
	failOnBadTime bool
}

func (g *GraphiteSplitter) Name() (name string) { return GRAPHITE_NAME }

func NewGraphiteSplitter(conf map[string]interface{}) (*GraphiteSplitter, error) {

	//<key> <value> <time> <things>
	job := &GraphiteSplitter{
		keyIndex:      0,
		timeIndex:     2,
		failOnBadTime: true,
	}
	// allow for a config option to pick the proper thing in the line
	if idx, ok := conf["key_index"].(int); ok {
		job.keyIndex = idx
	}
	if idx, ok := conf["time_index"].(int); ok {
		job.timeIndex = idx
	}

	if idx, ok := conf["fail_on_bad_time"].(bool); ok {
		job.failOnBadTime = idx
	}
	return job, nil
}

func (g *GraphiteSplitter) ProcessLine(line []byte) (SplitItem, error) {
	//<key> <value> <time> <more> <more>
	/*

		HMM the below is doing some really weird things under large loads
		not sure why yet so revert it

		"..", ".",
		",", "_",
		"==", "=",
		"*", "_",
		"(", "_",
		")", "_",
		"{", "_",
		"}", "_",
		"  ", " "
	*/

	/*outKey := grGetBytes()
	outTime := grGetBytes()
	outValue := grGetBytes()
	outRest := grGetBytes()
	defer func() {
		grPutBytes(outKey)
		grPutBytes(outTime)
		grPutBytes(outValue)
		grPutBytes(outRest)
	}()*/
	/*
	   	outKey := []byte{}
	   	outTime := []byte{}
	   	outValue := []byte{}
	   	outRest := []byte{}
	   	nLine := make([]byte, len(line))
	   	copy(nLine, line)
	   	var idx int
	   	var b byte
	   	line = bytes.Replace(line, []byte("  "), []byte(" "), -1)
	   	for _, b = range line {
	   		idx++
	   		switch b {
	   		case '{', '}', '(', ')', ',', '*':
	   			outKey = append(outKey, '_')
	   		case ' ':
	   			goto VAL
	   		default:
	   			outKey = append(outKey, b)

	   		}
	   	}

	   VAL:
	   	//value
	   	for _, b = range line[idx:] {
	   		idx++
	   		switch b {
	   		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '.':
	   			outValue = append(outValue, b)
	   		case ' ':
	   			goto TIME
	   		}
	   	}
	   TIME:
	   	//time
	   	for _, b = range line[idx:] {
	   		idx++
	   		switch b {
	   		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '.':
	   			outTime = append(outTime, b)
	   		case ' ':
	   			goto LAST
	   		}
	   	}
	   LAST:
	   	// need to copy this
	   	for _, b = range line[idx:] {
	   		outRest = append(outRest, b)
	   	}

	   	//oddly strings Fields is faster then bytes fields, and we need strings later
	   	//graphiteArray := strings.Fields(string(line))
	   	if len(outKey) > 0 {

	   		// graphite timestamps are in unix seconds
	   		t := time.Time{}
	   		if len(outTime) > 0 {
	   			i, err := strconv.ParseInt(string(outTime), 10, 64)
	   			if err == nil {
	   				// nano or second tstamps
	   				if i > 2147483647 {
	   					t = time.Unix(0, i)
	   				} else {
	   					t = time.Unix(i, 0)
	   				}

	   				if g.failOnBadTime && math.Abs(float64(t.Unix()-time.Now().Unix())) > MinTimeDelta {
	   					return nil, ErrBadGraphiteLineTime
	   				}
	   			} else if err != nil && g.failOnBadTime {
	   				return nil, ErrBadGraphiteLineTime
	   			}
	   		}

	   		gi := getGraphiteItem()
	   		gi.inkey = outKey
	   		gi.inline = gi.inline[:0]
	   		// seems silly but we need to copy the origin line otherwise the
	   		// incoming slice can get mangled by later things and effect this "pointer" to things
	   		for _, b := range line {
	   			gi.inline = append(gi.inline, b)
	   		}
	   		gi.intime = t
	   		if len(outRest) > 0 {
	   			gi.infields = [][]byte{outKey, outValue, outTime, outRest}
	   		} else {
	   			gi.infields = [][]byte{outKey, outValue, outTime}
	   		}
	   		gi.inphase = Parsed
	   		gi.inorigin = Other

	   		return gi, nil
	   	}*/

	// clean out not-so-good chars
	line = graphiteByteReplace(line)
	graphiteArray := strings.Fields(string(line))
	if len(graphiteArray) > g.keyIndex {
		t := time.Time{}
		if len(graphiteArray) > g.timeIndex {
			i, err := strconv.ParseInt(graphiteArray[g.timeIndex], 10, 64)
			if err == nil {
				// nano or second tstamps
				if i > 2147483647 {
					t = time.Unix(0, i)
				} else {
					t = time.Unix(i, 0)
				}
				if g.failOnBadTime && math.Abs(float64(t.Unix()-time.Now().Unix())) > MinTimeDelta {
					return nil, ErrBadGraphiteLineTime
				}
			} else if err != nil && g.failOnBadTime {
				return nil, ErrBadGraphiteLineTime
			}

		}
		/* XXX NOT USED large GC pressure, so kill it
		fs := [][]byte{}
		for _, j := range graphiteArray {
			fs = append(fs, []byte(j))
		}
		*/
		gi := getGraphiteItem()
		gi.inkey = []byte(graphiteArray[g.keyIndex])
		gi.inline = make([]byte, len(line))
		copy(gi.inline, line) // we need to copy the origin line
		gi.intime = t
		// XXX NOT USED large GC pressure, so kill it gi.infields = fs
		gi.inphase = Parsed
		gi.inorigin = Other

		graphiteArray = nil
		return gi, nil
	}
	graphiteArray = nil
	return nil, ErrBadGraphiteLine

}

/*** pools **/
var graphiteItemPool sync.Pool

func getGraphiteItem() *GraphiteSplitItem {
	x := graphiteItemPool.Get()
	if x == nil {
		return new(GraphiteSplitItem)
	}
	return x.(*GraphiteSplitItem)
}

func putGraphiteItem(spl *GraphiteSplitItem) {
	graphiteItemPool.Put(spl)
}
