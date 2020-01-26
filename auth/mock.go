package auth

//Mock 模拟数据
type Mock struct {
}

//Connect ...
func (slf *Mock) Connect(clientID, username, password string) (bool, error) {
	return true, nil
}

//ACL ...
func (slf *Mock) ACL(action, clientID, username, ip, topic string) (bool, error) {
	return true, nil
}
