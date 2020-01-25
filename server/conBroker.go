package server

import (
	"bufio"
	"io"
	"net"
	"sync"
	"time"

	"github.com/yamakiller/magicMqtt/blackboard"
	"github.com/yamakiller/magicMqtt/sessions"

	"github.com/yamakiller/magicMqtt/encoding"
	codec "github.com/yamakiller/magicMqtt/encoding/message"
	"github.com/yamakiller/magicMqtt/network"
)

func NewBrokerConn() *ConBroker {
	c := &ConBroker{
		_keepalive: blackboard.Instance().Deploy.Keepalive,
		_queue:     make(chan codec.Message, blackboard.Instance().Deploy.MessageQueueSize),
		_closed:    make(chan bool),
		_activity:  time.Now(),
		_state:     network.StateInit,
	}

	if c._keepalive > 0 {
		c._kicker = time.NewTimer(time.Duration(c._keepalive) * time.Second)
	}
	return c
}

type ConBroker struct {
	_conn      io.ReadWriteCloser
	_queue     chan codec.Message
	_kicker    *time.Timer
	_keepalive int
	_id        int64
	_addr      string
	_reader    *bufio.Reader
	_writer    *bufio.Writer
	_session   *sessions.Session
	_closed    chan bool
	_activity  time.Time
	_state     network.State
	_wg        sync.WaitGroup
}

func (slf *ConBroker) WithID(id int64) {
	slf._id = id
}

func (slf *ConBroker) GetID() int64 {
	return slf._id
}

func (slf *ConBroker) WithAddr(remote string) {
	slf._addr = remote
}

func (slf *ConBroker) GetAddr() string {
	return slf._addr
}

func (slf *ConBroker) WithSession(session *sessions.Session) {
	slf._session = session
}

func (slf *ConBroker) GetSession() *sessions.Session {
	return slf._session
}

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

func (slf *ConBroker) ParseMessage() (codec.Message, error) {
	if slf._keepalive > 0 {
		if cn, ok := slf._conn.(net.Conn); ok {
			cn.SetReadDeadline(slf._activity.Add(time.Duration(int(float64(slf._keepalive)*1.5)) * time.Second))
		}
	}

	message, err := codec.Parse(slf._reader, blackboard.Instance().Deploy.MessageSize)
	if err != nil {
		return nil, err
	}

	switch message.GetType() {
	case encoding.PTypeConnect:
		slf.onConnect(message.(*codec.Connect))
		break
	case encoding.PTypePublish:
		slf.onPublish(message.(*codec.Publish))
		break
	default:
		//报错误
	}

	slf.invalidateTimer()
	slf._activity = time.Now()
	return message, err
}

func (slf *ConBroker) onConnect(msg *codec.Connect) {

}

func (slf *ConBroker) onPublish(msg *codec.Publish) {

}

func (slf *ConBroker) Kicker() {
	if slf._keepalive > 0 {
		slf.Ping()
		slf._kicker.Reset(time.Second * time.Duration(slf._keepalive))
	}
}

func (slf *ConBroker) Ping() {
	if slf._state == network.StateClosed {
		return
	}
	//发送Ping消息x
}

func (slf *ConBroker) Terminate() {

}

func (slf *ConBroker) Close() error {
	slf._state = network.StateClosed
	slf._closed <- true

	return slf._conn.Close()
}
