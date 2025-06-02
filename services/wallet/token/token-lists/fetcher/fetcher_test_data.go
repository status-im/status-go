package fetcher

import "fmt"

const serverURLPlaceholder = "SERVER-URL"

const UniswapTokensListVersion = "100.101.102"
const UniswapSpecialTokenName = "TEST UNISWAP TOKEN" // #nosec
const UniswapSpecialTokenSymbol = "TUT"

const AaveTokensListVersion = "300.301.302"
const AaveSpecialTokenName = "TEST AAVE TOKEN" // #nosec
const AaveSpecialTokenSymbol = "TAT"

// #nosec G101
const listOfTokenListsJsonResponse = `[
  {
    "id": "uniswap",
    "sourceUrl": "SERVER-URL/uniswap.json"
  },
  {
    "id": "aave",
    "sourceUrl": "SERVER-URL/aave.json"
  }
]`

var listOfTokenLists = []TokenList{
	{
		ID:        "uniswap",
		SourceURL: fmt.Sprintf("%s/uniswap.json", serverURLPlaceholder),
	},
	{
		ID:        "aave",
		SourceURL: fmt.Sprintf("%s/aave.json", serverURLPlaceholder),
	},
}

var defaultTokensList = []TokenList{
	{
		ID:        "uniswap",
		SourceURL: "https://ipfs.io/ipns/tokens.uniswap.org",
		Schema:    "https://uniswap.org/tokenlist.schema.json",
	},
	{
		ID:        "aave",
		SourceURL: "https://raw.githubusercontent.com/bgd-labs/aave-address-book/main/tokenlist.json",
	},
}

