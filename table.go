/*
 * The DHT's routing table implementation in Golang.
 */
package ktable

import (
	"bytes"
	"crypto/rand"
	"sort"
	"sync"
	"time"
)

const (
	expiredAfter = 15 * time.Minute
)

func RandomID() ID {
	var id [20]byte
	b := make([]byte, 20)
	rand.Read(b)
	copy(id[:], b)
	return ID(id)
}

type OnPing interface {
	Ping(old []*Node, new *Node)
}

type byDistance struct {
	nodes  []*Node
	target ID
}

func (by *byDistance) Len() int {
	return len(by.nodes)
}

func (by *byDistance) Swap(i, j int) {
	by.nodes[i], by.nodes[j] = by.nodes[j], by.nodes[i]
}

func (by *byDistance) Less(i, j int) bool {
	d1 := by.nodes[i].Distance(by.target)
	d2 := by.nodes[j].Distance(by.target)
	return bytes.Compare(d1, d2) == -1
}

type Table struct {
	localID ID
	k       int
	root    *bucket
	onPing  OnPing
	rw      sync.RWMutex
}

func NewTable(numOfBucket int, localID ID) *Table {
	rt := &Table{localID: localID, k: numOfBucket}
	rt.root = createBucket()
	return rt
}

func (t *Table) Add(node *Node) {
	t.rw.Lock()
	b, bitIndex := t.locateBucket(node.id)
	if b.has(node.id) {
		t.rw.Unlock()
		return
	}
	if len(b.nodes) < t.k {
		b.add(node)
		t.rw.Unlock()
		return
	}
	if b.dontSplit {
		if old := b.stale(); t.onPing != nil && len(old) > 0 {
			go t.onPing.Ping(old, node)
		}
		t.rw.Unlock()
		return
	}
	b.split(bitIndex)
	b.farChild(t.localID, bitIndex).dontSplit = true
	t.rw.Unlock()

	t.Add(node)
}

func (t *Table) OnPing(op OnPing) {
	t.onPing = op
}

func (t *Table) Has(id ID) bool {
	t.rw.RLock()
	defer t.rw.RUnlock()
	b, _ := t.locateBucket(id)
	return b.has(id)
}

func (t *Table) Remove(id ID) {
	t.rw.Lock()
	defer t.rw.Unlock()
	b, _ := t.locateBucket(id)
	b.remove(id)
}

func (t *Table) Count() int {
	return len(t.Dump())
}

func (t *Table) Touch(id ID) {
	t.rw.Lock()
	defer t.rw.Unlock()
	bucket, _ := t.locateBucket(id)
	bucket.touch()
	if node := bucket.get(id); node != nil {
		node.touch()
	}
}

func (t *Table) Load(nodes []*Node) {
	for _, node := range nodes {
		t.Add(node)
	}
}

func (t *Table) Dump() []*Node {
	t.rw.RLock()
	defer t.rw.RUnlock()
	nodes := make([]*Node, 0)
	buckets := []*bucket{t.root}
	var bucket *bucket
	for len(buckets) > 0 {
		bucket, buckets = buckets[0], buckets[1:len(buckets)]
		if bucket.nodes == nil {
			buckets = append(buckets, bucket.left, bucket.right)
		} else {
			nodes = append(nodes, bucket.nodes...)
		}
	}
	return nodes
}

func (t *Table) Closest(target ID, limit int) []*Node {
	t.rw.RLock()
	defer t.rw.RUnlock()
	bitIndex := 0
	buckets := []*bucket{t.root}
	nodes := make([]*Node, 0, limit)
	var bucket *bucket
	for len(buckets) > 0 && len(nodes) < limit {
		bucket, buckets = buckets[len(buckets)-1], buckets[:len(buckets)-1]
		if bucket.nodes == nil {
			child := bucket.nearChild(target, bitIndex)
			if child == bucket.left {
				buckets = append(buckets, bucket.right)
			} else {
				buckets = append(buckets, bucket.left)
			}
			buckets = append(buckets, child)
			bitIndex++
		} else {
			nodes = append(nodes, bucket.nodes...)
		}
	}
	if length := len(nodes); limit > length {
		limit = length
	}
	sort.Sort(&byDistance{target: target, nodes: nodes})
	return nodes[:limit]
}

func (t *Table) Fresh() {
	if t.onPing == nil {
		return
	}
	t.rw.RLock()
	defer t.rw.RUnlock()
	buckets := []*bucket{t.root}
	var bucket *bucket
	now := time.Now()
	for len(buckets) > 0 {
		bucket, buckets = buckets[len(buckets)-1], buckets[:len(buckets)-1]
		if bucket.nodes == nil {
			buckets = append(buckets, bucket.right, bucket.left)
			continue
		}
		if now.Sub(bucket.lastUpdated) < expiredAfter {
			continue
		}
		if old := bucket.stale(); len(old) > 0 {
			go t.onPing.Ping(old, nil)
		}
	}
}

func (t *Table) locateBucket(id ID) (bucket *bucket, bitIndex int) {
	bucket = t.root
	for bucket.nodes == nil {
		bucket = bucket.nearChild(id, bitIndex)
		bitIndex++
	}
	return
}
