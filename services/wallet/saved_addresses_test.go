package wallet

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/sqlite"
)

const (
	ensMember int = iota
	isTestMember
	addressMember
)

func setupTestSavedAddressesDB(t *testing.T) (*SavedAddressesManager, func()) {
	db, err := appdatabase.InitializeDB(sqlite.InMemoryPath, "wallet-saved_addresses-tests", sqlite.ReducedKDFIterationsNumber)
	require.NoError(t, err)

	return &SavedAddressesManager{db}, func() {
		require.NoError(t, db.Close())
	}
}

func TestSavedAddressesAdd(t *testing.T) {
	manager, stop := setupTestSavedAddressesDB(t)
	defer stop()

	rst, err := manager.GetRawSavedAddresses()
	require.NoError(t, err)
	require.Nil(t, rst)

	sa := SavedAddress{
		Address:         common.Address{1},
		Name:            "Zilliqa",
		Favourite:       true,
		ChainShortNames: "eth:arb:",
		ENSName:         "test.stateofus.eth",
		IsTest:          false,
	}

	_, err = manager.UpdateMetadataAndUpsertSavedAddress(sa)
	require.NoError(t, err)

	rst, err = manager.GetRawSavedAddresses()
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
	require.Equal(t, sa.Address, rst[0].Address)
	require.Equal(t, sa.Name, rst[0].Name)
	require.Equal(t, sa.Favourite, rst[0].Favourite)
	require.Equal(t, sa.ChainShortNames, rst[0].ChainShortNames)
	require.Equal(t, sa.ENSName, rst[0].ENSName)
	require.Equal(t, sa.IsTest, rst[0].IsTest)
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
	return a.Address == b.Address && a.Name == b.Name && a.Favourite == b.Favourite &&
		a.ChainShortNames == b.ChainShortNames && a.ENSName == b.ENSName && a.IsTest == b.IsTest
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
		Name:      "Raw",
		Favourite: true,
		savedAddressMeta: savedAddressMeta{
			Removed:     false,
			UpdateClock: 234,
		},
		ChainShortNames: "eth:arb:",
		ENSName:         "test.stateofus.eth",
		IsTest:          false,
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
		Name:      "Simple",
		Favourite: false,
		IsTest:    false,
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
	require.Equal(t, sa2.Name, dbSavedAddresses[simpleIndex].Name)
	require.Equal(t, sa2.Favourite, dbSavedAddresses[simpleIndex].Favourite)
	require.Equal(t, sa2.IsTest, dbSavedAddresses[simpleIndex].IsTest)

	// Check the default values
	require.False(t, dbSavedAddresses[simpleIndex].Removed)
	require.Equal(t, dbSavedAddresses[simpleIndex].UpdateClock, sa2UpdatedClock)
	require.Greater(t, dbSavedAddresses[simpleIndex].UpdateClock, uint64(0))

	sa2Older := sa2
	sa2Older.Favourite = true
	sa2Older.IsTest = false

	sa2Newer := sa2
	sa2Newer.Favourite = false
	sa2Newer.IsTest = false

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
	updated, err = manager.DeleteSavedAddress(sa2Newer.Address, sa2Newer.ENSName, sa2Newer.IsTest, updatedDeleteClock)
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

func TestSavedAddressesGet(t *testing.T) {
	manager, stop := setupTestSavedAddressesDB(t)
	defer stop()

	sa := SavedAddress{
		Address: common.Address{1},
		ENSName: "test.ens.eth",
		IsTest:  false,
		savedAddressMeta: savedAddressMeta{
			Removed: true,
		},
	}

	err := manager.upsertSavedAddress(sa, nil)
	require.NoError(t, err)

	dbSavedAddresses, err := manager.GetRawSavedAddresses()
	require.NoError(t, err)
	require.Equal(t, 1, len(dbSavedAddresses))

	require.True(t, savedAddressDataIsEqual(sa, dbSavedAddresses[0]))

	dbSavedAddresses, err = manager.GetSavedAddresses()
	require.NoError(t, err)
	require.Equal(t, 0, len(dbSavedAddresses))
}

func TestSavedAddressesDelete(t *testing.T) {
	manager, stop := setupTestSavedAddressesDB(t)
	defer stop()

	sa0 := SavedAddress{
		Address: common.Address{1},
		IsTest:  false,
	}

	err := manager.upsertSavedAddress(sa0, nil)
	require.NoError(t, err)

	rst, err := manager.GetRawSavedAddresses()
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))

	require.True(t, savedAddressDataIsEqual(sa0, rst[0]))

	// Modify IsTest flag, insert
	sa1 := sa0
	sa1.IsTest = !sa1.IsTest

	err = manager.upsertSavedAddress(sa1, nil)
	require.NoError(t, err)

	// Delete s0, test that only s1 is left
	updateClock := uint64(time.Now().Unix())
	_, err = manager.DeleteSavedAddress(sa0.Address, sa0.ENSName, sa0.IsTest, updateClock)
	require.NoError(t, err)

	rst, err = manager.GetSavedAddresses()
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
	require.True(t, savedAddressDataIsEqual(sa1, rst[0]))

	// Test that we still have both addresses
	rst, err = manager.GetRawSavedAddresses()
	require.NoError(t, err)
	require.Equal(t, 2, len(rst))

	// Delete s0 one more time with the same timestamp
	deleted, err := manager.DeleteSavedAddress(sa0.Address, sa0.ENSName, sa0.IsTest, updateClock)
	require.NoError(t, err)
	require.False(t, deleted)
}

