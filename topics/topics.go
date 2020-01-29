package topics

import (
	"fmt"

	"github.com/yamakiller/magicMqtt/encoding/message"
)

const (
	// MWC is the multi-level wildcard
	MWC = "#"

	// SWC is the single level wildcard
	SWC = "+"

	// SEP is the topic level separator
	SEP = "/"

	// SYS is the starting character of the system level topics
	SYS = "$"

	// Both wildcards
	_WC = "#+"
)

var (
	providers = make(map[string]TopicsProvider)
)

//TopicsProvider 主题接口
type TopicsProvider interface {
	Subscribe(topic []byte, qos byte, subscriber interface{}) (byte, error)
	Unsubscribe(topic []byte, subscriber interface{}) error
	Subscribers(topic []byte, qos byte, subs *[]interface{}, qoss *[]byte) error
	Retain(msg *message.Publish) error
	Retained(topic []byte, msgs *[]*message.Publish) error
	Close() error
}

//Register 注册一个主题提供者
func Register(name string, provider TopicsProvider) {
	if provider == nil {
		panic("topics: Register provide is nil")
	}

	if _, dup := providers[name]; dup {
		panic("topics: Register called twice for provider " + name)
	}

	providers[name] = provider
}

//Unregister 注销
func Unregister(name string) {
	delete(providers, name)
}

//Manager Topic 管理器
type Manager struct {
	_p TopicsProvider
}

//NewManager 创建管理器
func NewManager(providerName string) (*Manager, error) {
	p, ok := providers[providerName]
	if !ok {
		return nil, fmt.Errorf("session: unknown provider %q", providerName)
	}

	return &Manager{_p: p}, nil
}

//Subscribe 订阅主题
func (slf *Manager) Subscribe(topic []byte, qos byte, subscriber interface{}) (byte, error) {
	return slf._p.Subscribe(topic, qos, subscriber)
}

//Unsubscribe 取消订阅
func (slf *Manager) Unsubscribe(topic []byte, subscriber interface{}) error {
	return slf._p.Unsubscribe(topic, subscriber)
}

//Subscribers 订阅多个主题
func (slf *Manager) Subscribers(topic []byte, qos byte, subs *[]interface{}, qoss *[]byte) error {
	return slf._p.Subscribers(topic, qos, subs, qoss)
}

//Retain ...
func (slf *Manager) Retain(msg *message.Publish) error {
	return slf._p.Retain(msg)
}

//Retained ...
func (slf *Manager) Retained(topic []byte, msgs *[]*message.Publish) error {
	return slf._p.Retained(topic, msgs)
}

//Close 关闭
func (slf *Manager) Close() error {
	return slf._p.Close()
}
