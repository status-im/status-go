package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	netUrl "net/url"
	"strings"
)

const coinsMarketsURL = "%s/coins/markets"

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

func (c *Client) FetchCoinsMarkets(ctx context.Context, ids []string, currency string) ([]GeckoMarketValues, error) {

	params := netUrl.Values{}
	params.Add("ids", strings.Join(ids, ","))
	params.Add("vs_currency", currency)
	params.Add("order", "market_cap_desc")
	params.Add("per_page", "250")
	params.Add("page", "1")
	params.Add("sparkline", "false")
	params.Add("price_change_percentage", "1h,24h")
	url := fmt.Sprintf(coinsMarketsURL, c.baseURL)
	response, err := c.httpClient.DoGetRequest(ctx, url, params)
	if err != nil {
		return nil, err
	}

	return handleFetchCoinsMarketsResponse(response)
}

func handleFetchCoinsMarketsResponse(response []byte) ([]GeckoMarketValues, error) {
	var marketValues []GeckoMarketValues
	err := json.Unmarshal(response, &marketValues)
	if err != nil {
		return nil, fmt.Errorf("%s - %s", err, string(response))
	}
	return marketValues, nil
}
