package leaderboard

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/logutils"
)

const DATA_STALE_THRESHOLD = 10 * time.Minute

// DataStorage manages the storage and retrieval of market data
type DataStorage struct {
	// Data and synchronization
	cryptoData []Cryptocurrency

	marketDataPersistence MarketDataPersistenceInterface
	priceData             PriceMap
	dataMutex             sync.RWMutex
	cryptoEtag            string
	priceEtag             string
	lastUpdateTime        time.Time
}

type FingerprintData map[string]string // map[crypto_id]fingerprint

// NewDataStorage creates a new data storage instance
func NewDataStorage(walletDB *sql.DB) *DataStorage {
	return &DataStorage{
		priceData:             make(PriceMap),
		marketDataPersistence: NewPersistance(walletDB),
	}
}

func (s *DataStorage) Start() {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()
	s.cryptoData, _ = s.marketDataPersistence.GetCryptocurrencies()
}

// UpdateCryptoDataWithEtag updates both cryptocurrency data and etag atomically
// Returns true if the data was actually updated
func (s *DataStorage) UpdateCryptoDataWithEtag(data []Cryptocurrency, etag string) bool {
	if data == nil {
		return false
	}

	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()

	currentIds := s.extractCryptocurrencyIDs(s.cryptoData)
	s.cryptoData = data
	s.cryptoEtag = etag
	s.lastUpdateTime = time.Now()
	err := s.marketDataPersistence.UpsertCryptocurrencies(s.cryptoData)
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

// UpdatePriceDataWithEtag updates both price data and etag atomically
// Returns true if the data was actually updated
func (s *DataStorage) UpdatePriceDataWithEtag(data PriceMap, etag string) bool {
	if data == nil {
		return false
	}

	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()

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
		symbol := crypto.Symbol

		// If we have updated price data for this symbol, update the cryptocurrency
		if priceUpdate, ok := s.priceData[symbol]; ok {
			// Update the price
			if crypto.CurrentPrice != priceUpdate.Price {
				crypto.CurrentPrice = priceUpdate.Price
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
		symbol := strings.ToUpper(data[i].Symbol)

		// If we have updated price data for this symbol, update the cryptocurrency
		if priceUpdate, ok := s.priceData[symbol]; ok {
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
