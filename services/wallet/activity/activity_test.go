package activity

import (
	"database/sql"
	"testing"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/services/wallet/testutils"
	"github.com/status-im/status-go/services/wallet/transfer"

	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"
)

func setupTestActivityDB(t *testing.T) (db *sql.DB, close func()) {
	db, err := appdatabase.SetupTestMemorySQLDB("wallet-activity-tests")
	require.NoError(t, err)

	return db, func() {
		require.NoError(t, db.Close())
	}
}

func insertTestPendingTransaction(t *testing.T, db *sql.DB, tr *transfer.TestTransaction) {
	_, err := db.Exec(`
		INSERT INTO pending_transactions (network_id, hash, timestamp, from_address, to_address,
			symbol, gas_price, gas_limit, value, data, type, additional_data, multi_transaction_id
		) VALUES (?, ?, ?, ?, ?, 'ETH', 0, 0, ?, '', 'test', '', ?)`,
		tr.ChainID, tr.Hash, tr.Timestamp, tr.From, tr.To, tr.Value, tr.MultiTransactionID)
	require.NoError(t, err)
}

type testData struct {
	tr1          transfer.TestTransaction // index 1
	pendingTr    transfer.TestTransaction // index 2
	singletonMTr transfer.TestTransaction // index 3
	mTr          transfer.TestTransaction // index 4
	subTr        transfer.TestTransaction // index 5
	subPendingTr transfer.TestTransaction // index 6

	singletonMTID transfer.MultiTransactionIDType
	mTrID         transfer.MultiTransactionIDType
}

// Generates and adds to the DB 6 transactions. 2 transactions, 2 pending and 2 multi transactions
// There are only 4 extractable transactions and multi-transaction with timestamps 1-4. The other 2 are associated with a multi-transaction
func fillTestData(t *testing.T, db *sql.DB) (td testData) {
	trs := transfer.GenerateTestTransactions(t, db, 1, 6)
	td.tr1 = trs[0]
	transfer.InsertTestTransfer(t, db, &td.tr1)

	td.pendingTr = trs[1]
	insertTestPendingTransaction(t, db, &td.pendingTr)

	td.singletonMTr = trs[2]
	td.singletonMTID = transfer.InsertTestMultiTransaction(t, db, &td.singletonMTr)

	td.mTr = trs[3]
	td.mTrID = transfer.InsertTestMultiTransaction(t, db, &td.mTr)

	td.subTr = trs[4]
	td.subTr.MultiTransactionID = td.mTrID
	transfer.InsertTestTransfer(t, db, &td.subTr)

	td.subPendingTr = trs[5]
	td.subPendingTr.MultiTransactionID = td.mTrID
	insertTestPendingTransaction(t, db, &td.subPendingTr)
	return
}

