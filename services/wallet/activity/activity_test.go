package activity

import (
	"context"
	"database/sql"
	"math/big"
	"testing"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/testutils"
	"github.com/status-im/status-go/services/wallet/transfer"

	eth "github.com/ethereum/go-ethereum/common"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/stretchr/testify/require"
)

func setupTestActivityDB(t *testing.T) (db *sql.DB, close func()) {
	db, err := appdatabase.SetupTestMemorySQLDB("wallet-activity-tests")
	require.NoError(t, err)

	return db, func() {
		require.NoError(t, db.Close())
	}
}

type testData struct {
	tr1               transfer.TestTransfer // index 1
	pendingTr         transfer.TestTransfer // index 2
	multiTx1Tr1       transfer.TestTransfer // index 3
	multiTx2Tr1       transfer.TestTransfer // index 4
	multiTx1Tr2       transfer.TestTransfer // index 5
	multiTx2Tr2       transfer.TestTransfer // index 6
	multiTx2PendingTr transfer.TestTransfer // index 7

	multiTx1   transfer.TestMultiTransaction
	multiTx1ID transfer.MultiTransactionIDType

	multiTx2   transfer.TestMultiTransaction
	multiTx2ID transfer.MultiTransactionIDType

	nextIndex int
}

func mockTestAccountsWithAddresses(t *testing.T, db *sql.DB, addresses []eth_common.Address) {
	mockedAccounts := []*accounts.Account{}
	for _, address := range addresses {
		mockedAccounts = append(mockedAccounts, &accounts.Account{
			Address: types.Address(address),
			Type:    accounts.AccountTypeWatch,
		})
	}
	accounts.MockTestAccounts(t, db, mockedAccounts)
}

// Generates and adds to the DB 7 transfers and 2 multitransactions.
// There are only 4 extractable activity entries (transactions + multi-transactions) with timestamps 1-4. The others are associated with a multi-transaction
func fillTestData(t *testing.T, db *sql.DB) (td testData, fromAddresses, toAddresses []eth_common.Address) {
	trs, fromAddresses, toAddresses := transfer.GenerateTestTransfers(t, db, 1, 7)

	// Plain transfer
	td.tr1 = trs[0]
	transfer.InsertTestTransfer(t, db, &td.tr1)

	// Pending transfer
	td.pendingTr = trs[1]
	transfer.InsertTestPendingTransaction(t, db, &td.pendingTr)

	// Send Multitransaction containing 2 x Plain transfers
	td.multiTx1Tr1 = trs[2]
	td.multiTx1Tr2 = trs[4]

	td.multiTx1Tr1.Token = testutils.SntSymbol

	td.multiTx1 = transfer.GenerateTestSendMultiTransaction(td.multiTx1Tr1)
	td.multiTx1.ToToken = testutils.DaiSymbol
	td.multiTx1ID = transfer.InsertTestMultiTransaction(t, db, &td.multiTx1)

	td.multiTx1Tr1.MultiTransactionID = td.multiTx1ID
	transfer.InsertTestTransfer(t, db, &td.multiTx1Tr1)

	td.multiTx1Tr2.MultiTransactionID = td.multiTx1ID
	transfer.InsertTestTransfer(t, db, &td.multiTx1Tr2)

	// Send Multitransaction containing 2 x Plain transfers + 1 x Pending transfer
	td.multiTx2Tr1 = trs[3]
	td.multiTx2Tr2 = trs[5]
	td.multiTx2PendingTr = trs[6]

	td.multiTx2 = transfer.GenerateTestSendMultiTransaction(td.multiTx2Tr1)
	td.multiTx1.ToToken = testutils.SntSymbol
	td.multiTx2ID = transfer.InsertTestMultiTransaction(t, db, &td.multiTx2)

	td.multiTx2Tr1.MultiTransactionID = td.multiTx2ID
	transfer.InsertTestTransfer(t, db, &td.multiTx2Tr1)

	td.multiTx2Tr2.MultiTransactionID = td.multiTx2ID
	transfer.InsertTestTransfer(t, db, &td.multiTx2Tr2)

	td.multiTx2PendingTr.MultiTransactionID = td.multiTx2ID
	transfer.InsertTestPendingTransaction(t, db, &td.multiTx2PendingTr)

	td.nextIndex = 8
	return td, fromAddresses, toAddresses
}

