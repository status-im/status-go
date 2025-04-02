package defaulttokenlists

import (
	"time"

	"github.com/status-im/status-go/services/wallet/token/token-lists/fetcher"
)

var StatusTokenList = fetcher.FetchedTokenList{
	TokenList: fetcher.TokenList{
		ID:        "status",
		SourceURL: "https://github.com/status-im/status-go/blob/develop/services/wallet/token/token-lists/default-lists/status.go",
	},
	Fetched: time.Unix(1742471186, 0),
	JsonData: `
	{
  "name": "Status Token List",
  "timestamp": "2025-04-02T10:10:04.000Z",
  "version": {
    "major": 1,
    "minor": 1,
    "patch": 0
  },
  "tags": {},
  "keywords": [
    "status",
    "default"
  ],
  "tokens": [
    {
      "id":"status",
      "address": "0xE452027cdEF746c7Cd3DB31CB700428b16cD8E51",
      "name": "Status Test Token",
      "symbol": "STT",
      "decimals": 18,
      "chainId": 11155111
    },
    {
      "id":"status",
      "address": "0xfDB3b57944943a7724fCc0520eE2B10659969a06",
      "name": "Status Test Token",
      "symbol": "STT",
      "decimals": 18,
      "chainId": 84532
    },
    {
      "id":"status",
      "address": "0x1C3Ac2a186c6149Ae7Cb4D716eBbD0766E4f898a",
      "name": "Status Test Token",
      "symbol": "STT",
      "decimals": 18,
      "chainId": 1660990954
    },
    {
      "id":"status",
      "address": "0x5fbdb2315678afecb367f032d93f642f64180aa3",
      "name": "Status",
      "symbol": "SNT",
      "decimals": 18,
      "chainId": 31337
    },
    {
      "id":"dai",
      "address": "0x3e622317f8c93f7328350cf0b56d9ed4c620c5d6",
      "name": "Dai Stablecoin",
      "symbol": "DAI",
      "decimals": 18,
      "chainId": 11155111
    },
    {
      "id":"weenus",
      "address": "0x7439E9Bb6D8a84dd3A23fe621A30F95403F87fB9",
      "name": "WEENUS Token",
      "symbol": "WEENUS",
      "decimals": 18,
      "chainId": 11155111
    },
    {
      "id":"xeenus",
      "address": "0xc21d97673B9E0B3AA53a06439F71fDc1facE393B",
      "name": "XEENUS Token",
      "symbol": "XEENUS",
      "decimals": 18,
      "chainId": 11155111
    },
    {
      "id":"yeenus",
      "address": "0x93fCA4c6E2525C09c95269055B46f16b1459BF9d",
      "name": "YEENUS Token",
      "symbol": "YEENUS",
      "decimals": 8,
      "chainId": 11155111
    },
    {
      "id":"zeenus",
      "address": "0xe9EF74A6568E9f0e42a587C9363C9BcC582dcC6c",
      "name": "ZEENUS Token",
      "symbol": "ZEENUS",
      "decimals": 0,
      "chainId": 11155111
    },
    {
      "id":"weth9",
      "address": "0x07391dbE03e7a0DEa0fce6699500da081537B6c3",
      "name": "WETH9 Token",
      "symbol": "WETH9",
      "decimals": 18,
      "chainId": 11155111
    },
    {
      "id":"euro-coin",
      "address": "0x08210F9170F89Ab7658F0B5E3fF39b0E03C594D4",
      "name": "Euro Coin",
      "symbol": "EURC",
      "decimals": 6,
      "chainId": 11155111
    },
    {
      "id":"euro-coin",
      "address": "0x808456652fdb597867f38412077A9182bf77359F",
      "name": "Euro Coin",
      "symbol": "EURC",
      "decimals": 6,
      "chainId": 84532
    },
    {
      "id":"usd-coin",
      "address": "0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238",
      "name": "USD Coin",
      "symbol": "USDC",
      "decimals": 6,
      "chainId": 11155111
    },
    {
      "id":"usd-coin",
      "address": "0x75faf114eafb1BDbe2F0316DF893fd58CE46AA4d",
      "name": "USD Coin",
      "symbol": "USDC",
      "decimals": 6,
      "chainId": 421614
    },
    {
      "id":"usd-coin",
      "address": "0x5fd84259d66Cd46123540766Be93DFE6D43130D7",
      "name": "USD Coin",
      "symbol": "USDC",
      "decimals": 6,
      "chainId": 11155420
    },
    {
      "id":"usd-coin",
      "address": "0x036CbD53842c5426634e7929541eC2318f3dCF7e",
      "name": "USD Coin",
      "symbol": "USDC",
      "decimals": 6,
      "chainId": 84532
    },
    {
      "id":"usd-coin",
      "address": "0xc445a18CA49190578DaD62Fba3048C07Efc07ffe",
      "name": "USD Coin",
      "symbol": "USDC",
      "decimals": 6,
      "chainId": 1660990954
    },
    {
      "id":"euro-coin",
      "address": "0xFe8bE27656b1508194D9302d12A940B4d7c35B99",
      "name": "Euro Coin",
      "symbol": "EURC",
      "decimals": 6,
      "chainId": 1660990954
    }
  ]
}
	`,
}