func TestGetActivityEntriesAll(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	td := fillTestData(t, db)

	var filter Filter
	entries, err := GetActivityEntries(db, []common.Address{}, []uint64{}, filter, 0, 10)
	require.NoError(t, err)
	require.Equal(t, 4, len(entries))

	// Ensure we have the correct order
	var curTimestamp int64 = 4
	for _, entry := range entries {
		require.Equal(t, curTimestamp, entry.timestamp, "entries are sorted by timestamp; expected %d, got %d", curTimestamp, entry.timestamp)
		curTimestamp--
	}

	require.True(t, testutils.StructExistsInSlice(Entry{
		transactionType: SimpleTransactionPT,
		transaction:     &transfer.TransactionIdentity{ChainID: td.tr1.ChainID, Hash: td.tr1.Hash, Address: td.tr1.To},
		id:              td.tr1.MultiTransactionID,
		timestamp:       td.tr1.Timestamp,
		activityType:    SendAT,
	}, entries))
	require.True(t, testutils.StructExistsInSlice(Entry{
		transactionType: PendingTransactionPT,
		transaction:     &transfer.TransactionIdentity{ChainID: td.pendingTr.ChainID, Hash: td.pendingTr.Hash},
		id:              td.pendingTr.MultiTransactionID,
		timestamp:       td.pendingTr.Timestamp,
		activityType:    SendAT,
	}, entries))
	require.True(t, testutils.StructExistsInSlice(Entry{
		transactionType: MultiTransactionPT,
		transaction:     nil,
		id:              td.singletonMTID,
		timestamp:       td.singletonMTr.Timestamp,
		activityType:    SendAT,
	}, entries))
	require.True(t, testutils.StructExistsInSlice(Entry{
		transactionType: MultiTransactionPT,
		transaction:     nil,
		id:              td.mTrID,
		timestamp:       td.mTr.Timestamp,
		activityType:    SendAT,
	}, entries))

	// Ensure the sub-transactions of the multi-transactions are not returned
	require.False(t, testutils.StructExistsInSlice(Entry{
		transactionType: SimpleTransactionPT,
		transaction:     &transfer.TransactionIdentity{ChainID: td.subTr.ChainID, Hash: td.subTr.Hash, Address: td.subTr.To},
		id:              td.subTr.MultiTransactionID,
		timestamp:       td.subTr.Timestamp,
		activityType:    SendAT,
	}, entries))
	require.False(t, testutils.StructExistsInSlice(Entry{
		transactionType: PendingTransactionPT,
		transaction:     &transfer.TransactionIdentity{ChainID: td.subPendingTr.ChainID, Hash: td.subPendingTr.Hash},
		id:              td.subPendingTr.MultiTransactionID,
		timestamp:       td.subPendingTr.Timestamp,
		activityType:    SendAT,
	}, entries))
}

// TestGetActivityEntriesWithSenderFilter covers the issue with returning the same transaction
// twice when the sender and receiver have entries in the transfers table
func TestGetActivityEntriesWithSameTransactionForSenderAndReceiverInDB(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	// Add 4 extractable transactions with timestamps 1-4
	td := fillTestData(t, db)
	// Add another transaction with sender and receiver reversed
	receiverTr := td.tr1
	prevTo := receiverTr.To
	receiverTr.To = td.tr1.From
	receiverTr.From = prevTo
	transfer.InsertTestTransfer(t, db, &receiverTr)

	var filter Filter
	entries, err := GetActivityEntries(db, []common.Address{}, []uint64{}, filter, 0, 10)
	require.NoError(t, err)
	// TODO: decide how should we handle this case filter out or include it in the result
	// For now we include both. Can be changed by using UNION instead of UNION ALL in the query or by filtering out
	require.Equal(t, 5, len(entries))
}

