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

var renameMapping = map[string]string{
	"STT": "SNT",
}

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
	MKTCAP          float64 `json:"MKTCAP"`
	HIGHDAY         float64 `json:"HIGHDAY"`
	LOWDAY          float64 `json:"LOWDAY"`
	CHANGEPCTHOUR   float64 `json:"CHANGEPCTHOUR"`
	CHANGEPCTDAY    float64 `json:"CHANGEPCTDAY"`
	CHANGEPCT24HOUR float64 `json:"CHANGEPCT24HOUR"`
	CHANGE24HOUR    float64 `json:"CHANGE24HOUR"`
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
	Raw map[string]map[string]MarketCoinValues `json:"Raw"`
}

func renameSymbols(symbols []string) (renames []string) {
	for _, symbol := range symbols {
		renames = append(renames, getRealSymbol(symbol))
	}
	return
}

func getRealSymbol(symbol string) string {
	if val, ok := renameMapping[strings.ToUpper(symbol)]; ok {
		return val
	}
	return strings.ToUpper(symbol)
}

func chunkSymbols(symbols []string) [][]string {
	var chunks [][]string
	chunkSize := 20
	for i := 0; i < len(symbols); i += chunkSize {
		end := i + chunkSize

		if end > len(symbols) {
			end = len(symbols)
		}

		chunks = append(chunks, symbols[i:end])
	}

	return chunks
}

func fetchCryptoComparePrices(symbols []string, currency string) (map[string]float64, error) {
	chunks := chunkSymbols(symbols)
	result := make(map[string]float64)
	for _, smbls := range chunks {
		realSymbols := renameSymbols(smbls)
		httpClient := http.Client{Timeout: time.Minute}
		url := fmt.Sprintf("%s/data/pricemulti?fsyms=%s&tsyms=%s&extraParams=Status.im", cryptocompareURL, strings.Join(realSymbols, ","), currency)
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

		for _, symbol := range smbls {
			result[symbol] = prices[getRealSymbol(symbol)][strings.ToUpper(currency)]
		}
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
		coins[symbol] = container.Data[getRealSymbol(symbol)]
	}

	return coins, nil
}

func fetchTokenMarketValues(symbols []string, currency string) (map[string]MarketCoinValues, error) {
	realSymbols := renameSymbols(symbols)
	item := map[string]MarketCoinValues{}
	httpClient := http.Client{Timeout: time.Minute}

	url := fmt.Sprintf("%s/data/pricemultifull?fsyms=%s&tsyms=%s&extraParams=Status.im", cryptocompareURL, strings.Join(realSymbols, ","), currency)
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

	for _, symbol := range symbols {
		item[symbol] = container.Raw[getRealSymbol(symbol)][strings.ToUpper(currency)]
	}

	return item, nil
}

func fetchHourlyMarketValues(symbol string, currency string, limit int, aggregate int) ([]TokenHistoricalPairs, error) {
	item := []TokenHistoricalPairs{}
	httpClient := http.Client{Timeout: time.Minute}

	url := fmt.Sprintf("%s/data/v2/histohour?fsym=%s&tsym=%s&aggregate=%d&limit=%d&extraParams=Status.im", cryptocompareURL, getRealSymbol(symbol), currency, aggregate, limit)
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

	url := fmt.Sprintf("%s/data/v2/histoday?fsym=%s&tsym=%s&aggregate=%d&limit=%d&allData=%v&extraParams=Status.im", cryptocompareURL, getRealSymbol(symbol), currency, aggregate, limit, allData)
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
