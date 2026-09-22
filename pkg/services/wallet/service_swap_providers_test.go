package wallet

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/params"
	pathProcessorCommon "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/common"
)

func swapProcessorNames(t *testing.T, walletConfig *params.WalletConfig) []string {
	t.Helper()
	names := []string{}
	for _, p := range buildSwapPathProcessors(nil, nil, nil, walletConfig) {
		names = append(names, p.Name())
	}
	return names
}

func TestBuildSwapPathProcessors(t *testing.T) {
	require.Empty(t, swapProcessorNames(t, &params.WalletConfig{}))

	// a single enabled provider is used as is
	require.Equal(t, []string{pathProcessorCommon.ProcessorSwapParaswapName},
		swapProcessorNames(t, &params.WalletConfig{EnableParaswapProvider: true}))
	require.Equal(t, []string{pathProcessorCommon.ProcessorLiFiName},
		swapProcessorNames(t, &params.WalletConfig{EnableLiFiProvider: true}))
	require.Equal(t, []string{pathProcessorCommon.ProcessorRelayName},
		swapProcessorNames(t, &params.WalletConfig{EnableRelayProvider: true}))

	// no fallback support for more than one swap/bridge provider: only the highest
	// priority enabled one is registered (Relay > LI.FI > Paraswap)
	require.Equal(t, []string{pathProcessorCommon.ProcessorRelayName},
		swapProcessorNames(t, &params.WalletConfig{EnableRelayProvider: true, EnableLiFiProvider: true, EnableParaswapProvider: true}))
	require.Equal(t, []string{pathProcessorCommon.ProcessorRelayName},
		swapProcessorNames(t, &params.WalletConfig{EnableRelayProvider: true, EnableParaswapProvider: true}))
	require.Equal(t, []string{pathProcessorCommon.ProcessorLiFiName},
		swapProcessorNames(t, &params.WalletConfig{EnableLiFiProvider: true, EnableParaswapProvider: true}))
}

// Hop is not an option at all: bridging is served by the swap provider (Relay or LI.FI).
func TestBuildPathProcessorsRegistersNoHop(t *testing.T) {
	for _, p := range buildPathProcessors(nil, nil, nil, nil, nil, &params.WalletConfig{EnableRelayProvider: true}) {
		require.NotEqual(t, pathProcessorCommon.ProcessorBridgeHopName, p.Name())
	}
}
