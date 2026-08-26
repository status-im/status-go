package leaderboard

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/pkg/security"
	"github.com/status-im/status-go/pkg/services/wallet/async"
	"github.com/status-im/status-go/pkg/services/wallet/walletevent"
)

const (
	priceInUSD = 78281.0
	priceInEUR = 67000.0
)

// stubCurrencySource stands in for the settings DB.
type stubCurrencySource struct{ currency string }

func (s *stubCurrencySource) GetCurrency() (string, error) { return s.currency, nil }

// currencyAwareFetcher produces a price that depends on the currency the
// storage is set to, so USD data can be told apart from EUR data. It stands in
// for the refresh loops: a trigger fetches and reports, but only while the
// loops are meant to be running.
type currencyAwareFetcher struct {
	storage *DataStorage
	subs    *SubscriptionManager

	mu      sync.Mutex
	running bool
}

func priceFor(currency string) float64 {
	if currency == "eur" {
		return priceInEUR
	}
	return priceInUSD
}

func (f *currencyAwareFetcher) FetchMarkets(ctx context.Context) error {
	currency := f.storage.GetCurrency()
	f.storage.UpdateCryptoDataWithEtag([]Cryptocurrency{
		{ID: "bitcoin", Symbol: "btc", CurrentPrice: priceFor(currency)},
	}, "etag-"+currency, currency)
	return nil
}

func (f *currencyAwareFetcher) FetchPrices(ctx context.Context) error {
	currency := f.storage.GetCurrency()
	f.storage.UpdatePriceDataWithEtag(PriceMap{
		"bitcoin": {ID: "bitcoin", Price: priceFor(currency)},
	}, "price-etag-"+currency, currency)
	return nil
}

func (f *currencyAwareFetcher) Start(ctx context.Context) {}

func (f *currencyAwareFetcher) StartRefreshLoops() { f.setRunning(true) }

func (f *currencyAwareFetcher) Stop() { f.setRunning(false) }

func (f *currencyAwareFetcher) RefreshNow() {
	if !f.isRunning() {
		return
	}
	ctx := context.Background()
	if err := f.FetchMarkets(ctx); err == nil {
		f.subs.Emit(ctx, TickerFullDataUpdateSource)
	}
	if err := f.FetchPrices(ctx); err == nil {
		f.subs.Emit(ctx, TickerPriceUpdateSource)
	}
}

func (f *currencyAwareFetcher) setRunning(running bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.running = running
}

func (f *currencyAwareFetcher) isRunning() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.running
}

type pushTestService struct {
	service   *MarketDataService
	events    chan walletevent.Event
	publisher *pubsub.Publisher
}

func newPushTestService(t *testing.T, db *sql.DB, currency string) *pushTestService {
	t.Helper()

	feed := &event.Feed{}
	events := make(chan walletevent.Event, 64)
	sub := feed.Subscribe(events)
	t.Cleanup(sub.Unsubscribe)

	publisher := pubsub.NewPublisher()
	t.Cleanup(publisher.Close)

	storage := NewDataStorage(db)
	subs := NewSubscriptionManager()
	service := &MarketDataService{
		feed:                feed,
		storage:             storage,
		subscriptionManager: subs,
		scheduler:           async.NewScheduler(),
		cache:               NewPageCache(),
		currencySource:      &stubCurrencySource{currency: currency},
		accountsPublisher:   publisher,
	}
	service.fetcher = &currencyAwareFetcher{storage: storage, subs: subs}
	service.Start(context.Background())

	return &pushTestService{service: service, events: events, publisher: publisher}
}

func (s *pushTestService) publishCurrency(currency string) {
	pubsub.Publish(s.publisher, settings.EventSettingChanged{
		Setting: settings.Currency,
		Value:   currency,
	})
}

func waitForEventType(t *testing.T, events chan walletevent.Event, want walletevent.EventType) string {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case ev := <-events:
			if ev.Type == want {
				return ev.Message
			}
		case <-deadline:
			t.Fatalf("timed out waiting for %s", want)
		}
	}
}

// TestCurrencyChangePushesConvertedPage covers a client that is listening when
// the setting changes: the new page has to reach it, or the tab keeps showing
// the old currency's figures.
func TestCurrencyChangePushesConvertedPage(t *testing.T) {
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()

	svc := newPushTestService(t, db, "usd")
	defer svc.service.Stop()

	svc.service.FetchLeaderboardPageAsync(1, 10, 0, "USD")
	waitForEventType(t, svc.events, EventFetchLeaderboardPageDone)
	require.Equal(t, priceInUSD, svc.service.storage.GetCryptoData()[0].CurrentPrice)

	svc.publishCurrency("EUR")

	msg := waitForEventType(t, svc.events, EventLeaderboardPageDataUpdated)
	var page LeaderboardPage
	require.NoError(t, json.Unmarshal([]byte(msg), &page))
	require.Equal(t, "EUR", page.Currency)
	require.Len(t, page.Data, 1)
	require.Equal(t, priceInEUR, page.Data[0].CurrentPrice)
}

