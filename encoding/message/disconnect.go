package message

import (
	"encoding/json"
	"io"
)

//Disconnect message
type Disconnect struct {
	FixedHeader
}

//WriteTo Disconnect message to io
func (slf Disconnect) WriteTo(w io.Writer) (int64, error) {
	var fsize = 0
	size, err := slf.FixedHeader.writeTo(fsize, w)
	if err != nil {
		return 0, err
	}

	return int64(fsize) + size, nil
}

//String Returns Disconnect messge object of string
func (slf *Disconnect) String() string {
	b, _ := json.Marshal(slf)
	return string(b)
}
