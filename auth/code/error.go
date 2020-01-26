package code

import "errors"

var (
	//ErrAuthClientNot 客户端未授权
	ErrAuthClientNot = errors.New("client not auth")
	//ErrAuthClientUserNameOrPwd 客户段账户密码错误
	ErrAuthClientUserNameOrPwd = errors.New("client password error")
)
