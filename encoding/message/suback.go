package message

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
)

//Suback message
type Suback struct {
	FixedHeader
	PacketIdentifier uint16
	Qos              []byte
}

//WriteTo Suback message write to io
func (slf *Suback) WriteTo(w io.Writer) (int64, error) {
	var fsize = 2 + len(slf.Qos)

	size, err := slf.FixedHeader.writeTo(fsize, w)
	if err != nil {
		return 0, err
	}

	binary.Write(w, binary.BigEndian, slf.PacketIdentifier)
	io.Copy(w, bytes.NewReader(slf.Qos))

	return int64(size) + int64(fsize), nil
}

func (slf *Suback) decode(reader io.Reader) error {
	var remaining uint8
	remaining = uint8(slf.FixedHeader.RemainingLength)
	binary.Read(reader, binary.BigEndian, &slf.PacketIdentifier)

	remaining -= 2
	buffer := bytes.NewBuffer(nil)
	for i := 0; i <= int(remaining); i++ {
		var value uint8 = 0
		binary.Read(reader, binary.BigEndian, &value)
		binary.Write(buffer, binary.BigEndian, value)
		remaining--
	}

	slf.Qos = buffer.Bytes()
	return nil
}

//String Returns Suback message object of string
func (slf *Suback) String() string {
	b, _ := json.Marshal(slf)
	return string(b)
}
