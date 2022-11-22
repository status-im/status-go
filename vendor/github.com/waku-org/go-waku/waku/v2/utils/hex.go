package utils

import (
	"encoding/hex"
	"strings"
)

func DecodeHexString(input string) ([]byte, error) {
	input = strings.TrimPrefix(input, "0x")
	return hex.DecodeString(input)
}
