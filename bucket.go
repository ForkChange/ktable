package ktable

import (
	"bytes"
	"time"
)

type bucket struct {
	nodes       []*Node
	left        *bucket
	right       *bucket
	lastUpdated time.Time
	dontSplit   bool
}

func createBucket() *bucket {
	k := &bucket{nodes: []*Node{}}
	k.touch()
	return k
}

func (b *bucket) touch() {
	b.lastUpdated = time.Now()
}

func (b *bucket) add(node *Node) {
	b.nodes = append(b.nodes, node)
	b.touch()
}

func (b *bucket) split(bitIndex int) {
	b.left = createBucket()
	b.right = createBucket()
	for _, c := range b.nodes {
		b.nearChild(c.id, bitIndex).add(c)
	}
	b.nodes = nil
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
	c := b.nearChild(id, bitIndex)
	if c == b.right {
		return b.left
	}
	return b.right
}

func (b *bucket) remove(id ID) {
	index := b.indexOf(id)
	if index >= 0 {
		b.nodes[index] = b.nodes[len(b.nodes)-1]
		b.nodes = b.nodes[:len(b.nodes)-1]
	}
}

func (b *bucket) indexOf(id ID) int {
	for i, c := range b.nodes {
		if bytes.Equal(c.id[:], id[:]) {
			return i
		}
	}
	return -1
}

func (b *bucket) get(id ID) *Node {
	index := b.indexOf(id)
	if index >= 0 {
		return b.nodes[index]
	}
	return nil
}

func (b *bucket) stale() []*Node {
	nodes := make([]*Node, 0)
	now := time.Now()
	for _, node := range b.nodes {
		if now.Sub(node.lastUpdated) > expiredAfter {
			nodes = append(nodes, node)
		}
	}
	return nodes
}
