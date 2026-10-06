package defaulttokenlists

import (
	_ "embed" // for go:embed
	"time"
)

//go:embed status.json
var statusTokenListJSON []byte

func init() {
	StatusTokenList.ID = "status"
	StatusTokenList.SourceURL = "https://prod.market.status.im/static/token-list.json"
	StatusTokenList.Fetched = time.Unix(1791450988, 0)
	StatusTokenList.JsonData = statusTokenListJSON
}
