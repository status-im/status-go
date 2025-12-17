package abispec

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"unicode"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/logutils"
)

var hexPrefixPattern = regexp.MustCompile("(?i)^0x")

var addressBasicPattern = regexp.MustCompile("(?i)^(0x)?[0-9a-f]{40}$")
var addressLowerCasePattern = regexp.MustCompile("^(0x|0X)?[0-9a-f]{40}$")
var addressUpperCasePattern = regexp.MustCompile("^(0x|0X)?[0-9A-F]{40}$")

var hexStrictPattern = regexp.MustCompile(`(?i)^((-)?0x[0-9a-f]+|(0x))$`)

func Sha3(str string) string {
	var bytes []byte
	var err error
	if hexStrictPattern.MatchString(str) {
		bytes, err = hex.DecodeString(str[2:])
		if err != nil {
			logutils.ZapLogger().Error("failed to decode hex string when sha3", zap.Error(err))
			return ""
		}
	} else {
		bytes = []byte(str)
	}
	bytes = crypto.Keccak256(bytes)
	return common.Bytes2Hex(bytes)
}

func removeHexPrefix(str string) string {
	found := hexPrefixPattern.FindString(str)
	return strings.Replace(str, found, "", 1)
}

// implementation referenced from https://github.com/ChainSafe/web3.js/blob/edcd215bf657a4bba62fabaafd08e6e70040976e/packages/web3-utils/src/utils.js#L107
func CheckAddressChecksum(address string) (bool, error) {
	address = removeHexPrefix(address)
	addressHash := Sha3(strings.ToLower(address))

	n := big.NewInt(0)
	for i := 0; i < 40; i++ {
		// the nth letter should be uppercase if the nth digit of casemap is 1
		n, success := n.SetString(addressHash[i:i+1], 16)
		if !success {
			return false, fmt.Errorf("failed to convert hex value '%s' to int", addressHash[i:i+1])
		}
		v := n.Int64()

		if (v > 7 && uint8(unicode.ToUpper(rune(address[i]))) != address[i]) || (v <= 7 && uint8(unicode.ToLower(rune(address[i]))) != address[i]) {
			return false, nil
		}
	}
	return true, nil
}

// implementation referenced from https://github.com/ChainSafe/web3.js/blob/edcd215bf657a4bba62fabaafd08e6e70040976e/packages/web3-utils/src/utils.js#L85
func IsAddress(address string) (bool, error) {
	// check if it has the basic requirements of an address
	if !addressBasicPattern.MatchString(address) {
		return false, nil
	} else if addressLowerCasePattern.MatchString(address) || addressUpperCasePattern.MatchString(address) {
		return true, nil
	} else {
		return CheckAddressChecksum(address)
	}
}
