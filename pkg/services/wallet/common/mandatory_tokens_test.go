package common

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	types "github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"
)

func TestMandatoryTokens(t *testing.T) {
	keys := MandatoryTokens()
	require.NotEmpty(t, keys)

	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		_, dup := seen[key]
		require.False(t, dup, "duplicate mandatory token key %s", key)
		seen[key] = struct{}{}
		require.True(t, IsMandatoryToken(key))
		chainID, address, ok := types.ChainAndAddressFromTokenKey(key)
		require.True(t, ok)
		require.True(t, IsMandatoryTokenAddress(chainID, address))
	}
	require.False(t, IsMandatoryTokenAddress(EthereumMainnet, common.HexToAddress("0x0000000000000000000000000000000000000001")))
	require.False(t, IsMandatoryToken(types.TokenKey(EthereumMainnet, common.HexToAddress("0x0000000000000000000000000000000000000001"))))
	require.False(t, IsMandatoryToken("not-a-token-key"))

	// Stable across calls: the data is built once.
	require.Equal(t, keys, MandatoryTokens())
}

func TestMandatoryTokensByChainID(t *testing.T) {
	total := 0
	for _, chainID := range []uint64{EthereumMainnet, BSCMainnet, OptimismMainnet, ArbitrumMainnet, BaseMainnet, LineaMainnet} {
		perChain := MandatoryTokensByChainID(chainID)
		for _, key := range perChain {
			gotChain, _, ok := types.ChainAndAddressFromTokenKey(key)
			require.True(t, ok)
			require.Equal(t, chainID, gotChain)
			require.True(t, IsMandatoryToken(key))
		}
		total += len(perChain)
	}
	require.LessOrEqual(t, total, len(MandatoryTokens()))
	require.Empty(t, MandatoryTokensByChainID(0))
}
