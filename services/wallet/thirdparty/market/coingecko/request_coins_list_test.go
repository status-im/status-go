package coingecko

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/services/wallet/thirdparty"
)

func TestFetchingTokens(t *testing.T) {
	expectedData := []GeckoToken{
		{
			ID:          "valobit",
			Symbol:      "vbit",
			Name:        "VALOBIT",
			EthPlatform: false,
		},
		{
			ID:          "dmx",
			Symbol:      "dmx",
			Name:        "DMX",
			EthPlatform: false,
		},
		{
			ID:          "pedro",
			Symbol:      "pedro",
			Name:        "PEDRO",
			EthPlatform: false,
		},
		{
			ID:          "volta-club",
			Symbol:      "volta",
			Name:        "Volta Club",
			EthPlatform: true,
		},
		{
			ID:          "don-t-sell-your-bitcoin",
			Symbol:      "bitcoin",
			Name:        "DON'T SELL YOUR BITCOIN",
			EthPlatform: false,
		},
		{
			ID:          "harrypotterobamasonic10in",
			Symbol:      "bitcoin",
			Name:        "HarryPotterObamaSonic10Inu (ETH)",
			EthPlatform: true,
		},
		{
			ID:          "dork-lord-coin",
			Symbol:      "dlord",
			Name:        "DORK LORD COIN",
			EthPlatform: false,
		},
		{
			ID:          "dork-lord-eth",
			Symbol:      "dorkl",
			Name:        "DORK LORD (SOL)",
			EthPlatform: false,
		},
		{
			ID:          "dotcom",
			Symbol:      "y2k",
			Name:        "Dotcom",
			EthPlatform: false,
		},
		{
			ID:          "sentinel-bot-ai",
			Symbol:      "snt",
			Name:        "Sentinel Bot Ai",
			EthPlatform: true,
		},
		{
			ID:          "status",
			Symbol:      "snt",
			Name:        "Status",
			EthPlatform: true,
		},
		{
			ID:          "usd-coin",
			Symbol:      "usdc",
			Name:        "USDC",
			EthPlatform: true,
		},
		{
			ID:          "bridged-usdt",
			Symbol:      "usdt",
			Name:        "Bridged USDT",
			EthPlatform: false,
		},
	}

	srv, stop := setupTest(t, responseCoinsListData)
	defer stop()

	geckoClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    srv.URL,
	}

	received, err := geckoClient.FetchTokens(context.Background())
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(expectedData, received))
}

func TestErrorWhenFetchingTokens(t *testing.T) {
	srv, stop := setupTest(t, responseError)
	defer stop()

	geckoClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    srv.URL,
	}

	received, err := geckoClient.FetchTokens(context.Background())
	require.Error(t, err)
	require.Nil(t, received)
}
