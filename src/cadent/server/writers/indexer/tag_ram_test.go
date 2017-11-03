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
	For RAM based indexes, we need to get a bit smarter about how we store things
	so we use a graph to "index" things for metric keys

	i.e. if the metric key is `my.metric.is.good` we have a little graph of my -> metric -> is -> good

	Tags are indexed as simple maps, depending on the cardinality this can get a bit big in RAM
		Names: map[Name|string][]string{values}
		Uids: map[Name=Value|string][]{uids}

*/

package indexer

import (
	//. "github.com/smartystreets/goconvey/convey"

	"cadent/server/schemas/repr"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"
)

var testTagsList = [][]string{
	{"moo", "goo"},
	{"dc", "us-east"},
	{"zone", "us-east-1a"},
	{"zone", "us-east-1b"},
	{"zone", "us-east-1c"},
	{"zone", "us-east-1d"},
	{"zone", "us-east-1e"},
	{"zoopy", "loopy"},
	{"type", "counter"},
	{"type", "rate"},
	{"type", "gauge"},
	{"type", "bool"},
	{"type", "rate"},
	{"tpe", "broken"},
	{"dummy", "dummy1"},
	{"dummy1", "dummy1"},
	{"dummy2", "dummy1"},
	{"dummy3", "dummy1"},
	{"dummy4", "dummy1"},
	{"dummy5", "dummy1"},
	{"dummy6", "dummy1"},
	{"dummy7", "dummy1"},

	{"bif", "baz"},
}

var testRegNames map[string][]string

func init() {
	testRegNames = make(map[string][]string)
	testRegNames["m*"] = []string{"goo"}
	testRegNames["zo*"] = []string{"us-east-1a", "us-east-1b", "us-east-1c", "us-east-1d", "us-east-1e", "loopy"}
	testRegNames["t[y]?pe"] = []string{"counter", "broken", "rate", "bool", "gauge"}
}

func testToSortingTags(tglen, listlen int) []repr.SortingTags {
	var out []repr.SortingTags

	l := len(testTagsList)
	for i := 0; i < listlen; i++ {
		j := 0
		var tgs repr.SortingTags
		for j < tglen {
			tg := testTagsList[rand.Intn(l)]
			if tgs.Find(tg[0]) != "" {
				continue
			}
			tgs.Set(tg[0], tg[1])
			j++
		}
		out = append(out, tgs)
	}
	return out
}

func testTagIndexSetup() (*TagIndex, []repr.SortingTags, []string) {
	tx := NewTagIndex()
	tags := testToSortingTags(5, 200)

	uid := 1000000000000
	uidStr := strconv.FormatUint(uint64(uid), 36)

	uidList := []string{}
	for _, tgList := range tags {
		tx.Add(uidStr, tgList)

		uidList = append(uidList, uidStr)
		uid++
		uidStr = strconv.FormatUint(uint64(uid), 36)
	}

	return tx, tags, uidList
}

func testTagIndexSetupForBench(tagLen, things int) (*TagIndex, []repr.SortingTags, []string) {
	tx := NewTagIndex()
	tags := testToSortingTags(tagLen, things)

	uid := 1000000000000
	uidStr := strconv.FormatUint(uint64(uid), 36)
	uidList := []string{}
	for _, tgList := range tags {
		tx.Add(uidStr, tgList)
		uidList = append(uidList, uidStr)
		uid++
		uidStr = strconv.FormatUint(uint64(uid), 36)
	}

	return tx, tags, uidList
}

func TestTagRamIndex__Add(t *testing.T) {

	tx, tags, uidList := testTagIndexSetup()

	fmt.Println("name index:", tx.nameIndex)
	for _, ontags := range tags {
		for _, tag := range ontags {
			if _, ok := tx.nameIndex[tag.Name]; !ok {
				t.Fatalf("Tag %s not in name index", tag.Name)
			}
		}
	}
	onIdx := 0
	for _, uid := range uidList {
		tgList := tags[onIdx]
		if g, ok := tx.uidInvertIndex[uid]; !ok {
			t.Fatalf("UID %s not in inverted uid index", uid)
		} else {
			t.Logf("UID: %s Inverted index: %v", uid, g)
		}
		idxTags := tx.uidInvertIndex[uid]
		for _, ont := range tgList {
			got := false
			for _, idxT := range idxTags {
				if idxT.Name == ont.Name && ont.Value == idxT.Value {
					got = true
					break
				}
			}
			if !got {

				t.Fatalf("Tag %v not in the uidIndex for uid %v :: \n\t Expected: %v \n\t Got %v", ont, uid, tgList, idxTags)
			}
		}
		onIdx++
	}

}

