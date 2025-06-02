package coingecko

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/services/wallet/thirdparty"
)

func TestFetchingSimplePriceData(t *testing.T) {
	expectedData := SymbolPriceMap{
		"ethereum": {
			"usd": 2419.65,
			"eur": 2304.62,
		},
		"status": {
			"usd": 0.02597611,
			"eur": 0.02474128,
		},
	}

	srv, stop := setupTest(t, responseSimplePriceData)
	defer stop()

	geckoClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    srv.URL,
	}

	received, err := geckoClient.FetchSimplePrice(context.Background(), []string{"status", "ethereum"}, []string{"usd", "eur"})
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(expectedData, received))
}

func TestErrorWhenFetchingSimplePriceData(t *testing.T) {
	srv, stop := setupTest(t, responseError)
	defer stop()

	geckoClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    srv.URL,
	}

	received, err := geckoClient.FetchSimplePrice(context.Background(), []string{"status", "ethereum"}, []string{"usd", "eur"})
	require.Error(t, err)
	require.Len(t, received, 0)
}
