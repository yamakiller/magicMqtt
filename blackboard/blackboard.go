package blackboard

import (
	"sync"

	"github.com/yamakiller/magicLibs/log"
	"github.com/yamakiller/magicMqtt/auth"
	"github.com/yamakiller/magicMqtt/sessions"
	"github.com/yamakiller/magicMqtt/topics"
)

var (
	defaultBoard *Board
	onceBoard    sync.Once
)

//Instance 黑板接口
func Instance() *Board {
	onceBoard.Do(func() {
		defaultBoard = &Board{}
	})

	return defaultBoard
}

//Board 黑板数据
type Board struct {
	Deploy   Config
	Auth     auth.Auth
	Log      log.LogAgent
	Sessions *sessions.SessionGroup
	Topics   *topics.Manager
}
