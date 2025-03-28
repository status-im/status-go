package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	netUrl "net/url"
)

const coinsListURL = "%s/coins/list"

type GeckoToken struct {
	ID        string    `json:"id"`
	Symbol    string    `json:"symbol"`
	Name      string    `json:"name"`
	Platforms Platforms `json:"platforms"`
}

type Platforms struct { // When we add a new chain we should update it here
	Ethereum string `json:"ethereum"`
	Optimism string `json:"optimistic-ethereum"`
	Arbitrum string `json:"arbitrum-one"`
	Base     string `json:"base"`
}

func (c *Client) FetchTokens(ctx context.Context) ([]GeckoToken, error) {

	params := netUrl.Values{}
	params.Add("include_platform", "true")
	url := fmt.Sprintf(coinsListURL, c.baseURL)
	response, err := c.httpClient.DoGetRequest(ctx, url, params)
	if err != nil {
		return nil, err
	}

	var tokens []GeckoToken
	err = json.Unmarshal(response, &tokens)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}
