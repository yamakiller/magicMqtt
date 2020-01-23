package message

import (
	"encoding/json"
	"io"
)

//Pingreq Pingreq message
type Pingreq struct {
	FixedHeader
}

//WriteTo Pingreq message to io
func (slf *Pingreq) WriteTo(w io.Writer) (int64, error) {
	var fsize = 0
	size, err := slf.FixedHeader.writeTo(fsize, w)
	if err != nil {
		return 0, err
	}

	return int64(fsize) + size, nil
}

//String Returns Pingreq object of message
func (slf *Pingreq) String() string {
	b, _ := json.Marshal(slf)
	return string(b)
}
