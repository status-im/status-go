package cryptocompare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/thirdparty/utils"
)

const baseID = "cryptocompare"
const extraParamStatus = "Status.im"
const baseURL = "https://min-api.cryptocompare.com"

// 300 is the max length for fsyms, but we need to subtract the length of the ETH symbol cause we want to add ETH to every chunk
// to suppress the error from the API for unknown symbols
const maxFsymsLength = 300 - len("ETH")

type HistoricalPricesContainer struct {
	Aggregated     bool                         `json:"Aggregated"`
	TimeFrom       int64                        `json:"TimeFrom"`
	TimeTo         int64                        `json:"TimeTo"`
	HistoricalData []thirdparty.HistoricalPrice `json:"Data"`
}

type HistoricalPricesData struct {
	Data HistoricalPricesContainer `json:"Data"`
}

type TokenDetailsContainer struct {
	Data map[string]thirdparty.TokenDetails `json:"Data"`
}

type MarketValuesContainer struct {
	Raw map[string]map[string]thirdparty.TokenMarketValues `json:"Raw"`
}

type Params struct {
	ID       string
	URL      string
	User     string
	Password string
}

type Client struct {
	id         string
	httpClient *thirdparty.HTTPClient
	baseURL    string
	creds      *thirdparty.BasicCreds
}

func NewClient() *Client {
	return NewClientWithParams(Params{
		ID:  baseID,
		URL: baseURL,
	})
}

func NewClientWithParams(params Params) *Client {
	var creds *thirdparty.BasicCreds
	if params.User != "" {
		creds = &thirdparty.BasicCreds{
			User:     params.User,
			Password: params.Password,
		}
	}

	// Configure HTTP client with detailed timeouts
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
	baseURL := strings.TrimSuffix(params.URL, "/")

	return &Client{
		id:         params.ID,
		httpClient: httpClient,
		baseURL:    baseURL,
		creds:      creds,
	}
}

// buildURL creates a URL by joining the base URL with the given path
// ensuring there are no double slashes
func (c *Client) buildURL(path string) string {
	baseURL := strings.TrimRight(c.baseURL, "/")
	trimmedPath := strings.TrimLeft(path, "/")

	return baseURL + "/" + trimmedPath
}

func (c *Client) FetchPrices(symbols []string, currencies []string) (map[string]map[string]float64, error) {
	chunkSymbolParams := utils.ChunkSymbolsParams{
		MaxCharsPerChunk:    maxFsymsLength,
		ExtraCharsPerSymbol: 1, // joined with a comma
	}
	chunks, err := utils.ChunkSymbols(symbols, chunkSymbolParams)
	if err != nil {
		return nil, err
	}
	result := make(map[string]map[string]float64)
	realCurrencies := utils.RenameSymbols(currencies)
	for _, smbls := range chunks {
		smbls = append(smbls, "ETH")
		realSymbols := utils.RenameSymbols(smbls)

		params := url.Values{}
		params.Add("fsyms", strings.Join(realSymbols, ","))
		params.Add("tsyms", strings.Join(realCurrencies, ","))
		params.Add("relaxedValidation", "true")
		params.Add("extraParams", extraParamStatus)

		url := c.buildURL("data/pricemulti")
		response, err := c.httpClient.DoGetRequestWithCredentials(context.Background(), url, params, c.creds)
		if err != nil {
			return nil, err
		}

		prices := make(map[string]map[string]float64)
		err = json.Unmarshal(response, &prices)
		if err != nil {
			return nil, fmt.Errorf("%s - %s", err, string(response))
		}

		for _, symbol := range smbls {
			result[symbol] = map[string]float64{}
			for _, currency := range currencies {
				result[symbol][currency] = prices[utils.GetRealSymbol(symbol)][utils.GetRealSymbol(currency)]
			}
		}
	}
	return result, nil
}

