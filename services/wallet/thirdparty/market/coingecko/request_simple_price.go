package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	netUrl "net/url"
	"strings"
)

const simplePriceURL = "%s/simple/price"

// SymbolPriceMap represents a map of symbols with price per currency for each.
type SymbolPriceMap map[string]map[string]float64

func (c *Client) FetchSimplePrice(ctx context.Context, ids []string, currencies []string) (SymbolPriceMap, error) {

	params := netUrl.Values{}
	params.Add("ids", strings.Join(ids, ","))
	params.Add("vs_currencies", strings.Join(currencies, ","))
	url := fmt.Sprintf(simplePriceURL, c.baseURL)
	response, err := c.httpClient.DoGetRequest(ctx, url, params)
	if err != nil {
		return nil, err
	}

	return handleFetchSimplePriceResponse(response)
}

func handleFetchSimplePriceResponse(response []byte) (SymbolPriceMap, error) {
	prices := make(SymbolPriceMap)
	err := json.Unmarshal(response, &prices)
	if err != nil {
		return nil, fmt.Errorf("%s - %s", err, string(response))
	}
	return prices, nil
}
