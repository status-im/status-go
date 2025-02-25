package token

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
		SourceURL:  "https://ipfs.io/ipns/tokens.uniswap.org",
		Schema:     "https://uniswap.org/tokenlist.schema.json",
		OutputFile: "services/wallet/token/uniswap.go",
	},
	AaveTokenListID: {
		Name:       "Aave Token List",
		SourceURL:  "https://raw.githubusercontent.com/bgd-labs/aave-address-book/main/tokenlist.json",
		OutputFile: "services/wallet/token/aave.go",
	},
}
