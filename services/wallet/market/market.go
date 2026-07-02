package market

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/status-go/internal/circuitbreaker"
	provider_errors "github.com/status-im/status-go/internal/healthmanager/provider_errors"
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"
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

type TokenManagerInterface interface {
	GetTokensForFetchingMarketData() ([]*tokentypes.Token, error)
	GetTokensByKeysForFetchingMarketData(tokenKeys []string) ([]*tokentypes.Token, error)
	GetTokenByKey(tokenKey string) (*tokentypes.Token, error)
}

type Manager struct {
	tokenManager    TokenManagerInterface
	feed            *event.Feed
	priceCache      MarketCache[TokenPriceCache]
	marketCache     MarketCache[TokenMarketCache]
	IsConnected     bool
	LastCheckedAt   int64
	IsConnectedLock sync.RWMutex
	circuitbreaker  *circuitbreaker.CircuitBreaker
	providers       []thirdparty.MarketDataProvider
}

func NewManager(providers []thirdparty.MarketDataProvider, tokenManager TokenManagerInterface, feed *event.Feed) *Manager {
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
	cmd.SetNonFailureErrorClassifier(provider_errors.IsIgnorableForConnectivity)
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
	if err := result.Error(); err != nil {
		isIgnorableError := provider_errors.IsIgnorableForConnectivity(err)
		pm.setIsConnected(isIgnorableError)
		if isIgnorableError {
			logutils.ZapLogger().Warn("market data unavailable for token mapping", zap.Error(err))
		} else {
			logutils.ZapLogger().Error("Error fetching prices", zap.Error(err))
		}
		return nil, err
	}

	pm.setIsConnected(true)
	return result.Result()[0], nil
}

func (pm *Manager) fetchHistoricalPricesWithFallback(
	token *tokentypes.Token,
	call func(provider thirdparty.MarketDataProvider, token *tokentypes.Token) ([]thirdparty.HistoricalPrice, error),
) ([]thirdparty.HistoricalPrice, error) {
	var sibling *tokentypes.Token
	if allTokens, err := pm.tokenManager.GetTokensForFetchingMarketData(); err == nil {
		sibling = tokentypes.EthereumMainnetSibling(token, allTokens)
	}

	for _, t := range []*tokentypes.Token{token, sibling} {
		if t == nil {
			continue
		}

		result, err := pm.makeCall(pm.providers, func(provider thirdparty.MarketDataProvider) (interface{}, error) {
			return call(provider, t)
		})
		if err == nil {
			return result.([]thirdparty.HistoricalPrice), nil
		}
		if !(provider_errors.IsIgnorableForConnectivity(err) && errors.Is(err, thirdparty.ErrTokenNotMapped)) {
			return nil, err
		}
	}

	logutils.ZapLogger().Warn("no mapped market history token found",
		zap.String("tokenKey", token.Key()),
		zap.String("crossChainID", token.CrossChainID))
	return []thirdparty.HistoricalPrice{}, nil
}

func (pm *Manager) getTokensByKeys(tokensKeys []string) (tokens []*tokentypes.Token, err error) {
	if len(tokensKeys) > 0 {
		tokens, err = pm.tokenManager.GetTokensByKeysForFetchingMarketData(tokensKeys)
	} else {
		tokens, err = pm.tokenManager.GetTokensForFetchingMarketData()
	}
	return
}

func (pm *Manager) FetchHistoricalDailyPrices(tokenKey string, currency string, limit int, allData bool, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	token, err := pm.tokenManager.GetTokenByKey(tokenKey)
	if err != nil {
		return nil, err
	}

	return pm.fetchHistoricalPricesWithFallback(token, func(provider thirdparty.MarketDataProvider, t *tokentypes.Token) ([]thirdparty.HistoricalPrice, error) {
		return provider.FetchHistoricalDailyPrices(t, currency, limit, allData, aggregate)
	})
}

func (pm *Manager) FetchHistoricalHourlyPrices(tokenKey string, currency string, limit int, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	token, err := pm.tokenManager.GetTokenByKey(tokenKey)
	if err != nil {
		return nil, err
	}

	return pm.fetchHistoricalPricesWithFallback(token, func(provider thirdparty.MarketDataProvider, t *tokentypes.Token) ([]thirdparty.HistoricalPrice, error) {
		return provider.FetchHistoricalHourlyPrices(t, currency, limit, aggregate)
	})
}

