package networkhelper

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/services/wallet/common"
)

func TestChainIDToChainAndNetwork(t *testing.T) {
	tests := []struct {
		name            string
		chainID         uint64
		expectedChain   string
		expectedNetwork string
		expectError     bool
	}{
		{
			name:            "Ethereum Mainnet",
			chainID:         common.EthereumMainnet,
			expectedChain:   "ethereum",
			expectedNetwork: "mainnet",
			expectError:     false,
		},
		{
			name:            "Ethereum Sepolia",
			chainID:         common.EthereumSepolia,
			expectedChain:   "ethereum",
			expectedNetwork: "sepolia",
			expectError:     false,
		},
		{
			name:            "Optimism Mainnet",
			chainID:         common.OptimismMainnet,
			expectedChain:   "optimism",
			expectedNetwork: "mainnet",
			expectError:     false,
		},
		{
			name:            "Optimism Sepolia",
			chainID:         common.OptimismSepolia,
			expectedChain:   "optimism",
			expectedNetwork: "sepolia",
			expectError:     false,
		},
		{
			name:            "Arbitrum Mainnet",
			chainID:         common.ArbitrumMainnet,
			expectedChain:   "arbitrum",
			expectedNetwork: "mainnet",
			expectError:     false,
		},
		{
			name:            "Arbitrum Sepolia",
			chainID:         common.ArbitrumSepolia,
			expectedChain:   "arbitrum",
			expectedNetwork: "sepolia",
			expectError:     false,
		},
		{
			name:            "Base Mainnet",
			chainID:         common.BaseMainnet,
			expectedChain:   "base",
			expectedNetwork: "mainnet",
			expectError:     false,
		},
		{
			name:            "Base Sepolia",
			chainID:         common.BaseSepolia,
			expectedChain:   "base",
			expectedNetwork: "sepolia",
			expectError:     false,
		},
		{
			name:            "Linea Mainnet",
			chainID:         common.LineaMainnet,
			expectedChain:   "linea",
			expectedNetwork: "mainnet",
			expectError:     false,
		},
		{
			name:            "Linea Sepolia",
			chainID:         common.LineaSepolia,
			expectedChain:   "linea",
			expectedNetwork: "sepolia",
			expectError:     false,
		},
		{
			name:            "Status Network Sepolia",
			chainID:         common.StatusNetworkSepolia,
			expectedChain:   "status",
			expectedNetwork: "sepolia",
			expectError:     false,
		},
		{
			name:            "BSC Mainnet",
			chainID:         common.BSCMainnet,
			expectedChain:   "bsc",
			expectedNetwork: "mainnet",
			expectError:     false,
		},
		{
			name:            "BSC Testnet",
			chainID:         common.BSCTestnet,
			expectedChain:   "bsc",
			expectedNetwork: "testnet",
			expectError:     false,
		},
		{
			name:            "Unsupported chain ID",
			chainID:         999999,
			expectedChain:   "",
			expectedNetwork: "",
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain, network, err := ChainIDToChainAndNetwork(tt.chainID)

			if tt.expectError {
				require.Error(t, err)
				require.Empty(t, chain)
				require.Empty(t, network)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedChain, chain)
				require.Equal(t, tt.expectedNetwork, network)
			}
		})
	}
}
