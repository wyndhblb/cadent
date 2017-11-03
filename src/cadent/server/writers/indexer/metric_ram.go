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
	For RAM based indexes

	"RAM" is a bit of misnomer, we actually maintain an on disk levelDB set of files
	as the "real ram" can get very big and eat real RAM that we need for the series
	themselves.  There are a good few sets of maps in RAM for direct lookups and expiration detection

	So this is basically an extension of the levelDB ram index w/o some of the other overhead that
	module incurs. As we basically "re-make it" each restart/startup.

	For a given string like

	my.metric.is.good

	we store

	name: my, pos: 0 -> value nil
	name: my.metric, pos: 1 -> value nil
	name: my.metric.is, pos: 2 -> value nil
	name: my.metric.is.good, pos: 3-> value uid

	so we have "4" index items for this one key since they can have an arbitrary number of children

	{pos}:{subpath} = uid|or nil

	when looking up via the a query say for `my.metric.*`

	we know there are at least 3 elements and so we can iterate over this where the key is

	prefix("2:my.metric.")

	then for each thing we find "apply" the real regex to things

	Since we may want to keep many millions of things in RAM, and performing the Iterator search
	over a million items can be expensive, we partition things into a few internal DBs

	The partition is determined the number of "." split items

	i.e.

	if the metric is

	part1.part2.part3.part4.part5.part6  we will put this into the

	DBindex = 6 % len(partitions)

	The downside is that queries for things that can create "hot spots" where there are alot of metrics
	of a given length, however, this is an acceptable penalty, since we are in ram


*/

package indexer

