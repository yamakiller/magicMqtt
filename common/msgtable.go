package common

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/yamakiller/magicMqtt/encoding/message"
)

//MessageContainer 消息内容句柄
type MessageContainer struct {
	_message message.Message
	_ref     int
	_created time.Time
	_updated time.Time
	_opaque  interface{}
}

//MessageTable 消息
type MessageTable struct {
	sync.RWMutex
	_id       uint16
	_hash     map[uint16]*MessageContainer
	_onFinish func(uint16, message.Message, interface{})
	_used     map[uint16]bool
}

//NewMessageTable 创建一个消息确认表
func NewMessageTable() *MessageTable {
	return &MessageTable{
		_id:   1,
		_hash: make(map[uint16]*MessageContainer),
		_used: make(map[uint16]bool),
	}
}

//WithOnFinish 设置消息完成回掉调函数
func (slf *MessageTable) WithOnFinish(callback func(uint16, message.Message, interface{})) {
	slf._onFinish = callback
}

//NewID 创建一个新的ID
func (slf *MessageTable) NewID() uint16 {
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

//Clean 清除
func (slf *MessageTable) Clean() {
	slf.Lock()
	defer slf.Unlock()
	slf._hash = make(map[uint16]*MessageContainer)
}

//Get 返回一个消息
func (slf *MessageTable) Get(id uint16) (message.Message, error) {
	slf.RLock()
	defer slf.RUnlock()

	if v, ok := slf._hash[id]; ok {

		return v._message, nil
	}

	return nil, errors.New("not found")
}

//Register 注册一个消息
func (slf *MessageTable) Register(id uint16, msg message.Message, opaque interface{}) {
	slf.Lock()
	defer slf.Unlock()

	slf._hash[id] = &MessageContainer{
		_message: msg,
		_ref:     1,
		_created: time.Now(),
		_updated: time.Now(),
		_opaque:  opaque,
	}
}

//Register2 注册一个消息并设置计数器
func (slf *MessageTable) Register2(id uint16, msg message.Message, count int, opaque interface{}) {
	slf.Lock()
	defer slf.Unlock()

	slf._hash[id] = &MessageContainer{
		_message: msg,
		_ref:     count,
		_created: time.Now(),
		_updated: time.Now(),
		_opaque:  opaque,
	}
}

//Unref 取消一个消息的引用
func (slf *MessageTable) Unref(id uint16) {
	slf.Lock()
	defer slf.Unlock()

	if v, ok := slf._hash[id]; ok {
		v._ref--

		if v._ref < 1 {
			if slf._onFinish != nil {
				slf._onFinish(id, slf._hash[id]._message, slf._hash[id]._opaque)
			}
			delete(slf._used, id)
			delete(slf._hash, id)
		}
	}
}

//Remove 移除一个消息
func (slf *MessageTable) Remove(id uint16) {
	slf.Lock()
	defer slf.Unlock()

	if _, ok := slf._hash[id]; ok {
		delete(slf._hash, id)
	}
}
