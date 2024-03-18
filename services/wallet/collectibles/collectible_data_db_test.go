package collectibles

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/services/wallet/bigint"
	w_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"

	"github.com/stretchr/testify/require"
)

func setupCollectibleDataDBTest(t *testing.T) (*CollectibleDataDB, func()) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	return NewCollectibleDataDB(db), func() {
		require.NoError(t, db.Close())
	}
}

func TestUpdateCollectiblesData(t *testing.T) {
	db, cleanDB := setupCollectibleDataDBTest(t)
	defer cleanDB()

	data := thirdparty.GenerateTestCollectiblesData(50)

	var err error

	err = db.SetData(data, true)
	require.NoError(t, err)

	ids := make([]thirdparty.CollectibleUniqueID, 0, len(data))
	for _, collectible := range data {
		ids = append(ids, collectible.ID)
	}

	// Check for missing IDs
	idsNotInDB, err := db.GetIDsNotInDB(ids)
	require.NoError(t, err)
	require.Empty(t, idsNotInDB)

	extraID0 := thirdparty.CollectibleUniqueID{
		ContractID: thirdparty.ContractID{
			ChainID: w_common.ChainID(100),
			Address: common.BigToAddress(big.NewInt(100)),
		},
		TokenID: &bigint.BigInt{Int: big.NewInt(100)},
	}
	extraID1 := thirdparty.CollectibleUniqueID{
		ContractID: thirdparty.ContractID{
			ChainID: w_common.ChainID(101),
			Address: common.BigToAddress(big.NewInt(101)),
		},
		TokenID: &bigint.BigInt{Int: big.NewInt(101)},
	}
	extraIds := []thirdparty.CollectibleUniqueID{extraID0, extraID1}

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

	// update some collectibles, changing the provider
	c0Orig := data[0]
	c0 := c0Orig
	c0.Name = "new collectible name 0"
	c0.Provider = "new collectible provider 0"

	c1Orig := data[1]
	c1 := c1Orig
	c1.Name = "new collectible name 1"
	c1.Provider = "new collectible provider 1"

	// Test allowUpdate = false
	err = db.SetData([]thirdparty.CollectibleData{c0, c1}, false)
	require.NoError(t, err)

	loadedMap, err = db.GetData([]thirdparty.CollectibleUniqueID{c0.ID, c1.ID})
	require.NoError(t, err)
	require.Equal(t, 2, len(loadedMap))

	require.Equal(t, c0Orig, loadedMap[c0.ID.HashKey()])
	require.Equal(t, c1Orig, loadedMap[c1.ID.HashKey()])

	// Test allowUpdate = true
	err = db.SetData([]thirdparty.CollectibleData{c0, c1}, true)
	require.NoError(t, err)

	loadedMap, err = db.GetData([]thirdparty.CollectibleUniqueID{c0.ID, c1.ID})
	require.NoError(t, err)
	require.Equal(t, 2, len(loadedMap))

	require.Equal(t, c0, loadedMap[c0.ID.HashKey()])
	require.Equal(t, c1, loadedMap[c1.ID.HashKey()])
}

func TestUpdateCommunityData(t *testing.T) {
	db, cleanDB := setupCollectibleDataDBTest(t)
	defer cleanDB()

	const nData = 50
	data := thirdparty.GenerateTestCollectiblesData(nData)
	communityData := thirdparty.GenerateTestCollectiblesCommunityData(nData)

	var err error

	err = db.SetData(data, true)
	require.NoError(t, err)

	for i := 0; i < nData; i++ {
		err = db.SetCommunityInfo(data[i].ID, communityData[i])
		require.NoError(t, err)
	}

	for i := 0; i < nData; i++ {
		loadedCommunityData, err := db.GetCommunityInfo(data[i].ID)
		require.NoError(t, err)
		require.Equal(t, communityData[i], *loadedCommunityData)
	}
}
