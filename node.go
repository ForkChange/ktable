package ktable

import (
	"bytes"
	"encoding/binary"
	"net"
	"time"
)

type ID [20]byte

type Node struct {
	address     *net.UDPAddr
	id          ID
	lastUpdated time.Time
}

func NewNode(address *net.UDPAddr, id ID) *Node {
	return &Node{address, id, time.Now()}
}

func (n *Node) Distance(target ID) []byte {
	d := make([]byte, 20)
	for i := range n.id {
		d[i] = n.id[i] ^ target[i]
	}
	return d
}

// Node ID + IP + BigEndian(Port)
func (n *Node) String() string {
	buf := bytes.NewBuffer(make([]byte, 26))
	buf.Write(n.id[:])
	buf.Write(n.address.IP)
	binary.Write(buf, binary.BigEndian, n.address.Port)
	return buf.String()
}

func (n *Node) touch() {
	n.lastUpdated = time.Now()
}
