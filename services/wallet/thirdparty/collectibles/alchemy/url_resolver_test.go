package alchemy

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/pkg/security"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

func TestDirectURLResolver_GetNFTBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		chainID  walletCommon.ChainID
		apiKey   string
		expected string
	}{
		{
			name:     "Ethereum mainnet",
			chainID:  walletCommon.ChainID(walletCommon.EthereumMainnet),
			apiKey:   "my-api-key",
			expected: "https://eth-mainnet.g.alchemy.com/nft/v3/my-api-key",
		},
		{
			name:     "Ethereum Sepolia",
			chainID:  walletCommon.ChainID(walletCommon.EthereumSepolia),
			apiKey:   "key",
			expected: "https://eth-sepolia.g.alchemy.com/nft/v3/key",
		},
		{
			name:     "Base mainnet",
			chainID:  walletCommon.ChainID(walletCommon.BaseMainnet),
			apiKey:   "abc",
			expected: "https://base-mainnet.g.alchemy.com/nft/v3/abc",
		},
		{
			name:     "Arbitrum Sepolia",
			chainID:  walletCommon.ChainID(walletCommon.ArbitrumSepolia),
			apiKey:   "xyz",
			expected: "https://arb-sepolia.g.alchemy.com/nft/v3/xyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &directURLResolver{apiKey: security.NewSensitiveString(tt.apiKey)}
			result, err := resolver.GetNFTBaseURL(tt.chainID)
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDirectURLResolver_EmptyAPIKey(t *testing.T) {
	resolver := &directURLResolver{apiKey: security.SensitiveString{}}
	result, err := resolver.GetNFTBaseURL(walletCommon.ChainID(walletCommon.EthereumMainnet))
	require.NoError(t, err)
	require.Equal(t, "https://eth-mainnet.g.alchemy.com/nft/v3/demo", result)
}

func TestDirectURLResolver_UnsupportedChain(t *testing.T) {
	resolver := &directURLResolver{apiKey: security.NewSensitiveString("key")}
	_, err := resolver.GetNFTBaseURL(walletCommon.ChainID(999999))
	require.Error(t, err)
}

func TestProxyURLResolver_GetNFTBaseURL(t *testing.T) {
	resolver := &proxyURLResolver{customURL: "", stageName: "test"}
	result, err := resolver.GetNFTBaseURL(walletCommon.ChainID(walletCommon.EthereumMainnet))
	require.NoError(t, err)
	require.Equal(t, "https://test.nft.status.im/ethereum/mainnet/nft/v3", result)
}

func TestProxyURLResolver_CustomURL(t *testing.T) {
	resolver := &proxyURLResolver{customURL: "https://custom.example.com", stageName: "test"}
	result, err := resolver.GetNFTBaseURL(walletCommon.ChainID(walletCommon.EthereumMainnet))
	require.NoError(t, err)
	require.Equal(t, "https://custom.example.com/ethereum/mainnet/nft/v3", result)
}

func TestProxyURLResolver_UnsupportedChain(t *testing.T) {
	resolver := &proxyURLResolver{customURL: "", stageName: "test"}
	_, err := resolver.GetNFTBaseURL(walletCommon.ChainID(999999))
	require.Error(t, err)
}

func TestDirectURLResolver_IsChainSupported(t *testing.T) {
	resolver := &directURLResolver{apiKey: security.NewSensitiveString("key")}
	require.True(t, resolver.IsChainSupported(walletCommon.ChainID(walletCommon.EthereumMainnet)))
	require.True(t, resolver.IsChainSupported(walletCommon.ChainID(walletCommon.BaseSepolia)))
	require.False(t, resolver.IsChainSupported(walletCommon.ChainID(walletCommon.BSCTestnet)))
	require.False(t, resolver.IsChainSupported(walletCommon.ChainID(walletCommon.EthereumHoodi)))
	require.False(t, resolver.IsChainSupported(walletCommon.ChainID(walletCommon.InkMainnet)))
	require.False(t, resolver.IsChainSupported(walletCommon.ChainID(walletCommon.InkSepolia)))
	require.False(t, resolver.IsChainSupported(walletCommon.ChainID(walletCommon.KatanaMainnet)))
	require.False(t, resolver.IsChainSupported(walletCommon.ChainID(walletCommon.KatanaBokuto)))
	require.False(t, resolver.IsChainSupported(walletCommon.ChainID(999999)))
}

func TestProxyURLResolver_IsChainSupported(t *testing.T) {
	resolver := &proxyURLResolver{customURL: "", stageName: "test"}
	require.True(t, resolver.IsChainSupported(walletCommon.ChainID(walletCommon.EthereumMainnet)))
	require.True(t, resolver.IsChainSupported(walletCommon.ChainID(walletCommon.ArbitrumSepolia)))
	require.False(t, resolver.IsChainSupported(walletCommon.ChainID(walletCommon.BSCTestnet)))
	require.False(t, resolver.IsChainSupported(walletCommon.ChainID(walletCommon.EthereumHoodi)))
	require.False(t, resolver.IsChainSupported(walletCommon.ChainID(walletCommon.InkMainnet)))
	require.False(t, resolver.IsChainSupported(walletCommon.ChainID(walletCommon.InkSepolia)))
	require.False(t, resolver.IsChainSupported(walletCommon.ChainID(walletCommon.KatanaMainnet)))
	require.False(t, resolver.IsChainSupported(walletCommon.ChainID(walletCommon.KatanaBokuto)))
	require.False(t, resolver.IsChainSupported(walletCommon.ChainID(999999)))
}
