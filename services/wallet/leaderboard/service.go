package leaderboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

const (
	// Contains a LeaderboardPage payload
	EventFetchLeaderboardPageDone walletevent.EventType = "wallet-leaderboard-fetch-page-done"
	// Contains a LeaderboardPage payload
	EvenLeaderboardPageDone walletevent.EventType = "wallet-leaderboard-page-updated"
	// Contains a EventLeaderboardPagePricesUpdate payload
	EventLeaderboardPagePricesUpdated walletevent.EventType = "wallet-leaderboard-page-prices-updated"

	// Signal source
	TickerFullDataUpdateSource int = 0
	TickerPriceUpdateSource    int = 1
)

var (
	fetchLeaderboardPageTask = async.TaskType{
		ID:     1,
		Policy: async.ReplacementPolicyCancelOld,
	}
)

// MarketDataService manages market data fetching and provides access to the latest data
type MarketDataService struct {
	config    ServiceConfig
	client    *http.Client
	scheduler *async.Scheduler
	feed      *event.Feed

	// Request handler
	requestHandler *RequestHandler

	// Data storage
	storage *DataStorage
	cache   *PageCache

	// Subscription management
	subscriptionManager    *SubscriptionManager
	pageUpdateSubscription chan Signal

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
func NewMarketDataService(config ServiceConfig, feed *event.Feed) *MarketDataService {
	// Set default values for intervals if not provided
	client := &http.Client{Timeout: 10 * time.Second}
	storage := NewDataStorage()

	return &MarketDataService{
		config:                 config,
		client:                 client,
		feed:                   feed,
		requestHandler:         NewRequestHandler(config, client),
		storage:                storage,
		subscriptionManager:    NewSubscriptionManager(),
		scheduler:              async.NewScheduler(),
		pageUpdateSubscription: nil,
		cache:                  NewPageCache(),
	}
}

// Start begins the data refresh loops
func (s *MarketDataService) Start(ctx context.Context) {
	s.isRunningMutex.Lock()
	defer s.isRunningMutex.Unlock()

	if s.isRunning {
		return // Already running
	}

	// TODO_ES optimize to stop fetching data when not needed
	// Create a cancellable context that can stop all data operations
	ctx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel
	s.isRunning = true

	// Start crypto data refresh loop
	go func() {
		defer common.LogOnPanic()
		s.cryptoRefreshLoop(ctx)
	}()

	// Start price data refresh loop
	go func() {
		defer common.LogOnPanic()
		s.priceRefreshLoop(ctx)
	}()
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

	s.UnsubscribeFromLeaderboard() //nolint:errcheck

	s.isRunning = false
}

// IsRunning returns whether the service is currently running
func (s *MarketDataService) IsRunning() bool {
	s.isRunningMutex.Lock()
	defer s.isRunningMutex.Unlock()
	return s.isRunning
}

func (s *MarketDataService) isSubscribed() bool {
	return s.pageUpdateSubscription != nil
}

// cryptoRefreshLoop periodically fetches the full cryptocurrency data
func (s *MarketDataService) cryptoRefreshLoop(ctx context.Context) {
	// Initial fetch
	s.fetchCryptoData(ctx)

	// Notify subscribers of the initial data
	s.subscriptionManager.Emit(ctx, TickerFullDataUpdateSource)

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
				s.subscriptionManager.Emit(ctx, TickerFullDataUpdateSource)
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
	s.subscriptionManager.Emit(ctx, TickerPriceUpdateSource)

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
				s.subscriptionManager.Emit(ctx, TickerPriceUpdateSource)
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
	endpoint := "/v1/leaderboard/markets"
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
	endpoint := "/v1/leaderboard/prices"
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

func (s *MarketDataService) sendLeaderboardPagePricesUpdate() {
	if !s.isSubscribed() {
		return
	}

	result := s.storage.GetLeaderboardPagePrices(s.cache.GetLastPage())
	if result == nil {
		logutils.ZapLogger().Error("No leaderboard page prices found")
		return
	}
	payload, err := json.Marshal(result)
	if err != nil {
		logutils.ZapLogger().Error("Error marshalling leaderboard page prices", zap.Error(err))
	}

	event := walletevent.Event{
		Type:    EventLeaderboardPagePricesUpdated,
		Message: string(payload),
	}
	s.feed.Send(event)
}

func (s *MarketDataService) sendLeaderboardPageUpdate() {
	if !s.isSubscribed() {
		return
	}

	lastPage := s.cache.GetLastPage()
	result, err := s.storage.GetLeaderboardPage(lastPage.Page, lastPage.PageSize, lastPage.SortOrder, lastPage.Currency)
	if err != nil {
		logutils.ZapLogger().Error("Error fetching leaderboard page", zap.Error(err))
		return
	}
	payload, err := json.Marshal(result)
	if err != nil {
		logutils.ZapLogger().Error("Error marshalling leaderboard page", zap.Error(err))
	}

	event := walletevent.Event{
		Type:    EvenLeaderboardPageDone,
		Message: string(payload),
	}
	s.feed.Send(event)
}

func (s *MarketDataService) FetchLeaderboardPageAsync(page, pageSize, sortOrder int, currency string) {
	s.scheduler.Enqueue(fetchLeaderboardPageTask, func(ctx context.Context) (interface{}, error) {
		result, err := s.storage.GetLeaderboardPage(page, pageSize, sortOrder, currency)
		s.cache.UpdateLastPage(result)
		return result, err
	}, func(result interface{}, taskType async.TaskType, resErr error) {
		payload, err := json.Marshal(result.(*LeaderboardPage))
		if err != nil {
			logutils.ZapLogger().Error("Error marshalling leaderboard page", zap.Error(err))
		}
		event := walletevent.Event{
			Type:    EventFetchLeaderboardPageDone,
			Message: string(payload),
		}
		s.subscribeToLeaderboard()
		s.feed.Send(event)
	})
}

func (s *MarketDataService) subscribeToLeaderboard() {
	if s.isSubscribed() {
		return
	}

	s.pageUpdateSubscription = s.subscriptionManager.Subscribe()
	go func() {
		defer common.LogOnPanic()
		for {
			select {
			case sig := <-s.pageUpdateSubscription:
				switch sig.Source() {
				case TickerFullDataUpdateSource:
					s.sendLeaderboardPageUpdate()
				case TickerPriceUpdateSource:
					s.sendLeaderboardPagePricesUpdate()
				}
			}
		}
	}()
}

func (s *MarketDataService) UnsubscribeFromLeaderboard() error {
	if !s.isSubscribed() {
		return fmt.Errorf("No subscription found")
	}
	s.subscriptionManager.Unsubscribe(s.pageUpdateSubscription)
	s.pageUpdateSubscription = nil
	s.cache.Clear()
	return nil
}
