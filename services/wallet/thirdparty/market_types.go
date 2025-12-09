package thirdparty

//go:generate go tool mockgen -package=mock_thirdparty -source=market_types.go -destination=mock/market_types.go

import tokentypes "github.com/status-im/status-go/services/wallet/token/types"

type HistoricalPrice struct {
	Timestamp int64   `json:"time"`
	Value     float64 `json:"close"`
}

type TokenMarketValues struct {
	MKTCAP          float64 `json:"MKTCAP"`
	HIGHDAY         float64 `json:"HIGHDAY"`
	LOWDAY          float64 `json:"LOWDAY"`
	CHANGEPCTHOUR   float64 `json:"CHANGEPCTHOUR"`
	CHANGEPCTDAY    float64 `json:"CHANGEPCTDAY"`
	CHANGEPCT24HOUR float64 `json:"CHANGEPCT24HOUR"`
	CHANGE24HOUR    float64 `json:"CHANGE24HOUR"`
}

type TokenDetails struct {
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

type MarketDataProvider interface {
	ID() string
	FetchPrices(tokens []*tokentypes.Token, currencies []string) (map[string]map[string]float64, error)
	FetchHistoricalDailyPrices(token *tokentypes.Token, currency string, limit int, allData bool, aggregate int) ([]HistoricalPrice, error)
	FetchHistoricalHourlyPrices(token *tokentypes.Token, currency string, limit int, aggregate int) ([]HistoricalPrice, error)
	FetchTokenMarketValues(tokens []*tokentypes.Token, currency string) (map[string]TokenMarketValues, error)
	FetchTokenDetails(tokens []*tokentypes.Token) (map[string]TokenDetails, error)
}