func (c *Client) FetchTokenDetails(symbols []string) (map[string]thirdparty.TokenDetails, error) {
	url := c.buildURL("data/all/coinlist")
	response, err := c.httpClient.DoGetRequestWithCredentials(context.Background(), url, nil, c.creds)
	if err != nil {
		return nil, err
	}

	container := TokenDetailsContainer{}
	err = json.Unmarshal(response, &container)
	if err != nil {
		return nil, err
	}

	tokenDetails := make(map[string]thirdparty.TokenDetails)

	for _, symbol := range symbols {
		tokenDetails[symbol] = container.Data[utils.GetRealSymbol(symbol)]
	}

	return tokenDetails, nil
}

func (c *Client) FetchTokenMarketValues(symbols []string, currency string) (map[string]thirdparty.TokenMarketValues, error) {
	chunkSymbolParams := utils.ChunkSymbolsParams{
		MaxCharsPerChunk:    maxFsymsLength,
		ExtraCharsPerSymbol: 1, // joined with a comma
	}
	chunks, err := utils.ChunkSymbols(symbols, chunkSymbolParams)
	if err != nil {
		return nil, err
	}
	realCurrency := utils.GetRealSymbol(currency)
	item := map[string]thirdparty.TokenMarketValues{}
	for _, smbls := range chunks {
		smbls = append(smbls, "ETH")
		realSymbols := utils.RenameSymbols(smbls)

		params := url.Values{}
		params.Add("fsyms", strings.Join(realSymbols, ","))
		params.Add("tsyms", realCurrency)
		params.Add("relaxedValidation", "true")
		params.Add("extraParams", extraParamStatus)

		url := c.buildURL("data/pricemultifull")
		response, err := c.httpClient.DoGetRequestWithCredentials(context.Background(), url, params, c.creds)
		if err != nil {
			return nil, err
		}

		container := MarketValuesContainer{}
		err = json.Unmarshal(response, &container)

		if len(container.Raw) == 0 {
			return nil, fmt.Errorf("no data found - %s", string(response))
		}
		if err != nil {
			return nil, fmt.Errorf("%s - %s", err, string(response))
		}

		for _, symbol := range smbls {
			item[symbol] = container.Raw[utils.GetRealSymbol(symbol)][realCurrency]
		}
	}
	return item, nil
}

func (c *Client) FetchHistoricalHourlyPrices(symbol string, currency string, limit int, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	item := []thirdparty.HistoricalPrice{}

	params := url.Values{}
	params.Add("fsym", utils.GetRealSymbol(symbol))
	params.Add("tsym", currency)
	params.Add("aggregate", fmt.Sprintf("%d", aggregate))
	params.Add("limit", fmt.Sprintf("%d", limit))
	params.Add("extraParams", extraParamStatus)

	url := c.buildURL("data/v2/histohour")
	response, err := c.httpClient.DoGetRequestWithCredentials(context.Background(), url, params, c.creds)
	if err != nil {
		return item, err
	}

	container := HistoricalPricesData{}
	err = json.Unmarshal(response, &container)
	if err != nil {
		return item, err
	}

	item = container.Data.HistoricalData

	return item, nil
}

func (c *Client) FetchHistoricalDailyPrices(symbol string, currency string, limit int, allData bool, aggregate int) ([]thirdparty.HistoricalPrice, error) {
	item := []thirdparty.HistoricalPrice{}

	params := url.Values{}
	params.Add("fsym", utils.GetRealSymbol(symbol))
	params.Add("tsym", currency)
	params.Add("aggregate", fmt.Sprintf("%d", aggregate))
	params.Add("limit", fmt.Sprintf("%d", limit))
	params.Add("allData", fmt.Sprintf("%v", allData))
	params.Add("extraParams", extraParamStatus)

	url := c.buildURL("data/v2/histoday")
	response, err := c.httpClient.DoGetRequestWithCredentials(context.Background(), url, params, c.creds)
	if err != nil {
		return item, err
	}

	container := HistoricalPricesData{}
	err = json.Unmarshal(response, &container)
	if err != nil {
		return item, err
	}

	item = container.Data.HistoricalData

	return item, nil
}

func (c *Client) ID() string {
	return c.id
}
