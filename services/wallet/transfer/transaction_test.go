package transfer

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/appdatabase"
)

func setupTestTransactionDB(t *testing.T) (*TransactionManager, func()) {
	db, err := appdatabase.SetupTestMemorySQLDB("wallet-transfer-transaction-tests")
	require.NoError(t, err)
	return &TransactionManager{db, nil, nil, nil, nil, nil, nil}, func() {
		require.NoError(t, db.Close())
	}
}

func areMultiTransactionsEqual(mt1, mt2 *MultiTransaction) bool {
	return mt1.Timestamp == mt2.Timestamp &&
		mt1.FromNetworkID == mt2.FromNetworkID &&
		mt1.ToNetworkID == mt2.ToNetworkID &&
		mt1.FromTxHash == mt2.FromTxHash &&
		mt1.ToTxHash == mt2.ToTxHash &&
		mt1.FromAddress == mt2.FromAddress &&
		mt1.ToAddress == mt2.ToAddress &&
		mt1.FromAsset == mt2.FromAsset &&
		mt1.ToAsset == mt2.ToAsset &&
		mt1.FromAmount.String() == mt2.FromAmount.String() &&
		mt1.ToAmount.String() == mt2.ToAmount.String() &&
		mt1.Type == mt2.Type &&
		mt1.CrossTxID == mt2.CrossTxID
}

func TestBridgeMultiTransactions(t *testing.T) {
	manager, stop := setupTestTransactionDB(t)
	defer stop()

	trx1 := MultiTransaction{
		Timestamp:     123,
		FromNetworkID: 0,
		ToNetworkID:   1,
		FromTxHash:    common.Hash{5},
		// Empty ToTxHash
		FromAddress: common.Address{1},
		ToAddress:   common.Address{2},
		FromAsset:   "fromAsset",
		ToAsset:     "toAsset",
		FromAmount:  (*hexutil.Big)(big.NewInt(123)),
		ToAmount:    (*hexutil.Big)(big.NewInt(234)),
		Type:        MultiTransactionBridge,
		CrossTxID:   "crossTxD1",
	}

	trx2 := MultiTransaction{
		Timestamp:     321,
		FromNetworkID: 1,
		ToNetworkID:   0,
		//Empty FromTxHash
		ToTxHash:    common.Hash{6},
		FromAddress: common.Address{2},
		ToAddress:   common.Address{1},
		FromAsset:   "fromAsset",
		ToAsset:     "toAsset",
		FromAmount:  (*hexutil.Big)(big.NewInt(123)),
		ToAmount:    (*hexutil.Big)(big.NewInt(234)),
		Type:        MultiTransactionBridge,
		CrossTxID:   "crossTxD2",
	}

	trxs := []*MultiTransaction{&trx1, &trx2}

	var err error
	ids := make([]MultiTransactionIDType, len(trxs))
	for i, trx := range trxs {
		ids[i], err = insertMultiTransaction(manager.db, trx)
		require.NoError(t, err)
		require.Equal(t, MultiTransactionIDType(i+1), ids[i])
	}

	rst, err := manager.GetBridgeOriginMultiTransaction(context.Background(), trx1.ToNetworkID, trx1.CrossTxID)
	require.NoError(t, err)
	require.NotEmpty(t, rst)
	require.True(t, areMultiTransactionsEqual(&trx1, rst))

	rst, err = manager.GetBridgeDestinationMultiTransaction(context.Background(), trx1.ToNetworkID, trx1.CrossTxID)
	require.NoError(t, err)
	require.Empty(t, rst)

	rst, err = manager.GetBridgeOriginMultiTransaction(context.Background(), trx2.ToNetworkID, trx2.CrossTxID)
	require.NoError(t, err)
	require.Empty(t, rst)

	rst, err = manager.GetBridgeDestinationMultiTransaction(context.Background(), trx2.ToNetworkID, trx2.CrossTxID)
	require.NoError(t, err)
	require.NotEmpty(t, rst)
	require.True(t, areMultiTransactionsEqual(&trx2, rst))
}

func TestMultiTransactions(t *testing.T) {
	manager, stop := setupTestTransactionDB(t)
	defer stop()

	trx1 := MultiTransaction{
		Timestamp:     123,
		FromNetworkID: 0,
		ToNetworkID:   1,
		FromTxHash:    common.Hash{5},
		ToTxHash:      common.Hash{6},
		FromAddress:   common.Address{1},
		ToAddress:     common.Address{2},
		FromAsset:     "fromAsset",
		ToAsset:       "toAsset",
		FromAmount:    (*hexutil.Big)(big.NewInt(123)),
		ToAmount:      (*hexutil.Big)(big.NewInt(234)),
		Type:          MultiTransactionBridge,
		CrossTxID:     "crossTxD",
	}
	trx2 := trx1
	trx2.FromAmount = (*hexutil.Big)(big.NewInt(456))
	trx2.ToAmount = (*hexutil.Big)(big.NewInt(567))

	trxs := []*MultiTransaction{&trx1, &trx2}

	var err error
	ids := make([]MultiTransactionIDType, len(trxs))
	for i, trx := range trxs {
		ids[i], err = insertMultiTransaction(manager.db, trx)
		require.NoError(t, err)
		require.Equal(t, MultiTransactionIDType(i+1), ids[i])
	}

	rst, err := manager.GetMultiTransactions(context.Background(), []MultiTransactionIDType{ids[0], 555})
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
	require.True(t, areMultiTransactionsEqual(trxs[0], rst[0]))

	trx1.FromAmount = (*hexutil.Big)(big.NewInt(789))
	trx1.ToAmount = (*hexutil.Big)(big.NewInt(890))
	err = updateMultiTransaction(manager.db, &trx1)
	require.NoError(t, err)

	rst, err = manager.GetMultiTransactions(context.Background(), ids)
	require.NoError(t, err)
	require.Equal(t, len(ids), len(rst))

	for i, id := range ids {
		found := false
		for _, trx := range rst {
			if id == MultiTransactionIDType(trx.ID) {
				found = true
				require.True(t, areMultiTransactionsEqual(trxs[i], trx))
				break
			}
		}
		require.True(t, found, "result contains transaction with id %d", id)
	}
}
