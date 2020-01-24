package server

import (
	"bufio"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/yamakiller/magicLibs/util"
	"github.com/yamakiller/magicMqtt/dashboard"
	"github.com/yamakiller/magicMqtt/encoding"
	codec "github.com/yamakiller/magicMqtt/encoding/message"
	"github.com/yamakiller/magicMqtt/sessions"
)

//State Broker Connection 状态
type State int32

//   ConnectionState: Idle, Send, Receive and Handshake?
const (
	StateInit State = iota
	StateConnecting
	StateConnected
	StateAccepted
	StateIdle
	StateDetached
	StateSend
	StateReceive
	StateShutdown
	StateClosed
)

var (
	//ErrDisconnect 连接断开错误
	ErrDisconnect = errors.New("error disconnect")
)

//newConnection 创建一个连接者
func newConnection() *BrokerConnection {
	c := &BrokerConnection{
		_events:    make(map[string]interface{}),
		_queue:     make(chan codec.Message, dashboard.Instance().BrokerQueueSize),
		_keepalive: dashboard.Instance().BrokerKeepalive,
		_state:     StateInit,
		_closed:    make(chan bool),
		_activity:  time.Now(),
	}

	c._events["connected"] = func() {
		c._state = StateConnected
	}

	c._events["connect"] = func(msg *codec.Connect) {
		if msg.Identifier == "" {
			msg.Identifier = util.SpawnUUID()
		}

		err := dashboard.Instance().Auth(msg.Identifier,
			string(msg.UserName),
			string(msg.Password))
		if err != nil {
			//回复connack包
			return
		}

		if msg.CleanSession {
			c._session = dashboard.Instance().Sessions().New(msg.Identifier)
		} else {
			session, ok := dashboard.Instance().Sessions().GetOrNew(msg.Identifier)
			if ok {
				//关闭旧的连接
				session.Disconnect()
			}

			//设置新的连接关联
			session.WithDisconnect(c.termination)
			c._session = session
		}

		if msg.Will != nil {

		}
	}

	/*c._events["connack"] = func(result uint8) {
		if result == 0 {
			c._state = StateConnected
			if
		}
	}*/

	return c
}

//HandleConnection Broker 连接者消息处理
func HandleConnection(conn *BrokerConnection) {
	conn._wgw.Add(2)
	go func() {
		defer conn._wgw.Done()
		for {
			select {
			case <-conn._closed:
				return
			case <-conn._kicker.C:
				conn.Ping()
				conn.invalidateTimer()
			case msg := <-conn._queue:
				state := conn._state
				if state == StateConnected || state == StateConnecting {
					if msg.GetType() == encoding.PTypePublish {
						sb := msg.(*codec.Publish)
						if sb.QosLevel < 0 {
							dashboard.Instance().LError("%d %s => QoS under zero %#v",
								conn._id,
								conn._addr,
								sb)
							break
						}

						if sb.QosLevel > 0 {
							sb.PacketIdentifier = conn._session.PushInFlight(sb)
						}
					}

					err := conn.writeMessage(msg)
					if err != nil {
						if v, ok := conn._events["error"].(func(error)); ok {
							v(err)
						}
					}
					conn.invalidateTimer()
				} else {
					//存入离线缓冲
					conn.offlinePush(msg)
				}
				//输出到网络
				b := conn._writer.Buffered()
				if b > 0 {
					err := conn._writer.Flush()
					if err != nil {
						dashboard.Instance().LError("%d %s => %s",
							conn._id,
							conn._addr,
							err.Error())
					}
				}
			}
		}
	}()

	go func() {
		defer conn._wgw.Done()
		for {
			_, err := conn.parseMessage()
			if conn._state == StateClosed {
				err = ErrDisconnect
			}

			if err != nil {
				if err != ErrDisconnect {
					conn.Will()
					dashboard.Instance().LError("%d %s => Error: %s",
						conn._id,
						conn._addr,
						err.Error())
				}
				//如果不是主动关闭连接，需要发送断开连接消息
				conn.Close()
				return
			}
		}
	}()

}

