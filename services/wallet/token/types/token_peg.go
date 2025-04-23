package tokentypes

func getTokenPegMap() map[string]string {
	return map[string]string{
		"aUSDC":     "USD",
		"DAI":       "USD",
		"EURC":      "EUR",
		"SAI":       "USD",
		"sUSD":      "USD",
		"PAXG":      "XAU",
		"TCAD":      "CAD",
		"TUSD":      "USD",
		"TGBP":      "GBP",
		"TAUD":      "AUD",
		"USDC":      "USD",
		"USDD":      "USD",
		"USDS":      "USD",
		"USDT":      "USD",
		"USDT(6)":   "USD",
		"USDT(18)":  "USD",
		"USDP":      "USD",
		"USDSC":     "USD",
		"USDSC(6)":  "USD",
		"USDSC(18)": "USD",
	}
}

func GetTokenPegSymbol(symbol string) string {
	return getTokenPegMap()[symbol]
}