func TestGetActivityEntriesFilterByTime(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	td := fillTestData(t, db)
	// Add 6 extractable transactions with timestamps 6-12
	trs := transfer.GenerateTestTransactions(t, db, 6, 6)
	for i := range trs {
		transfer.InsertTestTransfer(t, db, &trs[i])
	}

	// Test start only
	var filter Filter
	filter.Period.StartTimestamp = td.singletonMTr.Timestamp
	entries, err := GetActivityEntries(db, []common.Address{}, []uint64{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 8, len(entries))
	// Check start and end content
	require.Equal(t, Entry{
		transactionType: SimpleTransactionPT,
		transaction:     &transfer.TransactionIdentity{ChainID: trs[5].ChainID, Hash: trs[5].Hash, Address: trs[5].To},
		id:              0,
		timestamp:       trs[5].Timestamp,
		activityType:    SendAT,
	}, entries[0])
	require.Equal(t, Entry{
		transactionType: MultiTransactionPT,
		transaction:     nil,
		id:              td.singletonMTID,
		timestamp:       td.singletonMTr.Timestamp,
		activityType:    SendAT,
	}, entries[7])

	// Test complete interval
	filter.Period.EndTimestamp = trs[2].Timestamp
	entries, err = GetActivityEntries(db, []common.Address{}, []uint64{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 5, len(entries))
	// Check start and end content
	require.Equal(t, Entry{
		transactionType: SimpleTransactionPT,
		transaction:     &transfer.TransactionIdentity{ChainID: trs[2].ChainID, Hash: trs[2].Hash, Address: trs[2].To},
		id:              0,
		timestamp:       trs[2].Timestamp,
		activityType:    SendAT,
	}, entries[0])
	require.Equal(t, Entry{
		transactionType: MultiTransactionPT,
		transaction:     nil,
		id:              td.singletonMTID,
		timestamp:       td.singletonMTr.Timestamp,
		activityType:    SendAT,
	}, entries[4])

	// Test end only
	filter.Period.StartTimestamp = 0
	entries, err = GetActivityEntries(db, []common.Address{}, []uint64{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 7, len(entries))
	// Check start and end content
	require.Equal(t, Entry{
		transactionType: SimpleTransactionPT,
		transaction:     &transfer.TransactionIdentity{ChainID: trs[2].ChainID, Hash: trs[2].Hash, Address: trs[2].To},
		id:              0,
		timestamp:       trs[2].Timestamp,
		activityType:    SendAT,
	}, entries[0])
	require.Equal(t, Entry{
		transactionType: SimpleTransactionPT,
		transaction:     &transfer.TransactionIdentity{ChainID: td.tr1.ChainID, Hash: td.tr1.Hash, Address: td.tr1.To},
		id:              0,
		timestamp:       td.tr1.Timestamp,
		activityType:    SendAT,
	}, entries[6])
}

func TestGetActivityEntriesCheckOffsetAndLimit(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	// Add 10 extractable transactions with timestamps 1-10
	trs := transfer.GenerateTestTransactions(t, db, 1, 10)
	for i := range trs {
		transfer.InsertTestTransfer(t, db, &trs[i])
	}

	var filter Filter
	// Get all
	entries, err := GetActivityEntries(db, []common.Address{}, []uint64{}, filter, 0, 5)
	require.NoError(t, err)
	require.Equal(t, 5, len(entries))

	// Get time based interval
	filter.Period.StartTimestamp = trs[2].Timestamp
	filter.Period.EndTimestamp = trs[8].Timestamp
	entries, err = GetActivityEntries(db, []common.Address{}, []uint64{}, filter, 0, 3)
	require.NoError(t, err)
	require.Equal(t, 3, len(entries))
	// Check start and end content
	require.Equal(t, Entry{
		transactionType: SimpleTransactionPT,
		transaction:     &transfer.TransactionIdentity{ChainID: trs[8].ChainID, Hash: trs[8].Hash, Address: trs[8].To},
		id:              0,
		timestamp:       trs[8].Timestamp,
		activityType:    SendAT,
	}, entries[0])
	require.Equal(t, Entry{
		transactionType: SimpleTransactionPT,
		transaction:     &transfer.TransactionIdentity{ChainID: trs[6].ChainID, Hash: trs[6].Hash, Address: trs[6].To},
		id:              0,
		timestamp:       trs[6].Timestamp,
		activityType:    SendAT,
	}, entries[2])

	// Move window 2 entries forward
	entries, err = GetActivityEntries(db, []common.Address{}, []uint64{}, filter, 2, 3)
	require.NoError(t, err)
	require.Equal(t, 3, len(entries))
	// Check start and end content
	require.Equal(t, Entry{
		transactionType: SimpleTransactionPT,
		transaction:     &transfer.TransactionIdentity{ChainID: trs[6].ChainID, Hash: trs[6].Hash, Address: trs[6].To},
		id:              0,
		timestamp:       trs[6].Timestamp,
		activityType:    SendAT,
	}, entries[0])
	require.Equal(t, Entry{
		transactionType: SimpleTransactionPT,
		transaction:     &transfer.TransactionIdentity{ChainID: trs[4].ChainID, Hash: trs[4].Hash, Address: trs[4].To},
		id:              0,
		timestamp:       trs[4].Timestamp,
		activityType:    SendAT,
	}, entries[2])

	// Move window 4 more entries to test filter cap
	entries, err = GetActivityEntries(db, []common.Address{}, []uint64{}, filter, 6, 3)
	require.NoError(t, err)
	require.Equal(t, 1, len(entries))
	// Check start and end content
	require.Equal(t, Entry{
		transactionType: SimpleTransactionPT,
		transaction:     &transfer.TransactionIdentity{ChainID: trs[2].ChainID, Hash: trs[2].Hash, Address: trs[2].To},
		id:              0,
		timestamp:       trs[2].Timestamp,
		activityType:    SendAT,
	}, entries[0])
}

func TestGetActivityEntriesFilterByType(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	// Adds 4 extractable transactions
	fillTestData(t, db)
	// Add 6 extractable transactions: one MultiTransactionSwap, two MultiTransactionBridge rest Send
	trs := transfer.GenerateTestTransactions(t, db, 6, 6)
	trs[1].MultiTransactionType = transfer.MultiTransactionBridge
	trs[3].MultiTransactionType = transfer.MultiTransactionSwap
	trs[5].MultiTransactionType = transfer.MultiTransactionBridge

	for i := range trs {
		if trs[i].MultiTransactionType != transfer.MultiTransactionSend {
			transfer.InsertTestMultiTransaction(t, db, &trs[i])
		} else {
			transfer.InsertTestTransfer(t, db, &trs[i])
		}
	}

	// Test filtering out without address involved
	var filter Filter
	// TODO: add more types to cover all cases
	filter.Types = []Type{SendAT, SwapAT}
	entries, err := GetActivityEntries(db, []common.Address{}, []uint64{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 8, len(entries))
	swapCount := 0
	sendCount := 0
	for _, entry := range entries {
		if entry.activityType == SendAT {
			sendCount++
		}
		if entry.activityType == SwapAT {
			swapCount++
		}
	}
	require.Equal(t, 7, sendCount)
	require.Equal(t, 1, swapCount)

	// Test filtering out with address involved
	filter.Types = []Type{BridgeAT, ReceiveAT}
	// Include one "to" from transfers to be detected as receive
	addresses := []common.Address{trs[0].To, trs[1].To, trs[2].From, trs[3].From, trs[5].From}
	entries, err = GetActivityEntries(db, addresses, []uint64{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 3, len(entries))
	bridgeCount := 0
	receiveCount := 0
	for _, entry := range entries {
		if entry.activityType == BridgeAT {
			bridgeCount++
		}
		if entry.activityType == ReceiveAT {
			receiveCount++
		}
	}
	require.Equal(t, 2, bridgeCount)
	require.Equal(t, 1, receiveCount)
}

func TestGetActivityEntriesFilterByAddress(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	// Adds 4 extractable transactions
	td := fillTestData(t, db)
	// Add 6 extractable transactions: one MultiTransactionSwap, two MultiTransactionBridge rest Send
	trs := transfer.GenerateTestTransactions(t, db, 7, 6)
	for i := range trs {
		transfer.InsertTestTransfer(t, db, &trs[i])
	}

	var filter Filter
	addressesFilter := []common.Address{td.mTr.To, trs[1].From, trs[4].To}
	entries, err := GetActivityEntries(db, addressesFilter, []uint64{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 3, len(entries))
	require.Equal(t, Entry{
		transactionType: SimpleTransactionPT,
		transaction:     &transfer.TransactionIdentity{ChainID: trs[4].ChainID, Hash: trs[4].Hash, Address: trs[4].To},
		id:              0,
		timestamp:       trs[4].Timestamp,
		activityType:    ReceiveAT,
	}, entries[0])
	require.Equal(t, Entry{
		transactionType: SimpleTransactionPT,
		transaction:     &transfer.TransactionIdentity{ChainID: trs[1].ChainID, Hash: trs[1].Hash, Address: trs[1].To},
		id:              0,
		timestamp:       trs[1].Timestamp,
		activityType:    SendAT,
	}, entries[1])
	require.Equal(t, Entry{
		transactionType: MultiTransactionPT,
		transaction:     nil,
		id:              td.mTrID,
		timestamp:       td.mTr.Timestamp,
		activityType:    SendAT,
	}, entries[2])
}
