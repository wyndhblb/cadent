/**

Modified version of from github.com/hit9/trie

That basically makes things thread safe and add some child/flatten functions specifically meant for a graphite,statsd
like finder system
*/

package trie

import (
	"strings"
	"sync"
)

// graphite, statsd basic splitter
const DEFAULT_DELIM = "."

// Tree is the internal tree.
type Tree struct {
	sync.RWMutex
	value    interface{}
	children map[string]*Tree
}

// Trie is the trie tree.
type Trie struct {
	root   *Tree // root tree, won't be rewritten
	delim  string
	length int
	lk     sync.RWMutex // protects the whole trie

}

// newTree creates a new tree.
func newTree() *Tree {
	return &Tree{
		children: make(map[string]*Tree, 0),
	}
}

// New creates a new Trie.
func New(delim string) *Trie {
	if delim == "" {
		delim = DEFAULT_DELIM
	}
	return &Trie{
		root:  newTree(),
		delim: delim,
	}
}

// Len returns the trie length.
func (tr *Trie) Len() int {
	tr.lk.RLock()
	defer tr.lk.RUnlock()
	return tr.length
}

// Put an item to the trie.
// Replace if the key conflicts.
func (tr *Trie) Put(key string, value interface{}) {
	tr.lk.Lock()
	defer tr.lk.Unlock()
	parts := strings.Split(key, tr.delim)
	t := tr.root
	for i, part := range parts {
		t.RLock()
		child, ok := t.children[part]
		t.RUnlock()
		if !ok {
			child = newTree()
			t.Lock()
			t.children[part] = child
			t.Unlock()
		}
		if i == len(parts)-1 {
			if child.value == nil {
				tr.length++
			}
			child.value = value
			return
		}
		t = child
	}
	return
}

// Get an item from the trie.
// Returns nil if the given key is not in the trie.
func (tr *Trie) Get(key string) interface{} {
	tr.lk.RLock()
	defer tr.lk.RUnlock()
	parts := strings.Split(key, tr.delim)
	t := tr.root
	for i, part := range parts {
		child, ok := t.children[part]
		if !ok {
			return nil
		}
		if i == len(parts)-1 {
			return child.value
		}
		t = child
	}
	return nil
}

// Has checks if an item is in trie.
// Returns true if the given key is in the trie.
func (tr *Trie) Has(key string) bool {
	return tr.Get(key) != nil
}

// Pop an item from the trie.
// Returns nil if the given key is not in the trie.
func (tr *Trie) Pop(key string) interface{} {
	tr.lk.Lock()
	defer tr.lk.Unlock()
	parts := strings.Split(key, tr.delim)
	t := tr.root
	for i, part := range parts {
		child, ok := t.children[part]
		if !ok {
			return nil
		}
		if i == len(parts)-1 {
			if len(child.children) == 0 {
				delete(t.children, part)
			}
			value := child.value
			child.value = nil
			if value != nil {
				tr.length--
			}
			return value
		}
		t = child
	}
	return nil
}

// Clear the trie.
func (tr *Trie) Clear() {
	tr.lk.Lock()
	defer tr.lk.Unlock()
	tr.root.children = make(map[string]*Tree, 0)
	tr.length = 0
}

// Match a wildcard like pattern in the trie, the pattern is not a traditional
// wildcard, only "*" is supported.
func (tr *Trie) Match(pattern string) map[string]interface{} {
	return tr.root.match(tr.delim, nil, strings.Split(pattern, tr.delim))
}

// match keys in the tree recursively.
func (t *Tree) match(delim string, keys []string, parts []string) map[string]interface{} {

	m := make(map[string]interface{}, 0)
	if len(parts) == 0 && t.value != nil {
		m[strings.Join(keys, delim)] = t.value
		return m
	}
	for i, part := range parts {
		if part == "*" {
			t.RLock()
			for segment, child := range t.children {
				v := child.match(delim, append(keys, segment), parts[i+1:])
				for key, value := range v {
					m[key] = value
				}
			}
			t.RUnlock()
			return m
		}
		t.RLock()
		child, ok := t.children[part]
		t.RUnlock()
		if !ok {
			return m
		}
		keys = append(keys, part)
		if i == len(parts)-1 { // last part
			if child.value != nil {
				m[strings.Join(keys, delim)] = child.value
			}
			return m
		}
		t = child // child as parent
	}
	return m
}

