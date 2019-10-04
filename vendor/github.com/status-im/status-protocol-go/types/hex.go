// Code extracted from vendor/github.com/ethereum/go-ethereum/common/hexutil/hexutil.go

package statusproto

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

// MarshalText implements encoding.TextMarshaler
func (b HexBytes) MarshalText() ([]byte, error) {
	result := make([]byte, len(b)*2+2)
	copy(result, `0x`)
	hex.Encode(result[2:], b)
	return result, nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (b *HexBytes) UnmarshalJSON(input []byte) error {
	if !isString(input) {
		return errNonString(bytesT)
	}
	return wrapTypeError(b.UnmarshalText(input[1:len(input)-1]), bytesT)
}
