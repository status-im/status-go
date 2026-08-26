package leaderboard

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/panics"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/pkg/services/wallet/async"
	"github.com/status-im/status-go/pkg/services/wallet/walletevent"
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

// CurrencySource provides the user's selected display currency. It is
// satisfied by accounts.Database (which embeds the settings manager), the
// source of truth for the `currency` setting.
type CurrencySource interface {
	GetCurrency() (string, error)
}

// MarketDataService manages market data fetching and provides access to the latest data
type MarketDataService struct {
	config    ServiceConfig
	fetcher   DataFetcher
	scheduler *async.Scheduler
	feed      *event.Feed

	// Data storage
	storage *DataStorage
	cache   *PageCache

	// Display currency
	currencySource      CurrencySource
	accountsPublisher   *pubsub.Publisher
	subscriptionManager *SubscriptionManager

	// mu guards the two pieces of lifecycle state below. Both are reached from
	// the RPC handler, the scheduler callback and the settings watcher.
	mu sync.Mutex
	// currencyWatchStop closes to stop the settings watcher; nil while it is
	// not running.
	currencyWatchStop chan struct{}
	// pageUpdateSubscription is the channel feeding the client push events;
	// nil while no client is listening.
	pageUpdateSubscription chan Signal
}

type GetLeaderboardPageResponse struct {
	LeaderboardPage
	ErrorCode ErrorCode `json:"error_code"`
}

// NewMarketDataService creates a new market data service with the given configuration.
// currencySource and accountsPublisher may be nil, in which case the display
// currency is only taken from the client's page requests.
func NewMarketDataService(config ServiceConfig, walletDB *sql.DB, feed *event.Feed, currencySource CurrencySource, accountsPublisher *pubsub.Publisher) *MarketDataService {
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
		currencySource:      currencySource,
		accountsPublisher:   accountsPublisher,
	}
}

// Start begins the data refresh loops
func (s *MarketDataService) Start(ctx context.Context) {
	// The currency has to be known before the persisted snapshot is read, so
	// that a snapshot left over from another currency is not restored.
	s.applyStoredCurrency()
	s.storage.StartAsync()
	s.startCurrencyWatch()
	s.fetcher.Start(ctx)
}

// Stop halts all data refresh operations
func (s *MarketDataService) Stop() {
	s.releaseCurrencyWatch()
	s.fetcher.Stop()
	s.storage.WaitForStart()
	s.UnsubscribeFromLeaderboard() //nolint:errcheck
}

// applyStoredCurrency seeds the display currency from the settings DB
func (s *MarketDataService) applyStoredCurrency() {
	if s.currencySource == nil {
		return
	}
	currency, err := s.currencySource.GetCurrency()
	if err != nil {
		logutils.ZapLogger().Warn("Market - could not read the display currency setting", zap.Error(err))
		return
	}
	s.setCurrency(currency, false)
}

// startCurrencyWatch follows the display currency setting. Cached leaderboard
// values, the persisted snapshot and the ETags all belong to the currency they
// were fetched in, so a change to it invalidates every one of them and calls
// for fresh data.
func (s *MarketDataService) startCurrencyWatch() {
	if s.accountsPublisher == nil {
		return
	}

	stop, ok := s.claimCurrencyWatch()
	if !ok {
		return
	}

	events, unsub := pubsub.Subscribe[settings.EventSettingChanged](s.accountsPublisher, 1)
	go func() {
		defer panics.LogOnPanic()
		defer unsub()
		s.watchCurrency(stop, events)
	}()
}

// watchCurrency applies every currency the settings DB reports until stopped.
func (s *MarketDataService) watchCurrency(stop <-chan struct{}, events <-chan settings.EventSettingChanged) {
	for {
		select {
		case <-stop:
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			if !ev.Setting.Equals(settings.Currency) {
				continue
			}
			currency, ok := ev.Value.(string)
			if !ok {
				continue
			}
			s.setCurrency(currency, true)
		}
	}
}

// claimCurrencyWatch reserves the settings watch and returns the channel it
// should stop on. ok is false when a watch is already running.
func (s *MarketDataService) claimCurrencyWatch() (<-chan struct{}, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.currencyWatchStop != nil {
		return nil, false
	}
	s.currencyWatchStop = make(chan struct{})
	return s.currencyWatchStop, true
}

// releaseCurrencyWatch stops a running settings watch, if there is one.
func (s *MarketDataService) releaseCurrencyWatch() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.currencyWatchStop == nil {
		return
	}
	close(s.currencyWatchStop)
	s.currencyWatchStop = nil
}

// setCurrency switches the display currency, dropping everything cached for the
// previous one.
//
// When refetch is set the running refresh loops are triggered, so the
// replacement data travels the same path as any other refresh and reaches the
// client as the usual update events. While no client is listening the loops are
// stopped and the trigger is dropped; the cache stays invalidated, which leaves
// the empty data reading as stale for the next page request to fill.
func (s *MarketDataService) setCurrency(currency string, refetch bool) {
	if !s.storage.SetCurrency(currency) {
		return
	}

	s.cache.SetCurrency(currency)

	if refetch {
		s.fetcher.RefreshNow()
	}
}

// GetCombinedData returns cryptocurrency data with updated price information
func (s *MarketDataService) GetCombinedData() []Cryptocurrency {
	s.storage.WaitForStart()
	return s.storage.GetCombinedData()
}

func (s *MarketDataService) isSubscribed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
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
		s.storage.WaitForStart()
		// The requested currency is the client's display currency. Switching it
		// invalidates the cached data and ETags, so the fetch below is a full one.
		s.setCurrency(currency, false)
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
	subscription, ok := s.claimSubscription()
	if !ok {
		return
	}

	go func() {
		defer panics.LogOnPanic()
		s.pushRefreshes(subscription)
	}()
}

// pushRefreshes turns every completed refresh into the matching client event.
func (s *MarketDataService) pushRefreshes(subscription <-chan Signal) {
	for sig := range subscription {
		switch sig.Source() {
		case TickerFullDataUpdateSource:
			s.sendLeaderboardPageUpdate()
		case TickerPriceUpdateSource:
			s.sendLeaderboardPagePricesUpdate()
		}
	}
}

func (s *MarketDataService) UnsubscribeFromLeaderboard() error {
	s.fetcher.Stop()

	subscription := s.releaseSubscription()
	if subscription == nil {
		return fmt.Errorf("No subscription found")
	}
	s.subscriptionManager.Unsubscribe(subscription)
	s.cache.Clear()
	return nil
}

// claimSubscription starts the refresh loops and installs the subscription that
// carries their results to the client. ok is false when one is already
// installed.
func (s *MarketDataService) claimSubscription() (chan Signal, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.pageUpdateSubscription != nil {
		return nil, false
	}
	s.fetcher.StartRefreshLoops()
	s.pageUpdateSubscription = s.subscriptionManager.Subscribe()

	return s.pageUpdateSubscription, true
}

// releaseSubscription clears the client subscription and hands back the channel
// it was using, or nil when no client was listening.
func (s *MarketDataService) releaseSubscription() chan Signal {
	s.mu.Lock()
	defer s.mu.Unlock()

	subscription := s.pageUpdateSubscription
	s.pageUpdateSubscription = nil

	return subscription
}
