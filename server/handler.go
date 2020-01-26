package server

import (
	"github.com/yamakiller/magicMqtt/encoding"
	"github.com/yamakiller/magicMqtt/encoding/message"
	"github.com/yamakiller/magicMqtt/network"
)

//HandleConnection 处理句柄
func HandleConnection(conn *ConBroker) {
	conn._wg.Add(2)
	go func() {
		defer func() {
			conn._wg.Done()
			conn.Close()
		}()

		for {
			//接收消息

		}

	}()

	go func() {
		defer func() {
			conn._conn.Close()
			conn._wg.Done()
		}()
		for {
			select {
			case <-conn._closed:
				goto Exit
			case <-conn._kicker.C:
				conn.Kicker()
			case msg := <-conn._queue:
				state := conn._state
				if state == network.StateConnected ||
					state == network.StateConnecting {
					if msg.GetType() == encoding.PTypePublish {
						sb := msg.(*message.Publish)
						if sb.QosLevel > 0 {
							//注册
							session := conn._session
							if session != nil {
								sb.PacketIdentifier = session.RegisterMessage(sb)
							}
						}
					}

					if err := conn.write(msg); err != nil {
						//TODO: 记录发送错误日志，这个环境可能会丢失数据
					}
					conn.invalidateTimer()
				} else {
					ss := conn._session
					if ss != nil {
						ss.PushOfflineMessage(msg)
					}
				}
			}
		}
	Exit:
	}()
}
