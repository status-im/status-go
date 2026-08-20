package bigint

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
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

	// Check if string is long enough and has 0x prefix
	if len(hexStr) < 2 {
		return fmt.Errorf("hex string too short: %s", hexStr)
	}

	// Check for 0x or 0X prefix and remove it
	if !strings.HasPrefix(hexStr, "0x") && !strings.HasPrefix(hexStr, "0X") {
		return fmt.Errorf("hex string must start with 0x prefix: %s", hexStr)
	}

	hexValue := hexStr[2:]

	// Handle empty hex value (just "0x")
	if len(hexValue) == 0 {
		b.Int = big.NewInt(0)
		return nil
	}

	var ok bool
	b.Int, ok = new(big.Int).SetString(hexValue, 16)
	if !ok {
		return fmt.Errorf("not a valid big integer: %s", hexStr)
	}
	return nil
}

func (b *VarHexBigInt) MarshalJSON() ([]byte, error) {
	return []byte("\"0x" + b.Text(16) + "\""), nil
}
