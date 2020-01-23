package server

import "net"

//Broker server instance
type Broker interface {
	ListenAndServe() error
	Listener() net.Listener
	Serve(l net.Listener) error
	Shutdown()
}