// TestCurrencyChangeWhileTabClosed covers the path the desktop takes: the
// Market tab is torn down when the user opens Settings, so the change lands
// with nothing subscribed and the tab re-requests on its way back in.
func TestCurrencyChangeWhileTabClosed(t *testing.T) {
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()

	svc := newPushTestService(t, db, "usd")
	defer svc.service.Stop()

	svc.service.FetchLeaderboardPageAsync(1, 10, 0, "USD")
	waitForEventType(t, svc.events, EventFetchLeaderboardPageDone)

	// User leaves the Market tab for Settings.
	require.NoError(t, svc.service.UnsubscribeFromLeaderboard())

	svc.publishCurrency("EUR")
	require.Eventually(t, func() bool { return svc.service.storage.GetCurrency() == "eur" },
		2*time.Second, 10*time.Millisecond)

	// User returns to the Market tab.
	svc.service.FetchLeaderboardPageAsync(1, 10, 0, "EUR")
	msg := waitForEventType(t, svc.events, EventFetchLeaderboardPageDone)

	var res GetLeaderboardPageResponse
	require.NoError(t, json.Unmarshal([]byte(msg), &res))
	require.Len(t, res.Data, 1)
	require.Equal(t, priceInEUR, res.Data[0].CurrentPrice)
}

// TestFetchInFlightDuringCurrencyChangeIsDropped guards the case where a
// refresh issued in USD lands after the user has switched to EUR. Storing those
// dollar figures as EUR data would also refresh the timestamp and keep the USD
// ETag, leaving nothing to refetch and the tab showing dollar amounts behind a
// euro sign until the data went stale.
func TestFetchInFlightDuringCurrencyChangeIsDropped(t *testing.T) {
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()

	storage := NewDataStorage(db)

	released := make(chan struct{})
	inFlight := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No convert_currency: the request goes out while USD is selected.
		require.Empty(t, r.URL.Query().Get(convertCurrencyParam))
		close(inFlight)
		<-released
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Etag", "usd-etag")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "bitcoin", "symbol": "btc", "current_price": priceInUSD},
			},
		})
	}))
	defer server.Close()

	fetcher := NewProxyFetcher(ServiceConfig{
		User:        security.NewSensitiveString("test"),
		Password:    security.NewSensitiveString("password"),
		UrlOverride: security.NewSensitiveString(server.URL),
		AllowETag:   true,
	}, storage, NewSubscriptionManager()).(*ProxyFetcher)

	done := make(chan error, 1)
	go func() { done <- fetcher.FetchMarkets(context.Background()) }()

	<-inFlight
	require.True(t, storage.SetCurrency("EUR"))
	close(released)
	require.NoError(t, <-done)

	require.Equal(t, "eur", storage.GetCurrency())
	require.Empty(t, storage.GetCryptoData(), "USD payload was stored as EUR data")
	require.Empty(t, storage.GetCryptoEtag(), "USD ETag was kept for the EUR currency")
	require.True(t, storage.IsDataStale(), "stale flag cleared, so nothing would refetch in EUR")
}

// TestPricesInFlightDuringCurrencyChangeIsDropped covers the same race on the
// price refresh loop, which pushes straight to the client with no staleness
// check to catch a stale value later.
func TestPricesInFlightDuringCurrencyChangeIsDropped(t *testing.T) {
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()

	storage := NewDataStorage(db)

	released := make(chan struct{})
	inFlight := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(inFlight)
		<-released
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"bitcoin": map[string]interface{}{"price": priceInUSD},
		})
	}))
	defer server.Close()

	fetcher := NewProxyFetcher(ServiceConfig{
		User:        security.NewSensitiveString("test"),
		Password:    security.NewSensitiveString("password"),
		UrlOverride: security.NewSensitiveString(server.URL),
	}, storage, NewSubscriptionManager()).(*ProxyFetcher)

	done := make(chan error, 1)
	go func() { done <- fetcher.FetchPrices(context.Background()) }()

	<-inFlight
	require.True(t, storage.SetCurrency("EUR"))
	close(released)
	require.NoError(t, <-done)

	require.Empty(t, storage.GetPriceData(), "USD prices were stored as EUR prices")
}

// TestRefreshNowDrivesTheRunningLoops pins the mechanism a currency change
// relies on: the refetch travels through the same loops as every periodic
// refresh, and does nothing while they are stopped.
func TestRefreshNowDrivesTheRunningLoops(t *testing.T) {
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()

	requests := make(chan string, 8)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == PRICES_ENDPOINT {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
	}))
	defer server.Close()

	storage := NewDataStorage(db)
	subs := NewSubscriptionManager()
	signals := subs.Subscribe()

	// Intervals long enough that nothing here can be a tick.
	fetcher := NewProxyFetcher(ServiceConfig{
		UrlOverride:         security.NewSensitiveString(server.URL),
		FullDataInterval:    time.Hour,
		PriceUpdateInterval: time.Hour,
	}, storage, subs)

	// While the loops are stopped a trigger has to do nothing, which is what
	// leaves the tab-closed case to the next page request.
	fetcher.RefreshNow()
	select {
	case path := <-requests:
		t.Fatalf("stopped loops fetched %s", path)
	case <-time.After(200 * time.Millisecond):
	}

	fetcher.StartRefreshLoops()
	defer fetcher.Stop()

	fetcher.RefreshNow()

	seen := map[string]bool{}
	for len(seen) < 2 {
		select {
		case path := <-requests:
			seen[path] = true
		case <-time.After(5 * time.Second):
			t.Fatalf("triggered loops did not fetch, saw %v", seen)
		}
	}
	require.True(t, seen[MARKETS_ENDPOINT])
	require.True(t, seen[PRICES_ENDPOINT])

	// Each completed refresh reports itself, which is what carries the new
	// values out to the client.
	select {
	case <-signals:
	case <-time.After(5 * time.Second):
		t.Fatal("a triggered refresh reported nothing to the subscribers")
	}
}
