package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	netUrl "net/url"
)

const coinsListURL = "%s/coins/list"

type GeckoToken struct {
	ID          string `json:"id"`
	Symbol      string `json:"symbol"`
	Name        string `json:"name"`
	EthPlatform bool
}

func (gt *GeckoToken) UnmarshalJSON(data []byte) error {
	// Define an auxiliary struct to hold the JSON data
	var aux struct {
		ID        string `json:"id"`
		Symbol    string `json:"symbol"`
		Name      string `json:"name"`
		Platforms struct {
			Ethereum string `json:"ethereum"`
			// Other platforms can be added here if needed
		} `json:"platforms"`
	}

	// Unmarshal the JSON data into the auxiliary struct
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Set the fields of GeckoToken from the auxiliary struct
	gt.ID = aux.ID
	gt.Symbol = aux.Symbol
	gt.Name = aux.Name

	// Check if "ethereum" key exists in the platforms map
	if aux.Platforms.Ethereum != "" {
		gt.EthPlatform = true
	} else {
		gt.EthPlatform = false
	}
	return nil
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