//BrokerConnection mqtt 连接者对象
type BrokerConnection struct {
	_id          int64
	_addr        string
	_handle      io.ReadWriteCloser
	_session     *sessions.Session
	_reader      *bufio.Reader
	_writer      *bufio.Writer
	_willMessage *codec.Will
	_kicker      time.Timer
	_keepalive   int
	_queue       chan codec.Message
	_closed      chan bool
	_connected   bool
	_events      map[string]interface{}
	_state       State
	_wgw         sync.WaitGroup
	_activity    time.Time
}

//WithHandle 设置连接者IO句柄
func (slf *BrokerConnection) WithHandle(c io.ReadWriteCloser) {
	slf._handle = c
	slf._reader = bufio.NewReaderSize(slf._handle, dashboard.Instance().BrokerBufferSize)
	slf._writer = bufio.NewWriterSize(slf._handle, dashboard.Instance().BrokerBufferSize)
	slf._state = StateConnected
}

//WithID 设置连接者唯一编号
func (slf *BrokerConnection) WithID(id int64) {
	slf._id = id
}

//WithAddr 设置连接者的地址
func (slf *BrokerConnection) WithAddr(remote string) {
	slf._addr = remote
}

//GetID 返回连接者的唯一编号
func (slf *BrokerConnection) GetID() int64 {
	return slf._id
}

//GetAddr 返回连接者的地址
func (slf *BrokerConnection) GetAddr() string {
	return slf._addr
}

//GetRealAddr 返回连接者的当前地址信息
func (slf *BrokerConnection) GetRealAddr() string {
	return slf._handle.(net.Conn).RemoteAddr().String()
}

//Will 发送遗嘱消息，如果有
func (slf *BrokerConnection) Will() {
	if slf._willMessage == nil {
		return
	}
	will := slf._willMessage
	msg := codec.SpawnPublishMessage()
	msg.TopicName = will.Topic
	msg.Payload = []byte(will.Message)
	msg.QosLevel = int(will.Qos)
}

//Ping 发送Ping消息
func (slf *BrokerConnection) Ping() {
	if slf._state == StateClosed {
		return
	}
	slf._queue <- codec.SpawnPingreqMessage()
}

func (slf *BrokerConnection) Publish(msg *codec.Publish) {
	if len(msg.TopicName) < 1 {
		return
	}

	if msg.Retain > 0 {
		if len(msg.Payload) == 0 {
			//删除主题对应的消息
			dashboard.Instance().LDebug("%d %s => Deleted retain %s", msg.TopicName)
			return
		} else {
			//更新主题对应消息
		}

		//获取msg.TopicName主题目标
		//遍历群发
	}
}

//writeMessage 写消息
func (slf *BrokerConnection) writeMessage(msg codec.Message) error {
	slf._activity = time.Now()
	_, e := msg.WriteTo(slf._writer)
	slf._activity = time.Now()
	return e
}

