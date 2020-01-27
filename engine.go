package magicMqtt

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/yamakiller/magicMqtt/auth"
	"github.com/yamakiller/magicMqtt/blackboard"

	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"github.com/yamakiller/magicLibs/logger"
	"github.com/yamakiller/magicLibs/util"
	"github.com/yamakiller/magicMqtt/log"
	"github.com/yamakiller/magicMqtt/server"
	"github.com/yamakiller/magicMqtt/sessions"
	"github.com/yamakiller/magicMqtt/topics"
)

type Engine struct {
	FileConfig string
	AuthConfig string
	Model      string

	_broker      server.Broker
	_signalWatch *util.SignalWatch
}

func (slf *Engine) Start(addr string) error {
	model := strings.ToLower(slf.Model)
	logPath := ""
	logSize := 256
	logLevel := logger.DEBUGLEVEL

	if model == "release" {
		logPath = "./log"
		logLevel = logger.INFOLEVEL
	}

	logHandle := func() *log.LogBroker {
		l := log.LogBroker{}
		l.WithFilPath(logPath)
		l.WithHandle(logrus.New())
		l.WithMailMax(logSize)
		l.WithLevel(logrus.Level(logLevel))

		formatter := new(prefixed.TextFormatter)
		formatter.FullTimestamp = true
		formatter.TimestampFormat = "2006-01-02 15:04:05"
		formatter.SetColorScheme(&prefixed.ColorScheme{
			PrefixStyle:    "white+h",
			TimestampStyle: "black+h"})
		l.WithFormatter(formatter)
		l.Initial()
		l.Redirect()
		return &l
	}()

	blackboard.Instance().Log = logHandle
	blackboard.Instance().Log.Mount()
	//读取配置文件
	cnf := blackboard.Config{}
	content, err := ioutil.ReadFile(slf.FileConfig)
	if err != nil {
		return err
	}

	err = json.Unmarshal(content, &cnf)
	if err != nil {
		return err
	}

	blackboard.Instance().Deploy = cnf
	//创建验证器
	if slf.AuthConfig == "" {
		blackboard.Instance().Auth, _ = auth.New("mock", "")
	} else {
		authTmp, err := auth.New(auth.AuthDB, slf.AuthConfig)
		if err != nil {
			return err
		}
		blackboard.Instance().Auth = authTmp
	}

	blackboard.Instance().Sessions = sessions.NewGroup()
	blackboard.Instance().Topics, _ = topics.NewManager("men")
	//启动服务
	slf._broker = &server.TCPBroker{}
	if err := slf._broker.ListenAndServe(addr); err != nil {
		return err
	}

	//监听信号
	slf._signalWatch = &util.SignalWatch{}
	slf._signalWatch.Initial(slf.signalClose)
	slf._signalWatch.Watch()

	return nil
}

func (slf *Engine) signalClose() {
	if slf._broker != nil {
		slf._broker.Shutdown()
		slf._broker = nil
	}
}

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
