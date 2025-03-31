package market

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/status-go/circuitbreaker"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

const (
	EventMarketStatusChanged walletevent.EventType = "wallet-market-status-changed"
)

const (
	MaxAgeInSecondsForFresh    int64 = -1
	MaxAgeInSecondsForBalances int64 = 60
)

type DataPoint struct {
	Price     float64
	UpdatedAt int64
}

type MarketValuesSnapshot struct {
	MarketValues thirdparty.TokenMarketValues
	UpdatedAt    int64
}

type DataPerTokenAndCurrency = map[string]map[string]DataPoint
type MarketValuesPerCurrencyAndToken = map[string]map[string]MarketValuesSnapshot
type TokenMarketCache MarketValuesPerCurrencyAndToken
type TokenPriceCache DataPerTokenAndCurrency

type Manager struct {
	feed            *event.Feed
	priceCache      MarketCache[TokenPriceCache]
	marketCache     MarketCache[TokenMarketCache]
	IsConnected     bool
	LastCheckedAt   int64
	IsConnectedLock sync.RWMutex
	circuitbreaker  *circuitbreaker.CircuitBreaker
	providers       []thirdparty.MarketDataProvider
}

func NewManager(providers []thirdparty.MarketDataProvider, feed *event.Feed) *Manager {
	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		Timeout:               60000,
		MaxConcurrentRequests: 100,
		SleepWindow:           300000,
		ErrorPercentThreshold: 25,
	})

	return &Manager{
		feed:           feed,
		priceCache:     *NewCache(make(TokenPriceCache)),
		marketCache:    *NewCache(make(TokenMarketCache)),
		IsConnected:    true,
		LastCheckedAt:  time.Now().Unix(),
		circuitbreaker: cb,
		providers:      providers,
	}
}

func (pm *Manager) setIsConnected(value bool) {
	pm.IsConnectedLock.Lock()
	defer pm.IsConnectedLock.Unlock()
	pm.LastCheckedAt = time.Now().Unix()
	if value != pm.IsConnected {
		message := "down"
		if value {
			message = "up"
		}
		pm.feed.Send(walletevent.Event{
			Type:     EventMarketStatusChanged,
			Accounts: []common.Address{},
			Message:  message,
			At:       time.Now().Unix(),
		})
	}
	pm.IsConnected = value
}

func (pm *Manager) makeCall(providers []thirdparty.MarketDataProvider, f func(provider thirdparty.MarketDataProvider) (interface{}, error)) (interface{}, error) {
	cmd := circuitbreaker.NewCommand(context.Background(), nil)
	for _, provider := range providers {
		provider := provider
		// FIXME: we might want a different circuitName. See other uses of NewFunctor
		circuitName := provider.ID()
		cmd.Add(circuitbreaker.NewFunctor(func() ([]interface{}, error) {
			result, err := f(provider)
			return []interface{}{result}, err
		}, circuitName, provider.ID()))
	}

	result := pm.circuitbreaker.Execute(cmd)
	pm.setIsConnected(result.Error() == nil)

	if result.Error() != nil {
		logutils.ZapLogger().Error("Error fetching prices", zap.Error(result.Error()))
		return nil, result.Error()
	}

	return result.Result()[0], nil
}

func (pm *Manager) FetchHistoricalDailyPrices(symbol string, currency string, limit int, allData bool, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	result, err := pm.makeCall(pm.providers, func(provider thirdparty.MarketDataProvider) (interface{}, error) {
		return provider.FetchHistoricalDailyPrices(symbol, currency, limit, allData, aggregate)
	})

	if err != nil {
		logutils.ZapLogger().Error("Error fetching prices", zap.Error(err))
		return nil, err
	}

	prices := result.([]thirdparty.HistoricalPrice)
	return prices, nil
}

func (pm *Manager) FetchHistoricalHourlyPrices(symbol string, currency string, limit int, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	result, err := pm.makeCall(pm.providers, func(provider thirdparty.MarketDataProvider) (interface{}, error) {
		return provider.FetchHistoricalHourlyPrices(symbol, currency, limit, aggregate)
	})

	if err != nil {
		logutils.ZapLogger().Error("Error fetching prices", zap.Error(err))
		return nil, err
	}

	prices := result.([]thirdparty.HistoricalPrice)
	return prices, nil
}

