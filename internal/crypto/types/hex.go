// Code extracted from vendor/github.com/ethereum/go-ethereum/common/hexutil/hexutil.go

package types

import (
	"encoding/hex"
	"reflect"
)

var (
	bytesT = reflect.TypeOf(HexBytes(nil))
)

// HexBytes marshals/unmarshals as a JSON string with 0x prefix.
// The empty slice marshals as "0x".
type HexBytes []byte

func (b HexBytes) Bytes() []byte {
	result := make([]byte, len(b)*2+2)
	copy(result, `0x`)
	hex.Encode(result[2:], b)
	return result
}

// String returns the hex encoding of b.
func (b HexBytes) String() string {
	return EncodeHex(b)
}

// MarshalText implements encoding.TextMarshaler
func (b HexBytes) MarshalText() ([]byte, error) {
	return b.Bytes(), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (b *HexBytes) UnmarshalJSON(input []byte) error {
	if !isString(input) {
		return errNonString(bytesT)
	}
	return wrapTypeError(b.UnmarshalText(input[1:len(input)-1]), bytesT)
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (b *HexBytes) UnmarshalText(input []byte) error {
	raw, err := checkText(input, true)
	if err != nil {
		return err
	}
	dec := make([]byte, len(raw)/2)
	if _, err = hex.Decode(dec, raw); err != nil {
		err = mapError(err)
	} else {
		*b = dec
	}
	return err
}
