package networkhelper_test

import (
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api_common "github.com/status-im/status-go/api/common"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/params/networkhelper"
	"github.com/status-im/status-go/pkg/security"
	"github.com/status-im/status-go/rpc/network/testutil"
)

func TestMergeProvidersPreserveEnabledAndOrder(t *testing.T) {
	chainID := uint64(1)

	// Current providers with mixed types
	currentProviders := []params.RpcProvider{
		*params.NewUserProvider(chainID, "UserProvider1", security.NewSensitiveString("https://userprovider1.example.com"), true),
		*params.NewUserProvider(chainID, "UserProvider2", security.NewSensitiveString("https://userprovider2.example.com"), true),
		*params.NewDirectProvider(chainID, "EmbeddedProvider1", security.NewSensitiveString("https://embeddedprovider1.example.com"), true),
		*params.NewProxyProvider(chainID, "EmbeddedProvider2", security.NewSensitiveString("https://embeddedprovider2.example.com"), true),
	}
	currentProviders[1].Enabled = false // UserProvider2 is disabled
	currentProviders[2].Enabled = false // EmbeddedProvider1 is disabled

	// New providers to merge
	newProviders := []params.RpcProvider{
		*params.NewDirectProvider(chainID, "EmbeddedProvider1", security.NewSensitiveString("https://embeddedprovider1-new.example.com"), true), // Should retain Enabled: false
		*params.NewProxyProvider(chainID, "EmbeddedProvider3", security.NewSensitiveString("https://embeddedprovider3.example.com"), true),      // New embedded provider
		*params.NewDirectProvider(chainID, "EmbeddedProvider4", security.NewSensitiveString("https://embeddedprovider4.example.com"), true),     // New embedded provider
	}

	// Call MergeProviders
	mergedProviders := networkhelper.MergeProvidersPreservingUsersAndEnabledState(currentProviders, newProviders)

	expectedEmbeddedProvider1 := newProviders[0]
	expectedEmbeddedProvider1.Enabled = false // Should retain Enabled: false
	// Expected providers after merging
	expectedProviders := []params.RpcProvider{
		currentProviders[0],       // UserProvider1
		currentProviders[1],       // UserProvider2
		expectedEmbeddedProvider1, // EmbeddedProvider1 (should retain Enabled: false)
		newProviders[1],           // EmbeddedProvider3
		newProviders[2],           // EmbeddedProvider4
	}

	// Assertions
	require.True(t, reflect.DeepEqual(mergedProviders, expectedProviders), "Merged providers should match the expected providers")
}

func TestOverrideBasicAuth(t *testing.T) {
	// Arrange: Create a sample list of networks with various provider types
	networks := []params.Network{
		*testutil.CreateNetwork(api_common.MainnetChainID, "Ethereum Mainnet", []params.RpcProvider{
			*params.NewUserProvider(api_common.MainnetChainID, "Provider1", security.NewSensitiveString("https://userprovider.example.com"), true),
			*params.NewProxyProvider(api_common.MainnetChainID, "Provider2", security.NewSensitiveString("https://proxyprovider.example.com"), true),
			*params.NewEthRpcProxyProvider(api_common.MainnetChainID, "Provider3", security.NewSensitiveString("https://ethrpcproxy.example.com"), true),
		}),
		*testutil.CreateNetwork(api_common.OptimismChainID, "Optimism", []params.RpcProvider{
			*params.NewDirectProvider(api_common.OptimismChainID, "Provider4", security.NewSensitiveString("https://directprovider.example.com"), true),
			*params.NewProxyProvider(api_common.OptimismChainID, "Provider5", security.NewSensitiveString("https://proxyprovider2.example.com"), true),
			*params.NewEthRpcProxyProvider(api_common.OptimismChainID, "Provider6", security.NewSensitiveString("https://ethrpcproxy2.example.com"), true),
		}),
	}
	networks[0].RpcProviders[1].Enabled = false
	networks[0].RpcProviders[2].Enabled = false
	networks[1].RpcProviders[1].Enabled = false
	networks[1].RpcProviders[2].Enabled = false

	user := security.NewSensitiveString(gofakeit.Username())
	password := security.NewSensitiveString(gofakeit.LetterN(5))

	// Test updating EmbeddedProxyProviderType providers
	updatedNetworks := networkhelper.OverrideBasicAuth(networks, params.EmbeddedProxyProviderType, true, user, password)

	// Verify the networks
	for i, network := range updatedNetworks {
		networkCopy := network
		expectedNetwork := &networks[i]
		testutil.CompareNetworks(t, expectedNetwork, &networkCopy)

		for j, provider := range networkCopy.RpcProviders {
			expectedProvider := expectedNetwork.RpcProviders[j]
			if provider.Type == params.EmbeddedProxyProviderType {
				assert.True(t, provider.Enabled, "Provider Enabled state should be overridden")
				assert.Equal(t, user, provider.AuthLogin, "Provider AuthLogin should be overridden")
				assert.Equal(t, password, provider.AuthPassword, "Provider AuthPassword should be overridden")
				assert.Equal(t, params.BasicAuth, provider.AuthType, "Provider AuthType should be set to BasicAuth")
			} else {
				assert.Equal(t, expectedProvider.Enabled, provider.Enabled, "Provider Enabled state should remain unchanged")
				assert.Equal(t, expectedProvider.AuthLogin, provider.AuthLogin, "Provider AuthLogin should remain unchanged")
				assert.Equal(t, expectedProvider.AuthPassword, provider.AuthPassword, "Provider AuthPassword should remain unchanged")
			}
		}
	}

	// Test updating EmbeddedEthRpcProxyProviderType providers
	user2 := security.NewSensitiveString(gofakeit.Username())
	password2 := security.NewSensitiveString(gofakeit.LetterN(5))
	updatedNetworks = networkhelper.OverrideBasicAuth(networks, params.EmbeddedEthRpcProxyProviderType, true, user2, password2)

	// Verify the networks
	for i, network := range updatedNetworks {
		networkCopy := network
		expectedNetwork := &networks[i]
		testutil.CompareNetworks(t, expectedNetwork, &networkCopy)

		for j, provider := range networkCopy.RpcProviders {
			expectedProvider := expectedNetwork.RpcProviders[j]
			if provider.Type == params.EmbeddedEthRpcProxyProviderType {
				assert.True(t, provider.Enabled, "Provider Enabled state should be overridden")
				assert.Equal(t, user2, provider.AuthLogin, "Provider AuthLogin should be overridden")
				assert.Equal(t, password2, provider.AuthPassword, "Provider AuthPassword should be overridden")
				assert.Equal(t, params.BasicAuth, provider.AuthType, "Provider AuthType should be set to BasicAuth")
			} else {
				assert.Equal(t, expectedProvider.Enabled, provider.Enabled, "Provider Enabled state should remain unchanged")
				assert.Equal(t, expectedProvider.AuthLogin, provider.AuthLogin, "Provider AuthLogin should remain unchanged")
				assert.Equal(t, expectedProvider.AuthPassword, provider.AuthPassword, "Provider AuthPassword should remain unchanged")
			}
		}
	}
}

