package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	netUrl "net/url"
)

const platformsURL = "%s/asset_platforms"

type Chain struct {
	ID           string `json:"id"`
	ChainID      uint64 `json:"chain_identifier"`
	Name         string `json:"name"`
	NativeCoinID string `json:"native_coin_id"`
}

func (c *Client) FetchPlatforms(ctx context.Context) ([]Chain, error) {

	params := netUrl.Values{}
	url := fmt.Sprintf(platformsURL, c.baseURL)
	response, err := c.httpClient.DoGetRequest(ctx, url, params)
	if err != nil {
		return nil, err
	}

	var chains []Chain
	err = json.Unmarshal(response, &chains)
	if err != nil {
		return nil, err
	}

	if len(chains) == 0 {
		return chains, fmt.Errorf("no chains in response")
	}

	return chains, nil
}
