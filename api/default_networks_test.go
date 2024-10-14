package api

import (
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/protocol/requests"
)

func TestBuildDefaultNetworks(t *testing.T) {
	rpcToken := "infura-token"
	fallbackToken := ""
	stageName := "fast-n-bulbous"
	request := &requests.CreateAccount{
		WalletSecretsConfig: requests.WalletSecretsConfig{
			InfuraToken:          rpcToken,
			StatusProxyStageName: stageName,
		},
	}

	actualNetworks := BuildDefaultNetworks(&request.WalletSecretsConfig)

	require.Len(t, actualNetworks, 9)

	ignoreDefaultRPCURLCheck := false // TODO: used just because of Goerli, remove once we remove Goerli from the default networks

	for _, n := range actualNetworks {
		var err error
		switch n.ChainID {
		case mainnetChainID:
		case goerliChainID:
			ignoreDefaultRPCURLCheck = true
		case sepoliaChainID:
		case optimismChainID:
		case optimismGoerliChainID:
			ignoreDefaultRPCURLCheck = true
		case optimismSepoliaChainID:
		case arbitrumChainID:
		case arbitrumGoerliChainID:
			ignoreDefaultRPCURLCheck = true
		case arbitrumSepoliaChainID:
		default:
			err = errors.Errorf("unexpected chain id: %d", n.ChainID)
		}
		require.NoError(t, err)

		// check default chains
		if !ignoreDefaultRPCURLCheck {
			// DefaultRPCURL and DefaultFallbackURL are mandatory
			require.True(t, strings.Contains(n.DefaultRPCURL, stageName))
			require.True(t, strings.Contains(n.DefaultFallbackURL, stageName))
			if n.DefaultFallbackURL2 != "" {
				require.True(t, strings.Contains(actualNetworks[0].DefaultFallbackURL2, stageName))
			}
		}

		// check fallback options
		require.True(t, strings.Contains(n.RPCURL, rpcToken))
		require.True(t, strings.Contains(n.FallbackURL, fallbackToken))
	}
}

func TestBuildDefaultNetworksGanache(t *testing.T) {
	ganacheURL := "ganacheurl"
	request := &requests.CreateAccount{
		WalletSecretsConfig: requests.WalletSecretsConfig{
			GanacheURL: ganacheURL,
		},
	}

	actualNetworks := BuildDefaultNetworks(&request.WalletSecretsConfig)

	require.Len(t, actualNetworks, 9)

	for _, n := range actualNetworks {
		require.True(t, strings.Contains(n.RPCURL, ganacheURL))
		require.True(t, strings.Contains(n.FallbackURL, ganacheURL))

	}

	require.Equal(t, mainnetChainID, actualNetworks[0].ChainID)

	require.NotNil(t, actualNetworks[0].TokenOverrides)
	require.Len(t, actualNetworks[0].TokenOverrides, 1)
	require.Equal(t, sntSymbol, actualNetworks[0].TokenOverrides[0].Symbol)
	require.Equal(t, ganacheTokenAddress, actualNetworks[0].TokenOverrides[0].Address)

	require.Equal(t, goerliChainID, actualNetworks[1].ChainID)

	require.NotNil(t, actualNetworks[1].TokenOverrides)
	require.Len(t, actualNetworks[1].TokenOverrides, 1)
	require.Equal(t, sttSymbol, actualNetworks[1].TokenOverrides[0].Symbol)
	require.Equal(t, ganacheTokenAddress, actualNetworks[1].TokenOverrides[0].Address)

}
