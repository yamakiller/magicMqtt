package message

import (
	"errors"
	"fmt"
	"io"

	"github.com/yamakiller/magicMqtt/encoding"
)

//SpawnConnectMessage 创建Connect消息
func SpawnConnectMessage() *Connect {
	message := &Connect{
		FixedHeader: FixedHeader{
			Type:     encoding.PTypeConnect,
			Dupe:     false,
			QosLevel: 0,
		},
		Magic:   []byte("MQTT"),
		Version: 4,
	}
	return message
}

//SpawnSubscribeMessage 创建Subscribe消息
func SpawnSubscribeMessage() *Subscribe {
	message := &Subscribe{
		FixedHeader: FixedHeader{
			Type:     encoding.PTypeSubscribe,
			Dupe:     false,
			QosLevel: 1, // Must be 1
		},
	}
	return message
}

//SpawnUnsubscribeMessage 创建Unsubscribe消息
func SpawnUnsubscribeMessage() *Unsubscribe {
	message := &Unsubscribe{
		FixedHeader: FixedHeader{
			Type:     encoding.PTypeUnsubscribe,
			Dupe:     false,
			QosLevel: 1, // Must be 1
		},
	}
	return message
}

//SpawnSubackMessage 创建Suback消息
func SpawnSubackMessage() *Suback {
	message := &Suback{
		FixedHeader: FixedHeader{
			Type:     encoding.PTypeSuback,
			Dupe:     false,
			QosLevel: 0,
		},
	}

	return message
}

//SpawnUnsubackMessage 创建Unsuback消息
func SpawnUnsubackMessage() *Unsuback {
	message := &Unsuback{
		FixedHeader: FixedHeader{
			Type:     encoding.PTypeUnsuback,
			Dupe:     false,
			QosLevel: 0,
		},
	}
	return message
}

//SpawnPublishMessage 创建Publish消息
func SpawnPublishMessage() *Publish {
	message := &Publish{
		FixedHeader: FixedHeader{
			Type:     encoding.PTypePublish,
			Dupe:     false,
			QosLevel: 0,
		},
	}
	return message
}

//SpawnPubackMessage 创建Puback消息
func SpawnPubackMessage() *Puback {
	message := &Puback{
		FixedHeader: FixedHeader{
			Type:     encoding.PTypePuback,
			Dupe:     false,
			QosLevel: 0,
		},
	}

	return message
}

//SpawnPubrecMessage 创建Pubrec消息
func SpawnPubrecMessage() *Pubrec {
	message := &Pubrec{
		FixedHeader: FixedHeader{
			Type:     encoding.PTypePubrec,
			Dupe:     false,
			QosLevel: 0,
		},
	}

	return message
}

//SpawnPubrelMessage 创建Pubrel消息
func SpawnPubrelMessage() *Pubrel {
	message := &Pubrel{
		FixedHeader: FixedHeader{
			Type:     encoding.PTypePubrel,
			Dupe:     false,
			QosLevel: 1, // Must be 1
		},
	}

	return message
}

//SpawnPubcompMessage 创建Pubcomp消息
func SpawnPubcompMessage() *Pubcomp {
	message := &Pubcomp{
		FixedHeader: FixedHeader{
			Type:     encoding.PTypePubcomp,
			Dupe:     false,
			QosLevel: 0,
		},
	}

	return message
}

//SpawnPingreqMessage 创建Ping Request消息
func SpawnPingreqMessage() *Pingreq {
	message := &Pingreq{
		FixedHeader: FixedHeader{
			Type: encoding.PTypePingreq,
		},
	}
	return message
}

//SpawnPingrespMessage 创建Ping Response消息
func SpawnPingrespMessage() *Pingresp {
	message := &Pingresp{
		FixedHeader: FixedHeader{
			Type: encoding.PTypePingresp,
		},
	}
	return message
}

//SpawnDisconnectMessage 创建Disconnect消息
func SpawnDisconnectMessage() *Disconnect {
	message := &Disconnect{
		FixedHeader: FixedHeader{
			Type: encoding.PTypeDisconnect,
		},
	}
	return message
}

//SpawnConnackMessage 创建Connack消息
func SpawnConnackMessage() *Connack {
	message := &Connack{
		FixedHeader: FixedHeader{
			Type:     encoding.PTypeConnack,
			QosLevel: 0,
		},
	}
	return message
}

