package log

import (
	"fmt"
	"os"
	"path"
	"sync"
	"sync/atomic"
	"time"

	rotatelogs "github.com/lestrrat/go-file-rotatelogs"
	"github.com/pkg/errors"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"github.com/yamakiller/magicLibs/logger"
)

//LogBroker 日志系统
type LogBroker struct {
	_filPath    string
	_logLevel   logrus.Level
	_logHandle  *logrus.Logger
	_logMailNum int32
	_logMailMax int32
	_logMailbox chan logger.LogMessage
	_logStop    chan struct{}
	_logWait    sync.WaitGroup
}

//WithFilPath doc
//@Summary Setting log file name
//@Param (string) file name
func (slf *LogBroker) WithFilPath(v string) {
	slf._filPath = v
}

//WithLevel doc
//@Summary Setting log level limit
//@Param (logrus.Level) log level
func (slf *LogBroker) WithLevel(v logrus.Level) {
	slf._logLevel = v
}

//WithHandle doc
//@Summary Setting log object
//@Param (*logrus.Logger)
func (slf *LogBroker) WithHandle(v *logrus.Logger) {
	slf._logHandle = v
}

//WithMailMax doc
//@Summary Setting log mail max
//@Param (int)
func (slf *LogBroker) WithMailMax(v int) {
	slf._logMailMax = int32(v)
}

//WithFormatter doc
//@Summary Setting log format
//@Param (logrus.Formatter)
func (slf *LogBroker) WithFormatter(f logrus.Formatter) {
	slf._logHandle.SetFormatter(f)
}

//Initial doc
//@Summary initail logger
func (slf *LogBroker) Initial() {
	slf._logMailbox = make(chan logger.LogMessage, slf._logMailMax)
	slf._logStop = make(chan struct{})
	if slf._logHandle != nil {
		slf._logHandle.SetLevel(slf._logLevel)
	}
}

func (slf *LogBroker) run() int {
	select {
	case <-slf._logStop:
		return -1
	case msg := <-slf._logMailbox:
		slf.write(&msg)
		atomic.AddInt32(&slf._logMailNum, -1)
		return 0
	}
}

func (slf *LogBroker) exit() {
	slf._logWait.Done()
}

func (slf *LogBroker) write(msg *logger.LogMessage) {
	switch msg.Level {
	case uint32(logrus.ErrorLevel):
		slf._logHandle.WithFields(logrus.Fields{"prefix": msg.Prefix}).Errorln(msg.Message)
	case uint32(logrus.InfoLevel):
		slf._logHandle.WithFields(logrus.Fields{"prefix": msg.Prefix}).Infoln(msg.Message)
	case uint32(logrus.TraceLevel):
		slf._logHandle.WithFields(logrus.Fields{"prefix": msg.Prefix}).Traceln(msg.Message)
	case uint32(logrus.DebugLevel):
		slf._logHandle.WithFields(logrus.Fields{"prefix": msg.Prefix}).Debugln(msg.Message)
	case uint32(logrus.WarnLevel):
		slf._logHandle.WithFields(logrus.Fields{"prefix": msg.Prefix}).Warningln(msg.Message)
	case uint32(logrus.FatalLevel):
		slf._logHandle.WithFields(logrus.Fields{"prefix": msg.Prefix}).Fatalln(msg.Message)
	case uint32(logrus.PanicLevel):
		slf._logHandle.WithFields(logrus.Fields{"prefix": msg.Prefix}).Panicln(msg.Message)
	}
}

func (slf *LogBroker) getPrefix(owner int64, clientID string) string {
	if owner == 0 {
		return ""
	}

	return fmt.Sprintf("[.%016x/%s]", owner, clientID)
}

func (slf *LogBroker) push(data logger.LogMessage) {
	select {
	case slf._logMailbox <- data:
	}

	atomic.AddInt32(&slf._logMailNum, 1)
}

