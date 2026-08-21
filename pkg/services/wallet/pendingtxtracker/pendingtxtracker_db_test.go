package pendingtxtracker_test

import (
	crypto_rand "crypto/rand"
	"math/rand"
	"strconv"
	"testing"

	eth "github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/walletdb"
	"github.com/status-im/status-go/internal/testutils"
	ac "github.com/status-im/status-go/pkg/services/wallet/activity/common"
	"github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/pendingtxtracker"
)

func getRandomStatus() ac.TxStatus {
	switch rand.Intn(3) { // nolint: gosec
	case 0:
		return ac.Pending
	case 1:
		return ac.Success
	case 2:
		return ac.Failed
	}

	return ac.Pending
}

func getRandomTrackedTx() pendingtxtracker.TrackedTx {
	tx := pendingtxtracker.TrackedTx{
		ID: pendingtxtracker.TxIdentity{
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
	tx   pendingtxtracker.TrackedTx
} {
	testData := make([]struct {
		name string
		tx   pendingtxtracker.TrackedTx
	}, 10)

	for i := range testData {
		testData[i].name = "test_" + strconv.Itoa(i)
		testData[i].tx = getRandomTrackedTx()
	}

	return testData
}

func Test_PuTrackedTx(t *testing.T) {
	walletDB, closeFn, err := testutils.SetupTestSQLDB(walletdb.DbInitializer{}, "pendingtxtracker-tests")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, closeFn())
	}()

	db := pendingtxtracker.NewDB(walletDB)

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
