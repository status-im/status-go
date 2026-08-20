package coingecko

var responseError = []byte(`{
  "status": {
    "error_code": 429,
    "error_message": "You've exceeded the Rate Limit. Please visit https://www.coingecko.com/en/api/pricing to subscribe to our API plans for higher rate limits."
  }
}`)

var responseAssetPlatformsData = []byte(`[
  {
    "id": "valobit",
    "chain_identifier": null,
    "name": "Valobit",
    "shortname": "",
    "native_coin_id": "valobit",
    "image": {
      "thumb": "https://coin-images.coingecko.com/asset_platforms/images/182/thumb/valobit.png?1708317270",
      "small": "https://coin-images.coingecko.com/asset_platforms/images/182/small/valobit.png?1708317270",
      "large": "https://coin-images.coingecko.com/asset_platforms/images/182/large/valobit.png?1708317270"
    }
  },
  {
    "id": "factom",
    "chain_identifier": null,
    "name": "Factom",
    "shortname": "",
    "native_coin_id": "factom",
    "image": {
      "thumb": null,
      "small": null,
      "large": null
    }
  },
  {
    "id": "ethereum",
    "chain_identifier": 1,
    "name": "Ethereum",
    "shortname": "Ethereum",
    "native_coin_id": "ethereum",
    "image": {
      "thumb": "https://coin-images.coingecko.com/asset_platforms/images/279/thumb/ethereum.png?1706606803",
      "small": "https://coin-images.coingecko.com/asset_platforms/images/279/small/ethereum.png?1706606803",
      "large": "https://coin-images.coingecko.com/asset_platforms/images/279/large/ethereum.png?1706606803"
    }
  },
  {
    "id": "optimistic-ethereum",
    "chain_identifier": 10,
    "name": "Optimism",
    "shortname": "Optimism",
    "native_coin_id": "ethereum",
    "image": {
      "thumb": "https://coin-images.coingecko.com/asset_platforms/images/41/thumb/optimism.png?1706606778",
      "small": "https://coin-images.coingecko.com/asset_platforms/images/41/small/optimism.png?1706606778",
      "large": "https://coin-images.coingecko.com/asset_platforms/images/41/large/optimism.png?1706606778"
    }
  },
  {
    "id": "arbitrum-one",
    "chain_identifier": 42161,
    "name": "Arbitrum One",
    "shortname": "Arbitrum",
    "native_coin_id": "ethereum",
    "image": {
      "thumb": "https://coin-images.coingecko.com/asset_platforms/images/33/thumb/AO_logomark.png?1706606717",
      "small": "https://coin-images.coingecko.com/asset_platforms/images/33/small/AO_logomark.png?1706606717",
      "large": "https://coin-images.coingecko.com/asset_platforms/images/33/large/AO_logomark.png?1706606717"
    }
  },
  {
    "id": "base",
    "chain_identifier": 8453,
    "name": "Base",
    "shortname": "",
    "native_coin_id": "ethereum",
    "image": {
      "thumb": "https://coin-images.coingecko.com/asset_platforms/images/131/thumb/base-network.png?1720533039",
      "small": "https://coin-images.coingecko.com/asset_platforms/images/131/small/base-network.png?1720533039",
      "large": "https://coin-images.coingecko.com/asset_platforms/images/131/large/base-network.png?1720533039"
    }
  },
  {
    "id": "trustless-computer",
    "chain_identifier": null,
    "name": "Trustless Computer",
    "shortname": "",
    "native_coin_id": "bitcoin",
    "image": {
      "thumb": "https://coin-images.coingecko.com/asset_platforms/images/132/thumb/trustless.jpeg?1706606636",
      "small": "https://coin-images.coingecko.com/asset_platforms/images/132/small/trustless.jpeg?1706606636",
      "large": "https://coin-images.coingecko.com/asset_platforms/images/132/large/trustless.jpeg?1706606636"
    }
  },
  {
    "id": "ordinals",
    "chain_identifier": null,
    "name": "Bitcoin",
    "shortname": "Ordinals",
    "native_coin_id": "bitcoin",
    "image": {
      "thumb": "https://coin-images.coingecko.com/asset_platforms/images/127/thumb/ordinals.png?1706606816",
      "small": "https://coin-images.coingecko.com/asset_platforms/images/127/small/ordinals.png?1706606816",
      "large": "https://coin-images.coingecko.com/asset_platforms/images/127/large/ordinals.png?1706606816"
    }
  },
  {
    "id": "solana",
    "chain_identifier": null,
    "name": "Solana",
    "shortname": "Solana",
    "native_coin_id": "solana",
    "image": {
      "thumb": "https://coin-images.coingecko.com/asset_platforms/images/5/thumb/solana.png?1706606708",
      "small": "https://coin-images.coingecko.com/asset_platforms/images/5/small/solana.png?1706606708",
      "large": "https://coin-images.coingecko.com/asset_platforms/images/5/large/solana.png?1706606708"
    }
  }
]`)

