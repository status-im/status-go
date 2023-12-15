package collectibles

import (
	"fmt"
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

func generateTestCollectionsData(count int) (result []thirdparty.CollectionData) {
	result = make([]thirdparty.CollectionData, 0, count)
	for i := 0; i < count; i++ {
		bigI := big.NewInt(int64(count))
		traits := make(map[string]thirdparty.CollectionTrait)
		for j := 0; j < 3; j++ {
			traits[fmt.Sprintf("traittype-%d", j)] = thirdparty.CollectionTrait{
				Min: float64(i+j) / 2,
				Max: float64(i+j) * 2,
			}
		}

		newCollection := thirdparty.CollectionData{
			ID: thirdparty.ContractID{
				ChainID: w_common.ChainID(i),
				Address: common.BigToAddress(bigI),
			},
			Provider:     fmt.Sprintf("provider-%d", i),
			Name:         fmt.Sprintf("name-%d", i),
			Slug:         fmt.Sprintf("slug-%d", i),
			ImageURL:     fmt.Sprintf("imageurl-%d", i),
			ImagePayload: []byte(fmt.Sprintf("imagepayload-%d", i)),
			Traits:       traits,
			CommunityID:  fmt.Sprintf("community-%d", i),
		}
		result = append(result, newCollection)
	}
	return result
}

func TestUpdateCollectionsData(t *testing.T) {
	db, cleanDB := setupCollectionDataDBTest(t)
	defer cleanDB()

	data := generateTestCollectionsData(50)

	var err error

	err = db.SetData(data)
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
	require.Equal(t, extraIds, idsNotInDB)

	combinedIds := append(ids, extraIds...)
	idsNotInDB, err = db.GetIDsNotInDB(combinedIds)
	require.NoError(t, err)
	require.Equal(t, extraIds, idsNotInDB)

	// Check for loaded data
	loadedMap, err := db.GetData(ids)
	require.NoError(t, err)
	require.Equal(t, len(ids), len(loadedMap))

	for _, origC := range data {
		require.Equal(t, origC, loadedMap[origC.ID.HashKey()])
	}

	// update some collections, changing the provider
	c0 := data[0]
	c0.Name = "new collection name 0"
	c0.Provider = "new collection provider 0"

	c1 := data[1]
	c1.Name = "new collection name 1"
	c1.Provider = "new collection provider 1"

	err = db.SetData([]thirdparty.CollectionData{c0, c1})
	require.NoError(t, err)

	loadedMap, err = db.GetData([]thirdparty.ContractID{c0.ID, c1.ID})
	require.NoError(t, err)
	require.Equal(t, 2, len(loadedMap))

	require.Equal(t, c0, loadedMap[c0.ID.HashKey()])
	require.Equal(t, c1, loadedMap[c1.ID.HashKey()])
}
