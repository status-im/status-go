package defaulttokenlists

import (
	_ "embed" // for go:embed
	"time"
)

//go:embed coingecko_bsc.json
var coingeckoBscTokenListJSON []byte

func init() {
	CoingeckoBscTokenList.ID = "coingeckoBsc"
	CoingeckoBscTokenList.SourceURL = "https://prod.market.status.im/v1/token_lists/binance-smart-chain/all.json"
	CoingeckoBscTokenList.Fetched = time.Unix(1791450988, 0)
	CoingeckoBscTokenList.JsonData = coingeckoBscTokenListJSON
}
