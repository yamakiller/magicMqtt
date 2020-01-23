package message

import (
	"time"

	codec "github.com/yamakiller/magicMqtt/encoding/message"
)

//Context message context
type Context struct {
	Message codec.Message
	Created time.Time
	Updated time.Time
	Opaque  interface{}
	_ref    int
}
