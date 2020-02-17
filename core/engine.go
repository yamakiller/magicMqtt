package core

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/yamakiller/magicMqtt/blackboard"

	"github.com/yamakiller/magicLibs/log"
	"github.com/yamakiller/magicLibs/util"
	"github.com/yamakiller/magicMqtt/auth"
	"github.com/yamakiller/magicMqtt/server"
	"github.com/yamakiller/magicMqtt/sessions"
	"github.com/yamakiller/magicMqtt/topics"
)

//Engine 系统引擎
type Engine struct {
	FileConfig string
	AuthMode   string
	Model      string

	_closed      chan bool
	_broker      server.Broker
	_signalWatch *util.SignalWatch
}

//Start 启动系统
func (slf *Engine) Start(addr string) error {
	model := strings.ToLower(slf.Model)
	logPath := ""
	logName := ""
	logLevel := logrus.DebugLevel
	if model == "release" {
		logPath = "./log"
		logName = "mqtt"
		logLevel = logrus.InfoLevel
	}

	hlog, err := log.SpawnFileLogrus(logLevel,
		logPath,
		logName)

	if err != nil {
		return err
	}

	blackboard.Instance().Log = &log.DefaultAgent{}
	blackboard.Instance().Log.WithHandle(hlog)

	//读取配置文件
	cfg := blackboard.Config{}
	content, err := ioutil.ReadFile(slf.FileConfig)
	if err != nil {
		return err
	}

	err = json.Unmarshal(content, &cfg)
	if err != nil {
		return err
	}

	blackboard.Instance().Deploy = cfg
	//创建验证器
	slf.AuthMode = strings.ToLower(slf.AuthMode)
	switch slf.AuthMode {
	case "authdb":
		if cfg.AuthDB == "" {
			return errors.New("Please configure AuthDB configuration information")
		}
		au, err := auth.New(auth.AuthDB, cfg.AuthDB)
		if err != nil {
			return err
		}
		blackboard.Instance().Auth = au
	case "authfile":
	default:
		au, _ := auth.New("mock", "")
		blackboard.Instance().Auth = au
	}

	blackboard.Instance().Sessions = sessions.NewGroup()
	blackboard.Instance().Topics, _ = topics.NewManager("mem")
	//启动服务
	slf._broker = &server.TCPBroker{}
	if err := slf._broker.ListenAndServe(addr); err != nil {
		return err
	}

	//监听信号
	slf._closed = make(chan bool)
	slf._signalWatch = &util.SignalWatch{}
	slf._signalWatch.Initial(slf.signalClose)
	slf._signalWatch.Watch()

	return nil
}

func (slf *Engine) signalClose() {
	close(slf._closed)
}

//Info 输出消息级日志
func (slf *Engine) Info(fmt string, args ...interface{}) {
	blackboard.Instance().Log.Info(slf.getPrefix(), fmt, args...)
}

//Warning 输出警告级日志
func (slf *Engine) Warning(fmt string, args ...interface{}) {
	blackboard.Instance().Log.Warning(slf.getPrefix(), fmt, args...)
}

//Error 输出错误级日志
func (slf *Engine) Error(fmt string, args ...interface{}) {
	blackboard.Instance().Log.Error(slf.getPrefix(), fmt, args...)
}

//Debug 输出调试级日志
func (slf *Engine) Debug(fmt string, args ...interface{}) {
	blackboard.Instance().Log.Debug(slf.getPrefix(), fmt, args...)
}

//Panic 输出崩溃信息
func (slf *Engine) Panic(fmt string, args ...interface{}) {
	blackboard.Instance().Log.Panic(slf.getPrefix(), fmt, args...)
}

func (slf *Engine) getPrefix() string {
	return "mqtt@system"
}

//Wait 等待系统结束
func (slf *Engine) Wait() {
	if slf._closed == nil {
		goto Exit
	}

	for {
		select {
		case <-slf._closed:
			goto Exit
		}
	}
Exit:
}

//Shutdown 关闭系统
func (slf *Engine) Shutdown() {
	if slf._broker != nil {
		slf._broker.Shutdown()
		slf._broker = nil
	}

	if slf._signalWatch != nil {
		slf._signalWatch.Wait()
		slf._signalWatch = nil
	}

	if blackboard.Instance().Auth != nil {
		blackboard.Instance().Auth = nil
	}

	if blackboard.Instance().Log != nil {
		blackboard.Instance().Log.Close()
		blackboard.Instance().Log = nil
	}
}
