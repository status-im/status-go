package apdu

import (
	"bytes"
	"fmt"
	"io"
)

// ErrTagNotFound is an error returned if a tag is not found in a TLV sequence.
type ErrTagNotFound struct {
	tag uint8
}

// Error implements the error interface
func (e *ErrTagNotFound) Error() string {
	return fmt.Sprintf("tag %x not found", e.tag)
}

// FindTag searches for a tag value within a TLV sequence.
func FindTag(raw []byte, tags ...uint8) ([]byte, error) {
	if len(tags) == 0 {
		return raw, nil
	}

	target := tags[0]
	buf := bytes.NewBuffer(raw)

	var (
		tag    uint8
		length uint8
		err    error
	)

	for {
		tag, err = buf.ReadByte()
		switch {
		case err == io.EOF:
			return []byte{}, &ErrTagNotFound{target}
		case err != nil:
			return nil, err
		}

		length, err = buf.ReadByte()
		if err != nil {
			return nil, err
		}

		data := make([]byte, length)
		if length != 0 {
			_, err = buf.Read(data)
			if err != nil {
				return nil, err
			}
		}

		if tag == target {
			if len(tags) == 1 {
				return data, nil
			}

			return FindTag(data, tags[1:]...)
		}
	}
}