func TestGetActivityEntriesAll(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	td, fromAddresses, toAddresses := fillTestData(t, db)

	var filter Filter
	entries, err := getActivityEntries(context.Background(), db, append(toAddresses, fromAddresses...), []common.ChainID{}, filter, 0, 10)
	require.NoError(t, err)
	require.Equal(t, 4, len(entries))

	// Ensure we have the correct order
	var curTimestamp int64 = 4
	for _, entry := range entries {
		require.Equal(t, curTimestamp, entry.timestamp, "entries are sorted by timestamp; expected %d, got %d", curTimestamp, entry.timestamp)
		curTimestamp--
	}

	require.True(t, testutils.StructExistsInSlice(Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: td.tr1.ChainID, Hash: td.tr1.Hash, Address: td.tr1.From},
		id:             td.tr1.MultiTransactionID,
		timestamp:      td.tr1.Timestamp,
		activityType:   SendAT,
		activityStatus: CompleteAS,
		tokenType:      AssetTT,
		amountOut:      (*hexutil.Big)(big.NewInt(td.tr1.Value)),
		amountIn:       (*hexutil.Big)(big.NewInt(0)),
	}, entries))
	require.True(t, testutils.StructExistsInSlice(Entry{
		payloadType:    PendingTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: td.pendingTr.ChainID, Hash: td.pendingTr.Hash},
		id:             td.pendingTr.MultiTransactionID,
		timestamp:      td.pendingTr.Timestamp,
		activityType:   SendAT,
		activityStatus: PendingAS,
		tokenType:      AssetTT,
		amountOut:      (*hexutil.Big)(big.NewInt(td.pendingTr.Value)),
		amountIn:       (*hexutil.Big)(big.NewInt(0)),
	}, entries))
	require.True(t, testutils.StructExistsInSlice(Entry{
		payloadType:    MultiTransactionPT,
		transaction:    nil,
		id:             td.multiTx1ID,
		timestamp:      td.multiTx1.Timestamp,
		activityType:   SendAT,
		activityStatus: CompleteAS,
		tokenType:      AssetTT,
		amountOut:      (*hexutil.Big)(big.NewInt(td.multiTx1.FromAmount)),
		amountIn:       (*hexutil.Big)(big.NewInt(td.multiTx1.ToAmount)),
	}, entries))
	require.True(t, testutils.StructExistsInSlice(Entry{
		payloadType:    MultiTransactionPT,
		transaction:    nil,
		id:             td.multiTx2ID,
		timestamp:      td.multiTx2.Timestamp,
		activityType:   SendAT,
		activityStatus: PendingAS,
		tokenType:      AssetTT,
		amountOut:      (*hexutil.Big)(big.NewInt(td.multiTx2.FromAmount)),
		amountIn:       (*hexutil.Big)(big.NewInt(td.multiTx2.ToAmount)),
	}, entries))

	// Ensure the sub-transactions of the multi-transactions are not returned
	require.False(t, testutils.StructExistsInSlice(Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: td.multiTx1Tr1.ChainID, Hash: td.multiTx1Tr1.Hash, Address: td.multiTx1Tr1.To},
		id:             td.multiTx1Tr1.MultiTransactionID,
		timestamp:      td.multiTx1Tr1.Timestamp,
		activityType:   SendAT,
		activityStatus: CompleteAS,
		tokenType:      AssetTT,
		amountOut:      (*hexutil.Big)(big.NewInt(td.multiTx1Tr1.Value)),
		amountIn:       (*hexutil.Big)(big.NewInt(0)),
	}, entries))
	require.False(t, testutils.StructExistsInSlice(Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: td.multiTx1Tr2.ChainID, Hash: td.multiTx1Tr2.Hash, Address: td.multiTx1Tr2.To},
		id:             td.multiTx1Tr2.MultiTransactionID,
		timestamp:      td.multiTx1Tr2.Timestamp,
		activityType:   SendAT,
		activityStatus: CompleteAS,
		tokenType:      AssetTT,
		amountOut:      (*hexutil.Big)(big.NewInt(td.multiTx1Tr2.Value)),
		amountIn:       (*hexutil.Big)(big.NewInt(0)),
	}, entries))
	require.False(t, testutils.StructExistsInSlice(Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: td.multiTx2Tr1.ChainID, Hash: td.multiTx2Tr1.Hash, Address: td.multiTx2Tr1.To},
		id:             td.multiTx2Tr1.MultiTransactionID,
		timestamp:      td.multiTx2Tr1.Timestamp,
		activityType:   SendAT,
		activityStatus: CompleteAS,
		tokenType:      AssetTT,
		amountOut:      (*hexutil.Big)(big.NewInt(td.multiTx2Tr1.Value)),
		amountIn:       (*hexutil.Big)(big.NewInt(0)),
	}, entries))
	require.False(t, testutils.StructExistsInSlice(Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: td.multiTx2Tr2.ChainID, Hash: td.multiTx2Tr2.Hash, Address: td.multiTx2Tr2.To},
		id:             td.multiTx2Tr2.MultiTransactionID,
		timestamp:      td.multiTx2Tr2.Timestamp,
		activityType:   SendAT,
		activityStatus: CompleteAS,
		tokenType:      AssetTT,
		amountOut:      (*hexutil.Big)(big.NewInt(td.multiTx2Tr2.Value)),
		amountIn:       (*hexutil.Big)(big.NewInt(0)),
	}, entries))
	require.False(t, testutils.StructExistsInSlice(Entry{
		payloadType:    PendingTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: td.multiTx2PendingTr.ChainID, Hash: td.multiTx2PendingTr.Hash},
		id:             td.multiTx2PendingTr.MultiTransactionID,
		timestamp:      td.multiTx2PendingTr.Timestamp,
		activityType:   SendAT,
		activityStatus: PendingAS,
		tokenType:      AssetTT,
		amountOut:      (*hexutil.Big)(big.NewInt(td.multiTx2PendingTr.Value)),
		amountIn:       (*hexutil.Big)(big.NewInt(0)),
	}, entries))
}

