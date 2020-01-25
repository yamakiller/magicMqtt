package server

import (
	"github.com/yamakiller/magicMqtt/network"
)

type Broker interface {
	ListenAndServe(string) error
	Listener() network.IListener
	Serve()
	Shutdown()
}