// #nosec G101
const uniswapTokenListJsonResponse = `{
  "name": "Uniswap Labs Default",
  "timestamp": "2025-03-01T00:09:57.673Z",
  "version": {
    "major": 100,
    "minor": 101,
    "patch": 102
  },
  "tags": {},
  "logoURI": "ipfs://QmNa8mQkrNKp1WEEeGjFezDmDeodkWRevGFN8JCV7b4Xir",
  "keywords": [
    "uniswap",
    "default"
  ],
  "tokens": [
    {
      "chainId": 1,
      "address": "0x111111111117dC0aa78b770fA6A738034120C302",
      "name": "1inch",
      "symbol": "1INCH",
      "decimals": 18,
      "logoURI": "https://assets.coingecko.com/coins/images/13469/thumb/1inch-token.png?1608803028",
      "extensions": {
        "bridgeInfo": {
          "10": {
            "tokenAddress": "0xAd42D013ac31486B73b6b059e748172994736426"
          },
          "56": {
            "tokenAddress": "0x111111111117dC0aa78b770fA6A738034120C302"
          },
          "130": {
            "tokenAddress": "0xbe41cde1C5e75a7b6c2c70466629878aa9ACd06E"
          },
          "137": {
            "tokenAddress": "0x9c2C5fd7b07E95EE044DDeba0E97a665F142394f"
          },
          "8453": {
            "tokenAddress": "0xc5fecC3a29Fb57B5024eEc8a2239d4621e111CBE"
          },
          "42161": {
            "tokenAddress": "0x6314C31A7a1652cE482cffe247E9CB7c3f4BB9aF"
          },
          "43114": {
            "tokenAddress": "0xd501281565bf7789224523144Fe5D98e8B28f267"
          }
        }
      }
    },
    {
      "chainId": 1,
      "address": "0x3E5A19c91266aD8cE2477B91585d1856B84062dF",
      "name": "Ancient8",
      "symbol": "A8",
      "decimals": 18,
      "logoURI": "https://assets.coingecko.com/coins/images/39170/standard/A8_Token-04_200x200.png?1720798300",
      "extensions": {
        "bridgeInfo": {
          "130": {
            "tokenAddress": "0x44D618C366D7bC85945Bfc922ACad5B1feF7759A"
          }
        }
      }
    },
    {
      "chainId": 1,
      "address": "0x7Fc66500c84A76Ad7e9c93437bFc5Ac33E2DDaE9",
      "name": "Aave",
      "symbol": "AAVE",
      "decimals": 18,
      "logoURI": "https://assets.coingecko.com/coins/images/12645/thumb/AAVE.png?1601374110",
      "extensions": {
        "bridgeInfo": {
          "10": {
            "tokenAddress": "0x76FB31fb4af56892A25e32cFC43De717950c9278"
          },
          "56": {
            "tokenAddress": "0xfb6115445Bff7b52FeB98650C87f44907E58f802"
          },
          "130": {
            "tokenAddress": "0x02a24C380dA560E4032Dc6671d8164cfbEEAAE1e"
          },
          "137": {
            "tokenAddress": "0xD6DF932A45C0f255f85145f286eA0b292B21C90B"
          },
          "8453": {
            "tokenAddress": "0x63706e401c06ac8513145b7687A14804d17f814b"
          },
          "42161": {
            "tokenAddress": "0xba5DdD1f9d7F570dc94a51479a000E3BCE967196"
          },
          "43114": {
            "tokenAddress": "0x63a72806098Bd3D9520cC43356dD78afe5D386D9"
          }
        }
      }
    },
    {
      "chainId": 1,
      "address": "0x744d70FDBE2Ba4CF95131626614a1763DF805B9E",
      "name": "Status",
      "symbol": "SNT",
      "decimals": 18,
      "logoURI": "https://assets.coingecko.com/coins/images/779/thumb/status.png?1548610778",
      "extensions": {
        "bridgeInfo": {
          "10": {
            "tokenAddress": "0x650AF3C15AF43dcB218406d30784416D64Cfb6B2"
          },
          "130": {
            "tokenAddress": "0x914f7CE2B080B2186159C2213B1e193E265aBF5F"
          },
          "8453": {
            "tokenAddress": "0x662015EC830DF08C0FC45896FaB726542e8AC09E"
          },
          "42161": {
            "tokenAddress": "0x707F635951193dDaFBB40971a0fCAAb8A6415160"
          }
        }
      }
    },
		{
      "chainId": 10,
      "address": "0x650AF3C15AF43dcB218406d30784416D64Cfb6B2",
      "name": "Status",
      "symbol": "SNT",
      "decimals": 18,
      "logoURI": "https://assets.coingecko.com/coins/images/779/thumb/status.png?1548610778",
      "extensions": {
        "bridgeInfo": {
          "1": {
            "tokenAddress": "0x744d70FDBE2Ba4CF95131626614a1763DF805B9E"
          }
        }
      }
    },
		{
      "chainId": 8453,
      "address": "0x662015EC830DF08C0FC45896FaB726542e8AC09E",
      "name": "Status",
      "symbol": "SNT",
      "decimals": 18,
      "logoURI": "https://assets.coingecko.com/coins/images/779/thumb/status.png?1548610778",
      "extensions": {
        "bridgeInfo": {
          "1": {
            "tokenAddress": "0x744d70FDBE2Ba4CF95131626614a1763DF805B9E"
          }
        }
      }
    },
		{
      "name": "USDCoin",
      "address": "0x0b2C639c533813f4Aa9D7837CAf62653d097Ff85",
      "symbol": "USDC",
      "decimals": 6,
      "chainId": 10,
      "logoURI": "https://ethereum-optimism.github.io/data/USDC/logo.png",
      "extensions": {
        "bridgeInfo": {
          "1": {
            "tokenAddress": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
          }
        }
      }
    },
		{
      "name": "USDCoin",
      "address": "0xaf88d065e77c8cC2239327C5EDb3A432268e5831",
      "symbol": "USDC",
      "decimals": 6,
      "chainId": 42161,
      "logoURI": "https://raw.githubusercontent.com/trustwallet/assets/master/blockchains/ethereum/assets/0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48/logo.png",
      "extensions": {
        "bridgeInfo": {
          "1": {
            "tokenAddress": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
          }
        }
      }
    },
		{
      "name": "Wrapped Ether",
      "address": "0xA6FA4fB5f76172d178d61B04b0ecd319C5d1C0aa",
      "symbol": "WETH",
      "decimals": 18,
      "chainId": 80001,
      "logoURI": "https://raw.githubusercontent.com/trustwallet/assets/master/blockchains/ethereum/assets/0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2/logo.png"
    },
    {
      "name": "Wrapped Matic",
      "address": "0x9c3C9283D3e44854697Cd22D3Faa240Cfb032889",
      "symbol": "WMATIC",
      "decimals": 18,
      "chainId": 80001,
      "logoURI": "https://assets.coingecko.com/coins/images/4713/thumb/matic-token-icon.png?1624446912"
    },
    {
      "chainId": 81457,
      "address": "0xb1a5700fA2358173Fe465e6eA4Ff52E36e88E2ad",
      "name": "Blast",
      "symbol": "BLAST",
      "decimals": 18,
      "logoURI": "https://assets.coingecko.com/coins/images/35494/standard/Blast.jpg?1719385662"
    },
    {
      "chainId": 7777777,
      "address": "0xCccCCccc7021b32EBb4e8C08314bD62F7c653EC4",
      "name": "USD Coin (Bridged from Ethereum)",
      "symbol": "USDzC",
      "decimals": 6,
      "logoURI": "https://assets.coingecko.com/coins/images/35218/large/USDC_Icon.png?1707908537"
    },
    {
      "name": "Uniswap",
      "address": "0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984",
      "symbol": "UNI",
      "decimals": 18,
      "chainId": 11155111,
      "logoURI": "ipfs://QmXttGpZrECX5qCyXbBQiqgQNytVGeZW5Anewvh2jc4psg"
    },
    {
      "name": "Wrapped Ether",
      "address": "0xfFf9976782d46CC05630D1f6eBAb18b2324d6B14",
      "symbol": "WETH",
      "decimals": 18,
      "chainId": 11155111,
      "logoURI": "https://raw.githubusercontent.com/trustwallet/assets/master/blockchains/ethereum/assets/0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2/logo.png"
    },
		{
      "name": "TEST UNISWAP TOKEN",
      "address": "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "symbol": "TUT",
      "decimals": 18,
      "chainId": 1
    },
    {
      "name": "Test Token 1",
      "address": "0x0000000000000000000000000000000000053211",
      "symbol": "TXX",
      "decimals": 18,
      "chainId": 777333
    },
    {
      "name": "Test Token 2",
      "address": "0x0000000000000000000000000000000000073211",
      "symbol": "TXY",
      "decimals": 18,
      "chainId": 777333
    }
  ]
}`

