package defaulttokenlists

import (
	_ "embed" // for go:embed
	"time"
)

//go:embed uniswap.json
var uniswapTokenListJSON []byte

func init() {
	UniswapTokenList.ID = "uniswap"
	UniswapTokenList.SourceURL = "https://tokens.uniswap.org"
	UniswapTokenList.Fetched = time.Unix(1791450990, 0)
	UniswapTokenList.JsonData = uniswapTokenListJSON
}
