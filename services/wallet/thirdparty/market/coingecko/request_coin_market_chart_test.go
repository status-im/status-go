package coingecko

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/services/wallet/thirdparty"
)

func TestFetchHistoryMarketData(t *testing.T) {
	expectedData := HistoricalPriceContainer{
		Prices: [][]float64{
			{1737889461591, 0.0418141368281291},
			{1737893063477, 0.0420109710834123},
			{1737896662050, 0.041708338681602},
			{1737900535364, 0.0416724599435818},
			{1737903862165, 0.0417678066849023},
			{1737907469522, 0.0417437321514032},
			{1737911063016, 0.0418807698299526},
			{1737914672131, 0.0419228908628703},
			{1737918272316, 0.0420323651962845},
			{1737921865908, 0.0417301536472736},
		},
	}

	srv, stop := setupTest(t, responseCoinMarketChartData)
	defer stop()

	geckoClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    srv.URL,
	}

	received, err := geckoClient.FetchHistoryMarketData(context.Background(), "status", "usd")
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(expectedData, received))
}

func TestErrorWhenFetchingHistoryMarketData(t *testing.T) {
	srv, stop := setupTest(t, responseError)
	defer stop()

	geckoClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    srv.URL,
	}

	received, err := geckoClient.FetchHistoryMarketData(context.Background(), "status", "usd")
	require.Error(t, err)
	require.Len(t, received.Prices, 0)
}
