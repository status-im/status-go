package wallet

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/sqlite"
)

func setupTestSavedAddressesDB(t *testing.T) (*SavedAddressesManager, func()) {
	tmpfile, err := ioutil.TempFile("", "wallet-saved_addresses-tests-")
	require.NoError(t, err)
	db, err := appdatabase.InitializeDB(tmpfile.Name(), "wallet-saved_addresses-tests", sqlite.ReducedKDFIterationsNumber)
	require.NoError(t, err)
	return &SavedAddressesManager{db}, func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.Remove(tmpfile.Name()))
	}
}

func TestSavedAddresses(t *testing.T) {
	manager, stop := setupTestSavedAddressesDB(t)
	defer stop()

	rst, err := manager.GetSavedAddresses(777)
	require.NoError(t, err)
	require.Nil(t, rst)

	sa := SavedAddress{
		Address:   common.Address{1},
		Name:      "Zilliqa",
		Favourite: true,
		ChainID:   777,
	}

	err = manager.AddSavedAddress(sa)
	require.NoError(t, err)

	rst, err = manager.GetSavedAddresses(777)
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
	require.Equal(t, sa, rst[0])

	err = manager.DeleteSavedAddress(777, sa.Address)
	require.NoError(t, err)

	rst, err = manager.GetSavedAddresses(777)
	require.NoError(t, err)
	require.Equal(t, 0, len(rst))
}
