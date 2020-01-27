package server

import (
	"errors"
	"time"

	"github.com/yamakiller/magicMqtt/encoding"
	"github.com/yamakiller/magicMqtt/encoding/message"
	"github.com/yamakiller/magicMqtt/network"
)

//HandleConnection 处理句柄
func HandleConnection(conn *ConBroker) {
	conn._wg.Add(1)
	go func() {
		for {
			_, err := conn.ParseMessage()
			if conn._state == network.StateClosed {
				err = errors.New("error disconnect")
			}

			if err != nil {
				conn.Close()
				return
			}
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
						conn.Error("Write buffer error, %s", err.Error())
					}
					conn.invalidateTimer()
					conn._activity = time.Now()
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
