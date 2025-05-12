package leaderboard

// CryptoResponse represents the API response structure
type CryptoResponse struct {
	Data []Cryptocurrency `json:"data"`
}

// Cryptocurrency represents a cryptocurrency entry
type Cryptocurrency struct {
	ID                       string  `json:"id"`
	Symbol                   string  `json:"symbol"`
	Name                     string  `json:"name"`
	Image                    string  `json:"image"`
	CurrentPrice             float64 `json:"current_price"`
	MarketCap                float64 `json:"market_cap"`
	TotalVolume              float64 `json:"total_volume"`
	PriceChangePercentage24h float64 `json:"price_change_percentage_24h"`
}

// PriceData represents price data update
type PriceData struct {
	ID               string  `json:"id,omitempty"`
	Price            float64 `json:"current_price"`
	PercentChange24h float64 `json:"price_change_percentage_24h"`
}

type LeaderboardPage struct {
	TotalCount int              `json:"all_cryptocurrency_count"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	SortOrder  int              `json:"sorting"`
	Currency   string           `json:"currency"`
	Data       []Cryptocurrency `json:"cryptocurrencies"`
}

func (p *LeaderboardPage) Valid() bool {
	return p.Page > 0 && p.PageSize > 0
}

type LeaderboardPagePrices struct {
	Page      int         `json:"page"`
	PageSize  int         `json:"page_size"`
	SortOrder int         `json:"sorting"`
	Currency  string      `json:"currency"`
	Data      []PriceData `json:"prices"`
}

// PriceMap is a map of cryptocurrency symbols to their price data
type PriceMap map[string]PriceData
