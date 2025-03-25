package api

import (
	"strings"
	"testing"

	"github.com/status-im/status-go/api/common"
	"github.com/status-im/status-go/params"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/protocol/requests"
)

func TestBuildDefaultNetworks(t *testing.T) {
	infuraToken := "infura-token"
	poktToken := "pokt-token"
	stageName := "fast-n-bulbous"
	request := &requests.CreateAccount{
		WalletSecretsConfig: requests.WalletSecretsConfig{
			InfuraToken:          infuraToken,
			PoktToken:            poktToken,
			StatusProxyStageName: stageName,
		},
	}

	actualNetworks := BuildDefaultNetworks(&request.WalletSecretsConfig)

	require.Len(t, actualNetworks, 9)
	for _, n := range actualNetworks {
		var err error
		switch n.ChainID {
		case common.MainnetChainID:
		case common.SepoliaChainID:
		case common.OptimismChainID:
		case common.OptimismSepoliaChainID:
		case common.ArbitrumChainID:
		case common.ArbitrumSepoliaChainID:
		case common.BaseChainID:
		case common.BaseSepoliaChainID:
		case common.StatusNetworkSepoliaChainID:
		default:
			err = errors.Errorf("unexpected chain id: %d", n.ChainID)
		}
		require.NoError(t, err)

		// check default chains
		if n.DefaultRPCURL != "" {
			require.True(t, strings.Contains(n.DefaultRPCURL, stageName))
		}
		if n.DefaultFallbackURL != "" {
			require.True(t, strings.Contains(n.DefaultFallbackURL, stageName))
		}
		if n.DefaultFallbackURL2 != "" {
			require.True(t, strings.Contains(actualNetworks[0].DefaultFallbackURL2, stageName))
		}

		// check fallback options
		if strings.Contains(n.RPCURL, "infura.io") {
			require.True(t, strings.Contains(n.RPCURL, infuraToken))
		}
		if strings.Contains(n.FallbackURL, "grove.city") {
			require.True(t, strings.Contains(n.FallbackURL, poktToken))
		}

		// Check proxy providers for stageName
		for _, provider := range n.RpcProviders {
			if provider.Type == params.EmbeddedProxyProviderType {
				require.Contains(t, provider.URL, stageName, "Proxy provider URL should contain stageName")
			}
		}

		// Check direct providers for tokens
		for _, provider := range n.RpcProviders {
			if provider.Type != params.EmbeddedDirectProviderType {
				continue
			}
			if strings.Contains(provider.URL, "infura.io") {
				require.Equal(t, provider.AuthToken, infuraToken, "Direct provider URL should have infuraToken")
			} else if strings.Contains(provider.URL, "grove.city") {
				require.Equal(t, provider.AuthToken, poktToken, "Direct provider URL should have poktToken")
			}
		}
	}
}