var responseCoinMarketChartData = []byte(`{
  "prices": [
    [1737889461591, 0.0418141368281291],
    [1737893063477, 0.0420109710834123],
    [1737896662050, 0.041708338681602],
    [1737900535364, 0.0416724599435818],
    [1737903862165, 0.0417678066849023],
    [1737907469522, 0.0417437321514032],
    [1737911063016, 0.0418807698299526],
    [1737914672131, 0.0419228908628703],
    [1737918272316, 0.0420323651962845],
    [1737921865908, 0.0417301536472736]
  ],
  "market_caps": [
    [1737889461591, 165591498.777258],
    [1737893063477, 166380765.753509],
    [1737896662050, 165208432.620046],
    [1737900535364, 165051114.900981],
    [1737903862165, 165452244.192101],
    [1737907469522, 165178748.801635],
    [1737911063016, 165879865.149365],
    [1737914672131, 166021384.560373],
    [1737918272316, 166368152.842159],
    [1737921865908, 165295365.42146]
  ],
  "total_volumes": [
    [1737889461591, 5917250.91059339],
    [1737893063477, 5885861.57152575],
    [1737896662050, 5906997.36666343],
    [1737900535364, 5969858.78523582],
    [1737903862165, 5960034.56002101],
    [1737907469522, 5948846.59047273],
    [1737911063016, 5960627.84261061],
    [1737914672131, 5955884.44398737],
    [1737918272316, 5898649.12347374],
    [1737921865908, 5569047.29834655]
  ]
}`)

