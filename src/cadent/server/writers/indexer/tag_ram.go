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

	Tags are indexed as simple maps, depending on the cardinality this can get a bit big in RAM
		Names: map[Name|string][]string{values}
		Uids: map[Name=Value|string][]{uids}

*/

package indexer

import (
	"cadent/server/schemas/indexer"
	"cadent/server/schemas/repr"
	//"github.com/syndtr/goleveldb/leveldb/comparer"
	//"github.com/syndtr/goleveldb/leveldb/memdb"
	"strings"
	"sync"
)

const TAG_INDEX_CAPACITY = 1 * 1024 * 1024

// TagIndex object
type TagIndex struct {
	//db *memdb.DB

	nameIndex map[string][]string
	nmLock    sync.RWMutex

	uidIndex map[string][]string
	uidLock  sync.RWMutex

	uidInvertIndex map[string][]*repr.Tag
	uidInvLoc      sync.RWMutex
}

// NewTagIndex get a new TagIndex object
func NewTagIndex() *TagIndex {
	n := new(TagIndex)
	//n.db = memdb.New(comparer.DefaultComparer, TAG_INDEX_CAPACITY)
	n.nameIndex = make(map[string][]string)
	n.uidIndex = make(map[string][]string)
	n.uidInvertIndex = make(map[string][]*repr.Tag)
	return n
}

// Add adds a list of SortingTags to the index
func (tv *TagIndex) Add(uid string, tags repr.SortingTags) error {
	for _, t := range tags {
		nv := t.Name + "=" + t.Value
		isNew := false

		// already indexed
		tv.uidInvLoc.Lock()
		if _, ok := tv.uidInvertIndex[uid]; !ok {
			// new uid
			isNew = true

			tv.uidLock.Lock()
			tv.uidIndex[nv] = []string{uid}
			tv.uidLock.Unlock()

			tv.uidInvertIndex[uid] = []*repr.Tag{t}
		}
		tv.uidInvLoc.Unlock()

		tv.nmLock.Lock()
		// add to name -> values map if not there
		if g, ok := tv.nameIndex[t.Name]; ok {
			have := false
			for _, v := range g {
				if v == t.Value {
					have = true
					break
				}
			}
			if !have {
				tv.nameIndex[t.Name] = append(tv.nameIndex[t.Name], t.Value)
			}
		} else {
			tv.nameIndex[t.Name] = []string{t.Value}
		}
		tv.nmLock.Unlock()

		if !isNew {
			tv.uidInvLoc.Lock()
			tv.uidInvertIndex[uid] = append(tv.uidInvertIndex[uid], t)
			tv.uidInvLoc.Unlock()

			tv.uidLock.Lock()
			tv.uidIndex[nv] = append(tv.uidIndex[nv], uid)
			tv.uidLock.Unlock()
		}
	}
	return nil
}

// DeleteByUid purge an index by it's UID
func (tv *TagIndex) DeleteByUid(uid string) error {
	// already indexed
	tv.uidInvLoc.Lock()
	tv.uidLock.Lock()
	if tags, ok := tv.uidInvertIndex[uid]; ok {

		// need to remove the inverted
		for _, tag := range tags {
			nv := tag.Name + "=" + tag.Value
			if uids, tok := tv.uidIndex[nv]; tok {
				newList := []string{}
				for _, nvUid := range uids {
					if nvUid != uid {
						newList = append(newList, nvUid)
					}
				}
				tv.uidIndex[nv] = newList
			}
		}
		delete(tv.uidInvertIndex, uid)
	}
	tv.uidLock.Unlock()
	tv.uidInvLoc.Unlock()
	return nil
}

func (tv *TagIndex) FindTagsByUid(uid string) (repr.SortingTags, error) {
	tv.uidInvLoc.RLock()
	defer tv.uidInvLoc.RUnlock()
	if g, ok := tv.uidInvertIndex[uid]; ok {
		return g, nil
	}
	return nil, nil
}