func testInsertSameAddressWithOneChange(t *testing.T, member int) {
	manager, stop := setupTestSavedAddressesDB(t)
	defer stop()

	// Insert one address
	sa := SavedAddress{
		Address: common.Address{1},
		ENSName: "test.ens.eth",
		IsTest:  true,
	}

	err := manager.upsertSavedAddress(sa, nil)
	require.NoError(t, err)

	rst, err := manager.GetRawSavedAddresses()
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))

	require.True(t, savedAddressDataIsEqual(sa, rst[0]))

	sa2 := sa

	if member == isTestMember {
		sa2.IsTest = !sa2.IsTest
	} else if member == addressMember {
		sa2.Address = common.Address{7}
	} else if member == ensMember {
		sa2.ENSName += "_"
	} else {
		t.Error("Unsupported member change. Please add it to the list")
	}

	err = manager.upsertSavedAddress(sa2, nil)
	require.NoError(t, err)

	rst, err = manager.GetRawSavedAddresses()
	require.NoError(t, err)
	require.Equal(t, 2, len(rst))

	// The order of records returned by GetRawSavedAddresses is not
	// guaranteed to be the same as insertions, so swap indices if first record does not match
	firstIndex := 0
	secondIndex := 1
	if rst[firstIndex] != sa {
		firstIndex = 1
		secondIndex = 0
	}
	require.True(t, savedAddressDataIsEqual(sa, rst[firstIndex]))
	require.True(t, savedAddressDataIsEqual(sa2, rst[secondIndex]))
}

func TestSavedAddressesAddDifferentAddress(t *testing.T) {
	testInsertSameAddressWithOneChange(t, addressMember)
}

func TestSavedAddressesAddDifferentIsTest(t *testing.T) {
	testInsertSameAddressWithOneChange(t, isTestMember)
}

func TestSavedAddressesAddDifferentEns(t *testing.T) {
	testInsertSameAddressWithOneChange(t, ensMember)
}

func TestSavedAddressesAddSame(t *testing.T) {
	manager, stop := setupTestSavedAddressesDB(t)
	defer stop()

	// Insert one address
	sa := SavedAddress{
		Address: common.Address{1},
		ENSName: "test.ens.eth",
		IsTest:  true,
	}

	err := manager.upsertSavedAddress(sa, nil)
	require.NoError(t, err)

	rst, err := manager.GetRawSavedAddresses()
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))

	require.True(t, savedAddressDataIsEqual(sa, rst[0]))

	sa2 := sa
	err = manager.upsertSavedAddress(sa2, nil)
	require.NoError(t, err)

	rst, err = manager.GetRawSavedAddresses()
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
}
