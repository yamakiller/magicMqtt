package message

import (
	"encoding/json"
	"io"
)

//Pingresp message
type Pingresp struct {
	FixedHeader
}

//WriteTo Pingresp message write to io
func (slf Pingresp) WriteTo(w io.Writer) (int64, error) {
	var fsize = 0
	size, err := slf.FixedHeader.writeTo(fsize, w)
	if err != nil {
		return 0, err
	}

	return int64(fsize) + size, nil
}

//String Returns Pingresp object of string
func (slf *Pingresp) String() string {
	b, _ := json.Marshal(slf)
	return string(b)
}
