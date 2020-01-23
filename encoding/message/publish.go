package message

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

//Publish message
type Publish struct {
	FixedHeader      `json:"header"`
	TopicName        string      `json:"topic_name"`
	PacketIdentifier uint16      `json:"identifier"`
	Payload          []byte      `json:"payload"`
	Opaque           interface{} `json:"-"`
}

func (slf *Publish) decode(reader io.Reader) error {
	var length uint16
	remaining := slf.FixedHeader.RemainingLength
	binary.Read(reader, binary.BigEndian, &length)
	remaining -= 2

	if remaining < 1 {
		return fmt.Errorf("something wrong. (probably crouppted data?)")
	}

	buffer := make([]byte, remaining)
	offset := 0
	for offset < remaining {
		if offset > remaining {
			panic("something went to wrong(offset overs remianing length)")
		}

		i, err := reader.Read(buffer[offset:])
		offset += i
		if err != nil && offset < remaining {
			// if we read whole size of message, ignore error at this time.
			return err
		}
	}

	if int(length) > len(buffer) {
		return fmt.Errorf("publish length: %d, buffer: %d", length, len(buffer))
	}

	slf.TopicName = string(buffer[0:length])
	payloadOffset := length
	if slf.FixedHeader.QosLevel > 0 {
		binary.Read(bytes.NewReader(buffer[length:]), binary.BigEndian, &slf.PacketIdentifier)
		payloadOffset += 2
	}
	slf.Payload = buffer[payloadOffset:]
	return nil
}

//WriteTo Publish message write to IO
func (slf *Publish) WriteTo(w io.Writer) (int64, error) {
	var size uint16 = uint16(len(slf.TopicName))
	total := 2 + int(size)
	if slf.QosLevel > 0 {
		total += 2
	}
	total += len(slf.Payload)

	headerLen, e := slf.FixedHeader.writeTo(total, w)
	if e != nil {
		return 0, e
	}
	total += int(size)

	e = binary.Write(w, binary.BigEndian, size)
	if e != nil {
		return 0, e
	}
	w.Write([]byte(slf.TopicName))
	if slf.QosLevel > 0 {
		e = binary.Write(w, binary.BigEndian, slf.PacketIdentifier)
	}
	if e != nil {
		return 0, e
	}
	_, e = w.Write(slf.Payload)
	if e != nil {
		return 0, e
	}

	return int64(int(total) + int(headerLen)), nil
}

//String Returns Publish message object of string
func (slf *Publish) String() string {
	b, _ := json.Marshal(slf)
	return string(b)
}
