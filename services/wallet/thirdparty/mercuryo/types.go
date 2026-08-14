package mercuryo

import walletCommon "github.com/status-im/status-go/services/wallet/common"

func CommonChainIDToNetwork(chainID uint64) string {
	switch chainID {
	case walletCommon.EthereumMainnet:
		return "ETHEREUM"
	case walletCommon.ArbitrumMainnet:
		return "ARBITRUM"
	case walletCommon.OptimismMainnet:
		return "OPTIMISM"
	case walletCommon.BaseMainnet:
		return "BASE"
	case walletCommon.BSCMainnet:
		return "BINANCESMARTCHAIN"
	default:
		return ""
	}
}
