package topics

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/yamakiller/magicMqtt/encoding/message"
)

const (
	//QosAtMostOnce  == 0
	QosAtMostOnce byte = iota
	//QosAtLeastOnce == 1
	QosAtLeastOnce
	//QosExactlyOnce == 2
	QosExactlyOnce
	//QosFailure ...
	QosFailure = 0x80
)

var _ TopicsProvider = (*memTopics)(nil)

type memTopics struct {
	// Sub/unsub mutex
	_smu sync.RWMutex
	// Subscription tree
	_sroot *snode

	// Retained message mutex
	_rmu sync.RWMutex
	// Retained messages topic tree
	_rroot *rnode
}

func init() {
	Register("mem", newMemProvider())
}

// NewMemProvider returns an new instance of the memTopics, which is implements the
// TopicsProvider interface. memProvider is a hidden struct that stores the topic
// subscriptions and retained messages in memory. The content is not persistend so
// when the server goes, everything will be gone. Use with care.
func newMemProvider() *memTopics {
	return &memTopics{
		_sroot: newSNode(),
		_rroot: newRNode(),
	}
}

//ValidQos Qos 是否有效
func ValidQos(qos byte) bool {
	return qos == QosAtMostOnce || qos == QosAtLeastOnce || qos == QosExactlyOnce
}

//Subscribe 订阅
func (slf *memTopics) Subscribe(topic []byte, qos byte, sub interface{}) (byte, error) {
	if !ValidQos(qos) {
		return QosFailure, fmt.Errorf("Invalid QoS %d", qos)
	}

	if sub == nil {
		return QosFailure, fmt.Errorf("Subscriber cannot be nil")
	}

	slf._smu.Lock()
	defer slf._smu.Unlock()

	if qos > QosExactlyOnce {
		qos = QosExactlyOnce
	}

	if err := slf._sroot.sinsert(topic, qos, sub); err != nil {
		return QosFailure, err
	}

	return qos, nil
}

func (slf *memTopics) Unsubscribe(topic []byte, sub interface{}) error {
	slf._smu.Lock()
	defer slf._smu.Unlock()

	return slf._sroot.sremove(topic, sub)
}

// Returned values will be invalidated by the next Subscribers call
func (slf *memTopics) Subscribers(topic []byte, qos byte, subs *[]interface{}, qoss *[]byte) error {
	if !ValidQos(qos) {
		return fmt.Errorf("Invalid QoS %d", qos)
	}

	slf._smu.RLock()
	defer slf._smu.RUnlock()

	*subs = (*subs)[0:0]
	*qoss = (*qoss)[0:0]

	return slf._sroot.smatch(topic, qos, subs, qoss)
}

func (slf *memTopics) Retain(msg *message.Publish) error {
	slf._rmu.Lock()
	defer slf._rmu.Unlock()

	// So apparently, at least according to the MQTT Conformance/Interoperability
	// Testing, that a payload of 0 means delete the retain message.
	// https://eclipse.org/paho/clients/testing/
	if len(msg.Payload) == 0 {
		return slf._rroot.rremove([]byte(msg.TopicName))
	}

	return slf._rroot.rinsertOrUpdate([]byte(msg.TopicName), msg)
}

func (slf *memTopics) Retained(topic []byte, msgs *[]*message.Publish) error {
	slf._rmu.RLock()
	defer slf._rmu.RUnlock()

	return slf._rroot.rmatch(topic, msgs)
}

//Close 关闭
func (slf *memTopics) Close() error {
	slf._sroot = nil
	slf._rroot = nil
	return nil
}

// subscrition nodes
type snode struct {
	// If this is the end of the topic string, then add subscribers here
	_subs []interface{}
	_qos  []byte

	// Otherwise add the next topic level here
	_snodes map[string]*snode
}

func newSNode() *snode {
	return &snode{
		_snodes: make(map[string]*snode),
	}
}

func (slf *snode) sinsert(topic []byte, qos byte, sub interface{}) error {
	// If there's no more topic levels, that means we are at the matching snode
	// to insert the subscriber. So let's see if there's such subscriber,
	// if so, update it. Otherwise insert it.
	if len(topic) == 0 {
		// Let's see if the subscriber is already on the list. If yes, update
		// QoS and then return.
		for i := range slf._subs {
			if equal(slf._subs[i], sub) {
				slf._qos[i] = qos
				return nil
			}
		}

		// Otherwise add.
		slf._subs = append(slf._subs, sub)
		slf._qos = append(slf._qos, qos)

		return nil
	}

	// Not the last level, so let's find or create the next level snode, and
	// recursively call it's insert().

	// ntl = next topic level
	ntl, rem, err := nextTopicLevel(topic)
	if err != nil {
		return err
	}

	level := string(ntl)

	// Add snode if it doesn't already exist
	n, ok := slf._snodes[level]
	if !ok {
		n = newSNode()
		slf._snodes[level] = n
	}

	return n.sinsert(rem, qos, sub)
}

