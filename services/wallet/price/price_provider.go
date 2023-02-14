package price

type Provider interface {
	FetchPrices(symbols []string, currencies []string) (map[string]map[string]float64, error)
}
