package ownership

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/walletdatabase"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/pkg/services/wallet/bigint"
	w_common "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
)

func setupOwnershipDBTest(t *testing.T) (*OwnershipDB, func()) {
	db, err := testutils.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	return NewOwnershipDB(db), func() {
		require.NoError(t, db.Close())
	}
}

func generateTestCollectibles(chainID w_common.ChainID, offset int, count int) (result []thirdparty.CollectibleIDBalance) {
	result = make([]thirdparty.CollectibleIDBalance, 0, count)
	for i := offset; i < offset+count; i++ {
		contractAddress := common.BigToAddress(big.NewInt(int64(i % 10)))
		tokenID := &bigint.BigInt{Int: big.NewInt(int64(i))}

		result = append(result, thirdparty.CollectibleIDBalance{
			ID: thirdparty.CollectibleUniqueID{
				ContractID: thirdparty.ContractID{
					ChainID: chainID,
					Address: contractAddress,
				},
				TokenID: tokenID,
			},
			Balance:     &bigint.BigInt{Int: big.NewInt(int64(i%5 + 1))},
			TxTimestamp: int64(i),
		})
	}
	return result
}

func testCollectiblesToList(balances []thirdparty.CollectibleIDBalance) []thirdparty.CollectibleUniqueID {
	result := make([]thirdparty.CollectibleUniqueID, 0, len(balances))
	for _, balance := range balances {
		newCollectible := thirdparty.CollectibleUniqueID{
			ContractID: balance.ID.ContractID,
			TokenID:    balance.ID.TokenID,
		}
		result = append(result, newCollectible)
	}

	return result
}

