package networks

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/security"
	"github.com/status-im/status-go/pkg/services/wallet/common"
)

func providerNames(n params.Network) []string {
	names := make([]string, 0, len(n.RpcProviders))
	for _, p := range n.RpcProviders {
		names = append(names, p.Name)
	}
	return names
}

func TestBuildDefaultNetworks_FallbackProxy(t *testing.T) {
	t.Run("is the last provider of every network the proxy serves on prod", func(t *testing.T) {
		walletConfig := &params.WalletConfig{
			StatusProxyStageName: "prod",
			EthRpcProxyUser:      security.NewSensitiveString("user"),
			EthRpcProxyPassword:  security.NewSensitiveString("password"),
		}

		served := 0
		for _, n := range BuildDefaultNetworks(walletConfig, true) {
			names := providerNames(n)
			mainProxy := findProvider(n.RpcProviders, common.StatusSmartProxy)
			if mainProxy == nil {
				// Not served by the proxy (Katana), so not by the fallback either
				require.NotContains(t, names, common.StatusSmartProxyFallback, "chain %d", n.ChainID)
				continue
			}
			served++

			// Once, and last
			require.Equal(t, common.StatusSmartProxyFallback, names[len(names)-1], "chain %d", n.ChainID)
			require.Equal(t, 1, strings.Count(strings.Join(names, " "), common.StatusSmartProxyFallback))

			// The same path on the test host, with puzzle auth and no credentials
			fallback := n.RpcProviders[len(n.RpcProviders)-1]
			require.Equal(t, params.EmbeddedEthRpcProxyProviderType, fallback.Type)
			require.Equal(t,
				strings.Replace(mainProxy.URL.Reveal(), "https://prod.", "https://test.", 1),
				fallback.URL.Reveal())
			require.Equal(t, params.PuzzleAuth, fallback.AuthType)
			require.True(t, fallback.AuthLogin.Empty())
			require.True(t, fallback.AuthPassword.Empty())
			require.True(t, fallback.Enabled)
		}
		require.NotZero(t, served)
	})

	t.Run("is disabled when the main proxy is", func(t *testing.T) {
		// Basic auth without credentials: the main proxy is switched off
		walletConfig := &params.WalletConfig{StatusProxyStageName: "prod"}

		for _, n := range BuildDefaultNetworks(walletConfig, true) {
			mainProxy := findProvider(n.RpcProviders, common.StatusSmartProxy)
			if mainProxy == nil {
				continue
			}
			require.False(t, mainProxy.Enabled)
			require.False(t, findProvider(n.RpcProviders, common.StatusSmartProxyFallback).Enabled, "chain %d", n.ChainID)
		}
	})

	t.Run("is absent without the providers, off prod and with a custom proxy", func(t *testing.T) {
		testCases := []struct {
			name              string
			walletConfig      *params.WalletConfig
			thirdpartyEnabled bool
		}{
			{"third-party services off", &params.WalletConfig{StatusProxyStageName: "prod"}, false},
			{"test stage", &params.WalletConfig{StatusProxyStageName: "test"}, true},
			{"any other stage", &params.WalletConfig{StatusProxyStageName: "fast-n-bulbous"}, true},
			{"custom proxy URL", &params.WalletConfig{
				StatusProxyStageName: "prod",
				EthRpcProxyUrl:       security.NewSensitiveString("https://rpc.example.org"),
			}, true},
		}

		for _, tc := range testCases {
			for _, n := range BuildDefaultNetworks(tc.walletConfig, tc.thirdpartyEnabled) {
				require.NotContains(t, providerNames(n), common.StatusSmartProxyFallback, "%s, chain %d", tc.name, n.ChainID)
			}
		}
	})
}
