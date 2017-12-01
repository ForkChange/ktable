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


#### type Contact

```go
type Contact interface {
	ID() ID
	Address() net.UDPAddr
	Update()
	LastChanged() time.Time
}
```


#### type ID

```go
type ID [20]byte
```


#### type OnFindNode

```go
type OnFindNode interface {
	FindNode(contacts []Contact)
}
```


#### type OnPing

```go
type OnPing interface {
	Ping(doubtful []Contact, new Contact)
}
```


#### type Table

```go
type Table struct {
	ExpiredAfter time.Duration
	LocalID      ID
	NumOfBucket  int
	OnPing       OnPing
	OnFindNode   OnFindNode
}
```


#### func  New

```go
func New(localID ID, of OnFindNode, op OnPing, options ...func(*Table)) *Table
```

#### func (*Table) Add

```go
func (t *Table) Add(contact Contact)
```

#### func (*Table) Closest

```go
func (t *Table) Closest(target ID, limit int) []Contact
```

#### func (*Table) Count

```go
func (t *Table) Count() int
```

#### func (*Table) Dump

```go
func (t *Table) Dump() []Contact
```

#### func (*Table) Has

```go
func (t *Table) Has(id ID) bool
```

#### func (*Table) Refresh

```go
func (t *Table) Refresh()
```

#### func (*Table) Remove

```go
func (t *Table) Remove(id ID)
```

#### func (*Table) Update

```go
func (t *Table) Update(id ID)
```
