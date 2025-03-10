package token

var tokenPeg = map[string]string{
	"aUSDC": "USD",
	"DAI":   "USD",
	"EURC":  "EUR",
	"SAI":   "USD",
	"sUSD":  "USD",
	"PAXG":  "XAU",
	"TCAD":  "CAD",
	"TUSD":  "USD",
	"TGBP":  "GBP",
	"TAUD":  "AUD",
	"USDC":  "USD",
	"USDD":  "USD",
	"USDS":  "USD",
	"USDT":  "USD",
	"USDP":  "USD",
	"USDSC": "USD",
}

func GetTokenPegSymbol(symbol string) string {
	return tokenPeg[symbol]
}
