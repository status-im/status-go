package lifi

import (
	"context"
	"encoding/json"
	netUrl "net/url"
	"strconv"
)

type Token struct {
	Address  string `json:"address"`
	Symbol   string `json:"symbol"`
	Decimals uint   `json:"decimals"`
	ChainID  uint64 `json:"chainId"`
	Name     string `json:"name"`
	LogoURI  string `json:"logoURI"`
}

type tokensResponse struct {
	Tokens map[string][]Token `json:"tokens"`
}

func (c *Client) FetchTokensList(ctx context.Context) ([]Token, error) {
	params := netUrl.Values{}
	params.Add("chains", strconv.FormatUint(c.chainID, 10))

	response, err := c.httpClient.DoGetRequest(ctx, baseURL+"/tokens", params)
	if err != nil {
		return nil, err
	}

	var parsed tokensResponse
	if err := json.Unmarshal(response, &parsed); err != nil {
		return nil, err
	}

	return parsed.Tokens[strconv.FormatUint(c.chainID, 10)], nil
}
