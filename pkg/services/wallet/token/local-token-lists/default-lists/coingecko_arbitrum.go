package defaulttokenlists

import (
	_ "embed" // for go:embed
	"time"
)

//go:embed coingecko_arbitrum.json
var coingeckoArbitrumTokenListJSON []byte

func init() {
	CoingeckoArbitrumTokenList.ID = "coingeckoArbitrum"
	CoingeckoArbitrumTokenList.SourceURL = "https://prod.market.status.im/v1/token_lists/arbitrum-one/all.json"
	CoingeckoArbitrumTokenList.Fetched = time.Unix(1791450988, 0)
	CoingeckoArbitrumTokenList.JsonData = coingeckoArbitrumTokenListJSON
}
