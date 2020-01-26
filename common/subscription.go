package common

type Subscription struct {
	Client    string
	Topic     string
	Qos       byte
	Share     bool
	GroupName string
}
