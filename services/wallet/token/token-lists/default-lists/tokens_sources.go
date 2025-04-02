package defaulttokenlists

const (
	UniswapTokenListID = "uniswap"
	AaveTokenListID    = "aave"
)

type TokensSource struct {
	Name       string
	SourceURL  string
	Schema     string
	OutputFile string
}

var TokensSources = map[string]TokensSource{
	UniswapTokenListID: {
		Name:       "Uniswap Labs Default Token List",
		SourceURL:  "https://dummy.com/ipns/tokens.uniswap.org",
		Schema:     "https://dummy.com/tokenlist.schema.json",
		OutputFile: "services/wallet/token/token-lists/default-lists/uniswap.go",
	},
	AaveTokenListID: {
		Name:       "Aave Token List",
		SourceURL:  "https://dummy.com/bgd-labs/aave-address-book/main/tokenlist.json",
		OutputFile: "services/wallet/token/token-lists/default-lists/aave.go",
	},
}
