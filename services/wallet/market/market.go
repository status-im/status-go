package market

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"

	"github.com/patrickmn/go-cache"
	"github.com/status-im/status-go/circuitbreaker"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/walletevent"
	"golang.org/x/time/rate"
)

const (
	EventMarketStatusChanged walletevent.EventType = "wallet-market-status-changed"
)

type DataPoint struct {
	Price     float64
	UpdatedAt int64
}

type DataPerTokenAndCurrency = map[string]map[string]DataPoint

type Manager struct {
	feed            *event.Feed
	priceCache      DataPerTokenAndCurrency
	priceCacheLock  sync.RWMutex
	IsConnected     bool
	LastCheckedAt   int64
	IsConnectedLock sync.RWMutex
	circuitbreaker  *circuitbreaker.CircuitBreaker
	providers       []thirdparty.MarketDataProvider
	requestCounts   map[string]*atomic.Uint64
	requestCountsMu sync.RWMutex
	lastLogTime     time.Time
	cache           *cache.Cache
	rateLimiters    map[string]*rate.Limiter
}

func NewManager(providers []thirdparty.MarketDataProvider, feed *event.Feed) *Manager {
	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		Timeout:               10000,
		MaxConcurrentRequests: 100,
		SleepWindow:           300000,
		ErrorPercentThreshold: 25,
	})

	manager := &Manager{
		feed:           feed,
		priceCache:     make(DataPerTokenAndCurrency),
		IsConnected:    true,
		LastCheckedAt:  time.Now().Unix(),
		circuitbreaker: cb,
		providers:      providers,
		cache:          cache.New(5*time.Minute, 10*time.Minute),
		rateLimiters:   make(map[string]*rate.Limiter),
	}

	for _, provider := range providers {
		// Adjust the rate limit as needed for each provider
		manager.rateLimiters[provider.ID()] = rate.NewLimiter(rate.Every(time.Second), 5)
	}

	manager.initRequestCounts()
	return manager
}

func (pm *Manager) initRequestCounts() {
	pm.requestCountsMu.Lock()
	defer pm.requestCountsMu.Unlock()

	pm.requestCounts = make(map[string]*atomic.Uint64)
	for _, provider := range pm.providers {
		pm.requestCounts[provider.ID()] = &atomic.Uint64{}
	}
	pm.lastLogTime = time.Now()
}

func (pm *Manager) incrementRequestCount(providerID string) {
	pm.requestCountsMu.RLock()
	counter, exists := pm.requestCounts[providerID]
	pm.requestCountsMu.RUnlock()

	if exists {
		counter.Add(1)
	}
}

func (pm *Manager) logRequestCounts() {
	now := time.Now()
	if now.Sub(pm.lastLogTime) < time.Hour {
		return
	}

	pm.requestCountsMu.RLock()
	defer pm.requestCountsMu.RUnlock()

	for providerID, counter := range pm.requestCounts {
		count := counter.Load()
		log.Info("Market provider request count", "provider", providerID, "count", count)
		counter.Store(0) // Reset the counter
	}
	pm.lastLogTime = now
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
		cmd.Add(circuitbreaker.NewFunctor(func() ([]interface{}, error) {
			if err := pm.rateLimiters[provider.ID()].Wait(context.Background()); err != nil {
				return nil, err
			}
			pm.incrementRequestCount(provider.ID())
			result, err := f(provider)
			return []interface{}{result}, err
		}, provider.ID()))
	}

	result := pm.circuitbreaker.Execute(cmd)
	pm.setIsConnected(result.Error() == nil)

	pm.logRequestCounts() // Log request counts periodically

	if result.Error() != nil {
		log.Error("Error fetching prices", "error", result.Error())
		return nil, result.Error()
	}

	return result.Result()[0], nil
}

func (pm *Manager) FetchHistoricalDailyPrices(symbol string, currency string, limit int, allData bool, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	result, err := pm.makeCall(pm.providers, func(provider thirdparty.MarketDataProvider) (interface{}, error) {
		return provider.FetchHistoricalDailyPrices(symbol, currency, limit, allData, aggregate)
	})

	if err != nil {
		log.Error("Error fetching prices", "error", err)
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
		log.Error("Error fetching prices", "error", err)
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
		log.Error("Error fetching prices", "error", err)
		return nil, err
	}

	marketValues := result.(map[string]thirdparty.TokenMarketValues)
	return marketValues, nil
}

func (pm *Manager) FetchTokenDetails(symbols []string) (map[string]thirdparty.TokenDetails, error) {
	result, err := pm.makeCall(pm.providers, func(provider thirdparty.MarketDataProvider) (interface{}, error) {
		return provider.FetchTokenDetails(symbols)
	})

	if err != nil {
		log.Error("Error fetching prices", "error", err)
		return nil, err
	}

	tokenDetails := result.(map[string]thirdparty.TokenDetails)
	return tokenDetails, nil
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
	response, err := pm.makeCall(pm.providers, func(provider thirdparty.MarketDataProvider) (interface{}, error) {
		return provider.FetchPrices(symbols, currencies)
	})

	if err != nil {
		log.Error("Error fetching prices", "error", err)
		return nil, err
	}

	prices := response.(map[string]map[string]float64)
	pm.updatePriceCache(prices)
	return prices, nil
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
	pm.priceCacheLock.Lock()
	defer pm.priceCacheLock.Unlock()

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
	pm.priceCacheLock.RLock()
	defer pm.priceCacheLock.RUnlock()

	return pm.priceCache
}

// Return cached price if present in cache and age is less than maxAgeInSeconds. Fetch otherwise.
func (pm *Manager) GetOrFetchPrices(symbols []string, currencies []string, maxAgeInSeconds int64) (DataPerTokenAndCurrency, error) {
	symbolsToFetchMap := make(map[string]bool)
	symbolsToFetch := make([]string, 0, len(symbols))

	now := time.Now().Unix()

	for _, symbol := range symbols {
		tokenPriceCache, ok := pm.GetCachedPrices()[symbol]
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

func (pm *Manager) FetchPricesWithCache(symbols []string, currencies []string) (map[string]map[string]float64, error) {
	cacheKey := fmt.Sprintf("prices:%s:%s", strings.Join(symbols, ","), strings.Join(currencies, ","))

	if cached, found := pm.cache.Get(cacheKey); found {
		return cached.(map[string]map[string]float64), nil
	}

	prices, err := pm.FetchPrices(symbols, currencies)
	if err != nil {
		return nil, err
	}

	pm.cache.Set(cacheKey, prices, cache.DefaultExpiration)
	return prices, nil
}
