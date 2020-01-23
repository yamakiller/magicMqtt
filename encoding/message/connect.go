package message

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

//Connect message
type Connect struct {
	FixedHeader
	Magic        []byte `json:"magic"`
	Version      uint8  `json:"version"`
	Flag         uint8  `json:"flag"`
	KeepAlive    uint16 `json:"keep_alive"`
	Identifier   string `json:"identifier"`
	Will         *Will  `json:"will"`
	CleanSession bool   `json:"clean_session"`
	UserName     []byte `json:"user_name"`
	Password     []byte `json:"password"`
}

//WriteTo Write Connect message to io
func (slf *Connect) WriteTo(w io.Writer) (int64, error) {
	var headerLength uint16 = uint16(len(slf.Magic))
	var size int = 0

	if slf.CleanSession {
		slf.Flag |= 0x02
	}
	if slf.Will != nil {
		slf.Flag |= 0x04
		switch slf.Will.Qos {
		case 1:
			slf.Flag |= 0x08
		case 2:
			slf.Flag |= 0x18
		}
	}
	if len(slf.UserName) > 0 {
		slf.Flag |= 0x80
	}
	if len(slf.Password) > 0 {
		slf.Flag |= 0x40
	}

	size += 2 + len(slf.Magic)
	size += 1 + 1 + 2
	if slf.Identifier != "" {
		size += 2 + len(slf.Identifier)
	}
	if (int(slf.Flag)&0x04 > 0) && slf.Will != nil {
		size += slf.Will.Size()
	}
	if int(slf.Flag)&0x80 > 0 {
		size += 2 + len(slf.UserName)
	}
	if int(slf.Flag)&0x40 > 0 {
		size += 2 + len(slf.Password)
	}

	slf.FixedHeader.writeTo(size, w)
	err := binary.Write(w, binary.BigEndian, headerLength)
	if err != nil {
		return 0, err
	}

	w.Write(slf.Magic)
	binary.Write(w, binary.BigEndian, slf.Version)
	binary.Write(w, binary.BigEndian, slf.Flag)
	binary.Write(w, binary.BigEndian, slf.KeepAlive)

	var Length uint16 = 0

	if slf.Identifier != "" {
		Length = uint16(len(slf.Identifier))
	}
	binary.Write(w, binary.BigEndian, Length)
	if Length > 0 {
		w.Write([]byte(slf.Identifier))
	}

	if (int(slf.Flag)&0x04 > 0) && slf.Will != nil {
		slf.Will.WriteTo(w)
	}

	if int(slf.Flag)&0x80 > 0 {
		Length = uint16(len(slf.UserName))
		err = binary.Write(w, binary.BigEndian, Length)
		w.Write(slf.UserName)
	}
	if int(slf.Flag)&0x40 > 0 {
		Length = uint16(len(slf.Password))
		err = binary.Write(w, binary.BigEndian, Length)
		w.Write(slf.Password)
	}
	return int64(size), nil
}

func (slf *Connect) decode(reader io.Reader) error {
	var Length uint16

	offset := 0
	buffer := make([]byte, slf.FixedHeader.RemainingLength)

	w := bytes.NewBuffer(buffer)
	w.Reset()
	//TODO: should check error
	io.CopyN(w, reader, int64(slf.FixedHeader.RemainingLength))

	buffer = w.Bytes()
	reader = bytes.NewReader(buffer)

	binary.Read(reader, binary.BigEndian, &Length)
	offset += 2

	if slf.FixedHeader.RemainingLength < offset+int(Length) {
		return fmt.Errorf("Length overs buffer size. %d, %d",
			slf.FixedHeader.RemainingLength, offset+int(Length))
	}

	slf.Magic = buffer[offset : offset+int(Length)]
	offset += int(Length)

	if offset > len(buffer) {
		return fmt.Errorf("offset: %d, buffer: %d", offset, len(buffer))
	}

	nr := bytes.NewReader(buffer[offset:])
	binary.Read(nr, binary.BigEndian, &slf.Version)
	binary.Read(nr, binary.BigEndian, &slf.Flag)
	binary.Read(nr, binary.BigEndian, &slf.KeepAlive)
	offset += 1 + 1 + 2

	// order Client ClientIdentifier, Will Topic, Will Message, User Name, Password
	var ClientIdentifierLength uint16
	binary.Read(nr, binary.BigEndian, &ClientIdentifierLength)
	offset += 2

	if ClientIdentifierLength > 0 {
		slf.Identifier = string(buffer[offset : offset+int(ClientIdentifierLength)])
		offset += int(ClientIdentifierLength)
	}

	if int(slf.Flag)&0x04 > 0 {
		will := &Will{}

		nr := bytes.NewReader(buffer[offset:])
		binary.Read(nr, binary.BigEndian, &ClientIdentifierLength)
		offset += 2
		will.Topic = string(buffer[offset : offset+int(ClientIdentifierLength)])
		offset += int(ClientIdentifierLength)

		nr = bytes.NewReader(buffer[offset:])
		binary.Read(nr, binary.BigEndian, &ClientIdentifierLength)
		offset += 2
		will.Message = string(buffer[offset : offset+int(ClientIdentifierLength)])
		offset += int(ClientIdentifierLength)

		if int(slf.Flag)&0x32 > 0 {
			will.Retain = true
		}

		q := (int(slf.Flag) >> 3)

		if q&0x02 > 0 {
			will.Qos = 2
		} else if q&0x01 > 0 {
			will.Qos = 1
		}
		slf.Will = will
	}

	if int(slf.Flag)&0x80 > 0 {
		nr := bytes.NewReader(buffer[offset:])
		binary.Read(nr, binary.BigEndian, &Length)
		offset += 2
		slf.UserName = buffer[offset : offset+int(Length)]
		offset += int(Length)
	}

	if int(slf.Flag)&0x40 > 0 {
		nr := bytes.NewReader(buffer[offset:])
		offset += 2
		binary.Read(nr, binary.BigEndian, &Length)
		slf.Password = buffer[offset : offset+int(Length)]
		offset += int(Length)
	}

	if int(slf.Flag)&0x02 > 0 {
		slf.CleanSession = true
	}

	return nil
}

//String Returns Connect message object of string
func (slf *Connect) String() string {
	b, _ := json.Marshal(slf)
	return string(b)
}
