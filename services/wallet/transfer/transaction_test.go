package transfer

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/services/wallet/bigint"
)

func setupTestTransactionDB(t *testing.T) (*TransactionManager, func()) {
	db, err := appdatabase.SetupTestMemorySQLDB("wallet-transfer-transaction-tests")
	require.NoError(t, err)
	return &TransactionManager{db, nil, nil, nil, nil, &event.Feed{}}, func() {
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

func TestMultiTransactions(t *testing.T) {
	manager, stop := setupTestTransactionDB(t)
	defer stop()

	trx1 := MultiTransaction{
		Timestamp:   123,
		FromAddress: common.Address{1},
		ToAddress:   common.Address{2},
		FromAsset:   "fromAsset",
		ToAsset:     "toAsset",
		FromAmount:  (*hexutil.Big)(big.NewInt(123)),
		ToAmount:    (*hexutil.Big)(big.NewInt(234)),
		Type:        MultiTransactionBridge,
	}
	trx2 := trx1
	trx2.FromAmount = (*hexutil.Big)(big.NewInt(456))
	trx2.ToAmount = (*hexutil.Big)(big.NewInt(567))

	var err error
	ids := make([]MultiTransactionIDType, 2)
	ids[0], err = insertMultiTransaction(manager.db, &trx1)
	require.NoError(t, err)
	require.Equal(t, MultiTransactionIDType(1), ids[0])
	ids[1], err = insertMultiTransaction(manager.db, &trx2)
	require.NoError(t, err)
	require.Equal(t, MultiTransactionIDType(2), ids[1])

	rst, err := manager.GetMultiTransactions(context.Background(), []MultiTransactionIDType{ids[0], 555})
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))

	rst, err = manager.GetMultiTransactions(context.Background(), ids)
	require.NoError(t, err)
	require.Equal(t, 2, len(rst))

	for _, id := range ids {
		found := false
		for _, trx := range rst {
			found = found || id == MultiTransactionIDType(trx.ID)
		}
		require.True(t, found, "result contains transaction with id %d", id)
	}
}
