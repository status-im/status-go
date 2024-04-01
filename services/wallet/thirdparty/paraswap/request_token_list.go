package paraswap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

const tokensURL = "https://apiv5.paraswap.io/tokens/%d" // nolint: gosec

type Token struct {
	Symbol   string `json:"symbol"`
	Address  string `json:"address"`
	Decimals uint   `json:"decimals"`
	Img      string `json:"img"`
	Network  int    `json:"network"`
}

type TokensResponse struct {
	Tokens []Token `json:"tokens"`
	Error  string  `json:"error"`
}

func (c *ClientV5) FetchTokensList(ctx context.Context) ([]Token, error) {
	url := fmt.Sprintf(tokensURL, c.chainID)
	response, err := c.httpClient.doGetRequest(ctx, url, nil)
	if err != nil {
		return nil, err
	}

	return handleTokensListResponse(response)
}

func handleTokensListResponse(response []byte) ([]Token, error) {
	var tokensResponse TokensResponse
	err := json.Unmarshal(response, &tokensResponse)
	if err != nil {
		return nil, err
	}

	if tokensResponse.Error != "" {
		return nil, errors.New(tokensResponse.Error)
	}

	return tokensResponse.Tokens, nil
}
