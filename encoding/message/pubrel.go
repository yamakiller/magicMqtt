package message

import (
	"encoding/binary"
	"encoding/json"
	"io"
)

//Pubrel message
type Pubrel struct {
	FixedHeader
	PacketIdentifier uint16
}

//WriteTo Pubrel message write to io
func (slf *Pubrel) WriteTo(w io.Writer) (int64, error) {
	var fsize = 2
	size, err := slf.FixedHeader.writeTo(fsize, w)
	if err != nil {
		return 0, err
	}

	binary.Write(w, binary.BigEndian, slf.PacketIdentifier)
	return int64(size) + int64(fsize), nil
}

func (slf *Pubrel) decode(reader io.Reader) error {
	binary.Read(reader, binary.BigEndian, &slf.PacketIdentifier)
	return nil
}

//String Returns Pubrel message object of string
func (slf *Pubrel) String() string {
	b, _ := json.Marshal(slf)
	return string(b)
}