// Children get the children (or the value if that's what we hit) for a wild card pattern
// given a query like a.*.c return matches like {a.b.c.d}
func (tr *Trie) Children(pattern string) map[string]interface{} {
	spl := strings.Split(pattern, tr.delim)
	return tr.root._children(tr.delim, nil, spl, len(spl))
}

// _children keys in the tree recursively.
func (t *Tree) _children(delim string, keys []string, parts []string, bLen int) map[string]interface{} {

	m := make(map[string]interface{}, 0)
	if len(parts) == 0 {
		if len(t.children) > 0 {
			m[strings.Join(keys, delim)] = t.children
		} else {
			m[strings.Join(keys, delim)] = t.value
		}
		return m
	}

	for i, part := range parts {
		if part == "*" {
			t.RLock()
			for segment, child := range t.children {
				v := child._children(delim, append(keys, segment), parts[i+1:], bLen)
				for key, value := range v {
					m[key] = value
				}
			}
			t.RUnlock()
			return m
		}
		t.RLock()
		child, ok := t.children[part]
		t.RUnlock()
		if !ok {
			return m
		}
		keys = append(keys, part)
		if i == len(parts)-1 { // last part
			if len(child.children) > 0 {
				m[strings.Join(keys, delim)] = child.children
			} else {
				m[strings.Join(keys, delim)] = child.value
			}

			return m
		}
		t = child // child as parent
	}
	return m
}

// ChildrenFlat get the children (or the value if that's what we hit) for a wild card pattern
// given a query like a.*.c return matches like {a.b.c.d} but rather then a recursive map structure
// return one map with the keys "expanded" and the values either the "children" or the "value"
func (tr *Trie) ChildrenFlat(pattern string) map[string]interface{} {
	m := make(map[string]interface{}, 0)
	prefix := ""
	childs := tr.Children(pattern)

	_childrenFlat(tr.delim, prefix, childs, m)
	return m
}

// _childrenFlat flatten a child list into a single map
func _childrenFlat(delim string, prefix string, childs map[string]interface{}, m map[string]interface{}) map[string]interface{} {
	base := ""
	if prefix != "" {
		base = prefix + delim
	}
	for k, val := range childs {
		nextPref := base + k
		switch val.(type) {
		case map[string]interface{}:
			_childrenFlat(delim, nextPref, val.(map[string]interface{}), m)
		default:
			m[base+k] = val
		}
	}

	return m

}

// Map returns the full trie as a map.
func (tr *Trie) Map() map[string]interface{} {
	tr.lk.RLock()
	defer tr.lk.RUnlock()

	return tr.root._map(tr.delim, nil)
}

// map returns the full tree as a map.
func (t *Tree) _map(delim string, keys []string) map[string]interface{} {
	m := make(map[string]interface{}, 0)
	// Check current tree.
	if t.value != nil {
		m[strings.Join(keys, delim)] = t.value
	}
	// Check children.
	for segment, child := range t.children {
		d := child._map(delim, append(keys, segment))
		for key, value := range d {
			m[key] = value
		}
	}
	return m
}

// Matched uses the trie items as the wildcard like patterns, filters out the
// items matches the given string.
// Returns an empty map if the given strings matches no patterns.
func (tr *Trie) Matched(s string) map[string]interface{} {
	tr.lk.RLock()
	defer tr.lk.RUnlock()
	return tr.root.matched(tr.delim, nil, strings.Split(s, tr.delim))
}

// matched returns the patterns matched the given string.
func (t *Tree) matched(delim string, keys, parts []string) map[string]interface{} {
	m := make(map[string]interface{})
	if len(parts) == 0 && t.value != nil {
		m[strings.Join(keys, delim)] = t.value
		return m
	}
	if len(parts) > 0 {
		t.RLock()
		if child, ok := t.children["*"]; ok {
			for k, v := range child.matched(delim, append(keys, "*"), parts[1:]) {
				m[k] = v
			}
		}
		if child, ok := t.children[parts[0]]; ok {
			for k, v := range child.matched(delim, append(keys, parts[0]), parts[1:]) {
				m[k] = v
			}
		}
		t.RUnlock()
	}
	return m
}
