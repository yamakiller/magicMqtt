package message

import (
	"errors"
	"math"
	"sync"
	"time"

	codec "github.com/yamakiller/magicMqtt/encoding/message"
)

//NewTable Create an message table and Returns
func NewTable() *Table {
	return &Table{
		_id:   1,
		_hash: make(map[uint16]*Context),
		_used: make(map[uint16]bool),
	}
}

//Table message table
type Table struct {
	sync.RWMutex

	OnFinish func(uint16, codec.Message, interface{})
	_id      uint16
	_hash    map[uint16]*Context
	_used    map[uint16]bool
}

//WithOnFinish Set OnFinish event callback function
func (slf *Table) WithOnFinish(cb func(uint16, codec.Message, interface{})) {
	slf.OnFinish = cb
}

//NewID ID Inc and Returns
func (slf *Table) NewID() uint16 {
	slf.Lock()
	defer slf.Unlock()
	if slf._id == math.MaxUint16 {
		slf._id = 0
	}

	var id uint16
	ok := false
	for !ok {
		if _, ok = slf._used[slf._id]; !ok {
			id = slf._id
			slf._used[slf._id] = true
			slf._id++
			break
		}
	}

	return id
}

//Clean clear all message context
func (slf *Table) Clean() {
	slf.Lock()
	defer slf.Unlock()
	slf._hash = make(map[uint16]*Context)
}

//Get Return message
func (slf *Table) Get(id uint16) (codec.Message, error) {
	slf.RLock()
	defer slf.RUnlock()
	if v, ok := slf._hash[id]; ok {
		return v.Message, nil
	}

	return nil, errors.New("not fount")
}

//Register register an message
func (slf *Table) Register(id uint16, message codec.Message, opaque interface{}) {
	slf.Lock()
	defer slf.Unlock()
	slf._hash[id] = &Context{
		Message: message,
		_ref:    1,
		Created: time.Now(),
		Updated: time.Now(),
		Opaque:  opaque,
	}
}

//Register2 register an message
func (slf *Table) Register2(id uint16, message codec.Message, ref int, opaque interface{}) {
	slf.Lock()
	defer slf.Unlock()
	slf._hash[id] = &Context{
		Message: message,
		_ref:    ref,
		Created: time.Now(),
		Updated: time.Now(),
		Opaque:  opaque,
	}
}

//UnRef dec ref of numbe
func (slf *Table) UnRef(id uint16) {
	slf.Lock()
	defer slf.Unlock()

	if v, ok := slf._hash[id]; ok {
		v._ref--

		if v._ref < 1 {
			if slf.OnFinish != nil {
				slf.OnFinish(id, slf._hash[id].Message, slf._hash[id].Opaque)
			}
			delete(slf._used, id)
			delete(slf._hash, id)
		}
	}
}

//Remove delete in hash table
func (slf *Table) Remove(id uint16) {
	slf.Lock()
	defer slf.Unlock()

	if _, ok := slf._hash[id]; ok {
		delete(slf._hash, id)
	}

}