func TestUpdateOwnership(t *testing.T) {
	oDB, cleanDB := setupOwnershipDBTest(t)
	defer cleanDB()

	chainID0 := w_common.ChainID(0)
	chainID1 := w_common.ChainID(1)
	chainID2 := w_common.ChainID(2)

	ownerAddress1 := common.HexToAddress("0x1234")
	ownedBalancesChain0 := generateTestCollectibles(chainID0, 0, 10)
	ownedListChain0 := testCollectiblesToList(ownedBalancesChain0)
	timestampChain0 := int64(1234567890)
	ownedBalancesChain1 := generateTestCollectibles(chainID1, 0, 15)
	ownedListChain1 := testCollectiblesToList(ownedBalancesChain1)
	timestampChain1 := int64(1234567891)

	ownedList1 := append(ownedListChain0, ownedListChain1...)

	ownerAddress2 := common.HexToAddress("0x5678")
	ownedBalancesChain2 := generateTestCollectibles(chainID2, 0, 20)
	ownedListChain2 := testCollectiblesToList(ownedBalancesChain2)
	timestampChain2 := int64(1234567892)

	ownedList2 := ownedListChain2

	ownerAddress3 := common.HexToAddress("0xABCD")
	ownedBalancesChain1b := generateTestCollectibles(chainID1, len(ownedListChain1), 5)
	ownedListChain1b := testCollectiblesToList(ownedBalancesChain1b)
	timestampChain1b := timestampChain1 - 100
	ownedBalancesChain2b := generateTestCollectibles(chainID2, len(ownedListChain2), 20)
	// Add one collectible that is already owned by ownerAddress2
	commonChainID := chainID2
	commonBalance := ownedBalancesChain2[0]
	commonContractAddress := commonBalance.ID.ContractID.Address
	commonTokenID := commonBalance.ID.TokenID
	commonBalanceAddress2 := commonBalance.Balance
	commonBalanceAddress3 := &bigint.BigInt{Int: big.NewInt(5)}

	newBalance := thirdparty.CollectibleIDBalance{
		ID: thirdparty.CollectibleUniqueID{
			ContractID: thirdparty.ContractID{
				ChainID: chainID2,
				Address: commonContractAddress,
			},
			TokenID: commonTokenID,
		},
		Balance: commonBalanceAddress3,
	}
	ownedBalancesChain2b = append(ownedBalancesChain2b, newBalance)

	ownedListChain2b := testCollectiblesToList(ownedBalancesChain2b)
	timestampChain2b := timestampChain2 + 100

	ownedList3 := append(ownedListChain1b, ownedListChain2b...)

	allChains := []w_common.ChainID{chainID0, chainID1, chainID2}
	allOwnerAddresses := []common.Address{ownerAddress1, ownerAddress2, ownerAddress3}
	allCollectibles := make([]thirdparty.CollectibleUniqueID, 0, len(ownedList1)+len(ownedList2)+len(ownedList3))
	allCollectibles = append(allCollectibles, ownedList1[1:]...)
	allCollectibles = append(allCollectibles, ownedList2...)
	allCollectibles = append(allCollectibles, ownedList3[:len(ownedList3)-1]...) // the last element of ownerdList3 is a duplicate of the first element of ownedList2

	randomAddress := common.HexToAddress("0xFFFF")

	var err error
	var removedIDs, updatedIDs, insertedIDs []thirdparty.CollectibleUniqueID

	var loadedTimestamp int64
	var loadedList []thirdparty.CollectibleUniqueID

	loadedTimestamp, err = oDB.GetOwnershipUpdateTimestamp(ownerAddress1, chainID0)
	require.NoError(t, err)
	require.Equal(t, InvalidTimestamp, loadedTimestamp)

	loadedTimestamp, err = oDB.GetLatestOwnershipUpdateTimestamp(chainID0)
	require.NoError(t, err)
	require.Equal(t, InvalidTimestamp, loadedTimestamp)

	loadedTimestamp, err = oDB.GetOwnershipUpdateTimestamp(ownerAddress1, chainID1)
	require.NoError(t, err)
	require.Equal(t, InvalidTimestamp, loadedTimestamp)

	loadedTimestamp, err = oDB.GetLatestOwnershipUpdateTimestamp(chainID1)
	require.NoError(t, err)
	require.Equal(t, InvalidTimestamp, loadedTimestamp)

	loadedTimestamp, err = oDB.GetOwnershipUpdateTimestamp(ownerAddress2, chainID2)
	require.NoError(t, err)
	require.Equal(t, InvalidTimestamp, loadedTimestamp)

	loadedTimestamp, err = oDB.GetLatestOwnershipUpdateTimestamp(chainID2)
	require.NoError(t, err)
	require.Equal(t, InvalidTimestamp, loadedTimestamp)

	removedIDs, updatedIDs, insertedIDs, err = oDB.Update(chainID0, ownerAddress1, ownedBalancesChain0, timestampChain0)
	require.NoError(t, err)
	require.Empty(t, removedIDs)
	require.Empty(t, updatedIDs)
	require.ElementsMatch(t, ownedListChain0, insertedIDs)

	loadedTimestamp, err = oDB.GetOwnershipUpdateTimestamp(ownerAddress1, chainID0)
	require.NoError(t, err)
	require.Equal(t, timestampChain0, loadedTimestamp)

	loadedTimestamp, err = oDB.GetLatestOwnershipUpdateTimestamp(chainID0)
	require.NoError(t, err)
	require.Equal(t, timestampChain0, loadedTimestamp)

	loadedTimestamp, err = oDB.GetOwnershipUpdateTimestamp(ownerAddress1, chainID1)
	require.NoError(t, err)
	require.Equal(t, InvalidTimestamp, loadedTimestamp)

	loadedTimestamp, err = oDB.GetLatestOwnershipUpdateTimestamp(chainID1)
	require.NoError(t, err)
	require.Equal(t, InvalidTimestamp, loadedTimestamp)

	loadedTimestamp, err = oDB.GetOwnershipUpdateTimestamp(ownerAddress2, chainID2)
	require.NoError(t, err)
	require.Equal(t, InvalidTimestamp, loadedTimestamp)

	loadedTimestamp, err = oDB.GetLatestOwnershipUpdateTimestamp(chainID2)
	require.NoError(t, err)
	require.Equal(t, InvalidTimestamp, loadedTimestamp)

	removedIDs, updatedIDs, insertedIDs, err = oDB.Update(chainID1, ownerAddress1, ownedBalancesChain1, timestampChain1)
	require.NoError(t, err)
	require.Empty(t, removedIDs)
	require.Empty(t, updatedIDs)
	require.ElementsMatch(t, ownedListChain1, insertedIDs)

	removedIDs, updatedIDs, insertedIDs, err = oDB.Update(chainID2, ownerAddress2, ownedBalancesChain2, timestampChain2)
	require.NoError(t, err)
	require.Empty(t, removedIDs)
	require.Empty(t, updatedIDs)
	require.ElementsMatch(t, ownedListChain2, insertedIDs)

	removedIDs, updatedIDs, insertedIDs, err = oDB.Update(chainID1, ownerAddress3, ownedBalancesChain1b, timestampChain1b)
	require.NoError(t, err)
	require.Empty(t, removedIDs)
	require.Empty(t, updatedIDs)
	require.ElementsMatch(t, ownedListChain1b, insertedIDs)

	removedIDs, updatedIDs, insertedIDs, err = oDB.Update(chainID2, ownerAddress3, ownedBalancesChain2b, timestampChain2b)
	require.NoError(t, err)
	require.Empty(t, removedIDs)
	require.Empty(t, updatedIDs)
	require.ElementsMatch(t, ownedListChain2b, insertedIDs)

	loadedTimestamp, err = oDB.GetOwnershipUpdateTimestamp(ownerAddress1, chainID0)
	require.NoError(t, err)
	require.Equal(t, timestampChain0, loadedTimestamp)

	loadedTimestamp, err = oDB.GetLatestOwnershipUpdateTimestamp(chainID0)
	require.NoError(t, err)
	require.Equal(t, timestampChain0, loadedTimestamp)

	loadedTimestamp, err = oDB.GetOwnershipUpdateTimestamp(ownerAddress1, chainID1)
	require.NoError(t, err)
	require.Equal(t, timestampChain1, loadedTimestamp)

	loadedTimestamp, err = oDB.GetOwnershipUpdateTimestamp(ownerAddress3, chainID1)
	require.NoError(t, err)
	require.Equal(t, timestampChain1b, loadedTimestamp)

	loadedTimestamp, err = oDB.GetLatestOwnershipUpdateTimestamp(chainID1)
	require.NoError(t, err)
	require.Equal(t, timestampChain1, loadedTimestamp)

	loadedTimestamp, err = oDB.GetOwnershipUpdateTimestamp(ownerAddress2, chainID2)
	require.NoError(t, err)
	require.Equal(t, timestampChain2, loadedTimestamp)

	loadedTimestamp, err = oDB.GetOwnershipUpdateTimestamp(ownerAddress3, chainID2)
	require.NoError(t, err)
	require.Equal(t, timestampChain2b, loadedTimestamp)

	loadedTimestamp, err = oDB.GetLatestOwnershipUpdateTimestamp(chainID2)
	require.NoError(t, err)
	require.Equal(t, timestampChain2b, loadedTimestamp)

	loadedList, err = oDB.GetOwnedCollectibles([]w_common.ChainID{chainID0}, []common.Address{ownerAddress1}, 0, len(allCollectibles))
	require.NoError(t, err)
	require.ElementsMatch(t, ownedListChain0, loadedList)

	loadedList, err = oDB.GetOwnedCollectibles([]w_common.ChainID{chainID0, chainID1}, []common.Address{ownerAddress1, randomAddress}, 0, len(allCollectibles))
	require.NoError(t, err)
	require.ElementsMatch(t, ownedList1, loadedList)

	loadedList, err = oDB.GetOwnedCollectibles([]w_common.ChainID{chainID2}, []common.Address{ownerAddress2}, 0, len(allCollectibles))
	require.NoError(t, err)
	require.ElementsMatch(t, ownedList2, loadedList)

	loadedList, err = oDB.GetOwnedCollectibles(allChains, allOwnerAddresses, 0, len(allCollectibles))
	require.NoError(t, err)
	require.Equal(t, len(allCollectibles), len(loadedList))

	loadedList, err = oDB.GetOwnedCollectibles([]w_common.ChainID{chainID0}, []common.Address{randomAddress}, 0, len(allCollectibles))
	require.NoError(t, err)
	require.Empty(t, loadedList)

	// Test GetOwnership for common token
	commonID := thirdparty.CollectibleUniqueID{
		ContractID: thirdparty.ContractID{
			ChainID: commonChainID,
			Address: commonContractAddress,
		},
		TokenID: commonTokenID,
	}
	loadedOwnership, err := oDB.GetOwnership(commonID)
	require.NoError(t, err)

	expectedOwnership := []thirdparty.AccountBalance{
		{
			Address: ownerAddress2,
			Balance: commonBalanceAddress2,
		},
		{
			Address: ownerAddress3,
			Balance: commonBalanceAddress3,
		},
	}

	require.ElementsMatch(t, expectedOwnership, loadedOwnership)

	// Test GetOwnership for random token
	randomID := thirdparty.CollectibleUniqueID{
		ContractID: thirdparty.ContractID{
			ChainID: 0xABCDEF,
			Address: common.BigToAddress(big.NewInt(int64(123456789))),
		},
		TokenID: &bigint.BigInt{Int: big.NewInt(int64(987654321))},
	}

	loadedOwnership, err = oDB.GetOwnership(randomID)
	require.NoError(t, err)
	require.Empty(t, loadedOwnership)
}

