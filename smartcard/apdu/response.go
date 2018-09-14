package apdu

import (
	"bytes"
	"encoding/binary"
	"errors"
)

type Response struct {
	Data []byte
	Sw1  uint8
	Sw2  uint8
	Sw   uint16
}

var ErrBadRawResponse = errors.New("response data must be at least 2 bytes")

func (r *Response) deserialize(data []byte) error {
	if len(data) < 2 {
		return ErrBadRawResponse
	}

	r.Data = make([]byte, len(data)-2)
	buf := bytes.NewReader(data)

	if err := binary.Read(buf, binary.BigEndian, &r.Data); err != nil {
		return err
	}

	if err := binary.Read(buf, binary.BigEndian, &r.Sw1); err != nil {
		return err
	}

	if err := binary.Read(buf, binary.BigEndian, &r.Sw2); err != nil {
		return err
	}

	r.Sw = (uint16(r.Sw1) << 8) | uint16(r.Sw2)

	return nil
}

func ParseResponse(data []byte) (*Response, error) {
	r := &Response{}
	return r, r.deserialize(data)
}