// TestGetActivityEntriesWithSenderFilter covers the issue with returning the same transaction
// twice when the sender and receiver have entries in the transfers table
func TestGetActivityEntriesWithSameTransactionForSenderAndReceiverInDB(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	// Add 4 extractable transactions with timestamps 1-4
	td, fromAddresses, toAddresses := fillTestData(t, db)

	mockTestAccountsWithAddresses(t, db, append(fromAddresses, toAddresses...))

	// Add another transaction with sender and receiver reversed
	receiverTr := td.tr1
	prevTo := receiverTr.To
	receiverTr.To = td.tr1.From
	receiverTr.From = prevTo

	// TODO: test also when there is a transaction in the other direction

	// Ensure they are the oldest transactions (last in the list) and we have a consistent order
	receiverTr.Timestamp--
	transfer.InsertTestTransfer(t, db, &receiverTr)

	var filter Filter
	entries, err := getActivityEntries(context.Background(), db, []eth.Address{td.tr1.From, receiverTr.From}, []common.ChainID{}, filter, 0, 10)
	require.NoError(t, err)
	require.Equal(t, 2, len(entries))

	// Check that the transaction are labeled alternatively as send and receive
	require.Equal(t, ReceiveAT, entries[1].activityType)
	require.NotEqual(t, eth.Address{}, entries[1].transaction.Address)
	require.Equal(t, receiverTr.To, entries[1].transaction.Address)

	require.Equal(t, SendAT, entries[0].activityType)
	require.NotEqual(t, eth.Address{}, entries[0].transaction.Address)
	require.Equal(t, td.tr1.From, entries[0].transaction.Address)

	entries, err = getActivityEntries(context.Background(), db, []eth.Address{}, []common.ChainID{}, filter, 0, 10)
	require.NoError(t, err)
	require.Equal(t, 5, len(entries))

	// Check that the transaction are labeled alternatively as send and receive
	require.Equal(t, ReceiveAT, entries[4].activityType)
	require.Equal(t, SendAT, entries[3].activityType)
}