func (pm *Manager) FetchTokenMarketValues(symbols []string, currency string) (map[string]thirdparty.TokenMarketValues, error) {
	result, err := pm.makeCall(pm.providers, func(provider thirdparty.MarketDataProvider) (interface{}, error) {
		return provider.FetchTokenMarketValues(symbols, currency)
	})

	if err != nil {
		logutils.ZapLogger().Error("Error fetching prices", zap.Error(err))
		return nil, err
	}

	marketValues := result.(map[string]thirdparty.TokenMarketValues)
	return marketValues, nil
}

func (pm *Manager) updateMarketCache(currency string, marketValues map[string]thirdparty.TokenMarketValues) {
	Write(&pm.marketCache, func(tokenMarketCache TokenMarketCache) TokenMarketCache {
		for token, tokenMarketValues := range marketValues {
			if _, present := tokenMarketCache[currency]; !present {
				tokenMarketCache[currency] = make(map[string]MarketValuesSnapshot)
			}

			tokenMarketCache[currency][token] = MarketValuesSnapshot{
				UpdatedAt:    time.Now().Unix(),
				MarketValues: tokenMarketValues,
			}
		}

		return tokenMarketCache
	})
}

func (pm *Manager) GetOrFetchTokenMarketValues(symbols []string, currency string, maxAgeInSeconds int64) (map[string]thirdparty.TokenMarketValues, error) {
	// docs: Determine which token market data to fetch based on what's inside the cache and the last time the cache was updated
	symbolsToFetch := Read(&pm.marketCache, func(marketCache TokenMarketCache) []string {
		tokenMarketValuesCache, ok := marketCache[currency]
		if !ok {
			return symbols
		}

		now := time.Now().Unix()
		symbolsToFetchMap := make(map[string]bool)
		symbolsToFetch := make([]string, 0, len(symbols))

		for _, symbol := range symbols {
			marketValueSnapshot, found := tokenMarketValuesCache[symbol]
			if !found {
				if !symbolsToFetchMap[symbol] {
					symbolsToFetchMap[symbol] = true
					symbolsToFetch = append(symbolsToFetch, symbol)
				}
				continue
			}
			if now-marketValueSnapshot.UpdatedAt > maxAgeInSeconds {
				if !symbolsToFetchMap[symbol] {
					symbolsToFetchMap[symbol] = true
					symbolsToFetch = append(symbolsToFetch, symbol)
				}
				continue
			}
		}

		return symbolsToFetch
	})

	// docs: Fetch and cache the token market data for missing or stale token market data
	if len(symbolsToFetch) > 0 {
		marketValues, err := pm.FetchTokenMarketValues(symbolsToFetch, currency)
		if err != nil {
			return nil, err
		}
		pm.updateMarketCache(currency, marketValues)
	}

	// docs: Extract token market data from populated cache
	tokenMarketValues := Read(&pm.marketCache, func(tokenMarketCache TokenMarketCache) map[string]thirdparty.TokenMarketValues {
		tokenMarketValuesPerSymbol := make(map[string]thirdparty.TokenMarketValues)
		if cachedTokenMarketValues, ok := tokenMarketCache[currency]; ok {
			for _, symbol := range symbols {
				if marketValuesSnapshot, found := cachedTokenMarketValues[symbol]; found {
					tokenMarketValuesPerSymbol[symbol] = marketValuesSnapshot.MarketValues
				}
			}
		}
		return tokenMarketValuesPerSymbol
	})

	return tokenMarketValues, nil
}

func (pm *Manager) FetchTokenDetails(symbols []string) (map[string]thirdparty.TokenDetails, error) {
	result, err := pm.makeCall(pm.providers, func(provider thirdparty.MarketDataProvider) (interface{}, error) {
		return provider.FetchTokenDetails(symbols)
	})

	if err != nil {
		logutils.ZapLogger().Error("Error fetching prices", zap.Error(err))
		return nil, err
	}

	tokenDetails := result.(map[string]thirdparty.TokenDetails)
	return tokenDetails, nil
}