func (slf *BrokerConnection) parseMessage() (codec.Message, error) {
	message, err := codec.Parse(slf._reader, dashboard.Instance().BrokerMessageLimt)
	if err != nil {
		dashboard.Instance().LDebug("%d %s => Message: %s\n", slf._id, slf._addr, err)
		if v, ok := slf._events["error"].(func(error)); ok {
			v(err)
		}
		return nil, err
	}

	dashboard.Instance().LDebug("%d %s => Read Message: [%s] %+v",
		slf._id,
		slf._addr,
		message.GetTypeAsString(), message)

	if v, ok := slf._events["parsed"]; ok {
		if cb, ok := v.(func()); ok {
			cb()
		}
	}

	switch message.GetType() {
	case encoding.PTypePublish:
		p := message.(*codec.Publish)
		if v, ok := slf._events["publish"]; ok {
			if cb, ok := v.(func(*codec.Publish)); ok {
				cb(p)
			}
		}
		break
	case encoding.PTypeConnack:
		p := message.(*codec.Connack)
		if v, ok := slf._events["connack"]; ok {
			if cb, ok := v.(func(uint8)); ok {
				cb(p.ReturnCode)
			}
		}
		break

	case encoding.PTypePuback:
		p := message.(*codec.Puback)
		if v, ok := slf._events["puback"]; ok {
			if cb, ok := v.(func(uint16)); ok {
				cb(p.PacketIdentifier)
			}
		}
		break

	case encoding.PTypePubrec:
		p := message.(*codec.Pubrec)
		if v, ok := slf._events["pubrec"]; ok {
			if cb, ok := v.(func(uint16)); ok {
				cb(p.PacketIdentifier)
			}
		}
		break

	case encoding.PTypePubrel:
		p := message.(*codec.Pubrel)
		if v, ok := slf._events["pubrel"]; ok {
			if cb, ok := v.(func(uint16)); ok {
				cb(p.PacketIdentifier)
			}
		}
		break
	case encoding.PTypePubcomp:
		p := message.(*codec.Pubcomp)
		if v, ok := slf._events["pubcomp"]; ok {
			if cb, ok := v.(func(uint16)); ok {
				cb(p.PacketIdentifier)
			}
		}
		break
	case encoding.PTypePingreq:
		if v, ok := slf._events["pingreq"]; ok {
			if cb, ok := v.(func()); ok {
				cb()
			}
		}
		break

	case encoding.PTypePingresp:
		if v, ok := slf._events["pingresp"]; ok {
			if cb, ok := v.(func()); ok {
				cb()
			}
		}
		break

	case encoding.PTypeSuback:
		p := message.(*codec.Suback)
		if v, ok := slf._events["suback"]; ok {
			if cb, ok := v.(func(uint16, int)); ok {
				cb(p.PacketIdentifier, 0)
			}
		}
		break

	case encoding.PTypeUnsuback:
		p := message.(*codec.Unsuback)
		if v, ok := slf._events["unsuback"]; ok {
			if cb, ok := v.(func(uint16)); ok {
				cb(p.PacketIdentifier)
			}
		}
		break

	case encoding.PTypeConnect:
		p := message.(*codec.Connect)
		if v, ok := slf._events["connect"]; ok {
			if cb, ok := v.(func(*codec.Connect)); ok {
				cb(p)
			}
		}
		break

	case encoding.PTypeSubscribe:
		p := message.(*codec.Subscribe)
		if v, ok := slf._events["subscribe"]; ok {
			if cb, ok := v.(func(*codec.Subscribe)); ok {
				cb(p)
			}
		}
		break
	case encoding.PTypeDisconnect:
		if v, ok := slf._events["disconnect"]; ok {
			if cb, ok := v.(func()); ok {
				cb()
			}
		}
		break
	case encoding.PTypeUnsubscribe:
		p := message.(*codec.Unsubscribe)
		if v, ok := slf._events["unsubscribe"]; ok {
			if cb, ok := v.(func(uint16, int, []codec.SubscribePayload)); ok {
				cb(p.PacketIdentifier, 0, p.Payload)
			}
		}
		break
	default:
		dashboard.Instance().LError("%d %s => Unhandled message: %+v",
			slf._id,
			slf._addr,
			message)
	}

	slf.invalidateTimer()
	slf._activity = time.Now()
	return message, err
}

func (slf *BrokerConnection) invalidateTimer() {
	slf._kicker.Reset(time.Second * time.Duration(slf._keepalive))
}

func (slf *BrokerConnection) offlinePush(msg codec.Message) {
	switch msg.GetType() {
	case encoding.PTypeConnect:
	case encoding.PTypeConnack:
	case encoding.PTypeDisconnect:
	case encoding.PTypePingreq:
	case encoding.PTypePingresp:
	default:
		if slf._session != nil {
			slf._session.PushOfflineMessage(msg)
		} else {
			dashboard.Instance().LError("%d %s => offline message lost :%#v",
				msg)
		}
	}
}

func (slf *BrokerConnection) termination() {
	slf.Close()
}

//Close 关闭连接
func (slf *BrokerConnection) Close() error {
	slf._state = StateClosed
	slf._closed <- true
	err := slf._handle.Close()
	slf._wgw.Wait()
	close(slf._closed)

	for msg := range slf._queue {
		slf.offlinePush(msg)
	}
	//是否销毁
	slf._session = nil

	return err
}
