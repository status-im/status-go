package market

import (
	"context"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/exp/maps"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/status-go/circuitbreaker"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/token"
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
	tokenManager    *token.Manager
	feed            *event.Feed
	priceCache      MarketCache[TokenPriceCache]
	marketCache     MarketCache[TokenMarketCache]
	IsConnected     bool
	LastCheckedAt   int64
	IsConnectedLock sync.RWMutex
	circuitbreaker  *circuitbreaker.CircuitBreaker
	providers       []thirdparty.MarketDataProvider
}

func NewManager(providers []thirdparty.MarketDataProvider, tokenManager *token.Manager, feed *event.Feed) *Manager {
	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		Timeout:               60000,
		MaxConcurrentRequests: 100,
		SleepWindow:           300000,
		ErrorPercentThreshold: 25,
	})

	return &Manager{
		tokenManager:   tokenManager,
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
func (pm *Manager) symbolProviderSymbolMaps(symbols []string) (symbolsToProviderSymbols map[string]string, providerSymbolsToSymbols map[string][]string, err error) {
	symbolsToProviderSymbols = make(map[string]string)
	providerSymbolsToSymbols = make(map[string][]string)

	allTokens, err := pm.tokenManager.GetAllTokens()
	if err != nil {
		return
	}
	for _, symbol := range symbols {
		found := false
		for _, token := range allTokens {
			if strings.EqualFold(token.Symbol, symbol) || strings.EqualFold(token.TmpSymbol, symbol) {
				found = true
				symbolsToProviderSymbols[symbol] = token.TmpSymbol
				providerSymbolsToSymbols[token.TmpSymbol] = append(providerSymbolsToSymbols[token.TmpSymbol], symbol)
				break
			}
		}
		if !found {
			symbolsToProviderSymbols[symbol] = symbol
			providerSymbolsToSymbols[symbol] = append(providerSymbolsToSymbols[symbol], symbol)
		}
	}
	return
}

func (pm *Manager) FetchHistoricalDailyPrices(symbol string, currency string, limit int, allData bool, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	symbolsToProviderSymbols, _, err := pm.symbolProviderSymbolMaps([]string{symbol})
	if err != nil {
		logutils.ZapLogger().Error("Error mapping symbols to provider symbols", zap.Error(err))
		return nil, err
	}

	result, err := pm.makeCall(pm.providers, func(provider thirdparty.MarketDataProvider) (interface{}, error) {
		return provider.FetchHistoricalDailyPrices(symbolsToProviderSymbols[symbol], currency, limit, allData, aggregate)
	})

	if err != nil {
		logutils.ZapLogger().Error("Error fetching prices", zap.Error(err))
		return nil, err
	}

	prices := result.([]thirdparty.HistoricalPrice)
	return prices, nil
}

func (pm *Manager) FetchHistoricalHourlyPrices(symbol string, currency string, limit int, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	symbolsToProviderSymbols, _, err := pm.symbolProviderSymbolMaps([]string{symbol})
	if err != nil {
		logutils.ZapLogger().Error("Error mapping symbols to provider symbols", zap.Error(err))
		return nil, err
	}

	result, err := pm.makeCall(pm.providers, func(provider thirdparty.MarketDataProvider) (interface{}, error) {
		return provider.FetchHistoricalHourlyPrices(symbolsToProviderSymbols[symbol], currency, limit, aggregate)
	})

	if err != nil {
		logutils.ZapLogger().Error("Error fetching prices", zap.Error(err))
		return nil, err
	}

	prices := result.([]thirdparty.HistoricalPrice)
	return prices, nil
}

func (pm *Manager) FetchTokenMarketValues(symbols []string, currency string) (map[string]thirdparty.TokenMarketValues, error) {
	symbolsToProviderSymbols, providerSymbolsToSymbols, err := pm.symbolProviderSymbolMaps(symbols)
	if err != nil {
		logutils.ZapLogger().Error("Error mapping symbols to provider symbols", zap.Error(err))
		return nil, err
	}

	result, err := pm.makeCall(pm.providers, func(provider thirdparty.MarketDataProvider) (interface{}, error) {
		return provider.FetchTokenMarketValues(maps.Values(symbolsToProviderSymbols), currency)
	})

	if err != nil {
		logutils.ZapLogger().Error("Error fetching prices", zap.Error(err))
		return nil, err
	}

	mappedMarketValues := make(map[string]thirdparty.TokenMarketValues)
	marketValues := result.(map[string]thirdparty.TokenMarketValues)
	for providerSymbol, tokenMarketValues := range marketValues {
		symbols := providerSymbolsToSymbols[providerSymbol]
		for _, symbol := range symbols {
			mappedMarketValues[symbol] = tokenMarketValues
		}
	}
	return mappedMarketValues, nil
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
	symbolsToProviderSymbols, providerSymbolsToSymbols, err := pm.symbolProviderSymbolMaps(symbols)
	if err != nil {
		logutils.ZapLogger().Error("Error mapping symbols to provider symbols", zap.Error(err))
		return nil, err
	}

	result, err := pm.makeCall(pm.providers, func(provider thirdparty.MarketDataProvider) (interface{}, error) {
		return provider.FetchTokenDetails(maps.Values(symbolsToProviderSymbols))
	})

	if err != nil {
		logutils.ZapLogger().Error("Error fetching prices", zap.Error(err))
		return nil, err
	}

	mappedTokenDetails := make(map[string]thirdparty.TokenDetails)
	tokenDetails := result.(map[string]thirdparty.TokenDetails)
	for providerSymbol, tokenDetail := range tokenDetails {
		symbols := providerSymbolsToSymbols[providerSymbol]
		for _, symbol := range symbols {
			mappedTokenDetails[symbol] = tokenDetail
		}
	}
	return mappedTokenDetails, nil
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
	symbolsToProviderSymbols, providerSymbolsToSymbols, err := pm.symbolProviderSymbolMaps(symbols)
	if err != nil {
		logutils.ZapLogger().Error("Error mapping symbols to provider symbols", zap.Error(err))
		return nil, err
	}

	response, err := pm.makeCall(pm.providers, func(provider thirdparty.MarketDataProvider) (interface{}, error) {
		return provider.FetchPrices(maps.Values(symbolsToProviderSymbols), currencies)
	})

	if err != nil {
		logutils.ZapLogger().Error("Error fetching prices", zap.Error(err))
		return nil, err
	}

	mappedPrices := make(map[string]map[string]float64)
	pricesPerSymbolCurrencies := response.(map[string]map[string]float64)
	for providerSymbol, prices := range pricesPerSymbolCurrencies {
		symbols := providerSymbolsToSymbols[providerSymbol]
		for _, symbol := range symbols {
			mappedPrices[symbol] = prices
		}
	}

	pm.updatePriceCache(mappedPrices)

	return mappedPrices, nil
}

func (pm *Manager) getCachedPricesFor(symbols []string, currencies []string) DataPerTokenAndCurrency {
	return Read(&pm.priceCache, func(tokenPriceCache TokenPriceCache) DataPerTokenAndCurrency {
		prices := make(DataPerTokenAndCurrency)
		for _, symbol := range symbols {
			prices[symbol] = make(map[string]DataPoint)
			for _, currency := range currencies {
				prices[symbol][currency] = tokenPriceCache[symbol][currency]
			}
		}
		return prices
	})
}

func (pm *Manager) updatePriceCache(prices map[string]map[string]float64) {
	Write(&pm.priceCache, func(tokenPriceCache TokenPriceCache) TokenPriceCache {
		for token, pricesPerCurrency := range prices {
			_, present := tokenPriceCache[token]
			if !present {
				tokenPriceCache[token] = make(map[string]DataPoint)
			}
			for currency, price := range pricesPerCurrency {
				tokenPriceCache[token][currency] = DataPoint{
					Price:     price,
					UpdatedAt: time.Now().Unix(),
				}
			}
		}

		return tokenPriceCache
	})
}

// Return cached price if present in cache and age is less than maxAgeInSeconds. Fetch otherwise.
func (pm *Manager) GetOrFetchPrices(symbols []string, currencies []string, maxAgeInSeconds int64) (DataPerTokenAndCurrency, error) {
	symbolsToFetch := Read(&pm.priceCache, func(tokenPriceCache TokenPriceCache) []string {
		symbolsToFetchMap := make(map[string]bool)
		symbolsToFetch := make([]string, 0, len(symbols))

		now := time.Now().Unix()

		for _, symbol := range symbols {
			tokenPriceCache, ok := tokenPriceCache[symbol]
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

		return symbolsToFetch
	})

	if len(symbolsToFetch) > 0 {
		_, err := pm.FetchPrices(symbolsToFetch, currencies)
		if err != nil {
			return nil, err
		}
	}

	prices := pm.getCachedPricesFor(symbols, currencies)

	return prices, nil
}
