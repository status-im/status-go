package coingecko

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"golang.org/x/exp/maps"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/builder"
	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/status-im/status-go/pkg/security"
	walletcommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/thirdparty/utils"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"
)

const baseURL = "https://api.coingecko.com/api/v3"

const (
	requestDelay     = 500 * time.Millisecond
	pricesChunkLimit = 500
	tokensChunkLimit = 250
)

type Client struct {
	httpClient *thirdparty.HTTPClient
	baseURL    string
	creds      *thirdparty.BasicCreds
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
		baseURL:    clientBaseURL,
		creds:      creds,
	}
}

func (c *Client) ID() string {
	return "coingecko"
}

// getCoingeckoTokensByTokenKey returns a map of token keys to coingecko tokens
func (c *Client) getCoingeckoTokensByTokenKey() (map[string]GeckoToken, error) {
	tokens, err := c.fetchTokens(context.Background())
	if err != nil {
		return nil, err
	}

	coingeckoTokensByTokenKey := make(map[string]GeckoToken)

	// for native tokens, platform is empty (zero address), but the id is ethereum, that's why we need to add it manually
	for _, chainID := range walletcommon.AllChainIDsAsUint64() {
		token := tokentypes.Token{Token: &types.Token{ChainID: chainID}}

		// native token for BSC chain doesn't have the coingecko ID, so skip it
		if chainID == walletcommon.BSCMainnet || chainID == walletcommon.BSCTestnet {
			continue
		}

		coingeckoTokensByTokenKey[token.Key()] = GeckoToken{
			ID:     nativeEthTokenID,
			Name:   builder.EthereumNativeName,
			Symbol: builder.EthereumNativeSymbol,
		}
	}

	for _, token := range tokens {
		for _, key := range token.keys() {
			coingeckoTokensByTokenKey[key] = token
		}
	}

	return coingeckoTokensByTokenKey, nil
}

// mapTokensToIds maps wallet sdk tokens to coingecko tokens
// returns a map of coingecko ids to token keys as first and a list of unmapped token keys as second return value
func (c *Client) mapTokensToIds(tokens []*tokentypes.Token) (map[string][]string, []string, error) {
	coingeckoTokensByTokenKey, err := c.getCoingeckoTokensByTokenKey()
	if err != nil {
		return nil, nil, err
	}

	mappedTokens := make(map[string][]string) // map[coingeckoId][]tokenKey
	unmappedTokens := make([]string, 0)

	// Coingecko doesn't provide prices for test tokens, so we need to map them to the mainnet tokens.
	// We can do that only for test tokens that have a cross chain id.
	tokenKeysByCrossChainID := make(map[string][]string)
	for _, token := range tokens {
		// Skip tokens that don't have a cross chain id
		if token.CrossChainID == "" {
			continue
		}
		// Skip tokens that are on test networks
		if !walletcommon.ChainID(token.ChainID).IsMainnet() {
			continue
		}
		tokenKeysByCrossChainID[token.CrossChainID] = append(tokenKeysByCrossChainID[token.CrossChainID], token.Key())
	}

	for _, token := range tokens {
		coingeckoToken, ok := coingeckoTokensByTokenKey[token.Key()]
		if !ok {
			if walletcommon.ChainID(token.ChainID).IsMainnet() {
				unmappedTokens = append(unmappedTokens, token.Key())
				continue
			}

			// Check by cross chain id if any of the test tokens have a coingecko token
			crossChainID := token.CrossChainID
			// Sepecial handling for status test token STT, cause even it's the same token belongs to different group and has different symbol SNT/STT.
			if crossChainID == walletcommon.StatusTestTokenCrossChainID {
				crossChainID = walletcommon.StatusMainnetTokenCrossChainID
			}

			for _, tokenKey := range tokenKeysByCrossChainID[crossChainID] {
				coingeckoToken, ok = coingeckoTokensByTokenKey[tokenKey]
				if ok {
					break
				}
			}

			if !ok {
				unmappedTokens = append(unmappedTokens, token.Key())
				continue
			}
		}
		mappedTokens[coingeckoToken.ID] = append(mappedTokens[coingeckoToken.ID], token.Key())
	}

	return mappedTokens, unmappedTokens, nil
}

