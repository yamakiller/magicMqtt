package message

import (
	"encoding/binary"
	"encoding/json"
	"io"
)

//Will will protocol message
type Will struct {
	Qos     uint8  `json:"qos"`
	Topic   string `json:"topic"`
	Message string `json:"message"`
	Retain  bool   `json:"retain"`
}

//WriteTo Wirte will protocol message to io
func (slf *Will) WriteTo(w io.Writer) (int64, error) {
	var topicLen uint16
	var size int = 0

	topicLen = uint16(len(slf.Topic))
	err := binary.Write(w, binary.BigEndian, topicLen)
	w.Write([]byte(slf.Topic))
	size += 2 + int(topicLen)

	msgLen := uint16(len(slf.Message))
	err = binary.Write(w, binary.BigEndian, msgLen)
	w.Write([]byte(slf.Message))
	size += 2 + int(msgLen)

	return int64(size), err
}

//Size Returns will protocol message size
func (slf *Will) Size() int {
	var size int = 0

	topicLen := uint16(len(slf.Topic))
	size += 2 + int(topicLen)

	msgLen := uint16(len(slf.Message))
	size += 2 + int(msgLen)

	return size
}

//String Return will protocol message string
func (slf *Will) String() string {
	b, _ := json.Marshal(slf)
	return string(b)
}