//Parse 解析接受到的消息
func Parse(reader io.Reader, maxLen int) (Message, error) {
	var message Message
	var err error
	header := FixedHeader{}

	err = header.decode(reader)
	if err != nil {
		return nil, err
	}

	if maxLen > 0 && header.RemainingLength > maxLen {
		return nil, fmt.Errorf("Payload exceedes limit. %d bytes", header.RemainingLength)
	}

	switch header.GetType() {
	case encoding.PTypeConnect:
		mm := &Connect{
			FixedHeader: header,
		}
		err = mm.decode(reader)
		message = mm
	case encoding.PTypeConnack:
		mm := &Connack{
			FixedHeader: header,
		}
		err = mm.decode(reader)
		message = mm
	case encoding.PTypePublish:
		mm := &Publish{
			FixedHeader: header,
		}
		err = mm.decode(reader)
		message = mm
	case encoding.PTypeDisconnect:
		mm := &Disconnect{
			FixedHeader: header,
		}
		message = mm
	case encoding.PTypeSubscribe:
		mm := &Subscribe{
			FixedHeader: header,
		}
		err = mm.decode(reader)
		message = mm
	case encoding.PTypeSuback:
		mm := &Suback{
			FixedHeader: header,
		}
		err = mm.decode(reader)
		message = mm
	case encoding.PTypeUnsubscribe:
		mm := &Unsubscribe{
			FixedHeader: header,
		}
		err = mm.decode(reader)
		message = mm
	case encoding.PTypeUnsuback:
		mm := &Unsuback{
			FixedHeader: header,
		}
		err = mm.decode(reader)
		message = mm
	case encoding.PTypePingresp:
		mm := &Pingresp{
			FixedHeader: header,
		}
		message = mm
	case encoding.PTypePingreq:
		mm := &Pingreq{
			FixedHeader: header,
		}
		message = mm
	case encoding.PTypePuback:
		mm := &Puback{
			FixedHeader: header,
		}
		err = mm.decode(reader)
		message = mm
	case encoding.PTypePubrec:
		mm := &Pubrec{
			FixedHeader: header,
		}
		err = mm.decode(reader)
		message = mm
	case encoding.PTypePubrel:
		mm := &Pubrel{
			FixedHeader: header,
		}
		err = mm.decode(reader)
		message = mm
	case encoding.PTypePubcomp:
		mm := &Pubcomp{
			FixedHeader: header,
		}
		err = mm.decode(reader)
		message = mm
	default:
		return nil, fmt.Errorf("Not supported: %d", header.GetType())
	}
	if err != nil {
		return nil, err
	}

	return message, nil
}

//WriteMessageTo 输出消息
func WriteMessageTo(message Message, w io.Writer) (int64, error) {
	var written int64
	var e error

	switch message.GetType() {
	case encoding.PTypeConnect:
		m := message.(*Connect)
		written, e = m.WriteTo(w)
	case encoding.PTypeConnack:
		m := message.(*Connack)
		written, e = m.WriteTo(w)
	case encoding.PTypePublish:
		m := message.(*Publish)
		written, e = m.WriteTo(w)
	case encoding.PTypeSubscribe:
		m := message.(*Subscribe)
		written, e = m.WriteTo(w)
	case encoding.PTypeSuback:
		m := message.(*Suback)
		written, e = m.WriteTo(w)
	case encoding.PTypeUnsubscribe:
		m := message.(*Unsubscribe)
		written, e = m.WriteTo(w)
	case encoding.PTypeDisconnect:
		m := message.(*Disconnect)
		written, e = m.WriteTo(w)
	case encoding.PTypeUnsuback:
		m := message.(*Unsuback)
		written, e = m.WriteTo(w)
	case encoding.PTypePuback:
		m := message.(*Puback)
		written, e = m.WriteTo(w)
	case encoding.PTypePubrec:
		m := message.(*Pubrec)
		written, e = m.WriteTo(w)
	case encoding.PTypePubrel:
		m := message.(*Pubrel)
		written, e = m.WriteTo(w)
	case encoding.PTypePubcomp:
		m := message.(*Pubcomp)
		written, e = m.WriteTo(w)
	case encoding.PTypePingreq:
		m := message.(*Pingreq)
		written, e = m.WriteTo(w)
	case encoding.PTypePingresp:
		m := message.(*Pingresp)
		written, e = m.WriteTo(w)
	default:
		return 0, errors.New("Not supported message")
	}

	return written, e
}
