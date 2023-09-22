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

func setupOwnershipDBTest(t *testing.T) (*OwnershipDB, func()) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	return NewOwnershipDB(db), func() {
		require.NoError(t, db.Close())
	}
}

func generateTestCollectibles(chainID w_common.ChainID, offset int, count int) (result []thirdparty.CollectibleUniqueID) {
	result = make([]thirdparty.CollectibleUniqueID, 0, count)
	for i := offset; i < offset+count; i++ {
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

	chainID0 := w_common.ChainID(0)
	chainID1 := w_common.ChainID(1)
	chainID2 := w_common.ChainID(2)

	ownerAddress1 := common.HexToAddress("0x1234")
	ownedListChain0 := generateTestCollectibles(chainID0, 0, 10)
	timestampChain0 := int64(1234567890)
	ownedListChain1 := generateTestCollectibles(chainID1, 0, 15)
	timestampChain1 := int64(1234567891)

	ownedList1 := append(ownedListChain0, ownedListChain1...)

	ownerAddress2 := common.HexToAddress("0x5678")
	ownedListChain2 := generateTestCollectibles(chainID2, 0, 20)
	timestampChain2 := int64(1234567892)

	ownedList2 := ownedListChain2

	ownerAddress3 := common.HexToAddress("0xABCD")
	ownedListChain1b := generateTestCollectibles(chainID1, len(ownedListChain1), 5)
	timestampChain1b := timestampChain1 - 100
	ownedListChain2b := generateTestCollectibles(chainID2, len(ownedListChain2), 20)
	timestampChain2b := timestampChain2 + 100

	ownedList3 := append(ownedListChain1b, ownedListChain2b...)

	allChains := []w_common.ChainID{chainID0, chainID1, chainID2}
	allOwnerAddresses := []common.Address{ownerAddress1, ownerAddress2, ownerAddress3}
	allCollectibles := append(ownedList1, ownedList2...)
	allCollectibles = append(allCollectibles, ownedList3...)

	randomAddress := common.HexToAddress("0xFFFF")

	var err error

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

	err = oDB.Update(chainID0, ownerAddress1, ownedListChain0, timestampChain0)
	require.NoError(t, err)

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

	err = oDB.Update(chainID1, ownerAddress1, ownedListChain1, timestampChain1)
	require.NoError(t, err)

	err = oDB.Update(chainID2, ownerAddress2, ownedListChain2, timestampChain2)
	require.NoError(t, err)

	err = oDB.Update(chainID1, ownerAddress3, ownedListChain1b, timestampChain1b)
	require.NoError(t, err)

	err = oDB.Update(chainID2, ownerAddress3, ownedListChain2b, timestampChain2b)
	require.NoError(t, err)

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
	require.Equal(t, ownedListChain0, loadedList)

	loadedList, err = oDB.GetOwnedCollectibles([]w_common.ChainID{chainID0, chainID1}, []common.Address{ownerAddress1, randomAddress}, 0, len(allCollectibles))
	require.NoError(t, err)
	require.Equal(t, ownedList1, loadedList)

	loadedList, err = oDB.GetOwnedCollectibles([]w_common.ChainID{chainID2}, []common.Address{ownerAddress2}, 0, len(allCollectibles))
	require.NoError(t, err)
	require.Equal(t, ownedList2, loadedList)

	loadedList, err = oDB.GetOwnedCollectibles(allChains, allOwnerAddresses, 0, len(allCollectibles))
	require.NoError(t, err)
	require.Equal(t, len(allCollectibles), len(loadedList))

	loadedList, err = oDB.GetOwnedCollectibles([]w_common.ChainID{chainID0}, []common.Address{randomAddress}, 0, len(allCollectibles))
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

	timestamp := int64(1234567890)

	var err error

	err = oDB.Update(chainID, ownerAddress, ownedListChain, timestamp)
	require.NoError(t, err)

	loadedList, err := oDB.GetOwnedCollectibles([]w_common.ChainID{chainID}, []common.Address{ownerAddress}, 0, len(ownedListChain))
	require.NoError(t, err)
	require.Equal(t, ownedListChain, loadedList)
}
