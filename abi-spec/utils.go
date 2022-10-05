package abispec

import (
	"fmt"
	"math/big"
)

func HexToNumber(hex string) string {
	num, success := big.NewInt(0).SetString(hex, 16)
	if success {
		return num.String()
	}
	return ""
}

func NumberToHex(numString string) string {
	num, success := big.NewInt(0).SetString(numString, 0)
	if success {
		return fmt.Sprintf("%x", num)
	}
	return ""
}