func TestOverrideDirectProvidersAuth(t *testing.T) {
	// Create a sample list of networks with various provider types
	networks := []params.Network{
		*testutil.CreateNetwork(api_common.MainnetChainID, "Ethereum Mainnet", []params.RpcProvider{
			*params.NewUserProvider(api_common.MainnetChainID, "Provider1", security.NewSensitiveString("https://user.example.com/"), true),
			*params.NewDirectProvider(api_common.MainnetChainID, "Provider2", security.NewSensitiveString("https://mainnet.infura.io/v3/"), true),
			*params.NewDirectProvider(api_common.MainnetChainID, "Provider3", security.NewSensitiveString("https://eth-archival.rpc.grove.city/v1/"), true),
		}),
		*testutil.CreateNetwork(api_common.OptimismChainID, "Optimism", []params.RpcProvider{
			*params.NewDirectProvider(api_common.OptimismChainID, "Provider4", security.NewSensitiveString("https://optimism.infura.io/v3/"), true),
			*params.NewDirectProvider(api_common.OptimismChainID, "Provider5", security.NewSensitiveString("https://op.grove.city/v1/"), true),
		}),
	}

	authTokens := map[string]security.SensitiveString{
		"infura.io":   security.NewSensitiveString(gofakeit.UUID()),
		"grove.city":  security.NewSensitiveString(gofakeit.UUID()),
		"example.com": security.NewSensitiveString(gofakeit.UUID()),
	}

	// Call OverrideDirectProvidersAuth
	updatedNetworks := networkhelper.OverrideDirectProvidersAuth(networks, authTokens)

	// Verify the networks have updated auth tokens correctly
	for i, network := range updatedNetworks {
		for j, provider := range network.RpcProviders {
			expectedProvider := networks[i].RpcProviders[j]
			switch {
			case provider.URL.Contains("infura.io"):
				assert.Equal(t, params.TokenAuth, provider.AuthType)
				assert.Equal(t, authTokens["infura.io"], provider.AuthToken)
				assert.NotEqual(t, expectedProvider.AuthToken, provider.AuthToken)
			case provider.URL.Contains("grove.city"):
				assert.Equal(t, params.TokenAuth, provider.AuthType)
				assert.Equal(t, authTokens["grove.city"], provider.AuthToken)
				assert.NotEqual(t, expectedProvider.AuthToken, provider.AuthToken)
			case provider.URL.Contains("example.com"):
				assert.Equal(t, params.NoAuth, provider.AuthType) // should not update user providers
			default:
				assert.Equal(t, expectedProvider.AuthType, provider.AuthType)
				assert.Equal(t, expectedProvider.AuthToken, provider.AuthToken)
			}
		}
	}
}

func TestDeepCopyNetwork(t *testing.T) {
	originalNetwork := testutil.CreateNetwork(api_common.MainnetChainID, "Ethereum Mainnet", []params.RpcProvider{
		*params.NewUserProvider(api_common.MainnetChainID, "Provider1", security.NewSensitiveString("https://userprovider.example.com"), true),
		*params.NewDirectProvider(api_common.MainnetChainID, "Provider2", security.NewSensitiveString("https://mainnet.infura.io/v3/"), true),
	})

	originalNetwork.TokenOverrides = []params.TokenOverride{
		{Symbol: "token1", Address: common.HexToAddress("0x123")},
	}

	copiedNetwork := originalNetwork.DeepCopy()

	assert.True(t, reflect.DeepEqual(originalNetwork, &copiedNetwork), "Copied network should be deeply equal to the original")

	// Modify the copied network and verify that the original network remains unchanged
	copiedNetwork.RpcProviders[0].Enabled = false
	copiedNetwork.TokenOverrides[0].Symbol = "modifiedSymbol"
	assert.NotEqual(t, originalNetwork.RpcProviders[0].Enabled, copiedNetwork.RpcProviders[0].Enabled, "Original network should remain unchanged")
	assert.NotEqual(t, originalNetwork.TokenOverrides[0].Symbol, copiedNetwork.TokenOverrides[0].Symbol, "Original network should remain unchanged")
}
