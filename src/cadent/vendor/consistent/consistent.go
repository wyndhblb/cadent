// Copyright (C) 2012 Numerotron Inc
// Use of this source code is governed by an MIT-style license
// that can be found in the LICENSE file.

// Package consistent provides a consistent hashing function.
//
// Consistent hashing is often used to distribute requests to a changing set of servers.  For example,
// say you have some cache servers cacheA, cacheB, and cacheC.  You want to decide which cache server
// to use to look up information on a user.
//
// You could use a typical hash table and hash the user id
// to one of cacheA, cacheB, or cacheC.  But with a typical hash table, if you add or remove a server,
// almost all keys will get remapped to different results, which basically could bring your service
// to a grinding halt while the caches get rebuilt.
//
// With a consistent hash, adding or removing a server drastically reduces the number of keys that
// get remapped.
//
// Read more about consistent hashing on wikipedia:  http://en.wikipedia.org/wiki/Consistent_hashing
//
package consistent

import (
	"errors"
	"sort"
	"sync"
	//"log"
)

type uints []uint32
type hasherFunc func(string) uint32
type eltFunc func(string, int) string

const DEFAULT_HASHER = "crc"
const DEFAULT_ELTER = "graphite"

// Len returns the length of the uints array.
func (x uints) Len() int { return len(x) }

// Less returns true if element i is less than element j.
func (x uints) Less(i, j int) bool { return x[i] < x[j] }

// Swap exchanges elements i and j.
func (x uints) Swap(i, j int) { x[i], x[j] = x[j], x[i] }

// ErrEmptyCircle is the error returned when trying to get an element when nothing has been added to hash.
var ErrEmptyCircle = errors.New("empty circle")

// Consistent holds the information about the members of the consistent hash circle.
type Consistent struct {
	circle           map[uint32]string
	members          map[string]bool
	sortedHashes     uints
	NumberOfReplicas int
	count            int64
	scratch          [64]byte
	hasher           hasherFunc
	elter            eltFunc

	sync.RWMutex
}

// New creates a new Consistent object with a default setting of 20 replicas for each entry.
//
// To change the number of replicas, set NumberOfReplicas before adding entries.
func New() *Consistent {
	c := new(Consistent)
	c.NumberOfReplicas = 4
	c.circle = make(map[uint32]string)
	c.members = make(map[string]bool)
	c.SetHasherByName(DEFAULT_HASHER)
	c.SetElterByName(DEFAULT_ELTER)
	return c
}

func (c *Consistent) SetNumberOfReplicas(num int) {
	old_mems := c.Members()

	if num > 0 {
		c.NumberOfReplicas = num
	} else {
		c.NumberOfReplicas = 1 // need at least one
	}
	c.ResetCircle(old_mems)
}

func (c *Consistent) SetHasherByName(hasher string) {
	old_mems := c.Members()
	switch hasher {
	case "crc", "crc32":
		c.hasher = Crc32
	case "md5", "graphite":
		c.hasher = SmallMD5
	case "sha1":
		c.hasher = SmallSHA1
	case "fnv", "fnv32":
		c.hasher = Fnv32
	case "fnva", "fnv32a":
		c.hasher = Fnv32a
	case "statsd", "nodejs-hashring":
		c.hasher = NodejsHashringLike
	case "mmh", "mmh3", "mmh32":
		c.hasher = Mmh32
	default:
		c.hasher = Crc32

	}
	c.ResetCircle(old_mems)

}

func (c *Consistent) GetHasherValue(key string) uint32 {
	return c.hashKey(key)
}

func (c *Consistent) SetElterByName(elter string) {
	old_mems := c.Members()
	switch elter {
	case "statsd":
		c.elter = statsdElt
	case "consistent", "goconsistent", "simple":
		c.elter = consistentElt
	default:
		c.elter = graphiteElt
	}
	//reset the cirle after a change here
	c.ResetCircle(old_mems)
}

func (c *Consistent) Circle() map[uint32]string {
	return c.circle
}

func (c *Consistent) Sorted() uints {
	return c.sortedHashes
}

func (c *Consistent) ResetCircle(members []string) {

	for _, elt := range members {
		c.Remove(elt)
		c.Add(elt)
	}

}

// eltKey generates a string key for an element with an index.
func (c *Consistent) eltKey(elt string, idx int) string {
	// this is "graphite's" way of making the key
	return c.elter(elt, idx)
}

// Add inserts a string element in the consistent hash.
func (c *Consistent) Add(elt string) {
	//log.Print("SERVER::::: ", elt, " ", c.NumberOfReplicas)
	c.Lock()
	defer c.Unlock()
	c.add(elt)
}

