package mercuryo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshallCurrencies(t *testing.T) {
	requiredAssetIDs := []CryptoCurrency{
		{
			Network:  "ETHEREUM",
			Symbol:   "ETH",
			Contract: "",
		},
		{
			Network:  "OPTIMISM",
			Symbol:   "ETH",
			Contract: "",
		},
		{
			Network:  "ARBITRUM",
			Symbol:   "ETH",
			Contract: "",
		},
		{
			Network:  "ETHEREUM",
			Symbol:   "DAI",
			Contract: "0x6b175474e89094c44da98b954eedeac495271d0f",
		},
	}

	currencies, err := handleCurrenciesResponse(getTestCurrenciesOKResponse())
	assert.NoError(t, err)
	assert.Subset(t, currencies, requiredAssetIDs)
}
