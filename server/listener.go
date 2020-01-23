package server

import "net"

//Listener network listener interface
type Listener interface {
	Accept() (c net.Conn, err error)

	Close() error

	Addr() net.Addr
}
