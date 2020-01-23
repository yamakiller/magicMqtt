package message

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
)

//Unsubscribe message
type Unsubscribe struct {
	FixedHeader
	TopicName        string
	PacketIdentifier uint16
	Payload          []SubscribePayload
}

func (slf *Unsubscribe) decode(reader io.Reader) error {
	remaining := slf.RemainingLength

	binary.Read(reader, binary.BigEndian, &slf.PacketIdentifier)
	remaining -= int(2)

	buffer := bytes.NewBuffer(nil)
	for remaining > 0 {
		var length uint16 = 0

		m := SubscribePayload{}
		binary.Read(reader, binary.BigEndian, &length)
		_, _ = io.CopyN(buffer, reader, int64(length))
		m.TopicPath = string(buffer.Bytes())
		buffer.Reset()
		slf.Payload = append(slf.Payload, m)
		remaining -= (int(length) + 1 + 2)
	}

	return nil
}

//WriteTo Unsubscribe message write to io
func (slf *Unsubscribe) WriteTo(w io.Writer) (int64, error) {
	var total = 2
	for i := 0; i < len(slf.Payload); i++ {
		length := uint16(len(slf.Payload[i].TopicPath))
		total += 2 + int(length)
	}

	headerLen, _ := slf.FixedHeader.writeTo(total, w)
	total += total + int(headerLen)

	binary.Write(w, binary.BigEndian, slf.PacketIdentifier)
	for i := 0; i < len(slf.Payload); i++ {
		var length uint16 = 0
		length = uint16(len(slf.Payload[i].TopicPath))
		binary.Write(w, binary.BigEndian, length)
		w.Write([]byte(slf.Payload[i].TopicPath))
	}

	return int64(total), nil
}

//String Returns Unsubscribe message object of string
func (slf *Unsubscribe) String() string {
	b, _ := json.Marshal(slf)
	return string(b)
}
