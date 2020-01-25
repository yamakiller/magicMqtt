package sessions

type Session struct {
	_subscribeHistory map[string]int
	_subscribedTopics map[string]int
}
