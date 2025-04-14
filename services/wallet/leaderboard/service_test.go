package leaderboard

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/services/wallet/async"
)

// MockFetcher implements DataFetcher interface for testing
type MockFetcher struct {
	storage *DataStorage
}

func NewMockFetcher(storage *DataStorage) *MockFetcher {
	f := &MockFetcher{
		storage: storage,
	}
	// Initialize data
	f.storage.UpdateCryptoDataWithEtag(mockCrypto, "test-etag")
	f.storage.UpdatePriceDataWithEtag(mockPriceData, "test-etag")
	return f
}

func (f *MockFetcher) FetchMarkets(ctx context.Context) error {
	f.storage.UpdateCryptoDataWithEtag(mockCrypto, "test-etag")
	return nil
}

func (f *MockFetcher) FetchPrices(ctx context.Context) error {
	f.storage.UpdatePriceDataWithEtag(mockPriceData, "test-etag")
	return nil
}

func (f *MockFetcher) Start(ctx context.Context) {}
func (f *MockFetcher) Stop()                     {}

func setupMarketDatadService(t *testing.T, config ServiceConfig) *MarketDataService {
	config.Validate()
	storage := NewDataStorage()
	service := &MarketDataService{
		config:              config,
		feed:                &event.Feed{},
		requestHandler:      NewRequestHandler(config, &http.Client{Timeout: 10 * time.Second}),
		storage:             storage,
		subscriptionManager: NewSubscriptionManager(),
		scheduler:           async.NewScheduler(),
		cache:               NewPageCache(),
	}
	service.fetcher = NewMockFetcher(storage)
	return service
}

func TestServiceStartStop(t *testing.T) {
	config := ServiceConfig{}

	service := setupMarketDatadService(t, config)
	require.NotNil(t, service)

	service.Start(context.Background())
	service.Stop()
}

func TestUnsubscribeWhenNotSubscribed(t *testing.T) {
	config := ServiceConfig{}
	service := setupMarketDatadService(t, config)

	// Unsubscribe should not panic or error
	_ = service.UnsubscribeFromLeaderboard()
}

func TestSubsribe(t *testing.T) {
	config := ServiceConfig{}
	service := setupMarketDatadService(t, config)

	// Subscribe should not panic or error
	service.FetchLeaderboardPageAsync(0, 0, 0, "usd")

	time.Sleep(3 * time.Second) // Wait for the async operation to complete and events to be sent

	// TODO check for sent events

	_ = service.UnsubscribeFromLeaderboard() // Unsubscribe after the test
}
