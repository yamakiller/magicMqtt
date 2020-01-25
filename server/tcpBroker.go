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

					time.Sleep(tmpDelay)
					continue
				}

				if strings.Contains(err.Error(), "use of closed network connection") {
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

func (slf *TCPBroker) Shutdown() {
	close(slf._shutdown)
	slf._lst.Close()
	slf._wg.Wait()
}
