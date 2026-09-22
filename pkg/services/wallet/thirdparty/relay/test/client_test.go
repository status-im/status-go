package relay_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty/relay"
)

func TestReferrerForStage(t *testing.T) {
	require.Equal(t, relay.ReferrerProd, relay.ReferrerForStage("prod"))
	require.Equal(t, relay.ReferrerDev, relay.ReferrerForStage("test"))
	require.Equal(t, relay.ReferrerDev, relay.ReferrerForStage(""))
	require.Equal(t, relay.ReferrerDev, relay.ReferrerForStage("staging"))
}

// Keyed rate limits only apply to requests presenting the API key header, so
// every endpoint has to carry it — not just /quote/v2.
func TestApiKeyHeaderAppliedToAllEndpoints(t *testing.T) {
	for _, tc := range []struct {
		name   string
		apiKey string
	}{
		{name: "with api key", apiKey: "test-api-key"},
		{name: "without api key", apiKey: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			headerByPath := map[string]string{}

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				headerByPath[r.URL.Path] = r.Header.Get("x-api-key")
				mu.Unlock()
				_, err := w.Write([]byte("{}"))
				require.NoError(t, err)
			}))
			defer srv.Close()

			client := relay.NewClientWithBaseURL(srv.URL, walletCommon.EthereumMainnet, relay.ReferrerDev, tc.apiKey)

			_, err := client.FetchChains(context.Background())
			require.NoError(t, err)

			_, err = client.FetchQuote(context.Background(), relay.QuoteParams{})
			require.NoError(t, err)

			for _, path := range []string{"/chains", "/quote/v2"} {
				got, requested := headerByPath[path]
				require.True(t, requested, "%s was not requested", path)
				require.Equal(t, tc.apiKey, got, "unexpected x-api-key on %s", path)
			}
		})
	}
}

func TestSetChainIDPicksTestnetBaseURL(t *testing.T) {
	client := relay.NewClient(walletCommon.EthereumMainnet, relay.ReferrerDev, "")
	require.Equal(t, relay.MainnetBaseURL, client.BaseURL())

	client.SetChainID(walletCommon.EthereumSepolia)
	require.Equal(t, relay.TestnetBaseURL, client.BaseURL())

	client.SetChainID(walletCommon.OptimismMainnet)
	require.Equal(t, relay.MainnetBaseURL, client.BaseURL())
}

func TestExplicitBaseURLIsKeptAcrossChains(t *testing.T) {
	client := relay.NewClientWithBaseURL("http://relay.local", walletCommon.EthereumMainnet, relay.ReferrerDev, "")
	require.Equal(t, "http://relay.local", client.BaseURL())

	client.SetChainID(walletCommon.EthereumSepolia)
	require.Equal(t, "http://relay.local", client.BaseURL())
}
