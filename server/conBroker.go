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
	_closed       chan bool
	_connected    bool
	_cleanSession bool
	_ping         int
	_activity     time.Time
	_state        network.State
	_once         sync.Once
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
		slf.Error("Undefined message/%d", msg.GetType())
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
			slf.Error("flush error, %s", err.Error())
		}
	}
}

func (slf *ConBroker) onConnect(msg *message.Connect) {
	if slf._connected {
		slf.Debug("Client is connected")
		return
	}

	name := string(msg.UserName)
	pwd := string(msg.Password)
	connack := message.SpawnConnackMessage()
	connack.ReturnCode = 0

	defer func() {
		if err := slf.WriteMessage(connack); err != nil {
			slf.Error("Response/connack error, %s", err.Error())
		}
	}()

	if _, err := blackboard.Instance().Auth.Connect(msg.Identifier, name, pwd); err != nil {

		if err == code.ErrAuthClientNot {
			connack.ReturnCode = 0x05
		} else {
			connack.ReturnCode = 0x04
		}
		slf.Debug("Auth/%s/%s/%s connect fail, %s", msg.Identifier, name, pwd, err.Error())
		return
	}

	if msg.CleanSession {
		session := blackboard.Instance().Sessions.Get(msg.Identifier)
		if session != nil {
			session.DoDisconnect()
		}
		session = blackboard.Instance().Sessions.New(msg.Identifier, blackboard.Instance().Deploy.OfflineQueueSize)
		slf._session = session
	} else {
		session, org := blackboard.Instance().Sessions.GetOrNew(msg.Identifier, blackboard.Instance().Deploy.OfflineQueueSize)
		if org {
			session.DoDisconnect()
		}
		slf._session = session
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
			slf.Error("Response/puback error, %s", err.Error())
			return
		}
		slf.procPublish(msg)
	case topics.QosExactlyOnce:
		pubrec := message.SpawnPubrecMessage()
		pubrec.PacketIdentifier = msg.PacketIdentifier
		if err := slf.WriteMessage(pubrec); err != nil {
			slf.Error("Response/pubrec error, %s", err.Error())
			return
		}
		slf.procPublish(msg)
	default:
		slf.Error("publish message qos level error: %d", msg.QosLevel)
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
	var remsg []*message.Publish
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
			slf.Error("Sub %s error, %s", topic.TopicPath, err.Error())
			retcodes = append(retcodes, topics.QosFailure)
			continue
		}

		slf._subscription[topic.TopicPath] = sub
		slf._session.AddTopics(topic.TopicPath, topic.RequestedQos)
		retcodes = append(retcodes, rqos)
		blackboard.Instance().Topics.Retained([]byte(topic.TopicPath), &remsg)
	}
	suback.Qos = retcodes
	err := slf.WriteMessage(suback)
	if err != nil {
		slf.Error("Response sub ack error, %s", err.Error())
		return
	}

	for _, rm := range remsg {
		if err := slf.WriteMessage(rm); err != nil {
			slf.Error("Response/Retained %s error, %s", rm.TopicName, err.Error())
		} else {
			slf.Debug("Response/Retained %s success", rm.TopicName)
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
	if err := slf.WriteMessage(unsuback); err != nil {
		slf.Error("Response unsub ack error, %s", err.Error())
	}
}

func (slf *ConBroker) onPingresp(msg *message.Pingresp) {
	slf._ping++
}

func (slf *ConBroker) procPublish(msg *message.Publish) {
	if msg.Retain > 0 {
		if err := blackboard.Instance().Topics.Retain(msg); err != nil {
			slf.Error("Sub topic error, %s", err.Error())
		}
	}

	var subs []interface{}
	var qoss []byte

	err := blackboard.Instance().Topics.Subscribers([]byte(msg.TopicName),
		byte(msg.QosLevel), &subs, &qoss)
	if err != nil {
		slf.Error("Sub topic/%s error, %s", msg.TopicName, err.Error())
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
				slf.Debug("No client/%s associated sessions were found", s.Client)
				continue
			}

			if err := ss.WriteMessage(msg); err != nil {
				slf.Error("Distribution/%+v to client/%s error, %s", msg, s.Client, err.Error())
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
		slf.Error("Search sub client error, %s", err.Error())
		return
	}

	for _, sub := range subs {
		if sub != nil {

		}
		s, ok := sub.(*common.Subscription)
		if ok {
			ss := blackboard.Instance().Sessions.Get(s.Client)
			if ss == nil {
				slf.Debug("No client/%s associated sessions were found", s.Client)
				continue
			}

			if err := ss.WriteMessage(msg); err != nil {
				slf.Error("Distribution/%+v to client/%s error, %s", msg, s.Client, err.Error())
			}
		}
	}
}

//Terminate 终止连接器
func (slf *ConBroker) Terminate() {
	if err := slf.Close(); err != nil {
		slf.Error("Terminate error:%s", err.Error())
	}
}

//Close 关闭连接器
func (slf *ConBroker) Close() error {
	var err error
	slf._once.Do(func() {
		slf.Debug("closed connection")
		if slf._willMsg != nil {
			slf.Will()
		}

		if slf._cleanSession {
			if slf._session != nil {
				blackboard.Instance().Sessions.Remove(slf._session.GetClientID())
			} else {
				slf.Warning("Closing [clean session:true] client unconnect")
			}
		} else if slf._session != nil {
			for msg := range slf._queue {
				switch msg.GetType() {
				case encoding.PTypeConnect:
				case encoding.PTypeDisconnect:
				case encoding.PTypePingreq:
				case encoding.PTypePingresp:
				default:
					slf._session.PushOfflineMessage(msg)
				}
			}
		}

		slf._state = network.StateClosed
		slf._closed <- true
		err = slf._conn.Close()
		slf._wg.Wait()
		//开始移除订阅的主题
		subs := slf._subscription
		for _, sub := range subs {
			err = blackboard.Instance().Topics.Unsubscribe([]byte(sub.Topic), sub)
			if err != nil {
				slf.Error("Closing unsubscribe error:%s", err.Error())
			}
		}

		close(slf._queue)
		if slf._session != nil {
			slf._session.WithOnDisconnect(nil)
			slf._session.WithOnWrite(nil)
			slf._session = nil
		}
		slf.Debug("closed connected complate")
	})
	return err
}

//Info 输出等级为Info的日志
func (slf *ConBroker) Info(fmt string, args ...interface{}) {
	id, client := slf.getPrefix()
	blackboard.Instance().Log.Info(id, client, fmt, args...)
}

//Error 输出等级为Error的日志
func (slf *ConBroker) Error(fmt string, args ...interface{}) {
	id, client := slf.getPrefix()
	blackboard.Instance().Log.Error(id, client, fmt, args...)
}

//Warning 输出等级为Warning的日志
func (slf *ConBroker) Warning(fmt string, args ...interface{}) {
	id, client := slf.getPrefix()
	blackboard.Instance().Log.Error(id, client, fmt, args...)
}

//Debug 输出等级为Debug的日志
func (slf *ConBroker) Debug(fmt string, args ...interface{}) {
	id, client := slf.getPrefix()
	blackboard.Instance().Log.Debug(id, client, fmt, args...)
}

func (slf *ConBroker) getPrefix() (int64, string) {
	session := slf._session
	if session == nil {
		return slf._id, ""
	}
	return slf._id, session.GetClientID()
}