func TestUpdateOwnershipChanges(t *testing.T) {
	oDB, cleanDB := setupOwnershipDBTest(t)
	defer cleanDB()

	chainID0 := w_common.ChainID(0)
	ownerAddress1 := common.HexToAddress("0x1234")
	ownedBalancesChain0 := generateTestCollectibles(chainID0, 0, 10)
	ownedListChain0 := testCollectiblesToList(ownedBalancesChain0)
	timestampChain0 := int64(1234567890)

	var err error
	var removedIDs, updatedIDs, insertedIDs []thirdparty.CollectibleUniqueID

	var loadedList []thirdparty.CollectibleUniqueID

	removedIDs, updatedIDs, insertedIDs, err = oDB.Update(chainID0, ownerAddress1, ownedBalancesChain0, timestampChain0)
	require.NoError(t, err)
	require.Empty(t, removedIDs)
	require.Empty(t, updatedIDs)
	require.ElementsMatch(t, ownedListChain0, insertedIDs)

	loadedList, err = oDB.GetOwnedCollectibles([]w_common.ChainID{chainID0}, []common.Address{ownerAddress1}, 0, len(ownedListChain0))
	require.NoError(t, err)
	require.ElementsMatch(t, ownedListChain0, loadedList)

	// Remove one collectible and change balance of another
	var removedID, updatedID thirdparty.CollectibleUniqueID

	removedID = thirdparty.CollectibleUniqueID{
		ContractID: thirdparty.ContractID{
			ChainID: chainID0,
			Address: ownedBalancesChain0[0].ID.ContractID.Address,
		},
		TokenID: ownedBalancesChain0[0].ID.TokenID,
	}
	ownedBalancesChain0 = ownedBalancesChain0[1:]

	updatedID = thirdparty.CollectibleUniqueID{
		ContractID: thirdparty.ContractID{
			ChainID: chainID0,
			Address: ownedBalancesChain0[1].ID.ContractID.Address,
		},
		TokenID: ownedBalancesChain0[1].ID.TokenID,
	}
	ownedBalancesChain0[1].Balance = &bigint.BigInt{Int: big.NewInt(100)}

	ownedListChain0 = testCollectiblesToList(ownedBalancesChain0)

	removedIDs, updatedIDs, insertedIDs, err = oDB.Update(chainID0, ownerAddress1, ownedBalancesChain0, timestampChain0)
	require.NoError(t, err)
	require.ElementsMatch(t, []thirdparty.CollectibleUniqueID{removedID}, removedIDs)
	require.ElementsMatch(t, []thirdparty.CollectibleUniqueID{updatedID}, updatedIDs)
	require.Empty(t, insertedIDs)

	loadedList, err = oDB.GetOwnedCollectibles([]w_common.ChainID{chainID0}, []common.Address{ownerAddress1}, 0, len(ownedListChain0))
	require.NoError(t, err)
	require.ElementsMatch(t, ownedListChain0, loadedList)
}

