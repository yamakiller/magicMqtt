package blackboard

import (
	"sync"
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
	Deploy Config
}
