package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	netUrl "net/url"
)

const coinMarketChartURL = "%s/coins/%s/market_chart"

type HistoricalPriceContainer struct {
	Prices [][]float64 `json:"prices"`
}

func (c *Client) FetchHistoryMarketData(ctx context.Context, id string, currency string) (HistoricalPriceContainer, error) {

	params := netUrl.Values{}
	params.Add("vs_currency", currency)
	params.Add("days", "30")
	url := fmt.Sprintf(coinMarketChartURL, c.baseURL, id)
	response, err := c.httpClient.DoGetRequest(ctx, url, params)
	if err != nil {
		return HistoricalPriceContainer{}, err
	}

	var container HistoricalPriceContainer
	err = json.Unmarshal(response, &container)
	if err != nil {
		return container, err
	}

	if len(container.Prices) == 0 {
		return container, fmt.Errorf("no prices in response")
	}

	return container, nil
}
