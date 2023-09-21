package collectibles

import (
	"fmt"
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

func generateTestCollectiblesData(count int) (result []thirdparty.CollectibleData) {
	result = make([]thirdparty.CollectibleData, 0, count)
	for i := 0; i < count; i++ {
		bigI := big.NewInt(int64(count))
		newCollectible := thirdparty.CollectibleData{
			ID: thirdparty.CollectibleUniqueID{
				ContractID: thirdparty.ContractID{
					ChainID: w_common.ChainID(i),
					Address: common.BigToAddress(bigI),
				},
				TokenID: &bigint.BigInt{Int: bigI},
			},
			Provider:           fmt.Sprintf("provider-%d", i),
			Name:               fmt.Sprintf("name-%d", i),
			Description:        fmt.Sprintf("description-%d", i),
			Permalink:          fmt.Sprintf("permalink-%d", i),
			ImageURL:           fmt.Sprintf("imageurl-%d", i),
			AnimationURL:       fmt.Sprintf("animationurl-%d", i),
			AnimationMediaType: fmt.Sprintf("animationmediatype-%d", i),
			Traits: []thirdparty.CollectibleTrait{
				{
					TraitType:   fmt.Sprintf("traittype-%d", i),
					Value:       fmt.Sprintf("traitvalue-%d", i),
					DisplayType: fmt.Sprintf("displaytype-%d", i),
					MaxValue:    fmt.Sprintf("maxvalue-%d", i),
				},
				{
					TraitType:   fmt.Sprintf("traittype-%d", i),
					Value:       fmt.Sprintf("traitvalue-%d", i),
					DisplayType: fmt.Sprintf("displaytype-%d", i),
					MaxValue:    fmt.Sprintf("maxvalue-%d", i),
				},
				{
					TraitType:   fmt.Sprintf("traittype-%d", i),
					Value:       fmt.Sprintf("traitvalue-%d", i),
					DisplayType: fmt.Sprintf("displaytype-%d", i),
					MaxValue:    fmt.Sprintf("maxvalue-%d", i),
				},
			},
			BackgroundColor: fmt.Sprintf("backgroundcolor-%d", i),
			TokenURI:        fmt.Sprintf("tokenuri-%d", i),
			CommunityID:     fmt.Sprintf("communityid-%d", i),
		}
		result = append(result, newCollectible)
	}
	return result
}

func TestUpdateCollectiblesData(t *testing.T) {
	db, cleanDB := setupCollectibleDataDBTest(t)
	defer cleanDB()

	data := generateTestCollectiblesData(50)

	var err error

	err = db.SetData(data)
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

	// update some collectibles, changing the provider
	c0 := data[0]
	c0.Name = "new collectible name 0"
	c0.Provider = "new collectible provider 0"

	c1 := data[1]
	c1.Name = "new collectible name 1"
	c1.Provider = "new collectible provider 1"

	err = db.SetData([]thirdparty.CollectibleData{c0, c1})
	require.NoError(t, err)

	loadedMap, err = db.GetData([]thirdparty.CollectibleUniqueID{c0.ID, c1.ID})
	require.NoError(t, err)
	require.Equal(t, 2, len(loadedMap))

	require.Equal(t, c0, loadedMap[c0.ID.HashKey()])
	require.Equal(t, c1, loadedMap[c1.ID.HashKey()])
}
