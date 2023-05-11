package activity

import (
	"database/sql"
	"testing"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/testutils"
	"github.com/status-im/status-go/services/wallet/transfer"

	eth "github.com/ethereum/go-ethereum/common"
	eth_common "github.com/ethereum/go-ethereum/common"

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
	transfer.InsertTestPendingTransaction(t, db, &td.pendingTr)

	td.singletonMTr = trs[2]
	td.singletonMTr.FromToken = testutils.SntSymbol
	td.singletonMTr.ToToken = testutils.DaiSymbol
	td.singletonMTID = transfer.InsertTestMultiTransaction(t, db, &td.singletonMTr)

	td.mTr = trs[3]
	td.mTr.ToToken = testutils.SntSymbol
	td.mTrID = transfer.InsertTestMultiTransaction(t, db, &td.mTr)

	td.subTr = trs[4]
	td.subTr.MultiTransactionID = td.mTrID
	transfer.InsertTestTransfer(t, db, &td.subTr)

	td.subPendingTr = trs[5]
	td.subPendingTr.MultiTransactionID = td.mTrID
	transfer.InsertTestPendingTransaction(t, db, &td.subPendingTr)
	return
}

func TestGetActivityEntriesAll(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	td := fillTestData(t, db)

	var filter Filter
	entries, err := GetActivityEntries(db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 10)
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
		transaction:    &transfer.TransactionIdentity{ChainID: td.tr1.ChainID, Hash: td.tr1.Hash, Address: td.tr1.To},
		id:             td.tr1.MultiTransactionID,
		timestamp:      td.tr1.Timestamp,
		activityType:   ReceiveAT,
		activityStatus: FinalizedAS,
		tokenType:      AssetTT,
	}, entries))
	require.True(t, testutils.StructExistsInSlice(Entry{
		payloadType:    PendingTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: td.pendingTr.ChainID, Hash: td.pendingTr.Hash},
		id:             td.pendingTr.MultiTransactionID,
		timestamp:      td.pendingTr.Timestamp,
		activityType:   ReceiveAT,
		activityStatus: PendingAS,
		tokenType:      AssetTT,
	}, entries))
	require.True(t, testutils.StructExistsInSlice(Entry{
		payloadType:    MultiTransactionPT,
		transaction:    nil,
		id:             td.singletonMTID,
		timestamp:      td.singletonMTr.Timestamp,
		activityType:   SendAT,
		activityStatus: FinalizedAS,
		tokenType:      AssetTT,
	}, entries))
	require.True(t, testutils.StructExistsInSlice(Entry{
		payloadType:    MultiTransactionPT,
		transaction:    nil,
		id:             td.mTrID,
		timestamp:      td.mTr.Timestamp,
		activityType:   SendAT,
		activityStatus: FinalizedAS,
		tokenType:      AssetTT,
	}, entries))

	// Ensure the sub-transactions of the multi-transactions are not returned
	require.False(t, testutils.StructExistsInSlice(Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: td.subTr.ChainID, Hash: td.subTr.Hash, Address: td.subTr.To},
		id:             td.subTr.MultiTransactionID,
		timestamp:      td.subTr.Timestamp,
		activityType:   SendAT,
		activityStatus: FinalizedAS,
		tokenType:      AssetTT,
	}, entries))
	require.False(t, testutils.StructExistsInSlice(Entry{
		payloadType:    PendingTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: td.subPendingTr.ChainID, Hash: td.subPendingTr.Hash},
		id:             td.subPendingTr.MultiTransactionID,
		timestamp:      td.subPendingTr.Timestamp,
		activityType:   SendAT,
		activityStatus: PendingAS,
		tokenType:      AssetTT,
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

	// TODO: test also when there is a transaction in the other direction

	// Ensure they are the oldest transactions (last in the list) and we have a consistent order
	receiverTr.Timestamp--
	transfer.InsertTestTransfer(t, db, &receiverTr)

	var filter Filter
	entries, err := GetActivityEntries(db, []eth.Address{td.tr1.From, receiverTr.From}, []common.ChainID{}, filter, 0, 10)
	require.NoError(t, err)
	require.Equal(t, 2, len(entries))

	// Check that the transaction are labeled alternatively as send and receive
	require.Equal(t, ReceiveAT, entries[1].activityType)
	require.NotEqual(t, eth.Address{}, entries[1].transaction.Address)
	require.Equal(t, receiverTr.To, entries[1].transaction.Address)

	require.Equal(t, SendAT, entries[0].activityType)
	require.NotEqual(t, eth.Address{}, entries[0].transaction.Address)
	require.Equal(t, td.tr1.From, entries[0].transaction.Address)

	// add accounts to DB for proper detection of sender/receiver in all cases
	accounts.AddTestAccounts(t, db, []*accounts.Account{
		{Address: types.Address(td.tr1.From), Chat: false, Wallet: true},
		{Address: types.Address(receiverTr.From)},
	})

	entries, err = GetActivityEntries(db, []eth.Address{}, []common.ChainID{}, filter, 0, 10)
	require.NoError(t, err)
	require.Equal(t, 5, len(entries))

	// Check that the transaction are labeled alternatively as send and receive
	require.Equal(t, ReceiveAT, entries[4].activityType)
	require.Equal(t, SendAT, entries[3].activityType)
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
	filter.Period.EndTimestamp = NoLimitTimestampForPeriod
	entries, err := GetActivityEntries(db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 8, len(entries))
	// Check start and end content
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: trs[5].ChainID, Hash: trs[5].Hash, Address: trs[5].To},
		id:             0,
		timestamp:      trs[5].Timestamp,
		activityType:   ReceiveAT,
		activityStatus: FinalizedAS,
		tokenType:      AssetTT,
	}, entries[0])
	require.Equal(t, Entry{
		payloadType:    MultiTransactionPT,
		transaction:    nil,
		id:             td.singletonMTID,
		timestamp:      td.singletonMTr.Timestamp,
		activityType:   SendAT,
		activityStatus: FinalizedAS,
		tokenType:      AssetTT,
	}, entries[7])

	// Test complete interval
	filter.Period.EndTimestamp = trs[2].Timestamp
	entries, err = GetActivityEntries(db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 5, len(entries))
	// Check start and end content
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: trs[2].ChainID, Hash: trs[2].Hash, Address: trs[2].To},
		id:             0,
		timestamp:      trs[2].Timestamp,
		activityType:   ReceiveAT,
		activityStatus: FinalizedAS,
		tokenType:      AssetTT,
	}, entries[0])
	require.Equal(t, Entry{
		payloadType:    MultiTransactionPT,
		transaction:    nil,
		id:             td.singletonMTID,
		timestamp:      td.singletonMTr.Timestamp,
		activityType:   SendAT,
		activityStatus: FinalizedAS,
		tokenType:      AssetTT,
	}, entries[4])

	// Test end only
	filter.Period.StartTimestamp = NoLimitTimestampForPeriod
	entries, err = GetActivityEntries(db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 7, len(entries))
	// Check start and end content
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: trs[2].ChainID, Hash: trs[2].Hash, Address: trs[2].To},
		id:             0,
		timestamp:      trs[2].Timestamp,
		activityType:   ReceiveAT,
		activityStatus: FinalizedAS,
		tokenType:      AssetTT,
	}, entries[0])
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: td.tr1.ChainID, Hash: td.tr1.Hash, Address: td.tr1.To},
		id:             0,
		timestamp:      td.tr1.Timestamp,
		activityType:   ReceiveAT,
		activityStatus: FinalizedAS,
		tokenType:      AssetTT,
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
	entries, err := GetActivityEntries(db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 5)
	require.NoError(t, err)
	require.Equal(t, 5, len(entries))

	// Get time based interval
	filter.Period.StartTimestamp = trs[2].Timestamp
	filter.Period.EndTimestamp = trs[8].Timestamp
	entries, err = GetActivityEntries(db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 3)
	require.NoError(t, err)
	require.Equal(t, 3, len(entries))
	// Check start and end content
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: trs[8].ChainID, Hash: trs[8].Hash, Address: trs[8].To},
		id:             0,
		timestamp:      trs[8].Timestamp,
		activityType:   ReceiveAT,
		activityStatus: FinalizedAS,
		tokenType:      AssetTT,
	}, entries[0])
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: trs[6].ChainID, Hash: trs[6].Hash, Address: trs[6].To},
		id:             0,
		timestamp:      trs[6].Timestamp,
		activityType:   ReceiveAT,
		activityStatus: FinalizedAS,
		tokenType:      AssetTT,
	}, entries[2])

	// Move window 2 entries forward
	entries, err = GetActivityEntries(db, []eth_common.Address{}, []common.ChainID{}, filter, 2, 3)
	require.NoError(t, err)
	require.Equal(t, 3, len(entries))
	// Check start and end content
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: trs[6].ChainID, Hash: trs[6].Hash, Address: trs[6].To},
		id:             0,
		timestamp:      trs[6].Timestamp,
		activityType:   ReceiveAT,
		activityStatus: FinalizedAS,
		tokenType:      AssetTT,
	}, entries[0])
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: trs[4].ChainID, Hash: trs[4].Hash, Address: trs[4].To},
		id:             0,
		timestamp:      trs[4].Timestamp,
		activityType:   ReceiveAT,
		activityStatus: FinalizedAS,
		tokenType:      AssetTT,
	}, entries[2])

	// Move window 4 more entries to test filter cap
	entries, err = GetActivityEntries(db, []eth_common.Address{}, []common.ChainID{}, filter, 6, 3)
	require.NoError(t, err)
	require.Equal(t, 1, len(entries))
	// Check start and end content
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: trs[2].ChainID, Hash: trs[2].Hash, Address: trs[2].To},
		id:             0,
		timestamp:      trs[2].Timestamp,
		activityType:   ReceiveAT,
		activityStatus: FinalizedAS,
		tokenType:      AssetTT,
	}, entries[0])
}

