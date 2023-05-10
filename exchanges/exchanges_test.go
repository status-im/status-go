package exchanges

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
)

func TestNullAddress(t *testing.T) {
	address := common.HexToAddress("0x0")
	exchange := GetCentralizedExchangeWithAddress(address)

	require.Empty(t, exchange)
}

func TestBinanceWithCode(t *testing.T) {
	exchange, err := GetCentralizedExchangeWithCode("binance")

	require.NoError(t, err)
	require.NotEmpty(t, exchange)
	require.Equal(t, exchange.Name(), "Binance")
}

func TestBinanceWithAddress(t *testing.T) {
	// Address "Binance 3"
	// https://etherscan.io/address/0x564286362092d8e7936f0549571a803b203aaced
	address := common.HexToAddress("0x564286362092D8e7936f0549571a803B203aAceD")
	exchange := GetCentralizedExchangeWithAddress(address)

	require.NotEmpty(t, exchange)
	require.Equal(t, exchange.Name(), "Binance")
}

func TestKrakenWithAddress(t *testing.T) {
	// Address "Kraken 4"
	// https://etherscan.io/address/0x267be1c1d684f78cb4f6a176c4911b741e4ffdc0
	address := common.HexToAddress("0x267be1C1D684F78cb4F6a176C4911b741E4Ffdc0")
	exchange := GetCentralizedExchangeWithAddress(address)

	require.NotEmpty(t, exchange)
	require.Equal(t, exchange.Name(), "Kraken")
}
