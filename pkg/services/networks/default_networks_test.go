package networks

import (
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/security"
	"github.com/status-im/status-go/pkg/services/wallet/common"
)

func TestBuildDefaultNetworks(t *testing.T) {
	infuraToken := security.NewSensitiveString("infura-token")
	poktToken := security.NewSensitiveString("pokt-token")
	stageName := "fast-n-bulbous"
	walletConfig := &params.WalletConfig{
		InfuraAPIKey:         infuraToken,
		PoktAPIKey:           poktToken,
		StatusProxyStageName: stageName,
	}

	actualNetworks := BuildDefaultNetworks(walletConfig, true)

	require.Len(t, actualNetworks, 31)
	for _, n := range actualNetworks {
		var err error
		switch n.ChainID {
		case common.EthereumMainnet:
		case common.EthereumHoodi:
		case common.EthereumSepolia:
		case common.OptimismMainnet:
		case common.OptimismSepolia:
		case common.ArbitrumMainnet:
		case common.ArbitrumSepolia:
		case common.BaseMainnet:
		case common.BaseSepolia:
		case common.LineaMainnet:
		case common.LineaSepolia:
		case common.UnichainMainnet:
		case common.UnichainSepolia:
		case common.KatanaMainnet:
		case common.KatanaBokuto:
		case common.InkMainnet:
		case common.InkSepolia:
		case common.AbstractMainnet:
		case common.AbstractTestnet:
		case common.ZkSyncMainnet:
		case common.ZkSyncSepolia:
		case common.SoneiumMainnet:
		case common.SoneiumMinato:
		case common.ScrollMainnet:
		case common.ScrollSepolia:
		case common.BlastMainnet:
		case common.BlastSepolia:
		case common.RobinhoodMainnet:
		case common.RobinhoodTestnet:
		case common.BSCMainnet:
		case common.BSCTestnet:
		default:
			err = errors.Errorf("unexpected chain id: %d", n.ChainID)
		}
		require.NoError(t, err)

		// check fallback options
		if strings.Contains(n.RPCURL, "infura.io") {
			require.True(t, strings.Contains(n.RPCURL, infuraToken.Reveal()))
		}
		if strings.Contains(n.FallbackURL, "grove.city") {
			require.True(t, strings.Contains(n.FallbackURL, poktToken.Reveal()))
		}

		// Check direct providers for tokens
		for _, provider := range n.RpcProviders {
			if provider.Type != params.EmbeddedDirectProviderType {
				continue
			}
			if provider.URL.Contains("infura.io") {
				require.Equal(t, provider.AuthToken, infuraToken, "Direct provider URL should have infuraToken")
			} else if provider.URL.Contains("grove.city") {
				require.Equal(t, provider.AuthToken, poktToken, "Direct provider URL should have poktToken")
			}
		}
	}
}

func countDirectAlchemy(networks []params.Network) (count int, providers []params.RpcProvider) {
	for _, n := range networks {
		for _, p := range n.RpcProviders {
			if p.Name == common.DirectAlchemy {
				count++
				providers = append(providers, p)
			}
		}
	}
	return count, providers
}

func TestBuildDefaultNetworksDirectAlchemy(t *testing.T) {
	t.Run("absent without a key", func(t *testing.T) {
		count, _ := countDirectAlchemy(BuildDefaultNetworks(&params.WalletConfig{}, true))
		require.Zero(t, count)
	})

	t.Run("added next to every smart proxy alchemy provider with the key", func(t *testing.T) {
		alchemyKey := security.NewSensitiveString("alchemy-key")
		networks := BuildDefaultNetworks(&params.WalletConfig{AlchemyAPIKey: alchemyKey}, true)

		proxyCount := 0
		for _, n := range networks {
			for _, p := range n.RpcProviders {
				if p.Name == common.SmartProxyAlchemy {
					proxyCount++
				}
			}
		}

		count, providers := countDirectAlchemy(networks)
		require.Equal(t, proxyCount, count)
		for _, p := range providers {
			require.Equal(t, params.EmbeddedDirectProviderType, p.Type)
			require.True(t, p.Enabled)
			require.Equal(t, params.TokenAuth, p.AuthType)
			require.Equal(t, alchemyKey, p.AuthToken)
			require.True(t, strings.HasSuffix(p.GetHost(), ".g.alchemy.com"))
		}
	})
}
