package backend

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/protocol/requests"
	"github.com/status-im/status-go/pkg/security"
)

func boolPtr(v bool) *bool {
	return &v
}

func TestBuildWalletConfigSwapProviderDefaults(t *testing.T) {
	secrets := &requests.WalletSecretsConfig{}

	// Not specified: provider defaults apply.
	walletConfig := buildWalletConfig(&requests.WalletConfig{}, secrets)
	require.False(t, walletConfig.EnableParaswapProvider)
	require.False(t, walletConfig.EnableLiFiProvider)
	require.False(t, walletConfig.EnableRelayProvider)

	// Explicit values are honoured in both directions.
	walletConfig = buildWalletConfig(&requests.WalletConfig{
		EnableParaswapProvider: boolPtr(true),
		EnableLiFiProvider:     boolPtr(false),
	}, secrets)
	require.True(t, walletConfig.EnableParaswapProvider)
	require.False(t, walletConfig.EnableLiFiProvider)

	walletConfig = buildWalletConfig(&requests.WalletConfig{
		EnableParaswapProvider: boolPtr(false),
		EnableLiFiProvider:     boolPtr(true),
	}, secrets)
	require.False(t, walletConfig.EnableParaswapProvider)
	require.True(t, walletConfig.EnableLiFiProvider)
}

func TestBuildWalletConfigRelayProvider(t *testing.T) {
	walletConfig := buildWalletConfig(&requests.WalletConfig{
		EnableRelayProvider: boolPtr(true),
	}, &requests.WalletSecretsConfig{})
	require.True(t, walletConfig.EnableRelayProvider)
	require.True(t, walletConfig.RelayAPIKey.Empty())

	walletConfig = buildWalletConfig(&requests.WalletConfig{}, &requests.WalletSecretsConfig{
		RelayAPIKey: security.NewSensitiveString("relay-key"),
	})
	require.False(t, walletConfig.EnableRelayProvider)
	require.Equal(t, "relay-key", walletConfig.RelayAPIKey.Reveal())
}

func TestBuildWalletConfigMarketDataProxyPuzzleAuth(t *testing.T) {
	walletConfig := buildWalletConfig(&requests.WalletConfig{}, &requests.WalletSecretsConfig{})
	require.False(t, walletConfig.MarketDataProxyConfig.UsePuzzleAuth)

	walletConfig = buildWalletConfig(&requests.WalletConfig{}, &requests.WalletSecretsConfig{
		MarketDataProxyUsePuzzleAuth: true,
	})
	require.True(t, walletConfig.MarketDataProxyConfig.UsePuzzleAuth)
	// The other proxies keep their own switches.
	require.False(t, walletConfig.NftProxyConfig.UsePuzzleAuth)
	require.False(t, walletConfig.EthRpcProxyUsePuzzleAuth)
}
