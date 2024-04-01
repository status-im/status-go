package paraswap

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/services/wallet/bigint"
)

func TestUnmarshallPriceRoute(t *testing.T) {

	data := []byte(`{
			"blockNumber": 13015909,
			"network": 1,
			"srcToken": "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE",
			"srcDecimals": 18,
			"srcAmount": "1000000000000000000",
			"destToken": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
			"destDecimals": 18,
			"destAmount": "1000000000000000000",
			"bestRoute": {
				"percent": 100,
				"swaps": [
					{
						"srcToken": "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE",
						"srcDecimals": 0,
						"destToken": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
						"destDecimals": 0,
						"swapExchanges": [
							{
								"exchange": "UniswapV2",
								"srcAmount": "1000000000000000000",
								"destAmount": "1000000000000000000",
								"percent": 100,
								"data": {
									"router": "0x0000000000000000000000000000000000000000",
									"path": [
										"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
										"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
									],
									"factory": "0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f",
									"initCode": "0x96e8ac4277198ff8b6f785478aa9a39f403cb768dd02cbee326c3e7da348845f",
									"feeFactor": 10000,
									"pools": [
										{
											"address": "0xB4e16d0168e52d35CaCD2c6185b44281Ec28C9Dc",
											"fee": 30,
											"direction": false
										}
									],
									"gasUSD": "13.227195"
								}
							}
						]
					}
				]
			},
			"others": {
				"exchange": "UniswapV2",
				"srcAmount": "1000000000000000000",
				"destAmount": "3255989380",
				"unit": "3255989380",
				"data": {
					"router": "0x0000000000000000000000000000000000000000",
					"path": [
						"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
						"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
					],
					"factory": "0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f",
					"initCode": "0x96e8ac4277198ff8b6f785478aa9a39f403cb768dd02cbee326c3e7da348845f",
					"feeFactor": 10000,
					"pools": [
						{
							"address": "0xB4e16d0168e52d35CaCD2c6185b44281Ec28C9Dc",
							"fee": 30,
							"direction": false
						}
					],
					"gasUSD": "13.227195"
				}
			},
			"gasCostUSD": "11.947163",
			"gasCost": "111435",
			"side": "SELL",
			"tokenTransferProxy": "0x3e7d31751347BAacf35945074a4a4A41581B2271",
			"contractAddress": "0x485D2446711E141D2C8a94bC24BeaA5d5A110D74",
			"contractMethod": "swapOnUniswap",
			"srcUSD": "3230.3000000000",
			"destUSD": "3218.9300566052",
			"partner": "paraswap.io",
			"partnerFee": 0,
			"maxImpactReached": false,
			"hmac": "319c5cf83098a07aeebb11bed6310db51311201f"
	}`)

	responseData := []byte(fmt.Sprintf(`{"priceRoute":%s}`, string(data)))

	route := Route{
		GasCost:           &bigint.BigInt{Int: big.NewInt(111435)},
		SrcAmount:         &bigint.BigInt{Int: big.NewInt(1000000000000000000)},
		SrcTokenAddress:   common.HexToAddress("0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"),
		SrcTokenDecimals:  18,
		DestAmount:        &bigint.BigInt{Int: big.NewInt(1000000000000000000)},
		DestTokenAddress:  common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
		DestTokenDecimals: 18,
		RawPriceRoute:     data,
	}

	receivedRoute, err := handlePriceRouteResponse(responseData)
	assert.NoError(t, err)
	assert.Equal(t, route, receivedRoute)
}

func TestForErrorOnFetchingPriceRoute(t *testing.T) {
	data := []byte(`{
		"error": "Invalid tokens"
	}`)

	_, err := handlePriceRouteResponse(data)
	assert.Error(t, err)
}
