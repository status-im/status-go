package chain

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/t/helpers"

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

setupCachedClient(t *testing.T) (*CachedClient, func()) {
	db, closeDB := setupDBTest(t)



	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	return NewDB(db), func() {
		require.NoError(t, db.Close())
	}
}

func TestGetTransactionByHash(t *testing.T) {
	db, cleanup := setupDBTest(t)
	defer cleanup()

	chainID := uint64(1)
	txHash := common.HexToHash("0x123456789abcdef")
	tx := types.NewTransaction(0, common.HexToAddress("0x1"), big.NewInt(1), 100000, big.NewInt(1), nil)
	receipt := &types.Receipt{
		TxHash:  txHash,
		GasUsed: 100000,
		Status:  types.ReceiptStatusSuccessful,
	}

	err := db.PutTransaction(chainID, tx)
	require.NoError(t, err)

	retrievedTx, err := db.GetTransactionByHash(chainID, txHash)
	require.NoError(t, err)
	assert.Equal(t, tx, retrievedTx)
}
