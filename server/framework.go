package server

import "sync"

//Framework ...
type Framework struct {
	_configPath string
	_execPath   string
	_workingDir string
	_sync       sync.Mutex
}
