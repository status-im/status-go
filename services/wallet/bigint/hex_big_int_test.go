package bigint

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarshalUnmarshal(t *testing.T) {
	inputString := "0x09abc5177d51c36ef4c6a36197d023b60d8fec0100000000000001000000000a"
	inputInt := new(big.Int)
	inputInt.SetString(inputString[2:], 16)

	inputBytes, err := json.Marshal(inputString)

	require.NoError(t, err)

	u := new(HexBigInt)
	err = u.UnmarshalJSON(inputBytes)

	require.NoError(t, err)
	require.Equal(t, inputInt, u.Int)
}
