# ktable

The DHT's routing table implementation in Golang.

## Install
```bash
go get "github.com/fanpei91/ktable"
```

## Usage
```go
import "github.com/fanpei91/ktable"
```

#### type ID

```go
type ID [20]byte
```


#### func  RandomID

```go
func RandomID() ID
```

#### type Node

```go
type Node struct {
}
```


#### func  NewNode

```go
func NewNode(address *net.UDPAddr, id ID) *Node
```

#### func (*Node) Distance

```go
func (n *Node) Distance(target ID) []byte
```

#### func (*Node) String

```go
func (n *Node) String() string
```
Node ID + IP + BigEndian(Port)

#### type OnPing

```go
type OnPing interface {
	Ping(old []*Node, new *Node)
}
```


#### type Table

```go
type Table struct {
}
```


#### func  NewTable

```go
func NewTable(numOfBucket int, localID ID) *Table
```

#### func (*Table) Add

```go
func (t *Table) Add(node *Node)
```

#### func (*Table) Closest

```go
func (t *Table) Closest(target ID, limit int) []*Node
```

#### func (*Table) Count

```go
func (t *Table) Count() int
```

#### func (*Table) Dump

```go
func (t *Table) Dump() []*Node
```

#### func (*Table) Fresh

```go
func (t *Table) Fresh()
```

#### func (*Table) Has

```go
func (t *Table) Has(id ID) bool
```

#### func (*Table) Load

```go
func (t *Table) Load(nodes []*Node)
```

#### func (*Table) OnPing

```go
func (t *Table) OnPing(op OnPing)
```

#### func (*Table) Remove

```go
func (t *Table) Remove(id ID)
```

#### func (*Table) Touch

```go
func (t *Table) Touch(id ID)
```
