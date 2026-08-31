package networks

import (
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/security"
	"github.com/status-im/status-go/pkg/services/networks/testutil"
	walletcommon "github.com/status-im/status-go/pkg/services/wallet/common"
)

func TestMergeProvidersPreserveEnabledAndOrder(t *testing.T) {
	chainID := uint64(1)

	// Current providers with mixed types
	currentProviders := []params.RpcProvider{
		*params.NewUserProvider(chainID, "UserProvider1", security.NewSensitiveString("https://userprovider1.example.com"), true),
		*params.NewUserProvider(chainID, "UserProvider2", security.NewSensitiveString("https://userprovider2.example.com"), true),
		*params.NewDirectProvider(chainID, "EmbeddedProvider1", security.NewSensitiveString("https://embeddedprovider1.example.com"), true),
		*params.NewEthRpcProxyProvider(chainID, "EmbeddedProvider2", security.NewSensitiveString("https://embeddedprovider2.example.com"), true, false),
	}
	currentProviders[1].Enabled = false // UserProvider2 is disabled
	currentProviders[2].Enabled = false // EmbeddedProvider1 is disabled

	// New providers to merge
	newProviders := []params.RpcProvider{
		*params.NewDirectProvider(chainID, "EmbeddedProvider1", security.NewSensitiveString("https://embeddedprovider1-new.example.com"), true),         // Should retain Enabled: false
		*params.NewEthRpcProxyProvider(chainID, "EmbeddedProvider3", security.NewSensitiveString("https://embeddedprovider3.example.com"), true, false), // New embedded provider
		*params.NewDirectProvider(chainID, "EmbeddedProvider4", security.NewSensitiveString("https://embeddedprovider4.example.com"), true),             // New embedded provider
	}

	// Call MergeProviders
	mergedProviders := mergeProvidersPreservingUsersAndEnabledState(currentProviders, newProviders)

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
		*testutil.CreateNetwork(walletcommon.EthereumMainnet, "Ethereum Mainnet", []params.RpcProvider{
			*params.NewUserProvider(walletcommon.EthereumMainnet, "Provider1", security.NewSensitiveString("https://userprovider.example.com"), true),
			*params.NewEthRpcProxyProvider(walletcommon.EthereumMainnet, "Provider2", security.NewSensitiveString("https://ethrpcproxy.example.com"), true, false),
			*params.NewEthRpcProxyProvider(walletcommon.EthereumMainnet, "Provider3", security.NewSensitiveString("https://ethrpcproxy.example.com"), true, false),
		}),
		*testutil.CreateNetwork(walletcommon.OptimismMainnet, "Optimism", []params.RpcProvider{
			*params.NewDirectProvider(walletcommon.OptimismMainnet, "Provider4", security.NewSensitiveString("https://directprovider.example.com"), true),
			*params.NewEthRpcProxyProvider(walletcommon.OptimismMainnet, "Provider5", security.NewSensitiveString("https://ethrpcproxy2.example.com"), true, false),
			*params.NewEthRpcProxyProvider(walletcommon.OptimismMainnet, "Provider6", security.NewSensitiveString("https://ethrpcproxy2.example.com"), true, false),
		}),
	}
	networks[0].RpcProviders[1].Enabled = false
	networks[0].RpcProviders[2].Enabled = false
	networks[1].RpcProviders[1].Enabled = false
	networks[1].RpcProviders[2].Enabled = false

	// Test updating EmbeddedEthRpcProxyProviderType providers
	user2 := security.NewSensitiveString(gofakeit.Username())
	password2 := security.NewSensitiveString(gofakeit.LetterN(5))
	updatedNetworks := OverrideBasicAuth(networks, params.EmbeddedEthRpcProxyProviderType, true, user2, password2)

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

