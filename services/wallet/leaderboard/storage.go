//go:build gowaku_no_rln
// +build gowaku_no_rln

package leaderboard

import (
	"sync"
)

// DataStorage manages the storage and retrieval of market data
type DataStorage struct {
	// Data and synchronization
	cryptoData []Cryptocurrency
	priceData  PriceMap
	dataMutex  sync.RWMutex
	cryptoEtag string
	priceEtag  string

	// Statistics
	cryptoStats Stats
	priceStats  Stats
}

// NewDataStorage creates a new data storage instance
func NewDataStorage() *DataStorage {
	return &DataStorage{
		priceData: make(PriceMap),
	}
}

// UpdateCryptoData updates the cryptocurrency data
// Returns true if the data was actually updated
func (s *DataStorage) UpdateCryptoData(data []Cryptocurrency) bool {
	if data == nil {
		return false
	}

	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()

	s.cryptoData = data
	return true
}

// UpdatePriceData updates the price data
// Returns true if the data was actually updated
func (s *DataStorage) UpdatePriceData(data PriceMap) bool {
	if data == nil {
		return false
	}

	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()

	s.priceData = data
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
			// Update the price in the USD quote
			if crypto.Quote.USD.Price != priceUpdate.Price {
				crypto.Quote.USD.Price = priceUpdate.Price
			}

			// Update volume data if available
			if priceUpdate.Volume24h > 0 {
				crypto.Quote.USD.Volume24h = priceUpdate.Volume24h
			}

			// Update percentage change if available
			if priceUpdate.PercentChange24h != 0 {
				crypto.Quote.USD.PercentChange24h = priceUpdate.PercentChange24h
			}

			// Optionally add a field to indicate this has updated pricing
			// This could be done if the Cryptocurrency struct has such a field
		}
	}

	return result
}

// GetCryptoEtag returns the current crypto data etag
func (s *DataStorage) GetCryptoEtag() string {
	s.dataMutex.RLock()
	defer s.dataMutex.RUnlock()
	return s.cryptoEtag
}

// SetCryptoEtag sets the crypto data etag
func (s *DataStorage) SetCryptoEtag(etag string) {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()
	s.cryptoEtag = etag
}

// GetPriceEtag returns the current price data etag
func (s *DataStorage) GetPriceEtag() string {
	s.dataMutex.RLock()
	defer s.dataMutex.RUnlock()
	return s.priceEtag
}

// SetPriceEtag sets the price data etag
func (s *DataStorage) SetPriceEtag(etag string) {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()
	s.priceEtag = etag
}

// GetCryptoStats returns statistics for crypto data requests
func (s *DataStorage) GetCryptoStats() Stats {
	return s.cryptoStats
}

// GetPriceStats returns statistics for price data requests
func (s *DataStorage) GetPriceStats() Stats {
	return s.priceStats
}

// UpdateCryptoStats updates the crypto stats reference for request handling
func (s *DataStorage) UpdateCryptoStats(stats Stats) {
	s.cryptoStats = stats
}

// UpdatePriceStats updates the price stats reference for request handling
func (s *DataStorage) UpdatePriceStats(stats Stats) {
	s.priceStats = stats
}

// GetCryptoStatsRef returns a reference to crypto stats for updating
func (s *DataStorage) GetCryptoStatsRef() *Stats {
	return &s.cryptoStats
}

// GetPriceStatsRef returns a reference to price stats for updating
func (s *DataStorage) GetPriceStatsRef() *Stats {
	return &s.priceStats
}
