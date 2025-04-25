package tokentypes

func getTokenPegMap() map[string]string {
	return map[string]string{
		"aUSDC":       "USD",
		"DAI":         "USD",
		"EURC":        "EUR",
		"SAI":         "USD",
		"sUSD":        "USD",
		"PAXG":        "XAU",
		"TCAD":        "CAD",
		"TUSD":        "USD",
		"TGBP":        "GBP",
		"TAUD":        "AUD",
		"USDC":        "USD",
		"USDD":        "USD",
		"USDS":        "USD",
		"USDT":        "USD",
		"USDT (EVM)":  "USD",
		"USDT (BSC)":  "USD",
		"USDP":        "USD",
		"USDSC":       "USD",
		"USDSC (EVM)": "USD",
		"USDSC (BSC)": "USD",
	}
}

func GetTokenPegSymbol(symbol string) string {
	return getTokenPegMap()[symbol]
}
