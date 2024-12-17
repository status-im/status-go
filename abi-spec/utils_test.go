package abispec

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHexToNumber(t *testing.T) {
	//hex number is less than 53 bits, it returns a number
	num := HexToNumber("9")
	bytes, err := json.Marshal(num)
	require.NoError(t, err)
	require.JSONEq(t, `"9"`, string(bytes))

	num = HexToNumber("99999999")
	bytes, err = json.Marshal(num)
	require.NoError(t, err)
	require.JSONEq(t, `"2576980377"`, string(bytes))

	num = HexToNumber("1fffffffffffff")
	bytes, err = json.Marshal(num)
	require.NoError(t, err)
	require.JSONEq(t, `"9007199254740991"`, string(bytes))

	num = HexToNumber("9999999999999999")
	bytes, err = json.Marshal(num)
	require.NoError(t, err)
	require.JSONEq(t, `"11068046444225730969"`, string(bytes))
}

func TestNumberToHex(t *testing.T) {
	require.Equal(t, "20000000000002", NumberToHex("9007199254740994"))
}

func TestSha3(t *testing.T) {
	require.Equal(t, "48bed44d1bcd124a28c27f343a817e5f5243190d3c52bf347daf876de1dbbf77", Sha3("abcd"))
}

func TestHexToUtf8(t *testing.T) {
	str, err := HexToUtf8("0x49206861766520313030e282ac")
	require.NoError(t, err)
	require.Equal(t, "I have 100€", str)

	str, err = HexToUtf8("0xe4bda0e5a5bd")
	require.NoError(t, err)
	require.Equal(t, "你好", str)

	_, err = HexToUtf8("0xfffefd")
	require.ErrorContains(t, err, "invalid UTF-8 detected")
}

func TestUtf8ToHex(t *testing.T) {
	str, err := Utf8ToHex("\xff")
	require.NoError(t, err)
	require.Equal(t, "0xc3bf", str)

	str, err = Utf8ToHex("你好")
	require.NoError(t, err)
	require.Equal(t, "0xe4bda0e5a5bd", str)
}

const (
	address1 = "0x0eD343df16A5327aC13B689072804A2705a26F47"
	address2 = "0x0eD343df16A5327aC13B689072804A2705a26F47a"
	address3 = "0x0eD343df16A5327aC13B689072804A2705a26f47"
)

func TestIsAddress(t *testing.T) {
	valid, err := IsAddress(address1)
	require.NoError(t, err)
	require.True(t, valid)

	valid, err = IsAddress(address2)
	require.NoError(t, err)
	require.False(t, valid)

	valid, err = IsAddress(address3)
	require.NoError(t, err)
	require.False(t, valid)
}

func TestCheckAddressChecksum(t *testing.T) {
	valid, err := CheckAddressChecksum(address1)
	require.NoError(t, err)
	require.True(t, valid)

	valid, err = CheckAddressChecksum(address2)
	require.NoError(t, err)
	require.False(t, valid)

	valid, err = CheckAddressChecksum(address3)
	require.NoError(t, err)
	require.False(t, valid)
}

func TestToChecksumAddress(t *testing.T) {
	checksumAddress, err := ToChecksumAddress(address3)
	require.NoError(t, err)
	require.Equal(t, address1, checksumAddress)

	address := "0x0ed343df16A5327aC13B689072804A2705A26f47"
	checksumAddress, err = ToChecksumAddress(address)
	require.NoError(t, err)
	require.Equal(t, address1, checksumAddress)
}
