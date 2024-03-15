package collectibles

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	w_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"

	"github.com/stretchr/testify/require"
)

func setupCollectionDataDBTest(t *testing.T) (*CollectionDataDB, func()) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	return NewCollectionDataDB(db), func() {
		require.NoError(t, db.Close())
	}
}

func TestUpdateCollectionsData(t *testing.T) {
	db, cleanDB := setupCollectionDataDBTest(t)
	defer cleanDB()

	data := thirdparty.GenerateTestCollectionsData(50)

	var err error

	err = db.SetData(data, true)
	require.NoError(t, err)

	ids := make([]thirdparty.ContractID, 0, len(data))
	for _, collection := range data {
		ids = append(ids, collection.ID)
	}

	// Check for missing IDs
	idsNotInDB, err := db.GetIDsNotInDB(ids)
	require.NoError(t, err)
	require.Empty(t, idsNotInDB)

	extraID0 := thirdparty.ContractID{
		ChainID: w_common.ChainID(100),
		Address: common.BigToAddress(big.NewInt(100)),
	}
	extraID1 := thirdparty.ContractID{
		ChainID: w_common.ChainID(101),
		Address: common.BigToAddress(big.NewInt(101)),
	}
	extraIds := []thirdparty.ContractID{extraID0, extraID1}

	idsNotInDB, err = db.GetIDsNotInDB(extraIds)
	require.NoError(t, err)
	require.ElementsMatch(t, extraIds, idsNotInDB)

	combinedIds := append(ids, extraIds...)
	idsNotInDB, err = db.GetIDsNotInDB(combinedIds)
	require.NoError(t, err)
	require.ElementsMatch(t, extraIds, idsNotInDB)

	// Check for loaded data
	loadedMap, err := db.GetData(ids)
	require.NoError(t, err)
	require.Equal(t, len(ids), len(loadedMap))

	for _, origC := range data {
		require.Equal(t, origC, loadedMap[origC.ID.HashKey()])
	}

	// update some collections, changing the provider
	c0Orig := data[0]
	c0 := c0Orig
	c0.Name = "new collection name 0"
	c0.Provider = "new collection provider 0"

	c1Orig := data[1]
	c1 := c1Orig
	c1.Name = "new collection name 1"
	c1.Provider = "new collection provider 1"

	// Test allowUpdate = false
	err = db.SetData([]thirdparty.CollectionData{c0, c1}, false)
	require.NoError(t, err)

	loadedMap, err = db.GetData([]thirdparty.ContractID{c0.ID, c1.ID})
	require.NoError(t, err)
	require.Equal(t, 2, len(loadedMap))

	require.Equal(t, c0Orig, loadedMap[c0.ID.HashKey()])
	require.Equal(t, c1Orig, loadedMap[c1.ID.HashKey()])

	// Test allowUpdate = true
	err = db.SetData([]thirdparty.CollectionData{c0, c1}, true)
	require.NoError(t, err)

	loadedMap, err = db.GetData([]thirdparty.ContractID{c0.ID, c1.ID})
	require.NoError(t, err)
	require.Equal(t, 2, len(loadedMap))

	require.Equal(t, c0, loadedMap[c0.ID.HashKey()])
	require.Equal(t, c1, loadedMap[c1.ID.HashKey()])
}
