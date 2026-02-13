package alchemy

import (
	"testing"

	"github.com/stretchr/testify/require"

	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

func TestGetNftProxyHost(t *testing.T) {
	tests := []struct {
		name      string
		customURL string
		stageName string
		expected  string
	}{
		{
			name:      "custom URL with trailing slash",
			customURL: "https://custom.example.com/",
			stageName: "",
			expected:  "https://custom.example.com",
		},
		{
			name:      "custom URL without trailing slash",
			customURL: "https://custom.example.com",
			stageName: "",
			expected:  "https://custom.example.com",
		},
		{
			name:      "stage name provided",
			customURL: "",
			stageName: "production",
			expected:  "https://production.nft.status.im",
		},
		{
			name:      "empty stage name defaults to test",
			customURL: "",
			stageName: "",
			expected:  "https://test.nft.status.im",
		},
		{
			name:      "custom URL takes precedence over stage name",
			customURL: "https://override.example.com",
			stageName: "production",
			expected:  "https://override.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetNftProxyHost(tt.customURL, tt.stageName)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestGetNftProxyBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		chainID  walletCommon.ChainID
		expected string
		wantErr  bool
	}{
		{
			name:     "Supported chain - Ethereum mainnet",
			chainID:  walletCommon.ChainID(walletCommon.EthereumMainnet),
			expected: "https://test.nft.status.im/ethereum/mainnet/nft/v3",
		},
		{
			name:     "Supported chain - Arbitrum Sepolia",
			chainID:  walletCommon.ChainID(walletCommon.ArbitrumSepolia),
			expected: "https://test.nft.status.im/arbitrum/sepolia/nft/v3",
		},
		{
			name:    "Unsupported chain ID",
			chainID: walletCommon.ChainID(999999),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetNftProxyBaseURL("", "test", tt.chainID)
			if tt.wantErr {
				require.Error(t, err)
				require.Equal(t, thirdparty.ErrChainIDNotSupported, err)
				require.Empty(t, result)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}