func TestOverrideBasicAuth_PuzzleProvidersLeftUntouched(t *testing.T) {
	ethPuzzle := *params.NewEthRpcProxyProvider(walletcommon.EthereumMainnet, "SmartPuzzle",
		security.NewSensitiveString("https://eth.example.com/ethereum/mainnet/"), false, true)
	require.Equal(t, params.PuzzleAuth, ethPuzzle.AuthType)
	require.True(t, ethPuzzle.Enabled)

	ethNoPuzzleAuth := *params.NewEthRpcProxyProvider(walletcommon.EthereumMainnet, "NoPuzzleAuth",
		security.NewSensitiveString("https://eth.example.com/ethereum/mainnet/"), false, false)
	require.Equal(t, params.NoAuth, ethNoPuzzleAuth.AuthType)
	require.True(t, ethNoPuzzleAuth.AuthLogin.Empty())
	require.True(t, ethNoPuzzleAuth.AuthPassword.Empty())
	require.True(t, ethNoPuzzleAuth.Enabled)

	networks := []params.Network{
		*testutil.CreateNetwork(walletcommon.EthereumMainnet, "Ethereum", []params.RpcProvider{ethPuzzle, ethNoPuzzleAuth}),
	}
	user := security.NewSensitiveString("u")
	pass := security.NewSensitiveString("p")

	updated := OverrideBasicAuth(networks, params.EmbeddedEthRpcProxyProviderType, true, user, pass)
	require.Len(t, updated, 1)
	for _, p := range updated[0].RpcProviders {
		if p.AuthType == params.PuzzleAuth {
			require.Equal(t, params.PuzzleAuth, p.AuthType)
			require.True(t, p.Enabled) // puzzle provider stays enabled
			require.True(t, p.AuthLogin.Empty())
			require.True(t, p.AuthPassword.Empty())
		} else {
			require.Equal(t, params.BasicAuth, p.AuthType)
			require.True(t, p.Enabled) // basic auth provider is enabled
			require.False(t, p.AuthLogin.Empty())
			require.False(t, p.AuthPassword.Empty())
		}
	}

	disabled := OverrideBasicAuth(networks, params.EmbeddedEthRpcProxyProviderType, false, user, pass)
	for _, p := range disabled[0].RpcProviders {
		if p.AuthType == params.PuzzleAuth {
			require.Equal(t, params.PuzzleAuth, p.AuthType)
			require.True(t, p.Enabled) // puzzle provider stays enabled
			require.True(t, p.AuthLogin.Empty())
			require.True(t, p.AuthPassword.Empty())
		} else {
			require.Equal(t, params.BasicAuth, p.AuthType)
			require.False(t, p.Enabled) // direct provider is disabled
			require.False(t, p.AuthLogin.Empty())
			require.False(t, p.AuthPassword.Empty())
		}
	}
}

func TestOverrideDirectProvidersAuth(t *testing.T) {
	// Create a sample list of networks with various provider types
	networks := []params.Network{
		*testutil.CreateNetwork(walletcommon.EthereumMainnet, "Ethereum Mainnet", []params.RpcProvider{
			*params.NewUserProvider(walletcommon.EthereumMainnet, "Provider1", security.NewSensitiveString("https://user.example.com/"), true),
			*params.NewDirectProvider(walletcommon.EthereumMainnet, "Provider2", security.NewSensitiveString("https://mainnet.infura.io/v3/"), true),
			*params.NewDirectProvider(walletcommon.EthereumMainnet, "Provider3", security.NewSensitiveString("https://eth-archival.rpc.grove.city/v1/"), true),
		}),
		*testutil.CreateNetwork(walletcommon.OptimismMainnet, "Optimism", []params.RpcProvider{
			*params.NewDirectProvider(walletcommon.OptimismMainnet, "Provider4", security.NewSensitiveString("https://optimism.infura.io/v3/"), true),
			*params.NewDirectProvider(walletcommon.OptimismMainnet, "Provider5", security.NewSensitiveString("https://op.grove.city/v1/"), true),
		}),
	}

	authTokens := map[string]security.SensitiveString{
		"infura.io":   security.NewSensitiveString(gofakeit.UUID()),
		"grove.city":  security.NewSensitiveString(gofakeit.UUID()),
		"example.com": security.NewSensitiveString(gofakeit.UUID()),
	}

	// Call overrideDirectProvidersAuth
	updatedNetworks := overrideDirectProvidersAuth(networks, authTokens)

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
	originalNetwork := testutil.CreateNetwork(walletcommon.EthereumMainnet, "Ethereum Mainnet", []params.RpcProvider{
		*params.NewUserProvider(walletcommon.EthereumMainnet, "Provider1", security.NewSensitiveString("https://userprovider.example.com"), true),
		*params.NewDirectProvider(walletcommon.EthereumMainnet, "Provider2", security.NewSensitiveString("https://mainnet.infura.io/v3/"), true),
	})

	originalNetwork.TokenOverrides = []types.Token{
		{ChainID: walletcommon.EthereumMainnet, Address: common.HexToAddress("0x123")},
	}

	copiedNetwork := originalNetwork.DeepCopy()

	assert.True(t, reflect.DeepEqual(originalNetwork, &copiedNetwork), "Copied network should be deeply equal to the original")

	// Modify the copied network and verify that the original network remains unchanged
	copiedNetwork.RpcProviders[0].Enabled = false
	copiedNetwork.TokenOverrides[0].ChainID = walletcommon.OptimismMainnet
	copiedNetwork.TokenOverrides[0].Address = common.HexToAddress("0x456")
	assert.NotEqual(t, originalNetwork.RpcProviders[0].Enabled, copiedNetwork.RpcProviders[0].Enabled, "Original network should remain unchanged")
	assert.NotEqual(t, originalNetwork.TokenOverrides[0].ChainID, copiedNetwork.TokenOverrides[0].ChainID, "Original network should remain unchanged")
	assert.NotEqual(t, originalNetwork.TokenOverrides[0].Address, copiedNetwork.TokenOverrides[0].Address, "Original network should remain unchanged")
}
