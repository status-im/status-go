package wallet

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

func setupTestPriceDB(t *testing.T) (*PriceManager, func()) {
	db, err := appdatabase.InitializeDB(":memory:", "wallet-price-tests-", 1)
	require.NoError(t, err)
	return NewPriceManager(db, thirdparty.NewCryptoCompare()), func() {
		require.NoError(t, db.Close())
	}
}

func TestPrice(t *testing.T) {
	manager, stop := setupTestPriceDB(t)
	defer stop()

	rst, err := manager.GetCachedPrices()
	require.NoError(t, err)
	require.Empty(t, rst)

	pr1 := PricesPerTokenAndCurrency{
		"ETH": {
			"USD": 1.23456,
			"EUR": 2.34567,
			"DAI": 3.45678,
		},
		"BTC": {
			"USD": 4.56789,
			"EUR": 5.67891,
			"DAI": 6.78912,
		},
	}

	err = manager.updatePriceCache(pr1)
	require.NoError(t, err)

	rst, err = manager.GetCachedPrices()
	require.NoError(t, err)
	require.Equal(t, rst, pr1)

	pr2 := PricesPerTokenAndCurrency{
		"BTC": {
			"USD": 1.23456,
			"EUR": 2.34567,
			"DAI": 3.45678,
			"ARS": 9.87654,
		},
		"ETH": {
			"USD": 4.56789,
			"EUR": 5.67891,
			"DAI": 6.78912,
			"ARS": 8.76543,
		},
		"SNT": {
			"USD": 7.654,
			"EUR": 6.0,
			"DAI": 1455.12,
			"ARS": 0.0,
		},
	}

	err = manager.updatePriceCache(pr2)
	require.NoError(t, err)

	rst, err = manager.GetCachedPrices()
	require.NoError(t, err)
	require.Equal(t, rst, pr2)
}
