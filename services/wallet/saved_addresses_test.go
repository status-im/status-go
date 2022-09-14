package wallet

import (
	"io/ioutil"
	"os"
	"strconv"
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

	rst, err := manager.GetSavedAddressesForChainID(777)
	require.NoError(t, err)
	require.Nil(t, rst)

	sa := SavedAddress{
		Address:   common.Address{1},
		Name:      "Zilliqa",
		Favourite: true,
		ChainID:   777,
	}

	_, err = manager.UpdateMetadataAndUpsertSavedAddress(sa)
	require.NoError(t, err)

	rst, err = manager.GetSavedAddressesForChainID(777)
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
	require.Equal(t, sa.Address, rst[0].Address)
	require.Equal(t, sa.Name, rst[0].Name)
	require.Equal(t, sa.Favourite, rst[0].Favourite)
	require.Equal(t, sa.ChainID, rst[0].ChainID)

	_, err = manager.DeleteSavedAddress(777, sa.Address)
	require.NoError(t, err)

	rst, err = manager.GetSavedAddressesForChainID(777)
	require.NoError(t, err)
	require.Equal(t, 0, len(rst))
}

func contains[T comparable](container []T, element T, isEqual func(T, T) bool) bool {
	for _, e := range container {
		if isEqual(e, element) {
			return true
		}
	}
	return false
}

func haveSameElements[T comparable](a []T, b []T, isEqual func(T, T) bool) bool {
	for _, v := range a {
		if !contains(b, v, isEqual) {
			return false
		}
	}
	return true
}

func savedAddressDataIsEqual(a, b SavedAddress) bool {
	return a.Address == b.Address && a.ChainID == b.ChainID && a.Name == b.Name && a.Favourite == b.Favourite
}