// #nosec G101
const aaveTokenListJsonResponse = `{
  "name": "Aave token list",
  "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aave.svg",
  "keywords": ["audited", "verified", "aave"],
  "tags": {
    "underlying": {
      "name": "underlyingAsset",
      "description": "Tokens that are used as underlying assets in the Aave protocol"
    },
    "aaveV2": { "name": "Aave V2", "description": "Tokens related to aave v2" },
    "aaveV3": { "name": "Aave V3", "description": "Tokens related to aave v3" },
    "aTokenV2": {
      "name": "aToken V2",
      "description": "Tokens that earn interest on the Aave Protocol V2"
    },
    "aTokenV3": {
      "name": "aToken V3",
      "description": "Tokens that earn interest on the Aave Protocol V3"
    },
    "stataToken": {
      "name": "stata token",
      "description": "Tokens that are wrapped into a 4626 Vault"
    },
    "staticAT": {
      "name": "static a token",
      "description": "Tokens that are wrapped into a 4626 Vault"
    }
  },
  "tokens": [
    {
      "chainId": 1,
      "address": "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
      "name": "Wrapped Ether",
      "decimals": 18,
      "symbol": "WETH",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/weth.svg"
    },
    {
      "chainId": 1,
      "address": "0xf9Fb4AD91812b704Ba883B11d2B576E890a6730A",
      "name": "Aave AMM Market WETH",
      "decimals": 18,
      "symbol": "aAmmWETH",
      "tags": ["aTokenV2", "aaveV2"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/aweth.svg",
      "extensions": {
        "pool": "0x7937D4799803FbBe595ed57278Bc4cA21f3bFfCB",
        "underlying": "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
      }
    },
    {
      "chainId": 1,
      "address": "0x6B175474E89094C44Da98b954EedeAC495271d0F",
      "name": "Dai Stablecoin",
      "decimals": 18,
      "symbol": "DAI",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/dai.svg"
    },
		{
      "chainId": 1,
      "address": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
      "name": "USD Coin",
      "decimals": 6,
      "symbol": "USDC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdc.svg"
    },
		{
      "chainId": 10,
      "address": "0x7F5c764cBc14f9669B88837ca1490cCa17c31607",
      "name": "USD Coin",
      "decimals": 6,
      "symbol": "USDC",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdc.svg"
    },
		{
      "chainId": 146,
      "address": "0x29219dd400f2Bf60E5a23d13Be72B486D4038894",
      "name": "Bridged USDC (Sonic Labs)",
      "decimals": 6,
      "symbol": "USDCe",
      "tags": ["underlying"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/usdc.svg"
    },
    {
      "chainId": 146,
      "address": "0x578Ee1ca3a8E1b54554Da1Bf7C583506C4CD11c6",
      "name": "Aave Sonic USDC",
      "decimals": 6,
      "symbol": "aSonUSDC",
      "tags": ["aTokenV3", "aaveV3"],
      "logoURI": "https://raw.githubusercontent.com/bgd-labs/web3-icons/main/icons/full/ausdc.svg",
      "extensions": {
        "pool": "0x5362dBb1e601abF3a4c14c22ffEdA64042E5eAA3",
        "underlying": "0x29219dd400f2Bf60E5a23d13Be72B486D4038894"
      }
    },
    {
      "chainId": 146,
      "address": "0x039e2fB66102314Ce7b64Ce5Ce3E5183bc94aD38",
      "name": "Wrapped Sonic",
      "decimals": 18,
      "symbol": "wS",
      "tags": ["underlying"]
    },
    {
      "chainId": 146,
      "address": "0x6C5E14A212c1C3e4Baf6f871ac9B1a969918c131",
      "name": "Aave Sonic wS",
      "decimals": 18,
      "symbol": "aSonwS",
      "tags": ["aTokenV3", "aaveV3"],
      "extensions": {
        "pool": "0x5362dBb1e601abF3a4c14c22ffEdA64042E5eAA3",
        "underlying": "0x039e2fB66102314Ce7b64Ce5Ce3E5183bc94aD38"
      }
    },
		{
      "name": "TEST AAVE TOKEN",
      "address": "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
      "symbol": "TAT",
      "decimals": 18,
      "chainId": 1
    }
  ],
  "version": { "major": 300, "minor": 301, "patch": 302 },
  "timestamp": "2025-03-03T19:49:19.213Z"
}`
