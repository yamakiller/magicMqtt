package sessions

import (
	codec "github.com/yamakiller/magicMqtt/encoding/message"
	"github.com/yamakiller/magicMqtt/message"
)

//Session broker connecion session
type Session struct {
	_offlineQueue  []codec.Message
	_inflightTable *message.Table
}

//PushOfflineMessage 插入离线消息
func (slf *Session) PushOfflineMessage(msg codec.Message) {
	slf._offlineQueue = append(slf._offlineQueue, msg)
}

//PushInFlight 插入等待应答
func (slf *Session) PushInFlight(msg *codec.Publish) uint16 {
	id := slf._inflightTable.NewID()
	msg.PacketIdentifier = id
	slf._inflightTable.Register(id, msg, nil)
	return id
}
