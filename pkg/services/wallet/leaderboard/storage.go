package leaderboard

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/panics"
)

const DATA_STALE_THRESHOLD = 10 * time.Minute

// DataStorage manages the storage and retrieval of market data.
//
// It is the one owner of the invariant that the cached values, their ETags and
// the persisted snapshot are all expressed in `currency`: only SetCurrency
// changes it, and every write is checked against it first.
type DataStorage struct {
	marketDataPersistence MarketDataPersistenceInterface

	// dataMutex guards everything below it.
	dataMutex             sync.RWMutex
	cryptoData            []Cryptocurrency
	cryptoDataInitialized bool
	priceData             PriceMap
	cryptoEtag            string
	priceEtag             string
	lastUpdateTime        time.Time
	// currency is the display currency the cached values are expressed in.
	// Everything held here - data, etags and the persisted snapshot - is only
	// valid for this currency.
	currency string

	// startMu guards startDone. It stays separate from dataMutex because
	// WaitForStart blocks on startDone until Start finishes, and Start needs
	// dataMutex to do so.
	startMu   sync.Mutex
	startDone chan struct{}
}

type FingerprintData map[string]string // map[crypto_id]fingerprint

// NewDataStorage creates a new data storage instance
func NewDataStorage(walletDB *sql.DB) *DataStorage {
	return &DataStorage{
		priceData:             make(PriceMap),
		marketDataPersistence: NewPersistance(walletDB),
		currency:              DefaultCurrency,
	}
}

func (s *DataStorage) Start() {
	s.dataMutex.RLock()
	if s.cryptoDataInitialized {
		s.dataMutex.RUnlock()
		return
	}
	currency := normalizeCurrency(s.currency)
	s.dataMutex.RUnlock()

	// Only the snapshot persisted for the currently selected currency may be
	// restored; a snapshot left over from another currency is a cache miss.
	cryptoData, _ := s.marketDataPersistence.GetCryptocurrencies(currency)

	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()
	if !s.cryptoDataInitialized && normalizeCurrency(s.currency) == currency {
		s.cryptoData = cryptoData
		s.cryptoDataInitialized = true
	}
}

// GetCurrency returns the display currency the cached data is expressed in.
func (s *DataStorage) GetCurrency() string {
	s.dataMutex.RLock()
	defer s.dataMutex.RUnlock()
	return normalizeCurrency(s.currency)
}

// SetCurrency switches the display currency. Because the cached values, the
// persisted snapshot and the ETags are all currency-specific, a switch drops
// every one of them so the next fetch is a full fetch in the new currency.
// It reports whether the currency actually changed.
func (s *DataStorage) SetCurrency(currency string) bool {
	currency = normalizeCurrency(currency)

	s.dataMutex.Lock()
	if normalizeCurrency(s.currency) == currency {
		s.dataMutex.Unlock()
		return false
	}

	s.currency = currency
	s.cryptoData = nil
	s.cryptoDataInitialized = false
	s.priceData = make(PriceMap)
	s.cryptoEtag = ""
	s.priceEtag = ""
	s.lastUpdateTime = time.Time{}
	persistence := s.marketDataPersistence
	s.dataMutex.Unlock()

	if err := persistence.DeleteCryptocurrenciesNotIn(currency); err != nil {
		logutils.ZapLogger().Error("Market - error dropping data of the previous currency", zap.Error(err))
	}

	return true
}

func (s *DataStorage) StartAsync() {
	s.startMu.Lock()
	startDone := make(chan struct{})
	s.startDone = startDone
	s.startMu.Unlock()

	go func() {
		defer panics.LogOnPanic()
		defer close(startDone)
		s.Start()
	}()
}

func (s *DataStorage) WaitForStart() {
	s.startMu.Lock()
	startDone := s.startDone
	s.startMu.Unlock()
	if startDone != nil {
		<-startDone
	}
}

