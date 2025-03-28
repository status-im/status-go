package coingecko

var coinsList = []GeckoToken{
	{
		ID:     "valobit",
		Symbol: "vbit",
		Name:   "VALOBIT",
	},
	{
		ID:     "dmx",
		Symbol: "dmx",
		Name:   "DMX",
	},
	{
		ID:     "pedro",
		Symbol: "pedro",
		Name:   "PEDRO",
	},
	{
		ID:     "volta-club",
		Symbol: "volta",
		Name:   "Volta Club",
		Platforms: Platforms{
			Ethereum: "0x9b06f3c5de42d4623d7a2bd940ec735103c68a76",
			Arbitrum: "0x9b06f3c5de42d4623d7a2bd940ec735103c68a76",
		},
	},
	{
		ID:     "don-t-sell-your-bitcoin",
		Symbol: "bitcoin",
		Name:   "DON'T SELL YOUR BITCOIN",
	},
	{
		ID:     "harrypotterobamasonic10in",
		Symbol: "bitcoin",
		Name:   "HarryPotterObamaSonic10Inu (ETH)",
		Platforms: Platforms{
			Ethereum: "0x72e4f9f808c49a2a61de9c5896298920dc4eeea9",
			Base:     "0x2a06a17cbc6d0032cac2c6696da90f29d39a1a29",
		},
	},
	{
		ID:     "dork-lord-coin",
		Symbol: "dlord",
		Name:   "DORK LORD COIN",
	},
	{
		ID:     "dork-lord-eth",
		Symbol: "dorkl",
		Name:   "DORK LORD (SOL)",
	},
	{
		ID:     "dotcom",
		Symbol: "y2k",
		Name:   "Dotcom",
	},
	{
		ID:     "sentinel-bot-ai",
		Symbol: "snt",
		Name:   "Sentinel Bot Ai",
		Platforms: Platforms{
			Ethereum: "0x78ba134c3ace18e69837b01703d07f0db6fb0a60",
		},
	},
	{
		ID:     "status",
		Symbol: "snt",
		Name:   "Status",
		Platforms: Platforms{
			Ethereum: "0x744d70fdbe2ba4cf95131626614a1763df805b9e",
		},
	},
	{
		ID:     "usd-coin",
		Symbol: "usdc",
		Name:   "USDC",
		Platforms: Platforms{
			Ethereum: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			Optimism: "0x0b2c639c533813f4aa9d7837caf62653d097ff85",
			Arbitrum: "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
			Base:     "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
		},
	},
	{
		ID:     "bridged-usdt",
		Symbol: "usdt",
		Name:   "Bridged USDT",
		Platforms: Platforms{
			Optimism: "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58",
		},
	},
}
