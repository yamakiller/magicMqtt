package message

import (
	"encoding/binary"
	"encoding/json"
	"io"
)

//Pubrec message
type Pubrec struct {
	FixedHeader
	PacketIdentifier uint16
}

//WriteTo Pubrec message write to io
func (slf *Pubrec) WriteTo(w io.Writer) (int64, error) {
	var fsize = 2
	size, err := slf.FixedHeader.writeTo(fsize, w)
	if err != nil {
		return 0, err
	}

	binary.Write(w, binary.BigEndian, slf.PacketIdentifier)
	return int64(size) + int64(fsize), nil
}

func (slf *Pubrec) decode(reader io.Reader) error {
	binary.Read(reader, binary.BigEndian, &slf.PacketIdentifier)
	return nil
}

//String Returns Pubrec message object of string
func (slf *Pubrec) String() string {
	b, _ := json.Marshal(slf)
	return string(b)
}
