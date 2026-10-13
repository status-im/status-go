package defaulttokenlists

import (
	_ "embed" // for go:embed
	"time"
)

//go:embed coingecko_ethereum.json
var coingeckoEthereumTokenListJSON []byte

func init() {
	CoingeckoEthereumTokenList.ID = "coingeckoEthereum"
	CoingeckoEthereumTokenList.SourceURL = "https://prod.market.status.im/v1/token_lists/ethereum/all.json"
	CoingeckoEthereumTokenList.Fetched = time.Unix(1791450990, 0)
	CoingeckoEthereumTokenList.JsonData = coingeckoEthereumTokenListJSON
}
