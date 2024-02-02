package api

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/protocol/requests"
)

func TestBuildDefaultNetworks(t *testing.T) {
	poktToken := "grove-token"
	infuraToken := "infura-token"
	request := &requests.CreateAccount{
		WalletSecretsConfig: requests.WalletSecretsConfig{
			PoktToken:   poktToken,
			InfuraToken: infuraToken,
		},
	}

	actualNetworks := BuildDefaultNetworks(request)

	require.Len(t, actualNetworks, 9)

	require.Equal(t, mainnetChainID, actualNetworks[0].ChainID)

	require.True(t, strings.Contains(actualNetworks[0].RPCURL, poktToken))
	require.True(t, strings.Contains(actualNetworks[0].FallbackURL, infuraToken))

	require.Equal(t, goerliChainID, actualNetworks[1].ChainID)

	require.True(t, strings.Contains(actualNetworks[1].RPCURL, poktToken))
	require.True(t, strings.Contains(actualNetworks[1].FallbackURL, infuraToken))

	require.Equal(t, sepoliaChainID, actualNetworks[2].ChainID)

	require.True(t, strings.Contains(actualNetworks[2].RPCURL, poktToken))
	require.True(t, strings.Contains(actualNetworks[2].FallbackURL, infuraToken))

	require.Equal(t, optimismChainID, actualNetworks[3].ChainID)

	require.True(t, strings.Contains(actualNetworks[3].RPCURL, poktToken))
	require.True(t, strings.Contains(actualNetworks[3].FallbackURL, infuraToken))

	require.Equal(t, optimismGoerliChainID, actualNetworks[4].ChainID)

	require.True(t, strings.Contains(actualNetworks[4].RPCURL, infuraToken))
	require.Equal(t, "", actualNetworks[4].FallbackURL)

	require.Equal(t, optimismSepoliaChainID, actualNetworks[5].ChainID)

	require.True(t, strings.Contains(actualNetworks[5].RPCURL, infuraToken))
	require.Equal(t, "", actualNetworks[5].FallbackURL)

	require.Equal(t, arbitrumChainID, actualNetworks[6].ChainID)

	require.True(t, strings.Contains(actualNetworks[6].RPCURL, poktToken))
	require.True(t, strings.Contains(actualNetworks[6].FallbackURL, infuraToken))

	require.Equal(t, arbitrumGoerliChainID, actualNetworks[7].ChainID)

	require.True(t, strings.Contains(actualNetworks[7].RPCURL, infuraToken))
	require.Equal(t, "", actualNetworks[7].FallbackURL)

	require.Equal(t, arbitrumSepoliaChainID, actualNetworks[8].ChainID)

	require.True(t, strings.Contains(actualNetworks[8].RPCURL, infuraToken))
	require.Equal(t, "", actualNetworks[8].FallbackURL)
}

func TestBuildDefaultNetworksGanache(t *testing.T) {
	ganacheURL := "ganacheurl"
	request := &requests.CreateAccount{
		WalletSecretsConfig: requests.WalletSecretsConfig{
			GanacheURL: ganacheURL,
		},
	}

	actualNetworks := BuildDefaultNetworks(request)

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
