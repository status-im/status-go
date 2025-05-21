package testutil

import (
	"github.com/stretchr/testify/require"

	api_common "github.com/status-im/status-go/api/common"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/security"
)

// Helper function to create a provider
func CreateProvider(chainID uint64, name string, providerType params.RpcProviderType, enabled bool, url security.SensitiveString) params.RpcProvider {
	return params.RpcProvider{
		ChainID:          chainID,
		Name:             name,
		URL:              url,
		EnableRPSLimiter: true,
		Type:             providerType,
		Enabled:          enabled,
		AuthType:         params.BasicAuth,
		AuthLogin:        security.NewSensitiveString("user1"),
		AuthPassword:     security.NewSensitiveString("password1"),
		AuthToken:        security.NewSensitiveString(""),
	}
}

// Helper function to create a network
func CreateNetwork(chainID uint64, chainName string, providers []params.RpcProvider) *params.Network {
	return &params.Network{
		ChainID:                chainID,
		ChainName:              chainName,
		BlockExplorerURL:       "https://explorer.example.com",
		IconURL:                "network/Network=" + chainName,
		NativeCurrencyName:     "Ether",
		NativeCurrencySymbol:   "ETH",
		NativeCurrencyDecimals: 18,
		IsTest:                 false,
		Layer:                  2,
		Enabled:                true,
		ChainColor:             "#E90101",
		ShortName:              "eth",
		RelatedChainID:         api_common.OptimismSepoliaChainID,
		RpcProviders:           providers,
		IsActive:               true,
		IsDeactivatable:        true,
	}
}

// Helper function to compare two providers
func CompareProviders(t require.TestingT, expected, actual params.RpcProvider) {
	require.Equal(t, expected.ChainID, actual.ChainID)
	require.Equal(t, expected.Name, actual.Name)
	require.Equal(t, expected.URL, actual.URL)
	require.Equal(t, expected.EnableRPSLimiter, actual.EnableRPSLimiter)
	require.Equal(t, expected.Type, actual.Type)
	require.Equal(t, expected.Enabled, actual.Enabled)
	require.Equal(t, expected.AuthType, actual.AuthType)
	require.Equal(t, expected.AuthLogin, actual.AuthLogin)
	require.Equal(t, expected.AuthPassword, actual.AuthPassword)
	require.Equal(t, expected.AuthToken, actual.AuthToken)
}

// Helper function to compare two networks
func CompareNetworks(t require.TestingT, expected, actual *params.Network) {
	require.Equal(t, expected.ChainID, actual.ChainID, "ChainID does not match")
	require.Equal(t, expected.ChainName, actual.ChainName, "ChainName does not match for ChainID %d", actual.ChainID)
	require.Equal(t, expected.BlockExplorerURL, actual.BlockExplorerURL)
	require.Equal(t, expected.NativeCurrencyName, actual.NativeCurrencyName)
	require.Equal(t, expected.NativeCurrencySymbol, actual.NativeCurrencySymbol)
	require.Equal(t, expected.NativeCurrencyDecimals, actual.NativeCurrencyDecimals)
	require.Equal(t, expected.IsTest, actual.IsTest)
	require.Equal(t, expected.Layer, actual.Layer)
	require.Equal(t, expected.Enabled, actual.Enabled)
	require.Equal(t, expected.ChainColor, actual.ChainColor)
	require.Equal(t, expected.ShortName, actual.ShortName)
	require.Equal(t, expected.RelatedChainID, actual.RelatedChainID)
	require.Equal(t, expected.IsActive, actual.IsActive)
	require.Equal(t, expected.IsDeactivatable, actual.IsDeactivatable)
}

// Helper function to compare lists of providers
func CompareProvidersList(t require.TestingT, expectedProviders, actualProviders []params.RpcProvider) {
	require.Len(t, actualProviders, len(expectedProviders))
	expectedMap := make(map[string]params.RpcProvider, len(expectedProviders))
	for _, provider := range expectedProviders {
		expectedMap[provider.Name] = provider
	}

	for _, provider := range actualProviders {
		expectedProvider, exists := expectedMap[provider.Name]
		require.True(t, exists, "Unexpected provider '%s'", provider.Name)
		CompareProviders(t, expectedProvider, provider)
	}
}

// Helper function to compare lists of networks
func CompareNetworksList(t require.TestingT, expectedNetworks, actualNetworks []*params.Network) {
	require.Len(t, actualNetworks, len(expectedNetworks))
	expectedMap := make(map[uint64]*params.Network, len(expectedNetworks))
	for _, network := range expectedNetworks {
		expectedMap[network.ChainID] = network
	}

	for _, network := range actualNetworks {
		expectedNetwork, exists := expectedMap[network.ChainID]
		require.True(t, exists, "Unexpected network with ChainID %d", network.ChainID)
		CompareNetworks(t, expectedNetwork, network)
	}
}

func ConvertNetworksToPointers(networks []params.Network) []*params.Network {
	result := make([]*params.Network, len(networks))
	for i := range networks {
		// Create a copy of the current network
		network := networks[i]
		// Store a pointer to the copy
		result[i] = &network
	}
	return result
}