// This remove implementation ignores the QoS, as long as the subscriber
// matches then it's removed
func (slf *snode) sremove(topic []byte, sub interface{}) error {
	// If the topic is empty, it means we are at the final matching snode. If so,
	// let's find the matching subscribers and remove them.
	if len(topic) == 0 {
		// If subscriber == nil, then it's signal to remove ALL subscribers
		if sub == nil {
			slf._subs = slf._subs[0:0]
			slf._qos = slf._qos[0:0]
			return nil
		}

		// If we find the subscriber then remove it from the list. Technically
		// we just overwrite the slot by shifting all other items up by one.
		for i := range slf._subs {
			if equal(slf._subs[i], sub) {
				slf._subs = append(slf._subs[:i], slf._subs[i+1:]...)
				slf._qos = append(slf._qos[:i], slf._qos[i+1:]...)
				return nil
			}
		}

		return fmt.Errorf("No topic found for subscriber")
	}

	// Not the last level, so let's find the next level snode, and recursively
	// call it's remove().

	// ntl = next topic level
	ntl, rem, err := nextTopicLevel(topic)
	if err != nil {
		return err
	}

	level := string(ntl)

	// Find the snode that matches the topic level
	n, ok := slf._snodes[level]
	if !ok {
		return fmt.Errorf("No topic found")
	}

	// Remove the subscriber from the next level snode
	if err := n.sremove(rem, sub); err != nil {
		return err
	}

	// If there are no more subscribers and snodes to the next level we just visited
	// let's remove it
	if len(n._subs) == 0 && len(n._snodes) == 0 {
		delete(slf._snodes, level)
	}

	return nil
}

// smatch() returns all the subscribers that are subscribed to the topic. Given a topic
// with no wildcards (publish topic), it returns a list of subscribers that subscribes
// to the topic. For each of the level names, it's a match
// - if there are subscribers to '#', then all the subscribers are added to result set
func (slf *snode) smatch(topic []byte, qos byte, subs *[]interface{}, qoss *[]byte) error {
	// If the topic is empty, it means we are at the final matching snode. If so,
	// let's find the subscribers that match the qos and append them to the list.
	if len(topic) == 0 {
		slf.matchQos(qos, subs, qoss)
		if mwcn, _ := slf._snodes[MWC]; mwcn != nil {
			mwcn.matchQos(qos, subs, qoss)
		}
		return nil
	}

	// ntl = next topic level
	ntl, rem, err := nextTopicLevel(topic)
	if err != nil {
		return err
	}

	level := string(ntl)

	for k, n := range slf._snodes {
		// If the key is "#", then these subscribers are added to the result set
		if k == MWC {
			n.matchQos(qos, subs, qoss)
		} else if k == SWC || k == level {
			if err := n.smatch(rem, qos, subs, qoss); err != nil {
				return err
			}
		}
	}

	return nil
}

// retained message nodes
type rnode struct {
	// If this is the end of the topic string, then add retained messages here
	_msg *message.Publish
	// Otherwise add the next topic level here
	_rnodes map[string]*rnode
}

func newRNode() *rnode {
	return &rnode{
		_rnodes: make(map[string]*rnode),
	}
}

func (slf *rnode) rinsertOrUpdate(topic []byte, msg *message.Publish) error {
	// If there's no more topic levels, that means we are at the matching rnode.
	if len(topic) == 0 {
		// Reuse the message if possible
		slf._msg = msg

		return nil
	}

	// Not the last level, so let's find or create the next level snode, and
	// recursively call it's insert().

	// ntl = next topic level
	ntl, rem, err := nextTopicLevel(topic)
	if err != nil {
		return err
	}

	level := string(ntl)

	// Add snode if it doesn't already exist
	n, ok := slf._rnodes[level]
	if !ok {
		n = newRNode()
		slf._rnodes[level] = n
	}

	return n.rinsertOrUpdate(rem, msg)
}

// Remove the retained message for the supplied topic
func (slf *rnode) rremove(topic []byte) error {
	// If the topic is empty, it means we are at the final matching rnode. If so,
	// let's remove the buffer and message.
	if len(topic) == 0 {
		slf._msg = nil
		return nil
	}

	// Not the last level, so let's find the next level rnode, and recursively
	// call it's remove().

	// ntl = next topic level
	ntl, rem, err := nextTopicLevel(topic)
	if err != nil {
		return err
	}

	level := string(ntl)

	// Find the rnode that matches the topic level
	n, ok := slf._rnodes[level]
	if !ok {
		return fmt.Errorf("No topic found")
	}

	// Remove the subscriber from the next level rnode
	if err := n.rremove(rem); err != nil {
		return err
	}

	// If there are no more rnodes to the next level we just visited let's remove it
	if len(n._rnodes) == 0 {
		delete(slf._rnodes, level)
	}

	return nil
}

