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
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/thirdparty/utils"
)

const (
	baseURL = "https://api.coingecko.com/api/v3"

	ethTokenID = "ethereum"
)

type Client struct {
	httpClient       *thirdparty.HTTPClient
	tokenIDTokenMap  map[string]GeckoToken // map[id]GeckoToken
	tokenKeyIDMap    map[string]string     // map[tokenKey]id
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
		tokenKeyIDMap:   make(map[string]string),
		baseURL:         baseURL,
	}
}

func (c *Client) ID() string {
	return "coingecko"
}

func updateLocalMaps(tokens []GeckoToken, tokenIdMap map[string]GeckoToken, tokenKeyMap map[string]string) {
	for _, token := range tokens {
		tokenIdMap[token.ID] = token
		for _, chainID := range common.AllChains() {
			tokenKey, err := token.getKeyForChain(chainID)
			if err != nil {
				continue
			}
			tokenKeyMap[tokenKey] = token.ID
		}
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

	updateLocalMaps(tokens, c.tokenIDTokenMap, c.tokenKeyIDMap)
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

func (c *Client) getTokenKeyIDMap() (map[string]string, error) {
	err := c.refreshTokens()
	if err != nil {
		logutils.ZapLogger().Error("failed to refresh tokens", zap.Error(err))
		return nil, err
	}

	c.fetchTokensMutex.Lock()
	defer c.fetchTokensMutex.Unlock()
	return c.tokenKeyIDMap, nil
}

func mapTokenKeysToIDs(tokenKeys []string, tokenKeyMap map[string]string) (mappedKeys map[string]string, uniqueIDs []string, unmappedKeys []string) {
	mappedKeys = make(map[string]string, 0)
	uniqueIDsMap := make(map[string]struct{}, 0)
	unmappedKeys = make([]string, 0)
	for _, tokenKey := range tokenKeys {
		tokenKeyUpper := strings.ToUpper(tokenKey) // use upper case for search and comparison
		id, ok := tokenKeyMap[tokenKeyUpper]
		if ok {
			mappedKeys[tokenKey] = id
			uniqueIDsMap[id] = struct{}{}
		} else {
			if strings.Contains(tokenKeyUpper, strings.ToUpper(utils.EthAddress)) {
				mappedKeys[tokenKey] = ethTokenID
				uniqueIDsMap[ethTokenID] = struct{}{}
				continue
			}
			unmappedKeys = append(unmappedKeys, tokenKey)
		}
	}
	uniqueIDs = maps.Keys(uniqueIDsMap)
	return
}

// FetchPrices fetches the prices for the given tokens token keys (following token key pattern) in the given currencies.
// When providing a token key for ETH, use `0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE` address.
func (c *Client) FetchPrices(tokenKeys []string, currencies []string) (map[string]map[string]float64, error) {
	tokenKeyIDMap, err := c.getTokenKeyIDMap()
	if err != nil {
		return nil, err
	}
	mappedKeys, uniqueIDs, unmappedKeys := mapTokenKeysToIDs(tokenKeys, tokenKeyIDMap)

	simplePrices, err := c.FetchSimplePrice(context.Background(), uniqueIDs, currencies)
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string]float64)
	for tokenKey, id := range mappedKeys {
		result[tokenKey] = map[string]float64{}
		for _, currency := range currencies {
			result[tokenKey][currency] = simplePrices[id][strings.ToLower(currency)]
		}
	}

	for _, tokenKey := range unmappedKeys {
		result[tokenKey] = map[string]float64{}
		for _, currency := range currencies {
			result[tokenKey][currency] = 0
		}
	}

	return result, nil
}

// FetchTokenDetails fetches the token details for the given token keys (following token key pattern).
// When providing a token key for ETH, use `0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE` address.
func (c *Client) FetchTokenDetails(tokenKeys []string) (map[string]thirdparty.TokenDetails, error) {
	tokenKeyIDMap, err := c.getTokenKeyIDMap()
	if err != nil {
		return nil, err
	}

	tokenIDMap, err := c.getTokenIDTokenMap()
	if err != nil {
		return nil, err
	}

	mappedKeys, _, unmappedKeys := mapTokenKeysToIDs(tokenKeys, tokenKeyIDMap)

	result := make(map[string]thirdparty.TokenDetails)
	for tokenKey, id := range mappedKeys {
		token, ok := tokenIDMap[id]
		if ok {
			result[tokenKey] = thirdparty.TokenDetails{
				ID:     token.ID,
				Name:   token.Name,
				Symbol: token.Symbol,
			}
		} else {
			// should never be reached
			logutils.ZapLogger().Error("token not found", zap.String("tokenKey", tokenKey), zap.String("id", id))
		}
	}

	for _, tokenKey := range unmappedKeys {
		result[tokenKey] = thirdparty.TokenDetails{}
	}

	return result, nil
}

// FetchTokenMarketValues fetches the market values for the given token keys (following token key pattern) in the given currency.
// When providing a token key for ETH, use `0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE` address.
func (c *Client) FetchTokenMarketValues(tokenKeys []string, currency string) (map[string]thirdparty.TokenMarketValues, error) {
	tokenKeyIDMap, err := c.getTokenKeyIDMap()
	if err != nil {
		return nil, err
	}

	mappedKeys, uniqueIDs, unmappedKeys := mapTokenKeysToIDs(tokenKeys, tokenKeyIDMap)

	marketValues, err := c.FetchCoinsMarkets(context.Background(), uniqueIDs, currency)
	if err != nil {
		return nil, err
	}

	result := make(map[string]thirdparty.TokenMarketValues)
	for tokenKey, id := range mappedKeys {
		for _, marketValue := range marketValues {
			if id != marketValue.ID {
				continue
			}

			result[tokenKey] = thirdparty.TokenMarketValues{
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

	for _, symbol := range unmappedKeys {
		result[symbol] = thirdparty.TokenMarketValues{}
	}

	return result, nil
}

func (c *Client) FetchHistoricalHourlyPrices(tokenKey string, currency string, limit int, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	return []thirdparty.HistoricalPrice{}, nil
}

func (c *Client) FetchHistoricalDailyPrices(tokenKey string, currency string, limit int, allData bool, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	tokenKeyIDMap, err := c.getTokenKeyIDMap()
	if err != nil {
		return nil, err
	}

	mappedKeys, _, _ := mapTokenKeysToIDs([]string{tokenKey}, tokenKeyIDMap)
	if len(mappedKeys) != 1 {
		logutils.ZapLogger().Error("token not found", zap.String("tokenKey", tokenKey))
		return nil, fmt.Errorf("token not found")
	}

	container, err := c.FetchHistoryMarketData(context.Background(), mappedKeys[tokenKey], currency)
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
