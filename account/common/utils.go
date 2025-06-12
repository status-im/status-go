package common

import (
	"github.com/status-im/status-go/account/keystore/geth"
	"github.com/status-im/status-go/eth-node/types"
)

// has0xPrefix validates str begins with '0x' or '0X'.
func has0xPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}

func IsHexAddress(s string) bool {
	if has0xPrefix(s) {
		s = s[2:]
	}
	return len(s) == 2*AddressBytesLength && isHex(s)
}

// isHexCharacter returns bool of c being a valid hexadecimal.
func isHexCharacter(c byte) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}

// isHex validates whether each byte is valid hexadecimal string.
func isHex(str string) bool {
	if len(str)%2 != 0 {
		return false
	}
	for _, c := range []byte(str) {
		if !isHexCharacter(c) {
			return false
		}
	}
	return true
}

// TODO: this function is exposed from here just to be used by the tests that were using it before.
// It will be removed in the future, once we update the tests.
// The rest of the code should use the keystore package, via KeyStore interface.
func DecryptKey(keyjson []byte, auth string) (*types.Key, error) {
	return geth.DecryptKey(keyjson, auth)
}