// GetTagsByName get the  (the name can be a RegEx)
func (tv *TagIndex) GetTagsByName(nm string) (tags indexer.MetricTagItems, err error) {
	tv.nmLock.RLock()
	defer tv.nmLock.RLock()

	needR := needRegex(nm)
	if !needR {
		if g, ok := tv.nameIndex[nm]; ok {
			for _, val := range g {
				tags = append(tags, &indexer.MetricTagItem{Name: nm, Value: val})
			}
			return tags, nil
		}
	}
	// otherwise need to regex the list
	reg, err := regifyKey(nm)
	if err != nil {
		return nil, err
	}

	for nm, vs := range tv.nameIndex {
		if reg.MatchString(nm) {
			for _, val := range vs {
				tags = append(tags, &indexer.MetricTagItem{Name: nm, Value: val})
			}
		}
	}
	return
}

// FindTagValuesByName get the values associated with a tag name (the name can be a RegEx)
func (tv *TagIndex) FindTagValuesByName(nm string) ([]string, error) {
	tv.nmLock.RLock()
	defer tv.nmLock.RLock()
	outvals := []string{}

	needR := needRegex(nm)
	if !needR {
		if g, ok := tv.nameIndex[nm]; ok {
			return g, nil
		}
	}

	// otherwise need to regex the list
	reg, err := regifyKey(nm)
	if err != nil {
		return nil, err
	}
	for nm, vs := range tv.nameIndex {
		if reg.MatchString(nm) {
			outvals = append(outvals, vs...)
		}
	}

	return outvals, nil
}

// FindTagsByValue get the tags associated with a tag name and a value (the value can be a regex, NOT the name)
func (tv *TagIndex) FindTagsByValue(nm string, value string) ([]*repr.Tag, error) {
	tv.nmLock.RLock()
	defer tv.nmLock.RLock()
	var outvals []*repr.Tag
	var gotNm []string
	var ok bool

	if gotNm, ok = tv.nameIndex[nm]; !ok {
		return outvals, nil
	}

	needR := needRegex(value)

	if needR {
		// otherwise need to regex the list
		reg, err := regifyKey(value)
		if err != nil {
			return nil, err
		}

		for _, v := range gotNm {
			if reg.MatchString(v) {
				outvals = append(outvals, &repr.Tag{Name: nm, Value: v})
			}
		}
	} else {
		for _, v := range gotNm {
			if strings.EqualFold(v, value) {
				outvals = append(outvals, &repr.Tag{Name: nm, Value: v})
			}
		}
	}

	return outvals, nil
}

// GetTagsByNameValue get the tags associated with a tag name and a value (the value can be a regex, NOT the name)
func (tv *TagIndex) GetTagsByNameValue(nm string, value string) (tags indexer.MetricTagItems, err error) {
	tv.nmLock.RLock()
	defer tv.nmLock.RLock()

	var gotNm []string
	var ok bool

	// nothing to get
	if gotNm, ok = tv.nameIndex[nm]; !ok {
		return tags, nil
	}

	needR := needRegex(value)

	if needR {
		// otherwise need to regex the list
		reg, err := regifyKey(value)
		if err != nil {
			return nil, err
		}

		for _, v := range gotNm {
			if reg.MatchString(v) {
				tags = append(tags, &indexer.MetricTagItem{Name: nm, Value: v})
			}
		}
	} else {
		for _, v := range gotNm {
			if strings.EqualFold(v, value) {
				tags = append(tags, &indexer.MetricTagItem{Name: nm, Value: v})
			}
		}
	}

	return tags, nil
}

// FindUidByTag given a tag find the UIDs that have it
func (tv *TagIndex) FindUidByTag(tag *repr.Tag) ([]string, error) {
	tv.uidLock.RLock()
	defer tv.uidLock.RLock()

	tg := tag.Name + "=" + tag.Value
	if g, ok := tv.uidIndex[tg]; ok {
		return g, nil
	}
	return []string{}, nil
}

// FindUidByTags given a tag set, find the union of UIDs that have them
func (tv *TagIndex) FindUidByTags(tags repr.SortingTags) ([]string, error) {
	tv.uidLock.RLock()
	defer tv.uidLock.RLock()

	outSet := []string{}
	seenMap := make(map[string]bool)

	for _, tag := range tags {
		tg := tag.Name + "=" + tag.Value
		if g, ok := tv.uidIndex[tg]; !ok {
			continue
		} else {
			for _, uid := range g {
				if _, ok := seenMap[uid]; ok {
					continue
				} else {
					seenMap[uid] = true
					outSet = append(outSet, uid)
				}
			}
		}
	}

	return outSet, nil
}
