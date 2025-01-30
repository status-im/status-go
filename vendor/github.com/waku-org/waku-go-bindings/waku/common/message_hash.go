package common

import (
	"encoding/hex"
	"errors"
	"fmt"
)

// MessageHash represents an unique identifier for a message within a pubsub topic
type MessageHash string

func ToMessageHash(val string) (MessageHash, error) {
	if len(val) == 0 {
		return "", errors.New("empty string not allowed")
	}

	if len(val) < 2 || val[:2] != "0x" {
		return "", errors.New("string must start with 0x")
	}

	// Remove "0x" prefix for hex decoding
	hexStr := val[2:]

	// Verify the remaining string is valid hex
	_, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", fmt.Errorf("invalid hex string: %v", err)
	}

	return MessageHash(val), nil
}

func (h MessageHash) String() string {
	return string(h)
}

func (h MessageHash) Bytes() ([]byte, error) {
	return hex.DecodeString(string(h))
}
