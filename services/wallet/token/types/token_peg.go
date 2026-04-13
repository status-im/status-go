package tokentypes

func getTokenPegMap() map[string]string {
	return map[string]string{
		// USDT
		"1-0xdac17f958d2ee523a2206206994597c13d831ec7":     "USD",
		"10-0x94b008aa00579c1307b0ef2c499ad98a8ce58e58":    "USD",
		"42161-0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9": "USD",
		"8453-0xfde4c96c8593536e31f229ea8f37b2ada2699bb2":  "USD",
		"56-0x524bc91dc82d6b90ef29f76a3ecaabafffd490bc":    "USD",
		"56-0x55d398326f99059ff775485246999027b3197955":    "USD",

		// TUSD
		"1-0x0000000000085d4780b73119b644ae5ecd22b376":     "USD",
		"10-0xcb59a0a753fdb7491d5f3d794316f1ade197b21e":    "USD",
		"42161-0x4d15a3a2286d883af0aa1b3f21367843fac63e07": "USD",
		"56-0x14016e85a25aeb13065688cafb43044c2ef86784":    "USD",
		"56-0x40af3827f39d0eacbf4a168f8d4ee67c121d11c9":    "USD",

		// DAI
		"1-0x6b175474e89094c44da98b954eedeac495271d0f":        "USD",
		"10-0xda10009cbd5d07dd0cecc66161fc93d7c9000da1":       "USD",
		"42161-0xda10009cbd5d07dd0cecc66161fc93d7c9000da1":    "USD",
		"8453-0x50c5725949a6f0c72e6c4a641f24049a917db0cb":     "USD",
		"56-0x1af3f329e8be154074d8769d1ffa4ee058b1dbc3":       "USD",
		"11155111-0x3e622317f8c93f7328350cf0b56d9ed4c620c5d6": "USD",

		// USDS
		"42161-0xd74f5255d557944cf7dd0e45ff521520002d5748": "USD",
		"42161-0x2ea0be86990e8dac0d09e4316bb92086f304622d": "USD",
		"42161-0x6491c05a82219b8d1479057361ff1654749b876b": "USD",
		"8453-0x820c137fa70c8691f0e44dc420a5e53c168921dc":  "USD",

		// USDC
		"1-0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48":          "USD",
		"10-0x0b2c639c533813f4aa9d7837caf62653d097ff85":         "USD",
		"42161-0xaf88d065e77c8cc2239327c5edb3a432268e5831":      "USD",
		"8453-0x833589fcd6edb6e08f4c7c32d4f71b54bda02913":       "USD",
		"56-0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d":         "USD",
		"11155111-0x1c7d4b196cb0c7b01d743fbc6116a902379c7238":   "USD",
		"421614-0x75faf114eafb1bdbe2f0316df893fd58ce46aa4d":     "USD",
		"84532-0x036cbd53842c5426634e7929541ec2318f3dcf7e":      "USD",
		"11155420-0x5fd84259d66cd46123540766be93dfe6d43130d7":   "USD",
		// TODO: add Status Network Hoodi USDC peg once deployed

		// USDP
		"1-0x8e870d67f660d95d5be530380d0ec0bd388289e1":     "USD",
		"42161-0x78df3a6044ce3cb1905500345b967788b699df8f": "USD",

		// EURC
		"1-0x1abaea1f7c830bd89acc67ec4af516284b1bc33c":          "EUR",
		"42161-0x863708032b5c328e11abcbc0df9d79c71fc52a48":      "EUR",
		"8453-0x60a3e35cc302bfa44cb288bc5a4f316fdb1adb42":       "EUR",
		"11155111-0x08210f9170f89ab7658f0b5e3ff39b0e03c594d4":   "EUR",
		"84532-0x808456652fdb597867f38412077a9182bf77359f":      "EUR",
		// TODO: add Status Network Hoodi EURC peg once deployed

		// SAI
		"1-0x3567aa22cd3ab9aef23d7e18ee0d7cf16974d7e6":    "USD",
		"1-0x89d24a6b4ccb1b6faa2625fe562bdd9a23260359":    "USD",
		"8453-0xcff4429d8a323dd6b64b79a4460bec6d531fcfa8": "USD",

		// PAXG
		"1-0x45804880de22913dafe09f4980848ece6ecbaf78":     "XAU",
		"42161-0xfeb4dfc8c4cf7ed305bb08065d08ec6ee6728429": "XAU",

		// USDD
		"1-0x4f8e5de400de08b164e7421b3ee387f461becd1a":     "USD",
		"42161-0x680447595e8b7b3aa1b43beb9f6098c79ac2ab3f": "USD",
		"56-0x392004bee213f1ff580c867359c246924f21e6ad":    "USD",
	}
}

func GetTokenPegByTokenKey(tokenKey string) string {
	return getTokenPegMap()[tokenKey]
}
