package currency

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"

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

	tokenKey1 := fmt.Sprintf("1-%s", common.HexToAddress("0x1").Hex())
	tokenKey2 := fmt.Sprintf("1-%s", common.HexToAddress("0x2").Hex())
	tokenKey3 := fmt.Sprintf("1-%s", common.HexToAddress("0x3").Hex())

	pr1 := FormatPerKey{
		"A": {
			Key:                 "A",
			Symbol:              "A",
			DisplayDecimals:     1,
			StripTrailingZeroes: false,
		},
		"B": {
			Key:                 "B",
			Symbol:              "B",
			DisplayDecimals:     2,
			StripTrailingZeroes: true,
		},
		tokenKey1: {
			Key:                 tokenKey1,
			Symbol:              "TOKEN1",
			DisplayDecimals:     3,
			StripTrailingZeroes: true,
		},
		tokenKey2: {
			Key:                 tokenKey2,
			Symbol:              "TOKEN2",
			DisplayDecimals:     4,
			StripTrailingZeroes: true,
		},
	}

	err = db.UpdateCachedFormats(pr1)
	require.NoError(t, err)

	rst, err = db.GetCachedFormats()
	require.NoError(t, err)
	require.Equal(t, rst, pr1)

	pr2 := FormatPerKey{
		"B": {
			Key:                 "B",
			Symbol:              "B",
			DisplayDecimals:     3,
			StripTrailingZeroes: true,
		},
		"C": {
			Key:                 "C",
			Symbol:              "C",
			DisplayDecimals:     4,
			StripTrailingZeroes: false,
		},
		tokenKey2: {
			Key:                 tokenKey2,
			Symbol:              "TOKEN2",
			DisplayDecimals:     5,
			StripTrailingZeroes: true,
		},
		tokenKey3: {
			Key:                 tokenKey3,
			Symbol:              "TOKEN3",
			DisplayDecimals:     6,
			StripTrailingZeroes: true,
		},
	}

	err = db.UpdateCachedFormats(pr2)
	require.NoError(t, err)

	rst, err = db.GetCachedFormats()
	require.NoError(t, err)

	expected := FormatPerKey{
		"A": {
			Key:                 "A",
			Symbol:              "A",
			DisplayDecimals:     1,
			StripTrailingZeroes: false,
		},
		"B": {
			Key:                 "B",
			Symbol:              "B",
			DisplayDecimals:     3,
			StripTrailingZeroes: true,
		},
		"C": {
			Key:                 "C",
			Symbol:              "C",
			DisplayDecimals:     4,
			StripTrailingZeroes: false,
		},
		tokenKey1: {
			Key:                 tokenKey1,
			Symbol:              "TOKEN1",
			DisplayDecimals:     3,
			StripTrailingZeroes: true,
		},
		tokenKey2: {
			Key:                 tokenKey2,
			Symbol:              "TOKEN2",
			DisplayDecimals:     5,
			StripTrailingZeroes: true,
		},
		tokenKey3: {
			Key:                 tokenKey3,
			Symbol:              "TOKEN3",
			DisplayDecimals:     6,
			StripTrailingZeroes: true,
		},
	}

	require.Equal(t, rst, expected)
}
