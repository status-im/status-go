package wallet

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const cryptocompareURL = "https://min-api.cryptocompare.com"

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

type TokenHistoricalPairs struct {
	Timestamp  int64   `json:"time"`
	Value      float64 `json:"close"`
	Volumefrom float64 `json:"volumefrom"`
	Volumeto   float64 `json:"volumeto"`
}

type HistoricalValuesContainer struct {
	Aggregated     bool                   `json:"Aggregated"`
	TimeFrom       int64                  `json:"TimeFrom"`
	TimeTo         int64                  `json:"TimeTo"`
	HistoricalData []TokenHistoricalPairs `json:"Data"`
}

type HistoricalValuesData struct {
	Data HistoricalValuesContainer `json:"Data"`
}

type CoinsContainer struct {
	Data map[string]Coin `json:"Data"`
}

type MarketValuesContainer struct {
	Display map[string]map[string]MarketCoinValues `json:"Display"`
}

func fetchCryptoComparePrices(symbols []string, currency string) (map[string]float64, error) {
	httpClient := http.Client{Timeout: time.Minute}

	url := fmt.Sprintf("%s/data/pricemulti?fsyms=%s&tsyms=%s&extraParams=Status.im", cryptocompareURL, strings.Join(symbols, ","), currency)
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

	url := fmt.Sprintf("%s/data/all/coinlist", cryptocompareURL)
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

	url := fmt.Sprintf("%s/data/pricemultifull?fsyms=%s&tsyms=%s&extraParams=Status.im", cryptocompareURL, strings.Join(symbols, ","), currency)
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

func fetchHourlyMarketValues(symbol string, currency string, limit int, aggregate int) ([]TokenHistoricalPairs, error) {
	item := []TokenHistoricalPairs{}
	httpClient := http.Client{Timeout: time.Minute}

	url := fmt.Sprintf("%s/data/v2/histohour?fsym=%s&tsym=%s&aggregate=%d&limit=%d&extraParams=Status.im", cryptocompareURL, symbol, currency, aggregate, limit)
	resp, err := httpClient.Get(url)
	if err != nil {
		return item, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return item, err
	}

	container := HistoricalValuesData{}
	err = json.Unmarshal(body, &container)
	if err != nil {
		return item, err
	}

	item = container.Data.HistoricalData

	return item, nil
}

func fetchDailyMarketValues(symbol string, currency string, limit int, allData bool, aggregate int) ([]TokenHistoricalPairs, error) {
	item := []TokenHistoricalPairs{}
	httpClient := http.Client{Timeout: time.Minute}

	url := fmt.Sprintf("%s/data/v2/histoday?fsym=%s&tsym=%s&aggregate=%d&limit=%d&allData=%v&extraParams=Status.im", cryptocompareURL, symbol, currency, aggregate, limit, allData)
	resp, err := httpClient.Get(url)
	if err != nil {
		return item, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return item, err
	}

	container := HistoricalValuesData{}
	err = json.Unmarshal(body, &container)
	if err != nil {
		return item, err
	}

	item = container.Data.HistoricalData

	return item, nil
}