func TestGetActivityEntriesFilterByTime(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	td, fromTds, toTds := fillTestData(t, db)

	// Add 6 extractable transactions with timestamps 6-12
	trs, fromTrs, toTrs := transfer.GenerateTestTransfers(t, db, td.nextIndex, 6)
	for i := range trs {
		transfer.InsertTestTransfer(t, db, &trs[i])
	}

	mockTestAccountsWithAddresses(t, db, append(append(append(fromTds, toTds...), fromTrs...), toTrs...))

	// Test start only
	var filter Filter
	filter.Period.StartTimestamp = td.multiTx1.Timestamp
	filter.Period.EndTimestamp = NoLimitTimestampForPeriod
	entries, err := getActivityEntries(context.Background(), db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 8, len(entries))
	// Check start and end content
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: trs[5].ChainID, Hash: trs[5].Hash, Address: trs[5].From},
		id:             0,
		timestamp:      trs[5].Timestamp,
		activityType:   SendAT,
		activityStatus: CompleteAS,
		tokenType:      AssetTT,
		amountOut:      (*hexutil.Big)(big.NewInt(trs[5].Value)),
		amountIn:       (*hexutil.Big)(big.NewInt(0)),
	}, entries[0])
	require.Equal(t, Entry{
		payloadType:    MultiTransactionPT,
		transaction:    nil,
		id:             td.multiTx1ID,
		timestamp:      td.multiTx1.Timestamp,
		activityType:   SendAT,
		activityStatus: CompleteAS,
		tokenType:      AssetTT,
		amountOut:      (*hexutil.Big)(big.NewInt(td.multiTx1.FromAmount)),
		amountIn:       (*hexutil.Big)(big.NewInt(td.multiTx1.ToAmount)),
	}, entries[7])

	// Test complete interval
	filter.Period.EndTimestamp = trs[2].Timestamp
	entries, err = getActivityEntries(context.Background(), db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 5, len(entries))
	// Check start and end content
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: trs[2].ChainID, Hash: trs[2].Hash, Address: trs[2].From},
		id:             0,
		timestamp:      trs[2].Timestamp,
		activityType:   SendAT,
		activityStatus: CompleteAS,
		tokenType:      AssetTT,
		amountOut:      (*hexutil.Big)(big.NewInt(trs[2].Value)),
		amountIn:       (*hexutil.Big)(big.NewInt(0)),
	}, entries[0])
	require.Equal(t, Entry{
		payloadType:    MultiTransactionPT,
		transaction:    nil,
		id:             td.multiTx1ID,
		timestamp:      td.multiTx1.Timestamp,
		activityType:   SendAT,
		activityStatus: CompleteAS,
		tokenType:      AssetTT,
		amountOut:      (*hexutil.Big)(big.NewInt(td.multiTx1.FromAmount)),
		amountIn:       (*hexutil.Big)(big.NewInt(td.multiTx1.ToAmount)),
	}, entries[4])

	// Test end only
	filter.Period.StartTimestamp = NoLimitTimestampForPeriod
	entries, err = getActivityEntries(context.Background(), db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 7, len(entries))
	// Check start and end content
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: trs[2].ChainID, Hash: trs[2].Hash, Address: trs[2].From},
		id:             0,
		timestamp:      trs[2].Timestamp,
		activityType:   SendAT,
		activityStatus: CompleteAS,
		tokenType:      AssetTT,
		amountOut:      (*hexutil.Big)(big.NewInt(trs[2].Value)),
		amountIn:       (*hexutil.Big)(big.NewInt(0)),
	}, entries[0])
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: td.tr1.ChainID, Hash: td.tr1.Hash, Address: td.tr1.From},
		id:             0,
		timestamp:      td.tr1.Timestamp,
		activityType:   SendAT,
		activityStatus: CompleteAS,
		tokenType:      AssetTT,
		amountOut:      (*hexutil.Big)(big.NewInt(td.tr1.Value)),
		amountIn:       (*hexutil.Big)(big.NewInt(0)),
	}, entries[6])
}

