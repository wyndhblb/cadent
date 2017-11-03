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
	Ram indexes form the basis for all the other indexes,

	It uses the metric_ram and tag_ram modules to maintain internal states
*/

package indexer

import (
	"cadent/server/schemas/repr"
	"cadent/server/utils/options"
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

func testRamIndexSetup() (*RamIndexer, []repr.SortingTags, []string) {
	opts := options.New()
	opts.Set("local_index_dir", "/tmp/testy")
	tx := NewRamIndexer()
	tx.Config(&opts)

	uidList := []string{}
	tgList := []repr.SortingTags{}
	tglen := 4
	randTags := func() (tgs repr.SortingTags) {

		l := len(testTagsList)
		j := 0
		for j < tglen {
			tg := testTagsList[rand.Intn(l)]
			if tgs.Find(tg[0]) != "" {
				continue
			}
			tgs.Set(tg[0], tg[1])
			j++
		}
		return tgs
	}

	for _, metric := range testMetricList {
		rTags := randTags()
		tgList = append(tgList, rTags)
		mName := repr.StatName{Key: metric, Tags: rTags}
		tx.Write(mName)
		uidList = append(uidList, mName.UniqueIdString())
	}

	return tx, tgList, uidList
}

func Test_RamIndex__Add(t *testing.T) {

	tx, _, _ := testRamIndexSetup()
	for _, met := range testMetricList {
		if _, ok := tx.MetricIndex.nameIndex[met]; !ok {
			t.Fatalf("Metric %s not in name index", met)
		}
	}
}

func Test_RamIndex__Find(t *testing.T) {
	tx, _, _ := testRamIndexSetup()
	fi := testMetricList[0]

	items, err := tx.Find(fi, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("Did not find the one we wanted")
	}
	fmt.Println("Found Exact:", items[0].Path, items[0].UniqueId)

	// regexer
	for _, fi = range []string{"stats.timers.*", "stats.t[i]?mers.*", "{stats|servers}.timers.*"} {
		items, err = tx.Find(fi, nil)

		if err != nil {
			t.Fatal(err)
		}

		needMap := make(map[string]bool)
		//the should get
		for _, i := range testMetricList {
			want := strings.Split(i, ".")

			if len(want) > 3 {
				if strings.HasPrefix(i, "stats.timers.") {
					needMap[strings.Join(want[:3], ".")] = true
				}
			}
		}

		ct := 0
		for range items {
			ct++
		}
		if ct != len(needMap) {
			t.Logf("Not all root metrics indexed wanted %d, got %d", len(needMap), ct)
		}
	}
}

func Test_RamIndex__List(t *testing.T) {

	tx, _, _ := testRamIndexSetup()
	lister, err := tx.List(false, 0)
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("Added %d metrics, found %d in list", len(testMetricList), len(lister))
	if len(lister) != len(testMetricList) {
		t.Fatalf("Metric List is not the same size for List")

	}
}

func Test_RamIndex__Expand(t *testing.T) {

	tx, _, _ := testRamIndexSetup()

	for _, fi := range []string{"stats.timers.*", "stats.timers.*", "stats.t[i]?mers.*", "{stats,servers}.timers.*"} {
		items, err := tx.Expand(fi)

		if err != nil {
			t.Fatal(err)
		}

		needMap := make(map[string]bool)
		//the should get
		for _, i := range testMetricList {
			want := strings.Split(i, ".")

			if len(want) > 3 {
				if strings.HasPrefix(i, "stats.timers.") {
					needMap[strings.Join(want[:3], ".")] = true
				}
			}
		}

		ct := 0
		for range items.Results {
			ct++
		}
		if ct != len(needMap) {
			t.Logf("Not all root metrics indexed wanted %d, got %d", len(needMap), ct)
		}
	}
}

func Test_RamIndex__GetTagsByUid(t *testing.T) {

	tx, _, uids := testRamIndexSetup()

	for _, u := range uids {
		tgs, _, err := tx.GetTagsByUid(u)
		if err != nil {
			t.Fatalf(err.Error())
		}
		if len(tgs) == 0 {
			t.Fatalf("Did not find enough tags for uid: %s", u)
		}
	}

	tgs, _, err := tx.GetTagsByUid("IMNOTHERE")
	if err != nil {
		t.Fatalf(err.Error())
	}
	if len(tgs) != 0 {
		t.Fatalf("Found too many tags:  %s", tgs)
	}

}

func Test_RamIndex__GetTagsByName(t *testing.T) {

	tx, tgs, _ := testRamIndexSetup()

	for _, tag := range tgs {
		gott, err := tx.GetTagsByName(tag[0].Name, 0)
		if err != nil {
			t.Fatalf(err.Error())
		}
		if len(gott) == 0 {
			t.Fatalf("Did not find enough tags for tag: %s", tag)
		}
		for _, g := range gott {
			if g.Name != tag[0].Name {
				t.Fatalf("Did not find enough tags for uid: %s", tag)

			}
		}
	}

	gott, err := tx.GetTagsByName("IMNOTHERE", 0)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if len(gott) != 0 {
		t.Fatalf("Found too many tags:  %s", tgs)
	}

}

func Test_RamIndex__GetTagsByNameValue(t *testing.T) {

	tx, tgs, _ := testRamIndexSetup()

	for _, tag := range tgs {
		gott, err := tx.GetTagsByNameValue(tag[0].Name, tag[0].Value, 0)
		if err != nil {
			t.Fatalf(err.Error())
		}
		if len(gott) == 0 {
			t.Fatalf("Did not find enough tags for tag: %s", tag)
		}
		for _, g := range gott {
			if g.Name != tag[0].Name {
				t.Fatalf("Did not find enough tags for uid: %s", tag)

			}
		}
	}

	gott, err := tx.GetTagsByNameValue("IMNOTHERE", "NOPE", 0)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if len(gott) != 0 {
		t.Fatalf("Found too many tags:  %s", tgs)
	}

}

func Test_RamIndex__GetUidsByTags(t *testing.T) {

	tx, tgs, _ := testRamIndexSetup()

	for _, tag := range tgs {
		gott, err := tx.GetUidsByTags("", tag, 0)
		if err != nil {
			t.Fatalf(err.Error())
		}
		if len(gott) == 0 {
			t.Fatalf("Did not find enough tags for tag: %s", tag)
		}

	}

	gott, err := tx.GetUidsByTags("IMNOTHERE", nil, 0)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if len(gott) != 0 {
		t.Fatalf("Found too many uids:  %s", tgs)
	}

}

func Test_RamIndex__Delete(t *testing.T) {

	tx, _, _ := testRamIndexSetup()

	list, err := tx.List(true, 0)
	if err != nil {
		t.Fatalf(err.Error())
	}

	item := list[0]

	sName := &repr.StatName{
		Key:  item.Path,
		Tags: item.Tags,
	}
	t.Logf("StatName: deleting %s tags: %s uid: %s", sName.Key, sName.Tags, sName.UniqueIdString())

	if sName.UniqueIdString() != item.UniqueId {
		t.Fatalf("Different UIDs .. bad thing")
	}

	tx.Delete(sName)

	items, err := tx.Find(item.Path, nil)

	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("Delete failed, still found things")
	}
}
