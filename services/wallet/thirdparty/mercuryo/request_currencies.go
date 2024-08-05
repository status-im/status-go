package mercuryo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	currenciesURL = "https://api.mercuryo.io/v1.6/lib/currencies" // nolint: gosec
)

type Token struct {
	Symbol   string `json:"symbol"`
	Address  string `json:"address"`
	Decimals uint   `json:"decimals"`
	Img      string `json:"img"`
	Network  int    `json:"network"`
}

type CurrenciesResponse struct {
	Data   CurrenciesData `json:"data"`
	Status int            `json:"status"`
}

type CurrenciesData struct {
	Config Config `json:"config"`
}

type Config struct {
	CryptoCurrencies []CryptoCurrency `json:"crypto_currencies"`
}

type CryptoCurrency struct {
	Symbol   string `json:"currency"`
	Network  string `json:"network"`
	Contract string `json:"contract"`
}

func (c *Client) FetchCurrencies(ctx context.Context) ([]CryptoCurrency, error) {
	response, err := c.httpClient.DoGetRequest(ctx, currenciesURL, nil, nil)
	if err != nil {
		return nil, err
	}

	return handleCurrenciesResponse(response)
}

func handleCurrenciesResponse(response []byte) ([]CryptoCurrency, error) {
	var currenciesResponse CurrenciesResponse
	err := json.Unmarshal(response, &currenciesResponse)
	if err != nil {
		return nil, err
	}

	if currenciesResponse.Status != http.StatusOK {
		return nil, fmt.Errorf("unsuccessful request: %d %s", currenciesResponse.Status, http.StatusText(currenciesResponse.Status))
	}

	assets := currenciesResponse.Data.Config.CryptoCurrencies

	return assets, nil
}
