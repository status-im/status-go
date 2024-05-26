package transfer

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	wallet_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
)

func setupTestMultiTransactionDB(t *testing.T) (*MultiTransactionDB, func()) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	SetMultiTransactionIDGenerator(StaticIDCounter()) // to have different multi-transaction IDs even with fast execution
	return NewMultiTransactionDB(db), func() {
		require.NoError(t, db.Close())
	}
}
func TestCreateMultiTransaction(t *testing.T) {
	mtDB, cleanup := setupTestMultiTransactionDB(t)
	defer cleanup()

	tr := generateTestTransfer(0)
	multiTransaction := GenerateTestSendMultiTransaction(tr)

	err := mtDB.CreateMultiTransaction(&multiTransaction)
	require.NoError(t, err)

	// Add assertions here to verify the result of the CreateMultiTransaction method
	details := NewMultiTxDetails()
	details.IDs = []wallet_common.MultiTransactionIDType{multiTransaction.ID}
	mtx, err := mtDB.ReadMultiTransactions(details)
	require.NoError(t, err)
	require.Len(t, mtx, 1)
	require.True(t, areMultiTransactionsEqual(&multiTransaction, mtx[0]))
}
func TestReadMultiTransactions(t *testing.T) {
	mtDB, cleanup := setupTestMultiTransactionDB(t)
	defer cleanup()

	// Create test multi transactions
	tr := generateTestTransfer(0)
	mt1 := GenerateTestSendMultiTransaction(tr)
	tr20 := generateTestTransfer(1)
	tr21 := generateTestTransfer(2)
	mt2 := GenerateTestBridgeMultiTransaction(tr20, tr21)
	tr3 := generateTestTransfer(3)
	mt3 := GenerateTestSwapMultiTransaction(tr3, "SNT", 100)

	require.NotEqual(t, mt1.ID, mt2.ID)
	require.NotEqual(t, mt1.ID, mt3.ID)

	err := mtDB.CreateMultiTransaction(&mt1)
	require.NoError(t, err)
	err = mtDB.CreateMultiTransaction(&mt2)
	require.NoError(t, err)
	err = mtDB.CreateMultiTransaction(&mt3)
	require.NoError(t, err)

	// Read multi transactions
	details := NewMultiTxDetails()
	details.IDs = []wallet_common.MultiTransactionIDType{mt1.ID, mt2.ID, mt3.ID}
	mtx, err := mtDB.ReadMultiTransactions(details)
	require.NoError(t, err)
	require.Len(t, mtx, 3)
	require.True(t, areMultiTransactionsEqual(&mt1, mtx[0]))
	require.True(t, areMultiTransactionsEqual(&mt2, mtx[1]))
	require.True(t, areMultiTransactionsEqual(&mt3, mtx[2]))
}

func TestUpdateMultiTransaction(t *testing.T) {
	mtDB, cleanup := setupTestMultiTransactionDB(t)
	defer cleanup()

	// Create test multi transaction
	tr := generateTestTransfer(0)
	multiTransaction := GenerateTestSendMultiTransaction(tr)

	err := mtDB.CreateMultiTransaction(&multiTransaction)
	require.NoError(t, err)

	// Update the multi transaction
	multiTransaction.FromNetworkID = 1
	multiTransaction.FromTxHash = common.Hash{1}
	multiTransaction.FromAddress = common.Address{2}
	multiTransaction.FromAsset = "fromAsset1"
	multiTransaction.FromAmount = (*hexutil.Big)(big.NewInt(234))
	multiTransaction.ToNetworkID = 2
	multiTransaction.ToTxHash = common.Hash{3}
	multiTransaction.ToAddress = common.Address{4}
	multiTransaction.ToAsset = "toAsset1"
	multiTransaction.ToAmount = (*hexutil.Big)(big.NewInt(345))
	multiTransaction.Type = MultiTransactionBridge
	multiTransaction.CrossTxID = "crossTxD2"

	err = mtDB.UpdateMultiTransaction(&multiTransaction)
	require.NoError(t, err)

	// Read the updated multi transaction
	details := NewMultiTxDetails()
	details.IDs = []wallet_common.MultiTransactionIDType{multiTransaction.ID}
	mtx, err := mtDB.ReadMultiTransactions(details)
	require.NoError(t, err)
	require.Len(t, mtx, 1)
	require.True(t, areMultiTransactionsEqual(&multiTransaction, mtx[0]))
}

func TestDeleteMultiTransaction(t *testing.T) {
	mtDB, cleanup := setupTestMultiTransactionDB(t)
	defer cleanup()

	// Create test multi transaction
	tr := generateTestTransfer(0)
	multiTransaction := GenerateTestSendMultiTransaction(tr)

	err := mtDB.CreateMultiTransaction(&multiTransaction)
	require.NoError(t, err)

	// Delete the multi transaction
	err = mtDB.DeleteMultiTransaction(multiTransaction.ID)
	require.NoError(t, err)

	// Read the deleted multi transaction
	mtx, err := mtDB.ReadMultiTransactions(&MultiTxDetails{
		IDs: []wallet_common.MultiTransactionIDType{multiTransaction.ID}})
	require.NoError(t, err)
	require.Len(t, mtx, 0)
}