func (pm *Manager) FetchPrice(tokenKey string, currency string) (float64, error) {
	tokenKeys := [1]string{tokenKey}
	currencies := [1]string{currency}

	prices, err := pm.FetchPrices(tokenKeys[:], currencies[:])

	if err != nil {
		return 0, err
	}

	return prices[tokenKey][currency], nil
}

func (pm *Manager) FetchPrices(tokenKeys []string, currencies []string) (map[string]map[string]float64, error) {
	response, err := pm.makeCall(pm.providers, func(provider thirdparty.MarketDataProvider) (interface{}, error) {
		return provider.FetchPrices(tokenKeys, currencies)
	})

	if err != nil {
		logutils.ZapLogger().Error("Error fetching prices", zap.Error(err))
		return nil, err
	}

	prices, ok := response.(map[string]map[string]float64)
	if !ok {
		logutils.ZapLogger().Error("Unexpected response type", zap.Any("response", response))
		return nil, fmt.Errorf("unexpected response type: %T", response)
	}
	pm.updatePriceCache(prices)
	return prices, nil
}

func (pm *Manager) getCachedPricesFor(tokenKeys []string, currencies []string) DataPerTokenAndCurrency {
	return Read(&pm.priceCache, func(tokenPriceCache TokenPriceCache) DataPerTokenAndCurrency {
		prices := make(DataPerTokenAndCurrency)
		for _, tokenKey := range tokenKeys {
			prices[tokenKey] = make(map[string]DataPoint)
			for _, currency := range currencies {
				prices[tokenKey][currency] = tokenPriceCache[tokenKey][currency]
			}
		}
		return prices
	})
}

func (pm *Manager) updatePriceCache(prices map[string]map[string]float64) {
	Write(&pm.priceCache, func(tokenPriceCache TokenPriceCache) TokenPriceCache {
		for tokenKey, pricesPerCurrency := range prices {
			_, present := tokenPriceCache[tokenKey]
			if !present {
				tokenPriceCache[tokenKey] = make(map[string]DataPoint)
			}
			for currency, price := range pricesPerCurrency {
				tokenPriceCache[tokenKey][currency] = DataPoint{
					Price:     price,
					UpdatedAt: time.Now().Unix(),
				}
			}
		}

		return tokenPriceCache
	})
}

// Return cached price if present in cache and age is less than maxAgeInSeconds. Fetch otherwise.
func (pm *Manager) GetOrFetchPrices(tokenKeys []string, currencies []string, maxAgeInSeconds int64) (DataPerTokenAndCurrency, error) {
	tokenKeysToFetch := Read(&pm.priceCache, func(tokenPriceCache TokenPriceCache) []string {
		tokenKeysToFetchMap := make(map[string]bool)
		tokenKeysToFetch := make([]string, 0, len(tokenKeys))

		now := time.Now().Unix()

		for _, tokenKey := range tokenKeys {
			tokenPriceCache, ok := tokenPriceCache[tokenKey]
			if !ok {
				if !tokenKeysToFetchMap[tokenKey] {
					tokenKeysToFetchMap[tokenKey] = true
					tokenKeysToFetch = append(tokenKeysToFetch, tokenKey)
				}
				continue
			}
			for _, currency := range currencies {
				if now-tokenPriceCache[currency].UpdatedAt > maxAgeInSeconds {
					if !tokenKeysToFetchMap[tokenKey] {
						tokenKeysToFetchMap[tokenKey] = true
						tokenKeysToFetch = append(tokenKeysToFetch, tokenKey)
					}
					break
				}
			}
		}

		return tokenKeysToFetch
	})

	if len(tokenKeysToFetch) > 0 {
		_, err := pm.FetchPrices(tokenKeysToFetch, currencies)
		if err != nil {
			return nil, err
		}
	}

	prices := pm.getCachedPricesFor(tokenKeys, currencies)

	return prices, nil
}
