package tokentypes

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
