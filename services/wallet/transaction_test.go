package wallet

import (
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/services/wallet/bigint"
)

func setupTestTransactionDB(t *testing.T) (*TransactionManager, func()) {
	tmpfile, err := ioutil.TempFile("", "wallet-transactions-tests-")
	require.NoError(t, err)
	db, err := appdatabase.InitializeDB(tmpfile.Name(), "wallet-tests")
	require.NoError(t, err)
	return &TransactionManager{db, nil, nil, nil, nil}, func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.Remove(tmpfile.Name()))
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
		Value:          bigint.BigInt{big.NewInt(123)},
		GasLimit:       bigint.BigInt{big.NewInt(21000)},
		GasPrice:       bigint.BigInt{big.NewInt(1)},
		ChainID:        777,
	}

	rst, err := manager.getAllPendings([]uint64{777})
	require.NoError(t, err)
	require.Nil(t, rst)

	rst, err = manager.getPendingByAddress([]uint64{777}, trx.From)
	require.NoError(t, err)
	require.Nil(t, rst)

	err = manager.addPending(trx)
	require.NoError(t, err)

	rst, err = manager.getPendingByAddress([]uint64{777}, trx.From)
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
	require.Equal(t, trx, *rst[0])

	rst, err = manager.getAllPendings([]uint64{777})
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
	require.Equal(t, trx, *rst[0])

	rst, err = manager.getPendingByAddress([]uint64{777}, common.Address{2})
	require.NoError(t, err)
	require.Nil(t, rst)

	err = manager.deletePending(777, trx.Hash)
	require.NoError(t, err)

	rst, err = manager.getPendingByAddress([]uint64{777}, trx.From)
	require.NoError(t, err)
	require.Equal(t, 0, len(rst))

	rst, err = manager.getAllPendings([]uint64{777})
	require.NoError(t, err)
	require.Equal(t, 0, len(rst))
}
