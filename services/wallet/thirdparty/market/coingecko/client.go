package coingecko

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/exp/maps"

	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/thirdparty/utils"
)

const baseURL = "https://api.coingecko.com/api/v3"

type Client struct {
	httpClient       *thirdparty.HTTPClient
	tokenIDTokenMap  map[string]GeckoToken // map[id]GeckoToken
	baseURL          string
	fetchTokensMutex sync.Mutex
}

func NewClient() *Client {
	// Configure HTTP client with detailed timeouts:
	// - 5 seconds for connection establishment (dialTimeout)
	// - 5 seconds for TLS handshake (tlsHandshakeTimeout)
	// - 5 seconds for receiving response headers (responseHeaderTimeout)
	// - 20 seconds for overall request timeout (requestTimeout)
	httpClient := thirdparty.NewHTTPClient(
		thirdparty.WithDetailedTimeouts(
			5*time.Second,
			5*time.Second,
			5*time.Second,
			20*time.Second,
		),
		thirdparty.WithMaxRetries(5),
	)

	return &Client{
		httpClient:      httpClient,
		tokenIDTokenMap: make(map[string]GeckoToken),
		baseURL:         baseURL,
	}
}

func (c *Client) ID() string {
	return "coingecko"
}

func updateLocalMaps(tokens []GeckoToken, tokenIdMap map[string]GeckoToken) {
	for _, token := range tokens {
		tokenIdMap[token.ID] = token
	}
}

func (c *Client) refreshTokens() error {
	c.fetchTokensMutex.Lock()
	defer c.fetchTokensMutex.Unlock()

	// TODO: check how often we should refresh tokens
	// From coingecko website: "Cache/Update Frequency: every 5 minutes for Pro API (Analyst, Lite, Pro, Enterprise)."
	if len(c.tokenIDTokenMap) > 0 {
		return nil
	}

	tokens, err := c.FetchTokens(context.Background())
	if err != nil {
		return err
	}

	updateLocalMaps(tokens, c.tokenIDTokenMap)
	return nil
}

func (c *Client) getTokenIDTokenMap() (map[string]GeckoToken, error) {
	err := c.refreshTokens()
	if err != nil {
		logutils.ZapLogger().Error("failed to refresh tokens", zap.Error(err))
		return nil, err
	}

	c.fetchTokensMutex.Lock()
	defer c.fetchTokensMutex.Unlock()
	return c.tokenIDTokenMap, nil
}

func mapGroupedTokensKeysToIDs(groupedTokensKeys []string, tokenIdMap map[string]GeckoToken) (mappedKeys map[string]bool, uniqueIDs []string) {
	mappedKeys = make(map[string]bool, 0)
	uniqueIDsMap := make(map[string]struct{}, 0)
	for _, gtKey := range groupedTokensKeys {
		mapped, ok := tokenIdMap[gtKey]
		if ok {
			mappedKeys[gtKey] = true
			uniqueIDsMap[mapped.ID] = struct{}{}
		} else {
			mappedKeys[gtKey] = false
		}
	}
	uniqueIDs = maps.Keys(uniqueIDsMap)
	return
}

// FetchPrices fetches the prices for the given tokens token keys (coingecko id param is used for token key) in the given currencies.
func (c *Client) FetchPrices(groupedTokensKeys []string, currencies []string) (map[string]map[string]float64, error) {
	if len(groupedTokensKeys) == 0 {
		return nil, fmt.Errorf("no tokens provided")
	}
	if len(currencies) == 0 {
		return nil, fmt.Errorf("no currencies provided")
	}
	tokenIDMap, err := c.getTokenIDTokenMap()
	if err != nil {
		return nil, err
	}
	mappedKeys, uniqueIDs := mapGroupedTokensKeysToIDs(groupedTokensKeys, tokenIDMap)
	chunkParams := utils.ChunkSymbolsParams{
		MaxSymbolsPerChunk: 400,
	}
	chunks, err := utils.ChunkSymbols(uniqueIDs, chunkParams)
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string]float64)
	for _, ids := range chunks {
		simplePrices, err := c.FetchSimplePrice(context.Background(), ids, currencies)
		if err != nil {
			return nil, err
		}

		for gtKey, mappedKey := range mappedKeys {
			result[gtKey] = map[string]float64{}
			for _, currency := range currencies {
				if !mappedKey {
					result[gtKey][currency] = 0
					continue
				}
				result[gtKey][currency] = simplePrices[gtKey][strings.ToLower(currency)]
			}
		}
	}

	return result, nil
}