func TestLargeTokenID(t *testing.T) {
	oDB, cleanDB := setupOwnershipDBTest(t)
	defer cleanDB()

	ownerAddress := common.HexToAddress("0xABCD")
	chainID := w_common.ChainID(0)
	contractAddress := common.HexToAddress("0x1234")
	tokenID := &bigint.BigInt{Int: big.NewInt(0).SetBytes([]byte("0x1234567890123456789012345678901234567890"))}
	balance := &bigint.BigInt{Int: big.NewInt(100)}

	ownedBalancesChain := []thirdparty.CollectibleIDBalance{
		{
			ID: thirdparty.CollectibleUniqueID{
				ContractID: thirdparty.ContractID{
					ChainID: chainID,
					Address: contractAddress,
				},
				TokenID: tokenID,
			},
			Balance:     balance,
			TxTimestamp: InvalidTimestamp,
		},
	}
	ownedListChain := testCollectiblesToList(ownedBalancesChain)

	ownership := []thirdparty.AccountBalance{
		{
			Address:     ownerAddress,
			Balance:     balance,
			TxTimestamp: InvalidTimestamp,
		},
	}

	timestamp := int64(1234567890)

	var err error

	_, _, _, err = oDB.Update(chainID, ownerAddress, ownedBalancesChain, timestamp)
	require.NoError(t, err)

	loadedList, err := oDB.GetOwnedCollectibles([]w_common.ChainID{chainID}, []common.Address{ownerAddress}, 0, len(ownedListChain))
	require.NoError(t, err)
	require.Equal(t, ownedListChain, loadedList)

	// Test GetOwnership
	id := thirdparty.CollectibleUniqueID{
		ContractID: thirdparty.ContractID{
			ChainID: chainID,
			Address: contractAddress,
		},
		TokenID: tokenID,
	}
	loadedOwnership, err := oDB.GetOwnership(id)
	require.NoError(t, err)
	require.Equal(t, ownership, loadedOwnership)
}

