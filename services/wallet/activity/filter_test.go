package activity

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	eth "github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/services/wallet/testutils"
	"github.com/status-im/status-go/services/wallet/transfer"

	"github.com/stretchr/testify/require"
)

func setupTestFilterDB(t *testing.T) (db *sql.DB, close func()) {
	db, err := appdatabase.SetupTestMemorySQLDB("wallet-activity-tests-filter")
	require.NoError(t, err)

	return db, func() {
		require.NoError(t, db.Close())
	}
}

func TestGetRecipientsEmptyDB(t *testing.T) {
	db, close := setupTestFilterDB(t)
	defer close()

	entries, hasMore, err := GetRecipients(context.Background(), db, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 0, len(entries))
	require.False(t, hasMore)
}

func TestGetRecipients(t *testing.T) {
	db, close := setupTestFilterDB(t)
	defer close()

	// Add 6 extractable transactions
	trs, _, toTrs := transfer.GenerateTestTransfers(t, db, 0, 6)
	for i := range trs {
		transfer.InsertTestTransfer(t, db, trs[i].To, &trs[i])
	}

	entries, hasMore, err := GetRecipients(context.Background(), db, 0, 15)
	require.NoError(t, err)
	require.False(t, hasMore)
	require.Equal(t, 6, len(entries))
	for i := range entries {
		found := false
		for j := range toTrs {
			if entries[i] == toTrs[j] {
				found = true
				break
			}
		}
		require.True(t, found, fmt.Sprintf("recipient %s not found in toTrs", entries[i].Hex()))
	}

	entries, hasMore, err = GetRecipients(context.Background(), db, 0, 4)
	require.NoError(t, err)
	require.Equal(t, 4, len(entries))
	require.True(t, hasMore)
}

func TestGetOldestTimestampEmptyDB(t *testing.T) {
	db, close := setupTestFilterDB(t)
	defer close()

	timestamp, err := GetOldestTimestamp(context.Background(), db, []eth.Address{eth.HexToAddress("0x1")})
	require.NoError(t, err)
	require.Equal(t, int64(0), timestamp)
}

func TestGetOldestTimestamp(t *testing.T) {
	db, close := setupTestFilterDB(t)
	defer close()

	// Add 6 extractable transactions
	trs, _, _ := transfer.GenerateTestTransfers(t, db, 0, 7)
	for i := range trs {
		if i < 5 {
			transfer.InsertTestTransfer(t, db, trs[i].To, &trs[i])
		} else {
			transfer.InsertTestPendingTransaction(t, db, &trs[i])
		}
	}

	multiTxs := []transfer.TestMultiTransaction{
		transfer.GenerateTestBridgeMultiTransaction(trs[0], trs[1]),
		transfer.GenerateTestSwapMultiTransaction(trs[2], testutils.SntSymbol, 100),
	}

	// Extract oldest timestamp, no filter
	timestamp, err := GetOldestTimestamp(context.Background(), db, []eth.Address{})
	require.NoError(t, err)
	require.Equal(t, multiTxs[0].Timestamp, timestamp)

	// Test to filter
	timestamp, err = GetOldestTimestamp(context.Background(), db, []eth.Address{
		trs[3].To,
	})
	require.NoError(t, err)
	require.Equal(t, trs[3].Timestamp, timestamp)

	// Test from filter
	timestamp, err = GetOldestTimestamp(context.Background(), db, []eth.Address{
		trs[4].From,
	})
	require.NoError(t, err)
	require.Equal(t, trs[4].Timestamp, timestamp)

	// Test MT
	timestamp, err = GetOldestTimestamp(context.Background(), db, []eth.Address{
		multiTxs[1].FromAddress, trs[4].To,
	})
	require.NoError(t, err)
	require.Equal(t, multiTxs[1].Timestamp, timestamp)

	// Test Pending
	timestamp, err = GetOldestTimestamp(context.Background(), db, []eth.Address{
		trs[6].To,
	})
	require.NoError(t, err)
	require.Equal(t, trs[6].Timestamp, timestamp)
}