func TestGetActivityEntriesCheckOffsetAndLimit(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	// Add 10 extractable transactions with timestamps 1-10
	trs, fromTrs, toTrs := transfer.GenerateTestTransfers(t, db, 1, 10)
	for i := range trs {
		transfer.InsertTestTransfer(t, db, &trs[i])
	}

	mockTestAccountsWithAddresses(t, db, append(fromTrs, toTrs...))

	var filter Filter
	// Get all
	entries, err := getActivityEntries(context.Background(), db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 5)
	require.NoError(t, err)
	require.Equal(t, 5, len(entries))

	// Get time based interval
	filter.Period.StartTimestamp = trs[2].Timestamp
	filter.Period.EndTimestamp = trs[8].Timestamp
	entries, err = getActivityEntries(context.Background(), db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 3)
	require.NoError(t, err)
	require.Equal(t, 3, len(entries))
	// Check start and end content
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: trs[8].ChainID, Hash: trs[8].Hash, Address: trs[8].From},
		id:             0,
		timestamp:      trs[8].Timestamp,
		activityType:   SendAT,
		activityStatus: CompleteAS,
		tokenType:      AssetTT,
		amountOut:      (*hexutil.Big)(big.NewInt(trs[8].Value)),
		amountIn:       (*hexutil.Big)(big.NewInt(0)),
	}, entries[0])
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: trs[6].ChainID, Hash: trs[6].Hash, Address: trs[6].From},
		id:             0,
		timestamp:      trs[6].Timestamp,
		activityType:   SendAT,
		activityStatus: CompleteAS,
		tokenType:      AssetTT,
		amountOut:      (*hexutil.Big)(big.NewInt(trs[6].Value)),
		amountIn:       (*hexutil.Big)(big.NewInt(0)),
	}, entries[2])

	// Move window 2 entries forward
	entries, err = getActivityEntries(context.Background(), db, []eth_common.Address{}, []common.ChainID{}, filter, 2, 3)
	require.NoError(t, err)
	require.Equal(t, 3, len(entries))
	// Check start and end content
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: trs[6].ChainID, Hash: trs[6].Hash, Address: trs[6].From},
		id:             0,
		timestamp:      trs[6].Timestamp,
		activityType:   SendAT,
		activityStatus: CompleteAS,
		tokenType:      AssetTT,
		amountOut:      (*hexutil.Big)(big.NewInt(trs[6].Value)),
		amountIn:       (*hexutil.Big)(big.NewInt(0)),
	}, entries[0])
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: trs[4].ChainID, Hash: trs[4].Hash, Address: trs[4].From},
		id:             0,
		timestamp:      trs[4].Timestamp,
		activityType:   SendAT,
		activityStatus: CompleteAS,
		tokenType:      AssetTT,
		amountOut:      (*hexutil.Big)(big.NewInt(trs[4].Value)),
		amountIn:       (*hexutil.Big)(big.NewInt(0)),
	}, entries[2])

	// Move window 4 more entries to test filter cap
	entries, err = getActivityEntries(context.Background(), db, []eth_common.Address{}, []common.ChainID{}, filter, 6, 3)
	require.NoError(t, err)
	require.Equal(t, 1, len(entries))
	// Check start and end content
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: trs[2].ChainID, Hash: trs[2].Hash, Address: trs[2].From},
		id:             0,
		timestamp:      trs[2].Timestamp,
		activityType:   SendAT,
		activityStatus: CompleteAS,
		tokenType:      AssetTT,
		amountOut:      (*hexutil.Big)(big.NewInt(trs[2].Value)),
		amountIn:       (*hexutil.Big)(big.NewInt(0)),
	}, entries[0])
}

func countTypes(entries []Entry) (sendCount, receiveCount, swapCount, buyCount, bridgeCount int) {
	for _, entry := range entries {
		switch entry.activityType {
		case SendAT:
			sendCount++
		case ReceiveAT:
			receiveCount++
		case SwapAT:
			swapCount++
		case BuyAT:
			buyCount++
		case BridgeAT:
			bridgeCount++
		}
	}
	return
}

