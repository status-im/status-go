package transfer

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/sqlite"
)

func setupTestTransactionDB(t *testing.T) (*TransactionManager, func()) {
	db, err := appdatabase.InitializeDB(sqlite.InMemoryPath, "wallet-tests", sqlite.ReducedKDFIterationsNumber)
	require.NoError(t, err)
	return &TransactionManager{db, nil, nil, nil, nil}, func() {
		require.NoError(t, db.Close())
	}
}

func TestPendingTransactions(t *testing.T) {
	manager, stop := setupTestTransactionDB(t)
	defer stop()

	trx := PendingTransaction{
		Hash:           common.Hash{1},
		From:           common.Address{1},
		To:             common.Address{2},
		Type:           RegisterENS,
		AdditionalData: "someuser.stateofus.eth",
		Value:          bigint.BigInt{Int: big.NewInt(123)},
		GasLimit:       bigint.BigInt{Int: big.NewInt(21000)},
		GasPrice:       bigint.BigInt{Int: big.NewInt(1)},
		ChainID:        777,
	}

	rst, err := manager.GetAllPending([]uint64{777})
	require.NoError(t, err)
	require.Nil(t, rst)

	rst, err = manager.GetPendingByAddress([]uint64{777}, trx.From)
	require.NoError(t, err)
	require.Nil(t, rst)

	err = manager.AddPending(trx)
	require.NoError(t, err)

	rst, err = manager.GetPendingByAddress([]uint64{777}, trx.From)
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
	require.Equal(t, trx, *rst[0])

	rst, err = manager.GetAllPending([]uint64{777})
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
	require.Equal(t, trx, *rst[0])

	rst, err = manager.GetPendingByAddress([]uint64{777}, common.Address{2})
	require.NoError(t, err)
	require.Nil(t, rst)

	err = manager.DeletePending(777, trx.Hash)
	require.NoError(t, err)

	rst, err = manager.GetPendingByAddress([]uint64{777}, trx.From)
	require.NoError(t, err)
	require.Equal(t, 0, len(rst))

	rst, err = manager.GetAllPending([]uint64{777})
	require.NoError(t, err)
	require.Equal(t, 0, len(rst))
}