func TestGetActivityEntriesFilterByType(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	// Adds 4 extractable transactions
	fillTestData(t, db)
	// Add 6 extractable transactions: one MultiTransactionSwap, two MultiTransactionBridge rest MultiTransactionSend
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

	filter.Types = allActivityTypesFilter()
	entries, err := GetActivityEntries(db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 10, len(entries))

	filter.Types = []Type{SendAT, SwapAT}
	entries, err = GetActivityEntries(db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 8, len(entries))
	swapCount := 0
	sendCount := 0
	receiveCount := 0
	for _, entry := range entries {
		if entry.activityType == SendAT {
			sendCount++
		}
		if entry.activityType == ReceiveAT {
			receiveCount++
		}
		if entry.activityType == SwapAT {
			swapCount++
		}
	}
	require.Equal(t, 2, sendCount)
	require.Equal(t, 5, receiveCount)
	require.Equal(t, 1, swapCount)

	// Test filtering out with address involved
	filter.Types = []Type{BridgeAT, ReceiveAT}
	// Include one "to" from transfers to be detected as receive
	addresses := []eth_common.Address{trs[0].To, trs[1].To, trs[2].From, trs[3].From, trs[5].From}
	entries, err = GetActivityEntries(db, addresses, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 3, len(entries))
	bridgeCount := 0
	receiveCount = 0
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

func TestGetActivityEntriesFilterByAddresses(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	// Adds 4 extractable transactions
	td := fillTestData(t, db)
	trs := transfer.GenerateTestTransactions(t, db, 7, 6)
	for i := range trs {
		transfer.InsertTestTransfer(t, db, &trs[i])
	}

	var filter Filter

	addressesFilter := allAddressesFilter()
	entries, err := GetActivityEntries(db, addressesFilter, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 10, len(entries))

	addressesFilter = []eth_common.Address{td.mTr.To, trs[1].From, trs[4].To}
	entries, err = GetActivityEntries(db, addressesFilter, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 3, len(entries))
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: trs[4].ChainID, Hash: trs[4].Hash, Address: trs[4].To},
		id:             0,
		timestamp:      trs[4].Timestamp,
		activityType:   ReceiveAT,
		activityStatus: FinalizedAS,
		tokenType:      AssetTT,
	}, entries[0])
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: trs[1].ChainID, Hash: trs[1].Hash, Address: trs[1].From},
		id:             0,
		timestamp:      trs[1].Timestamp,
		activityType:   SendAT,
		activityStatus: FinalizedAS,
		tokenType:      AssetTT,
	}, entries[1])
	require.Equal(t, Entry{
		payloadType:    MultiTransactionPT,
		transaction:    nil,
		id:             td.mTrID,
		timestamp:      td.mTr.Timestamp,
		activityType:   SendAT,
		activityStatus: FinalizedAS,
		tokenType:      AssetTT,
	}, entries[2])
}