func TestGetActivityEntriesFilterByType(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	// Adds 4 extractable transactions
	td, _, _ := fillTestData(t, db)
	// Add 5 extractable transactions: one MultiTransactionSwap, two MultiTransactionBridge and two MultiTransactionSend
	multiTxs := make([]transfer.TestMultiTransaction, 5)
	trs, _, _ := transfer.GenerateTestTransfers(t, db, td.nextIndex, len(multiTxs)*2)
	multiTxs[0] = transfer.GenerateTestBridgeMultiTransaction(trs[0], trs[1])
	multiTxs[1] = transfer.GenerateTestSwapMultiTransaction(trs[2], testutils.SntSymbol, 100) // trs[3]
	multiTxs[2] = transfer.GenerateTestSendMultiTransaction(trs[4])                           // trs[5]
	multiTxs[3] = transfer.GenerateTestBridgeMultiTransaction(trs[6], trs[7])
	multiTxs[4] = transfer.GenerateTestSendMultiTransaction(trs[8]) // trs[9]

	var lastMT transfer.MultiTransactionIDType
	for i := range trs {
		if i%2 == 0 {
			lastMT = transfer.InsertTestMultiTransaction(t, db, &multiTxs[i/2])
		}
		trs[i].MultiTransactionID = lastMT
		transfer.InsertTestTransfer(t, db, &trs[i])
	}

	// Test filtering out without address involved
	var filter Filter

	filter.Types = allActivityTypesFilter()
	// Set tr1 to Receive and pendingTr to Send; rest of two MT remain default Send
	addresses := []eth_common.Address{td.tr1.To, td.pendingTr.From, td.multiTx1.FromAddress, td.multiTx2.FromAddress, trs[0].From, trs[2].From, trs[4].From, trs[6].From, trs[8].From}
	entries, err := getActivityEntries(context.Background(), db, addresses, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 9, len(entries))

	filter.Types = []Type{SendAT, SwapAT}
	entries, err = getActivityEntries(context.Background(), db, addresses, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	// 3 from td Send + 2 trs MT Send + 1 (swap)
	require.Equal(t, 6, len(entries))

	sendCount, receiveCount, swapCount, _, bridgeCount := countTypes(entries)

	require.Equal(t, 5, sendCount)
	require.Equal(t, 0, receiveCount)
	require.Equal(t, 1, swapCount)
	require.Equal(t, 0, bridgeCount)

	filter.Types = []Type{BridgeAT, ReceiveAT}
	entries, err = getActivityEntries(context.Background(), db, addresses, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 3, len(entries))

	sendCount, receiveCount, swapCount, _, bridgeCount = countTypes(entries)
	require.Equal(t, 0, sendCount)
	require.Equal(t, 1, receiveCount)
	require.Equal(t, 0, swapCount)
	require.Equal(t, 2, bridgeCount)
}

func TestGetActivityEntriesFilterByAddresses(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	// Adds 4 extractable transactions
	td, fromTds, toTds := fillTestData(t, db)
	trs, fromTrs, toTrs := transfer.GenerateTestTransfers(t, db, td.nextIndex, 6)
	for i := range trs {
		transfer.InsertTestTransfer(t, db, &trs[i])
	}

	mockTestAccountsWithAddresses(t, db, append(append(append(fromTds, toTds...), fromTrs...), toTrs...))

	var filter Filter

	addressesFilter := allAddressesFilter()
	entries, err := getActivityEntries(context.Background(), db, addressesFilter, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 10, len(entries))

	addressesFilter = []eth_common.Address{td.multiTx2.ToAddress, trs[1].From, trs[4].To}
	entries, err = getActivityEntries(context.Background(), db, addressesFilter, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 3, len(entries))
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: trs[4].ChainID, Hash: trs[4].Hash, Address: trs[4].To},
		id:             0,
		timestamp:      trs[4].Timestamp,
		activityType:   ReceiveAT,
		activityStatus: CompleteAS,
		tokenType:      AssetTT,
		amountOut:      (*hexutil.Big)(big.NewInt(0)),
		amountIn:       (*hexutil.Big)(big.NewInt(trs[4].Value)),
	}, entries[0])
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: trs[1].ChainID, Hash: trs[1].Hash, Address: trs[1].From},
		id:             0,
		timestamp:      trs[1].Timestamp,
		activityType:   SendAT,
		activityStatus: CompleteAS,
		tokenType:      AssetTT,
		amountOut:      (*hexutil.Big)(big.NewInt(trs[1].Value)),
		amountIn:       (*hexutil.Big)(big.NewInt(0)),
	}, entries[1])
	require.Equal(t, Entry{
		payloadType:    MultiTransactionPT,
		transaction:    nil,
		id:             td.multiTx2ID,
		timestamp:      td.multiTx2.Timestamp,
		activityType:   SendAT,
		activityStatus: PendingAS,
		tokenType:      AssetTT,
		amountOut:      (*hexutil.Big)(big.NewInt(td.multiTx2.FromAmount)),
		amountIn:       (*hexutil.Big)(big.NewInt(td.multiTx2.ToAmount)),
	}, entries[2])
}

