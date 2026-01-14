package tokentypes

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestParseCollectibleKey(t *testing.T) {
	collectibleKey := "RANDOM"
	_, _, _, success := ParseCollectibleKey(collectibleKey)
	require.False(t, success)

	chainID := uint64(1)
	contractAddress := common.HexToAddress("0xfC43ac5f309499385e91e059862bDb0Bfa2Cd16c")
	collectibleTokenID := big.NewInt(123)
	collectibleKey = fmt.Sprintf("%d%s%s%s%s", chainID, tokenKeySeparator, contractAddress.Hex(), tokenKeySeparator, collectibleTokenID.String())
	parsedChainID, parsedContractAddress, parsedCollectibleTokenID, success := ParseCollectibleKey(collectibleKey)
	require.True(t, success)
	require.Equal(t, chainID, parsedChainID)
	require.Equal(t, contractAddress, parsedContractAddress)
	require.Equal(t, collectibleTokenID.String(), parsedCollectibleTokenID.String())
}