// matchesCurrency reports whether a response fetched in fetchedIn still belongs
// here. Fetching and storing are not one atomic step, so the display currency
// can change while a request is in flight; storing such a response would label
// one currency's values as another's, and would refresh the timestamp that
// decides whether anything needs fetching at all.
// Callers must hold dataMutex.
func (s *DataStorage) matchesCurrency(fetchedIn string) bool {
	return normalizeCurrency(fetchedIn) == normalizeCurrency(s.currency)
}

// UpdateCryptoDataWithEtag updates both cryptocurrency data and etag atomically.
// fetchedIn is the display currency the response was requested in; an update
// that no longer matches the selected currency is dropped. It reports whether
// the data was stored.
func (s *DataStorage) UpdateCryptoDataWithEtag(data []Cryptocurrency, etag string, fetchedIn string) bool {
	if data == nil {
		return false
	}

	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()

	if !s.matchesCurrency(fetchedIn) {
		return false
	}

	s.cryptoDataInitialized = true
	currentIds := s.extractCryptocurrencyIDs(s.cryptoData)
	s.cryptoData = data
	s.cryptoEtag = etag
	s.lastUpdateTime = time.Now()
	err := s.marketDataPersistence.UpsertCryptocurrencies(s.cryptoData, s.currency)
	if err != nil {
		logutils.ZapLogger().Error("Market - error creating database snapshot", zap.Error(err))
	}
	// Remove old data
	updatedIDs := make(map[string]bool)
	for _, crypto := range s.cryptoData {
		updatedIDs[crypto.ID] = true
	}
	var idsToDelete []string
	for id := range currentIds {
		if !updatedIDs[id] {
			idsToDelete = append(idsToDelete, id)
		}
	}
	if len(idsToDelete) > 0 {
		err = s.marketDataPersistence.DeleteCryptocurrencies(idsToDelete)
		if err != nil {
			logutils.ZapLogger().Error("Market - error deleting old data", zap.Error(err))
		}
	}
	return true
}

func (s *DataStorage) extractCryptocurrencyIDs(cryptos []Cryptocurrency) map[string]bool {
	ids := make(map[string]bool, len(cryptos))
	for _, crypto := range cryptos {
		ids[crypto.ID] = true
	}
	return ids
}

func (s *DataStorage) IsDataStale() bool {
	if s.lastUpdateTime.IsZero() {
		return true
	}
	// Check if the data is older than 5 minutes
	return time.Since(s.lastUpdateTime) > DATA_STALE_THRESHOLD
}

// UpdatePriceDataWithEtag updates both price data and etag atomically.
// fetchedIn is the display currency the response was requested in; an update
// that no longer matches the selected currency is dropped. It reports whether
// the data was stored.
func (s *DataStorage) UpdatePriceDataWithEtag(data PriceMap, etag string, fetchedIn string) bool {
	if data == nil {
		return false
	}

	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()

	if !s.matchesCurrency(fetchedIn) {
		return false
	}

	s.priceData = data
	s.priceEtag = etag
	return true
}

// GetCryptoData returns the latest cryptocurrency data
func (s *DataStorage) GetCryptoData() []Cryptocurrency {
	s.dataMutex.RLock()
	defer s.dataMutex.RUnlock()

	// Create a copy to avoid data races
	result := make([]Cryptocurrency, len(s.cryptoData))
	copy(result, s.cryptoData)

	return result
}

func (s *DataStorage) GetCryptoDataSize() int {
	s.dataMutex.RLock()
	defer s.dataMutex.RUnlock()

	return len(s.cryptoData)
}

// GetPriceData returns the latest price data
func (s *DataStorage) GetPriceData() PriceMap {
	s.dataMutex.RLock()
	defer s.dataMutex.RUnlock()

	// Create a copy to avoid data races
	result := make(PriceMap, len(s.priceData))
	for k, v := range s.priceData {
		result[k] = v
	}

	return result
}