func TestCollectibleTransferID(t *testing.T) {
	oDB, cleanDB := setupOwnershipDBTest(t)
	defer cleanDB()

	chainID0 := w_common.ChainID(0)
	ownerAddress1 := common.HexToAddress("0x1234")
	ownedBalancesChain0 := generateTestCollectibles(chainID0, 0, 10)
	ownedListChain0 := testCollectiblesToList(ownedBalancesChain0)
	timestampChain0 := int64(1234567890)

	var err error
	var changed bool

	_, _, _, err = oDB.Update(chainID0, ownerAddress1, ownedBalancesChain0, timestampChain0)
	require.NoError(t, err)

	loadedList, err := oDB.GetCollectiblesWithNoTransferID(ownerAddress1, chainID0)
	require.NoError(t, err)
	require.ElementsMatch(t, ownedListChain0, loadedList)

	for _, id := range ownedListChain0 {
		loadedTransferID, err := oDB.GetTransferID(ownerAddress1, id)
		require.NoError(t, err)
		require.Nil(t, loadedTransferID)
	}

	randomAddress := common.HexToAddress("0xFFFF")
	randomCollectibleID := thirdparty.CollectibleUniqueID{
		ContractID: thirdparty.ContractID{
			ChainID: 0xABCDEF,
			Address: common.BigToAddress(big.NewInt(int64(123456789))),
		},
		TokenID: &bigint.BigInt{Int: big.NewInt(int64(987654321))},
	}
	randomTxID := common.HexToHash("0xEEEE")
	changed, err = oDB.SetTransferID(randomAddress, randomCollectibleID, randomTxID)
	require.NoError(t, err)
	require.False(t, changed)

	firstCollectibleID := ownedListChain0[0]
	firstTxID := common.HexToHash("0x1234")
	changed, err = oDB.SetTransferID(ownerAddress1, firstCollectibleID, firstTxID)
	require.NoError(t, err)
	require.True(t, changed)

	for _, id := range ownedListChain0 {
		loadedTransferID, err := oDB.GetTransferID(ownerAddress1, id)
		require.NoError(t, err)
		if id == firstCollectibleID {
			require.Equal(t, firstTxID, *loadedTransferID)
		} else {
			require.Nil(t, loadedTransferID)
		}
	}
}
