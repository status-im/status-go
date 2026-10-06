package defaulttokenlists

import (
	_ "embed" // for go:embed
	"time"
)

//go:embed coingecko_optimism.json
var coingeckoOptimismTokenListJSON []byte

func init() {
	CoingeckoOptimismTokenList.ID = "coingeckoOptimism"
	CoingeckoOptimismTokenList.SourceURL = "https://prod.market.status.im/v1/token_lists/optimistic-ethereum/all.json"
	CoingeckoOptimismTokenList.Fetched = time.Unix(1791450991, 0)
	CoingeckoOptimismTokenList.JsonData = coingeckoOptimismTokenListJSON
}