func TestTagRamIndex__FindTagsByUid(t *testing.T) {

	tx, tags, uidList := testTagIndexSetup()

	onIdx := 0
	for _, uid := range uidList {
		tgList := tags[onIdx]
		gotTags, _ := tx.FindTagsByUid(uid)
		for _, ont := range tgList {
			got := false
			for _, idxT := range gotTags {
				if idxT.Name == ont.Name && ont.Value == idxT.Value {
					got = true
					break
				}
			}
			if !got {

				t.Fatalf("Tag %v not in the FindTagsByUid for uid %v :: \n\t Expected: %v \n\t Got %v", ont, uid, tgList, gotTags)
			}
		}
		onIdx++
	}
}

func TestTagRamIndex__FindTagsByValue(t *testing.T) {

	tx, tagsList, _ := testTagIndexSetup()

	for _, tags := range tagsList {

		// standard test
		for _, ont := range tags {
			got, err := tx.FindTagsByValue(ont.Name, ont.Value)
			if len(got) != 1 || err != nil {
				t.Fatalf("Could not find Tag by Value: %v", ont)
			}

			if got[0].Value != ont.Value {
				t.Fatalf("Found wrong value: %v (got %v)", ont, got[0])
			}
		}

		// regex test
		for _, ont := range tags {
			vReg := ont.Value[:len(ont.Value)-2] + "*"
			got, err := tx.FindTagsByValue(ont.Name, vReg)
			if len(got) == 0 || err != nil {
				t.Fatalf("Could not find Tag by Value: %v", ont)
			}
		}
	}
}

func TestTagRamIndex__FindUidByTag(t *testing.T) {

	tx, tagsSet, _ := testTagIndexSetup()
	for _, tags := range tagsSet {
		for _, tag := range tags {
			uids, err := tx.FindUidByTag(tag)
			if err != nil {
				t.Fatal(err)
			}
			if len(uids) == 0 {
				t.Fatal("nothing found, when it should have been")

			}
		}
	}

	uids, err := tx.FindUidByTag(&repr.Tag{Name: "nothere", Value: "Nope"})
	if err != nil {
		t.Fatal(err)
	}
	if len(uids) != 0 {
		t.Fatal("nothing something, when it should not have")
	}
}

func TestTagRamIndex__FindUidByTags(t *testing.T) {

	tx, tagsSet, _ := testTagIndexSetup()
	for _, tags := range tagsSet {

		uids, err := tx.FindUidByTags(tags)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Found: %v", uids)
		if len(uids) == 0 {
			t.Fatal("nothing found, when it should have been")
		}
	}
}

func TestTagRamIndex__DeleteByUid(t *testing.T) {

	tx, tagsSet, uidList := testTagIndexSetup()

	err := tx.DeleteByUid(uidList[0])
	if err != nil {
		t.Fatalf(err.Error())
	}
	for _, tags := range tagsSet {

		uids, err := tx.FindUidByTags(tags)
		if err != nil {
			t.Fatal(err)
		}
		for _, uid := range uids {
			if uid == uidList[0] {
				t.Fatalf("Should not have found (FindUidByTags) uid %s after a delete", uid)
			}
		}
	}

	gots, _ := tx.FindTagsByUid(uidList[0])
	if gots != nil {
		t.Fatalf("Should not have found (FindTagsByUid) uid %s after a delete", uidList[0])
	}
}

func Benchmark_TagRamIndex__FindUidByTagsMulti(b *testing.B) {
	procs := []int{1, 2, 4, 8}
	mEles := []int{1000, 10000, 100000, 1000000}

	for _, proc := range procs {
		for _, maxEle := range mEles {
			b.Run(fmt.Sprintf("Find_Procs_%d_eles_%d", proc, maxEle), func(b *testing.B) {
				tx, tagsSet, _ := testTagIndexSetupForBench(8, maxEle)

				findF := make(chan repr.SortingTags, proc)
				var wg sync.WaitGroup

				doFind := func() {
					for {
						tg, more := <-findF
						if !more {
							return
						}
						tx.FindUidByTags(tg)
						wg.Done()
					}
				}

				for i := 0; i < proc; i++ {
					go doFind()
				}
				start := time.Now()
				finds := 0

				b.ResetTimer()
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					for _, tags := range tagsSet {
						wg.Add(1)
						findF <- tags
						finds++
					}
					wg.Wait()
				}
				endT := time.Now()
				b.Logf("Did %d Finds :: %f per second for %d elements at %d procs", finds, float64(time.Second)*float64(finds)/float64(endT.UnixNano()-start.UnixNano()), maxEle, proc)
				close(findF)
			})
		}
	}
}
