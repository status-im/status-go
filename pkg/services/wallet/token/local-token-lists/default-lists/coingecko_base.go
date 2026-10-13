package defaulttokenlists

import (
	_ "embed" // for go:embed
	"time"
)

//go:embed coingecko_base.json
var coingeckoBaseTokenListJSON []byte

func init() {
	CoingeckoBaseTokenList.ID = "coingeckoBase"
	CoingeckoBaseTokenList.SourceURL = "https://prod.market.status.im/v1/token_lists/base/all.json"
	CoingeckoBaseTokenList.Fetched = time.Unix(1791450988, 0)
	CoingeckoBaseTokenList.JsonData = coingeckoBaseTokenListJSON
}
