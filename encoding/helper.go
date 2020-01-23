package encoding

import (
	"encoding/binary"
	"errors"
	"io"
)

//overflow error
var errOverflow = errors.New("readvarint: varint overflows a 32-bit integer")

//ReadVarint Return int
func ReadVarint(reader io.Reader) (int, error) {
	var RemainingLength uint8
	m := 1
	v := 0

	for i := 0; i < 4; i++ {
		binary.Read(reader, binary.BigEndian, &RemainingLength)
		v += (int(RemainingLength) & 0x7F) * m

		m *= 0x80
		if m > 0x200000 {
			return 0, errOverflow
		}
		if (RemainingLength & 0x80) == 0 {
			break
		}
	}

	return v, nil
}

//WriteVarint write size
func WriteVarint(writer io.Writer, size int) (int, error) {
	var encodeByte uint8
	x := size
	var i int

	for i = 0; x > 0 && i < 4; i++ {
		encodeByte = uint8(x % 0x80)
		x = x / 0x80
		if x > 0 {
			encodeByte |= 0x80
		}

		binary.Write(writer, binary.BigEndian, encodeByte)
	}

	return i, nil
}