import (
	"bytes"
	"cadent/server/schemas/indexer"
	"cadent/server/utils/trie"
	"errors"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	leveldb_filter "github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	leveldb_opt "github.com/syndtr/goleveldb/leveldb/opt"
	leveldb_util "github.com/syndtr/goleveldb/leveldb/util"
	"golang.org/x/net/context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// how many DB partitions do we want
const METRIC_RAM_NUM_PARITIIONS = 16

// default location (directory) for the indexes
const METRIC_RAM_TMP_LOCATION = "/opt/cadent/data/cadent_index"

// read cache size
const METRIC_RAM_READ_CACHE_SIZE = 8

// basic size in MB of a given chunk of levelDB files
const METRIC_RAM_LEVELDB_FILE_SIZE = 20

var ErrorCannotDeleteByRegex = errors.New("Cannot delete by regex")

type trieItem struct {
	Uid     string
	LastHit time.Time
	Path    string
}

// this is the iterator for the case we want to loop over everythgin
type AllIterator struct {
	slice   *leveldb_util.Range
	dbs     []*leveldb.DB
	curDB   *leveldb.DB
	curIdx  int
	curIter iterator.Iterator
}

func NewAllIter(dbs []*leveldb.DB, slice *leveldb_util.Range) *AllIterator {
	n := new(AllIterator)
	n.dbs = dbs
	n.slice = slice
	n.curDB = n.dbs[0]
	n.curIter = n.curDB.NewIterator(n.slice, nil)
	return n
}

// match the iterator Interface
// Next next item
func (a *AllIterator) Next() bool {
	for a.curIter.Next() {
		return true
	}
	a.curIdx++
	a.curIter.Release()
	if len(a.dbs) == a.curIdx {
		return false
	}
	a.curDB = a.dbs[a.curIdx]
	a.curIter = a.curDB.NewIterator(a.slice, nil)
	return true
}

// Error last error
func (a *AllIterator) Error() error {
	return a.curIter.Error()
}

// First move to start
func (a *AllIterator) First() bool {
	a.curIdx = 0
	a.curDB = a.dbs[a.curIdx]
	a.curIter = a.curDB.NewIterator(a.slice, nil)
	return a.curIter.First()
}

// Last move to end
func (a *AllIterator) Last() bool {
	a.curIdx = len(a.dbs) - 1
	a.curDB = a.dbs[a.curIdx]
	a.curIter = a.curDB.NewIterator(a.slice, nil)
	return a.curIter.Last()
}

// Prev go back one iteration
func (a *AllIterator) Prev() bool {
	return a.curIter.Prev()
}

// Seek go to a current key
func (a *AllIterator) Seek(key []byte) bool {
	if a.curIter != nil {
		return a.curIter.Seek(key)
	}
	return false
}

// Key the current Key
func (a *AllIterator) Key() []byte {
	if a.curIter != nil {
		return a.curIter.Key()
	}
	return nil
}

// Value the current Key
func (a *AllIterator) Value() []byte {
	if a.curIter != nil {
		return a.curIter.Value()
	}
	return nil
}

// Valid do we have a iterator
func (a *AllIterator) Valid() bool {
	if a.curIter != nil {
		return true
	}
	return false
}

// Release do we have a iterator
func (a *AllIterator) Release() {
	if a.curIter != nil {
		a.curIter.Release()
	}
}

// SetReleaser set the releaser object
func (a *AllIterator) SetReleaser(rl leveldb_util.Releaser) {
	if a.curIter != nil {
		a.curIter.SetReleaser(rl)
	}
}

// MetricIndex object
type MetricIndex struct {
	nameIndex map[string]string
	nmLock    sync.RWMutex
	uidIndex  map[string]string
	uidLock   sync.RWMutex

	lastHit map[string]time.Time
	lsLock  sync.RWMutex

	tmpLocation    string
	numPartitions  int
	partitionSplit int
	dbs            []*leveldb.DB
	dbOpts         *leveldb_opt.Options
	TrieIndex      *trie.Trie // trie radix tree

}

// NewTagIndex get a new MetricIndex object
func NewMetricIndex(tmpLocation string) (*MetricIndex, error) {
	n := new(MetricIndex)
	n.tmpLocation = METRIC_RAM_TMP_LOCATION
	if len(tmpLocation) > 0 {
		n.tmpLocation = tmpLocation
	}
	n.nameIndex = make(map[string]string)
	n.uidIndex = make(map[string]string)
	n.lastHit = make(map[string]time.Time)
	n.numPartitions = METRIC_RAM_NUM_PARITIIONS
	n.TrieIndex = trie.New(".")
	n.dbs = make([]*leveldb.DB, n.numPartitions)

	n.dbOpts = new(leveldb_opt.Options)
	n.dbOpts.Filter = leveldb_filter.NewBloomFilter(10)

	n.dbOpts.BlockCacheCapacity = METRIC_RAM_READ_CACHE_SIZE * leveldb_opt.MiB
	n.dbOpts.CompactionTableSize = METRIC_RAM_LEVELDB_FILE_SIZE * leveldb_opt.MiB

	for i := 0; i < n.numPartitions; i++ {

		idxFile := filepath.Join(n.tmpLocation, fmt.Sprintf("idx_%d", i))

		// wipe out the local index to start "fresh" each time
		info, err := os.Stat(idxFile)
		if err == nil && info.IsDir() {
			err = os.RemoveAll(idxFile)
			if err != nil {
				return nil, err
			}
		}

		err = os.MkdirAll(idxFile, 0755)
		if err != nil {
			return nil, err
		}

		n.dbs[i], err = leveldb.OpenFile(idxFile, n.dbOpts)
		if err != nil {
			return nil, err
		}
	}
	return n, nil
}

func (mi *MetricIndex) whichPartition(met []byte) *leveldb.DB {
	part := bytes.Count(met, []byte(".")) + 1
	s := (part % mi.numPartitions) - 1 // there should always be at least "1"
	return mi.dbs[s]
}

// CullExpired every hour or so run through the paths and remove things that have "expired"
// returns a map of uid -> path that were purged
func (mi *MetricIndex) CullExpired(dur time.Duration) map[string]string {

	if dur > 0 {
		dur = -dur
	}

	toDel := make(map[string]string)
	mi.uidLock.RLock()
	for uid, pth := range mi.uidIndex {
		mi.lsLock.RLock()
		if t, ok := mi.lastHit[uid]; ok {
			if t.Before(time.Now().Add(dur)) {
				toDel[uid] = pth
			}
		} else {
			toDel[uid] = pth
		}
		mi.lsLock.RUnlock()
	}
	mi.uidLock.RUnlock()

	for uid, path := range toDel {
		mi.TrieIndex.Pop(path)

		mi.lsLock.Lock()
		delete(mi.lastHit, uid)
		mi.lsLock.Unlock()

		mi.uidLock.Lock()
		delete(mi.uidIndex, uid)
		mi.uidLock.Unlock()

		mi.nmLock.Lock()
		delete(mi.nameIndex, uid)
		mi.nmLock.Unlock()
	}
	return toDel
}

// Add adds a UID and Path to the big list to the index
func (mi *MetricIndex) Add(uid string, path string) error {

	// already indexed
	mi.nmLock.RLock()
	if _, ok := mi.nameIndex[path]; ok {
		mi.lsLock.Lock()
		mi.lastHit[uid] = time.Now()
		mi.lsLock.Unlock()
		mi.nmLock.RUnlock()

		return nil
	}
	mi.nmLock.RUnlock()

	/*
		segments := NewParsedPath(path, uid)
		for _, seg := range segments.Paths {
			levelKey := []byte(fmt.Sprintf("%d:%s", seg.Segment.Pos, seg.Segment.Segment))
			useDB := mi.whichPartition(levelKey)

			if seg.Segment.Pos == segments.Len-1 {
				useDB.Put(levelKey, []byte(uid), nil)
			} else {
				useDB.Put(levelKey, nil, nil)
			}
		}*/
	mi.nmLock.Lock()
	mi.nameIndex[path] = uid
	mi.nmLock.Unlock()

	mi.uidLock.Lock()
	mi.uidIndex[uid] = path
	mi.uidLock.Unlock()

	tItem := new(trieItem)
	tItem.Uid = uid
	tItem.Path = path
	tItem.LastHit = time.Now()
	mi.TrieIndex.Put(path, tItem)

	return nil
}

// SetLastSeen set the last seen time
func (mi *MetricIndex) SetLastSeen(uid string, lastseen time.Time) error {
	mi.lsLock.Lock()
	mi.lastHit[uid] = lastseen
	mi.lsLock.Unlock()
	return nil
}

// GetLastSeen last time a metrics was "added"
func (mi *MetricIndex) GetLastSeen(uid string) time.Time {
	mi.lsLock.RLock()
	g, _ := mi.lastHit[uid]
	mi.lsLock.RUnlock()
	return g
}

// Contains is this metric index?
func (mi *MetricIndex) Contains(metric string) string {
	mi.nmLock.RLock()
	defer mi.nmLock.RUnlock()
	if g, ok := mi.nameIndex[metric]; ok {
		return g
	}
	return ""
}

// ContainsUid is the uid indexed?
func (mi *MetricIndex) ContainsUid(uid string) string {
	mi.uidLock.RLock()
	defer mi.uidLock.RUnlock()
	if g, ok := mi.uidIndex[uid]; ok {
		return g
	}
	return ""
}

// FindAll get all indexed metrics
func (mi *MetricIndex) FindAll() (out []string) {

	mi.nmLock.RLock()
	for k := range mi.nameIndex {
		out = append(out, k)
	}
	mi.nmLock.RUnlock()

	return out
}

// FindIter get the iterator for a given prefix
func (mi *MetricIndex) FindIter(prefix string, useDB *leveldb.DB) iterator.Iterator {
	bt := []byte(prefix)
	if prefix == "" {
		return NewAllIter(mi.dbs, leveldb_util.BytesPrefix(bt))
	}

	if useDB == nil {
		useDB = mi.whichPartition(bt)
	}
	return useDB.NewIterator(leveldb_util.BytesPrefix(bt), nil)
}

// FindLowestPrefix get the iterator to the lowest prefix we desire
func (mi *MetricIndex) FindLowestPrefix(path string) (iter iterator.Iterator, reg *regexp.Regexp, prefix string, err error) {

	segs := strings.Split(path, ".")
	pLen := len(segs)

	// find the longest chunk w/o a reg and that will be the level db prefix filter
	needsRegex := needRegex(path)

	longChunk := ""
	useKey := path
	useKeyLen := pLen - 1
	if needsRegex {
		for _, pth := range segs {
			if needRegex(pth) {
				useKey = longChunk
				break
			}
			if len(longChunk) > 0 {
				longChunk += "."
			}
			longChunk += pth
		}
		reg, err = regifyKey(path)
		if err != nil {
			return nil, nil, "", err
		}

	}

	// need to figure out the proper partition to use from the raw key
	useDB := mi.whichPartition([]byte(path))

	// we simply troll the {len}:{prefix} world
	prefix = fmt.Sprintf("%d:%s", useKeyLen, useKey)

	return mi.FindIter(prefix, useDB), reg, prefix, err
}

// FindByUid find things by uid
func (mi *MetricIndex) FindByUid(uid string) (mt indexer.MetricFindItems, err error) {
	mi.uidLock.RLock()
	pth, ok := mi.uidIndex[uid]
	mi.uidLock.RUnlock()

	if !ok {
		return
	}
	ms := new(indexer.MetricFindItem)
	ms.Text = pth
	ms.Id = pth
	ms.Path = pth

	ms.Expandable = 0
	ms.Leaf = 1
	ms.AllowChildren = 0
	mt = append(mt, ms)

	return mt, err
}

// FindRoot special case for "root" == "*" finder
func (mi *MetricIndex) FindRoot() (mt indexer.MetricFindItems, err error) {

	have := mi.TrieIndex.Children("*")
	if len(have) > 0 {
		for k, v := range have {
			ms := new(indexer.MetricFindItem)
			ms.Text = k
			ms.Id = k
			ms.Path = k

			switch v.(type) {
			case time.Time:
				ms.Expandable = 0
				ms.Leaf = 1
				ms.AllowChildren = 0
			default:
				ms.Expandable = 1
				ms.Leaf = 0
				ms.AllowChildren = 1
			}
			mt = append(mt, ms)
		}
	}
	return mt, nil

	/*
		// log.Printf("USE KEY: %s", prefix)
		iter := mi.FindIter("0:", nil)
		for iter.Next() {
			value := iter.Key() // should be {post}:{path}
			valueArr := strings.Split(string(value), ":")
			onPath := valueArr[1]

			ms := new(indexer.MetricFindItem)
			ms.Text = onPath
			ms.Id = onPath
			ms.Path = onPath
			if len(iter.Value()) > 0 {
				ms.Expandable = 0
				ms.Leaf = 1
				ms.AllowChildren = 0
			} else {
				ms.Expandable = 1
				ms.Leaf = 0
				ms.AllowChildren = 1
			}

			mt = append(mt, ms)
		}

		iter.Release()
		err = iter.Error()
		return mt, err
	*/
}

// Find the metrics that match as MetricFindItems
func (mi *MetricIndex) Find(ctx context.Context, metric string, hasDataOnly bool) (mt indexer.MetricFindItems, err error) {

	// special case for "root" == "*"
	if metric == "*" {
		return mi.FindRoot()
	}

	errorList := make([]error, 0)
	seenList := make(map[string]bool)
	for _, targ := range splitMetricsPath(metric) {

		spl := strings.Split(targ, ".")
		last := spl[len(spl)-1]

		// direct match
		mi.nmLock.RLock()
		if got, ok := mi.nameIndex[targ]; ok {
			if _, ok := seenList[metric]; ok {
				mi.nmLock.RUnlock()
				continue
			}
			ms := new(indexer.MetricFindItem)
			ms.Text = last
			ms.Id = metric
			ms.Path = metric
			ms.UniqueId = got
			ms.Expandable = 0
			ms.Leaf = 1
			ms.AllowChildren = 0
			mt = append(mt, ms)
			mi.nmLock.RUnlock()
			seenList[metric] = true
			continue
		}
		mi.nmLock.RUnlock()

		needReg := needRegex(targ)
		var reg *regexp.Regexp
		trieQ := metric
		if needReg {
			reg, err = regifyKey(targ)
			if err != nil {
				errorList = append(errorList, err)
				continue
			}
			trieQ = toTrieQuery(targ)
		}

		have := mi.TrieIndex.ChildrenFlat(trieQ)

		//prefixPath := strings.Join(spl[0:len(spl)-2], ".")
		if len(have) > 0 {
			for k, v := range have {
				if _, ok := seenList[k]; ok {
					continue
				}
				spl := strings.Split(k, ".")
				nm := spl[len(spl)-1]

				//markPath := prefixPath + "." + nm
				if needReg && !reg.MatchString(k) {
					continue
				}

				ms := new(indexer.MetricFindItem)
				ms.Text = nm
				ms.Id = k
				ms.Path = k
				if hasDataOnly {
					switch v.(type) {
					case *trieItem:
						ms.Expandable = 0
						ms.UniqueId = v.(*trieItem).Uid
						ms.Leaf = 1
						ms.AllowChildren = 0
						mt = append(mt, ms)

					default:
						continue
					}
				} else {
					switch v.(type) {
					case *trieItem:
						ms.Expandable = 0
						ms.UniqueId = v.(*trieItem).Uid
						ms.Leaf = 1
						ms.AllowChildren = 0
					default:
						ms.Expandable = 1
						ms.Leaf = 0
						ms.AllowChildren = 1
					}
					mt = append(mt, ms)
				}
				seenList[k] = true

			}
		}
	}
	if len(errorList) > 0 {
		st := make([]string, 0)
		for _, e := range errorList {
			st = append(st, e.Error())
		}
		err = fmt.Errorf(strings.Join(st, ", "))
	}
	return mt, err

	/*
		iter, reged, _, err := mi.FindLowestPrefix(metric)

		if iter == nil || err != nil {
			return
		}
		// we simply troll the {len}:{prefix} world
		for iter.Next() {
			ms := new(indexer.MetricFindItem)
			// Remember that the contents of the returned slice should not be modified, and
			// only valid until the next call to Next.
			value := iter.Key() // should be {len}:{path} == uid (if data)
			valueArr := strings.Split(string(value), ":")
			if len(valueArr) != 2 {
				continue //bad data
			}
			hasData := len(iter.Value()) > 0
			onPath := valueArr[1]
			onId := string(iter.Value())

			spl := strings.Split(onPath, ".")

			if reged != nil {
				if reged.MatchString(onPath) {
					ms.Text = spl[len(spl)-1]
					ms.Id = onPath
					ms.Path = onPath
					ms.UniqueId = onId

					if hasData {
						ms.Expandable = 0
						ms.Leaf = 1
						ms.AllowChildren = 0
					} else if !hasDataOnly {
						ms.Expandable = 1
						ms.Leaf = 0
						ms.AllowChildren = 1
					}

					mt = append(mt, ms)
				}

			} else {
				ms.Text = spl[len(spl)-1]
				ms.Id = onPath
				ms.Path = onPath
				ms.UniqueId = onId

				if hasData {
					ms.Expandable = 0
					ms.Leaf = 1
					ms.AllowChildren = 0
				} else if !hasDataOnly {
					ms.Expandable = 1
					ms.Leaf = 0
					ms.AllowChildren = 1
				}

				mt = append(mt, ms)
			}
		}
		iter.Release()
		err = iter.Error()
		return mt, err
	*/
}

// Delete a metric path from the index
func (mi *MetricIndex) Delete(pth string) (err error) {
	needReg := needRegex(pth)
	if needReg {
		return ErrorCannotDeleteByRegex
	}

	// purge from exact matcher
	mi.nmLock.Lock()
	mi.uidLock.Lock()
	if uid, ok := mi.nameIndex[pth]; ok {
		delete(mi.nameIndex, pth)
		delete(mi.uidIndex, uid)
	}
	mi.uidLock.Unlock()
	mi.nmLock.Unlock()

	mi.lsLock.Lock()
	if _, ok := mi.lastHit[pth]; ok {
		delete(mi.lastHit, pth)
	}
	mi.lsLock.Unlock()

	// delete from the trei
	mi.TrieIndex.Pop(pth)

	// keys are {len}:{path}
	ct := len(strings.Split(pth, "."))

	bpth := []byte(fmt.Sprintf("%d:%s", ct-1, pth))
	usePart := mi.whichPartition([]byte(pth))

	return usePart.Delete(bpth, nil)
}

// DeleteByUid deleted a index based on the uid
func (mi *MetricIndex) DeleteByUid(uid string) (err error) {

	// purge from exact matcher
	mi.uidLock.RLock()
	if pth, ok := mi.uidIndex[uid]; ok {
		mi.uidLock.RUnlock()
		err = mi.Delete(pth)

		return err
	}
	mi.uidLock.RUnlock()
	return err
}
