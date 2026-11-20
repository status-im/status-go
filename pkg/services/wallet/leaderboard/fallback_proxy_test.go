package leaderboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/pkg/security"
	"github.com/status-im/status-go/pkg/services/wallet/puzzleauth/puzzleauthtest"
)

func TestFallbackProxyConfigs(t *testing.T) {
	prod := ServiceConfig{
		StageName: "prod",
		User:      security.NewSensitiveString("user"),
		Password:  security.NewSensitiveString("password"),
		AllowGzip: true,
	}

	// The test stage with puzzle auth and no credentials; the rest as configured
	require.Equal(t, []ServiceConfig{{
		StageName:     "test",
		UsePuzzleAuth: true,
		AllowGzip:     true,
	}}, fallbackProxyConfigs(prod))

	require.Empty(t, fallbackProxyConfigs(ServiceConfig{StageName: "test"}))

	prod.UrlOverride = security.NewSensitiveString("https://market.example.org")
	require.Empty(t, fallbackProxyConfigs(prod))
}

func TestNewProxyFetcherProxies(t *testing.T) {
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()
	storage := NewDataStorage(db)

	hosts := func(config ServiceConfig) []string {
		var out []string
		for _, proxy := range NewProxyFetcher(config, storage, NewSubscriptionManager()).(*ProxyFetcher).proxies {
			out = append(out, proxy.host)
		}
		return out
	}

	// The fallback goes last
	require.Equal(t,
		[]string{"https://prod.market.status.im", "https://test.market.status.im"},
		hosts(ServiceConfig{StageName: "prod"}))
	require.Equal(t, []string{"https://test.market.status.im"}, hosts(ServiceConfig{StageName: "test"}))
}

// proxyStub is a market proxy that answers every request with the given
// status and counts the requests it got.
type proxyStub struct {
	*httptest.Server
	requests atomic.Int32
}

func newProxyStub(t *testing.T, status int, body string) *proxyStub {
	stub := &proxyStub{}
	stub.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		stub.requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(stub.Close)
	return stub
}

// newFetcherWithProxies returns a fetcher that asks the proxies at the given
// addresses in order, and its storage.
func newFetcherWithProxies(t *testing.T, urls ...string) (*ProxyFetcher, *DataStorage) {
	db, cleanup := setupTestWalletDB(t)
	t.Cleanup(cleanup)
	storage := NewDataStorage(db)

	fetcher := NewProxyFetcher(ServiceConfig{}, storage, NewSubscriptionManager()).(*ProxyFetcher)
	fetcher.proxies = nil
	for _, url := range urls {
		fetcher.proxies = append(fetcher.proxies, newMarketProxy(ServiceConfig{UrlOverride: security.NewSensitiveString(url)}))
	}
	return fetcher, storage
}

const (
	mainPrices     = `{"bitcoin":{"price":1.0}}`
	fallbackPrices = `{"bitcoin":{"price":2.0}}`
)

func TestProxyFetcherAsksTheNextProxy(t *testing.T) {
	t.Run("not while the first one answers", func(t *testing.T) {
		main := newProxyStub(t, http.StatusOK, mainPrices)
		fallback := newProxyStub(t, http.StatusOK, fallbackPrices)
		fetcher, storage := newFetcherWithProxies(t, main.URL, fallback.URL)

		require.NoError(t, fetcher.FetchPrices(context.Background()))

		require.Equal(t, 1.0, storage.GetPriceData()["bitcoin"].Price)
		require.Zero(t, fallback.requests.Load())
	})

	t.Run("when the first one answers with an error", func(t *testing.T) {
		main := newProxyStub(t, http.StatusInternalServerError, "down")
		fallback := newProxyStub(t, http.StatusOK, fallbackPrices)
		fetcher, storage := newFetcherWithProxies(t, main.URL, fallback.URL)

		require.NoError(t, fetcher.FetchPrices(context.Background()))

		require.Equal(t, 2.0, storage.GetPriceData()["bitcoin"].Price)
	})

	t.Run("when the first one is unreachable", func(t *testing.T) {
		main := newProxyStub(t, http.StatusOK, mainPrices)
		fallback := newProxyStub(t, http.StatusOK, fallbackPrices)
		fetcher, storage := newFetcherWithProxies(t, main.URL, fallback.URL)
		main.Close()

		require.NoError(t, fetcher.FetchPrices(context.Background()))

		require.Equal(t, 2.0, storage.GetPriceData()["bitcoin"].Price)
	})

	t.Run("not for a request the first one rejects as malformed", func(t *testing.T) {
		main := newProxyStub(t, http.StatusBadRequest, `{"error":"bad request"}`)
		fallback := newProxyStub(t, http.StatusOK, fallbackPrices)
		fetcher, storage := newFetcherWithProxies(t, main.URL, fallback.URL)

		require.NoError(t, fetcher.FetchPrices(context.Background()))

		require.Zero(t, fallback.requests.Load())
		require.Empty(t, storage.GetPriceData())
	})
}

// A fallback proxy gets a puzzle token even when the main one is reached with
// basic credentials.
func TestProxyFetcherFallbackUsesPuzzleAuth(t *testing.T) {
	main := newProxyStub(t, http.StatusInternalServerError, "down")
	var authorization string
	fallback := puzzleauthtest.NewServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fallbackPrices))
	}))

	mainConfig := ServiceConfig{
		UrlOverride: security.NewSensitiveString(main.URL),
		User:        security.NewSensitiveString("user"),
		Password:    security.NewSensitiveString("password"),
	}
	// What fallbackProxyConfigs makes of a config, at the test server's address
	fallbackConfig := ServiceConfig{UrlOverride: security.NewSensitiveString(fallback.URL), UsePuzzleAuth: true}

	fetcher, storage := newFetcherWithProxies(t)
	fetcher.proxies = []marketProxy{newMarketProxy(mainConfig), newMarketProxy(fallbackConfig)}

	require.NoError(t, fetcher.FetchPrices(context.Background()))

	require.Equal(t, 2.0, storage.GetPriceData()["bitcoin"].Price)
	require.Equal(t, "Bearer "+puzzleauthtest.Token, authorization)
}