//Redirect doc
//@Summary Redirect log file
func (slf *LogBroker) Redirect() {
	slf._logHandle.SetOutput(os.Stdout)
	if slf._filPath == "" {
		return
	}
	baseLogPath := path.Join(slf._filPath, "log")
	writer, err := rotatelogs.New(
		baseLogPath+".%Y-%m-%d",
		rotatelogs.WithLinkName(baseLogPath),
		rotatelogs.WithMaxAge(7*24*time.Hour),
		rotatelogs.WithRotationTime(24*time.Hour),
	)

	if err != nil {
		slf.Error(0, "", "config local file system logger error. %+v", errors.WithStack(err))
	}

	formatter := new(prefixed.TextFormatter)
	formatter.FullTimestamp = true
	formatter.TimestampFormat = "2006-01-02 15:04:05"
	formatter.DisableColors = true

	lfHook := lfshook.NewHook(lfshook.WriterMap{
		logrus.DebugLevel: writer,
		logrus.InfoLevel:  writer,
		logrus.WarnLevel:  writer,
		logrus.ErrorLevel: writer,
		logrus.FatalLevel: writer,
		logrus.PanicLevel: writer,
	}, formatter)

	slf._logHandle.AddHook(lfHook)
}

//Mount doc
//@Summary Mount log module
func (slf *LogBroker) Mount() {
	slf._logWait.Add(1)
	go func() {
		for {
			if slf.run() != 0 {
				break
			}
		}
		slf.exit()
	}()
}

//Close doc
//@Summary Turn off the logging system
func (slf *LogBroker) Close() {
	for {
		if atomic.LoadInt32(&slf._logMailNum) > 0 {
			time.Sleep(time.Millisecond * 10)
			continue
		}
		break
	}

	close(slf._logStop)
	slf._logWait.Wait()
	close(slf._logMailbox)
}

//Error doc
//@Summary Output error log
//@Param (int32) owner
//@Param (string) format
//@Param (...interface{}) args
func (slf *LogBroker) Error(owner int64, client string, fmrt string, args ...interface{}) {
	slf.push(spawnMessage(uint32(logrus.ErrorLevel), slf.getPrefix(owner, client), fmt.Sprintf(fmrt, args...)))
}

//Info doc
//@Summary Output information log
//@Param (int32) owner
//@Param (string) format
//@Param (...interface{}) args
func (slf *LogBroker) Info(owner int64, client string, fmrt string, args ...interface{}) {
	slf.push(spawnMessage(uint32(logrus.InfoLevel), slf.getPrefix(owner, client), fmt.Sprintf(fmrt, args...)))
}

//Warning doc
//@Summary Output warning log
//@Param (int32) owner
//@Param (string) format
//@Param (...interface{}) args
func (slf *LogBroker) Warning(owner int64, client string, fmrt string, args ...interface{}) {
	slf.push(spawnMessage(uint32(logrus.WarnLevel), slf.getPrefix(owner, client), fmt.Sprintf(fmrt, args...)))
}

//Panic doc
//@Summary Output program crash log
//@Param (int32) owner
//@Param (string) format
//@Param (...interface{}) args
func (slf *LogBroker) Panic(owner int64, client string, fmrt string, args ...interface{}) {
	slf.push(spawnMessage(uint32(logrus.PanicLevel), slf.getPrefix(owner, client), fmt.Sprintf(fmrt, args...)))
}

//Fatal doc
//@Summary Output critical error log
//@Param (int32) owner
//@Param (string) format
//@Param (...interface{}) args
func (slf *LogBroker) Fatal(owner int64, client string, fmrt string, args ...interface{}) {
	slf.push(spawnMessage(uint32(logrus.FatalLevel), slf.getPrefix(owner, client), fmt.Sprintf(fmrt, args...)))
}

//Debug doc
//@Summary Output Debug log
//@Param (int32) owner
//@Param (string) format
//@Param (...interface{}) args
func (slf *LogBroker) Debug(owner int64, client string, fmrt string, args ...interface{}) {
	slf.push(spawnMessage(uint32(logrus.DebugLevel), slf.getPrefix(owner, client), fmt.Sprintf(fmrt, args...)))
}

//Trace doc
//@Summary Output trace log
//@Param (int32) owner
//@Param (string) format
//@Param (...interface{}) args
func (slf *LogBroker) Trace(owner int64, client string, fmrt string, args ...interface{}) {
	slf.push(spawnMessage(uint32(logrus.TraceLevel), slf.getPrefix(owner, client), fmt.Sprintf(fmrt, args...)))
}

func spawnMessage(level uint32, prefix string, message string) logger.LogMessage {
	return logger.LogMessage{Level: level, Prefix: prefix, Message: message}
}
