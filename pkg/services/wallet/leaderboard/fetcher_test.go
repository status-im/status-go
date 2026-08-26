package leaderboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/pkg/security"
)

func TestProxyFetcherFetchPrices(t *testing.T) {
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()
	storage := NewDataStorage(db)

	t.Run("Valid price data with new fields", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			priceResponse := map[string]interface{}{
				"bitcoin": map[string]interface{}{
					"price":              80000.0,
					"market_cap":         1600000000000.0,
					"volume_24h":         80000000000.0,
					"percent_change_24h": 7.5,
				},
				"ethereum": map[string]interface{}{
					"price":              1600.0,
					"market_cap":         195000000000.0,
					"volume_24h":         40000000000.0,
					"percent_change_24h": 10.0,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Etag", "test-etag-123")
			_ = json.NewEncoder(w).Encode(priceResponse)
		}))
		defer server.Close()

		config := ServiceConfig{
			User:                security.NewSensitiveString("test"),
			Password:            security.NewSensitiveString("password"),
			UrlOverride:         security.NewSensitiveString(server.URL),
			AllowGzip:           false,
			AllowETag:           true,
			PriceUpdateInterval: 1 * time.Minute,
			FullDataInterval:    10 * time.Minute,
		}

		fetcher := NewProxyFetcher(config, storage, NewSubscriptionManager()).(*ProxyFetcher)
		err := fetcher.FetchPrices(context.Background())
		require.NoError(t, err)

		priceData := storage.GetPriceData()
		require.Equal(t, 2, len(priceData))

		bitcoin, ok := priceData["bitcoin"]
		require.True(t, ok)
		require.Equal(t, "bitcoin", bitcoin.ID)
		require.Equal(t, 80000.0, bitcoin.Price)
		require.Equal(t, 1600000000000.0, bitcoin.MarketCap)
		require.Equal(t, 80000000000.0, bitcoin.Volume24h)
		require.Equal(t, 7.5, bitcoin.PercentChange24h)

		ethereum, ok := priceData["ethereum"]
		require.True(t, ok)
		require.Equal(t, "ethereum", ethereum.ID)
		require.Equal(t, 1600.0, ethereum.Price)
		require.Equal(t, 195000000000.0, ethereum.MarketCap)
		require.Equal(t, 40000000000.0, ethereum.Volume24h)
		require.Equal(t, 10.0, ethereum.PercentChange24h)

		require.Equal(t, "test-etag-123", storage.GetPriceEtag())
	})

	t.Run("Invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("invalid json{"))
		}))
		defer server.Close()

		config := ServiceConfig{
			User:        security.NewSensitiveString("test"),
			Password:    security.NewSensitiveString("password"),
			UrlOverride: security.NewSensitiveString(server.URL),
			AllowETag:   false,
		}

		fetcher := NewProxyFetcher(config, storage, NewSubscriptionManager()).(*ProxyFetcher)
		err := fetcher.FetchPrices(context.Background())
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to unmarshal price data")
	})

	t.Run("ETag not modified", func(t *testing.T) {
		storage.UpdatePriceDataWithEtag(mockPriceData, "existing-etag", DefaultCurrency)
		oldPriceData := storage.GetPriceData()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("If-None-Match") == "existing-etag" {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		config := ServiceConfig{
			User:        security.NewSensitiveString("test"),
			Password:    security.NewSensitiveString("password"),
			UrlOverride: security.NewSensitiveString(server.URL),
			AllowETag:   true,
		}

		fetcher := NewProxyFetcher(config, storage, NewSubscriptionManager()).(*ProxyFetcher)
		err := fetcher.FetchPrices(context.Background())
		require.NoError(t, err)

		newPriceData := storage.GetPriceData()
		require.Equal(t, len(oldPriceData), len(newPriceData))
	})

	t.Run("Server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal Server Error"))
		}))
		defer server.Close()

		config := ServiceConfig{
			User:        security.NewSensitiveString("test"),
			Password:    security.NewSensitiveString("password"),
			UrlOverride: security.NewSensitiveString(server.URL),
		}

		fetcher := NewProxyFetcher(config, storage, NewSubscriptionManager()).(*ProxyFetcher)
		err := fetcher.FetchPrices(context.Background())
		require.NoError(t, err)
	})
}

func TestProxyFetcherFetchMarkets(t *testing.T) {
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()
	storage := NewDataStorage(db)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		marketResponse := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":                          "bitcoin",
					"symbol":                      "btc",
					"name":                        "Bitcoin",
					"image":                       "https://example.com/bitcoin.png",
					"current_price":               80000.0,
					"market_cap":                  1600000000000.0,
					"total_volume":                80000000000.0,
					"price_change_percentage_24h": 7.5,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Etag", "market-etag-456")
		_ = json.NewEncoder(w).Encode(marketResponse)
	}))
	defer server.Close()

	config := ServiceConfig{
		User:        security.NewSensitiveString("test"),
		Password:    security.NewSensitiveString("password"),
		UrlOverride: security.NewSensitiveString(server.URL),
		AllowETag:   true,
	}

	fetcher := NewProxyFetcher(config, storage, NewSubscriptionManager()).(*ProxyFetcher)
	err := fetcher.FetchMarkets(context.Background())
	require.NoError(t, err)

	cryptoData := storage.GetCryptoData()
	require.Equal(t, 1, len(cryptoData))
	require.Equal(t, "bitcoin", cryptoData[0].ID)
	require.Equal(t, 80000.0, cryptoData[0].CurrentPrice)
	require.Equal(t, "market-etag-456", storage.GetCryptoEtag())
}

func TestProxyFetcherStartStop(t *testing.T) {
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()
	storage := NewDataStorage(db)

	config := ServiceConfig{
		User:                security.NewSensitiveString("test"),
		Password:            security.NewSensitiveString("password"),
		UrlOverride:         security.NewSensitiveString("http://localhost:9999"),
		PriceUpdateInterval: 100 * time.Millisecond,
		FullDataInterval:    200 * time.Millisecond,
	}

	fetcher := NewProxyFetcher(config, storage, NewSubscriptionManager()).(*ProxyFetcher)

	ctx, cancel := context.WithCancel(context.Background())
	fetcher.Start(ctx)

	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(50 * time.Millisecond)

	require.Nil(t, fetcher.cancelFunc)
}

func TestProxyFetcherRefreshLoops(t *testing.T) {
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()
	storage := NewDataStorage(db)

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == PRICES_ENDPOINT {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
		} else {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
		}
	}))
	defer server.Close()

	config := ServiceConfig{
		User:                security.NewSensitiveString("test"),
		Password:            security.NewSensitiveString("password"),
		UrlOverride:         security.NewSensitiveString(server.URL),
		PriceUpdateInterval: 100 * time.Millisecond,
		FullDataInterval:    100 * time.Millisecond,
	}

	fetcher := NewProxyFetcher(config, storage, NewSubscriptionManager()).(*ProxyFetcher)

	fetcher.StartRefreshLoops()
	time.Sleep(350 * time.Millisecond)
	fetcher.Stop()

	require.Greater(t, requestCount, 0)
}
