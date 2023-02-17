package price

import (
	"time"
)

type DataPoint struct {
	Price     float64
	UpdatedAt int64
}

type DataPerTokenAndCurrency = map[string]map[string]DataPoint

type Manager struct {
	priceProvider Provider
	priceCache    DataPerTokenAndCurrency
}

func NewManager(priceProvider Provider) *Manager {
	return &Manager{
		priceProvider: priceProvider,
		priceCache:    make(DataPerTokenAndCurrency),
	}
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
	result, err := pm.priceProvider.FetchPrices(symbols, currencies)
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
