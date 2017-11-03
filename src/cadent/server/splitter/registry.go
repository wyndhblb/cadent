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
   New maker of splitters
*/

package splitter

import (
	"fmt"
)

// note opentsb is not here is it makes a "graphitesplititem" when it's done
func PutSplitItem(item SplitItem) {
	switch item.(type) {
	case *GraphiteSplitItem:
		putGraphiteItem(item.(*GraphiteSplitItem))
	case *StatsdSplitItem:
		putStatsdItem(item.(*StatsdSplitItem))
	case *CarbonTwoSplitItem:
		putCarbonTwoItem(item.(*CarbonTwoSplitItem))
	case *RegexSplitItem:
		putRegexItem(item.(*RegexSplitItem))
	case *JsonSplitItem:
		putJsonItem(item.(*JsonSplitItem))
	}

}

func NewSplitterItem(name string, conf map[string]interface{}) (Splitter, error) {
	switch {
	case name == "statsd":
		return NewStatsdSplitter(conf)
	case name == "graphite":
		return NewGraphiteSplitter(conf)
	case name == "carbon2":
		return NewCarbonTwoSplitter(conf)
	case name == "regex":
		return NewRegExSplitter(conf)
	case name == "opentsdb":
		return NewOpenTSDBSplitter(conf)
	case name == "json":
		return NewJsonSplitter(conf)
	default:
		return nil, fmt.Errorf("Invalid splitter `%s`", name)
	}
}
