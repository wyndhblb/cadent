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
   statsd Runner and parser <key>:<value>|<type>
*/

package splitter

import (
	"cadent/server/schemas/repr"
	"fmt"
	"strings"
	"sync"
	"time"
)

const STATSD_NAME = "statsd"

type StatsdSplitItem struct {
	inkey  []byte
	inline []byte
	// XXX NOT USED infields [][]byte
	inphase  Phase
	inorigin Origin
	inoname  string
	tags     [][][]byte
}

func (g *StatsdSplitItem) Key() []byte {
	return g.inkey
}

func (g *StatsdSplitItem) HasTime() bool {
	return false
}

func (g *StatsdSplitItem) Tags() [][][]byte {
	return g.tags
}

func (g *StatsdSplitItem) Timestamp() time.Time {
	return time.Time{}
}

func (g *StatsdSplitItem) Line() []byte {
	return g.inline
}

/* XXX NOT USE GC PRESSURE OPTIMIZATION
func (g *StatsdSplitItem) Fields() [][]byte {
	return g.infields
}
*/

func (g *StatsdSplitItem) Phase() Phase {
	return g.inphase
}

func (g *StatsdSplitItem) SetPhase(n Phase) {
	g.inphase = n
}

func (g *StatsdSplitItem) Origin() Origin {
	return g.inorigin
}

func (g *StatsdSplitItem) SetOrigin(n Origin) {
	g.inorigin = n
}

func (g *StatsdSplitItem) OriginName() string {
	return g.inoname
}

func (g *StatsdSplitItem) SetOriginName(n string) {
	g.inoname = n
}
func (g *StatsdSplitItem) IsValid() bool {
	return len(g.inline) > 0
}

func (job *StatsdSplitItem) String() string {
	return fmt.Sprintf("Splitter: Statsd: %s", job.inkey)
}

type StatsdSplitter struct {
}

func (job *StatsdSplitter) Name() (name string) { return STATSD_NAME }

func NewStatsdSplitter(conf map[string]interface{}) (*StatsdSplitter, error) {

	//<key>:blaaa
	job := &StatsdSplitter{}
	return job, nil
}

func (job *StatsdSplitter) ProcessLine(line []byte) (SplitItem, error) {

	inL := string(line)
	statd_array := strings.Split(inL, repr.COLON_SEPARATOR)

	/* XXX NOT USED ANYWHERE, large GC drain so kill it
	fs := [][]byte{}
	for _, j := range statd_array {
		fs = append(fs, []byte(j))
	}*/

	if len(statd_array) >= 2 {
		si := getStatsdItem()
		si.inkey = []byte(statd_array[0])
		// seems silly but we need to copy the origin line otherwise the
		// incoming slice can get mangled by later things and effect this "pointer" to things
		si.inline = make([]byte, len(line))
		copy(si.inline, line) // we need to copy the origin line
		// XXX NOT USED DROP IT si.infields = fs
		si.inphase = Parsed
		si.inorigin = Other

		return si, nil
	}
	return nil, fmt.Errorf("Invalid Statsd line: " + string(line))

}

var statsdItemPool sync.Pool

func getStatsdItem() *StatsdSplitItem {
	x := statsdItemPool.Get()
	if x == nil {
		return new(StatsdSplitItem)
	}
	return x.(*StatsdSplitItem)
}

func putStatsdItem(spl *StatsdSplitItem) {
	statsdItemPool.Put(spl)
}
