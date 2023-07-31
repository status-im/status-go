package collectibles

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/services/wallet/bigint"
	w_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"

	"github.com/stretchr/testify/require"
)

func setupOwnershipDBTest(t *testing.T) (*OwnershipDB, func()) {
	db, err := appdatabase.InitializeDB(":memory:", "wallet-collecitibles-ownership-db-tests", 1)
	require.NoError(t, err)
	return NewOwnershipDB(db), func() {
		require.NoError(t, db.Close())
	}
}

func generateTestCollectibles(chainID w_common.ChainID, count int) (result []thirdparty.CollectibleUniqueID) {
	result = make([]thirdparty.CollectibleUniqueID, 0, count)
	for i := 0; i < count; i++ {
		bigI := big.NewInt(int64(i))
		newCollectible := thirdparty.CollectibleUniqueID{
			ContractID: thirdparty.ContractID{
				ChainID: chainID,
				Address: common.BigToAddress(bigI),
			},
			TokenID: &bigint.BigInt{Int: bigI},
		}
		result = append(result, newCollectible)
	}
	return result
}

func TestUpdateOwnership(t *testing.T) {
	oDB, cleanDB := setupOwnershipDBTest(t)
	defer cleanDB()

	ownerAddress1 := common.HexToAddress("0x1234")
	chainID0 := w_common.ChainID(0)
	ownedListChain0 := generateTestCollectibles(chainID0, 10)
	chainID1 := w_common.ChainID(1)
	ownedListChain1 := generateTestCollectibles(chainID1, 15)

	ownedList1 := append(ownedListChain0, ownedListChain1...)

	ownerAddress2 := common.HexToAddress("0x5678")
	chainID2 := w_common.ChainID(2)
	ownedListChain2 := generateTestCollectibles(chainID2, 20)

	ownedList2 := ownedListChain2

	fullList := append(ownedList1, ownedList2...)

	randomAddress := common.HexToAddress("0xABCD")

	var err error

	err = oDB.Update(chainID0, ownerAddress1, ownedListChain0)
	require.NoError(t, err)
	err = oDB.Update(chainID1, ownerAddress1, ownedListChain1)
	require.NoError(t, err)

	err = oDB.Update(chainID2, ownerAddress2, ownedListChain2)
	require.NoError(t, err)

	loadedList, err := oDB.GetOwnedCollectibles([]w_common.ChainID{chainID0}, []common.Address{ownerAddress1}, 0, len(fullList))
	require.NoError(t, err)
	require.Equal(t, ownedListChain0, loadedList)

	loadedList, err = oDB.GetOwnedCollectibles([]w_common.ChainID{chainID0, chainID1}, []common.Address{ownerAddress1, randomAddress}, 0, len(fullList))
	require.NoError(t, err)
	require.Equal(t, ownedList1, loadedList)

	loadedList, err = oDB.GetOwnedCollectibles([]w_common.ChainID{chainID2}, []common.Address{ownerAddress2}, 0, len(fullList))
	require.NoError(t, err)
	require.Equal(t, ownedList2, loadedList)

	loadedList, err = oDB.GetOwnedCollectibles([]w_common.ChainID{chainID0, chainID1, chainID2}, []common.Address{ownerAddress1, ownerAddress2}, 0, len(fullList))
	require.NoError(t, err)
	require.Equal(t, fullList, loadedList)

	loadedList, err = oDB.GetOwnedCollectibles([]w_common.ChainID{chainID0}, []common.Address{randomAddress}, 0, len(fullList))
	require.NoError(t, err)
	require.Empty(t, loadedList)
}

func TestLargeTokenID(t *testing.T) {
	oDB, cleanDB := setupOwnershipDBTest(t)
	defer cleanDB()

	ownerAddress := common.HexToAddress("0xABCD")
	chainID := w_common.ChainID(0)

	ownedListChain := []thirdparty.CollectibleUniqueID{
		{
			ContractID: thirdparty.ContractID{
				ChainID: chainID,
				Address: common.HexToAddress("0x1234"),
			},
			TokenID: &bigint.BigInt{Int: big.NewInt(0).SetBytes([]byte("0x1234567890123456789012345678901234567890"))},
		},
	}

	var err error

	err = oDB.Update(chainID, ownerAddress, ownedListChain)
	require.NoError(t, err)

	loadedList, err := oDB.GetOwnedCollectibles([]w_common.ChainID{chainID}, []common.Address{ownerAddress}, 0, len(ownedListChain))
	require.NoError(t, err)
	require.Equal(t, ownedListChain, loadedList)
}