func TestGetActivityEntriesFilterByStatus(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	// Adds 4 extractable transactions
	fillTestData(t, db)
	// Add 6 extractable transactions
	trs := transfer.GenerateTestTransactions(t, db, 7, 6)
	for i := range trs {
		transfer.InsertTestTransfer(t, db, &trs[i])
	}

	var filter Filter
	filter.Statuses = []Status{}
	entries, err := GetActivityEntries(db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 10, len(entries))

	filter.Statuses = allActivityStatusesFilter()
	entries, err = GetActivityEntries(db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 10, len(entries))

	// TODO: enabled and finish tests after extending DB with transaction status
	//
	// filter.Statuses = []Status{PendingAS}
	// entries, err = GetActivityEntries(db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	// require.NoError(t, err)
	// require.Equal(t, 1, len(entries))

	// filter.Statuses = []Status{FailedAS, CompleteAS}
	// entries, err = GetActivityEntries(db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	// require.NoError(t, err)
	// require.Equal(t, 9, len(entries))
}

func TestGetActivityEntriesFilterByTokenType(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	// Adds 4 extractable transactions 2 transactions ETH, one MT SNT to DAI and another MT ETH to SNT
	fillTestData(t, db)
	// Add 6 extractable transactions with USDC (only erc20 as type in DB)
	trs := transfer.GenerateTestTransactions(t, db, 7, 6)
	for i := range trs {
		trs[i].FromToken = "USDC"
		transfer.InsertTestTransfer(t, db, &trs[i])
	}

	var filter Filter
	filter.Tokens = noAssetsFilter()
	entries, err := GetActivityEntries(db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 0, len(entries))

	filter.Tokens = allTokensFilter()
	entries, err = GetActivityEntries(db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 10, len(entries))

	// Regression when collectibles is nil
	filter.Tokens = Tokens{[]TokenCode{}, nil, []TokenType{}}
	entries, err = GetActivityEntries(db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 10, len(entries))

	filter.Tokens = Tokens{Assets: []TokenCode{"ETH"}, EnabledTypes: []TokenType{AssetTT}}
	entries, err = GetActivityEntries(db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 3, len(entries))

	// TODO: update tests after adding token type to transfers
	filter.Tokens = Tokens{Assets: []TokenCode{"USDC", "DAI"}, EnabledTypes: []TokenType{AssetTT}}
	entries, err = GetActivityEntries(db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 1, len(entries))

	// Regression when EnabledTypes ar empty
	filter.Tokens = Tokens{Assets: []TokenCode{"USDC", "DAI"}, EnabledTypes: []TokenType{}}
	entries, err = GetActivityEntries(db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 1, len(entries))
}