var responseCoinsListData = []byte(`[
  {
    "id": "valobit",
    "symbol": "vbit",
    "name": "VALOBIT",
    "platforms": {
      "valobit": ""
    }
  },
  {
    "id": "dmx",
    "symbol": "dmx",
    "name": "DMX",
    "platforms": {
      "factom": "0x0ec581b1f76ee71fb9feefd058e0ecf90ebab63e"
    }
  },
  {
    "id": "pedro",
    "symbol": "pedro",
    "name": "PEDRO",
    "platforms": {
      "factom": "0x51165e8ce5d6e99c570f4601a6c8409394295065"
    }
  },
  {
    "id": "volta-club",
    "symbol": "volta",
    "name": "Volta Club",
    "platforms": {
      "ethereum": "0x9b06f3c5de42d4623d7a2bd940ec735103c68a76",
      "factom": "0x9b06f3c5de42d4623d7a2bd940ec735103c68a76",
      "avalanche": "0x9b06f3c5de42d4623d7a2bd940ec735103c68a76",
      "arbitrum-one": "0x9b06f3c5de42d4623d7a2bd940ec735103c68a76"
    }
  },
  {
    "id": "don-t-sell-your-bitcoin",
    "symbol": "bitcoin",
    "name": "DON'T SELL YOUR BITCOIN",
    "platforms": {
      "solana": "RrUiMy3j9bzhgBJpXCqpF33vfrGD5Y9qAfbBVbRMkQv"
    }
  },
  {
    "id": "harrypotterobamasonic10in",
    "symbol": "bitcoin",
    "name": "HarryPotterObamaSonic10Inu (ETH)",
    "platforms": {
      "ethereum": "0x72e4f9f808c49a2a61de9c5896298920dc4eeea9",
      "solana": "CTgiaZUK12kCcB8sosn4Nt2NZtzLgtPqDwyQyr2syATC",
      "base": "0x2a06a17cbc6d0032cac2c6696da90f29d39a1a29"
    }
  },
  {
    "id": "dork-lord-coin",
    "symbol": "dlord",
    "name": "DORK LORD COIN",
    "platforms": {
      "solana": "3krWsXrweUbpsDJ9NKiwzNJSxLQKdPJNGzeEU5MZKkrb"
    }
  },
  {
    "id": "dork-lord-eth",
    "symbol": "dorkl",
    "name": "DORK LORD (SOL)",
    "platforms": {
      "solana": "8uwcmeA46XfLUc4MJ1WFQeV81rDTHTVer1B5Rc6M4iyn"
    }
  },
  {
    "id": "dotcom",
    "symbol": "y2k",
    "name": "Dotcom",
    "platforms": {
      "solana": "8YiB8B43EwDeSx5Jp91VQjgBU4mfCgVvyNahadtzpump"
    }
  },
  {
    "id": "sentinel-bot-ai",
    "symbol": "snt",
    "name": "Sentinel Bot Ai",
    "platforms": {
      "ethereum": "0x78ba134c3ace18e69837b01703d07f0db6fb0a60"
    }
  },
  {
    "id": "status",
    "symbol": "snt",
    "name": "Status",
    "platforms": {
      "ethereum": "0x744d70fdbe2ba4cf95131626614a1763df805b9e",
      "energi": "0x6bb14afedc740dce4904b7a65807fe3b967f4c94"
    }
  },
  {
    "id": "usd-coin",
    "symbol": "usdc",
    "name": "USDC",
    "platforms": {
      "ethereum": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
      "unichain": "0x078d782b760474a361dda0af3839290b0ef57ad6",
      "zksync": "0x1d17cbcf0d6d143135ae902365d2e5e2a16538d4",
      "optimistic-ethereum": "0x0b2c639c533813f4aa9d7837caf62653d097ff85",
      "polkadot": "1337",
      "tron": "TEkxiTehnzSmSe2XqrBj4w32RUN966rdz8",
      "near-protocol": "17208628f84f5d6ad33f0da3bbbeb27ffcb398eac501a31bd6ad2011e36133a1",
      "hedera-hashgraph": "0.0.456858",
      "aptos": "0xbae207659db88bea0cbead6da0ed00aac12edcdda169e591cd41c94180b46f3b",
      "algorand": "31566704",
      "stellar": "USDC-GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN",
      "celo": "0xceba9300f2b948710d2653dd7b07f33a8b32118c",
      "sui": "0xdba34672e30cb065b1f93e3ab55318768fd6fef66c15942c9f7cb846e2f900e7::usdc::USDC",
      "avalanche": "0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e",
      "arbitrum-one": "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
      "polygon-pos": "0x3c499c542cef5e3811e1192ce70d8cc03d5c3359",
      "base": "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
      "solana": "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
    }
  },
  {
    "id": "bridged-usdt",
    "symbol": "usdt",
    "name": "Bridged USDT",
    "platforms": {
      "optimistic-ethereum": "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58",
      "kardiachain": "0x551a5dcac57c66aa010940c2dcff5da9c53aa53b",
      "metis-andromeda": "0xbb06dca3ae6887fabf931640f67cab3e3a16f4dc",
      "boba": "0x5de1677344d3cb0d7d465c10b72a8f60699c062d",
      "rollux": "0x28c9c7fb3fe3104d2116af26cc8ef7905547349c",
      "velas": "0xb44a9b6905af7c801311e8f4e76932ee959c663c",
      "iotex": "0x3cdb7c48e70b854ed2fa392e21687501d84b3afc",
      "harmony-shard-0": "0x3c2b8be99c50593081eaa2a724f0b8285f5aba8f",
      "canto": "0xd567b3d7b8fe3c79a1ad8da978812cfc4fa05e75",
      "zksync": "0x493257fd37edb34451f62edf8d2a0c418852ba4c",
      "osmosis": "ibc/4ABBEF4C8926DDDB320AE5188CFD63267ABBCEFC0583E4AE05D6E5AA2401DDAB",
      "bitgert": "0xde14b85cf78f2add2e867fee40575437d5f10c06",
      "fuse": "0xfadbbf8ce7d5b7041be672561bba99f79c532e10",
      "meter": "0x5fa41671c48e3c951afc30816947126ccc8c162e",
      "ethereumpow": "0x2ad7868ca212135c6119fd7ad1ce51cfc5702892",
      "okex-chain": "0x382bb369d343125bfb2117af9c149795c6c65c50",
      "oasys": "0xdc3af65ecbd339309ec55f109cb214e0325c5ed4",
      "milkomeda-cardano": "0x80a16016cc4a2e6a2caca8a4a498b1699ff0f844"
    }
  }
]`)