// rmatch() finds the retained messages for the topic and qos provided. It's somewhat
// of a reverse match compare to match() since the supplied topic can contain
// wildcards, whereas the retained message topic is a full (no wildcard) topic.
func (slf *rnode) rmatch(topic []byte, msgs *[]*message.Publish) error {
	// If the topic is empty, it means we are at the final matching rnode. If so,
	// add the retained msg to the list.
	if len(topic) == 0 {
		if slf._msg != nil {
			*msgs = append(*msgs, slf._msg)
		}
		return nil
	}

	// ntl = next topic level
	ntl, rem, err := nextTopicLevel(topic)
	if err != nil {
		return err
	}

	level := string(ntl)

	if level == MWC {
		// If '#', add all retained messages starting this node
		slf.allRetained(msgs)
	} else if level == SWC {
		// If '+', check all nodes at this level. Next levels must be matched.
		for _, n := range slf._rnodes {
			if err := n.rmatch(rem, msgs); err != nil {
				return err
			}
		}
	} else {
		// Otherwise, find the matching node, go to the next level
		if n, ok := slf._rnodes[level]; ok {
			if err := n.rmatch(rem, msgs); err != nil {
				return err
			}
		}
	}

	return nil
}

func (slf *rnode) allRetained(msgs *[]*message.Publish) {
	if slf._msg != nil {
		*msgs = append(*msgs, slf._msg)
	}

	for _, n := range slf._rnodes {
		n.allRetained(msgs)
	}
}

const (
	stateCHR byte = iota // Regular character
	stateMWC             // Multi-level wildcard
	stateSWC             // Single-level wildcard
	stateSEP             // Topic level separator
	stateSYS             // System level topic ($)
)

// Returns topic level, remaining topic levels and any errors
func nextTopicLevel(topic []byte) ([]byte, []byte, error) {
	s := stateCHR

	for i, c := range topic {
		switch c {
		case '/':
			if s == stateMWC {
				return nil, nil, fmt.Errorf("Multi-level wildcard found in topic and it's not at the last level")
			}

			if i == 0 {
				return []byte(SWC), topic[i+1:], nil
			}

			return topic[:i], topic[i+1:], nil

		case '#':
			if i != 0 {
				return nil, nil, fmt.Errorf("Wildcard character '#' must occupy entire topic level")
			}

			s = stateMWC

		case '+':
			if i != 0 {
				return nil, nil, fmt.Errorf("Wildcard character '+' must occupy entire topic level")
			}

			s = stateSWC

		// case '$':
		// 	if i == 0 {
		// 		return nil, nil, fmt.Errorf("Cannot publish to $ topics")
		// 	}

		// 	s = stateSYS

		default:
			if s == stateMWC || s == stateSWC {
				return nil, nil, fmt.Errorf("Wildcard characters '#' and '+' must occupy entire topic level")
			}

			s = stateCHR
		}
	}

	// If we got here that means we didn't hit the separator along the way, so the
	// topic is either empty, or does not contain a separator. Either way, we return
	// the full topic
	return topic, nil, nil
}

// The QoS of the payload messages sent in response to a subscription must be the
// minimum of the QoS of the originally published message (in this case, it's the
// qos parameter) and the maximum QoS granted by the server (in this case, it's
// the QoS in the topic tree).
//
// It's also possible that even if the topic matches, the subscriber is not included
// due to the QoS granted is lower than the published message QoS. For example,
// if the client is granted only QoS 0, and the publish message is QoS 1, then this
// client is not to be send the published message.
func (slf *snode) matchQos(qos byte, subs *[]interface{}, qoss *[]byte) {
	for _, sub := range slf._subs {
		// If the published QoS is higher than the subscriber QoS, then we skip the
		// subscriber. Otherwise, add to the list.
		// if qos >= this.qos[i] {
		*subs = append(*subs, sub)
		*qoss = append(*qoss, qos)
		// }
	}
}

func equal(k1, k2 interface{}) bool {
	if reflect.TypeOf(k1) != reflect.TypeOf(k2) {
		return false
	}

	if reflect.ValueOf(k1).Kind() == reflect.Func {
		return &k1 == &k2
	}

	if k1 == k2 {
		return true
	}

	switch k1 := k1.(type) {
	case string:
		return k1 == k2.(string)

	case int64:
		return k1 == k2.(int64)

	case int32:
		return k1 == k2.(int32)

	case int16:
		return k1 == k2.(int16)

	case int8:
		return k1 == k2.(int8)

	case int:
		return k1 == k2.(int)

	case float32:
		return k1 == k2.(float32)

	case float64:
		return k1 == k2.(float64)

	case uint:
		return k1 == k2.(uint)

	case uint8:
		return k1 == k2.(uint8)

	case uint16:
		return k1 == k2.(uint16)

	case uint32:
		return k1 == k2.(uint32)

	case uint64:
		return k1 == k2.(uint64)

	case uintptr:
		return k1 == k2.(uintptr)
	}

	return false
}
