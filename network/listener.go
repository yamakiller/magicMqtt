package network

import (
	"net"
	"sync"
)

//IListener 监听接口
type IListener interface {
	Accept() (net.Conn, error)
	Addr() net.Addr
	Wait()
	Close() error
}

//MListener
type MListener struct {
	net.Listener
	_wg sync.WaitGroup
}

//Accept 接受链接
func (slf *MListener) Accept() (net.Conn, error) {
	c, err := slf.Listener.Accept()
	if err != nil {
		return nil, err
	}

	slf._wg.Add(1)
	return &MConn{Conn: c, _wg: &slf._wg}, nil
}

//Wait 等待所有客户端结束
func (slf *MListener) Wait() {
	slf._wg.Wait()
}

type MConn struct {
	net.Conn
	_wg *sync.WaitGroup
}

func (slf *MConn) Close() error {
	if slf._wg != nil {
		slf._wg.Done()
	}
	slf._wg = nil
	return slf.Conn.Close()
}
