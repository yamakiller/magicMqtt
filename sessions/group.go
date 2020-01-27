package sessions

import (
	"sync"
)

func NewGroup() *SessionGroup {
	return &SessionGroup{_ss: make(map[string]*Session)}
}

//SessionGroup session 管理器
type SessionGroup struct {
	_ss map[string]*Session
	_sy sync.RWMutex
}

//Remove 删除指定的session
func (slf *SessionGroup) Remove(clientID string) {
	slf._sy.Lock()
	defer slf._sy.Unlock()
	if _, ok := slf._ss[clientID]; ok {
		delete(slf._ss, clientID)
	}
}

//Get 返回一个Session
func (slf *SessionGroup) Get(clientID string) *Session {
	slf._sy.RLock()
	defer slf._sy.RUnlock()
	if s, ok := slf._ss[clientID]; ok {
		return s
	}
	return nil
}

//New 创建一个新的Session
func (slf *SessionGroup) New(clientID string, offlineLimit int) *Session {
	slf._sy.Lock()
	defer slf._sy.Unlock()
	s := newSession(offlineLimit)
	slf._ss[clientID] = s
	return s
}

//GetOrNew 返回一个或创建一个Session
func (slf *SessionGroup) GetOrNew(clientID string, offlineLimit int) (*Session, bool) {
	slf._sy.RLock()
	if s, ok := slf._ss[clientID]; ok {
		slf._sy.RUnlock()
		return s, true
	}
	slf._sy.RUnlock()

	slf._sy.Lock()
	defer slf._sy.Unlock()

	s := newSession(offlineLimit)
	slf._ss[clientID] = s

	return s, false
}
