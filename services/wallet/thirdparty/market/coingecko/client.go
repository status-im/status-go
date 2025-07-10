package coingecko

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/exp/maps"

	"github.com/status-im/status-go/pkg/security"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/thirdparty/utils"
)

const baseURL = "https://api.coingecko.com/api/v3"

const (
	requestDelay     = 500 * time.Millisecond
	pricesChunkLimit = 500
	tokensChunkLimit = 250
)

var coinGeckoMapping = map[string]string{
	"STT":   "status",
	"SNT":   "status",
	"ETH":   "ethereum",
	"AST":   "airswap",
	"ABT":   "arcblock",
	"BNB":   "binancecoin",
	"BLT":   "bloom",
	"COMP":  "compound-coin",
	"EDG":   "edgeless",
	"ENG":   "enigma",
	"EOS":   "eos",
	"GEN":   "daostack",
	"MANA":  "decentraland-wormhole",
	"LEND":  "ethlend",
	"LRC":   "loopring",
	"MET":   "metronome",
	"POLY":  "polymath",
	"PPT":   "populous",
	"SAN":   "santiment-network-token",
	"DNT":   "district0x",
	"SPN":   "sapien",
	"USDS":  "stableusd",
	"STX":   "stox",
	"SUB":   "substratum",
	"PAY":   "tenx",
	"GRT":   "the-graph",
	"TNT":   "tierion",
	"TRX":   "tron",
	"RARE":  "superrare",
	"UNI":   "uniswap",
	"USDC":  "usd-coin",
	"USDP":  "paxos-standard",
	"USDT":  "tether",
	"SHIB":  "shiba-inu",
	"LINK":  "chainlink",
	"MATIC": "matic-network",
	"DAI":   "dai",
	"ARB":   "arbitrum",
	"OP":    "optimism",
}

type Client struct {
	httpClient       *thirdparty.HTTPClient
	tokens           map[string][]GeckoToken
	baseURL          string
	creds            *thirdparty.BasicCreds
	fetchTokensMutex sync.Mutex
}

type Params struct {
	URL      string
	User     security.SensitiveString
	Password security.SensitiveString
}

func NewClient() *Client {
	return NewClientWithParams(Params{
		URL:      baseURL,
		User:     security.SensitiveString{},
		Password: security.SensitiveString{},
	})
}

func NewClientWithParams(params Params) *Client {
	var creds *thirdparty.BasicCreds
	if !params.User.Empty() {
		creds = &thirdparty.BasicCreds{
			User:     params.User,
			Password: params.Password,
		}
	}

	httpClient := thirdparty.NewHTTPClient(
		thirdparty.WithDetailedTimeouts(
			5*time.Second,  // dialTimeout
			5*time.Second,  // tlsHandshakeTimeout
			5*time.Second,  // responseHeaderTimeout
			20*time.Second, // requestTimeout
		),
		thirdparty.WithMaxRetries(5),
	)

	// Ensure baseURL doesn't end with a slash
	clientBaseURL := strings.TrimSuffix(params.URL, "/")
	if clientBaseURL == "" {
		clientBaseURL = baseURL
	}

	return &Client{
		httpClient: httpClient,
		tokens:     make(map[string][]GeckoToken),
		baseURL:    clientBaseURL,
		creds:      creds,
	}
}

func (c *Client) ID() string {
	return "coingecko"
}

func mapTokensToSymbols(tokens []GeckoToken, tokenMap map[string][]GeckoToken) {
	for _, token := range tokens {
		symbol := strings.ToUpper(token.Symbol)
		if id, ok := coinGeckoMapping[symbol]; ok {
			if id != token.ID {
				continue
			}
		}
		tokenMap[symbol] = append(tokenMap[symbol], token)
	}
}

func getGeckoTokenFromSymbol(tokens map[string][]GeckoToken, symbol string) (GeckoToken, error) {
	tokenList, ok := tokens[strings.ToUpper(symbol)]
	if !ok {
		return GeckoToken{}, fmt.Errorf("token not found for symbol %s", symbol)
	}
	for _, t := range tokenList {
		if t.EthPlatform {
			return t, nil
		}
	}
	return tokenList[0], nil
}

func getIDFromSymbol(tokens map[string][]GeckoToken, symbol string) (string, error) {
	token, err := getGeckoTokenFromSymbol(tokens, symbol)
	if err != nil {
		return "", err
	}
	return token.ID, nil
}

func (c *Client) getTokens() (map[string][]GeckoToken, error) {
	c.fetchTokensMutex.Lock()
	defer c.fetchTokensMutex.Unlock()

	if len(c.tokens) > 0 {
		return c.tokens, nil
	}

	tokens, err := c.FetchTokens(context.Background())
	if err != nil {
		return nil, err
	}

	mapTokensToSymbols(tokens, c.tokens)
	return c.tokens, nil
}

