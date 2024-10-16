package transactions_test

import (
	"math/rand"
	"strconv"
	"testing"

	crypto_rand "crypto/rand"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/transactions"
	"github.com/status-im/status-go/walletdatabase"

	"github.com/stretchr/testify/require"
)

func getRandomStatus() transactions.TxStatus {
	switch rand.Intn(3) { // nolint: gosec
	case 0:
		return transactions.Pending
	case 1:
		return transactions.Success
	case 2:
		return transactions.Failed
	}

	return transactions.Pending
}

func getRandomTrackedTx() transactions.TrackedTx {
	tx := transactions.TrackedTx{
		ID: transactions.TxIdentity{
			ChainID: common.ChainID(rand.Uint64() % 10), // nolint: gosec
			Hash:    eth.Hash{},
		},
		Timestamp: 123,
		Status:    getRandomStatus(),
	}
	_, _ = crypto_rand.Read(tx.ID.Hash[:])

	return tx
}

func getTestData() []struct {
	name string
	tx   transactions.TrackedTx
} {
	testData := make([]struct {
		name string
		tx   transactions.TrackedTx
	}, 10)

	for i := range testData {
		testData[i].name = "test_" + strconv.Itoa(i)
		testData[i].tx = getRandomTrackedTx()
	}

	return testData
}

func Test_PuTrackedTx(t *testing.T) {
	walletDB, closeFn, err := helpers.SetupTestSQLDB(walletdatabase.DbInitializer{}, "pendingtxtracker-tests")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, closeFn())
	}()

	db := transactions.NewDB(walletDB)

	for _, tt := range getTestData() {
		t.Run(tt.name, func(t *testing.T) {
			err := db.PutTx(tt.tx)
			require.NoError(t, err)

			readTx, err := db.GetTx(tt.tx.ID)
			require.NoError(t, err)
			require.EqualExportedValues(t, tt.tx, readTx)

			newStatus := getRandomStatus()
			err = db.UpdateTxStatus(tt.tx.ID, newStatus)
			require.NoError(t, err)

			readTx, err = db.GetTx(tt.tx.ID)
			require.NoError(t, err)
			require.Equal(t, newStatus, readTx.Status)
		})
	}
}
