package leaderboard

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

type ErrorCode = int

const (
	// Contains a LeaderboardPage payload
	EventFetchLeaderboardPageDone walletevent.EventType = "wallet-fetch-leaderboard-page-done"
	// Contains a LeaderboardPage payload
	EventLeaderboardPageDataUpdated walletevent.EventType = "wallet-leaderboard-page-data-updated"
	// Contains a EventLeaderboardPagePricesUpdate payload
	EventLeaderboardPagePricesUpdated walletevent.EventType = "wallet-leaderboard-page-prices-updated"

	// Signal source
	TickerFullDataUpdateSource int = 0
	TickerPriceUpdateSource    int = 1

	// Error codes
	ErrorCodeSuccess      ErrorCode = 1
	ErrorCodeTaskCanceled ErrorCode = 2
	ErrorCodeFailed       ErrorCode = 3
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
	fetcher   DataFetcher
	scheduler *async.Scheduler
	feed      *event.Feed

	// Data storage
	storage *DataStorage
	cache   *PageCache

	// Subscription management
	subscriptionManager    *SubscriptionManager
	pageUpdateSubscription chan Signal
}

type GetLeaderboardPageResponse struct {
	LeaderboardPage
	ErrorCode ErrorCode `json:"error_code"`
}

// NewMarketDataService creates a new market data service with the given configuration
func NewMarketDataService(config ServiceConfig, walletDB *sql.DB, feed *event.Feed) *MarketDataService {
	storage := NewDataStorage(walletDB)
	subscriptionManager := NewSubscriptionManager()
	return &MarketDataService{
		config:              config,
		fetcher:             NewProxyFetcher(config, storage, subscriptionManager),
		feed:                feed,
		storage:             storage,
		subscriptionManager: subscriptionManager,
		scheduler:           async.NewScheduler(),
		cache:               NewPageCache(),
	}
}

// Start begins the data refresh loops
func (s *MarketDataService) Start(ctx context.Context) {
	s.storage.Start()
	s.fetcher.Start(ctx)
}

// Stop halts all data refresh operations
func (s *MarketDataService) Stop() {
	s.fetcher.Stop()
	s.UnsubscribeFromLeaderboard() //nolint:errcheck
}

// GetCombinedData returns cryptocurrency data with updated price information
func (s *MarketDataService) GetCombinedData() []Cryptocurrency {
	return s.storage.GetCombinedData()
}

func (s *MarketDataService) isSubscribed() bool {
	return s.pageUpdateSubscription != nil
}

func (s *MarketDataService) sendLeaderboardPagePricesUpdate() {
	if !s.isSubscribed() {
		return
	}

	lastPage := s.cache.GetLastPage()
	if !lastPage.Valid() {
		return
	}

	result := s.storage.GetLeaderboardPagePrices(lastPage)
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
	if !lastPage.Valid() {
		return
	}
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
		Type:    EventLeaderboardPageDataUpdated,
		Message: string(payload),
	}
	s.feed.Send(event)
}

func (s *MarketDataService) FetchLeaderboardPageAsync(page, pageSize, sortOrder int, currency string) {
	s.scheduler.Enqueue(fetchLeaderboardPageTask, func(ctx context.Context) (interface{}, error) {
		if s.storage.IsDataStale() {
			s.fetcher.FetchMarkets(ctx) //nolint:errcheck
		}
		result, err := s.storage.GetLeaderboardPage(page, pageSize, sortOrder, currency)
		if err != nil {
			logutils.ZapLogger().Error("Error fetching leaderboard page", zap.Error(err))
			return nil, err
		}
		s.cache.UpdateLastPage(result)
		return result, err
	}, func(result interface{}, taskType async.TaskType, resErr error) {
		res := GetLeaderboardPageResponse{
			ErrorCode: ErrorCodeFailed,
		}
		if errors.Is(resErr, context.Canceled) || errors.Is(resErr, async.ErrTaskOverwritten) {
			res.ErrorCode = ErrorCodeTaskCanceled
		} else if resErr == nil {
			res.ErrorCode = ErrorCodeSuccess
			res.LeaderboardPage = *(result.(*LeaderboardPage))
			s.subscribeToLeaderboard()
		}

		payload, err := json.Marshal(res)
		if err != nil {
			logutils.ZapLogger().Error("Error marshalling leaderboard page response", zap.Error(err))
		}
		event := walletevent.Event{
			Type:    EventFetchLeaderboardPageDone,
			Message: string(payload),
		}
		s.feed.Send(event)
	})
}

func (s *MarketDataService) subscribeToLeaderboard() {
	if s.isSubscribed() {
		return
	}

	s.fetcher.StartRefreshLoops()

	s.pageUpdateSubscription = s.subscriptionManager.Subscribe()
	go func() {
		defer common.LogOnPanic()
		for sig := range s.pageUpdateSubscription {
			switch sig.Source() {
			case TickerFullDataUpdateSource:
				s.sendLeaderboardPageUpdate()
			case TickerPriceUpdateSource:
				s.sendLeaderboardPagePricesUpdate()
			}
		}
	}()
}

func (s *MarketDataService) UnsubscribeFromLeaderboard() error {
	s.fetcher.Stop()
	if !s.isSubscribed() {
		return fmt.Errorf("No subscription found")
	}
	s.subscriptionManager.Unsubscribe(s.pageUpdateSubscription)
	s.pageUpdateSubscription = nil
	s.cache.Clear()
	return nil
}
