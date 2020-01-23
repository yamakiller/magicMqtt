package subscribe

import "encoding/json"

//Set subscribe set
type Set struct {
	ClientID    string `json:"client_id"`
	TopicFilter string `json:"topic_filter"`
	QoS         int    `json:"qos"`
}

//String Returns subscribe set object of string [json]
func (slf *Set) String() string {
	b, _ := json.Marshal(slf)
	return string(b)
}
