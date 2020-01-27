package auth

import "github.com/yamakiller/magicMqtt/auth/authdb"

const (
	//AuthDB mysql 验证器
	AuthDB = "authdb"
	//AuthFile files 验证器
	AuthFile = "authfile"
)

//Auth 授权验证器-接口
type Auth interface {
	ACL(action, clientID, username, ip, topic string) (bool, error)
	Connect(clientID, username, password string) (bool, error)
}

//New 创建授权验证器
func New(name string, conf string) (Auth, error) {
	switch name {
	case AuthDB:
		return authdb.Init(conf)
	default:
		return &Mock{}, nil
	}
}
