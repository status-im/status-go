package cryptocompare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/thirdparty/utils"
)

const baseURL = "https://min-api.cryptocompare.com"
const CryptoCompareStatusProxyURL = "https://cryptocompare.test.api.status.im"
const extraParamStatus = "Status.im"

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

type Client struct {
	httpClient *thirdparty.HTTPClient
	baseURL    string
}

func NewClient() *Client {
	return &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    baseURL,
	}
}

func NewClientWithURL(url string) *Client {
	return &Client{
		httpClient: thirdparty.NewHTTPClient(),
		baseURL:    url,
	}
}

func (c *Client) FetchPrices(symbols []string, currencies []string) (map[string]map[string]float64, error) {
	chunks := utils.ChunkSymbols(symbols, 60)
	result := make(map[string]map[string]float64)
	realCurrencies := utils.RenameSymbols(currencies)
	for _, smbls := range chunks {
		realSymbols := utils.RenameSymbols(smbls)

		params := url.Values{}
		params.Add("fsyms", strings.Join(realSymbols, ","))
		params.Add("tsyms", strings.Join(realCurrencies, ","))
		params.Add("extraParams", extraParamStatus)

		url := fmt.Sprintf("%s/data/pricemulti", c.baseURL)
		response, err := c.httpClient.DoGetRequest(context.Background(), url, params)
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
	url := fmt.Sprintf("%s/data/all/coinlist", c.baseURL)
	response, err := c.httpClient.DoGetRequest(context.Background(), url, nil)
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
	chunks := utils.ChunkSymbols(symbols)
	realCurrency := utils.GetRealSymbol(currency)
	item := map[string]thirdparty.TokenMarketValues{}
	for _, smbls := range chunks {
		realSymbols := utils.RenameSymbols(smbls)

		params := url.Values{}
		params.Add("fsyms", strings.Join(realSymbols, ","))
		params.Add("tsyms", realCurrency)
		params.Add("extraParams", extraParamStatus)

		url := fmt.Sprintf("%s/data/pricemultifull", c.baseURL)
		response, err := c.httpClient.DoGetRequest(context.Background(), url, params)
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
			item[symbol] = container.Raw[utils.GetRealSymbol(symbol)][utils.GetRealSymbol(currency)]
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

	url := fmt.Sprintf("%s/data/v2/histohour", c.baseURL)
	response, err := c.httpClient.DoGetRequest(context.Background(), url, params)
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

	url := fmt.Sprintf("%s/data/v2/histoday", c.baseURL)
	response, err := c.httpClient.DoGetRequest(context.Background(), url, params)
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
	return "cryptocompare"
}