var responseCoinsMarketsData = []byte(`[
  {
    "id": "ethereum",
    "symbol": "eth",
    "name": "Ethereum",
    "image": "https://coin-images.coingecko.com/coins/images/279/large/ethereum.png?1696501628",
    "current_price": 2432.43,
    "market_cap": 292522384022,
    "market_cap_rank": 2,
    "fully_diluted_valuation": 292522384022,
    "total_volume": 40015594530,
    "high_24h": 2691.69,
    "low_24h": 2337.39,
    "price_change_24h": -240.706759028237,
    "price_change_percentage_24h": -9.00467,
    "market_cap_change_24h": -29681683595.8272,
    "market_cap_change_percentage_24h": -9.21208,
    "circulating_supply": 120573726.51257,
    "total_supply": 120573726.51257,
    "max_supply": null,
    "ath": 4878.26,
    "ath_change_percentage": -50.17725,
    "ath_date": "2021-11-10T14:24:19.604Z",
    "atl": 0.432979,
    "atl_change_percentage": 561239.97795,
    "atl_date": "2015-10-20T00:00:00.000Z",
    "roi": {
      "times": 35.4324501976974,
      "currency": "btc",
      "percentage": 3543.24501976974
    },
    "last_updated": "2025-02-25T13:09:18.886Z",
    "price_change_percentage_1h_in_currency": -0.145646222497177,
    "price_change_percentage_24h_in_currency": -9.00466887593407
  },
  {
    "id": "status",
    "symbol": "snt",
    "name": "Status",
    "image": "https://coin-images.coingecko.com/coins/images/779/large/status.png?1696501931",
    "current_price": 0.02601938,
    "market_cap": 103150993,
    "market_cap_rank": 447,
    "fully_diluted_valuation": 177233175,
    "total_volume": 7469207,
    "high_24h": 0.02934234,
    "low_24h": 0.02488445,
    "price_change_24h": -0.00310646933671397,
    "price_change_percentage_24h": -10.66568,
    "market_cap_change_24h": -12320027.3974616,
    "market_cap_change_percentage_24h": -10.66937,
    "circulating_supply": 3960483788.3097,
    "total_supply": 6804870174,
    "max_supply": null,
    "ath": 0.684918,
    "ath_change_percentage": -96.20738,
    "ath_date": "2018-01-03T00:00:00.000Z",
    "atl": 0.00592935,
    "atl_change_percentage": 338.09791,
    "atl_date": "2020-03-13T02:10:36.877Z",
    "roi": null,
    "last_updated": "2025-02-25T13:09:24.088Z",
    "price_change_percentage_1h_in_currency": -0.253535324764873,
    "price_change_percentage_24h_in_currency": -10.6656782658222
  }
]`)

var responseSimplePriceData = []byte(`{
  "ethereum": {
    "usd": 2419.65,
    "eur": 2304.62
  },
  "status": {
    "usd": 0.02597611,
    "eur": 0.02474128
  }
}`)
