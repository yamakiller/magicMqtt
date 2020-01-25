package network

import "github.com/yamakiller/magicMqtt/sessions"

type State int32

const (
	StateInit State = iota
	StateConnecting
	StateConnected
	StateAccepted
	StateIdle_IDLE
	StateDetached
	StateSend
	StateReceive
	StateShutdown
	StateClosed
)

type Connection interface {
	WithID(int64)
	GetID() int64
	WithAddr(string)
	GetAddr() string
	WithSession(*sessions.Session)
	GetSession() *sessions.Session

	Close() error
}
