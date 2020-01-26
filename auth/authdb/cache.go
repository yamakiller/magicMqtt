package authdb

type authCache struct {
	_action   string
	_username string
	_clientID string
	_password string
	_topic    string
}

/*func doAuthCache(action, clientID, username, password, topic string) *authCache {
	authc, found := ca.Get(username)
	if found {
		return authc.(*authCache)
	}
	return nil
}

func addAuthCache(action, clientID, username, password, topic string) {
	ca.Set(username, &authCache{_action: action, _username: username, _clientID: clientID, _password: password, _topic: topic}, cache.DefaultExpiration)
}*/
