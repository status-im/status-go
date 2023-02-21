package market

import (
	"sync"
	"time"

	"github.com/status-im/status-go/services/wallet/thirdparty"
)

type DataPoint struct {
	Price     float64
	UpdatedAt int64
}

type DataPerTokenAndCurrency = map[string]map[string]DataPoint

type Manager struct {
	provider        thirdparty.MarketDataProvider
	priceCache      DataPerTokenAndCurrency
	IsConnected     bool
	LastCheckedAt   int64
	IsConnectedLock sync.RWMutex
}

func NewManager(provider thirdparty.MarketDataProvider) *Manager {
	return &Manager{
		provider:      provider,
		priceCache:    make(DataPerTokenAndCurrency),
		IsConnected:   true,
		LastCheckedAt: time.Now().Unix(),
	}
}

func (pm *Manager) FetchHistoricalDailyPrices(symbol string, currency string, limit int, allData bool, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	return pm.provider.FetchHistoricalDailyPrices(symbol, currency, limit, allData, aggregate)
}

func (pm *Manager) FetchHistoricalHourlyPrices(symbol string, currency string, limit int, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	return pm.provider.FetchHistoricalHourlyPrices(symbol, currency, limit, aggregate)
}

func (pm *Manager) FetchTokenMarketValues(symbols []string, currency string) (map[string]thirdparty.TokenMarketValues, error) {
	return pm.provider.FetchTokenMarketValues(symbols, currency)
}

func (pm *Manager) FetchTokenDetails(symbols []string) (map[string]thirdparty.TokenDetails, error) {
	return pm.provider.FetchTokenDetails(symbols)
}

func (pm *Manager) FetchPrice(symbol string, currency string) (float64, error) {
	symbols := [1]string{symbol}
	currencies := [1]string{currency}

	prices, err := pm.FetchPrices(symbols[:], currencies[:])

	if err != nil {
		return 0, err
	}

	return prices[symbol][currency], nil
}

func (pm *Manager) FetchPrices(symbols []string, currencies []string) (map[string]map[string]float64, error) {
	result, err := pm.provider.FetchPrices(symbols, currencies)
	if err != nil {
		return nil, err
	}

	pm.updatePriceCache(result)

	return result, nil
}

func (pm *Manager) getCachedPricesFor(symbols []string, currencies []string) DataPerTokenAndCurrency {
	prices := make(DataPerTokenAndCurrency)

	for _, symbol := range symbols {
		prices[symbol] = make(map[string]DataPoint)
		for _, currency := range currencies {
			prices[symbol][currency] = pm.priceCache[symbol][currency]
		}
	}

	return prices
}

func (pm *Manager) updatePriceCache(prices map[string]map[string]float64) {
	for token, pricesPerCurrency := range prices {
		_, present := pm.priceCache[token]
		if !present {
			pm.priceCache[token] = make(map[string]DataPoint)
		}
		for currency, price := range pricesPerCurrency {
			pm.priceCache[token][currency] = DataPoint{
				Price:     price,
				UpdatedAt: time.Now().Unix(),
			}
		}
	}
}

func (pm *Manager) GetCachedPrices() DataPerTokenAndCurrency {
	return pm.priceCache
}

// Return cached price if present in cache and age is less than maxAgeInSeconds. Fetch otherwise.
func (pm *Manager) GetOrFetchPrices(symbols []string, currencies []string, maxAgeInSeconds int64) (DataPerTokenAndCurrency, error) {
	symbolsToFetchMap := make(map[string]bool)
	symbolsToFetch := make([]string, 0, len(symbols))

	now := time.Now().Unix()

	for _, symbol := range symbols {
		tokenPriceCache, ok := pm.priceCache[symbol]
		if !ok {
			if !symbolsToFetchMap[symbol] {
				symbolsToFetchMap[symbol] = true
				symbolsToFetch = append(symbolsToFetch, symbol)
			}
			continue
		}
		for _, currency := range currencies {
			if now-tokenPriceCache[currency].UpdatedAt > maxAgeInSeconds {
				if !symbolsToFetchMap[symbol] {
					symbolsToFetchMap[symbol] = true
					symbolsToFetch = append(symbolsToFetch, symbol)
				}
				break
			}
		}
	}

	if len(symbolsToFetch) > 0 {
		_, err := pm.FetchPrices(symbolsToFetch, currencies)
		if err != nil {
			return nil, err
		}
	}

	prices := pm.getCachedPricesFor(symbols, currencies)

	return prices, nil
}
