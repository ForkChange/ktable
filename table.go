/*
 * The DHT's routing table implementation in Golang.
 */
package ktable

import (
	"bytes"
	"net"
	"sort"
	"sync"
	"time"
)

const (
	expiredAfter = 15 * time.Minute
)

type ID [20]byte

type Contact interface {
	ID() ID
	Address() net.UDPAddr
	Update()
	LastChanged() time.Time
	Distance(target ID) []byte
	Equal(target ID) bool
}

type OnPing interface {
	Ping(questionable []Contact, new Contact)
}

type OnFindNode interface {
	FindNode(contacts []Contact)
}

type byDistance struct {
	contacts []Contact
	target   ID
}

func (by *byDistance) Len() int {
	return len(by.contacts)
}

func (by *byDistance) Swap(i, j int) {
	by.contacts[i], by.contacts[j] = by.contacts[j], by.contacts[i]
}

func (by *byDistance) Less(i, j int) bool {
	d1 := by.contacts[i].Distance(by.target)
	d2 := by.contacts[j].Distance(by.target)
	return bytes.Compare(d1, d2) == -1
}

type Table struct {
	localID    ID
	k          int
	root       *bucket
	onPing     OnPing
	onFindNode OnFindNode
	rw         sync.RWMutex
}

func New(localID ID, numOfBucket int, of OnFindNode, op OnPing) *Table {
	rt := &Table{
		localID:    localID,
		k:          numOfBucket,
		onPing:     op,
		onFindNode: of,
	}
	rt.root = createBucket()
	return rt
}

func (t *Table) Add(contact Contact) {
	t.rw.Lock()
	b, bitIndex := t.locateBucket(contact.ID())
	if b.has(contact.ID()) {
		t.rw.Unlock()
		return
	}
	if len(b.contacts) < t.k {
		b.add(contact)
		t.rw.Unlock()
		return
	}
	if b.dontSplit {
		if contacts := b.questionable(); len(contacts) > 0 {
			go t.onPing.Ping(contacts, contact)
		}
		t.rw.Unlock()
		return
	}
	b.split(bitIndex)
	b.farChild(t.localID, bitIndex).dontSplit = true
	t.rw.Unlock()

	t.Add(contact)
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
	t.rw.RLock()
	defer t.rw.RUnlock()
	n := 0
	for _, bucket := range t.nonEmptyBuckets() {
		n += len(bucket.contacts)
	}
	return n
}

func (t *Table) Update(id ID) {
	t.rw.Lock()
	defer t.rw.Unlock()
	bucket, _ := t.locateBucket(id)
	bucket.update()
	if contact := bucket.find(id); contact != nil {
		contact.Update()
	}
}

func (t *Table) Dump() []Contact {
	t.rw.RLock()
	defer t.rw.RUnlock()
	contacts := make([]Contact, 0)
	for _, bucket := range t.nonEmptyBuckets() {
		contacts = append(contacts, bucket.contacts...)
	}
	return contacts
}

func (t *Table) Closest(target ID, limit int) []Contact {
	t.rw.RLock()
	defer t.rw.RUnlock()
	bitIndex := 0
	buckets := []*bucket{t.root}
	contacts := make([]Contact, 0, limit)
	var bucket *bucket
	for len(buckets) > 0 && len(contacts) < limit {
		bucket, buckets = buckets[len(buckets)-1], buckets[:len(buckets)-1]
		if bucket.contacts == nil {
			near := bucket.nearChild(target, bitIndex)
			far := bucket.farChild(target, bitIndex)
			buckets = append(buckets, far, near)
			bitIndex++
		} else {
			contacts = append(contacts, bucket.contacts...)
		}
	}
	if length := len(contacts); limit > length {
		limit = length
	}
	sort.Sort(&byDistance{target: target, contacts: contacts})
	return contacts[:limit]
}

func (t *Table) Refresh() {
	t.rw.RLock()
	defer t.rw.RUnlock()
	now := time.Now()
	for _, bucket := range t.nonEmptyBuckets() {
		if now.Sub(bucket.lastChanged) > expiredAfter {
			go t.onFindNode.FindNode(bucket.contacts)
		}
	}
}

func (t *Table) nonEmptyBuckets() []*bucket {
	nonEmpty := make([]*bucket, 0)
	buckets := []*bucket{t.root}
	var bucket *bucket
	for len(buckets) > 0 {
		bucket, buckets = buckets[0], buckets[1:len(buckets)]
		if bucket.contacts == nil {
			buckets = append(buckets, bucket.left, bucket.right)
			continue
		}
		if len(bucket.contacts) > 0 {
			nonEmpty = append(nonEmpty, bucket)
		}
	}
	return nonEmpty
}

func (t *Table) locateBucket(id ID) (bucket *bucket, bitIndex int) {
	bucket = t.root
	for bucket.contacts == nil {
		bucket = bucket.nearChild(id, bitIndex)
		bitIndex++
	}
	return
}
