package common

// When adding a new network, look up for the right patform name here https://api.coingecko.com/api/v3/asset_platforms
var CoinGeckoPlatformChainMapper = map[string]uint64{
	"ethereum":            EthereumMainnet,
	"optimistic-ethereum": OptimismMainnet,
	"arbitrum-one":        ArbitrumMainnet,
	"base":                BaseMainnet,
	"binance-smart-chain": BSCMainnet,
	"linea":               LineaMainnet,
	"unichain":            UnichainMainnet,
	"katana":              KatanaMainnet,
	"ink":                 InkMainnet,
	"abstract":            AbstractMainnet,
	"zksync":              ZkSyncMainnet,
	"soneium":             SoneiumMainnet,
	"scroll":              ScrollMainnet,
	"blast":               BlastMainnet,
	"robinhood":           RobinhoodMainnet,
}
