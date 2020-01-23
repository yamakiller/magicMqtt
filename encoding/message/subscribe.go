package message

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
)

//SubscribePayload ...
type SubscribePayload struct {
	TopicPath    string
	RequestedQos uint8
}

//Subscribe message
type Subscribe struct {
	FixedHeader
	PacketIdentifier uint16
	Payload          []SubscribePayload
}

//WriteTo Subscribe message write to io
func (slf *Subscribe) WriteTo(w io.Writer) (int64, error) {
	var total int = 0
	total += 2

	for i := 0; i < len(slf.Payload); i++ {
		var length uint16 = uint16(len(slf.Payload[i].TopicPath))
		total += 2 + int(length) + 1
	}

	headerLen, _ := slf.FixedHeader.writeTo(total, w)
	total += total + int(headerLen)

	binary.Write(w, binary.BigEndian, slf.PacketIdentifier)
	for i := 0; i < len(slf.Payload); i++ {
		var length uint16 = uint16(len(slf.Payload[i].TopicPath))
		binary.Write(w, binary.BigEndian, length)
		w.Write([]byte(slf.Payload[i].TopicPath))
		binary.Write(w, binary.BigEndian, slf.Payload[i].RequestedQos)
	}

	return int64(total), nil
}

func (slf *Subscribe) decode(reader io.Reader) error {
	remaining := slf.RemainingLength

	binary.Read(reader, binary.BigEndian, &slf.PacketIdentifier)
	remaining -= int(2)

	buffer := bytes.NewBuffer(nil)
	for remaining > 0 {
		var length uint16

		m := SubscribePayload{}
		binary.Read(reader, binary.BigEndian, &length)

		_, _ = io.CopyN(buffer, reader, int64(length))

		m.TopicPath = string(buffer.Bytes())
		binary.Read(reader, binary.BigEndian, &m.RequestedQos)
		slf.Payload = append(slf.Payload, m)

		buffer.Reset()
		remaining -= (int(length) + 1 + 2)
	}

	return nil
}

//String Returns Subscribe message object of string
func (slf *Subscribe) String() string {
	b, _ := json.Marshal(slf)
	return string(b)
}
