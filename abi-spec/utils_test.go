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
