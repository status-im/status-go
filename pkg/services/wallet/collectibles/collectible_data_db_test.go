package collectibles

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/walletdb"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/pkg/services/wallet/bigint"
	w_common "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
)

func setupCollectibleDataDBTest(t *testing.T) (*CollectibleDataDB, func()) {
	db, err := testutils.SetupTestMemorySQLDB(walletdb.DbInitializer{})
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
	idsNeedingFetch, err := db.GetIDsNeedingFetch(ids)
	require.NoError(t, err)
	require.Empty(t, idsNeedingFetch)

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

	idsNeedingFetch, err = db.GetIDsNeedingFetch(extraIds)
	require.NoError(t, err)
	require.ElementsMatch(t, extraIds, idsNeedingFetch)

	combinedIds := append(ids, extraIds...)
	idsNeedingFetch, err = db.GetIDsNeedingFetch(combinedIds)
	require.NoError(t, err)
	require.ElementsMatch(t, extraIds, idsNeedingFetch)

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

func TestCollectiblesCachedByOlderMappingAreRefetched(t *testing.T) {
	db, cleanDB := setupCollectibleDataDBTest(t)
	defer cleanDB()

	data := thirdparty.GenerateTestCollectiblesData(3)
	require.NoError(t, db.SetData(data, true))

	ids := make([]thirdparty.CollectibleUniqueID, 0, len(data))
	for _, collectible := range data {
		ids = append(ids, collectible.ID)
	}

	// Rows just written by the current mapping need nothing.
	needingFetch, err := db.GetIDsNeedingFetch(ids)
	require.NoError(t, err)
	require.Empty(t, needingFetch)

	// Rows left behind by an older mapping - the state every cached row is in
	// right after a migration adds a field - must be refetched even though they
	// are present.
	_, err = db.db.Exec(`UPDATE collectible_data_cache SET metadata_version = ?`, collectibleMetadataVersion-1)
	require.NoError(t, err)

	needingFetch, err = db.GetIDsNeedingFetch(ids)
	require.NoError(t, err)
	require.ElementsMatch(t, ids, needingFetch)

	// Writing without allowUpdate still records that the mapping ran, so a
	// collectible whose data cannot be improved is not refetched on every read.
	require.NoError(t, db.SetData(data, false))

	needingFetch, err = db.GetIDsNeedingFetch(ids)
	require.NoError(t, err)
	require.Empty(t, needingFetch)
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
