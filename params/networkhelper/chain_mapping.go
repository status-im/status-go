package networkhelper

import (
	"fmt"

	"github.com/status-im/status-go/services/wallet/common"
)

// ChainIDToChainAndNetwork converts a chain ID to its corresponding chain and network string identifiers.
// These strings are used in proxy URLs and other network-related configurations.
// Returns an error if the chain ID is not supported.
func ChainIDToChainAndNetwork(chainID uint64) (chain string, network string, err error) {
	switch chainID {
	case common.EthereumMainnet:
		return "ethereum", "mainnet", nil
	case common.EthereumSepolia:
		return "ethereum", "sepolia", nil
	case common.OptimismMainnet:
		return "optimism", "mainnet", nil
	case common.OptimismSepolia:
		return "optimism", "sepolia", nil
	case common.ArbitrumMainnet:
		return "arbitrum", "mainnet", nil
	case common.ArbitrumSepolia:
		return "arbitrum", "sepolia", nil
	case common.BaseMainnet:
		return "base", "mainnet", nil
	case common.BaseSepolia:
		return "base", "sepolia", nil
	case common.LineaMainnet:
		return "linea", "mainnet", nil
	case common.LineaSepolia:
		return "linea", "sepolia", nil
	case common.StatusNetworkSepolia:
		return "status", "sepolia", nil
	case common.BSCMainnet:
		return "bsc", "mainnet", nil
	case common.BSCTestnet:
		return "bsc", "testnet", nil
	default:
		return "", "", fmt.Errorf("unsupported chain ID: %d", chainID)
	}
}
