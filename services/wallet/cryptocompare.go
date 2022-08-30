package wallet

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type Coin struct {
	ID                   string  `json:"Id"`
	Name                 string  `json:"Name"`
	Symbol               string  `json:"Symbol"`
	Description          string  `json:"Description"`
	TotalCoinsMined      float64 `json:"TotalCoinsMined"`
	AssetLaunchDate      string  `json:"AssetLaunchDate"`
	AssetWhitepaperURL   string  `json:"AssetWhitepaperUrl"`
	AssetWebsiteURL      string  `json:"AssetWebsiteUrl"`
	BuiltOn              string  `json:"BuiltOn"`
	SmartContractAddress string  `json:"SmartContractAddress"`
}

type MarketCoinValues struct {
	MKTCAP          string `json:"MKTCAP"`
	HIGHDAY         string `json:"HIGHDAY"`
	LOWDAY          string `json:"LOWDAY"`
	CHANGEPCTHOUR   string `json:"CHANGEPCTHOUR"`
	CHANGEPCTDAY    string `json:"CHANGEPCTDAY"`
	CHANGEPCT24HOUR string `json:"CHANGEPCT24HOUR"`
	CHANGE24HOUR    string `json:"CHANGE24HOUR"`
}

type CoinsContainer struct {
	Data map[string]Coin `json:"Data"`
}

type MarketValuesContainer struct {
	Display map[string]map[string]MarketCoinValues `json:"Display"`
}

func fetchCryptoComparePrices(symbols []string, currency string) (map[string]float64, error) {
	httpClient := http.Client{Timeout: time.Minute}

	url := fmt.Sprintf("https://min-api.cryptocompare.com/data/pricemulti?fsyms=%s&tsyms=%s&extraParams=Status.im", strings.Join(symbols, ","), currency)
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

func fetchCryptoCompareTokenDetails(symbols []string) (map[string]Coin, error) {
	httpClient := http.Client{Timeout: time.Minute}

	url := "https://min-api.cryptocompare.com/data/all/coinlist"
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	container := CoinsContainer{}
	err = json.Unmarshal(body, &container)
	if err != nil {
		return nil, err
	}

	coins := make(map[string]Coin)

	for _, symbol := range symbols {
		coins[symbol] = container.Data[symbol]
	}

	return coins, nil
}

func fetchTokenMarketValues(symbols []string, currency string) (map[string]MarketCoinValues, error) {
	item := map[string]MarketCoinValues{}
	httpClient := http.Client{Timeout: time.Minute}

	url := fmt.Sprintf("https://min-api.cryptocompare.com/data/pricemultifull?fsyms=%s&tsyms=%s&extraParams=Status.im", strings.Join(symbols, ","), currency)
	resp, err := httpClient.Get(url)
	if err != nil {
		return item, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return item, err
	}

	container := MarketValuesContainer{}
	err = json.Unmarshal(body, &container)
	if err != nil {
		return item, err
	}

	for key, element := range container.Display {
		item[key] = element[strings.ToUpper(currency)]
	}

	return item, nil

}
