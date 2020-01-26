package sessions

import (
	"errors"
	"sync"

	"github.com/yamakiller/magicMqtt/common"
	"github.com/yamakiller/magicMqtt/encoding/message"
)

func newSession() *Session {
	return &Session{
		_waitAck:      common.NewMessageTable(),
		_offlineQueue: make([]message.Message, 0),
	}
}

//Session 连接会话状态
type Session struct {
	_clientid     string
	_onWrite      func(message.Message) error
	_offlineQueue []message.Message
	_waitAck      *common.MessageTable
	_offlineLimit int
	_onDisconnect func()
	_sync         sync.Mutex
}

//WithClientID 设置client id
func (slf *Session) WithClientID(id string) {
	slf._clientid = id
}

//GetClientID 返回ClientID
func (slf *Session) GetClientID() string {
	return slf._clientid
}

//WithOnDisconnect 设置断开连接函数
func (slf *Session) WithOnDisconnect(f func()) {
	slf._sync.Lock()
	defer slf._sync.Unlock()
	slf._onDisconnect = f
}

//DoDisconnect 执行断开连接操作
func (slf *Session) DoDisconnect() {
	slf._sync.Lock()
	defer slf._sync.Unlock()
	if slf._onDisconnect == nil {
		return
	}
	slf._onDisconnect()
}

//WithOnWrite 设置写数据函数
func (slf *Session) WithOnWrite(f func(message.Message) error) {
	slf._sync.Lock()
	defer slf._sync.Unlock()
	slf._onWrite = f
}

//WriteMessage 写消息数据
func (slf *Session) WriteMessage(msg message.Message) error {
	slf._sync.Lock()
	defer slf._sync.Unlock()
	if slf._onWrite == nil {
		if len(slf._offlineQueue) >= slf._offlineLimit {
			return errors.New("Offline Queue full")
		}
		slf._offlineQueue = append(slf._offlineQueue, msg)
		return nil
	}

	return slf._onWrite(msg)
}

//PushOfflineMessage 插入离线消息
func (slf *Session) PushOfflineMessage(msg message.Message) error {
	slf._sync.Lock()
	defer slf._sync.Unlock()
	if len(slf._offlineQueue) >= slf._offlineLimit {
		return errors.New("Offline Queue full")
	}
	slf._offlineQueue = append(slf._offlineQueue, msg)
	return nil
}

//OfflineMessages 返回所有离线消息
func (slf *Session) OfflineMessages() []message.Message {
	slf._sync.Lock()
	defer slf._sync.Unlock()
	nlen := len(slf._offlineQueue)
	rs := make([]message.Message, nlen)
	if nlen == 0 {
		return rs
	}
	for i, msg := range slf._offlineQueue {
		rs[i] = msg
	}
	slf._offlineQueue = slf._offlineQueue[nlen:]
	return rs
}

//WithOnFinish 设置消息完成回掉函数
func (slf *Session) WithOnFinish(callback func(uint16, message.Message, interface{})) {
	slf._waitAck.WithOnFinish(callback)
}

//RegisterMessage 注册一个消息到等待确认池
func (slf *Session) RegisterMessage(msg *message.Publish) uint16 {
	id := slf._waitAck.NewID()
	msg.PacketIdentifier = id
	slf._waitAck.Register(id, msg, nil)
	return id
}

//UnRefMessage 取消一个消息的引用
func (slf *Session) UnRefMessage(id uint16) {
	slf._waitAck.Unref(id)
}
