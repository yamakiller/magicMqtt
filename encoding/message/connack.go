package message

import (
	"encoding/binary"
	"encoding/json"
	"io"
)

//Connack message
type Connack struct {
	FixedHeader
	Reserved   uint8
	ReturnCode uint8
}

func (slf *Connack) decode(reader io.Reader) error {
	binary.Read(reader, binary.BigEndian, &slf.Reserved)
	binary.Read(reader, binary.BigEndian, &slf.ReturnCode)

	return nil
}

//WriteTo Connack message write to io
func (slf Connack) WriteTo(w io.Writer) (int64, error) {
	var fsize = 2
	size, err := slf.FixedHeader.writeTo(fsize, w)
	if err != nil {
		return 0, err
	}

	binary.Write(w, binary.BigEndian, slf.Reserved)
	binary.Write(w, binary.BigEndian, slf.ReturnCode)

	return int64(fsize) + size, nil
}

//String Returns Connack message object of string
func (slf *Connack) String() string {
	b, _ := json.Marshal(slf)
	return string(b)
}
