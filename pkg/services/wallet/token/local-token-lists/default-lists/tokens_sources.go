package defaulttokenlists

import "github.com/status-im/status-go/pkg/services/wallet/common"

type TokensSource struct {
	Name       string
	SourceURL  string
	Schema     string
	OutputFile string
}

var TokensSources = map[string]TokensSource{
	common.StatusTokenListID: {
		Name:       "Status Token List",
		SourceURL:  "https://prod.market.status.im/static/token-list.json",
		OutputFile: "services/wallet/token/local-token-lists/default-lists/status.go",
	},
	common.UniswapTokenListID: {
		Name:       "Uniswap Labs Default Token List",
		SourceURL:  "https://ipfs.io/ipns/tokens.uniswap.org",
		Schema:     "https://uniswap.org/tokenlist.schema.json",
		OutputFile: "services/wallet/token/local-token-lists/default-lists/uniswap.go",
	},
	common.CoingeckoEthereumTokenListID: {
		Name:       "Coingecko",
		SourceURL:  "https://prod.market.status.im/v1/token_lists/ethereum/all.json",
		OutputFile: "services/wallet/token/local-token-lists/default-lists/coingecko_ethereum.go",
	},
	common.CoingeckoOptimismTokenListID: {
		Name:       "Coingecko",
		SourceURL:  "https://prod.market.status.im/v1/token_lists/optimistic-ethereum/all.json",
		OutputFile: "services/wallet/token/local-token-lists/default-lists/coingecko_optimism.go",
	},
	common.CoingeckoArbitrumTokenListID: {
		Name:       "Coingecko",
		SourceURL:  "https://prod.market.status.im/v1/token_lists/arbitrum-one/all.json",
		OutputFile: "services/wallet/token/local-token-lists/default-lists/coingecko_arbitrum.go",
	},
	common.CoingeckoBSCTokenListID: {
		Name:       "Coingecko",
		SourceURL:  "https://prod.market.status.im/v1/token_lists/binance-smart-chain/all.json",
		OutputFile: "services/wallet/token/local-token-lists/default-lists/coingecko_bsc.go",
	},
	common.CoingeckoBaseTokenListID: {
		Name:       "Coingecko",
		SourceURL:  "https://prod.market.status.im/v1/token_lists/base/all.json",
		OutputFile: "services/wallet/token/local-token-lists/default-lists/coingecko_base.go",
	},
	common.CoingeckoLineaTokenListID: {
		Name:       "Coingecko",
		SourceURL:  "https://prod.market.status.im/v1/token_lists/linea/all.json",
		OutputFile: "services/wallet/token/local-token-lists/default-lists/coingecko_linea.go",
	},
}
