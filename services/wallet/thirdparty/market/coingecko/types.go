package coingecko

type GeckoToken struct {
	ID        string    `json:"id"`
	Symbol    string    `json:"symbol"`
	Name      string    `json:"name"`
	Platforms Platforms `json:"platforms"`
}

type Platforms struct { // When we add a new chain we should update it here
	Ethereum string `json:"ethereum"`
	Optimism string `json:"optimistic-ethereum"`
	Arbitrum string `json:"arbitrum-one"`
	Base     string `json:"base"`
}
