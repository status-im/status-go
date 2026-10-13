package defaulttokenlists

import (
	_ "embed" // for go:embed
	"time"
)

//go:embed coingecko_linea.json
var coingeckoLineaTokenListJSON []byte

func init() {
	CoingeckoLineaTokenList.ID = "coingeckoLinea"
	CoingeckoLineaTokenList.SourceURL = "https://prod.market.status.im/v1/token_lists/linea/all.json"
	CoingeckoLineaTokenList.Fetched = time.Unix(1791450988, 0)
	CoingeckoLineaTokenList.JsonData = coingeckoLineaTokenListJSON
}
