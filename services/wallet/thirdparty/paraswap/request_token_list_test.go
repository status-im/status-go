package paraswap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshallTokensList(t *testing.T) {

	tokens := []Token{
		{
			Symbol:   "ETH",
			Address:  "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE",
			Decimals: 18,
			Img:      "https://img.paraswap.network/ETH.png",
			Network:  1,
		},
		{
			Symbol:   "USDT",
			Address:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
			Decimals: 6,
			Img:      "https://img.paraswap.network/USDT.png",
			Network:  1,
		},
	}

	data := []byte(`{
			"tokens": [
				{
					"symbol": "ETH",
					"address": "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE",
					"decimals": 18,
					"img": "https://img.paraswap.network/ETH.png",
					"network": 1
				},
				{
					"symbol": "USDT",
					"address": "0xdac17f958d2ee523a2206206994597c13d831ec7",
					"decimals": 6,
					"img": "https://img.paraswap.network/USDT.png",
					"network": 1
				}
			]
		}`)

	receivedTokens, err := handleTokensListResponse(data)
	assert.NoError(t, err)
	assert.Equal(t, tokens, receivedTokens)
}

func TestForErrorOnFetchingTokensList(t *testing.T) {
	data := []byte(`{
		"error": "Only chainId 1 is supported"
	}`)

	_, err := handleTokensListResponse(data)
	assert.Error(t, err)
}
