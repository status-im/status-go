package leaderboard

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/status-go/internal/db/walletdatabase"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/services/wallet/async"
)

// MockFetcher implements DataFetcher interface for testing
type MockFetcher struct {
	storage *DataStorage
}

type blockingMarketDataPersistence struct {
	started chan struct{}
	release chan struct{}
	data    []Cryptocurrency
}

func (p *blockingMarketDataPersistence) UpsertCryptocurrencies([]Cryptocurrency) error {
	return nil
}

func (p *blockingMarketDataPersistence) GetCryptocurrencies() ([]Cryptocurrency, error) {
	close(p.started)
	<-p.release
	return p.data, nil
}

func (p *blockingMarketDataPersistence) DeleteCryptocurrencies([]string) error {
	return nil
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
func (f *MockFetcher) StartRefreshLoops()        {}

func setupTestWalletDB(t *testing.T) (*sql.DB, func()) {
	db, cleanup, err := testutils.SetupTestSQLDB(walletdatabase.DbInitializer{}, "wallet-tests")
	require.NoError(t, err)
	return db, func() { require.NoError(t, cleanup()) }
}

func setupMarketDatadService(t *testing.T, config ServiceConfig, db *sql.DB) *MarketDataService {
	storage := NewDataStorage(db)
	service := &MarketDataService{
		config:              config,
		feed:                &event.Feed{},
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

	db, cleanup := setupTestWalletDB(t)
	defer cleanup()
	service := setupMarketDatadService(t, config, db)
	require.NotNil(t, service)

	service.Start(context.Background())
	service.Stop()
}

func TestServiceWaitsForAsyncStorageStart(t *testing.T) {
	persistence := &blockingMarketDataPersistence{
		started: make(chan struct{}),
		release: make(chan struct{}),
		data:    mockCrypto,
	}
	storage := &DataStorage{
		priceData:             make(PriceMap),
		marketDataPersistence: persistence,
	}
	service := &MarketDataService{
		storage:             storage,
		fetcher:             &MockFetcher{storage: storage},
		subscriptionManager: NewSubscriptionManager(),
	}

	service.Start(context.Background())
	<-persistence.started

	combinedData := make(chan []Cryptocurrency, 1)
	go func() {
		combinedData <- service.GetCombinedData()
	}()
	stopDone := make(chan struct{})
	go func() {
		service.Stop()
		close(stopDone)
	}()

	select {
	case <-combinedData:
		t.Fatal("GetCombinedData returned before storage initialization completed")
	default:
	}
	select {
	case <-stopDone:
		t.Fatal("Stop returned before storage initialization completed")
	default:
	}

	close(persistence.release)
	require.Equal(t, mockCrypto, <-combinedData)
	<-stopDone
}

func TestUnsubscribeWhenNotSubscribed(t *testing.T) {
	config := ServiceConfig{}
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()
	service := setupMarketDatadService(t, config, db)

	// Unsubscribe should not panic or error
	_ = service.UnsubscribeFromLeaderboard()
}

func TestSubsribe(t *testing.T) {
	config := ServiceConfig{}
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()
	service := setupMarketDatadService(t, config, db)

	// Subscribe should not panic or error
	service.FetchLeaderboardPageAsync(0, 0, 0, "usd")

	time.Sleep(3 * time.Second) // Wait for the async operation to complete and events to be sent

	// TODO check for sent events

	_ = service.UnsubscribeFromLeaderboard() // Unsubscribe after the test
}
