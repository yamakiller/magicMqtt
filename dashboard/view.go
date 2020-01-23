package dashboard

import (
	"sync"

	"github.com/yamakiller/magicMqtt/sessions"

	"github.com/yamakiller/magicLibs/logger"
	"github.com/yamakiller/magicLibs/util"
)

var (
	defaultView *View
	viewOnce    sync.Once
)

//Instance 黑板视图访问接口
func Instance() *View {
	viewOnce.Do(func() {
		defaultView = &View{
			_sessionStore: sessions.New(),
		}
	})
	return defaultView
}

//View 黑板数据视图
type View struct {
	BrokerAddr        string
	BrokerBufferSize  int
	BrokerMessageLimt int
	BrokerQueueSize   int
	BrokerKeepalive   int
	_snk              *util.SnowFlake
	_log              logger.Logger
	_sessionStore     *sessions.SessionGroups
}

//WithLogger 设置日志上下文
func (slf *View) WithLogger(log logger.Logger) {
	slf._log = log
}

//WithWorkSpace 设置ID生成空间
func (slf *View) WithWorkSpace(workID, localID int64) {
	slf._snk = util.NewSnowFlake(workID, localID)
}

//NextID 生成一个64位唯一ID
func (slf *View) NextID() int64 {
	id, _ := slf._snk.NextID()
	return id
}

//Auth 授权客户端
func (slf *View) Auth(clientIdentifier, username, password string) error {
	return nil
}

//Sessions 返回Session管理组对象
func (slf *View) Sessions() *sessions.SessionGroups {
	return slf._sessionStore
}

//LInfo 输出消息等级日志
func (slf *View) LInfo(sfmt string, args ...interface{}) {
	slf._log.Info(0, sfmt, args...)
}

//LWarning 输出警告等级日志
func (slf *View) LWarning(sfmt string, args ...interface{}) {
	slf._log.Warning(0, sfmt, args...)
}

//LDebug 输出调试等级日志
func (slf *View) LDebug(sfmt string, args ...interface{}) {
	slf._log.Debug(0, sfmt, args...)
}

//LError 输出错误等级日志
func (slf *View) LError(sfmt string, args ...interface{}) {
	slf._log.Error(0, sfmt, args...)
}

//LTrace 输出跟踪等级日志
func (slf *View) LTrace(sfmt string, args ...interface{}) {
	slf._log.Trace(0, sfmt, args...)
}
