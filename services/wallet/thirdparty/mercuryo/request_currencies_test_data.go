package mercuryo

func getTestCurrenciesOKResponse() []byte {
	return []byte(`{
		"status": 200,
		"data": {
			"fiat": [
				"AED",
				"AMD",
				"AUD",
				"BGN",
				"BRL",
				"CAD",
				"CHF",
				"COP",
				"CZK",
				"DKK",
				"DOP",
				"EUR",
				"GBP",
				"GHS",
				"HKD",
				"HUF",
				"IDR",
				"ILS",
				"INR",
				"ISK",
				"JOD",
				"JPY",
				"KRW",
				"KZT",
				"LKR",
				"MXN",
				"NOK",
				"NZD",
				"PEN",
				"PHP",
				"PLN",
				"QAR",
				"RON",
				"SEK",
				"SGD",
				"THB",
				"TRY",
				"TWD",
				"USD",
				"UYU",
				"VND",
				"ZAR"
			],
			"crypto": [
				"BTC",
				"ETH",
				"BAT",
				"USDT",
				"ALGO",
				"TRX",
				"OKB",
				"BCH",
				"DAI",
				"TON",
				"BNB",
				"1INCH",
				"NEAR",
				"SOL",
				"DOT",
				"ADA",
				"KSM",
				"MATIC",
				"ATOM",
				"AVAX",
				"XLM",
				"XRP",
				"LTC",
				"SAND",
				"DYDX",
				"MANA",
				"USDC",
				"CRV",
				"SHIB",
				"FTM",
				"DOGE",
				"LINK",
				"XTZ",
				"DASH",
				"WEMIX",
				"TIA",
				"ARB",
				"NOT",
				"SWEAT",
				"INJ"
			],
			"config": {
				"base": {
					"BTC": "BTC",
					"ETH": "ETH",
					"BAT": "ETH",
					"USDT": "ETH",
					"ALGO": "ALGO",
					"TRX": "TRX",
					"OKB": "ETH",
					"BCH": "BCH",
					"DAI": "ETH",
					"TON": "TON",
					"BNB": "BNB",
					"1INCH": "BNB",
					"NEAR": "NEAR",
					"SOL": "SOL",
					"DOT": "DOT",
					"ADA": "ADA",
					"KSM": "KSM",
					"MATIC": "MATIC",
					"ATOM": "ATOM",
					"AVAX": "AVAX",
					"XLM": "XLM",
					"XRP": "XRP",
					"LTC": "LTC",
					"SAND": "ETH",
					"DYDX": "ETH",
					"MANA": "ETH",
					"USDC": "ETH",
					"CRV": "ETH",
					"SHIB": "ETH",
					"FTM": "FTM",
					"DOGE": "DOGE",
					"LINK": "ETH",
					"XTZ": "XTZ",
					"DASH": "DASH",
					"WEMIX": "WEMIX",
					"TIA": "TIA",
					"ARB": "ARB",
					"NOT": "NOT",
					"SWEAT": "NEAR",
					"INJ": "INJ"
				},
				"has_withdrawal_fee": {
					"BTC": true,
					"ETH": true,
					"BAT": true,
					"USDT": true,
					"ALGO": true,
					"TRX": true,
					"OKB": true,
					"BCH": true,
					"DAI": true,
					"TON": false,
					"BNB": true,
					"1INCH": true,
					"NEAR": true,
					"SOL": true,
					"DOT": true,
					"ADA": true,
					"KSM": true,
					"MATIC": true,
					"ATOM": true,
					"AVAX": true,
					"XLM": true,
					"XRP": true,
					"LTC": true,
					"SAND": true,
					"DYDX": true,
					"MANA": true,
					"USDC": true,
					"CRV": true,
					"SHIB": true,
					"FTM": true,
					"DOGE": true,
					"LINK": true,
					"XTZ": true,
					"DASH": true,
					"WEMIX": true,
					"TIA": true,
					"ARB": true,
					"NOT": true,
					"SWEAT": true,
					"INJ": true
				},
				"display_options": {
					"AED": {
						"fullname": "United Arab Emirates Dirham",
						"total_digits": 2,
						"display_digits": 2
					},
					"AMD": {
						"fullname": "Armenian Dram",
						"total_digits": 2,
						"display_digits": 2
					},
					"ARS": {
						"fullname": "Argentine peso",
						"total_digits": 2,
						"display_digits": 2
					},
					"AUD": {
						"fullname": "Australian dollar",
						"total_digits": 2,
						"display_digits": 2
					},
					"BGN": {
						"fullname": "Bulgarian lev",
						"total_digits": 2,
						"display_digits": 2
					},
					"BRL": {
						"fullname": "Brazilian real",
						"total_digits": 2,
						"display_digits": 2
					},
					"CAD": {
						"fullname": "Canadian dollar",
						"total_digits": 2,
						"display_digits": 2
					},
					"CHF": {
						"fullname": "Swiss frank",
						"total_digits": 2,
						"display_digits": 2
					},
					"COP": {
						"fullname": "Colombian Peso",
						"total_digits": 2,
						"display_digits": 2
					},
					"CZK": {
						"fullname": "Czech koruna",
						"total_digits": 2,
						"display_digits": 2
					},
					"DKK": {
						"fullname": "Danish krone",
						"total_digits": 2,
						"display_digits": 2
					},
					"DOP": {
						"fullname": "Dominican Peso",
						"total_digits": 2,
						"display_digits": 2
					},
					"EUR": {
						"fullname": "Euro",
						"total_digits": 2,
						"display_digits": 2
					},
					"GBP": {
						"fullname": "Pound sterling",
						"total_digits": 2,
						"display_digits": 2
					},
					"GEL": {
						"fullname": "Georgian Lari",
						"total_digits": 2,
						"display_digits": 2
					},
					"GHS": {
						"fullname": "Ghanaian cedi",
						"total_digits": 2,
						"display_digits": 2
					},
					"HKD": {
						"fullname": "Hong Kong dollar",
						"total_digits": 2,
						"display_digits": 2
					},
					"HUF": {
						"fullname": "Hungarian Forint",
						"total_digits": 2,
						"display_digits": 2
					},
					"IDR": {
						"fullname": "Indonesian rupiah",
						"total_digits": 0,
						"display_digits": 0
					},
					"ILS": {
						"fullname": "Israeli shekel",
						"total_digits": 2,
						"display_digits": 2
					},
					"INR": {
						"fullname": "Indian rupee",
						"total_digits": 2,
						"display_digits": 2
					},
					"ISK": {
						"fullname": "Icelandic Krona",
						"total_digits": 0,
						"display_digits": 0
					},
					"JOD": {
						"fullname": "Jordanian Dinar",
						"total_digits": 2,
						"display_digits": 2
					},
					"JPY": {
						"fullname": "Japanese yen",
						"total_digits": 0,
						"display_digits": 0
					},
					"KES": {
						"fullname": "Kenyan shilling",
						"total_digits": 2,
						"display_digits": 2
					},
					"KRW": {
						"fullname": "South Korean won",
						"total_digits": 0,
						"display_digits": 0
					},
					"KZT": {
						"fullname": "Kazakhstani Tenge",
						"total_digits": 2,
						"display_digits": 2
					},
					"LKR": {
						"fullname": "Sri Lankan Rupee",
						"total_digits": 2,
						"display_digits": 2
					},
					"MXN": {
						"fullname": "Mexican peso",
						"total_digits": 2,
						"display_digits": 2
					},
					"NGN": {
						"fullname": "Nigerian naira",
						"total_digits": 2,
						"display_digits": 2
					},
					"NOK": {
						"fullname": "Norwegian krone",
						"total_digits": 2,
						"display_digits": 2
					},
					"NZD": {
						"fullname": "New Zealand Dollar",
						"total_digits": 2,
						"display_digits": 2
					},
					"PEN": {
						"fullname": "Peruvian Nuevo Sol",
						"total_digits": 2,
						"display_digits": 2
					},
					"PHP": {
						"fullname": "Philippine peso",
						"total_digits": 2,
						"display_digits": 2
					},
					"PLN": {
						"fullname": "Polish zloty",
						"total_digits": 2,
						"display_digits": 2
					},
					"QAR": {
						"fullname": "Qatari Riyal",
						"total_digits": 2,
						"display_digits": 2
					},
					"RON": {
						"fullname": "New Romanian Lei",
						"total_digits": 2,
						"display_digits": 2
					},
					"RUB": {
						"fullname": "Russian ruble",
						"total_digits": 2,
						"display_digits": 2
					},
					"SEK": {
						"fullname": "Swedish krona",
						"total_digits": 2,
						"display_digits": 2
					},
					"SGD": {
						"fullname": "Singapore Dollar",
						"total_digits": 2,
						"display_digits": 2
					},
					"THB": {
						"fullname": "Thai Baht",
						"total_digits": 2,
						"display_digits": 2
					},
					"TRY": {
						"fullname": "Turkish lira",
						"total_digits": 2,
						"display_digits": 2
					},
					"TWD": {
						"fullname": "New Taiwan dollar",
						"total_digits": 2,
						"display_digits": 2
					},
					"TZS": {
						"fullname": "Tanzanian shilling",
						"total_digits": 2,
						"display_digits": 2
					},
					"UAH": {
						"fullname": "Ukrainian hryvnia",
						"total_digits": 2,
						"display_digits": 2
					},
					"UGX": {
						"fullname": "Ugandan shilling",
						"total_digits": 2,
						"display_digits": 2
					},
					"USD": {
						"fullname": "US dollar",
						"total_digits": 2,
						"display_digits": 2
					},
					"UYU": {
						"fullname": "Uruguayan Peso",
						"total_digits": 2,
						"display_digits": 2
					},
					"VND": {
						"fullname": "Vietnamese Dong",
						"total_digits": 2,
						"display_digits": 2
					},
					"ZAR": {
						"fullname": "South African Rand",
						"total_digits": 2,
						"display_digits": 2
					},
					"BTC": {
						"fullname": "Bitcoin",
						"total_digits": 8,
						"display_digits": 5
					},
					"ETH": {
						"fullname": "ETH",
						"total_digits": 18,
						"display_digits": 5
					},
					"BAT": {
						"fullname": "Basic attention token",
						"total_digits": 18,
						"display_digits": 5
					},
					"USDT": {
						"fullname": "Tether",
						"total_digits": 6,
						"display_digits": 2
					},
					"ALGO": {
						"fullname": "Algorand",
						"total_digits": 6,
						"display_digits": 6
					},
					"TRX": {
						"fullname": "Tron",
						"total_digits": 8,
						"display_digits": 2
					},
					"OKB": {
						"fullname": "OKB",
						"total_digits": 18,
						"display_digits": 4
					},
					"BCH": {
						"fullname": "Bitcoin cash",
						"total_digits": 8,
						"display_digits": 5
					},
					"DAI": {
						"fullname": "Dai Stablecoin",
						"total_digits": 18,
						"display_digits": 5
					},
					"TON": {
						"fullname": "The Open Network",
						"total_digits": 9,
						"display_digits": 4
					},
					"BNB": {
						"fullname": "Binance Coin",
						"total_digits": 18,
						"display_digits": 6
					},
					"1INCH": {
						"fullname": "1inch Network",
						"total_digits": 18,
						"display_digits": 6
					},
					"NEAR": {
						"fullname": "NEAR Protocol",
						"total_digits": 18,
						"display_digits": 6
					},
					"SOL": {
						"fullname": "Solana",
						"total_digits": 9,
						"display_digits": 6
					},
					"DOT": {
						"fullname": "Polkadot",
						"total_digits": 10,
						"display_digits": 6
					},
					"ADA": {
						"fullname": "Cardano",
						"total_digits": 18,
						"display_digits": 6
					},
					"KSM": {
						"fullname": "Kusama",
						"total_digits": 18,
						"display_digits": 6
					},
					"MATIC": {
						"fullname": "Polygon",
						"total_digits": 18,
						"display_digits": 6
					},
					"ATOM": {
						"fullname": "Cosmos",
						"total_digits": 18,
						"display_digits": 6
					},
					"AVAX": {
						"fullname": "Avalanche (C-Chain)",
						"total_digits": 18,
						"display_digits": 6
					},
					"XLM": {
						"fullname": "Stellar",
						"total_digits": 18,
						"display_digits": 6
					},
					"XRP": {
						"fullname": "XRP",
						"total_digits": 6,
						"display_digits": 6
					},
					"LTC": {
						"fullname": "Litecoin",
						"total_digits": 8,
						"display_digits": 8
					},
					"SAND": {
						"fullname": "The Sandbox",
						"total_digits": 18,
						"display_digits": 6
					},
					"DYDX": {
						"fullname": "dYdX",
						"total_digits": 18,
						"display_digits": 6
					},
					"MANA": {
						"fullname": "Decentraland",
						"total_digits": 18,
						"display_digits": 6
					},
					"USDC": {
						"fullname": "USDC",
						"total_digits": 6,
						"display_digits": 6
					},
					"CRV": {
						"fullname": "Curve DAO Token",
						"total_digits": 18,
						"display_digits": 6
					},
					"SHIB": {
						"fullname": "Shiba Inu",
						"total_digits": 18,
						"display_digits": 6
					},
					"FTM": {
						"fullname": "Fantom",
						"total_digits": 18,
						"display_digits": 6
					},
					"DOGE": {
						"fullname": "Dogecoin",
						"total_digits": 8,
						"display_digits": 8
					},
					"LINK": {
						"fullname": "Chainlink",
						"total_digits": 18,
						"display_digits": 6
					},
					"XTZ": {
						"fullname": "Tezos",
						"total_digits": 18,
						"display_digits": 6
					},
					"DASH": {
						"fullname": "DASH",
						"total_digits": 18,
						"display_digits": 6
					},
					"WEMIX": {
						"fullname": "WEMIX",
						"total_digits": 10,
						"display_digits": 6
					},
					"TIA": {
						"fullname": "TIA",
						"total_digits": 10,
						"display_digits": 6
					},
					"ARB": {
						"fullname": "ARB",
						"total_digits": 10,
						"display_digits": 6
					},
					"NOT": {
						"fullname": "NOTCOIN",
						"total_digits": 10,
						"display_digits": 3
					},
					"SWEAT": {
						"fullname": "Sweat",
						"total_digits": 18,
						"display_digits": 4
					},
					"INJ": {
						"fullname": "Injective",
						"total_digits": 18,
						"display_digits": 6
					}
				},
				"icons": {
					"AED": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/aed.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/aed.svg",
							"png": "v1.6/img/icons/currencies/aed.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/aed.png"
					},
					"AMD": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/amd.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/amd.svg",
							"png": "v1.6/img/icons/currencies/amd.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/amd.png"
					},
					"ARS": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/ars.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/ars.svg",
							"png": "v1.6/img/icons/currencies/ars.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/ars.png"
					},
					"AUD": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/aud.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/aud.svg",
							"png": "v1.6/img/icons/currencies/aud.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/aud.png"
					},
					"BGN": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/bgn.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/bgn.svg",
							"png": "v1.6/img/icons/currencies/bgn.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/bgn.png"
					},
					"BRL": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/brl.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/brl.svg",
							"png": "v1.6/img/icons/currencies/brl.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/brl.png"
					},
					"CAD": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/cad.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/cad.svg",
							"png": "v1.6/img/icons/currencies/cad.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/cad.png"
					},
					"CHF": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/chf.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/chf.svg",
							"png": "v1.6/img/icons/currencies/chf.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/chf.png"
					},
					"COP": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/cop.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/cop.svg",
							"png": "v1.6/img/icons/currencies/cop.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/cop.png"
					},
					"CZK": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/czk.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/czk.svg",
							"png": "v1.6/img/icons/currencies/czk.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/czk.png"
					},
					"DKK": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/dkk.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/dkk.svg",
							"png": "v1.6/img/icons/currencies/dkk.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/dkk.png"
					},
					"DOP": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/dop.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/dop.svg",
							"png": "v1.6/img/icons/currencies/dop.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/dop.png"
					},
					"EUR": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/eur.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/eur.svg",
							"png": "v1.6/img/icons/currencies/eur.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/eur.png"
					},
					"GBP": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/gbp.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/gbp.svg",
							"png": "v1.6/img/icons/currencies/gbp.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/gbp.png"
					},
					"GEL": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/default.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/default.svg",
							"png": "v1.6/img/icons/currencies/default.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/default.png"
					},
					"GHS": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/ghs.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/ghs.svg",
							"png": "v1.6/img/icons/currencies/ghs.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/ghs.png"
					},
					"HKD": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/hkd.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/hkd.svg",
							"png": "v1.6/img/icons/currencies/hkd.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/hkd.png"
					},
					"HUF": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/huf.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/huf.svg",
							"png": "v1.6/img/icons/currencies/huf.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/huf.png"
					},
					"IDR": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/idr.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/idr.svg",
							"png": "v1.6/img/icons/currencies/idr.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/idr.png"
					},
					"ILS": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/ils.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/ils.svg",
							"png": "v1.6/img/icons/currencies/ils.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/ils.png"
					},
					"INR": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/inr.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/inr.svg",
							"png": "v1.6/img/icons/currencies/inr.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/inr.png"
					},
					"ISK": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/isk.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/isk.svg",
							"png": "v1.6/img/icons/currencies/isk.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/isk.png"
					},
					"JOD": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/jod.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/jod.svg",
							"png": "v1.6/img/icons/currencies/jod.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/jod.png"
					},
					"JPY": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/jpy.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/jpy.svg",
							"png": "v1.6/img/icons/currencies/jpy.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/jpy.png"
					},
					"KES": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/kes.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/kes.svg",
							"png": "v1.6/img/icons/currencies/kes.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/kes.png"
					},
					"KRW": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/krw.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/krw.svg",
							"png": "v1.6/img/icons/currencies/krw.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/krw.png"
					},
					"KZT": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/kzt.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/kzt.svg",
							"png": "v1.6/img/icons/currencies/kzt.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/kzt.png"
					},
					"LKR": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/lkr.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/lkr.svg",
							"png": "v1.6/img/icons/currencies/lkr.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/lkr.png"
					},
					"MXN": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/mxn.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/mxn.svg",
							"png": "v1.6/img/icons/currencies/mxn.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/mxn.png"
					},
					"NGN": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/ngn.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/ngn.svg",
							"png": "v1.6/img/icons/currencies/ngn.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/ngn.png"
					},
					"NOK": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/nok.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/nok.svg",
							"png": "v1.6/img/icons/currencies/nok.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/nok.png"
					},
					"NZD": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/nzd.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/nzd.svg",
							"png": "v1.6/img/icons/currencies/nzd.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/nzd.png"
					},
					"PEN": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/pen.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/pen.svg",
							"png": "v1.6/img/icons/currencies/pen.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/pen.png"
					},
					"PHP": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/php.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/php.svg",
							"png": "v1.6/img/icons/currencies/php.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/php.png"
					},
					"PLN": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/pln.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/pln.svg",
							"png": "v1.6/img/icons/currencies/pln.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/pln.png"
					},
					"QAR": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/qar.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/qar.svg",
							"png": "v1.6/img/icons/currencies/qar.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/qar.png"
					},
					"RON": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/ron.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/ron.svg",
							"png": "v1.6/img/icons/currencies/ron.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/ron.png"
					},
					"RUB": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/rub.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/rub.svg",
							"png": "v1.6/img/icons/currencies/rub.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/rub.png"
					},
					"SEK": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/sek.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/sek.svg",
							"png": "v1.6/img/icons/currencies/sek.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/sek.png"
					},
					"SGD": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/sgd.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/sgd.svg",
							"png": "v1.6/img/icons/currencies/sgd.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/sgd.png"
					},
					"THB": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/thb.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/thb.svg",
							"png": "v1.6/img/icons/currencies/thb.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/thb.png"
					},
					"TRY": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/try.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/try.svg",
							"png": "v1.6/img/icons/currencies/try.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/try.png"
					},
					"TWD": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/twd.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/twd.svg",
							"png": "v1.6/img/icons/currencies/twd.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/twd.png"
					},
					"TZS": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/tzs.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/tzs.svg",
							"png": "v1.6/img/icons/currencies/tzs.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/tzs.png"
					},
					"UAH": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/uah.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/uah.svg",
							"png": "v1.6/img/icons/currencies/uah.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/uah.png"
					},
					"UGX": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/ugx.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/ugx.svg",
							"png": "v1.6/img/icons/currencies/ugx.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/ugx.png"
					},
					"USD": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/usd.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/usd.svg",
							"png": "v1.6/img/icons/currencies/usd.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/usd.png"
					},
					"UYU": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/uyu.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/uyu.svg",
							"png": "v1.6/img/icons/currencies/uyu.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/uyu.png"
					},
					"VND": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/vnd.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/vnd.svg",
							"png": "v1.6/img/icons/currencies/vnd.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/vnd.png"
					},
					"ZAR": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/zar.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/zar.svg",
							"png": "v1.6/img/icons/currencies/zar.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/zar.png"
					},
					"BTC": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/btc.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/btc.svg",
							"png": "v1.6/img/icons/currencies/btc.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/btc.png"
					},
					"ETH": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/eth.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/eth.svg",
							"png": "v1.6/img/icons/currencies/eth.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/eth.png"
					},
					"BAT": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/bat.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/bat.svg",
							"png": "v1.6/img/icons/currencies/bat.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/bat.png"
					},
					"USDT": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/usdt.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/usdt.svg",
							"png": "v1.6/img/icons/currencies/usdt.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/usdt.png"
					},
					"ALGO": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/algo.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/algo.svg",
							"png": "v1.6/img/icons/currencies/algo.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/algo.png"
					},
					"TRX": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/trx.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/trx.svg",
							"png": "v1.6/img/icons/currencies/trx.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/trx.png"
					},
					"OKB": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/okb.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/okb.svg",
							"png": "v1.6/img/icons/currencies/okb.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/okb.png"
					},
					"BCH": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/bch.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/bch.svg",
							"png": "v1.6/img/icons/currencies/bch.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/bch.png"
					},
					"DAI": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/dai.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/dai.svg",
							"png": "v1.6/img/icons/currencies/dai.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/dai.png"
					},
					"TON": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/ton.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/ton.svg",
							"png": "v1.6/img/icons/currencies/ton.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/ton.png"
					},
					"BNB": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/bnb.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/bnb.svg",
							"png": "v1.6/img/icons/currencies/bnb.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/bnb.png"
					},
					"1INCH": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/1inch.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/1inch.svg",
							"png": "v1.6/img/icons/currencies/1inch.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/1inch.png"
					},
					"NEAR": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/near.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/near.svg",
							"png": "v1.6/img/icons/currencies/near.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/near.png"
					},
					"SOL": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/sol.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/sol.svg",
							"png": "v1.6/img/icons/currencies/sol.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/sol.png"
					},
					"DOT": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/dot.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/dot.svg",
							"png": "v1.6/img/icons/currencies/dot.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/dot.png"
					},
					"ADA": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/ada.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/ada.svg",
							"png": "v1.6/img/icons/currencies/ada.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/ada.png"
					},
					"KSM": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/ksm.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/ksm.svg",
							"png": "v1.6/img/icons/currencies/ksm.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/ksm.png"
					},
					"MATIC": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/matic.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/matic.svg",
							"png": "v1.6/img/icons/currencies/matic.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/matic.png"
					},
					"ATOM": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/atom.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/atom.svg",
							"png": "v1.6/img/icons/currencies/atom.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/atom.png"
					},
					"AVAX": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/avax.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/avax.svg",
							"png": "v1.6/img/icons/currencies/avax.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/avax.png"
					},
					"XLM": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/xlm.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/xlm.svg",
							"png": "v1.6/img/icons/currencies/xlm.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/xlm.png"
					},
					"XRP": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/xrp.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/xrp.svg",
							"png": "v1.6/img/icons/currencies/xrp.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/xrp.png"
					},
					"LTC": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/ltc.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/ltc.svg",
							"png": "v1.6/img/icons/currencies/ltc.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/ltc.png"
					},
					"SAND": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/sand.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/sand.svg",
							"png": "v1.6/img/icons/currencies/sand.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/sand.png"
					},
					"DYDX": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/dydx.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/dydx.svg",
							"png": "v1.6/img/icons/currencies/dydx.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/dydx.png"
					},
					"MANA": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/mana.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/mana.svg",
							"png": "v1.6/img/icons/currencies/mana.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/mana.png"
					},
					"USDC": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/usdc.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/usdc.svg",
							"png": "v1.6/img/icons/currencies/usdc.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/usdc.png"
					},
					"CRV": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/crv.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/crv.svg",
							"png": "v1.6/img/icons/currencies/crv.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/crv.png"
					},
					"SHIB": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/shib.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/shib.svg",
							"png": "v1.6/img/icons/currencies/shib.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/shib.png"
					},
					"FTM": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/ftm.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/ftm.svg",
							"png": "v1.6/img/icons/currencies/ftm.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/ftm.png"
					},
					"DOGE": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/doge.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/doge.svg",
							"png": "v1.6/img/icons/currencies/doge.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/doge.png"
					},
					"LINK": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/link.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/link.svg",
							"png": "v1.6/img/icons/currencies/link.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/link.png"
					},
					"XTZ": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/xtz.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/xtz.svg",
							"png": "v1.6/img/icons/currencies/xtz.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/xtz.png"
					},
					"DASH": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/dash.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/dash.svg",
							"png": "v1.6/img/icons/currencies/dash.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/dash.png"
					},
					"WEMIX": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/wemix.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/wemix.svg",
							"png": "v1.6/img/icons/currencies/wemix.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/wemix.png"
					},
					"TIA": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/tia.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/tia.svg",
							"png": "v1.6/img/icons/currencies/tia.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/tia.png"
					},
					"ARB": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/arb.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/arb.svg",
							"png": "v1.6/img/icons/currencies/arb.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/arb.png"
					},
					"NOT": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/not.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/not.svg",
							"png": "v1.6/img/icons/currencies/not.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/not.png"
					},
					"SWEAT": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/sweat.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/sweat.svg",
							"png": "v1.6/img/icons/currencies/sweat.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/sweat.png"
					},
					"INJ": {
						"svg": "https://api.mercuryo.io/v1.6/img/icons/currencies/inj.svg",
						"relative": {
							"svg": "v1.6/img/icons/currencies/inj.svg",
							"png": "v1.6/img/icons/currencies/inj.png"
						},
						"png": "https://api.mercuryo.io/v1.6/img/icons/currencies/inj.png"
					}
				},
				"networks": {
					"ALGORAND": {
						"name": "ALGORAND",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/default.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/default.svg",
								"png": "v1.6/img/icons/networks/default.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/default.png"
						}
					},
					"ARBITRUM": {
						"name": "ARBITRUM",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/arbitrum.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/arbitrum.svg",
								"png": "v1.6/img/icons/networks/arbitrum.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/arbitrum.png"
						}
					},
					"AVALANCHE": {
						"name": "AVALANCHE",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/default.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/default.svg",
								"png": "v1.6/img/icons/networks/default.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/default.png"
						}
					},
					"BASE": {
						"name": "BASE",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/base.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/base.svg",
								"png": "v1.6/img/icons/networks/base.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/base.png"
						}
					},
					"BINANCESMARTCHAIN": {
						"name": "BINANCESMARTCHAIN",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/binancesmartchain.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/binancesmartchain.svg",
								"png": "v1.6/img/icons/networks/binancesmartchain.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/binancesmartchain.png"
						}
					},
					"BITCOIN": {
						"name": "BITCOIN",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/default.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/default.svg",
								"png": "v1.6/img/icons/networks/default.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/default.png"
						}
					},
					"BITCOINCASH": {
						"name": "BITCOINCASH",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/default.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/default.svg",
								"png": "v1.6/img/icons/networks/default.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/default.png"
						}
					},
					"CARDANO": {
						"name": "CARDANO",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/default.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/default.svg",
								"png": "v1.6/img/icons/networks/default.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/default.png"
						}
					},
					"CELESTIA": {
						"name": "CELESTIA",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/celestia.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/celestia.svg",
								"png": "v1.6/img/icons/networks/celestia.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/celestia.png"
						}
					},
					"COSMOS": {
						"name": "COSMOS",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/default.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/default.svg",
								"png": "v1.6/img/icons/networks/default.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/default.png"
						}
					},
					"CRONOS": {
						"name": "CRONOS",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/default.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/default.svg",
								"png": "v1.6/img/icons/networks/default.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/default.png"
						}
					},
					"DASH": {
						"name": "DASH",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/default.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/default.svg",
								"png": "v1.6/img/icons/networks/default.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/default.png"
						}
					},
					"DOGECOIN": {
						"name": "DOGECOIN",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/default.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/default.svg",
								"png": "v1.6/img/icons/networks/default.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/default.png"
						}
					},
					"ETHEREUM": {
						"name": "ETHEREUM",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/ethereum.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/ethereum.svg",
								"png": "v1.6/img/icons/networks/ethereum.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/ethereum.png"
						}
					},
					"FANTOM": {
						"name": "FANTOM",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/default.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/default.svg",
								"png": "v1.6/img/icons/networks/default.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/default.png"
						}
					},
					"FLOW": {
						"name": "FLOW",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/default.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/default.svg",
								"png": "v1.6/img/icons/networks/default.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/default.png"
						}
					},
					"INJECTIVE": {
						"name": "INJECTIVE",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/injective.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/injective.svg",
								"png": "v1.6/img/icons/networks/injective.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/injective.png"
						}
					},
					"KAVA": {
						"name": "KAVA",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/default.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/default.svg",
								"png": "v1.6/img/icons/networks/default.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/default.png"
						}
					},
					"KUSAMA": {
						"name": "KUSAMA",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/default.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/default.svg",
								"png": "v1.6/img/icons/networks/default.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/default.png"
						}
					},
					"LINEA": {
						"name": "LINEA",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/linea.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/linea.svg",
								"png": "v1.6/img/icons/networks/linea.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/linea.png"
						}
					},
					"LITECOIN": {
						"name": "LITECOIN",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/default.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/default.svg",
								"png": "v1.6/img/icons/networks/default.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/default.png"
						}
					},
					"NEAR_PROTOCOL": {
						"name": "NEAR_PROTOCOL",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/near_protocol.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/near_protocol.svg",
								"png": "v1.6/img/icons/networks/near_protocol.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/near_protocol.png"
						}
					},
					"NEWTON": {
						"name": "NEWTON",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/default.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/default.svg",
								"png": "v1.6/img/icons/networks/default.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/default.png"
						}
					},
					"OPTIMISM": {
						"name": "OPTIMISM",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/optimism.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/optimism.svg",
								"png": "v1.6/img/icons/networks/optimism.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/optimism.png"
						}
					},
					"POLKADOT": {
						"name": "POLKADOT",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/default.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/default.svg",
								"png": "v1.6/img/icons/networks/default.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/default.png"
						}
					},
					"POLYGON": {
						"name": "POLYGON",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/polygon.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/polygon.svg",
								"png": "v1.6/img/icons/networks/polygon.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/polygon.png"
						}
					},
					"RIPPLE": {
						"name": "RIPPLE",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/default.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/default.svg",
								"png": "v1.6/img/icons/networks/default.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/default.png"
						}
					},
					"SOLANA": {
						"name": "SOLANA",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/solana.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/solana.svg",
								"png": "v1.6/img/icons/networks/solana.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/solana.png"
						}
					},
					"STELLAR": {
						"name": "STELLAR",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/stellar.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/stellar.svg",
								"png": "v1.6/img/icons/networks/stellar.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/stellar.png"
						}
					},
					"TERRA": {
						"name": "TERRA",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/terra.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/terra.svg",
								"png": "v1.6/img/icons/networks/terra.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/terra.png"
						}
					},
					"TEZOS": {
						"name": "TEZOS",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/default.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/default.svg",
								"png": "v1.6/img/icons/networks/default.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/default.png"
						}
					},
					"TRON": {
						"name": "TRON",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/tron.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/tron.svg",
								"png": "v1.6/img/icons/networks/tron.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/tron.png"
						}
					},
					"WEMIX": {
						"name": "WEMIX",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/wemix.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/wemix.svg",
								"png": "v1.6/img/icons/networks/wemix.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/wemix.png"
						}
					},
					"ZKSYNC": {
						"name": "ZKSYNC",
						"icons": {
							"svg": "https://api.mercuryo.io/v1.6/img/icons/networks/zksync.svg",
							"relative": {
								"svg": "v1.6/img/icons/networks/zksync.svg",
								"png": "v1.6/img/icons/networks/zksync.png"
							},
							"png": "https://api.mercuryo.io/v1.6/img/icons/networks/zksync.png"
						}
					}
				},
				"crypto_currencies": [
					{
						"currency": "BTC",
						"network": "BITCOIN",
						"show_network_icon": false,
						"network_label": "BITCOIN",
						"contract": ""
					},
					{
						"currency": "ETH",
						"network": "ZKSYNC",
						"show_network_icon": true,
						"network_label": "ZKSYNC",
						"contract": ""
					},
					{
						"currency": "ETH",
						"network": "ARBITRUM",
						"show_network_icon": true,
						"network_label": "ARBITRUM",
						"contract": ""
					},
					{
						"currency": "ETH",
						"network": "OPTIMISM",
						"show_network_icon": true,
						"network_label": "OPTIMISM",
						"contract": ""
					},
					{
						"currency": "ETH",
						"network": "BASE",
						"show_network_icon": true,
						"network_label": "BASE",
						"contract": ""
					},
					{
						"currency": "ETH",
						"network": "LINEA",
						"show_network_icon": true,
						"network_label": "LINEA",
						"contract": ""
					},
					{
						"currency": "ETH",
						"network": "ETHEREUM",
						"show_network_icon": true,
						"network_label": "ETHEREUM",
						"contract": ""
					},
					{
						"currency": "BAT",
						"network": "ETHEREUM",
						"show_network_icon": true,
						"network_label": "ERC-20",
						"contract": "0x0d8775f648430679a709e98d2b0cb6250d2887ef"
					},
					{
						"currency": "USDT",
						"network": "POLYGON",
						"show_network_icon": true,
						"network_label": "POLYGON",
						"contract": "0xc2132D05D31c914a87C6611C10748AEb04B58e8F"
					},
					{
						"currency": "USDT",
						"network": "NEWTON",
						"show_network_icon": true,
						"network_label": "TON",
						"contract": "EQCxE6mUtQJKFnGfaROTKOt1lZbDiiX1kCixRv7Nw2Id_sDs"
					},
					{
						"currency": "USDT",
						"network": "ETHEREUM",
						"show_network_icon": true,
						"network_label": "ERC-20",
						"contract": "0xdac17f958d2ee523a2206206994597c13d831ec7",
						"network_ud": "ERC20"
					},
					{
						"currency": "USDT",
						"network": "TRON",
						"show_network_icon": true,
						"network_label": "TRC-20",
						"contract": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
						"network_ud": "TRON"
					},
					{
						"currency": "ALGO",
						"network": "ALGORAND",
						"show_network_icon": false,
						"network_label": "ALGORAND",
						"contract": ""
					},
					{
						"currency": "TRX",
						"network": "TRON",
						"show_network_icon": false,
						"network_label": "TRC-20",
						"contract": ""
					},
					{
						"currency": "OKB",
						"network": "ETHEREUM",
						"show_network_icon": true,
						"network_label": "ERC-20",
						"contract": "0x75231f58b43240c9718dd58b4967c5114342a86c"
					},
					{
						"currency": "BCH",
						"network": "BITCOINCASH",
						"show_network_icon": false,
						"network_label": "BITCOINCASH",
						"contract": ""
					},
					{
						"currency": "DAI",
						"network": "ETHEREUM",
						"show_network_icon": true,
						"network_label": "ERC-20",
						"contract": "0x6b175474e89094c44da98b954eedeac495271d0f"
					},
					{
						"currency": "TON",
						"network": "NEWTON",
						"show_network_icon": false,
						"network_label": "NEWTON",
						"contract": ""
					},
					{
						"currency": "BNB",
						"network": "BINANCESMARTCHAIN",
						"show_network_icon": false,
						"network_label": "BEP-20",
						"contract": ""
					},
					{
						"currency": "1INCH",
						"network": "BINANCESMARTCHAIN",
						"show_network_icon": true,
						"network_label": "BEP-20",
						"contract": "0x111111111117dc0aa78b770fa6a738034120c302"
					},
					{
						"currency": "NEAR",
						"network": "NEAR_PROTOCOL",
						"show_network_icon": false,
						"network_label": "NEAR_PROTOCOL",
						"contract": ""
					},
					{
						"currency": "SOL",
						"network": "SOLANA",
						"show_network_icon": false,
						"network_label": "SOLANA",
						"contract": ""
					},
					{
						"currency": "DOT",
						"network": "POLKADOT",
						"show_network_icon": false,
						"network_label": "POLKADOT",
						"contract": ""
					},
					{
						"currency": "ADA",
						"network": "CARDANO",
						"show_network_icon": false,
						"network_label": "CARDANO",
						"contract": ""
					},
					{
						"currency": "KSM",
						"network": "KUSAMA",
						"show_network_icon": false,
						"network_label": "KUSAMA",
						"contract": ""
					},
					{
						"currency": "MATIC",
						"network": "POLYGON",
						"show_network_icon": false,
						"network_label": "POLYGON",
						"contract": "",
						"network_ud": "MATIC"
					},
					{
						"currency": "ATOM",
						"network": "COSMOS",
						"show_network_icon": false,
						"network_label": "COSMOS",
						"contract": ""
					},
					{
						"currency": "AVAX",
						"network": "AVALANCHE",
						"show_network_icon": false,
						"network_label": "AVALANCHE",
						"contract": ""
					},
					{
						"currency": "XLM",
						"network": "STELLAR",
						"show_network_icon": false,
						"network_label": "STELLAR",
						"contract": ""
					},
					{
						"currency": "XRP",
						"network": "RIPPLE",
						"show_network_icon": false,
						"network_label": "RIPPLE",
						"contract": ""
					},
					{
						"currency": "LTC",
						"network": "LITECOIN",
						"show_network_icon": false,
						"network_label": "LITECOIN",
						"contract": ""
					},
					{
						"currency": "SAND",
						"network": "ETHEREUM",
						"show_network_icon": true,
						"network_label": "ERC-20",
						"contract": "0x3845badade8e6dff049820680d1f14bd3903a5d0",
						"network_ud": "ERC20"
					},
					{
						"currency": "DYDX",
						"network": "ETHEREUM",
						"show_network_icon": true,
						"network_label": "ERC-20",
						"contract": "0x92d6c1e31e14520e676a687f0a93788b716beff5"
					},
					{
						"currency": "MANA",
						"network": "ETHEREUM",
						"show_network_icon": true,
						"network_label": "ERC-20",
						"contract": "0x0f5d2fb29fb7d3cfee444a200298f468908cc942",
						"network_ud": "ERC20"
					},
					{
						"currency": "USDC",
						"network": "ETHEREUM",
						"show_network_icon": true,
						"network_label": "ETHEREUM",
						"contract": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
					},
					{
						"currency": "USDC",
						"network": "POLYGON",
						"show_network_icon": true,
						"network_label": "POLYGON",
						"contract": "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359"
					},
					{
						"currency": "USDC",
						"network": "ARBITRUM",
						"show_network_icon": true,
						"network_label": "ARBITRUM",
						"contract": "0xaf88d065e77c8cC2239327C5EDb3A432268e5831"
					},
					{
						"currency": "USDC",
						"network": "NEAR_PROTOCOL",
						"show_network_icon": true,
						"network_label": "NEAR",
						"contract": "17208628f84f5d6ad33f0da3bbbeb27ffcb398eac501a31bd6ad2011e36133a1"
					},
					{
						"currency": "USDC",
						"network": "SOLANA",
						"show_network_icon": true,
						"network_label": "SOLANA",
						"contract": "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
					},
					{
						"currency": "USDC",
						"network": "STELLAR",
						"show_network_icon": true,
						"network_label": "STELLAR",
						"contract": "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN"
					},
					{
						"currency": "CRV",
						"network": "ETHEREUM",
						"show_network_icon": true,
						"network_label": "ERC-20",
						"contract": "0xd533a949740bb3306d119cc777fa900ba034cd52",
						"network_ud": "ERC20"
					},
					{
						"currency": "SHIB",
						"network": "ETHEREUM",
						"show_network_icon": true,
						"network_label": "ERC-20",
						"contract": "0x95aD61b0a150d79219dCF64E1E6Cc01f0B64C4cE",
						"network_ud": "ERC20"
					},
					{
						"currency": "FTM",
						"network": "FANTOM",
						"show_network_icon": false,
						"network_label": "FANTOM",
						"contract": ""
					},
					{
						"currency": "DOGE",
						"network": "DOGECOIN",
						"show_network_icon": false,
						"network_label": "DOGECOIN",
						"contract": ""
					},
					{
						"currency": "LINK",
						"network": "ETHEREUM",
						"show_network_icon": true,
						"network_label": "ERC-20",
						"contract": "0x514910771AF9Ca656af840dff83E8264EcF986CA"
					},
					{
						"currency": "XTZ",
						"network": "TEZOS",
						"show_network_icon": false,
						"network_label": "TEZOS",
						"contract": ""
					},
					{
						"currency": "DASH",
						"network": "DASH",
						"show_network_icon": false,
						"network_label": "DASH",
						"contract": ""
					},
					{
						"currency": "WEMIX",
						"network": "WEMIX",
						"show_network_icon": false,
						"network_label": "WEMIX",
						"contract": ""
					},
					{
						"currency": "TIA",
						"network": "CELESTIA",
						"show_network_icon": false,
						"network_label": "CELESTIA",
						"contract": ""
					},
					{
						"currency": "ARB",
						"network": "ARBITRUM",
						"show_network_icon": true,
						"network_label": "ARBITRUM",
						"contract": "0x912CE59144191C1204E64559FE8253a0e49E6548"
					},
					{
						"currency": "NOT",
						"network": "NEWTON",
						"show_network_icon": true,
						"network_label": "TON",
						"contract": "EQAvlWFDxGF2lXm67y4yzC17wYKD9A0guwPkMs1gOsM__NOT"
					},
					{
						"currency": "SWEAT",
						"network": "NEAR_PROTOCOL",
						"show_network_icon": true,
						"network_label": "NEAR",
						"contract": "token.sweat"
					},
					{
						"currency": "INJ",
						"network": "INJECTIVE",
						"show_network_icon": false,
						"network_label": "INJECTIVE",
						"contract": ""
					}
				],
				"default_networks": {
					"BTC": "BITCOIN",
					"ETH": "ETHEREUM",
					"BAT": "ETHEREUM",
					"USDT": "ETHEREUM",
					"ALGO": "ALGORAND",
					"TRX": "TRON",
					"OKB": "ETHEREUM",
					"BCH": "BITCOINCASH",
					"DAI": "ETHEREUM",
					"TON": "NEWTON",
					"BNB": "BINANCESMARTCHAIN",
					"1INCH": "BINANCESMARTCHAIN",
					"NEAR": "NEAR_PROTOCOL",
					"SOL": "SOLANA",
					"DOT": "POLKADOT",
					"ADA": "CARDANO",
					"KSM": "KUSAMA",
					"MATIC": "POLYGON",
					"ATOM": "COSMOS",
					"AVAX": "AVALANCHE",
					"XLM": "STELLAR",
					"XRP": "RIPPLE",
					"LTC": "LITECOIN",
					"SAND": "ETHEREUM",
					"DYDX": "ETHEREUM",
					"MANA": "ETHEREUM",
					"USDC": "ETHEREUM",
					"CRV": "ETHEREUM",
					"SHIB": "ETHEREUM",
					"FTM": "FANTOM",
					"DOGE": "DOGECOIN",
					"LINK": "ETHEREUM",
					"XTZ": "TEZOS",
					"DASH": "DASH",
					"WEMIX": "WEMIX",
					"TIA": "CELESTIA",
					"ARB": "ARBITRUM",
					"NOT": "NEWTON",
					"SWEAT": "NEAR_PROTOCOL",
					"INJ": "INJECTIVE"
				}
			}
		}
	}`)
}
