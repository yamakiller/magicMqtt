package server

import (
	"bufio"
	"io"
	"net"
	"sync"
	"time"

	"github.com/yamakiller/magicMqtt/auth/code"
	"github.com/yamakiller/magicMqtt/topics"

	"github.com/yamakiller/magicMqtt/blackboard"
	"github.com/yamakiller/magicMqtt/common"
	"github.com/yamakiller/magicMqtt/sessions"

	"github.com/yamakiller/magicMqtt/encoding"
	"github.com/yamakiller/magicMqtt/encoding/message"
	"github.com/yamakiller/magicMqtt/network"
)

//NewBrokerConn 创建一个连接器
func NewBrokerConn() *ConBroker {
	c := &ConBroker{
		_keepalive:    blackboard.Instance().Deploy.Keepalive,
		_queue:        make(chan message.Message, blackboard.Instance().Deploy.MessageQueueSize),
		_closed:       make(chan bool),
		_subscription: make(map[string]*common.Subscription),
		_activity:     time.Now(),
		_state:        network.StateInit,
	}

	if c._keepalive > 0 {
		c._kicker = time.NewTimer(time.Duration(c._keepalive) * time.Second)
	}
	return c
}

//ConBroker 连接器
type ConBroker struct {
	_conn         io.ReadWriteCloser
	_queue        chan message.Message
	_kicker       *time.Timer
	_keepalive    int
	_id           int64
	_addr         string
	_reader       *bufio.Reader
	_writer       *bufio.Writer
	_session      *sessions.Session
	_willMsg      *message.Will
	_subscription map[string]*common.Subscription
	_rmsgs        []*message.Publish
	_closed       chan bool
	_connected    bool
	_cleanSession bool
	_ping         int
	_activity     time.Time
	_state        network.State
	_wg           sync.WaitGroup
}

//WithID 设置ID
func (slf *ConBroker) WithID(id int64) {
	slf._id = id
}

//GetID 返回ID
func (slf *ConBroker) GetID() int64 {
	return slf._id
}

//WithAddr 设置连接者地址
func (slf *ConBroker) WithAddr(remote string) {
	slf._addr = remote
}

//GetAddr 返回连接者地址
func (slf *ConBroker) GetAddr() string {
	return slf._addr
}

//WithSession 设置session对象
func (slf *ConBroker) WithSession(session *sessions.Session) {
	slf._session = session
}

//GetSession 返回关联的session对象
func (slf *ConBroker) GetSession() *sessions.Session {
	return slf._session
}

//WithConn 设置连接器关键通信句柄
func (slf *ConBroker) WithConn(conn io.ReadWriteCloser) {
	slf._conn = conn
	slf._reader = bufio.NewReaderSize(slf._conn, blackboard.Instance().Deploy.BufferSize)
	slf._writer = bufio.NewWriterSize(slf._conn, blackboard.Instance().Deploy.BufferSize)
	slf._state = network.StateConnected
}

func (slf *ConBroker) invalidateTimer() {
	if slf._kicker != nil {
		slf._kicker.Reset(time.Second * time.Duration(slf._keepalive))
	}
}