func TestGetActivityEntriesFilterByStatus(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	// Adds 4 extractable transactions: 1 T, 1 T pending, 1 MT pending, 1 MT with 2xT success
	td, fromTds, toTds := fillTestData(t, db)
	// Add 7 extractable transactions: 1 pending, 1 Tr failed, 1 MT failed, 4 success
	trs, fromTrs, toTrs := transfer.GenerateTestTransfers(t, db, td.nextIndex, 7)
	multiTx := transfer.GenerateTestSendMultiTransaction(trs[6])
	failedMTID := transfer.InsertTestMultiTransaction(t, db, &multiTx)
	trs[6].MultiTransactionID = failedMTID
	for i := range trs {
		if i == 1 {
			transfer.InsertTestPendingTransaction(t, db, &trs[i])
		} else {
			trs[i].Success = i != 3 && i != 6
			transfer.InsertTestTransfer(t, db, &trs[i])
		}
	}

	mockTestAccountsWithAddresses(t, db, append(append(append(fromTds, toTds...), fromTrs...), toTrs...))

	var filter Filter
	filter.Statuses = allActivityStatusesFilter()
	entries, err := getActivityEntries(context.Background(), db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 11, len(entries))

	filter.Statuses = []Status{PendingAS}
	entries, err = getActivityEntries(context.Background(), db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 3, len(entries))
	require.Equal(t, td.pendingTr.Hash, entries[2].transaction.Hash)
	require.Equal(t, td.multiTx2ID, entries[1].id)
	require.Equal(t, trs[1].Hash, entries[0].transaction.Hash)

	filter.Statuses = []Status{FailedAS}
	entries, err = getActivityEntries(context.Background(), db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 2, len(entries))

	filter.Statuses = []Status{CompleteAS}
	entries, err = getActivityEntries(context.Background(), db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 6, len(entries))

	// Finalized is treated as Complete, would need dynamic blockchain status to track the Finalized level
	filter.Statuses = []Status{FinalizedAS}
	entries, err = getActivityEntries(context.Background(), db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 6, len(entries))

	// Combined filter
	filter.Statuses = []Status{FailedAS, PendingAS}
	entries, err = getActivityEntries(context.Background(), db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 5, len(entries))
}

func TestGetActivityEntriesFilterByTokenType(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	// Adds 4 extractable transactions 2 transactions ETH, one MT SNT to DAI and another MT ETH to SNT
	td, fromTds, toTds := fillTestData(t, db)
	// Add 6 extractable transactions with USDC (only erc20 as type in DB)
	trs, fromTrs, toTrs := transfer.GenerateTestTransfers(t, db, td.nextIndex, 6)
	for i := range trs {
		trs[i].Token = "USDC"
		transfer.InsertTestTransfer(t, db, &trs[i])
	}

	mockTestAccountsWithAddresses(t, db, append(append(append(fromTds, toTds...), fromTrs...), toTrs...))

	var filter Filter
	filter.Tokens = noAssetsFilter()
	entries, err := getActivityEntries(context.Background(), db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 0, len(entries))

	filter.Tokens = allTokensFilter()
	entries, err = getActivityEntries(context.Background(), db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 10, len(entries))

	// Regression when collectibles is nil
	filter.Tokens = Tokens{[]TokenCode{}, nil, []TokenType{}}
	entries, err = getActivityEntries(context.Background(), db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 10, len(entries))

	filter.Tokens = Tokens{Assets: []TokenCode{"ETH"}, EnabledTypes: []TokenType{AssetTT}}
	entries, err = getActivityEntries(context.Background(), db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 3, len(entries))

	// TODO: update tests after adding token type to transfers
	filter.Tokens = Tokens{Assets: []TokenCode{"USDC", "DAI"}, EnabledTypes: []TokenType{AssetTT}}
	entries, err = getActivityEntries(context.Background(), db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 1, len(entries))

	// Regression when EnabledTypes ar empty
	filter.Tokens = Tokens{Assets: []TokenCode{"USDC", "DAI"}, EnabledTypes: []TokenType{}}
	entries, err = getActivityEntries(context.Background(), db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 1, len(entries))
}