// need c.Lock() before calling
func (c *Consistent) add(elt string) {
	for i := 0; i < c.NumberOfReplicas; i++ {
		//log.Print("SERVER::::: ", elt, " ", c.eltKey(elt, i), " ", c.hashKey(c.eltKey(elt, i)))
		c.circle[c.hashKey(c.eltKey(elt, i))] = elt
	}
	c.members[elt] = true
	c.updateSortedHashes()
	c.count++
}

// Remove removes an element from the hash.
func (c *Consistent) Remove(elt string) {
	c.Lock()
	defer c.Unlock()
	c.remove(elt)
}

// need c.Lock() before calling
func (c *Consistent) remove(elt string) {
	for i := 0; i < c.NumberOfReplicas; i++ {
		delete(c.circle, c.hashKey(c.eltKey(elt, i)))
	}
	delete(c.members, elt)
	c.updateSortedHashes()
	c.count--
}

// Set sets all the elements in the hash.  If there are existing elements not
// present in elts, they will be removed.
func (c *Consistent) Set(elts []string) {
	c.Lock()
	defer c.Unlock()
	for k := range c.members {
		found := false
		for _, v := range elts {
			if k == v {
				found = true
				break
			}
		}
		if !found {
			c.remove(k)
		}
	}
	for _, v := range elts {
		_, exists := c.members[v]
		if exists {
			continue
		}
		c.add(v)
	}
}

func (c *Consistent) Members() []string {
	c.RLock()
	defer c.RUnlock()
	var m []string
	for k := range c.members {
		m = append(m, k)
	}
	return m
}

// Get returns an element close to where name hashes to in the circle.
func (c *Consistent) Get(name string) (string, error) {
	c.RLock()
	defer c.RUnlock()
	if len(c.circle) == 0 {
		return "", ErrEmptyCircle
	}
	key := c.hashKey(name)
	i := c.search(key)
	//log.Print(name, " ", key, " ", i, " ",c.sortedHashes, " ",  c.circle[c.sortedHashes[i]], " ")
	return c.circle[c.sortedHashes[i]], nil
}

func (c *Consistent) search(key uint32) (i int) {
	f := func(x int) bool {
		return c.sortedHashes[x] > key
	}
	i = sort.Search(len(c.sortedHashes), f)
	if i >= len(c.sortedHashes) {
		i = 0
	}
	return
}

// GetTwo returns the two closest distinct elements to the name input in the circle.
func (c *Consistent) GetTwo(name string) (string, string, error) {
	c.RLock()
	defer c.RUnlock()
	if len(c.circle) == 0 {
		return "", "", ErrEmptyCircle
	}
	key := c.hashKey(name)
	i := c.search(key)
	a := c.circle[c.sortedHashes[i]]

	if c.count == 1 {
		return a, "", nil
	}

	start := i
	var b string
	for i = start + 1; i != start; i++ {
		if i >= len(c.sortedHashes) {
			i = 0
		}
		b = c.circle[c.sortedHashes[i]]
		if b != a {
			break
		}
	}
	return a, b, nil
}

// GetN returns the N closest distinct elements to the name input in the circle.
func (c *Consistent) GetN(name string, n int) ([]string, error) {
	c.RLock()
	defer c.RUnlock()

	if len(c.circle) == 0 {
		return nil, ErrEmptyCircle
	}

	//quick bail
	if n == 1 || len(c.circle) == 1 {
		key, err := c.Get(name)
		return []string{key}, err
	}

	if c.count < int64(n) {
		n = int(c.count)
	}

	var (
		key   = c.hashKey(name)
		i     = c.search(key)
		start = i
		res   = make([]string, 0, n)
		elem  = c.circle[c.sortedHashes[i]]
	)

	res = append(res, elem)

	if len(res) == n {
		return res, nil
	}

	for i = start + 1; i != start; i++ {
		if i >= len(c.sortedHashes) {
			i = 0
		}
		elem = c.circle[c.sortedHashes[i]]
		if !sliceContainsMember(res, elem) {
			res = append(res, elem)
		}
		if len(res) == n {
			break
		}
	}

	return res, nil
}

func (c *Consistent) hashKey(key string) uint32 {

	return c.hasher(key)
}

func (c *Consistent) updateSortedHashes() {

	hashes := c.sortedHashes[:0]
	if c.NumberOfReplicas > 0 {
		//reallocate if we're holding on to too much (1/4th)
		if cap(c.sortedHashes)/(c.NumberOfReplicas*4) > len(c.circle) {
			hashes = nil
		}
	} else {
		hashes = nil
	}
	for k := range c.circle {
		hashes = append(hashes, k)
	}
	sort.Sort(hashes)
	c.sortedHashes = hashes
}

func sliceContainsMember(set []string, member string) bool {
	for _, m := range set {
		if m == member {
			return true
		}
	}
	return false
}