// FetchTokenMarketValues fetches market values for a given token keys and currency. If no tokens are provided, all tokens are fetched.
func (pm *Manager) FetchTokenMarketValues(tokensKeys []string, currency string) (map[string]thirdparty.TokenMarketValues, error) {
	allTokens, err := pm.getTokensByKeys(tokensKeys)
	if err != nil {
		return nil, err
	}

	result, err := pm.makeCall(pm.providers, func(provider thirdparty.MarketDataProvider) (interface{}, error) {
		return provider.FetchTokenMarketValues(allTokens, currency)
	})

	if err != nil {
		logutils.ZapLogger().Error("Error fetching prices", zap.Error(err))
		return nil, err
	}

	marketValues := result.(map[string]thirdparty.TokenMarketValues)
	return marketValues, nil
}

// FetchTokenDetails fetches token details for a given tokens. If no tokens are provided, all tokens are fetched.
func (pm *Manager) FetchTokenDetails(tokensKeys []string) (map[string]thirdparty.TokenDetails, error) {
	allTokens, err := pm.getTokensByKeys(tokensKeys)
	if err != nil {
		return nil, err
	}

	result, err := pm.makeCall(pm.providers, func(provider thirdparty.MarketDataProvider) (interface{}, error) {
		return provider.FetchTokenDetails(allTokens)
	})

	if err != nil {
		logutils.ZapLogger().Error("Error fetching prices", zap.Error(err))
		return nil, err
	}

	tokenDetails := result.(map[string]thirdparty.TokenDetails)
	return tokenDetails, nil
}

// FetchPrices fetches prices for a given token keys and currencies. If no tokens are provided, all tokens are fetched.
func (pm *Manager) FetchPrices(tokensKeys []string, currencies []string) (map[string]map[string]float64, error) {
	allTokens, err := pm.getTokensByKeys(tokensKeys)
	if err != nil {
		return nil, err
	}

	response, err := pm.makeCall(pm.providers, func(provider thirdparty.MarketDataProvider) (interface{}, error) {
		return provider.FetchPrices(allTokens, currencies)
	})

	if err != nil {
		logutils.ZapLogger().Error("Error fetching prices", zap.Error(err))
		return nil, err
	}

	pricesPerSymbolCurrencies := response.(map[string]map[string]float64)
	pm.updatePriceCache(pricesPerSymbolCurrencies)

	return pricesPerSymbolCurrencies, nil
}

func (pm *Manager) getCachedPricesFor(tokensKeys []string, currencies []string) DataPerTokenAndCurrency {
	return Read(&pm.priceCache, func(tokenPriceCache TokenPriceCache) DataPerTokenAndCurrency {
		prices := make(DataPerTokenAndCurrency)
		for _, tokenKey := range tokensKeys {
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
func (pm *Manager) GetOrFetchPrices(tokensKeys []string, currencies []string, maxAgeInSeconds int64) (DataPerTokenAndCurrency, error) {
	tokensToFetch := Read(&pm.priceCache, func(tokenPriceCache TokenPriceCache) []string {
		tokensToFetchMap := make(map[string]bool)
		tokensToFetch := make([]string, 0, len(tokensKeys))

		now := time.Now().Unix()

		for _, tokenKey := range tokensKeys {
			tokenPriceCache, ok := tokenPriceCache[tokenKey]
			if !ok {
				if !tokensToFetchMap[tokenKey] {
					tokensToFetchMap[tokenKey] = true
					tokensToFetch = append(tokensToFetch, tokenKey)
				}
				continue
			}
			for _, currency := range currencies {
				if now-tokenPriceCache[currency].UpdatedAt > maxAgeInSeconds {
					if !tokensToFetchMap[tokenKey] {
						tokensToFetchMap[tokenKey] = true
						tokensToFetch = append(tokensToFetch, tokenKey)
					}
					break
				}
			}
		}

		return tokensToFetch
	})

	if len(tokensToFetch) > 0 {
		_, err := pm.FetchPrices(tokensToFetch, currencies)
		if err != nil {
			return nil, err
		}
	}

	prices := pm.getCachedPricesFor(tokensToFetch, currencies)

	return prices, nil
}
