//go:build gowaku_no_rln
// +build gowaku_no_rln

package leaderboard

// CryptoResponse represents the API response structure
type CryptoResponse struct {
	Data []Cryptocurrency `json:"data"`
}

// Cryptocurrency represents a cryptocurrency entry
type Cryptocurrency struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
	Quote  Quote  `json:"quote"`
}

// Quote contains price information for different currencies
type Quote struct {
	USD QuoteDetails `json:"USD"`
}

// QuoteDetails contains detailed price information
type QuoteDetails struct {
	Price            float64 `json:"price"`
	Volume24h        float64 `json:"volume_24h"`
	MarketCap        float64 `json:"market_cap"`
	PercentChange24h float64 `json:"percent_change_24h"`
}

// PriceData represents price data for a specific cryptocurrency
type PriceData struct {
	Price            float64 `json:"price"`
	Volume24h        float64 `json:"volume_24h"`
	PercentChange24h float64 `json:"percent_change_24h"`
}

// PriceMap is a map of cryptocurrency symbols to their price data
type PriceMap map[string]PriceData