//ParseMessage 解析消息
func (slf *ConBroker) ParseMessage() (message.Message, error) {
	if slf._keepalive > 0 {
		if cn, ok := slf._conn.(net.Conn); ok {
			cn.SetReadDeadline(slf._activity.Add(time.Duration(int(float64(slf._keepalive)*1.5)) * time.Second))
		}
	}

	msg, err := message.Parse(slf._reader, blackboard.Instance().Deploy.MessageSize)
	if err != nil {
		return nil, err
	}

	switch msg.GetType() {
	case encoding.PTypePublish:
		slf.onPublish(msg.(*message.Publish))
		break
	case encoding.PTypePuback:
		slf.onPuback(msg.(*message.Puback))
		break
	case encoding.PTypePubrec:
		slf.onPubrec(msg.(*message.Pubrec))
		break
	case encoding.PTypePubrel:
		slf.onPubrel(msg.(*message.Pubrel))
		break
	case encoding.PTypePubcomp:
		slf.onPubcomp(msg.(*message.Pubcomp))
		break
	case encoding.PTypeSubscribe:
		slf.onSubscribe(msg.(*message.Subscribe))
		break
	case encoding.PTypeUnsubscribe:
		slf.onUnSubscribe(msg.(*message.Unsubscribe))
		break
	case encoding.PTypeConnect:
		slf.onConnect(msg.(*message.Connect))
		break
	case encoding.PTypeDisconnect:
		slf.onDisconnect(msg.(*message.Disconnect))
		break
	case encoding.PTypePingresp:
		slf.onPingresp(msg.(*message.Pingresp))
		break
	default:
		//报错误
	}

	slf.invalidateTimer()
	slf._activity = time.Now()
	return msg, err
}

//WriteMessage 写入消息
func (slf *ConBroker) WriteMessage(msg message.Message) error {
	slf._queue <- msg
	return nil
}

func (slf *ConBroker) write(msg message.Message) error {
	slf._activity = time.Now()
	_, err := message.WriteMessageTo(msg, slf._writer)
	if err != nil {
		return err
	}

	slf.flusher()
	slf._activity = time.Now()
	return nil
}

func (slf *ConBroker) flusher() {
	n := slf._writer.Buffered()
	if n > 0 {
		if err := slf._writer.Flush(); err != nil {
			//TODO: 记录日志
		}
	}
}

func (slf *ConBroker) onConnect(msg *message.Connect) {
	if slf._connected {
		//TODO:记录日志关闭连接
		return
	}

	name := string(msg.UserName)
	pwd := string(msg.Password)
	connack := message.SpawnConnackMessage()
	connack.ReturnCode = 0

	defer func() {
		if err := slf.WriteMessage(connack); err != nil {
			//TODO: 写入错误日志
		}
	}()

	if _, err := blackboard.Instance().Auth.Connect(msg.Identifier, name, pwd); err != nil {
		//TODO:记录日志回复connack包
		if err == code.ErrAuthClientNot {
			connack.ReturnCode = 0x05
		} else {
			connack.ReturnCode = 0x04
		}

		return
	}

	if msg.CleanSession {
		session := blackboard.Instance().Sessions.Get(msg.Identifier)
		if session != nil {
			session.DoDisconnect()
		}
		session = blackboard.Instance().Sessions.New(msg.Identifier)
	} else {
		session, org := blackboard.Instance().Sessions.GetOrNew(msg.Identifier)
		if org {
			session.DoDisconnect()
		}

		slf._session = session

		//重新注册主题
		ts, qos, err := slf._session.Topics()
		if err != nil {
			for i, topic := range ts {
				sub := &common.Subscription{
					Topic:  topic,
					Qos:    qos[i],
					Client: slf._session.GetClientID(),
				}

				slf._subscription[topic] = sub
				blackboard.Instance().Topics.Subscribe([]byte(topic), qos[i], sub)
			}
		}

		offlineMessage := slf._session.OfflineMessages()

		for _, offMsg := range offlineMessage {
			slf.WriteMessage(offMsg)
		}
	}

	slf._session.WithOnDisconnect(slf.Terminate)
	slf._session.WithOnWrite(slf.WriteMessage)
	slf._session.WithClientID(msg.Identifier)
	slf._cleanSession = msg.CleanSession

	if msg.Will != nil {
		slf._willMsg = msg.Will
	}

	slf._connected = true
}

func (slf *ConBroker) onDisconnect(msg *message.Disconnect) {
	if slf._session != nil {
		slf._session.WithOnDisconnect(nil)
		//TODO: 考虑删除创建的主题
		slf._willMsg = nil
	}
	slf.Close()
}

