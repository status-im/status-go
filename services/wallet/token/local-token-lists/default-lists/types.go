package defaulttokenlists

import (
	"time"
)

type DownloadedTokenList struct {
	ID        string
	SourceURL string
	Schema    string
	Fetched   time.Time
	JsonData  []byte
}

var (
	StatusTokenList            DownloadedTokenList
	UniswapTokenList           DownloadedTokenList
	CoingeckoEthereumTokenList DownloadedTokenList
	CoingeckoOptimismTokenList DownloadedTokenList
	CoingeckoArbitrumTokenList DownloadedTokenList
	CoingeckoBaseTokenList     DownloadedTokenList
	CoingeckoBscTokenList      DownloadedTokenList
)
