package wallet

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/security"
)

func TestFallbackMarketProxyConfigs(t *testing.T) {
	prod := params.MarketDataProxyConfig{
		StageName:            "prod",
		User:                 security.NewSensitiveString("user"),
		Password:             security.NewSensitiveString("password"),
		PriceRefreshInterval: 7,
	}

	// The test stage with puzzle auth and no credentials; the rest as configured
	require.Equal(t, []params.MarketDataProxyConfig{{
		StageName:            "test",
		UsePuzzleAuth:        true,
		PriceRefreshInterval: 7,
	}}, fallbackMarketProxyConfigs(prod))

	require.Empty(t, fallbackMarketProxyConfigs(params.MarketDataProxyConfig{StageName: "test"}))

	prod.UrlOverride = security.NewSensitiveString("https://market.example.org")
	require.Empty(t, fallbackMarketProxyConfigs(prod))
}

func TestFallbackNftProxyConfigs(t *testing.T) {
	prod := params.NftProxyConfig{
		StageName: "prod",
		User:      security.NewSensitiveString("user"),
		Password:  security.NewSensitiveString("password"),
	}

	require.Equal(t, []params.NftProxyConfig{{
		StageName:     "test",
		UsePuzzleAuth: true,
	}}, fallbackNftProxyConfigs(prod))

	require.Empty(t, fallbackNftProxyConfigs(params.NftProxyConfig{StageName: "test"}))

	prod.UrlOverride = security.NewSensitiveString("https://nft.example.org")
	require.Empty(t, fallbackNftProxyConfigs(prod))
}
