package server

import (
	"github.com/yamakiller/magicMqtt/network"
)

//Broker mqtt服务主框架
type Broker interface {
	ListenAndServe(string) error
	Listener() network.IListener
	Serve()
	Shutdown()
}
