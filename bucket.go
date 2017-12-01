package ktable

import (
	"bytes"
	"time"
)

type bucket struct {
	contacts    []Contact
	left        *bucket
	right       *bucket
	lastChanged time.Time
	dontSplit   bool
}

func createBucket() *bucket {
	k := &bucket{contacts: []Contact{}}
	k.update()
	return k
}

func (b *bucket) update() {
	b.lastChanged = time.Now()
}

func (b *bucket) add(contact Contact) {
	b.contacts = append(b.contacts, contact)
	b.update()
}

func (b *bucket) split(bitIndex int) {
	b.left = createBucket()
	b.right = createBucket()
	for _, c := range b.contacts {
		b.nearChild(c.ID(), bitIndex).add(c)
	}
	b.contacts = nil
}

func (b *bucket) nearChild(id ID, bitIndex int) *bucket {
	bitIndexWithinByte := bitIndex % 8
	desiredByte := id[bitIndex/8]
	if desiredByte&(1<<(uint(7-bitIndexWithinByte))) == 1 {
		return b.right
	}
	return b.left
}

func (b *bucket) has(id ID) bool {
	return b.indexOf(id) >= 0
}

func (b *bucket) farChild(id ID, bitIndex int) *bucket {
	if c := b.nearChild(id, bitIndex); c == b.right {
		return b.left
	}
	return b.right
}

func (b *bucket) remove(id ID) {
	if index := b.indexOf(id); index > -1 {
		b.contacts[index] = b.contacts[len(b.contacts)-1]
		b.contacts = b.contacts[:len(b.contacts)-1]
	}
}

func (b *bucket) indexOf(id ID) int {
	for i, c := range b.contacts {
		if b.equal(c.ID(), id) {
			return i
		}
	}
	return -1
}

func (b *bucket) find(id ID) Contact {
	if index := b.indexOf(id); index > -1 {
		return b.contacts[index]
	}
	return nil
}

func (b *bucket) equal(first ID, second ID) bool {
	return bytes.Equal(first[:], second[:])
}
