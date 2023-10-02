package activity

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	eth "github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/testutils"
	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"

	"github.com/stretchr/testify/require"
)

func setupTestFilterDB(t *testing.T) (db *sql.DB, close func()) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	return db, func() {
		require.NoError(t, db.Close())
	}
}

// insertTestData inserts 6 extractable activity entries: 2 transfers, 2 pending transactions and 2 multi transactions
func insertTestData(t *testing.T, db *sql.DB, nullifyToForIndexes []int) (trs []transfer.TestTransfer, toTrs []eth.Address, multiTxs []transfer.TestMultiTransaction) {
	// Add 6 extractable transactions
	trs, _, toTrs = transfer.GenerateTestTransfers(t, db, 0, 7)
	multiTxs = []transfer.TestMultiTransaction{
		transfer.GenerateTestBridgeMultiTransaction(trs[0], trs[1]),
		transfer.GenerateTestSwapMultiTransaction(trs[2], testutils.SntSymbol, 100),
	}
	for j := range nullifyToForIndexes {
		if nullifyToForIndexes[j] == 1 {
			multiTxs[0].ToAddress = eth.Address{}
		}
		if nullifyToForIndexes[j] == 2 {
			multiTxs[1].ToAddress = eth.Address{}
		}
	}

	trs[0].MultiTransactionID = transfer.InsertTestMultiTransaction(t, db, &multiTxs[0])
	trs[1].MultiTransactionID = trs[0].MultiTransactionID
	trs[2].MultiTransactionID = transfer.InsertTestMultiTransaction(t, db, &multiTxs[1])

	for i := range trs {
		if i < 5 {
			var nullifyAddresses []eth.Address
			for j := range nullifyToForIndexes {
				if i == nullifyToForIndexes[j] {
					nullifyAddresses = append(nullifyAddresses, trs[i].To)
				}
			}
			transfer.InsertTestTransferWithOptions(t, db, trs[i].To, &trs[i], &transfer.TestTransferOptions{
				NullifyAddresses: nullifyAddresses,
			})
		} else {
			for j := range nullifyToForIndexes {
				if i == nullifyToForIndexes[j] {
					trs[i].To = eth.Address{}
				}
			}
			transfer.InsertTestPendingTransaction(t, db, &trs[i])
		}
	}
	return
}

func TestGetRecipients(t *testing.T) {
	db, close := setupTestFilterDB(t)
	defer close()

	trs, toTrs, _ := insertTestData(t, db, nil)

	// Generate and insert transactions with the same to address
	dupTrs, _, _ := transfer.GenerateTestTransfers(t, db, 8, 4)
	dupTrs[0].To = trs[1].To
	dupTrs[2].To = trs[2].To
	dupMultiTxs := []transfer.TestMultiTransaction{
		transfer.GenerateTestSendMultiTransaction(dupTrs[0]),
		transfer.GenerateTestSwapMultiTransaction(dupTrs[2], testutils.SntSymbol, 100),
	}
	dupTrs[0].MultiTransactionID = transfer.InsertTestMultiTransaction(t, db, &dupMultiTxs[0])
	transfer.InsertTestTransfer(t, db, dupTrs[0].To, &dupTrs[0])
	dupTrs[2].MultiTransactionID = transfer.InsertTestMultiTransaction(t, db, &dupMultiTxs[1])
	transfer.InsertTestPendingTransaction(t, db, &dupTrs[2])

	dupTrs[1].To = trs[3].To
	transfer.InsertTestTransfer(t, db, dupTrs[1].To, &dupTrs[1])
	dupTrs[3].To = trs[5].To
	transfer.InsertTestPendingTransaction(t, db, &dupTrs[3])

	entries, hasMore, err := GetRecipients(context.Background(), db, []common.ChainID{}, []eth.Address{}, 0, 15)
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

	entries, hasMore, err = GetRecipients(context.Background(), db, []common.ChainID{}, []eth.Address{}, 0, 2)
	require.NoError(t, err)
	require.Equal(t, 2, len(entries))
	require.True(t, hasMore)

	// Get Recipients from specific chains
	entries, hasMore, err = GetRecipients(context.Background(), db, []common.ChainID{10}, []eth.Address{}, 0, 15)

	require.NoError(t, err)
	require.Equal(t, 2, len(entries))
	require.False(t, hasMore)
	require.Equal(t, trs[5].To, entries[0])
	require.Equal(t, trs[2].To, entries[1])

	// Get Recipients from specific addresses
	entries, hasMore, err = GetRecipients(context.Background(), db, []common.ChainID{}, []eth.Address{trs[0].From}, 0, 15)

	require.NoError(t, err)
	require.Equal(t, 1, len(entries))
	require.False(t, hasMore)
	require.Equal(t, trs[1].To, entries[0])
}

func TestGetRecipients_NullAddresses(t *testing.T) {
	db, close := setupTestFilterDB(t)
	defer close()

	insertTestData(t, db, []int{1, 2, 3, 5})

	entries, hasMore, err := GetRecipients(context.Background(), db, []common.ChainID{}, []eth.Address{}, 0, 15)
	require.NoError(t, err)
	require.False(t, hasMore)
	require.Equal(t, 3, len(entries))
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

	trs, _, multiTxs := insertTestData(t, db, nil)

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

func TestGetOldestTimestamp_NullAddresses(t *testing.T) {
	db, close := setupTestFilterDB(t)
	defer close()

	trs, _, _ := transfer.GenerateTestTransfers(t, db, 0, 3)
	nullifyAddresses := []eth.Address{
		trs[0].To, trs[2].To, trs[1].From,
	}
	for i := range trs {
		transfer.InsertTestTransferWithOptions(t, db, trs[i].To, &trs[i], &transfer.TestTransferOptions{
			NullifyAddresses: nullifyAddresses,
		})
	}

	// Extract oldest timestamp, no filter
	timestamp, err := GetOldestTimestamp(context.Background(), db, []eth.Address{})
	require.NoError(t, err)
	require.Equal(t, trs[0].Timestamp, timestamp)

	// Test to filter
	timestamp, err = GetOldestTimestamp(context.Background(), db, []eth.Address{
		trs[1].To, trs[2].To,
	})
	require.NoError(t, err)
	require.Equal(t, trs[1].Timestamp, timestamp)

	// Test from filter
	timestamp, err = GetOldestTimestamp(context.Background(), db, []eth.Address{
		trs[1].From,
	})
	require.NoError(t, err)
	require.Equal(t, int64(0), timestamp)
}
