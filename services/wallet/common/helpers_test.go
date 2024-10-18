package common

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"
)

func TestPackApprovalInputData(t *testing.T) {

	expectedData := "095ea7b3000000000000000000000000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa0000000000000000000000000000000000000000000000000000000000000064"

	addr := common.HexToAddress("0xaAaAaAaaAaAaAaaAaAAAAAAAAaaaAaAaAaaAaaAa")
	data, err := PackApprovalInputData(big.NewInt(100), &addr)
	require.NoError(t, err)
	require.Equal(t, expectedData, hex.EncodeToString(data))
}

func TestGetTokenIdFromSymbol(t *testing.T) {

	expectedData := big.NewInt(100)

	data, err := GetTokenIdFromSymbol(expectedData.String())
	require.NoError(t, err)
	require.Equal(t, expectedData, data)
}
