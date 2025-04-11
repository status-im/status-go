package leaderboard

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// DataFetcher defines the interface for fetching market and price data
type DataFetcher interface {
	// FetchMarkets fetches the full market data
	FetchMarkets(ctx context.Context) (*FetchResult[[]Cryptocurrency], error)
	// FetchPrices fetches the latest price data
	FetchPrices(ctx context.Context) (*FetchResult[PriceMap], error)
}

// FetchResult represents the result of a fetch operation
type FetchResult[T any] struct {
	Updated bool
	Data    T
	ETag    string
}

// ProxyFetcher implements DataFetcher interface using HTTP proxy
type ProxyFetcher struct {
	requestHandler *RequestHandler
	storage        *DataStorage
}

// NewProxyFetcher creates a new proxy data fetcher
func NewProxyFetcher(config ServiceConfig, storage *DataStorage) DataFetcher {
	client := &http.Client{Timeout: 10 * time.Second}
	return &ProxyFetcher{
		requestHandler: NewRequestHandler(config, client),
		storage:        storage,
	}
}

// FetchMarkets fetches the full market data
func (f *ProxyFetcher) FetchMarkets(ctx context.Context) (*FetchResult[[]Cryptocurrency], error) {
	endpoint := "/v1/leaderboard/markets"
	etag := f.storage.GetCryptoEtag()

	body, updated := f.requestHandler.FetchData(ctx, endpoint, &etag)
	if !updated {
		return &FetchResult[[]Cryptocurrency]{Updated: false}, nil
	}

	var data CryptoResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	// Store data and etag atomically
	f.storage.UpdateCryptoDataWithEtag(data.Data, etag)

	return &FetchResult[[]Cryptocurrency]{
		Updated: true,
		Data:    data.Data,
		ETag:    etag,
	}, nil
}

// FetchPrices fetches the latest price data
func (f *ProxyFetcher) FetchPrices(ctx context.Context) (*FetchResult[PriceMap], error) {
	endpoint := "/v1/leaderboard/prices"
	etag := f.storage.GetPriceEtag()

	body, updated := f.requestHandler.FetchData(ctx, endpoint, &etag)
	if !updated {
		return &FetchResult[PriceMap]{Updated: false}, nil
	}

	var priceData PriceMap
	if err := json.Unmarshal(body, &priceData); err != nil {
		return nil, err
	}

	// Store data and etag atomically
	f.storage.UpdatePriceDataWithEtag(priceData, etag)

	return &FetchResult[PriceMap]{
		Updated: true,
		Data:    priceData,
		ETag:    etag,
	}, nil
}
