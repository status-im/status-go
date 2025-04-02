package defaulttokenlists

const (
	Coingecko = "coingecko"
)

type TokensSource struct {
	Name       string
	SourceURL  string
	Schema     string
	OutputFile string
}

// When updating token sources, make sure that it follows `defaultListOfTokenLists` form the fetcher package, constatnts.go file.
var TokensSources = map[string]TokensSource{
	Coingecko: {
		Name:       "CoinGecko",
		SourceURL:  "https://api.coingecko.com/api/v3/coins/list?include_platform=true",
		OutputFile: "services/wallet/token/token-lists/default-lists/coingecko.go",
	},
}
