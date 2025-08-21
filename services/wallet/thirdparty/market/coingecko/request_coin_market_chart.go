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

func (c *Client) FetchHistoryMarketData(ctx context.Context, id string, currency string, days string) (HistoricalPriceContainer, error) {
	if days == "" {
		days = "30"
	}

	params := netUrl.Values{}
	params.Add("vs_currency", currency)
	params.Add("days", days)
	url := fmt.Sprintf(coinMarketChartURL, c.baseURL, id)
	response, err := c.doGetRequestWithOptionalAuth(ctx, url, params)
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
