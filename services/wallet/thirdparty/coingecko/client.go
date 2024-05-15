package coingecko

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/thirdparty/utils"
)

var coinGeckoMapping = map[string]string{
	"STT":   "status",
	"SNT":   "status",
	"ETH":   "ethereum",
	"AST":   "airswap",
	"ABT":   "arcblock",
	"ATM":   "",
	"BNB":   "binancecoin",
	"BLT":   "bloom",
	"CDT":   "",
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
	"VRS":   "",
	"USDT":  "tether",
	"SHIB":  "shiba-inu",
	"LINK":  "chainlink",
	"MATIC": "matic-network",
	"DAI":   "dai",
	"ARB":   "arbitrum",
	"OP":    "optimism",
}

const baseURL = "https://api.coingecko.com/api/v3/"

type HistoricalPriceContainer struct {
	Prices [][]float64 `json:"prices"`
}
type GeckoMarketValues struct {
	ID                                string  `json:"id"`
	Symbol                            string  `json:"symbol"`
	Name                              string  `json:"name"`
	MarketCap                         float64 `json:"market_cap"`
	High24h                           float64 `json:"high_24h"`
	Low24h                            float64 `json:"low_24h"`
	PriceChange24h                    float64 `json:"price_change_24h"`
	PriceChangePercentage24h          float64 `json:"price_change_percentage_24h"`
	PriceChangePercentage1hInCurrency float64 `json:"price_change_percentage_1h_in_currency"`
}

type GeckoToken struct {
	ID          string `json:"id"`
	Symbol      string `json:"symbol"`
	Name        string `json:"name"`
	EthPlatform bool
}

type Client struct {
	client           *http.Client
	tokens           map[string][]GeckoToken
	tokensURL        string
	fetchTokensMutex sync.Mutex
}

func NewClient() *Client {
	return &Client{client: &http.Client{Timeout: time.Minute}, tokens: make(map[string][]GeckoToken), tokensURL: fmt.Sprintf("%scoins/list?include_platform=true", baseURL)}
}

func (gt *GeckoToken) UnmarshalJSON(data []byte) error {
	// Define an auxiliary struct to hold the JSON data
	var aux struct {
		ID        string `json:"id"`
		Symbol    string `json:"symbol"`
		Name      string `json:"name"`
		Platforms struct {
			Ethereum string `json:"ethereum"`
			// Other platforms can be added here if needed
		} `json:"platforms"`
	}

	// Unmarshal the JSON data into the auxiliary struct
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Set the fields of GeckoToken from the auxiliary struct
	gt.ID = aux.ID
	gt.Symbol = aux.Symbol
	gt.Name = aux.Name

	// Check if "ethereum" key exists in the platforms map
	if aux.Platforms.Ethereum != "" {
		gt.EthPlatform = true
	} else {
		gt.EthPlatform = false
	}
	return nil
}

func (c *Client) DoQuery(url string) (*http.Response, error) {
	resp, err := c.client.Get(url)

	if err != nil {
		return nil, err
	}
	return resp, nil
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

	resp, err := c.DoQuery(c.tokensURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tokens []GeckoToken
	err = json.Unmarshal(body, &tokens)
	if err != nil {
		return nil, err
	}

	mapTokensToSymbols(tokens, c.tokens)

	return c.tokens, nil
}

func (c *Client) mapSymbolsToIds(symbols []string) ([]string, error) {
	tokens, err := c.getTokens()
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0)
	for _, symbol := range utils.RenameSymbols(symbols) {
		id, err := getIDFromSymbol(tokens, symbol)
		if err == nil {
			ids = append(ids, id)
		}
	}
	ids = utils.RemoveDuplicates(ids)

	return ids, nil
}

func (c *Client) FetchPrices(symbols []string, currencies []string) (map[string]map[string]float64, error) {
	ids, err := c.mapSymbolsToIds(symbols)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%ssimple/price?ids=%s&vs_currencies=%s", baseURL, strings.Join(ids, ","), strings.Join(currencies, ","))
	resp, err := c.DoQuery(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	prices := make(map[string]map[string]float64)
	err = json.Unmarshal(body, &prices)
	if err != nil {
		return nil, fmt.Errorf("%s - %s", err, string(body))
	}

	tokens, err := c.getTokens()
	if err != nil {
		return nil, err
	}
	result := make(map[string]map[string]float64)
	for _, symbol := range symbols {
		result[symbol] = map[string]float64{}
		id, err := getIDFromSymbol(tokens, utils.GetRealSymbol(symbol))
		if err != nil {
			return nil, err
		}
		for _, currency := range currencies {
			result[symbol][currency] = prices[id][strings.ToLower(currency)]
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
	url := fmt.Sprintf("%scoins/markets?ids=%s&vs_currency=%s&order=market_cap_desc&per_page=250&page=1&sparkline=false&price_change_percentage=%s", baseURL, strings.Join(ids, ","), currency, "1h%2C24h")

	resp, err := c.DoQuery(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var marketValues []GeckoMarketValues
	err = json.Unmarshal(body, &marketValues)
	if err != nil {
		return nil, fmt.Errorf("%s - %s", err, string(body))
	}

	tokens, err := c.getTokens()
	if err != nil {
		return nil, err
	}

	result := make(map[string]thirdparty.TokenMarketValues)
	for _, symbol := range symbols {
		id, err := getIDFromSymbol(tokens, utils.GetRealSymbol(symbol))
		if err != nil {
			return nil, err
		}
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

	url := fmt.Sprintf("%scoins/%s/market_chart?vs_currency=%s&days=30", baseURL, id, currency)
	resp, err := c.DoQuery(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var container HistoricalPriceContainer
	err = json.Unmarshal(body, &container)
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
