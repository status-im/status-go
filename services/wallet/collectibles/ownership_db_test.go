package collectibles

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/services/wallet/bigint"
	w_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"

	"github.com/stretchr/testify/require"
)

func setupOwnershipDBTest(t *testing.T) (*OwnershipDB, func()) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	return NewOwnershipDB(db), func() {
		require.NoError(t, db.Close())
	}
}

func generateTestCollectibles(offset int, count int) (result thirdparty.TokenBalancesPerContractAddress) {
	result = make(thirdparty.TokenBalancesPerContractAddress)
	for i := offset; i < offset+count; i++ {
		contractAddress := common.BigToAddress(big.NewInt(int64(i % 10)))
		tokenID := &bigint.BigInt{Int: big.NewInt(int64(i))}

		result[contractAddress] = append(result[contractAddress], thirdparty.TokenBalance{
			TokenID: tokenID,
			Balance: &bigint.BigInt{Int: big.NewInt(int64(i%5 + 1))},
		})
	}
	return result
}

func testCollectiblesToList(chainID w_common.ChainID, balances thirdparty.TokenBalancesPerContractAddress) (result []thirdparty.CollectibleUniqueID) {
	result = make([]thirdparty.CollectibleUniqueID, 0, len(balances))
	for contractAddress, balances := range balances {
		for _, balance := range balances {
			newCollectible := thirdparty.CollectibleUniqueID{
				ContractID: thirdparty.ContractID{
					ChainID: chainID,
					Address: contractAddress,
				},
				TokenID: balance.TokenID,
			}
			result = append(result, newCollectible)
		}
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
	ownedBalancesChain0 := generateTestCollectibles(0, 10)
	ownedListChain0 := testCollectiblesToList(chainID0, ownedBalancesChain0)
	timestampChain0 := int64(1234567890)
	ownedBalancesChain1 := generateTestCollectibles(0, 15)
	ownedListChain1 := testCollectiblesToList(chainID1, ownedBalancesChain1)
	timestampChain1 := int64(1234567891)

	ownedList1 := append(ownedListChain0, ownedListChain1...)

	ownerAddress2 := common.HexToAddress("0x5678")
	ownedBalancesChain2 := generateTestCollectibles(0, 20)
	ownedListChain2 := testCollectiblesToList(chainID2, ownedBalancesChain2)
	timestampChain2 := int64(1234567892)

	ownedList2 := ownedListChain2

	ownerAddress3 := common.HexToAddress("0xABCD")
	ownedBalancesChain1b := generateTestCollectibles(len(ownedListChain1), 5)
	ownedListChain1b := testCollectiblesToList(chainID1, ownedBalancesChain1b)
	timestampChain1b := timestampChain1 - 100
	ownedBalancesChain2b := generateTestCollectibles(len(ownedListChain2), 20)
	// Add one collectible that is already owned by ownerAddress2
	commonChainID := chainID2
	var commonContractAddress common.Address
	var commonTokenID *bigint.BigInt
	var commonBalanceAddress2 *bigint.BigInt
	commonBalanceAddress3 := &bigint.BigInt{Int: big.NewInt(5)}

	for contractAddress, balances := range ownedBalancesChain2 {
		for _, balance := range balances {
			commonContractAddress = contractAddress
			commonTokenID = balance.TokenID
			commonBalanceAddress2 = balance.Balance

			newBalance := thirdparty.TokenBalance{
				TokenID: commonTokenID,
				Balance: commonBalanceAddress3,
			}
			ownedBalancesChain2b[commonContractAddress] = append(ownedBalancesChain2b[commonContractAddress], newBalance)
			break
		}
		break
	}

	ownedListChain2b := testCollectiblesToList(chainID2, ownedBalancesChain2b)
	timestampChain2b := timestampChain2 + 100

	ownedList3 := append(ownedListChain1b, ownedListChain2b...)

	allChains := []w_common.ChainID{chainID0, chainID1, chainID2}
	allOwnerAddresses := []common.Address{ownerAddress1, ownerAddress2, ownerAddress3}
	allCollectibles := append(ownedList1, ownedList2...)
	allCollectibles = append(allCollectibles, ownedList3...)

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
			Address:     ownerAddress2,
			Balance:     commonBalanceAddress2,
			TxTimestamp: unknownUpdateTimestamp,
		},
		{
			Address:     ownerAddress3,
			Balance:     commonBalanceAddress3,
			TxTimestamp: unknownUpdateTimestamp,
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
	ownedBalancesChain0 := generateTestCollectibles(0, 10)
	ownedListChain0 := testCollectiblesToList(chainID0, ownedBalancesChain0)
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

	count := 0
	for contractAddress, balances := range ownedBalancesChain0 {
		for i, balance := range balances {
			if count == 0 {
				count++
				ownedBalancesChain0[contractAddress] = ownedBalancesChain0[contractAddress][1:]
				removedID = thirdparty.CollectibleUniqueID{
					ContractID: thirdparty.ContractID{
						ChainID: chainID0,
						Address: contractAddress,
					},
					TokenID: balance.TokenID,
				}
			} else if count == 1 {
				count++
				ownedBalancesChain0[contractAddress][i].Balance = &bigint.BigInt{Int: big.NewInt(100)}
				updatedID = thirdparty.CollectibleUniqueID{
					ContractID: thirdparty.ContractID{
						ChainID: chainID0,
						Address: contractAddress,
					},
					TokenID: balance.TokenID,
				}
			}
		}
	}
	ownedListChain0 = testCollectiblesToList(chainID0, ownedBalancesChain0)

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

	ownedBalancesChain := thirdparty.TokenBalancesPerContractAddress{
		contractAddress: []thirdparty.TokenBalance{
			{
				TokenID: tokenID,
				Balance: balance,
			},
		},
	}
	ownedListChain := testCollectiblesToList(chainID, ownedBalancesChain)

	ownership := []thirdparty.AccountBalance{
		{
			Address:     ownerAddress,
			Balance:     balance,
			TxTimestamp: unknownUpdateTimestamp,
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
	ownedBalancesChain0 := generateTestCollectibles(0, 10)
	ownedListChain0 := testCollectiblesToList(chainID0, ownedBalancesChain0)
	timestampChain0 := int64(1234567890)

	var err error

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

	firstCollectibleID := ownedListChain0[0]
	firstTxID := common.HexToHash("0x1234")
	err = oDB.SetTransferID(ownerAddress1, firstCollectibleID, firstTxID)
	require.NoError(t, err)

	for _, id := range ownedListChain0 {
		loadedTransferID, err := oDB.GetTransferID(ownerAddress1, id)
		require.NoError(t, err)
		if id == firstCollectibleID {
			require.Equal(t, firstTxID, *loadedTransferID)
		} else {
			require.Nil(t, loadedTransferID)
		}
	}

	// Even though the first collectible has a TransferID set, since there's no matching entry in the transfers table it
	// should return unknownUpdateTimestamp
	firstOwnership, err := oDB.GetOwnership(firstCollectibleID)
	require.NoError(t, err)
	require.Equal(t, unknownUpdateTimestamp, firstOwnership[0].TxTimestamp)

	trs, _, _ := transfer.GenerateTestTransfers(t, oDB.db, 1, 5)
	trs[0].To = ownerAddress1
	trs[0].ChainID = chainID0
	trs[0].Hash = firstTxID

	for i := range trs {
		if i == 0 {
			transfer.InsertTestTransferWithOptions(t, oDB.db, trs[i].To, &trs[i], &transfer.TestTransferOptions{
				TokenAddress: firstCollectibleID.ContractID.Address,
				TokenID:      firstCollectibleID.TokenID.Int,
			})
		} else {
			transfer.InsertTestTransfer(t, oDB.db, trs[i].To, &trs[i])
		}
	}

	// There should now be a valid timestamp
	firstOwnership, err = oDB.GetOwnership(firstCollectibleID)
	require.NoError(t, err)
	require.Equal(t, trs[0].Timestamp, firstOwnership[0].TxTimestamp)
}
