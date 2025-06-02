package coingecko

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/services/wallet/thirdparty"
)

func TestFetchingCoinsMarketsData(t *testing.T) {
	expectedData := []GeckoMarketValues{
		{
			ID:                                "ethereum",
			Symbol:                            "eth",
			Name:                              "Ethereum",
			MarketCap:                         292522384022,
			High24h:                           2691.69,
			Low24h:                            2337.39,
			PriceChange24h:                    -240.706759028237,
			PriceChangePercentage24h:          -9.00467,
			PriceChangePercentage1hInCurrency: -0.145646222497177,
		},
		{
			ID:                                "status",
			Symbol:                            "snt",
			Name:                              "Status",
			MarketCap:                         103150993,
			High24h:                           0.02934234,
			Low24h:                            0.02488445,
			PriceChange24h:                    -0.00310646933671397,
			PriceChangePercentage24h:          -10.66568,
			PriceChangePercentage1hInCurrency: -0.253535324764873,
		},
	}

	srv, stop := setupTest(t, responseCoinsMarketsData)
	defer stop()

	geckoClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    srv.URL,
	}

	received, err := geckoClient.FetchCoinsMarkets(context.Background(), []string{"status", "ethereum"}, "usd")
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(expectedData, received))
}

func TestErrorWhenFetchingCoinsMarketsData(t *testing.T) {
	srv, stop := setupTest(t, responseError)
	defer stop()

	geckoClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    srv.URL,
	}

	received, err := geckoClient.FetchCoinsMarkets(context.Background(), []string{"status", "ethereum"}, "usd")
	require.Error(t, err)
	require.Len(t, received, 0)
}
