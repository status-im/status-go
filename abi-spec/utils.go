package abispec

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"unicode"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/eth-node/crypto"
)

var unicodeZeroPattern = regexp.MustCompile("^(?:\u0000)*")
var hexZeroPattern = regexp.MustCompile("^(?:00)*")
var hexStringPattern = regexp.MustCompile("(?i)^[0-9a-f]+$")
var hexPrefixPattern = regexp.MustCompile("(?i)^0x")

var addressBasicPattern = regexp.MustCompile("(?i)^(0x)?[0-9a-f]{40}$")
var addressLowerCasePattern = regexp.MustCompile("^(0x|0X)?[0-9a-f]{40}$")
var addressUpperCasePattern = regexp.MustCompile("^(0x|0X)?[0-9A-F]{40}$")

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

func Sha3(str string) string {
	bytes := crypto.Keccak256([]byte(str))
	return common.Bytes2Hex(bytes)
}

func reverse(str string) string {
	bytes := []byte(str)
	var out []byte
	for i := len(bytes) - 1; i >= 0; i-- {
		out = append(out, bytes[i])
	}
	return string(out)
}

// remove \u0000 padding from either side
func removeUnicodeZeroPadding(str string) string {
	found := unicodeZeroPattern.FindString(str)
	str = strings.Replace(str, found, "", 1)
	str = reverse(str)
	found = unicodeZeroPattern.FindString(str)
	str = strings.Replace(str, found, "", 1)
	return reverse(str)
}

// remove 00 padding from either side
func removeHexZeroPadding(str string) string {
	found := hexZeroPattern.FindString(str)
	str = strings.Replace(str, found, "", 1)
	str = reverse(str)
	found = hexZeroPattern.FindString(str)
	str = strings.Replace(str, found, "", 1)
	return reverse(str)
}

// implementation referenced from https://github.com/ChainSafe/web3.js/blob/edcd215bf657a4bba62fabaafd08e6e70040976e/packages/web3-utils/src/utils.js#L165
func Utf8ToHex(str string) (string, error) {
	str, err := Utf8encode(str)
	if err != nil {
		return "", err
	}
	str = removeUnicodeZeroPadding(str)

	var hex = ""
	for _, r := range str {
		n := fmt.Sprintf("%x", r)
		if len(n) < 2 {
			hex += "0" + n
		} else {
			hex += n
		}
	}
	return "0x" + hex, nil
}

// implementation referenced from https://github.com/ChainSafe/web3.js/blob/edcd215bf657a4bba62fabaafd08e6e70040976e/packages/web3-utils/src/utils.js#L193
func HexToUtf8(hexString string) (string, error) {
	hexString = removeHexPrefix(hexString)
	if !hexStringPattern.MatchString(hexString) {
		return "", fmt.Errorf("the parameter '%s' must be a valid HEX string", hexString)
	}
	if len(hexString)%2 != 0 {
		return "", fmt.Errorf("the parameter '%s' must have a even number of digits", hexString)
	}

	hexString = removeHexZeroPadding(hexString)

	n := big.NewInt(0)
	var bytes []byte
	for i := 0; i < len(hexString); i += 2 {
		hex := hexString[i : i+2]
		n, success := n.SetString(hex, 16)
		if !success {
			return "", fmt.Errorf("invalid hex value %s", hex)
		}
		r := rune(n.Int64())
		bs := stringFromCharCode(r)
		bytes = appendBytes(bytes, bs)
	}

	bytes, err := Utf8decode(string(bytes))
	if err != nil {
		return "", err
	}
	return string(bytes), nil
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

// implementation referenced from https://github.com/ChainSafe/web3.js/blob/2022b17d52d31ce95559d18d5530d18c83eb4d1c/packages/web3-utils/src/index.js#L283
func ToChecksumAddress(address string) (string, error) {
	if strings.Trim(address, "") == "" {
		return "", nil
	}
	if !addressBasicPattern.MatchString(address) {
		return "", fmt.Errorf("given address '%s' is not a valid Ethereum address", address)
	}

	address = strings.ToLower(address)
	address = removeHexPrefix(address)
	addressHash := Sha3(address)

	var checksumAddress = []rune("0x")
	var n = big.NewInt(0)
	for i := 0; i < len(address); i++ {
		// If ith character is 9 to f then make it uppercase
		n, success := n.SetString(addressHash[i:i+1], 16)
		if !success {
			return "", fmt.Errorf("failed to convert hex value '%s' to int", addressHash[i:i+1])
		}
		if n.Int64() > 7 {
			upper := unicode.ToUpper(rune(address[i]))
			checksumAddress = append(checksumAddress, upper)
		} else {
			checksumAddress = append(checksumAddress, rune(address[i]))
		}
	}
	return string(checksumAddress), nil
}
