package mercuryo

import walletCommon "github.com/status-im/status-go/services/wallet/common"

func NetworkToCommonChainID(network string) uint64 {
	switch network {
	case "ETHEREUM":
		return walletCommon.EthereumMainnet
	case "OPTIMISM":
		return walletCommon.OptimismMainnet
	case "ARBITRUM":
		return walletCommon.ArbitrumMainnet
	case "BASE":
		return walletCommon.BaseMainnet
	case "BINANCESMARTCHAIN":
		return walletCommon.BSCMainnet
	}
	return walletCommon.UnknownChainID
}

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
