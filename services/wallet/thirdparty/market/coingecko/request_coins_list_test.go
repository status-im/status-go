package coingecko

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/services/wallet/thirdparty"
)

func TestFetchingTokens(t *testing.T) {
	srv, stop := setupTest(t, responseCoinsListData)
	defer stop()

	geckoClient := &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    srv.URL,
	}

	received, err := geckoClient.FetchTokens(context.Background())
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(coinsList, received))
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