func TestGetActivityEntriesFilterByToAddresses(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	// Adds 4 extractable transactions
	td := fillTestData(t, db)
	// Add 6 extractable transactions
	trs := transfer.GenerateTestTransactions(t, db, 7, 6)
	for i := range trs {
		transfer.InsertTestTransfer(t, db, &trs[i])
	}

	var filter Filter
	filter.CounterpartyAddresses = allAddressesFilter()
	entries, err := GetActivityEntries(db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 10, len(entries))

	filter.CounterpartyAddresses = []eth_common.Address{eth_common.HexToAddress("0x567890")}
	entries, err = GetActivityEntries(db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 0, len(entries))

	filter.CounterpartyAddresses = []eth_common.Address{td.pendingTr.To, td.mTr.To, trs[3].To}
	entries, err = GetActivityEntries(db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 3, len(entries))

	filter.CounterpartyAddresses = []eth_common.Address{td.tr1.To, td.pendingTr.From, trs[3].From, trs[5].To}
	entries, err = GetActivityEntries(db, []eth_common.Address{}, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 2, len(entries))
}
func TestGetActivityEntriesFilterByNetworks(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	// Adds 4 extractable transactions
	td := fillTestData(t, db)
	// Add 6 extractable transactions
	trs := transfer.GenerateTestTransactions(t, db, 7, 6)
	for i := range trs {
		transfer.InsertTestTransfer(t, db, &trs[i])
	}

	var filter Filter
	chainIDs := allNetworksFilter()
	entries, err := GetActivityEntries(db, []eth_common.Address{}, chainIDs, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 10, len(entries))

	chainIDs = []common.ChainID{5674839210}
	entries, err = GetActivityEntries(db, []eth_common.Address{}, chainIDs, filter, 0, 15)
	require.NoError(t, err)
	// TODO: update after multi-transactions are filterable by ChainID
	require.Equal(t, 2 /*0*/, len(entries))

	chainIDs = []common.ChainID{td.pendingTr.ChainID, td.mTr.ChainID, trs[3].ChainID}
	entries, err = GetActivityEntries(db, []eth_common.Address{}, chainIDs, filter, 0, 15)
	require.NoError(t, err)
	// TODO: update after multi-transactions are filterable by ChainID
	require.Equal(t, 4 /*3*/, len(entries))
}

