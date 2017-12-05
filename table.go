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

type ID [20]byte

type Contact interface {
	ID() ID
	Address() net.UDPAddr
	Update()
	LastChanged() time.Time
}

type OnPing interface {
	Ping(doubtful []Contact, new Contact)
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
	d1 := by.distance(by.contacts[i].ID())
	d2 := by.distance(by.contacts[j].ID())
	return bytes.Compare(d1, d2) == -1
}

func (by *byDistance) distance(id ID) []byte {
	d := make([]byte, 20)
	for i := range id {
		d[i] = id[i] ^ by.target[i]
	}
	return d
}

type Table struct {
	expiredAfter   time.Duration
	localID        ID
	numOfPerBucket int
	onPing         OnPing
	onFindNode     OnFindNode
	refreshPeriod  time.Duration
	root           *bucket
	rw             sync.RWMutex
}

type option func(*Table)

func NumOfPerBucket(n int) option {
	return func(t *Table) {
		t.numOfPerBucket = n
	}
}

func ExpiredAfter(d time.Duration) option {
	return func(t *Table) {
		t.expiredAfter = d
	}
}

func RefreshPeriod(d time.Duration) option {
	return func(t *Table) {
		t.refreshPeriod = d
	}
}

func New(localID ID, of OnFindNode, op OnPing, options ...option) *Table {
	rt := &Table{
		localID:        localID,
		numOfPerBucket: 20,
		onPing:         op,
		onFindNode:     of,
		expiredAfter:   15 * time.Minute,
		refreshPeriod:  1 * time.Minute,
		root:           createBucket(),
	}
	for _, option := range options {
		option(rt)
	}
	go func() {
		for range time.Tick(rt.refreshPeriod) {
			rt.Refresh()
		}
	}()
	return rt
}

func (t *Table) Add(contact Contact) {
	t.rw.Lock()
	b, bitIndex := t.locateBucket(contact.ID())
	if b.has(contact.ID()) {
		t.rw.Unlock()
		return
	}
	if len(b.contacts) < t.numOfPerBucket {
		b.add(contact)
		t.rw.Unlock()
		return
	}
	if b.dontSplit {
		if contacts := t.doubtful(b); len(contacts) > 0 {
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
		if now.Sub(bucket.lastChanged) > t.expiredAfter {
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

func (t *Table) doubtful(b *bucket) []Contact {
	contacts := make([]Contact, 0)
	now := time.Now()
	for _, contact := range b.contacts {
		if now.Sub(contact.LastChanged()) > t.expiredAfter {
			contacts = append(contacts, contact)
		}
	}
	return contacts
}