func (slf *ConBroker) onPublish(msg *message.Publish) {
	//topic := msg.TopicName
	switch byte(msg.QosLevel) {
	case topics.QosAtMostOnce:
		slf.procPublish(msg)
	case topics.QosAtLeastOnce:
		puback := message.SpawnPubackMessage()
		puback.PacketIdentifier = msg.PacketIdentifier
		if err := slf.WriteMessage(puback); err != nil {
			//TODO: 记录日志
			return
		}
		slf.procPublish(msg)
	case topics.QosExactlyOnce:
		pubrec := message.SpawnPubrecMessage()
		pubrec.PacketIdentifier = msg.PacketIdentifier
		if err := slf.WriteMessage(pubrec); err != nil {
			//TODO: 记录日志
			return
		}
		slf.procPublish(msg)
	default:
		//TODO: 记录日志
		return
	}
}

func (slf *ConBroker) onPuback(msg *message.Puback) {
	session := slf._session
	if session != nil {
		session.UnRefMessage(msg.PacketIdentifier)
	}
}

func (slf *ConBroker) onPubrec(msg *message.Pubrec) {
	ack := message.SpawnPubrelMessage()
	ack.PacketIdentifier = msg.PacketIdentifier
	slf.WriteMessage(ack)
}

func (slf *ConBroker) onPubrel(msg *message.Pubrel) {
	ack := message.SpawnPubcompMessage()
	ack.PacketIdentifier = msg.PacketIdentifier
	slf.WriteMessage(ack)
	session := slf._session
	if session != nil {
		session.UnRefMessage(msg.PacketIdentifier)
	}
}

func (slf *ConBroker) onPubcomp(msg *message.Pubcomp) {
	session := slf._session
	if session != nil {
		session.UnRefMessage(msg.PacketIdentifier)
	}
}

func (slf *ConBroker) onSubscribe(msg *message.Subscribe) {
	ts := msg.Payload

	suback := message.SpawnSubackMessage()
	suback.PacketIdentifier = msg.PacketIdentifier
	var retcodes []byte
	for _, topic := range ts {
		if oldSub, exist := slf._subscription[topic.TopicPath]; exist {
			blackboard.Instance().Topics.Unsubscribe([]byte(oldSub.Topic), oldSub)
			delete(slf._subscription, topic.TopicPath)
		}

		sub := &common.Subscription{
			Topic:  topic.TopicPath,
			Qos:    topic.RequestedQos,
			Client: slf._session.GetClientID(),
		}

		rqos, err := blackboard.Instance().Topics.Subscribe([]byte(topic.TopicPath),
			topic.RequestedQos, sub)
		if err != nil {
			//TODO: 记录日志
			retcodes = append(retcodes, topics.QosFailure)
			continue
		}

		slf._subscription[topic.TopicPath] = sub
		slf._session.AddTopics(topic.TopicPath, topic.RequestedQos)
		retcodes = append(retcodes, rqos)
		blackboard.Instance().Topics.Retained([]byte(topic.TopicPath), &slf._rmsgs)
	}
	suback.Qos = retcodes
	err := slf.WriteMessage(suback)
	if err != nil {
		//TODO: 记录日志
		return
	}

	for _, rm := range slf._rmsgs {
		if err := slf.WriteMessage(rm); err != nil {
			//TODO: 记录日志
		} else {
			//TODO: 记录日志
		}
	}
}

func (slf *ConBroker) onUnSubscribe(msg *message.Unsubscribe) {
	topics := msg.Payload

	for _, topic := range topics {
		sub, exist := slf._subscription[topic.TopicPath]
		if exist {
			blackboard.Instance().Topics.Unsubscribe([]byte(sub.Topic), sub)
			session := slf._session
			if session != nil {
				session.RemoveTopics(topic.TopicPath)
				delete(slf._subscription, topic.TopicPath)
			}
		}
	}

	unsuback := message.SpawnSubackMessage()
	unsuback.PacketIdentifier = msg.PacketIdentifier
	err := slf.WriteMessage(unsuback)
	if err != nil {
		//TODO: 记录日志
	}
}

func (slf *ConBroker) onPingresp(msg *message.Pingresp) {
	slf._ping++
}