// FetchPrices fetches the prices for the given tokens and currencies
// returns a map[tokenKey]map[currency]price
func (c *Client) FetchPrices(tokens []*tokentypes.Token, currencies []string) (map[string]map[string]float64, error) {
	mappedTokens, unmappedTokens, err := c.mapTokensToIds(tokens)
	if err != nil {
		return nil, err
	}

	simplePrices, err := utils.ChunkMapFetcher[CurrencyPriceMap](
		context.Background(),
		maps.Keys(mappedTokens),
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

	for coingeckoId, prices := range simplePrices {
		for _, tokenKey := range mappedTokens[coingeckoId] {
			result[tokenKey] = map[string]float64{}
			for _, currency := range currencies {
				result[tokenKey][currency] = prices[strings.ToLower(currency)]
			}
		}
	}

	for _, tokenKey := range unmappedTokens {
		result[tokenKey] = map[string]float64{}
		for _, currency := range currencies {
			result[tokenKey][currency] = 0
		}
	}

	return result, nil
}

// FetchTokenDetails fetches the token details for the given tokens
// returns a map[tokenKey]TokenDetails
func (c *Client) FetchTokenDetails(tokens []*tokentypes.Token) (map[string]thirdparty.TokenDetails, error) {
	// Since coingecko doesn't provide additional information about the token we don't need to fetch anything, just map tokens to TokenDetails
	result := make(map[string]thirdparty.TokenDetails)
	for _, token := range tokens {
		result[token.Key()] = thirdparty.TokenDetails{
			Name:   token.Name,
			Symbol: token.Symbol,
		}
	}
	return result, nil
}

// FetchTokenMarketValues fetches the market values for the given tokens and currency
// returns a map[tokenKey]TokenMarketValues
func (c *Client) FetchTokenMarketValues(tokens []*tokentypes.Token, currency string) (map[string]thirdparty.TokenMarketValues, error) {
	mappedTokens, unmappedTokens, err := c.mapTokensToIds(tokens)
	if err != nil {
		return nil, err
	}

	marketValues, err := utils.ChunkArrayFetcher[GeckoMarketValues](
		context.Background(),
		maps.Keys(mappedTokens),
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
	for _, marketValue := range marketValues {
		for _, tokenKey := range mappedTokens[marketValue.ID] {
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

	for _, tokenKey := range unmappedTokens {
		result[tokenKey] = thirdparty.TokenMarketValues{}
	}

	return result, nil
}

// FetchHistoricalHourlyPrices fetches the hourly prices for the given token and currency
// returns a list of HistoricalPrice
func (c *Client) FetchHistoricalHourlyPrices(token *tokentypes.Token, currency string, limit int, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	days := (limit + 23) / 24 // round up
	if days == 0 {
		days = 1 // minimum 1 day
	}
	if days > 90 {
		days = 90 // maximum 90 days
	}

	return c.FetchHistoricalDailyPrices(token, currency, days, false, aggregate)
}

// FetchHistoricalDailyPrices fetches the daily prices for the given token and currency
// returns a list of HistoricalPrice
func (c *Client) FetchHistoricalDailyPrices(token *tokentypes.Token, currency string, limit int, allData bool, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	coingeckoTokensByTokenKey, err := c.getCoingeckoTokensByTokenKey()
	if err != nil {
		return nil, err
	}

	coingeckoToken, ok := coingeckoTokensByTokenKey[token.Key()]
	if !ok {
		return nil, fmt.Errorf("coingecko id not found for token %s", token.Key())
	}

	var days string
	if allData {
		days = "max"
	} else {
		days = fmt.Sprintf("%d", limit)
	}

	container, err := c.FetchHistoryMarketData(context.Background(), coingeckoToken.ID, currency, days)
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
