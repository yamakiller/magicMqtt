package authdb

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/yamakiller/magicLibs/dbs"
)

//Setup 安装authdb所需资源
func Setup(configFile string) {
	content, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Println("read config error, ", err.Error())
		return
	}

	var config dbs.MySQLGormDeploy
	err = json.Unmarshal(content, &config)
	if err != nil {
		fmt.Println("json unmarshal error, ", err.Error())
		return
	}
	client := &dbs.MySQLGORM{}
	err = client.Initial(config.DSN, config.Max, config.Idle, config.Life)
	if err != nil {
		fmt.Println("connect mysql error, ", err.Error())
		return
	}

	if err := client.DB().CreateTable(AuthUser{}).Error; err != nil {
		fmt.Println("auth user table create error, ", err.Error())
		return
	}

	client.Close()
	fmt.Println("setup complate")
}
