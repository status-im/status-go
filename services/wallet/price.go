package wallet

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

func fetchCryptoComparePrices(symbols []string, currency string) (map[string]float64, error) {
	httpClient := http.Client{Timeout: time.Minute}

	url := fmt.Sprintf("https://min-api.cryptocompare.com/data/pricemulti?fsyms=%s&tsyms=%s", strings.Join(symbols, ","), currency)
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	prices := make(map[string]map[string]float64)
	err = json.Unmarshal(body, &prices)
	if err != nil {
		return nil, err
	}

	result := make(map[string]float64)
	for _, symbol := range symbols {
		result[symbol] = prices[strings.ToUpper(symbol)][strings.ToUpper(currency)]
	}
	return result, nil
}
