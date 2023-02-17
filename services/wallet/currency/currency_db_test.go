package currency

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
)

func setupTestCurrencyDB(t *testing.T) (*DB, func()) {
	db, err := appdatabase.InitializeDB(":memory:", "wallet-currency-db-tests-", 1)
	require.NoError(t, err)
	return NewCurrencyDB(db), func() {
		require.NoError(t, db.Close())
	}
}

func TestCurrencyFormats(t *testing.T) {
	db, stop := setupTestCurrencyDB(t)
	defer stop()

	rst, err := db.GetCachedFormats()
	require.NoError(t, err)
	require.Empty(t, rst)

	pr1 := FormatPerSymbol{
		"A": {
			Symbol:              "A",
			DisplayDecimals:     1,
			StripTrailingZeroes: false,
		},
		"B": {
			Symbol:              "B",
			DisplayDecimals:     2,
			StripTrailingZeroes: true,
		},
	}

	err = db.UpdateCachedFormats(pr1)
	require.NoError(t, err)

	rst, err = db.GetCachedFormats()
	require.NoError(t, err)
	require.Equal(t, rst, pr1)

	pr2 := FormatPerSymbol{
		"B": {
			Symbol:              "B",
			DisplayDecimals:     3,
			StripTrailingZeroes: true,
		},
		"C": {
			Symbol:              "C",
			DisplayDecimals:     4,
			StripTrailingZeroes: false,
		},
	}

	err = db.UpdateCachedFormats(pr2)
	require.NoError(t, err)

	rst, err = db.GetCachedFormats()
	require.NoError(t, err)

	expected := FormatPerSymbol{
		"A": {
			Symbol:              "A",
			DisplayDecimals:     1,
			StripTrailingZeroes: false,
		},
		"B": {
			Symbol:              "B",
			DisplayDecimals:     3,
			StripTrailingZeroes: true,
		},
		"C": {
			Symbol:              "C",
			DisplayDecimals:     4,
			StripTrailingZeroes: false,
		},
	}

	require.Equal(t, rst, expected)
}
