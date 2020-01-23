package message

import (
	"encoding/binary"
	"encoding/json"
	"io"

	"github.com/yamakiller/magicMqtt/encoding"
)

//FixedHeader packet fixed header
type FixedHeader struct {
	Type            encoding.PType
	Dupe            bool
	QosLevel        int
	Retain          int
	RemainingLength int
}

//GetType Return pakcet type
func (slf *FixedHeader) GetType() encoding.PType {
	return slf.Type
}

//GetTypeAsString  Return packet type string
func (slf *FixedHeader) GetTypeAsString() string {
	switch slf.Type {
	case encoding.PTypeReserved1:
		return "unknown"
	case encoding.PTypeConnect:
		return "connect"
	case encoding.PTypeConnack:
		return "connack"
	case encoding.PTypePublish:
		return "publish"
	case encoding.PTypePuback:
		return "puback"
	case encoding.PTypePubrec:
		return "pubrec"
	case encoding.PTypePubrel:
		return "pubrel"
	case encoding.PTypePubcomp:
		return "pubcomp"
	case encoding.PTypeSubscribe:
		return "subscribe"
	case encoding.PTypeSuback:
		return "suback"
	case encoding.PTypeUnsubscribe:
		return "unsubscribe"
	case encoding.PTypeUnsuback:
		return "unsuback"
	case encoding.PTypePingreq:
		return "pingreq"
	case encoding.PTypePingresp:
		return "pingresp"
	case encoding.PTypeDisconnect:
		return "disconnect"
	case encoding.PTypeReserved2:
		return "unknown"
	default:
		return "unknown"
	}
}

func (slf *FixedHeader) writeTo(length int, w io.Writer) (int64, error) {
	var flag uint8 = uint8(slf.Type << 0x04)

	if slf.Retain > 0 {
		flag |= 0x01
	}

	if slf.QosLevel > 0 {
		if slf.QosLevel == 1 {
			flag |= 0x02
		} else if slf.QosLevel == 2 {
			flag |= 0x04
		}
	}

	if slf.Dupe {
		flag |= 0x08
	}

	err := binary.Write(w, binary.BigEndian, flag)
	if err != nil {
		return 0, err
	}

	_, err = encoding.WriteVarint(w, length)
	if err != nil {
		return 0, err
	}

	return int64(2), nil
}

func (slf *FixedHeader) decode(reader io.Reader) error {
	var FirstByte uint8
	err := binary.Read(reader, binary.BigEndian, &FirstByte)
	if err != nil {
		return err
	}

	mt := FirstByte >> 4
	flag := FirstByte & 0x0f

	length, _ := encoding.ReadVarint(reader)

	slf.Type = encoding.PType(mt)
	slf.Dupe = ((flag & 0x08) > 0)

	if (flag & 0x01) > 0 {
		slf.Retain = 1
	}

	if (flag & 0x04) > 0 {
		slf.QosLevel = 2
	} else if (flag & 0x02) > 0 {
		slf.QosLevel = 1
	}
	slf.RemainingLength = length
	return nil
}

//String Retuns string
func (slf *FixedHeader) String() string {
	b, _ := json.Marshal(slf)
	return string(b)
}