func TestSavedAddressesMetadata(t *testing.T) {
	manager, stop := setupTestSavedAddressesDB(t)
	defer stop()

	savedAddresses, err := manager.GetRawSavedAddresses()
	require.NoError(t, err)
	require.Nil(t, savedAddresses)

	// Add raw saved addresses
	sa1 := SavedAddress{
		Address:   common.Address{1},
		ChainID:   777,
		Name:      "Raw",
		Favourite: true,
		savedAddressMeta: savedAddressMeta{
			Removed:     false,
			UpdateClock: 234,
		},
	}

	err = manager.upsertSavedAddress(sa1, nil)
	require.NoError(t, err)

	dbSavedAddresses, err := manager.GetRawSavedAddresses()
	require.NoError(t, err)
	require.Equal(t, 1, len(dbSavedAddresses))
	require.Equal(t, sa1, dbSavedAddresses[0])

	// Add simple saved address without sync metadata
	sa2 := SavedAddress{
		Address:   common.Address{2},
		ChainID:   777,
		Name:      "Simple",
		Favourite: false,
	}

	var sa2UpdatedClock uint64
	sa2UpdatedClock, err = manager.UpdateMetadataAndUpsertSavedAddress(sa2)
	require.NoError(t, err)

	dbSavedAddresses, err = manager.GetRawSavedAddresses()
	require.NoError(t, err)
	require.Equal(t, 2, len(dbSavedAddresses))
	// The order is not guaranteed check raw entry to decide
	rawIndex := 0
	simpleIndex := 1
	if dbSavedAddresses[0] != sa1 {
		rawIndex = 1
		simpleIndex = 0
	}
	require.Equal(t, sa1, dbSavedAddresses[rawIndex])
	require.Equal(t, sa2.Address, dbSavedAddresses[simpleIndex].Address)
	require.Equal(t, sa2.ChainID, dbSavedAddresses[simpleIndex].ChainID)
	require.Equal(t, sa2.Name, dbSavedAddresses[simpleIndex].Name)
	require.Equal(t, sa2.Favourite, dbSavedAddresses[simpleIndex].Favourite)

	// Check the default values
	require.False(t, dbSavedAddresses[simpleIndex].Removed)
	require.Equal(t, dbSavedAddresses[simpleIndex].UpdateClock, sa2UpdatedClock)
	require.Greater(t, dbSavedAddresses[simpleIndex].UpdateClock, uint64(0))

	sa2Older := sa2
	sa2Older.Name = "Conditional, NOT updated"
	sa2Older.Favourite = true

	sa2Newer := sa2
	sa2Newer.Name = "Conditional, updated"
	sa2Newer.Favourite = false

	// Try to add an older entry
	updated := false
	updated, err = manager.AddSavedAddressIfNewerUpdate(sa2Older, dbSavedAddresses[simpleIndex].UpdateClock-1)
	require.NoError(t, err)
	require.False(t, updated)

	dbSavedAddresses, err = manager.GetRawSavedAddresses()
	require.NoError(t, err)

	rawIndex = 0
	simpleIndex = 1
	if dbSavedAddresses[0] != sa1 {
		rawIndex = 1
		simpleIndex = 0
	}

	require.Equal(t, 2, len(dbSavedAddresses))
	require.True(t, haveSameElements([]SavedAddress{sa1, sa2}, dbSavedAddresses, savedAddressDataIsEqual))
	require.Equal(t, sa1.savedAddressMeta, dbSavedAddresses[rawIndex].savedAddressMeta)

	// Try to update sa2 with a newer entry
	updatedClock := dbSavedAddresses[simpleIndex].UpdateClock + 1
	updated, err = manager.AddSavedAddressIfNewerUpdate(sa2Newer, updatedClock)
	require.NoError(t, err)
	require.True(t, updated)

	dbSavedAddresses, err = manager.GetRawSavedAddresses()
	require.NoError(t, err)

	simpleIndex = 1
	if dbSavedAddresses[0] != sa1 {
		simpleIndex = 0
	}

	require.Equal(t, 2, len(dbSavedAddresses))
	require.True(t, haveSameElements([]SavedAddress{sa1, sa2Newer}, dbSavedAddresses, savedAddressDataIsEqual))
	require.Equal(t, updatedClock, dbSavedAddresses[simpleIndex].UpdateClock)

	// Try to delete the sa2 newer entry
	updatedDeleteClock := updatedClock + 1
	updated, err = manager.DeleteSavedAddressIfNewerUpdate(sa2Newer.ChainID, sa2Newer.Address, updatedDeleteClock)
	require.NoError(t, err)
	require.True(t, updated)

	dbSavedAddresses, err = manager.GetRawSavedAddresses()
	require.NoError(t, err)

	simpleIndex = 1
	if dbSavedAddresses[0] != sa1 {
		simpleIndex = 0
	}

	require.Equal(t, 2, len(dbSavedAddresses))
	require.True(t, dbSavedAddresses[simpleIndex].Removed)

	// Check that deleted entry is not returned with the regular API (non-raw)
	dbSavedAddresses, err = manager.GetSavedAddresses()
	require.NoError(t, err)
	require.Equal(t, 1, len(dbSavedAddresses))
}

func TestSavedAddressesCleanSoftDeletes(t *testing.T) {
	manager, stop := setupTestSavedAddressesDB(t)
	defer stop()

	firstTimestamp := 10
	for i := 0; i < 5; i++ {
		sa := SavedAddress{
			Address:   common.Address{byte(i)},
			ChainID:   777,
			Name:      "Test" + strconv.Itoa(i),
			Favourite: false,
			savedAddressMeta: savedAddressMeta{
				Removed:     true,
				UpdateClock: uint64(firstTimestamp + i),
			},
		}

		err := manager.upsertSavedAddress(sa, nil)
		require.NoError(t, err)
	}

	err := manager.DeleteSoftRemovedSavedAddresses(uint64(firstTimestamp + 3))
	require.NoError(t, err)

	dbSavedAddresses, err := manager.GetRawSavedAddresses()
	require.NoError(t, err)
	require.Equal(t, 2, len(dbSavedAddresses))
	require.True(t, haveSameElements([]uint64{dbSavedAddresses[0].UpdateClock,
		dbSavedAddresses[1].UpdateClock}, []uint64{uint64(firstTimestamp + 3), uint64(firstTimestamp + 4)},
		func(a, b uint64) bool {
			return a == b
		},
	))
}
