package lifi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIntegratorForStage(t *testing.T) {
	require.Equal(t, IntegratorProd, IntegratorForStage("prod"))
	require.Equal(t, IntegratorDev, IntegratorForStage("test"))
	require.Equal(t, IntegratorDev, IntegratorForStage(""))
	require.Equal(t, IntegratorDev, IntegratorForStage("staging"))
}

// Keyed rate limits only apply to requests presenting the API key header, so
// every endpoint has to carry it — not just /quote.
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
				headerByPath[r.URL.Path] = r.Header.Get("x-lifi-api-key")
				mu.Unlock()
				_, err := w.Write([]byte("{}"))
				require.NoError(t, err)
			}))
			defer srv.Close()

			client := NewClient(1, IntegratorDev, tc.apiKey)
			client.baseURL = srv.URL

			_, err := client.FetchTokensList(context.Background())
			require.NoError(t, err)

			_, err = client.FetchQuote(context.Background(), QuoteParams{})
			require.NoError(t, err)

			for _, path := range []string{"/tokens", "/quote"} {
				got, requested := headerByPath[path]
				require.True(t, requested, "%s was not requested", path)
				require.Equal(t, tc.apiKey, got, "unexpected x-lifi-api-key on %s", path)
			}
		})
	}
}
