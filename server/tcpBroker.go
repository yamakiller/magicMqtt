package server

import (
	"net"
	"strings"
	"sync"
	"time"

	"github.com/yamakiller/magicLibs/util"
	"github.com/yamakiller/magicMqtt/blackboard"
	"github.com/yamakiller/magicMqtt/network"
)

//TCPBroker mqtt tcp 服务
type TCPBroker struct {
	_shutdown chan bool
	_lst      network.IListener
	_wg       sync.WaitGroup
	_sn       *util.SnowFlake
}

//ListenAndServe 启动监听并启动服务
func (slf *TCPBroker) ListenAndServe(address string) error {

	addr, err := net.ResolveTCPAddr("tcp4", address)
	lst, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}

	slf._sn = util.NewSnowFlake(blackboard.Instance().Deploy.WorkGroupID,
		blackboard.Instance().Deploy.WorkID)

	slf._shutdown = make(chan bool)
	slf._lst = &network.MListener{Listener: lst}
	slf._wg.Add(1)
	go slf.Serve()

	return nil
}

//Serve 服务逻辑
func (slf *TCPBroker) Serve() error {
	defer func() {
		slf._wg.Done()
	}()

	var err error
	tmpDelay := time.Duration(1) * time.Millisecond
	for {
		select {
		case <-slf._shutdown:
			goto Exit
		default:
			c, e := slf._lst.Accept()
			if e != nil {
				if v, ok := c.(*net.TCPConn); ok {
					v.SetNoDelay(true)
					v.SetKeepAlive(true)
				}

				if ne, ok := e.(net.Error); ok && ne.Temporary() {
					tmpDelay *= 5
					if max := 1 * time.Second; tmpDelay > max {
						tmpDelay = max
					}

					slf.Debug("Accept error, %s", e.Error())
					time.Sleep(tmpDelay)
					continue
				}

				if strings.Contains(e.Error(), "use of closed network connection") {
					continue
				}

				err = e
				goto Exit
			}

			tmpDelay = time.Duration(1) * time.Millisecond

			conn := NewBrokerConn()
			cuid, _ := slf._sn.NextID()
			conn.WithID(cuid)
			conn.WithAddr(c.RemoteAddr().String())
			conn.WithConn(c)
			HandleConnection(conn)
		}
	}
Exit:
	slf._lst.(*network.MListener).Wait()
	return err
}

//Listener listener
func (slf *TCPBroker) Listener() network.IListener {
	return slf._lst
}

//Shutdown 关闭mqtt tcp服务
func (slf *TCPBroker) Shutdown() {
	close(slf._shutdown)
	slf._lst.Close()
	slf._wg.Wait()
}

//Info 输出等级为Info的日志
func (slf *TCPBroker) Info(fmt string, args ...interface{}) {
	blackboard.Instance().Log.Info(0, "", fmt, args...)
}

//Error 输出等级为Error的日志
func (slf *TCPBroker) Error(fmt string, args ...interface{}) {
	blackboard.Instance().Log.Error(0, "", fmt, args...)
}

//Warning 输出等级为Warning的日志
func (slf *TCPBroker) Warning(fmt string, args ...interface{}) {
	blackboard.Instance().Log.Error(0, "", fmt, args...)
}

//Debug 输出等级为Debug的日志
func (slf *TCPBroker) Debug(fmt string, args ...interface{}) {
	blackboard.Instance().Log.Debug(0, "", fmt, args...)
}
