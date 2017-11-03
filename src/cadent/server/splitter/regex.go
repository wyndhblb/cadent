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
   a Regex runner that takes an arbitrary regex to yank it's key from
*/

package splitter

import (
	"fmt"
	"regexp"
	"sync"
	"time"
)

const REGEX_NAME = "regex"

type RegexSplitItem struct {
	inkey    []byte
	inline   []byte
	intime   time.Time
	regexed  [][][]byte
	inphase  Phase
	inorigin Origin
	inoname  string
	tags     [][][]byte
}

func (g *RegexSplitItem) Key() []byte {
	return g.inkey
}

func (g *RegexSplitItem) Tags() [][][]byte {
	return g.tags
}

func (g *RegexSplitItem) Line() []byte {
	return g.inline
}

func (g *RegexSplitItem) HasTime() bool {
	return g.intime.IsZero()
}

func (g *RegexSplitItem) Timestamp() time.Time {
	return g.intime
}

/* XXX NOT USED large GC pressure, so kill it
func (g *RegexSplitItem) Fields() [][]byte {
	return g.regexed[0]
}
*/

func (g *RegexSplitItem) Phase() Phase {
	return g.inphase
}
func (g *RegexSplitItem) SetPhase(n Phase) {
	g.inphase = n
}

func (g *RegexSplitItem) Origin() Origin {
	return g.inorigin
}

func (g *RegexSplitItem) SetOrigin(n Origin) {
	g.inorigin = n
}

func (g *RegexSplitItem) OriginName() string {
	return g.inoname
}

func (g *RegexSplitItem) SetOriginName(n string) {
	g.inoname = n
}

func (g *RegexSplitItem) IsValid() bool {
	return len(g.inline) > 0
}

type RegExSplitter struct {
	key_regex       *regexp.Regexp
	key_regex_names []string
	key_index       int
	time_index      int
	time_layout     string
}

func (job *RegExSplitter) Name() (name string) { return REGEX_NAME }

func NewRegExSplitter(conf map[string]interface{}) (*RegExSplitter, error) {

	//<key>:<timestamp>:blaaa
	job := &RegExSplitter{}
	job.key_regex = conf["regexp"].(*regexp.Regexp)
	job.key_regex_names = conf["regexpNames"].([]string)

	// see if there is a time_layout
	if val, ok := conf["timeLayout"]; ok {
		job.time_layout = val.(string)
	}
	// get the "Key" index for easy lookup
	for i, n := range job.key_regex_names {
		if n == "Key" {
			job.key_index = i
		}
		if n == "Timestamp" {
			job.time_index = i
		}
	}

	//job.key_regex_names := job.key_regex.SubexpNames()
	return job, nil
}

func (job *RegExSplitter) ProcessLine(line []byte) (SplitItem, error) {
	var key_param, time_param []byte

	matched := job.key_regex.FindAllSubmatch(line, -1)
	if matched == nil {
		return nil, fmt.Errorf("Regex not matched")
	}
	if len(matched[0]) < (job.key_index + 1) {
		return nil, fmt.Errorf("Named matches not found")
	}
	key_param = matched[0][job.key_index+1] // first string is always the original string

	t := time.Time{}
	if job.time_index > 0 && len(job.time_layout) > 0 {
		time_param = matched[0][job.time_index+1]
		_time, err := time.Parse(job.time_layout, string(time_param))
		if err == nil {
			t = _time
		}
	}

	if len(key_param) > 0 {
		ri := getRegexItem()
		ri.inkey = key_param
		ri.inline = line
		ri.intime = t
		ri.regexed = matched
		ri.inphase = Parsed
		ri.inorigin = Other

		return ri, nil
	}
	return nil, fmt.Errorf("Invalid Regex (cannot find key) line")

}

var regexItemPool sync.Pool

func getRegexItem() *RegexSplitItem {
	x := regexItemPool.Get()
	if x == nil {
		return new(RegexSplitItem)
	}
	return x.(*RegexSplitItem)
}

func putRegexItem(spl *RegexSplitItem) {
	regexItemPool.Put(spl)
}
