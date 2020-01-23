package message

import (
	"io"

	"github.com/yamakiller/magicMqtt/encoding"
)

//Message Mqtts message interface
type Message interface {
	GetType() encoding.PType
	GetTypeAsString() string
	WriteTo(w io.Writer) (int64, error)
}
