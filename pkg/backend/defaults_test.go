package backend

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/protocol/requests"
)

func boolPtr(v bool) *bool {
	return &v
}

func TestBuildWalletConfigSwapProviderDefaults(t *testing.T) {
	secrets := &requests.WalletSecretsConfig{}

	// Not specified: provider defaults apply.
	walletConfig := buildWalletConfig(&requests.WalletConfig{}, secrets)
	require.False(t, walletConfig.EnableParaswapProvider)

	// Explicit values are honoured in both directions.
	walletConfig = buildWalletConfig(&requests.WalletConfig{
		EnableParaswapProvider: boolPtr(true),
	}, secrets)
	require.True(t, walletConfig.EnableParaswapProvider)

	walletConfig = buildWalletConfig(&requests.WalletConfig{
		EnableParaswapProvider: boolPtr(false),
	}, secrets)
	require.False(t, walletConfig.EnableParaswapProvider)
}