func TestGetActivityEntriesFilterByToAddresses(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	// Adds 4 extractable transactions
	td, fromTds, toTds := fillTestData(t, db)
	// Add 6 extractable transactions
	trs, fromTrs, toTrs := transfer.GenerateTestTransfers(t, db, td.nextIndex, 6)
	for i := range trs {
		transfer.InsertTestTransfer(t, db, &trs[i])
	}

	mockTestAccountsWithAddresses(t, db, append(append(append(fromTds, toTds...), fromTrs...), toTrs...))

	var filter Filter
	filter.CounterpartyAddresses = allAddressesFilter()
	entries, err := getActivityEntries(context.Background(), db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 10, len(entries))

	filter.CounterpartyAddresses = []eth_common.Address{eth_common.HexToAddress("0x567890")}
	entries, err = getActivityEntries(context.Background(), db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 0, len(entries))

	filter.CounterpartyAddresses = []eth_common.Address{td.pendingTr.To, td.multiTx2.ToAddress, trs[3].To}
	entries, err = getActivityEntries(context.Background(), db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 3, len(entries))

	filter.CounterpartyAddresses = []eth_common.Address{td.tr1.To, td.pendingTr.From, trs[3].From, trs[5].To}
	entries, err = getActivityEntries(context.Background(), db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 2, len(entries))
}
func TestGetActivityEntriesFilterByNetworks(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	// Adds 4 extractable transactions
	td, fromTds, toTds := fillTestData(t, db)
	// Add 6 extractable transactions
	trs, fromTrs, toTrs := transfer.GenerateTestTransfers(t, db, td.nextIndex, 6)
	for i := range trs {
		transfer.InsertTestTransfer(t, db, &trs[i])
	}
	mockTestAccountsWithAddresses(t, db, append(append(append(fromTds, toTds...), fromTrs...), toTrs...))

	var filter Filter
	chainIDs := allNetworksFilter()
	entries, err := getActivityEntries(context.Background(), db, []eth_common.Address{}, chainIDs, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 10, len(entries))

	chainIDs = []common.ChainID{5674839210}
	entries, err = getActivityEntries(context.Background(), db, []eth_common.Address{}, chainIDs, filter, 0, 15)
	require.NoError(t, err)
	// TODO: update after multi-transactions are filterable by ChainID
	require.Equal(t, 2 /*0*/, len(entries))

	chainIDs = []common.ChainID{td.pendingTr.ChainID, td.multiTx2Tr1.ChainID, trs[3].ChainID}
	entries, err = getActivityEntries(context.Background(), db, []eth_common.Address{}, chainIDs, filter, 0, 15)
	require.NoError(t, err)
	// TODO: update after multi-transactions are filterable by ChainID
	require.Equal(t, 4 /*3*/, len(entries))
}

func TestGetActivityEntriesCheckToAndFrom(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	// Adds 6 transactions from which 4 are filered out
	td, _, _ := fillTestData(t, db)

	// Add extra transactions to test To address
	trs, _, _ := transfer.GenerateTestTransfers(t, db, td.nextIndex, 2)
	transfer.InsertTestTransfer(t, db, &trs[0])
	transfer.InsertTestPendingTransaction(t, db, &trs[1])

	addresses := []eth_common.Address{td.tr1.From, td.pendingTr.From,
		td.multiTx1.FromAddress, td.multiTx2.ToAddress, trs[0].To, trs[1].To}

	var filter Filter
	entries, err := getActivityEntries(context.Background(), db, addresses, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 6, len(entries))

	require.Equal(t, SendAT, entries[5].activityType)                  // td.tr1
	require.NotEqual(t, eth.Address{}, entries[5].transaction.Address) // td.tr1
	require.Equal(t, td.tr1.From, entries[5].transaction.Address)      // td.tr1

	require.Equal(t, SendAT, entries[4].activityType) // td.pendingTr

	// Multi-transactions are always considered as SendAT
	require.Equal(t, SendAT, entries[3].activityType) // td.multiTx1
	require.Equal(t, SendAT, entries[2].activityType) // td.multiTx2

	require.Equal(t, ReceiveAT, entries[1].activityType)               // trs[0]
	require.NotEqual(t, eth.Address{}, entries[1].transaction.Address) // trs[0]
	require.Equal(t, trs[0].To, entries[1].transaction.Address)        // trs[0]

	require.Equal(t, ReceiveAT, entries[0].activityType) // trs[1] (pending)

	// TODO: add accounts to DB for proper detection of sender/receiver
	// TODO: Test with all addresses
}

// TODO test sub-transaction count for multi-transactions

func TestGetActivityEntriesCheckContextCancellation(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	_, _, _ = fillTestData(t, db)

	cancellableCtx, cancelFn := context.WithCancel(context.Background())
	cancelFn()

	activities, err := getActivityEntries(cancellableCtx, db, []eth.Address{}, []common.ChainID{}, Filter{}, 0, 10)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 0, len(activities))
}
