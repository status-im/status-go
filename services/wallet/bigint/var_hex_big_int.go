package bigint

import (
	"encoding/json"
	"fmt"
	"math/big"
)

// Unmarshals hex string as a variable-length hex string with 0x prefix and possible leading zeros
type VarHexBigInt struct {
	*big.Int
}

func (b *VarHexBigInt) UnmarshalJSON(input []byte) error {
	var hexStr string
	if err := json.Unmarshal(input, &hexStr); err != nil {
		return err
	}

	var ok bool
	b.Int, ok = new(big.Int).SetString(hexStr[2:], 16)
	if !ok {
		return fmt.Errorf("not a valid big integer: %s", hexStr)
	}
	return nil
}
