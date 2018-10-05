package apdu

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	// SwOK is returned from smartcards as a positive response code.
	SwOK = 0x9000
)

// ErrBadResponse defines an error conaining the returned Sw code and a description message.
type ErrBadResponse struct {
	sw      uint16
	message string
}

// NewErrBadResponse returns a ErrBadResponse with the specified sw and message values.
func NewErrBadResponse(sw uint16, message string) *ErrBadResponse {
	return &ErrBadResponse{
		sw:      sw,
		message: message,
	}
}

// Error implements the error interface.
func (e *ErrBadResponse) Error() string {
	return fmt.Sprintf("bad response %x: %s", e.sw, e.message)
}

// Response represents a struct containing the smartcard response fields.
type Response struct {
	Data []byte
	Sw1  uint8
	Sw2  uint8
	Sw   uint16
}

// ErrBadRawResponse is an error returned by ParseResponse in case the response data is not long enough.
var ErrBadRawResponse = errors.New("response data must be at least 2 bytes")

// ParseResponse parses a raw response and return a Response.
func ParseResponse(data []byte) (*Response, error) {
	r := &Response{}
	return r, r.deserialize(data)
}

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

// IsOK returns true if the response Sw code is 0x9000.
func (r *Response) IsOK() bool {
	return r.Sw == SwOK
}
