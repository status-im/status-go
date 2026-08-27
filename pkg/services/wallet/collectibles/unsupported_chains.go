package collectibles

import (
	"slices"

	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
)

// UnsupportedCollectibleChains are chain IDs where multichain collectibles (NFT proxy/direct) are disabled.
// Sorted ascending for stable JSON from RPC.
var UnsupportedCollectibleChains = []uint64{
	walletCommon.BSCMainnet,
	walletCommon.BSCTestnet,
	walletCommon.RobinhoodMainnet,
	walletCommon.RobinhoodTestnet,
	walletCommon.InkMainnet,
	walletCommon.EthereumHoodi,
	walletCommon.KatanaBokuto,
	walletCommon.KatanaMainnet,
	walletCommon.InkSepolia,
}

// IsUnsupportedCollectibleChain reports whether chainID is intentionally excluded for collectibles.
func IsUnsupportedCollectibleChain(chainID uint64) bool {
	return slices.Contains(UnsupportedCollectibleChains, chainID)
}
