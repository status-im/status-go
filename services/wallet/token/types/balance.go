package tokentypes

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type TokenMarketValues struct {
	MarketCap       float64 `json:"marketCap"`
	HighDay         float64 `json:"highDay"`
	LowDay          float64 `json:"lowDay"`
	ChangePctHour   float64 `json:"changePctHour"`
	ChangePctDay    float64 `json:"changePctDay"`
	ChangePct24hour float64 `json:"changePct24hour"`
	Change24hour    float64 `json:"change24hour"`
	Price           float64 `json:"price"`
	HasError        bool    `json:"hasError"`
}

type ChainBalance struct {
	RawBalance     string         `json:"rawBalance"`
	Balance        *big.Float     `json:"balance"`
	Balance1DayAgo string         `json:"balance1DayAgo"`
	Address        common.Address `json:"address"`
	ChainID        uint64         `json:"chainId"`
	HasError       bool           `json:"hasError"`
}
