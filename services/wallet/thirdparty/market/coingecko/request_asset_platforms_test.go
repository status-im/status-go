package coingecko

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/services/wallet/thirdparty"
)

func TestFetchingPlatforms(t *testing.T) {
	expectedData := []Chain{
		{
			ID:           "valobit",
			ChainID:      0,
			Name:         "Valobit",
			NativeCoinID: "valobit",
		},
		{
			ID:           "factom",
			ChainID:      0,
			Name:         "Factom",
			NativeCoinID: "factom",
		},
		{
			ID:           "ethereum",
			ChainID:      1,
			Name:         "Ethereum",
			NativeCoinID: "ethereum",
		},
		{
			ID:           "optimistic-ethereum",
			ChainID:      10,
			Name:         "Optimism",
			NativeCoinID: "ethereum",
		},
		{
			ID:           "arbitrum-one",
			ChainID:      42161,
			Name:         "Arbitrum One",
			NativeCoinID: "ethereum",
		},
		{
			ID:           "base",
			ChainID:      8453,
			Name:         "Base",
			NativeCoinID: "ethereum",
		},
		{
			ID:           "trustless-computer",
			ChainID:      0,
			Name:         "Trustless Computer",
			NativeCoinID: "bitcoin",
		},
		{
			ID:           "ordinals",
			ChainID:      0,
			Name:         "Bitcoin",
			NativeCoinID: "bitcoin",
		},
		{
			ID:           "solana",
			ChainID:      0,
			Name:         "Solana",
			NativeCoinID: "solana",
		},
	}

	srv, stop := setupTest(t, responseAssetPlatformsData)
	defer stop()

	geckoClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    srv.URL,
	}

	received, err := geckoClient.FetchPlatforms(context.Background())
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(expectedData, received))
}

func TestErrorWhenFetchingPlatforms(t *testing.T) {
	srv, stop := setupTest(t, responseError)
	defer stop()

	geckoClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    srv.URL,
	}

	received, err := geckoClient.FetchPlatforms(context.Background())
	require.Error(t, err)
	require.Nil(t, received)
}
