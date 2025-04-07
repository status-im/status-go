package currency

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
)

func setupTestCurrencyDB(t *testing.T) (*DB, func()) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
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

	pr1 := Formats{
		"A": {
			ID:                  "A",
			DisplayDecimals:     1,
			StripTrailingZeroes: false,
		},
		"B": {
			ID:                  "B",
			DisplayDecimals:     2,
			StripTrailingZeroes: true,
		},
	}

	err = db.UpdateCachedFormats(pr1)
	require.NoError(t, err)

	rst, err = db.GetCachedFormats()
	require.NoError(t, err)
	require.Equal(t, rst, pr1)

	pr2 := Formats{
		"B": {
			ID:                  "B",
			DisplayDecimals:     3,
			StripTrailingZeroes: true,
		},
		"C": {
			ID:                  "C",
			DisplayDecimals:     4,
			StripTrailingZeroes: false,
		},
	}

	err = db.UpdateCachedFormats(pr2)
	require.NoError(t, err)

	rst, err = db.GetCachedFormats()
	require.NoError(t, err)

	expected := Formats{
		"A": {
			ID:                  "A",
			DisplayDecimals:     1,
			StripTrailingZeroes: false,
		},
		"B": {
			ID:                  "B",
			DisplayDecimals:     3,
			StripTrailingZeroes: true,
		},
		"C": {
			ID:                  "C",
			DisplayDecimals:     4,
			StripTrailingZeroes: false,
		},
	}

	require.Equal(t, rst, expected)
}
