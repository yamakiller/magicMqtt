package authdb

import (
	"encoding/json"
	"io/ioutil"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/yamakiller/magicLibs/dbs"
	"github.com/yamakiller/magicMqtt/auth/code"
)

//Init 创建一个MYSQL授权，验证器
func Init() (*AuthMYSQL, error) {
	content, err := ioutil.ReadFile("./auth/authdb/mysql.json")
	if err != nil {
		return nil, err
	}

	var config dbs.MySQLGormDeploy
	err = json.Unmarshal(content, &config)
	if err != nil {
		return nil, err
	}
	client := &dbs.MySQLGORM{}
	err = client.Initial(config.DSN, config.Max, config.Idle, config.Life)
	if err != nil {
		return nil, err
	}

	return &AuthMYSQL{_sql: client, _ca: cache.New(5*time.Minute, 10*time.Minute)}, nil
}

//AuthMYSQL mysql 授权验证器
type AuthMYSQL struct {
	_sql *dbs.MySQLGORM
	_ca  *cache.Cache
}

//Connect 验证连接请求
func (slf *AuthMYSQL) Connect(clientID, username, password string) (bool, error) {
	action := "connect"
	{
		aCache := slf.doAuthCache(action, clientID, username, password, "")
		if aCache != nil {
			if aCache._password == password &&
				aCache._username == username &&
				aCache._action == action {
				return true, nil
			}
		}
	}

	usr := AuthUser{}
	if err := slf._sql.DB().Where("client_id = ?", clientID).First(&usr).Error; err != nil {
		return false, code.ErrAuthClientNot
	}

	if usr.UserName != username || usr.Password != password {
		return false, code.ErrAuthClientUserNameOrPwd
	}

	slf.addAuthCache(action, clientID, username, password, "")

	return true, nil
}

//ACL 验证访问主题授权
func (slf *AuthMYSQL) ACL(action, clientID, username, ip, topic string) (bool, error) {
	return true, nil
}

func (slf *AuthMYSQL) doAuthCache(action, clientID, username, password, topic string) *authCache {
	authc, found := slf._ca.Get(username)
	if found {
		return authc.(*authCache)
	}
	return nil
}

func (slf *AuthMYSQL) addAuthCache(action, clientID, username, password, topic string) {
	slf._ca.Set(username, &authCache{_action: action, _username: username, _clientID: clientID, _password: password, _topic: topic}, cache.DefaultExpiration)
}
