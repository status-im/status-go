package leaderboard

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// MarketDataService manages market data fetching and provides access to the latest data
type MarketDataService struct {
	config ServiceConfig
	client *http.Client

	// Request handler
	requestHandler *RequestHandler

	// Data storage
	storage *DataStorage

	// Subscription management
	subscriptionManager *SubscriptionManager

	// Context management
	cancelFunc     context.CancelFunc
	isRunning      bool
	isRunningMutex sync.Mutex
}

// Stats tracks API request statistics
type Stats struct {
	TotalRequests     int
	CacheHits         int
	CacheMisses       int
	TotalResponseSize int64
	NotModifiedCount  int
	GzipResponseCount int
}

// NewMarketDataService creates a new market data service with the given configuration
func NewMarketDataService(config ServiceConfig) *MarketDataService {
	// Set default values for intervals if not provided
	if config.FullDataInterval <= 0 {
		config.FullDataInterval = 10
	}
	if config.PriceUpdateInterval <= 0 {
		config.PriceUpdateInterval = 1
	}

	client := &http.Client{Timeout: 10 * time.Second}
	storage := NewDataStorage()

	return &MarketDataService{
		config:              config,
		client:              client,
		requestHandler:      NewRequestHandler(config, client),
		storage:             storage,
		subscriptionManager: NewSubscriptionManager(),
	}
}

// Start begins the data refresh loops
func (s *MarketDataService) Start(ctx context.Context) {
	s.isRunningMutex.Lock()
	defer s.isRunningMutex.Unlock()

	if s.isRunning {
		return // Already running
	}

	// Create a cancellable context that can stop all data operations
	ctx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel
	s.isRunning = true

	// Start crypto data refresh loop
	go s.cryptoRefreshLoop(ctx)

	// Start price data refresh loop
	go s.priceRefreshLoop(ctx)
}

// Stop halts all data refresh operations
func (s *MarketDataService) Stop() {
	s.isRunningMutex.Lock()
	defer s.isRunningMutex.Unlock()

	if !s.isRunning {
		return // Not running
	}

	// Cancel the context to stop all loops
	if s.cancelFunc != nil {
		s.cancelFunc()
		s.cancelFunc = nil
	}

	s.isRunning = false
}

// IsRunning returns whether the service is currently running
func (s *MarketDataService) IsRunning() bool {
	s.isRunningMutex.Lock()
	defer s.isRunningMutex.Unlock()
	return s.isRunning
}

// Subscribe creates a subscription to data updates
func (s *MarketDataService) Subscribe() chan struct{} {
	return s.subscriptionManager.Subscribe()
}

// Unsubscribe removes a subscription
func (s *MarketDataService) Unsubscribe(ch chan struct{}) {
	s.subscriptionManager.Unsubscribe(ch)
}

// GetCryptoData returns the latest cryptocurrency data
func (s *MarketDataService) GetCryptoData() []Cryptocurrency {
	return s.storage.GetCryptoData()
}

// GetPriceData returns the latest price data
func (s *MarketDataService) GetPriceData() PriceMap {
	return s.storage.GetPriceData()
}

// GetCombinedData returns cryptocurrency data with updated price information
func (s *MarketDataService) GetCombinedData() []Cryptocurrency {
	return s.storage.GetCombinedData()
}

// GetCryptoStats returns statistics for crypto data requests
func (s *MarketDataService) GetCryptoStats() Stats {
	return s.storage.GetCryptoStats()
}

// GetPriceStats returns statistics for price data requests
func (s *MarketDataService) GetPriceStats() Stats {
	return s.storage.GetPriceStats()
}

// cryptoRefreshLoop periodically fetches the full cryptocurrency data
func (s *MarketDataService) cryptoRefreshLoop(ctx context.Context) {
	// Initial fetch
	s.fetchCryptoData(ctx)

	// Notify subscribers of the initial data
	s.subscriptionManager.Emit(ctx)

	// Set up ticker for periodic updates
	ticker := time.NewTicker(time.Duration(s.config.FullDataInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return // Context cancelled, stop the loop
		case <-ticker.C:
			if s.fetchCryptoData(ctx) {
				// Only notify if data was actually updated
				s.subscriptionManager.Emit(ctx)
			}
		}
	}
}

// priceRefreshLoop periodically fetches price updates
func (s *MarketDataService) priceRefreshLoop(ctx context.Context) {
	// Wait a short time before starting price updates
	time.Sleep(1 * time.Second)

	// Initial fetch
	s.fetchPriceData(ctx)

	// Notify subscribers of the initial data
	s.subscriptionManager.Emit(ctx)

	// Set up ticker for periodic updates
	ticker := time.NewTicker(time.Duration(s.config.PriceUpdateInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return // Context cancelled, stop the loop
		case <-ticker.C:
			if s.fetchPriceData(ctx) {
				// Only notify if data was actually updated
				s.subscriptionManager.Emit(ctx)
			}
		}
	}
}

// fetchCryptoData fetches the latest cryptocurrency data
// Returns true if data was updated, false if using cached data (304)
func (s *MarketDataService) fetchCryptoData(ctx context.Context) bool {
	// Get current etag
	cryptoEtag := s.storage.GetCryptoEtag()

	// Fetch data using the request handler
	endpoint := "/v1/cryptocurrency/listings/latest"
	body, updated := s.requestHandler.FetchData(ctx, endpoint, &cryptoEtag, s.storage.GetCryptoStatsRef())
	if !updated {
		return false
	}

	// Update etag if changed
	s.storage.SetCryptoEtag(cryptoEtag)

	// Parse the response
	var cryptoResp CryptoResponse
	if err := json.Unmarshal(body, &cryptoResp); err != nil {
		return false
	}

	// Update the data
	return s.storage.UpdateCryptoData(cryptoResp.Data)
}

// fetchPriceData fetches the latest price data
// Returns true if data was updated, false if using cached data (304)
func (s *MarketDataService) fetchPriceData(ctx context.Context) bool {
	// Get current etag
	priceEtag := s.storage.GetPriceEtag()

	// Fetch data using the request handler
	endpoint := "/v1/prices"
	body, updated := s.requestHandler.FetchData(ctx, endpoint, &priceEtag, s.storage.GetPriceStatsRef())
	if !updated {
		return false
	}

	// Update etag if changed
	s.storage.SetPriceEtag(priceEtag)

	// Parse the response
	var priceData PriceMap
	if err := json.Unmarshal(body, &priceData); err != nil {
		return false
	}

	// Update the data
	return s.storage.UpdatePriceData(priceData)
}
