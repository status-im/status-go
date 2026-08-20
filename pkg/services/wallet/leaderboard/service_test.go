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
	"github.com/status-im/status-go/pkg/services/wallet/async"
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

type startAttempt struct {
	started chan struct{}
	release chan struct{}
}

type restartableMarketDataPersistence struct {
	attempts chan startAttempt
	data     []Cryptocurrency
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

func (p *restartableMarketDataPersistence) UpsertCryptocurrencies([]Cryptocurrency) error {
	return nil
}

func (p *restartableMarketDataPersistence) GetCryptocurrencies() ([]Cryptocurrency, error) {
	attempt := <-p.attempts
	close(attempt.started)
	<-attempt.release
	return p.data, nil
}

func (p *restartableMarketDataPersistence) DeleteCryptocurrencies([]string) error {
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

func TestDataStorageStartDoesNotOverwriteFetchedData(t *testing.T) {
	persistence := &blockingMarketDataPersistence{
		started: make(chan struct{}),
		release: make(chan struct{}),
		data:    mockCrypto,
	}
	storage := &DataStorage{
		priceData:             make(PriceMap),
		marketDataPersistence: persistence,
	}
	fetchedData := []Cryptocurrency{{ID: "fresh-market-data"}}

	storage.StartAsync()
	<-persistence.started
	require.True(t, storage.UpdateCryptoDataWithEtag(fetchedData, "fresh-etag"))

	close(persistence.release)
	storage.WaitForStart()

	require.Equal(t, fetchedData, storage.GetCryptoData())
}

func TestDataStorageCanRestartAfterStart(t *testing.T) {
	firstAttempt := startAttempt{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	persistence := &restartableMarketDataPersistence{
		attempts: make(chan startAttempt, 1),
		data:     mockCrypto,
	}
	storage := &DataStorage{
		priceData:             make(PriceMap),
		marketDataPersistence: persistence,
	}

	persistence.attempts <- firstAttempt
	storage.StartAsync()
	<-firstAttempt.started
	close(firstAttempt.release)
	storage.WaitForStart()

	storage.StartAsync()
	storage.WaitForStart()
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
