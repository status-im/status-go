package chain

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupDBTest(t *testing.T) (*DB, func()) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	return NewDB(db), func() {
		require.NoError(t, db.Close())
	}
}

func TestGetBlockByNumber(t *testing.T) {
	db, cleanup := setupDBTest(t)
	defer cleanup()

	chainID := uint64(1)
	blockNumber := big.NewInt(123)
	block := types.NewBlock(&types.Header{Number: blockNumber}, nil, nil, nil, nil)

	err := db.PutBlock(chainID, block)
	require.NoError(t, err)

	retrievedBlock, err := db.GetBlockByNumber(chainID, blockNumber)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedBlock)
	assert.Equal(t, block.Hash(), retrievedBlock.Hash())
}

func TestGetBlockByHash(t *testing.T) {
	db, cleanup := setupDBTest(t)
	defer cleanup()

	chainID := uint64(1)
	blockNumber := big.NewInt(123)
	block := types.NewBlock(&types.Header{Number: blockNumber}, nil, nil, nil, nil)

	err := db.PutBlock(chainID, block)
	require.NoError(t, err)

	retrievedBlock, err := db.GetBlockByHash(chainID, block.Hash())
	assert.NoError(t, err)
	assert.NotNil(t, retrievedBlock)
	assert.Equal(t, block.Number(), retrievedBlock.Number())
}

func TestGetBlockHeaderByNumber(t *testing.T) {
	db, cleanup := setupDBTest(t)
	defer cleanup()

	chainID := uint64(1)
	blockNumber := big.NewInt(123)
	header := &types.Header{Number: blockNumber}
	block := types.NewBlock(header, nil, nil, nil, nil)

	err := db.PutBlock(chainID, block)
	require.NoError(t, err)

	retrievedHeader, err := db.GetBlockHeaderByNumber(chainID, blockNumber)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedHeader)
	assert.Equal(t, header.Hash(), retrievedHeader.Hash())
}

func TestGetBlockHeaderByHash(t *testing.T) {
	db, cleanup := setupDBTest(t)
	defer cleanup()

	chainID := uint64(1)
	blockNumber := big.NewInt(123)
	header := &types.Header{Number: blockNumber}
	block := types.NewBlock(header, nil, nil, nil, nil)

	err := db.PutBlock(chainID, block)
	require.NoError(t, err)

	retrievedHeader, err := db.GetBlockHeaderByHash(chainID, block.Hash())
	assert.NoError(t, err)
	assert.NotNil(t, retrievedHeader)
	assert.Equal(t, header.Number, retrievedHeader.Number)
}

func TestPutAndGetTransactions(t *testing.T) {
	db, cleanup := setupDBTest(t)
	defer cleanup()

	chainID := uint64(1)
	blockNumber := big.NewInt(123)
	tx1 := types.NewTransaction(0, common.Address{}, big.NewInt(100), 21000, big.NewInt(1), nil)
	tx2 := types.NewTransaction(1, common.Address{}, big.NewInt(200), 21000, big.NewInt(1), nil)
	txs := types.Transactions{tx1, tx2}

	block := types.NewBlock(&types.Header{Number: blockNumber}, txs, nil, nil, nil)

	err := db.PutBlock(chainID, block)
	require.NoError(t, err)

	// Test GetTransactionsByBlockHash
	retrievedTxs, err := db.GetTransactionsByBlockHash(chainID, block.Hash())
	assert.NoError(t, err)
	assert.Len(t, retrievedTxs, 2)
	assert.Equal(t, tx1.Hash(), retrievedTxs[0].Hash())
	assert.Equal(t, tx2.Hash(), retrievedTxs[1].Hash())

	// Test GetTransactionsByBlockNumber
	retrievedTxs, err = db.GetTransactionsByBlockNumber(chainID, blockNumber)
	assert.NoError(t, err)
	assert.Len(t, retrievedTxs, 2)
	assert.Equal(t, tx1.Hash(), retrievedTxs[0].Hash())
	assert.Equal(t, tx2.Hash(), retrievedTxs[1].Hash())

	// Test GetTransactionByHash
	retrievedTx, err := db.GetTransactionByHash(chainID, tx1.Hash())
	assert.NoError(t, err)
	assert.NotNil(t, retrievedTx)
	assert.Equal(t, tx1.Hash(), retrievedTx.Hash())
}

func TestPutAndGetTransactionReceipt(t *testing.T) {
	db, cleanup := setupDBTest(t)
	defer cleanup()

	chainID := uint64(1)
	tx := types.NewTransaction(0, common.Address{}, big.NewInt(100), 21000, big.NewInt(1), nil)
	receipt := &types.Receipt{
		TxHash:      tx.Hash(),
		GasUsed:     21000,
		Status:      types.ReceiptStatusSuccessful,
		BlockNumber: big.NewInt(123),
	}

	err := db.PutTransactionReceipt(chainID, receipt)
	require.NoError(t, err)

	retrievedReceipt, err := db.GetTransactionReceipt(chainID, tx.Hash())
	assert.NoError(t, err)
	assert.NotNil(t, retrievedReceipt)
	assert.Equal(t, receipt.TxHash, retrievedReceipt.TxHash)
	assert.Equal(t, receipt.GasUsed, retrievedReceipt.GasUsed)
	assert.Equal(t, receipt.Status, retrievedReceipt.Status)
}

// Add more test functions as needed...
