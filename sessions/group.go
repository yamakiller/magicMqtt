package sessions

import (
	"sync"

	codec "github.com/yamakiller/magicMqtt/encoding/message"
	"github.com/yamakiller/magicMqtt/message"
)

//New 创建一个Session组
func New() *SessionGroups {
	return &SessionGroups{_s: make(map[string]*Session)}
}

//SessionGroups Session管理组
type SessionGroups struct {
	_s map[string]*Session
	_m sync.RWMutex
}

//New 创建一个新的Session
func (slf *SessionGroups) New(clientIdentifier string) *Session {
	slf._m.Lock()
	defer slf._m.Unlock()
	s := slf.allocSession()

	slf._s[clientIdentifier] = s

	return s
}

//GetOrNew 如果Session已存在就返回,不存在就新建
func (slf *SessionGroups) GetOrNew(clientIdentifier string) (*Session, bool) {
	slf._m.RLock()
	if v, ok := slf._s[clientIdentifier]; ok {
		slf._m.RUnlock()
		return v, true
	}
	slf._m.RUnlock()

	slf._m.Lock()
	defer slf._m.Unlock()
	v := slf.allocSession()
	slf._s[clientIdentifier] = v
	return v, false
}

func (slf *SessionGroups) allocSession() *Session {
	return &Session{
		_offlineQueue:  make([]codec.Message, 0),
		_inflightTable: message.NewTable(),
	}
}