// GetCombinedData returns cryptocurrency data with updated price information
func (s *DataStorage) GetCombinedData() []Cryptocurrency {
	s.dataMutex.RLock()
	defer s.dataMutex.RUnlock()

	// Create a copy of the crypto data
	result := make([]Cryptocurrency, len(s.cryptoData))
	copy(result, s.cryptoData)

	// Update with the latest price data where available
	for i := range result {
		crypto := &result[i]
		// Use the cryptocurrency ID (in lowercase) to lookup price data
		cryptoID := strings.ToLower(crypto.ID)

		// If we have updated price data for this crypto ID, update the cryptocurrency
		if priceUpdate, ok := s.priceData[cryptoID]; ok {
			// Update the price
			if crypto.CurrentPrice != priceUpdate.Price {
				crypto.CurrentPrice = priceUpdate.Price
			}

			// Update market cap if available
			if priceUpdate.MarketCap != 0 {
				crypto.MarketCap = priceUpdate.MarketCap
			}

			// Update volume if available
			if priceUpdate.Volume24h != 0 {
				crypto.TotalVolume = priceUpdate.Volume24h
			}

			// Update percentage change if available
			if priceUpdate.PercentChange24h != 0 {
				crypto.PriceChangePercentage24h = priceUpdate.PercentChange24h
			}
		}
	}

	return result
}

func (s *DataStorage) GetCryptoDataForPage(page, pageSize int) []Cryptocurrency {
	if pageSize <= 0 || page <= 0 {
		return nil
	}
	s.dataMutex.RLock()
	defer s.dataMutex.RUnlock()

	start := (page - 1) * pageSize
	totalCount := len(s.cryptoData)

	if start >= totalCount {
		return []Cryptocurrency{}
	}
	end := start + pageSize
	if end > totalCount {
		end = totalCount
	}
	return append([]Cryptocurrency{}, s.cryptoData[start:end]...)
}

// GetCryptoEtag returns the current crypto data etag
func (s *DataStorage) GetCryptoEtag() string {
	s.dataMutex.RLock()
	defer s.dataMutex.RUnlock()
	return s.cryptoEtag
}

// GetPriceEtag returns the current price data etag
func (s *DataStorage) GetPriceEtag() string {
	s.dataMutex.RLock()
	defer s.dataMutex.RUnlock()
	return s.priceEtag
}

func (s *DataStorage) GetLeaderboardPagePrices(page LeaderboardPage) *LeaderboardPagePrices {
	if page.PageSize <= 0 || page.Page <= 0 {
		return nil
	}
	data := s.GetCryptoDataForPage(page.Page, page.PageSize)

	s.dataMutex.RLock()
	defer s.dataMutex.RUnlock()

	result := &LeaderboardPagePrices{
		Page:      page.Page,
		PageSize:  page.PageSize,
		SortOrder: page.SortOrder,
		Currency:  page.Currency,
	}

	for i := range data {
		// coingecko id lowercase (e.g., "bitcoin", "ethereum")
		cryptoID := strings.ToLower(data[i].ID)

		// If we have updated price data for this crypto ID, add it to results
		if priceUpdate, ok := s.priceData[cryptoID]; ok {
			priceUpdate.ID = data[i].ID
			result.Data = append(result.Data, priceUpdate)
		}
	}
	return result
}

func (s *DataStorage) GetLeaderboardPage(page, pageSize, sortOrder int, currency string) (*LeaderboardPage, error) {
	if pageSize <= 0 {
		return nil, fmt.Errorf("Invalid page size")
	}

	if page <= 0 {
		return nil, fmt.Errorf("Invalid page")
	}

	totalCount := s.GetCryptoDataSize()

	totalPages := (totalCount + pageSize - 1) / pageSize
	if page <= 0 || (page > totalPages && totalCount > 0) {
		return nil, fmt.Errorf("Invalid page")
	}

	result := &LeaderboardPage{
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
		SortOrder:  sortOrder,
		Currency:   currency,
		Data:       s.GetCryptoDataForPage(page, pageSize),
	}
	return result, nil
}
