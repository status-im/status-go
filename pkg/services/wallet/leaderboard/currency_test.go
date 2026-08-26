package leaderboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/pkg/security"
)

func testFetcherConfig(url string) ServiceConfig {
	return ServiceConfig{
		User:        security.NewSensitiveString("test"),
		Password:    security.NewSensitiveString("password"),
		UrlOverride: security.NewSensitiveString(url),
		AllowETag:   true,
	}
}

func writeMarketsResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"data": []map[string]interface{}{
			{"id": "bitcoin", "symbol": "btc", "current_price": 80000.0},
		},
	})
}

// recordingServer collects the convert_currency values the fetcher sent.
type recordingServer struct {
	mu       sync.Mutex
	queries  []string
	reject   map[string]bool
	server   *httptest.Server
	requests int
}

func newRecordingServer(reject ...string) *recordingServer {
	rs := &recordingServer{reject: make(map[string]bool)}
	for _, currency := range reject {
		rs.reject[currency] = true
	}
	rs.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		convert := r.URL.Query().Get(convertCurrencyParam)

		rs.mu.Lock()
		rs.queries = append(rs.queries, convert)
		rs.requests++
		reject := rs.reject[convert]
		rs.mu.Unlock()

		if reject {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"unsupported convert_currency: ` + convert + `"}`))
			return
		}
		writeMarketsResponse(w)
	}))
	return rs
}

func (rs *recordingServer) sentQueries() []string {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return append([]string{}, rs.queries...)
}

func TestFetcherSendsConvertCurrency(t *testing.T) {
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()

	t.Run("non-USD currency is converted by the proxy", func(t *testing.T) {
		rs := newRecordingServer()
		defer rs.server.Close()

		storage := NewDataStorage(db)
		require.True(t, storage.SetCurrency("EUR"))

		fetcher := NewProxyFetcher(testFetcherConfig(rs.server.URL), storage, NewSubscriptionManager()).(*ProxyFetcher)
		require.NoError(t, fetcher.FetchMarkets(context.Background()))

		require.Equal(t, []string{"eur"}, rs.sentQueries())
	})

	t.Run("USD needs no conversion", func(t *testing.T) {
		rs := newRecordingServer()
		defer rs.server.Close()

		storage := NewDataStorage(db)
		fetcher := NewProxyFetcher(testFetcherConfig(rs.server.URL), storage, NewSubscriptionManager()).(*ProxyFetcher)
		require.NoError(t, fetcher.FetchMarkets(context.Background()))

		require.Equal(t, []string{""}, rs.sentQueries())
	})
}

func TestFetcherFallsBackWhenCurrencyIsRejected(t *testing.T) {
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()

	rs := newRecordingServer("xyz")
	defer rs.server.Close()

	storage := NewDataStorage(db)
	require.True(t, storage.SetCurrency("xyz"))

	fetcher := NewProxyFetcher(testFetcherConfig(rs.server.URL), storage, NewSubscriptionManager()).(*ProxyFetcher)

	// The rejected request is retried once without the conversion, so the tab
	// gets data at all, in USD.
	require.NoError(t, fetcher.FetchMarkets(context.Background()))
	require.Equal(t, []string{"xyz", ""}, rs.sentQueries())
	require.Equal(t, 1, storage.GetCryptoDataSize())

	// Later fetches do not ask for the rejected currency again.
	require.NoError(t, fetcher.FetchMarkets(context.Background()))
	require.Equal(t, []string{"xyz", "", ""}, rs.sentQueries())
}

func TestFetcherKeepsRetryingOnUnrelatedErrors(t *testing.T) {
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer server.Close()

	storage := NewDataStorage(db)
	require.True(t, storage.SetCurrency("eur"))

	fetcher := NewProxyFetcher(testFetcherConfig(server.URL), storage, NewSubscriptionManager()).(*ProxyFetcher)
	require.NoError(t, fetcher.FetchMarkets(context.Background()))

	// A server error must not disable the currency.
	require.Equal(t, "eur", fetcher.convertCurrencyFor("eur"))
}

func TestSetCurrencyInvalidatesCachedData(t *testing.T) {
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()

	storage := NewDataStorage(db)
	storage.UpdateCryptoDataWithEtag(mockCrypto, "crypto-etag", DefaultCurrency)
	storage.UpdatePriceDataWithEtag(mockPriceData, "price-etag", DefaultCurrency)

	require.False(t, storage.IsDataStale())

	require.True(t, storage.SetCurrency("eur"))

	require.Equal(t, "eur", storage.GetCurrency())
	require.Empty(t, storage.GetCryptoEtag())
	require.Empty(t, storage.GetPriceEtag())
	require.Equal(t, 0, storage.GetCryptoDataSize())
	require.Empty(t, storage.GetPriceData())
	require.True(t, storage.IsDataStale())

	// The persisted snapshot of the previous currency is gone too.
	persisted, err := NewPersistance(db).GetCryptocurrencies(DefaultCurrency)
	require.NoError(t, err)
	require.Empty(t, persisted)
}

func TestSetCurrencyIsCaseInsensitive(t *testing.T) {
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()

	storage := NewDataStorage(db)
	require.True(t, storage.SetCurrency("EUR"))
	storage.UpdateCryptoDataWithEtag(mockCrypto, "crypto-etag", "eur")

	// status-desktop sends the code uppercased, the settings DB stores it
	// lowercased - that must not count as a change.
	require.False(t, storage.SetCurrency("eur"))
	require.Equal(t, "crypto-etag", storage.GetCryptoEtag())
}

func TestStartRestoresOnlyMatchingCurrency(t *testing.T) {
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()

	require.NoError(t, NewPersistance(db).UpsertCryptocurrencies(mockCrypto, DefaultCurrency))

	t.Run("same currency is restored", func(t *testing.T) {
		storage := NewDataStorage(db)
		storage.Start()
		require.Equal(t, len(mockCrypto), storage.GetCryptoDataSize())
	})

	t.Run("other currency is a cache miss", func(t *testing.T) {
		storage := NewDataStorage(db)
		storage.currency = "eur"
		storage.Start()
		require.Equal(t, 0, storage.GetCryptoDataSize())
	})
}

func TestPersistenceIsScopedToCurrency(t *testing.T) {
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()

	persistence := NewPersistance(db)
	require.NoError(t, persistence.UpsertCryptocurrencies(mockCrypto, "eur"))

	stored, err := persistence.GetCryptocurrencies("eur")
	require.NoError(t, err)
	require.Equal(t, len(mockCrypto), len(stored))

	stored, err = persistence.GetCryptocurrencies(DefaultCurrency)
	require.NoError(t, err)
	require.Empty(t, stored)

	require.NoError(t, persistence.DeleteCryptocurrenciesNotIn(DefaultCurrency))
	stored, err = persistence.GetCryptocurrencies("eur")
	require.NoError(t, err)
	require.Empty(t, stored)
}

func TestFetchLeaderboardPageAppliesRequestedCurrency(t *testing.T) {
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()

	service := setupMarketDatadService(t, ServiceConfig{}, db)
	service.storage.UpdateCryptoDataWithEtag(mockCrypto, "crypto-etag", DefaultCurrency)

	service.setCurrency("EUR", false)

	require.Equal(t, "eur", service.storage.GetCurrency())
	require.Empty(t, service.storage.GetCryptoEtag())
	require.True(t, service.storage.IsDataStale())
}
