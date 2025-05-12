package leaderboard

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/logutils"
)

// DataFetcher defines the interface for fetching market and price data
type DataFetcher interface {
	// FetchMarkets fetches the full market data
	FetchMarkets(ctx context.Context) error
	// FetchPrices fetches the latest price data
	FetchPrices(ctx context.Context) error
	// StartRefreshLoops starts the data refresh loops
	StartRefreshLoops()
	// Start begins the data refresh loops
	Start(ctx context.Context)
	// Stop halts all data refresh operations
	Stop()
}

// ProxyFetcher implements DataFetcher interface using HTTP proxy
type ProxyFetcher struct {
	requestHandler      *RequestHandler
	storage             *DataStorage
	subscriptionManager *SubscriptionManager
	config              ServiceConfig

	// Background polling state
	isRunning      bool
	isRunningMutex sync.Mutex
	cancelFunc     context.CancelFunc
	ctx            context.Context
}

// NewProxyFetcher creates a new proxy data fetcher
func NewProxyFetcher(config ServiceConfig, storage *DataStorage, subscriptionManager *SubscriptionManager) DataFetcher {
	client := &http.Client{Timeout: 10 * time.Second}
	return &ProxyFetcher{
		requestHandler:      NewRequestHandler(config, client),
		storage:             storage,
		subscriptionManager: subscriptionManager,
		config:              config,
		isRunning:           false,
	}
}

// Start begins the data refresh loops
func (f *ProxyFetcher) Start(ctx context.Context) {
	f.isRunningMutex.Lock()
	defer f.isRunningMutex.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	f.cancelFunc = cancel
	f.ctx = ctx

	go func() {
		defer common.LogOnPanic()
		<-ctx.Done()
		f.Stop() // gracefully stop if running
		f.cancelFunc = nil
		f.ctx = nil
	}()
}

// Stop halts all data refresh operations
func (f *ProxyFetcher) Stop() {
	f.isRunningMutex.Lock()
	defer f.isRunningMutex.Unlock()

	if !f.isRunning {
		return // Not running
	}

	// Cancel the context to stop all loops
	if f.cancelFunc != nil {
		f.cancelFunc()
	}

	f.isRunning = false
}

func (f *ProxyFetcher) StartRefreshLoops() {
	f.isRunningMutex.Lock()
	defer f.isRunningMutex.Unlock()

	if f.isRunning || f.ctx == nil {
		return
	}
	f.isRunning = true

	// Start crypto data refresh loop
	go func() {
		defer common.LogOnPanic()
		f.cryptoRefreshLoop(f.ctx)
	}()

	// Start price data refresh loop
	go func() {
		defer common.LogOnPanic()
		f.priceRefreshLoop(f.ctx)
	}()
}

// cryptoRefreshLoop periodically fetches the full cryptocurrency data
func (f *ProxyFetcher) cryptoRefreshLoop(ctx context.Context) {
	// Set up ticker for periodic updates
	ticker := time.NewTicker(time.Duration(f.config.FullDataInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return // Context cancelled, stop the loop
		case <-ticker.C:
			if err := f.FetchMarkets(ctx); err != nil {
				logutils.ZapLogger().Error("Error fetching crypto data", zap.Error(err))
			} else {
				f.subscriptionManager.Emit(ctx, TickerFullDataUpdateSource)
			}
		}
	}
}

// priceRefreshLoop periodically fetches price updates
func (f *ProxyFetcher) priceRefreshLoop(ctx context.Context) {
	// Wait a short time before starting price updates
	time.Sleep(1 * time.Second)

	// Set up ticker for periodic updates
	ticker := time.NewTicker(time.Duration(f.config.PriceUpdateInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return // Context cancelled, stop the loop
		case <-ticker.C:
			if err := f.FetchPrices(ctx); err != nil {
				logutils.ZapLogger().Error("Error fetching price data", zap.Error(err))
			} else {
				f.subscriptionManager.Emit(ctx, TickerPriceUpdateSource)
			}
		}
	}
}

// FetchMarkets fetches the full market data
func (f *ProxyFetcher) FetchMarkets(ctx context.Context) error {
	endpoint := "/v1/leaderboard/markets"
	etag := f.storage.GetCryptoEtag()

	body, updated := f.requestHandler.FetchData(ctx, endpoint, &etag)
	if !updated {
		return nil
	}

	var data CryptoResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return err
	}

	// Store data and etag atomically
	f.storage.UpdateCryptoDataWithEtag(data.Data, etag)

	return nil
}

// FetchPrices fetches the latest price data
func (f *ProxyFetcher) FetchPrices(ctx context.Context) error {
	endpoint := "/v1/leaderboard/prices"
	etag := f.storage.GetPriceEtag()

	body, updated := f.requestHandler.FetchData(ctx, endpoint, &etag)
	if !updated {
		return nil
	}

	var priceData PriceMap
	if err := json.Unmarshal(body, &priceData); err != nil {
		return err
	}

	// Store data and etag atomically
	f.storage.UpdatePriceDataWithEtag(priceData, etag)

	return nil
}
