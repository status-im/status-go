package collectibles

import (
	"testing"

	"github.com/stretchr/testify/require"

	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
)

func TestUnsupportedCollectibleChains_sorted(t *testing.T) {
	for i := 1; i < len(UnsupportedCollectibleChains); i++ {
		require.Less(t, UnsupportedCollectibleChains[i-1], UnsupportedCollectibleChains[i])
	}
}

func TestIsUnsupportedCollectibleChain(t *testing.T) {
	require.True(t, IsUnsupportedCollectibleChain(walletCommon.InkMainnet))
	require.True(t, IsUnsupportedCollectibleChain(walletCommon.BSCMainnet))
	require.True(t, IsUnsupportedCollectibleChain(walletCommon.RobinhoodMainnet))
	require.True(t, IsUnsupportedCollectibleChain(walletCommon.RobinhoodTestnet))
	require.False(t, IsUnsupportedCollectibleChain(walletCommon.EthereumMainnet))
}
