package api

import (
	"strings"
	"testing"

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

	require.Len(t, actualNetworks, 8)
	for _, n := range actualNetworks {
		var err error
		switch n.ChainID {
		case MainnetChainID:
		case SepoliaChainID:
		case OptimismChainID:
		case OptimismSepoliaChainID:
		case ArbitrumChainID:
		case ArbitrumSepoliaChainID:
		case BaseChainID:
		case BaseSepoliaChainID:
		default:
			err = errors.Errorf("unexpected chain id: %d", n.ChainID)
		}
		require.NoError(t, err)

		// check default chains
		// DefaultRPCURL and DefaultFallbackURL are mandatory
		require.True(t, strings.Contains(n.DefaultRPCURL, stageName))
		require.True(t, strings.Contains(n.DefaultFallbackURL, stageName))
		if n.DefaultFallbackURL2 != "" {
			require.True(t, strings.Contains(actualNetworks[0].DefaultFallbackURL2, stageName))
		}

		// check fallback options
		require.True(t, strings.Contains(n.RPCURL, infuraToken))
		require.True(t, strings.Contains(n.FallbackURL, poktToken))

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