func (c *Client) mapSymbolsToIds(symbols []string) (mappedSymbols map[string]string, unmappedSymbols []string, err error) {
	tokens, err := c.getTokens()
	if err != nil {
		return nil, nil, err
	}
	mappedSymbols = make(map[string]string, 0)
	unmappedSymbols = make([]string, 0)
	for _, symbol := range symbols {
		id, err := getIDFromSymbol(tokens, utils.GetRealSymbol(symbol))
		if err != nil {
			unmappedSymbols = append(unmappedSymbols, symbol)
		} else {
			mappedSymbols[symbol] = id
		}
	}

	return mappedSymbols, unmappedSymbols, nil
}

func (c *Client) FetchPrices(symbols []string, currencies []string) (map[string]map[string]float64, error) {
	mappedSymbols, unmappedSymbols, err := c.mapSymbolsToIds(symbols)
	if err != nil {
		return nil, err
	}

	simplePrices, err := utils.ChunkMapFetcher[CurrencyPriceMap](
		context.Background(),
		maps.Values(mappedSymbols),
		pricesChunkLimit,
		requestDelay,
		func(ctx context.Context, chunkIds []string) (map[string]CurrencyPriceMap, error) {
			return c.FetchSimplePrice(ctx, chunkIds, currencies)
		},
	)
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string]float64)
	for symbol, id := range mappedSymbols {
		result[symbol] = map[string]float64{}
		for _, currency := range currencies {
			result[symbol][currency] = simplePrices[id][strings.ToLower(currency)]
		}
	}

	for _, symbol := range unmappedSymbols {
		result[symbol] = map[string]float64{}
		for _, currency := range currencies {
			result[symbol][currency] = 0
		}
	}

	return result, nil
}

func (c *Client) FetchTokenDetails(symbols []string) (map[string]thirdparty.TokenDetails, error) {
	tokens, err := c.getTokens()
	if err != nil {
		return nil, err
	}
	result := make(map[string]thirdparty.TokenDetails)
	for _, symbol := range symbols {
		token, err := getGeckoTokenFromSymbol(tokens, utils.GetRealSymbol(symbol))
		if err == nil {
			result[symbol] = thirdparty.TokenDetails{
				ID:     token.ID,
				Name:   token.Name,
				Symbol: symbol,
			}
		}
	}
	return result, nil
}

func (c *Client) FetchTokenMarketValues(symbols []string, currency string) (map[string]thirdparty.TokenMarketValues, error) {
	mappedSymbols, unmappedSymbols, err := c.mapSymbolsToIds(symbols)
	if err != nil {
		return nil, err
	}

	marketValues, err := utils.ChunkArrayFetcher[GeckoMarketValues](
		context.Background(),
		maps.Values(mappedSymbols),
		tokensChunkLimit,
		requestDelay,
		func(ctx context.Context, chunkIds []string) ([]GeckoMarketValues, error) {
			return c.FetchCoinsMarkets(ctx, chunkIds, currency)
		},
	)
	if err != nil {
		return nil, err
	}

	result := make(map[string]thirdparty.TokenMarketValues)
	for symbol, id := range mappedSymbols {
		for _, marketValue := range marketValues {
			if id != marketValue.ID {
				continue
			}

			result[symbol] = thirdparty.TokenMarketValues{
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

	for _, symbol := range unmappedSymbols {
		result[symbol] = thirdparty.TokenMarketValues{}
	}

	return result, nil
}

func (c *Client) FetchHistoricalHourlyPrices(symbol string, currency string, limit int, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	days := (limit + 23) / 24 // round up
	if days == 0 {
		days = 1 // minimum 1 day
	}
	if days > 90 {
		days = 90 // maximum 90 days
	}

	return c.FetchHistoricalDailyPrices(symbol, currency, days, false, aggregate)
}

func (c *Client) FetchHistoricalDailyPrices(symbol string, currency string, limit int, allData bool, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	tokens, err := c.getTokens()
	if err != nil {
		return nil, err
	}

	id, err := getIDFromSymbol(tokens, utils.GetRealSymbol(symbol))
	if err != nil {
		return nil, err
	}

	var days string
	if allData {
		days = "max"
	} else {
		days = fmt.Sprintf("%d", limit)
	}

	container, err := c.FetchHistoryMarketData(context.Background(), id, currency, days)
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

// doGetRequestWithOptionalAuth performs a GET request, using credentials if they are available
func (c *Client) doGetRequestWithOptionalAuth(ctx context.Context, url string, params url.Values) ([]byte, error) {
	if c.creds != nil {
		return c.httpClient.DoGetRequestWithCredentials(ctx, url, params, c.creds)
	}
	return c.httpClient.DoGetRequest(ctx, url, params)
}
