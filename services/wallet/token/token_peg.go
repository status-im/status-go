package token

var tokenPeg = map[string]string{
	"aave-usdc-v1":      "USD", // AUSDC
	"dai":               "USD", // DAI
	"euro-coin":         "EUR", // EURC
	"sai":               "USD", // SAI
	"nusd":              "USD", // SUSD
	"pax-gold":          "XAU", // PAXG
	"caduceus-protocol": "CAD", // CAD
	"true-usd":          "USD", // TUSD
	"usd-coin":          "USD", // USDC
	"usdd":              "USD", // USDD
	"usds":              "USD", // USDS
	"tether":            "USD", // USDT
	"paxos-standard":    "USD", // USDP
}

func GetTokenPegSymbol(id string) string {
	return tokenPeg[id]
}