// FetchTokenDetails fetches the token details for the given token keys (coingecko id param is used for token key).
func (c *Client) FetchTokenDetails(groupedTokensKeys []string) (map[string]thirdparty.TokenDetails, error) {
	if len(groupedTokensKeys) == 0 {
		return nil, fmt.Errorf("no tokens provided")
	}
	tokenIDMap, err := c.getTokenIDTokenMap()
	if err != nil {
		return nil, err
	}

	mappedKeys, _ := mapGroupedTokensKeysToIDs(groupedTokensKeys, tokenIDMap)

	result := make(map[string]thirdparty.TokenDetails)
	for gtKey, mappedKey := range mappedKeys {
		if !mappedKey {
			result[gtKey] = thirdparty.TokenDetails{}
			continue
		}
		token, ok := tokenIDMap[gtKey]
		if ok {
			result[gtKey] = thirdparty.TokenDetails{
				ID:     token.ID,
				Name:   token.Name,
				Symbol: token.Symbol,
			}
		} else {
			// should never be reached
			logutils.ZapLogger().Error("token not found", zap.String("tokenKey", gtKey))
		}
	}

	return result, nil
}

// FetchTokenMarketValues fetches the market values for the given token keys (coingecko id param is used for token key) in the given currency.
func (c *Client) FetchTokenMarketValues(groupedTokensKeys []string, currency string) (map[string]thirdparty.TokenMarketValues, error) {
	if len(groupedTokensKeys) == 0 {
		return nil, fmt.Errorf("no tokens provided")
	}
	if currency == "" {
		return nil, fmt.Errorf("no currency provided")
	}
	tokenIDMap, err := c.getTokenIDTokenMap()
	if err != nil {
		return nil, err
	}

	mappedKeys, uniqueIDs := mapGroupedTokensKeysToIDs(groupedTokensKeys, tokenIDMap)

	chunkParams := utils.ChunkSymbolsParams{
		MaxSymbolsPerChunk: 400,
	}
	chunks, err := utils.ChunkSymbols(uniqueIDs, chunkParams)
	if err != nil {
		return nil, err
	}

	result := make(map[string]thirdparty.TokenMarketValues)
	for _, ids := range chunks {
		marketValues, err := c.FetchCoinsMarkets(context.Background(), ids, currency)
		if err != nil {
			return nil, err
		}

		for gtKey, mappedKey := range mappedKeys {
			if !mappedKey {
				result[gtKey] = thirdparty.TokenMarketValues{}
				continue
			}
			for _, marketValue := range marketValues {
				if gtKey != marketValue.ID {
					continue
				}

				result[gtKey] = thirdparty.TokenMarketValues{
					MKTCAP:          marketValue.MarketCap,
					HIGHDAY:         marketValue.High24h,
					LOWDAY:          marketValue.Low24h,
					CHANGEPCTHOUR:   marketValue.PriceChangePercentage1hInCurrency,
					CHANGEPCTDAY:    marketValue.PriceChangePercentage24h,
					CHANGEPCT24HOUR: marketValue.PriceChangePercentage24h,
					CHANGE24HOUR:    marketValue.PriceChange24h,
				}
			}
		}
	}

	return result, nil
}

// FetchHistoricalHourlyPrices fetches the historical hourly prices for the given token key (coingecko id param is used for token key) in the given currency.
func (c *Client) FetchHistoricalHourlyPrices(groupedTokensKey string, currency string, limit int, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	return []thirdparty.HistoricalPrice{}, nil
}

// FetchHistoricalDailyPrices fetches the historical daily prices for the given token key (coingecko id param is used for token key) in the given currency.
func (c *Client) FetchHistoricalDailyPrices(groupedTokensKey string, currency string, limit int, allData bool, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	if groupedTokensKey == "" {
		return nil, fmt.Errorf("no token provided")
	}
	if currency == "" {
		return nil, fmt.Errorf("no currency provided")
	}
	tokenIDMap, err := c.getTokenIDTokenMap()
	if err != nil {
		return nil, err
	}

	_, uniqueIDs := mapGroupedTokensKeysToIDs([]string{groupedTokensKey}, tokenIDMap)
	container, err := c.FetchHistoryMarketData(context.Background(), uniqueIDs[0], currency) // since tokenKey cannot be empty, we can safely access uniqueIDs[0]
	if err != nil {
		return nil, err
	}

	result := make([]thirdparty.HistoricalPrice, 0)
	for _, price := range container.Prices {
		result = append(result, thirdparty.HistoricalPrice{
			Timestamp: int64(price[0]),
			Value:     price[1],
		})
	}

	return result, nil
}