func TestGetActivityEntriesCheckToAndFrom(t *testing.T) {
	db, close := setupTestActivityDB(t)
	defer close()

	// Adds 6 transactions from which 4 are filered out
	td := fillTestData(t, db)

	// Add extra transactions to test To address
	trs := transfer.GenerateTestTransactions(t, db, 7, 2)
	transfer.InsertTestTransfer(t, db, &trs[0])
	transfer.InsertTestPendingTransaction(t, db, &trs[1])

	addresses := []eth_common.Address{td.tr1.From, td.pendingTr.From,
		td.singletonMTr.From, td.mTr.To, trs[0].To, trs[1].To}

	var filter Filter
	entries, err := GetActivityEntries(db, addresses, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 6, len(entries))

	require.Equal(t, SendAT, entries[5].activityType)                  // td.tr1
	require.NotEqual(t, eth.Address{}, entries[5].transaction.Address) // td.tr1
	require.Equal(t, td.tr1.From, entries[5].transaction.Address)      // td.tr1

	require.Equal(t, SendAT, entries[4].activityType) // td.pendingTr

	// Multi-transactions are always considered as SendAT
	require.Equal(t, SendAT, entries[3].activityType) // td.singletonMTr
	require.Equal(t, SendAT, entries[2].activityType) // td.mTr

	require.Equal(t, ReceiveAT, entries[1].activityType)               // trs[0]
	require.NotEqual(t, eth.Address{}, entries[1].transaction.Address) // trs[0]
	require.Equal(t, trs[0].To, entries[1].transaction.Address)        // trs[0]

	require.Equal(t, ReceiveAT, entries[0].activityType) // trs[1] (pending)

	// TODO: add accounts to DB for proper detection of sender/receiver
	// TODO: Test with all addresses
}
