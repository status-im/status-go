package relay

import (
	"context"
	"encoding/json"
)

type Chain struct {
	ID             uint64 `json:"id"`
	Name           string `json:"name"`
	DisplayName    string `json:"displayName"`
	DepositEnabled bool   `json:"depositEnabled"`
	Disabled       bool   `json:"disabled"`
}

type chainsResponse struct {
	Chains []Chain `json:"chains"`
}

func (c *Client) FetchChains(ctx context.Context) ([]Chain, error) {
	response, err := c.httpClient.DoGetRequest(ctx, c.baseURL+"/chains", nil, c.requestOptions()...)
	if err != nil {
		return nil, err
	}

	var resp chainsResponse
	if err := json.Unmarshal(response, &resp); err != nil {
		return nil, err
	}
	return resp.Chains, nil
}
