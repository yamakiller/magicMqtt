package message

import (
	"encoding/binary"
	"encoding/json"
	"io"
)

//Puback message
type Puback struct {
	FixedHeader
	PacketIdentifier uint16
}

func (slf *Puback) decode(reader io.Reader) error {
	binary.Read(reader, binary.BigEndian, &slf.PacketIdentifier)
	return nil
}

//WriteTo Puback message write to io
func (slf *Puback) WriteTo(w io.Writer) (int64, error) {
	var fsize = 2
	size, err := slf.FixedHeader.writeTo(fsize, w)
	if err != nil {
		return 0, err
	}

	binary.Write(w, binary.BigEndian, slf.PacketIdentifier)
	return int64(size) + int64(fsize), nil
}

//String Returns Puback message object of string
func (slf *Puback) String() string {
	b, _ := json.Marshal(slf)
	return string(b)
}
