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