func (slf *ConBroker) procPublish(msg *message.Publish) {
	if msg.Retain > 0 {
		if err := blackboard.Instance().Topics.Retain(msg); err != nil {
			//TODO: 记录错误日志
		}
	}

	var subs []interface{}
	var qoss []byte

	err := blackboard.Instance().Topics.Subscribers([]byte(msg.TopicName),
		byte(msg.QosLevel), &subs, &qoss)
	if err != nil {
		//TODO: 记录错误日志
		return
	}

	if len(subs) == 0 {
		return
	}

	for _, sub := range subs {
		s, ok := sub.(*common.Subscription)
		if ok {
			ss := blackboard.Instance().Sessions.Get(s.Client)
			if ss == nil {
				//TODO: 记录错误信息
				continue
			}

			if err := ss.WriteMessage(msg); err != nil {
				//TODO: 记录错误日志
			}
		}
	}
}

//Kicker 处理心跳
func (slf *ConBroker) Kicker() {
	if slf._keepalive > 0 {
		slf.Ping()
		slf._kicker.Reset(time.Second * time.Duration(slf._keepalive))
	}
}

//Ping 发送Ping给客户端
func (slf *ConBroker) Ping() {
	if slf._state == network.StateClosed {
		return
	}

	slf.WriteMessage(message.SpawnPingreqMessage())
}

//Will 发送遗嘱消息
func (slf *ConBroker) Will() {
	will := slf._willMsg
	msg := message.SpawnPublishMessage()
	msg.TopicName = will.Topic
	msg.Payload = []byte(will.Message)
	msg.QosLevel = int(will.Qos)
	//发送Publish消息
	slf.SendPublishMessage(msg)
}

//SendPublishMessage 发送publish消息
func (slf *ConBroker) SendPublishMessage(msg *message.Publish) {
	var subs []interface{}
	var qoss []byte

	err := blackboard.Instance().Topics.Subscribers([]byte(msg.TopicName),
		byte(msg.QosLevel), &subs, &qoss)
	if err != nil {
		//log.Error("search sub client error,  ", zap.Error(err))
		return
	}

	for _, sub := range subs {
		if sub != nil {

		}
		s, ok := sub.(*common.Subscription)
		if ok {
			//可以考虑如果没有找到在线的连接，可以通过ClientID找到目标session放入离线数据中
			ss := blackboard.Instance().Sessions.Get(s.Client)
			if ss == nil {
				//TODO: 记录日志
				continue
			}

			if err := ss.WriteMessage(msg); err != nil {
				//TODO: 写日志
			}
		}
	}
}

//Terminate 终止连接器
func (slf *ConBroker) Terminate() {
	if err := slf.Close(); err != nil {
		//TODO: 记录日志
	}
}

//Close 关闭连接器
func (slf *ConBroker) Close() error {
	if slf._willMsg != nil {
		slf.Will()
	}

	if slf._cleanSession {
		if slf._session != nil {
			blackboard.Instance().Sessions.Remove(slf._session.GetClientID())
			//TODO: 移出已订阅的主题
		} else {
			//TODO:记录异常日志
		}
	} else if slf._session != nil {
		for msg := range slf._queue {
			switch msg.GetType() {
			case encoding.PTypeConnect:
			case encoding.PTypeDisconnect:
			case encoding.PTypePingreq:
			default:
				slf._session.PushOfflineMessage(msg)
			}
		}
	}

	slf._state = network.StateClosed
	slf._closed <- true
	err := slf._conn.Close()
	slf._wg.Wait()
	subs := slf._subscription
	for _, sub := range subs {
		err := blackboard.Instance().Topics.Unsubscribe([]byte(sub.Topic), sub)
		if err != nil {
			//TODO: 记录错误日志
		}
	}

	close(slf._queue)
	if slf._session != nil {
		slf._session.WithOnDisconnect(nil)
		slf._session.WithOnWrite(nil)
		slf._session = nil
	}
	return err
}
