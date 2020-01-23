package message

import (
	"encoding/binary"
	"encoding/json"
	"io"
)

//Unsuback ...
type Unsuback struct {
	FixedHeader
	PacketIdentifier uint16
}

//WriteTo 写协议数据包
func (slf *Unsuback) WriteTo(w io.Writer) (int64, error) {
	var fsize = 2
	size, err := slf.FixedHeader.writeTo(fsize, w)
	if err != nil {
		return 0, err
	}

	binary.Write(w, binary.BigEndian, slf.PacketIdentifier)
	return int64(size) + int64(fsize), nil
}

func (slf *Unsuback) decode(reader io.Reader) error {
	binary.Read(reader, binary.BigEndian, &slf.PacketIdentifier)
	return nil
}

//String unsuback 转换为json字符串
func (slf *Unsuback) String() string {
	b, _ := json.Marshal(slf)
	return string(b)
}
