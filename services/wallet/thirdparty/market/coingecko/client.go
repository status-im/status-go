package coingecko

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/thirdparty/utils"
)

const baseURL = "https://api.coingecko.com/api/v3"

type Client struct {
	httpClient *thirdparty.HTTPClient
	baseURL    string
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
		httpClient: httpClient,
		baseURL:    baseURL,
	}
}

func (c *Client) ID() string {
	return "coingecko"
}

// FetchPrices fetches the prices for the given tokens token keys (coingecko id param is used for token key) in the given currencies.
func (c *Client) FetchPrices(groupedTokensKeys []string, currencies []string) (map[string]map[string]float64, error) {
	if len(groupedTokensKeys) == 0 {
		return nil, fmt.Errorf("no tokens provided")
	}
	if len(currencies) == 0 {
		return nil, fmt.Errorf("no currencies provided")
	}

	chunkParams := utils.ChunkSymbolsParams{
		MaxSymbolsPerChunk: 400,
	}
	chunks, err := utils.ChunkSymbols(groupedTokensKeys, chunkParams)
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string]float64)
	for _, ids := range chunks {
		simplePrices, err := c.FetchSimplePrice(context.Background(), ids, currencies)
		if err != nil {
			return nil, err
		}

		for _, gtKey := range groupedTokensKeys {
			result[gtKey] = map[string]float64{}
			_, cgKey := simplePrices[gtKey]
			for _, currency := range currencies {
				if !cgKey {
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
	return nil, fmt.Errorf("not implemented") // for fetching token details we need pro-api key
}

// FetchTokenMarketValues fetches the market values for the given token keys (coingecko id param is used for token key) in the given currency.
func (c *Client) FetchTokenMarketValues(groupedTokensKeys []string, currency string) (map[string]thirdparty.TokenMarketValues, error) {
	if len(groupedTokensKeys) == 0 {
		return nil, fmt.Errorf("no tokens provided")
	}
	if currency == "" {
		return nil, fmt.Errorf("no currency provided")
	}

	chunkParams := utils.ChunkSymbolsParams{
		MaxSymbolsPerChunk: 400,
	}
	chunks, err := utils.ChunkSymbols(groupedTokensKeys, chunkParams)
	if err != nil {
		return nil, err
	}

	result := make(map[string]thirdparty.TokenMarketValues)
	for _, ids := range chunks {
		marketValues, err := c.FetchCoinsMarkets(context.Background(), ids, currency)
		if err != nil {
			return nil, err
		}

		for _, gtKey := range groupedTokensKeys {
			result[gtKey] = thirdparty.TokenMarketValues{}
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
	return nil, fmt.Errorf("not implemented")
}

// FetchHistoricalDailyPrices fetches the historical daily prices for the given token key (coingecko id param is used for token key) in the given currency.
func (c *Client) FetchHistoricalDailyPrices(groupedTokensKey string, currency string, limit int, allData bool, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	if groupedTokensKey == "" {
		return nil, fmt.Errorf("no token provided")
	}
	if currency == "" {
		return nil, fmt.Errorf("no currency provided")
	}

	container, err := c.FetchHistoryMarketData(context.Background(), groupedTokensKey, currency)
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
