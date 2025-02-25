package coingecko

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/exp/maps"

	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/thirdparty/utils"
)

const baseURL = "https://api.coingecko.com/api/v3"

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
	fetchTokensMutex sync.Mutex
}

func NewClient() *Client {
	return &Client{
		httpClient: thirdparty.NewHTTPClient(),
		tokens:     make(map[string][]GeckoToken),
		baseURL:    baseURL,
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

func (c *Client) mapSymbolsToIds(symbols []string) (map[string]string, error) {
	tokens, err := c.getTokens()
	if err != nil {
		return nil, err
	}
	ids := make(map[string]string, 0)
	for _, symbol := range symbols {
		id, err := getIDFromSymbol(tokens, utils.GetRealSymbol(symbol))
		if err == nil {
			ids[symbol] = id
		}
	}

	return ids, nil
}

func (c *Client) FetchPrices(symbols []string, currencies []string) (map[string]map[string]float64, error) {
	ids, err := c.mapSymbolsToIds(symbols)
	if err != nil {
		return nil, err
	}

	simplePrices, err := c.FetchSimplePrice(context.Background(), maps.Values(ids), currencies)
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string]float64)
	for symbol, id := range ids {
		result[symbol] = map[string]float64{}
		for _, currency := range currencies {
			result[symbol][currency] = simplePrices[id][strings.ToLower(currency)]
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
	ids, err := c.mapSymbolsToIds(symbols)
	if err != nil {
		return nil, err
	}

	marketValues, err := c.FetchCoinsMarkets(context.Background(), maps.Values(ids), currency)
	if err != nil {
		return nil, err
	}

	result := make(map[string]thirdparty.TokenMarketValues)
	for symbol, id := range ids {
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

	return result, nil
}

func (c *Client) FetchHistoricalHourlyPrices(symbol string, currency string, limit int, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	return []thirdparty.HistoricalPrice{}, nil
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

	container, err := c.FetchHistoryMarketData(context.Background(), id, currency)
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
