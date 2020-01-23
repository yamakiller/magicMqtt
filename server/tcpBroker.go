package server

import (
	"net"
	"strings"
	"sync"
	"time"

	"github.com/yamakiller/magicMqtt/dashboard"
)

//TCPBroker mqtt tcp 服务
type TCPBroker struct {
	_lst      Listener
	_gw       sync.WaitGroup
	_shutdown chan bool
}

//ListenAndServe 监听并启动服务
func (slf *TCPBroker) ListenAndServe() error {
	addr, err := net.ResolveTCPAddr("tcp4", dashboard.Instance().BrokerAddr)
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}
	slf._lst = &BrokerListener{Listener: l}
	dashboard.Instance().LDebug("Started tcp broker => %s", slf._lst.Addr().String())
	slf._gw.Add(1)
	go slf.Serve(slf._lst)

	return nil
}

//Serve Accept服务
func (slf *TCPBroker) Serve(l Listener) error {
	defer func() {
		l.Close()
		slf._gw.Done()
	}()

	tmpDelay := 1 * time.Microsecond
Accept:
	for {
		select {
		case <-slf._shutdown:
			break Accept
		default:
			c, err := l.Accept()
			if err != nil {
				if v, ok := c.(*net.TCPConn); ok {
					v.SetNoDelay(true)
					v.SetKeepAlive(true)
				}

				if ne, ok := err.(net.Error); ok && ne.Temporary() {
					tmpDelay *= 5
					if max := 1 * time.Second; tmpDelay > max {
						tmpDelay = max
					}

					dashboard.Instance().LWarning("Accept error: %v; retrying in %v", err, tmpDelay)
					time.Sleep(tmpDelay)
					continue
				}

				if strings.Contains(err.Error(), "use of closed network connection") {
					dashboard.Instance().LError("Accept Failed: %s", err.Error())
					continue
				}

				dashboard.Instance().LError("Accept Error: %s", err.Error())
				return err
			}

			tmpDelay = 1 * time.Microsecond
			conn := newConnection()
			conn.WithHandle(c)
			conn.WithID(dashboard.Instance().NextID())
			conn.WithAddr(c.RemoteAddr().String())
			//go slf.HandleConnection(conn)
			//conn.Serve()
		}
	}

	slf._lst.(*BrokerListener)._wg.Wait()
	//TODO: 等待所有session关闭
	return nil
}

//Listener 返回监听器
func (slf *TCPBroker) Listener() net.Listener {
	dashboard.Instance().LDebug("LIS: %#v", slf._lst)
	return slf._lst
}

//Shutdown 关闭服务
func (slf *TCPBroker) Shutdown() {
	close(slf._shutdown)
	slf._lst.Close()
	slf._gw.Wait()
}

/*
//TCPBroker mqtt broker tcp protocol
type TCPBroker struct {
	Address string
	Engine  *MagicMQTT

	_shutdown chan bool
	_listener Listener
	_wg       *sync.WaitGroup
}

//ListenAndServe mqtt broker listen and start server
func (slf *TCPBroker) ListenAndServe() error {
	addr, err := net.ResolveTCPAddr("tcp4", slf.Address)
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}

	slf._listener = &BrokerListener{Listener: listener}
	slf.Engine.LDebug("Started tcp broker %s", slf._listener.Addr().String())
	go slf.Serve(slf._listener)

	return nil
}

//Serve mqtt broker serve
func (slf *TCPBroker) Serve(l net.Listener) error {
	defer func() {
		l.Close()
	}()

	var tempDelay time.Duration
Accept:
	for {
		select {
		case <-slf._shutdown:
			break Accept
		default:
			c, err := l.Accept()
			if err != nil {
				if v, ok := c.(*net.TCPConn); ok {
					v.SetNoDelay(true)
					v.SetKeepAlive(true)
				}

				if ne, ok := err.(net.Error); ok && ne.Temporary() {
					if tempDelay == 0 {
						tempDelay = 1 * time.Microsecond
					}

					tempDelay *= 5
					if max := 1 * time.Second; tempDelay > max {
						tempDelay = max
					}

					slf.Engine.LWarning("Accept error: %v; retrying in %v", err, tempDelay)
					time.Sleep(tempDelay)
					continue
				}

				if strings.Contains(err.Error(), "use of closed network connection") {
					slf.Engine.LError("Accept Failed: %s", err.Error())
					continue
				}

				slf.Engine.LError("Accept Error: %s", err.Error())
				return err
			}
			tempDelay = 0
			client := spawnBrokerClient(slf.Engine._config)
			client.WithConn(c)
			client.WithID(slf.Engine.NextID())
			client.WithAddr(c.RemoteAddr().String())
			slf.Engine.LDebug("Accepted client remote %s => %d", client.GetAddr(), client.GetID())
			go slf.Engine.HandleClient(client)
		}
	}

	slf._listener.(*BrokerListener)._wg.Wait()
	slf._wg.Done()

	return nil
}

//Listener Returns tcp broker listener
func (slf *TCPBroker) Listener() Listener {
	slf.Engine.LDebug("LIS: %#v", slf._listener)
	return slf._listener
}

//Shutdown mqtt tcp broker shutdown
func (slf *TCPBroker) Shutdown() {
	close(slf._shutdown)
	slf._listener.Close()
}
*/
