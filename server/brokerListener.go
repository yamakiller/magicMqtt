package server

import (
	"net"
	"sync"
)

//BrokerListener mqtt broker listener
type BrokerListener struct {
	net.Listener
	_wg    sync.WaitGroup
	_close chan bool
}

//Accept 接收连接
func (slf *BrokerListener) Accept() (net.Conn, error) {
	var c net.Conn

	c, err := slf.Listener.Accept()
	if err != nil {
		return nil, err
	}

	slf._wg.Add(1)
	return &BrokerClient{Conn: c, _wg: &slf._wg}, nil
}

//BrokerClient mqtt broker 的访问连接
type BrokerClient struct {
	net.Conn
	_wg   *sync.WaitGroup
	_once sync.Once
}

//Close 关闭客户端连接
func (slf *BrokerClient) Close() error {
	err := slf.Conn.Close()
	slf._once.Do(func() {
		slf._wg.Done()
	})
	return err
}
