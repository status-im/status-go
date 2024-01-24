package activity

import (
	"context"
	"database/sql"
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/testutils"
	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/stretchr/testify/require"
)

var mockupTime = time.Unix(946724400, 0) // 2000-01-01 12:00:00

func tokenFromSymbol(chainID *common.ChainID, symbol string) *Token {
	for i, t := range transfer.TestTokens {
		if (chainID == nil || t.ChainID == uint64(*chainID)) && t.Symbol == symbol {
			tokenType := Erc20
			if testutils.SliceContains(transfer.NativeTokenIndices, i) {
				tokenType = Native
			}
			return &Token{
				TokenType: tokenType,
				ChainID:   common.ChainID(t.ChainID),
				Address:   t.Address,
			}
		}
	}
	return nil
}

func tokenFromCollectible(c *transfer.TestCollectible) Token {
	return Token{
		TokenType: Erc721,
		ChainID:   c.ChainID,
		Address:   c.TokenAddress,
		TokenID:   (*hexutil.Big)(c.TokenID),
	}
}

func setupTestActivityDBStorageChoice(tb testing.TB, inMemory bool) (deps FilterDependencies, close func()) {
	var db *sql.DB
	var err error
	cleanupDB := func() error { return nil }
	cleanupWalletDB := func() error { return nil }
	if inMemory {
		db, err = helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
		require.NoError(tb, err)
	} else {
		db, cleanupWalletDB, err = helpers.SetupTestSQLDB(walletdatabase.DbInitializer{}, "wallet-activity-tests")
		require.NoError(tb, err)
	}

	deps = FilterDependencies{
		db: db,
		tokenSymbol: func(token Token) string {
			switch token.TokenType {
			case Native:
				for i, t := range transfer.TestTokens {
					if t.ChainID == uint64(token.ChainID) && testutils.SliceContains(transfer.NativeTokenIndices, i) {
						return t.Symbol
					}
				}
			case Erc20:
				for _, t := range transfer.TestTokens {
					if t.ChainID == uint64(token.ChainID) && t.Address == token.Address {
						return t.Symbol
					}
				}
			}
			// In case of ERC721 and ERC1155 we don't have a symbol and they are not yet handled
			return ""
		},
		// tokenFromSymbol nil chainID accepts first symbol found
		tokenFromSymbol: tokenFromSymbol,
		currentTimestamp: func() int64 {
			return mockupTime.Unix()
		},
	}

	return deps, func() {
		require.NoError(tb, cleanupDB())
		require.NoError(tb, cleanupWalletDB())
	}
}

func setupTestActivityDB(tb testing.TB) (deps FilterDependencies, close func()) {
	return setupTestActivityDBStorageChoice(tb, true)
}

type testData struct {
	tr1               transfer.TestTransfer // index 1, ETH/Goerli
	pendingTr         transfer.TestTransfer // index 2, ETH/Optimism
	multiTx1Tr1       transfer.TestTransfer // index 3, USDC/Mainnet
	multiTx2Tr1       transfer.TestTransfer // index 4, USDC/Goerli
	multiTx1Tr2       transfer.TestTransfer // index 5, USDC/Optimism
	multiTx2Tr2       transfer.TestTransfer // index 6, SNT/Mainnet
	multiTx2PendingTr transfer.TestTransfer // index 7, DAI/Mainnet

	multiTx1   transfer.TestMultiTransaction
	multiTx1ID transfer.MultiTransactionIDType

	multiTx2   transfer.TestMultiTransaction
	multiTx2ID transfer.MultiTransactionIDType

	nextIndex int
}

// Generates and adds to the DB 7 transfers and 2 multitransactions.
// There are only 4 extractable activity entries (transactions + multi-transactions) with timestamps 1-4. The others are associated with a multi-transaction
func fillTestData(t *testing.T, db *sql.DB) (td testData, fromAddresses, toAddresses []eth.Address) {
	// Generates ETH/Goerli, ETH/Optimism, USDC/Mainnet, USDC/Goerli, USDC/Optimism, SNT/Mainnet, DAI/Mainnet
	trs, fromAddresses, toAddresses := transfer.GenerateTestTransfers(t, db, 1, 7)

	// Plain transfer
	td.tr1 = trs[0]
	transfer.InsertTestTransfer(t, db, td.tr1.To, &td.tr1)

	// Pending transfer
	td.pendingTr = trs[1]
	transfer.InsertTestPendingTransaction(t, db, &td.pendingTr)

	// Send Multitransaction containing 2 x Plain transfers
	td.multiTx1Tr1 = trs[2]
	td.multiTx1Tr2 = trs[4]

	td.multiTx1 = transfer.GenerateTestSendMultiTransaction(td.multiTx1Tr1)
	td.multiTx1.ToToken = testutils.DaiSymbol
	td.multiTx1ID = transfer.InsertTestMultiTransaction(t, db, &td.multiTx1)

	td.multiTx1Tr1.MultiTransactionID = td.multiTx1ID
	transfer.InsertTestTransfer(t, db, td.multiTx1Tr1.To, &td.multiTx1Tr1)

	td.multiTx1Tr2.MultiTransactionID = td.multiTx1ID
	transfer.InsertTestTransfer(t, db, td.multiTx1Tr2.To, &td.multiTx1Tr2)

	// Send Multitransaction containing 2 x Plain transfers + 1 x Pending transfer
	td.multiTx2Tr1 = trs[3]
	td.multiTx2Tr2 = trs[5]
	td.multiTx2PendingTr = trs[6]

	td.multiTx2 = transfer.GenerateTestSendMultiTransaction(td.multiTx2Tr1)
	td.multiTx2.ToToken = testutils.SntSymbol
	td.multiTx2ID = transfer.InsertTestMultiTransaction(t, db, &td.multiTx2)

	td.multiTx2Tr1.MultiTransactionID = td.multiTx2ID
	transfer.InsertTestTransfer(t, db, td.multiTx2Tr1.To, &td.multiTx2Tr1)

	td.multiTx2Tr2.MultiTransactionID = td.multiTx2ID
	transfer.InsertTestTransfer(t, db, td.multiTx2Tr2.To, &td.multiTx2Tr2)

	td.multiTx2PendingTr.MultiTransactionID = td.multiTx2ID
	transfer.InsertTestPendingTransaction(t, db, &td.multiTx2PendingTr)

	td.nextIndex = 8
	return td, fromAddresses, toAddresses
}

func TTrToToken(t *testing.T, tt *transfer.TestTransaction) *Token {
	token, isNative := transfer.TestTrToToken(t, tt)
	tokenType := Erc20
	if isNative {
		tokenType = Native
	}
	return &Token{
		TokenType: tokenType,
		ChainID:   common.ChainID(token.ChainID),
		Address:   token.Address,
	}
}

func expectedTokenType(tokenAddress eth.Address) *TransferType {
	transferType := new(TransferType)
	if (tokenAddress != eth.Address{}) {
		*transferType = TransferTypeErc20
	} else {
		*transferType = TransferTypeEth
	}
	return transferType
}

func TestGetActivityEntriesAll(t *testing.T) {
	deps, close := setupTestActivityDB(t)
	defer close()

	td, fromAddresses, toAddresses := fillTestData(t, deps.db)

	var filter Filter
	entries, err := getActivityEntries(context.Background(), deps, append(toAddresses, fromAddresses...), true, []common.ChainID{}, filter, 0, 10)
	require.NoError(t, err)
	require.Equal(t, 4, len(entries))

	// Ensure we have the correct order
	var curTimestamp int64 = 4
	for _, entry := range entries {
		require.Equal(t, curTimestamp, entry.timestamp, "entries are sorted by timestamp; expected %d, got %d", curTimestamp, entry.timestamp)
		curTimestamp--
	}

	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: td.tr1.ChainID, Hash: td.tr1.Hash, Address: td.tr1.To},
		id:             td.tr1.MultiTransactionID,
		timestamp:      td.tr1.Timestamp,
		activityType:   ReceiveAT,
		activityStatus: FinalizedAS,
		amountOut:      (*hexutil.Big)(big.NewInt(0)),
		amountIn:       (*hexutil.Big)(big.NewInt(td.tr1.Value)),
		tokenOut:       nil,
		tokenIn:        TTrToToken(t, &td.tr1.TestTransaction),
		symbolOut:      nil,
		symbolIn:       common.NewAndSet("ETH"),
		sender:         &td.tr1.From,
		recipient:      &td.tr1.To,
		chainIDOut:     nil,
		chainIDIn:      &td.tr1.ChainID,
		transferType:   expectedTokenType(td.tr1.Token.Address),
	}, entries[3])
	require.Equal(t, Entry{
		payloadType:    PendingTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: td.pendingTr.ChainID, Hash: td.pendingTr.Hash},
		id:             td.pendingTr.MultiTransactionID,
		timestamp:      td.pendingTr.Timestamp,
		activityType:   SendAT,
		activityStatus: PendingAS,
		amountOut:      (*hexutil.Big)(big.NewInt(td.pendingTr.Value)),
		amountIn:       (*hexutil.Big)(big.NewInt(0)),
		tokenOut:       TTrToToken(t, &td.pendingTr.TestTransaction),
		tokenIn:        nil,
		symbolOut:      common.NewAndSet("ETH"),
		symbolIn:       nil,
		sender:         &td.pendingTr.From,
		recipient:      &td.pendingTr.To,
		chainIDOut:     &td.pendingTr.ChainID,
		chainIDIn:      nil,
		transferType:   expectedTokenType(eth.Address{}),
	}, entries[2])
	require.Equal(t, Entry{
		payloadType:    MultiTransactionPT,
		transaction:    nil,
		id:             td.multiTx1ID,
		timestamp:      td.multiTx1.Timestamp,
		activityType:   SendAT,
		activityStatus: FinalizedAS,
		amountOut:      (*hexutil.Big)(big.NewInt(td.multiTx1.FromAmount)),
		amountIn:       (*hexutil.Big)(big.NewInt(td.multiTx1.ToAmount)),
		tokenOut:       tokenFromSymbol(nil, td.multiTx1.FromToken),
		tokenIn:        tokenFromSymbol(nil, td.multiTx1.ToToken),
		symbolOut:      common.NewAndSet("USDC"),
		symbolIn:       common.NewAndSet("DAI"),
		sender:         &td.multiTx1.FromAddress,
		recipient:      &td.multiTx1.ToAddress,
	}, entries[1])
	require.Equal(t, Entry{
		payloadType:    MultiTransactionPT,
		transaction:    nil,
		id:             td.multiTx2ID,
		timestamp:      td.multiTx2.Timestamp,
		activityType:   SendAT,
		activityStatus: PendingAS,
		amountOut:      (*hexutil.Big)(big.NewInt(td.multiTx2.FromAmount)),
		amountIn:       (*hexutil.Big)(big.NewInt(td.multiTx2.ToAmount)),
		symbolOut:      common.NewAndSet("USDC"),
		symbolIn:       common.NewAndSet("SNT"),
		tokenOut:       tokenFromSymbol(nil, td.multiTx2.FromToken),
		tokenIn:        tokenFromSymbol(nil, td.multiTx2.ToToken),
		sender:         &td.multiTx2.FromAddress,
		recipient:      &td.multiTx2.ToAddress,
	}, entries[0])
}

// TestGetActivityEntriesWithSenderFilter covers the corner-case of having both sender and receiver in the filter.
// In this specific case we expect that there will be two transactions (one probably backed by a multi-transaction)
// In case of both sender and receiver are included we validate we receive both entries otherwise only the "owned"
// transactions should be retrieved by the filter
func TestGetActivityEntriesWithSameTransactionForSenderAndReceiverInDB(t *testing.T) {
	deps, close := setupTestActivityDB(t)
	defer close()

	// Add 4 extractable transactions with timestamps 1-4
	td, _, _ := fillTestData(t, deps.db)

	// Add another transaction with owner reversed
	senderTr := td.tr1
	// Ensure we have a consistent order
	senderTr.Timestamp++
	// add sender as owner, fillTestData adds receiver as owner
	transfer.InsertTestTransfer(t, deps.db, senderTr.From, &senderTr)

	var filter Filter
	entries, err := getActivityEntries(context.Background(), deps, []eth.Address{td.tr1.To, senderTr.From}, false, []common.ChainID{}, filter, 0, 10)
	require.NoError(t, err)
	require.Equal(t, 2, len(entries))

	// Check that the transaction are labeled alternatively as send and receive
	require.Equal(t, SendAT, entries[0].activityType)
	require.Equal(t, senderTr.From, entries[0].transaction.Address)
	require.Equal(t, senderTr.From, *entries[0].sender)
	require.Equal(t, senderTr.To, *entries[0].recipient)

	require.Equal(t, ReceiveAT, entries[1].activityType)
	require.Equal(t, td.tr1.To, *entries[1].recipient)
	require.Equal(t, td.tr1.From, *entries[1].sender)
	require.Equal(t, td.tr1.To, *entries[1].recipient)
}

func TestGetActivityEntriesFilterByTime(t *testing.T) {
	deps, close := setupTestActivityDB(t)
	defer close()

	td, fromTds, toTds := fillTestData(t, deps.db)

	// Add 6 extractable transactions with timestamps 6-12
	trs, fromTrs, toTrs := transfer.GenerateTestTransfers(t, deps.db, td.nextIndex, 6)
	for i := range trs {
		transfer.InsertTestTransfer(t, deps.db, trs[i].To, &trs[i])
	}

	allAddresses := append(append(append(fromTds, toTds...), fromTrs...), toTrs...)

	// Test start only
	var filter Filter
	filter.Period.StartTimestamp = td.multiTx1.Timestamp
	filter.Period.EndTimestamp = NoLimitTimestampForPeriod
	entries, err := getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
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
		amountOut:      (*hexutil.Big)(big.NewInt(0)),
		amountIn:       (*hexutil.Big)(big.NewInt(trs[5].Value)),
		tokenOut:       nil,
		tokenIn:        TTrToToken(t, &trs[5].TestTransaction),
		symbolOut:      nil,
		symbolIn:       common.NewAndSet("USDC"),
		sender:         &trs[5].From,
		recipient:      &trs[5].To,
		chainIDOut:     nil,
		chainIDIn:      &trs[5].ChainID,
		transferType:   expectedTokenType(trs[5].Token.Address),
	}, entries[0])
	require.Equal(t, Entry{
		payloadType:    MultiTransactionPT,
		transaction:    nil,
		id:             td.multiTx1ID,
		timestamp:      td.multiTx1.Timestamp,
		activityType:   SendAT,
		activityStatus: FinalizedAS,
		amountOut:      (*hexutil.Big)(big.NewInt(td.multiTx1.FromAmount)),
		amountIn:       (*hexutil.Big)(big.NewInt(td.multiTx1.ToAmount)),
		tokenOut:       tokenFromSymbol(nil, td.multiTx1.FromToken),
		tokenIn:        tokenFromSymbol(nil, td.multiTx1.ToToken),
		symbolOut:      common.NewAndSet("USDC"),
		symbolIn:       common.NewAndSet("DAI"),
		sender:         &td.multiTx1.FromAddress,
		recipient:      &td.multiTx1.ToAddress,
		chainIDOut:     nil,
		chainIDIn:      nil,
		transferType:   nil,
	}, entries[7])

	// Test complete interval
	filter.Period.EndTimestamp = trs[2].Timestamp
	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
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
		amountOut:      (*hexutil.Big)(big.NewInt(0)),
		amountIn:       (*hexutil.Big)(big.NewInt(trs[2].Value)),
		tokenOut:       nil,
		tokenIn:        TTrToToken(t, &trs[2].TestTransaction),
		symbolOut:      nil,
		symbolIn:       common.NewAndSet("ETH"),
		sender:         &trs[2].From,
		recipient:      &trs[2].To,
		chainIDOut:     nil,
		chainIDIn:      &trs[2].ChainID,
		transferType:   expectedTokenType(trs[2].Token.Address),
	}, entries[0])
	require.Equal(t, Entry{
		payloadType:    MultiTransactionPT,
		transaction:    nil,
		id:             td.multiTx1ID,
		timestamp:      td.multiTx1.Timestamp,
		activityType:   SendAT,
		activityStatus: FinalizedAS,
		amountOut:      (*hexutil.Big)(big.NewInt(td.multiTx1.FromAmount)),
		amountIn:       (*hexutil.Big)(big.NewInt(td.multiTx1.ToAmount)),
		tokenOut:       tokenFromSymbol(nil, td.multiTx1.FromToken),
		tokenIn:        tokenFromSymbol(nil, td.multiTx1.ToToken),
		symbolOut:      common.NewAndSet("USDC"),
		symbolIn:       common.NewAndSet("DAI"),
		sender:         &td.multiTx1.FromAddress,
		recipient:      &td.multiTx1.ToAddress,
		chainIDOut:     nil,
		chainIDIn:      nil,
		transferType:   nil,
	}, entries[4])

	// Test end only
	filter.Period.StartTimestamp = NoLimitTimestampForPeriod
	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
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
		amountOut:      (*hexutil.Big)(big.NewInt(0)),
		amountIn:       (*hexutil.Big)(big.NewInt(trs[2].Value)),
		tokenOut:       nil,
		tokenIn:        TTrToToken(t, &trs[2].TestTransaction),
		symbolOut:      nil,
		symbolIn:       common.NewAndSet("ETH"),
		sender:         &trs[2].From,
		recipient:      &trs[2].To,
		chainIDOut:     nil,
		chainIDIn:      &trs[2].ChainID,
		transferType:   expectedTokenType(trs[2].Token.Address),
	}, entries[0])
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: td.tr1.ChainID, Hash: td.tr1.Hash, Address: td.tr1.To},
		id:             0,
		timestamp:      td.tr1.Timestamp,
		activityType:   ReceiveAT,
		activityStatus: FinalizedAS,
		amountOut:      (*hexutil.Big)(big.NewInt(0)),
		amountIn:       (*hexutil.Big)(big.NewInt(td.tr1.Value)),
		tokenOut:       nil,
		tokenIn:        TTrToToken(t, &td.tr1.TestTransaction),
		symbolOut:      nil,
		symbolIn:       common.NewAndSet("ETH"),
		sender:         &td.tr1.From,
		recipient:      &td.tr1.To,
		chainIDOut:     nil,
		chainIDIn:      &td.tr1.ChainID,
		transferType:   expectedTokenType(td.tr1.Token.Address),
	}, entries[6])
}

func TestGetActivityEntriesCheckOffsetAndLimit(t *testing.T) {
	deps, close := setupTestActivityDB(t)
	defer close()

	// Add 10 extractable transactions with timestamps 1-10
	trs, fromTrs, toTrs := transfer.GenerateTestTransfers(t, deps.db, 1, 10)
	for i := range trs {
		transfer.InsertTestTransfer(t, deps.db, trs[i].To, &trs[i])
	}

	allAddresses := append(fromTrs, toTrs...)

	var filter Filter
	// Get all
	entries, err := getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 5)
	require.NoError(t, err)
	require.Equal(t, 5, len(entries))

	// Get time based interval
	filter.Period.StartTimestamp = trs[2].Timestamp
	filter.Period.EndTimestamp = trs[8].Timestamp
	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 3)
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
		amountOut:      (*hexutil.Big)(big.NewInt(0)),
		amountIn:       (*hexutil.Big)(big.NewInt(trs[8].Value)),
		tokenOut:       nil,
		tokenIn:        TTrToToken(t, &trs[8].TestTransaction),
		symbolOut:      nil,
		symbolIn:       common.NewAndSet("ETH"),
		sender:         &trs[8].From,
		recipient:      &trs[8].To,
		chainIDOut:     nil,
		chainIDIn:      &trs[8].ChainID,
		transferType:   expectedTokenType(trs[8].Token.Address),
	}, entries[0])
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: trs[6].ChainID, Hash: trs[6].Hash, Address: trs[6].To},
		id:             0,
		timestamp:      trs[6].Timestamp,
		activityType:   ReceiveAT,
		activityStatus: FinalizedAS,
		amountOut:      (*hexutil.Big)(big.NewInt(0)),
		amountIn:       (*hexutil.Big)(big.NewInt(trs[6].Value)),
		tokenOut:       nil,
		tokenIn:        TTrToToken(t, &trs[6].TestTransaction),
		symbolOut:      nil,
		symbolIn:       common.NewAndSet("DAI"),
		sender:         &trs[6].From,
		recipient:      &trs[6].To,
		chainIDOut:     nil,
		chainIDIn:      &trs[6].ChainID,
		transferType:   expectedTokenType(trs[6].Token.Address),
	}, entries[2])

	// Move window 2 entries forward
	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 2, 3)
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
		amountOut:      (*hexutil.Big)(big.NewInt(0)),
		amountIn:       (*hexutil.Big)(big.NewInt(trs[6].Value)),
		tokenOut:       nil,
		tokenIn:        TTrToToken(t, &trs[6].TestTransaction),
		symbolOut:      nil,
		symbolIn:       common.NewAndSet("DAI"),
		sender:         &trs[6].From,
		recipient:      &trs[6].To,
		chainIDOut:     nil,
		chainIDIn:      &trs[6].ChainID,
		transferType:   expectedTokenType(trs[6].Token.Address),
	}, entries[0])
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: trs[4].ChainID, Hash: trs[4].Hash, Address: trs[4].To},
		id:             0,
		timestamp:      trs[4].Timestamp,
		activityType:   ReceiveAT,
		activityStatus: FinalizedAS,
		amountOut:      (*hexutil.Big)(big.NewInt(0)),
		amountIn:       (*hexutil.Big)(big.NewInt(trs[4].Value)),
		tokenOut:       nil,
		tokenIn:        TTrToToken(t, &trs[4].TestTransaction),
		symbolOut:      nil,
		symbolIn:       common.NewAndSet("USDC"),
		sender:         &trs[4].From,
		recipient:      &trs[4].To,
		chainIDOut:     nil,
		chainIDIn:      &trs[4].ChainID,
		transferType:   expectedTokenType(trs[4].Token.Address),
	}, entries[2])

	// Move window 4 more entries to test filter cap
	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 6, 3)
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
		amountOut:      (*hexutil.Big)(big.NewInt(0)),
		amountIn:       (*hexutil.Big)(big.NewInt(trs[2].Value)),
		tokenOut:       nil,
		tokenIn:        TTrToToken(t, &trs[2].TestTransaction),
		symbolOut:      nil,
		symbolIn:       common.NewAndSet("USDC"),
		sender:         &trs[2].From,
		recipient:      &trs[2].To,
		chainIDOut:     nil,
		chainIDIn:      &trs[2].ChainID,
		transferType:   expectedTokenType(trs[2].Token.Address),
	}, entries[0])
}

func countTypes(entries []Entry) (sendCount, receiveCount, contractCount, mintCount, swapCount, buyCount, bridgeCount int) {
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
		case ContractDeploymentAT:
			contractCount++
		case MintAT:
			mintCount++
		}
	}
	return
}

func TestGetActivityEntriesFilterByType(t *testing.T) {
	deps, close := setupTestActivityDB(t)
	defer close()

	// Adds 4 extractable transactions
	td, tdFromAdds, tdToAddrs := fillTestData(t, deps.db)
	// Add 5 extractable transactions: one MultiTransactionSwap, two MultiTransactionBridge and two MultiTransactionSend
	multiTxs := make([]transfer.TestMultiTransaction, 5)
	trs, fromAddrs, toAddrs := transfer.GenerateTestTransfers(t, deps.db, td.nextIndex, len(multiTxs)*2)
	multiTxs[0] = transfer.GenerateTestBridgeMultiTransaction(trs[0], trs[1])
	multiTxs[1] = transfer.GenerateTestSwapMultiTransaction(trs[2], testutils.SntSymbol, 100) // trs[3]
	multiTxs[2] = transfer.GenerateTestSendMultiTransaction(trs[4])                           // trs[5]
	multiTxs[3] = transfer.GenerateTestBridgeMultiTransaction(trs[6], trs[7])
	multiTxs[4] = transfer.GenerateTestSendMultiTransaction(trs[8]) // trs[9]

	var lastMT transfer.MultiTransactionIDType
	for i := range trs {
		if i%2 == 0 {
			lastMT = transfer.InsertTestMultiTransaction(t, deps.db, &multiTxs[i/2])
		}
		trs[i].MultiTransactionID = lastMT
		transfer.InsertTestTransfer(t, deps.db, trs[i].To, &trs[i])
	}

	trsSpecial, fromSpecial, toSpecial := transfer.GenerateTestTransfers(t, deps.db, 100, 3)

	// Here not to include the modified To and From addresses
	allAddresses := append(append(append(append(append(tdFromAdds, tdToAddrs...), fromAddrs...), toAddrs...), fromSpecial...), toSpecial...)

	// Insert MintAT Collectible
	trsSpecial[0].From = eth.HexToAddress("0x0")
	transfer.InsertTestTransferWithOptions(t, deps.db, trsSpecial[0].To, &trsSpecial[0], &transfer.TestTransferOptions{
		TokenAddress: eth.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
		TokenID:      (big.NewInt(1318)),
	})

	// Insert MintAT Token
	trsSpecial[1].From = eth.HexToAddress("0x0")
	inputMethod, err := hex.DecodeString("1b5ee6ae")
	require.NoError(t, err)
	transfer.InsertTestTransferWithOptions(t, deps.db, trsSpecial[1].To, &trsSpecial[1], &transfer.TestTransferOptions{
		TokenAddress: eth.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb49"),
		Tx:           transfer.GenerateTxField(inputMethod),
	})

	// Insert ContractDeploymentAt
	trsSpecial[2].To = eth.HexToAddress("0x0")
	transfer.InsertTestTransferWithOptions(t, deps.db, trsSpecial[2].From, &trsSpecial[2], &transfer.TestTransferOptions{
		NullifyAddresses: []eth.Address{trsSpecial[2].To},
	})

	// Test filtering out without address involved
	var filter Filter

	filter.Types = allActivityTypesFilter()
	// Set tr1 to Receive and pendingTr to Send; rest of two MT remain default Send
	addresses := []eth.Address{td.tr1.To, td.pendingTr.From, td.multiTx1.FromAddress, td.multiTx2.FromAddress, trs[0].From, trs[2].From, trs[4].From, trs[6].From, trs[8].From, trsSpecial[0].To, trsSpecial[1].To, trsSpecial[2].From}
	entries, err := getActivityEntries(context.Background(), deps, addresses, false, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 12, len(entries))

	filter.Types = []Type{SendAT, SwapAT}
	entries, err = getActivityEntries(context.Background(), deps, addresses, false, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)

	// 3 from td Send + 2 trs MT Send + 1 (swap)
	require.Equal(t, 6, len(entries))

	sendCount, receiveCount, contractCount, mintCount, swapCount, _, bridgeCount := countTypes(entries)

	require.Equal(t, 5, sendCount)
	require.Equal(t, 0, receiveCount)
	require.Equal(t, 0, contractCount)
	require.Equal(t, 0, mintCount)
	require.Equal(t, 1, swapCount)
	require.Equal(t, 0, bridgeCount)

	filter.Types = []Type{BridgeAT, ReceiveAT}
	entries, err = getActivityEntries(context.Background(), deps, addresses, false, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 3, len(entries))

	sendCount, receiveCount, contractCount, mintCount, swapCount, _, bridgeCount = countTypes(entries)
	require.Equal(t, 0, sendCount)
	require.Equal(t, 1, receiveCount)
	require.Equal(t, 0, contractCount)
	require.Equal(t, 0, mintCount)
	require.Equal(t, 0, swapCount)
	require.Equal(t, 2, bridgeCount)

	filter.Types = []Type{MintAT}
	entries, err = getActivityEntries(context.Background(), deps, addresses, false, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 2, len(entries))

	sendCount, receiveCount, contractCount, mintCount, swapCount, _, bridgeCount = countTypes(entries)
	require.Equal(t, 0, sendCount)
	require.Equal(t, 0, receiveCount)
	require.Equal(t, 0, contractCount)
	require.Equal(t, 2, mintCount)
	require.Equal(t, 0, swapCount)
	require.Equal(t, 0, bridgeCount)

	filter.Types = []Type{ContractDeploymentAT}
	entries, err = getActivityEntries(context.Background(), deps, addresses, false, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 1, len(entries))

	sendCount, receiveCount, contractCount, mintCount, swapCount, _, bridgeCount = countTypes(entries)
	require.Equal(t, 0, sendCount)
	require.Equal(t, 0, receiveCount)
	require.Equal(t, 1, contractCount)
	require.Equal(t, 0, mintCount)
	require.Equal(t, 0, swapCount)
	require.Equal(t, 0, bridgeCount)

	// Filter with all addresses regression
	filter.Types = []Type{SendAT}
	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	// We have 6 but one is not matched because is a receive, having owner the to address
	require.Equal(t, 5, len(entries))
}

func TestStatusMintCustomEvent(t *testing.T) {
	deps, close := setupTestActivityDB(t)
	defer close()

	td, fromTds, toTds := fillTestData(t, deps.db)
	trs, fromTrs, toTrs := transfer.GenerateTestTransfers(t, deps.db, td.nextIndex, 3)

	allAddresses := append(append(append(fromTds, toTds...), fromTrs...), toTrs...)

	trs[0].From = eth.HexToAddress("0x0")
	transfer.InsertTestTransferWithOptions(t, deps.db, trs[0].To, &trs[0], &transfer.TestTransferOptions{
		TokenAddress: eth.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
		Receipt: &types.Receipt{
			Logs: []*types.Log{
				{Topics: []eth.Hash{eth.HexToHash("0xea667487ed28493de38fd2808b00affaee21d875a9e95aa01ef8352151292297")}},
				{Topics: []eth.Hash{eth.HexToHash("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925")}},
			},
		},
	})
	// StatusMint - 0x28c427b0611d99da5c4f7368abe57e86b045b483c4689ae93e90745802335b87
	trs[1].From = eth.HexToAddress("0x0")
	transfer.InsertTestTransferWithOptions(t, deps.db, trs[1].To, &trs[1], &transfer.TestTransferOptions{
		TokenAddress: eth.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb49"),
		Receipt: &types.Receipt{
			Logs: []*types.Log{
				{Topics: []eth.Hash{eth.HexToHash("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925")}},
				{Topics: []eth.Hash{eth.HexToHash("0x28c427b0611d99da5c4f7368abe57e86b045b483c4689ae93e90745802335b87")}},
			},
		},
	})

	// Log order should not matter
	trs[2].From = eth.HexToAddress("0x0")
	transfer.InsertTestTransferWithOptions(t, deps.db, trs[2].To, &trs[2], &transfer.TestTransferOptions{
		TokenAddress: eth.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb49"),
		Receipt: &types.Receipt{
			Logs: []*types.Log{
				{Topics: []eth.Hash{eth.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")}},
				{Topics: []eth.Hash{eth.HexToHash("0x28c427b0611d99da5c4f7368abe57e86b045b483c4689ae93e90745802335b87")}},
				{Topics: []eth.Hash{eth.HexToHash("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925")}},
			},
		},
	})

	var filter Filter

	entries, err := getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 7, len(entries))

	filter.Types = []Type{MintAT}

	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 2, len(entries))
	require.Equal(t, trs[2].Hash, entries[0].transaction.Hash)
	require.Equal(t, trs[1].Hash, entries[1].transaction.Hash)
}

func TestGetActivityEntriesFilterByAddresses(t *testing.T) {
	deps, close := setupTestActivityDB(t)
	defer close()

	// Adds 4 extractable transactions
	td, fromTds, toTds := fillTestData(t, deps.db)
	trs, fromTrs, toTrs := transfer.GenerateTestTransfers(t, deps.db, td.nextIndex, 6)
	for i := range trs {
		transfer.InsertTestTransfer(t, deps.db, trs[i].From, &trs[i])
	}

	allAddresses := append(append(append(fromTds, toTds...), fromTrs...), toTrs...)

	var filter Filter

	entries, err := getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 10, len(entries))

	addressesFilter := []eth.Address{td.multiTx1.ToAddress, td.multiTx2.FromAddress, trs[1].From, trs[4].From, trs[3].To}
	// The td.multiTx1.ToAddress and trs[3].To are missing not having them as owner address
	entries, err = getActivityEntries(context.Background(), deps, addressesFilter, false, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 3, len(entries))
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: trs[4].ChainID, Hash: trs[4].Hash, Address: trs[4].From},
		id:             0,
		timestamp:      trs[4].Timestamp,
		activityType:   SendAT,
		activityStatus: FinalizedAS,
		amountOut:      (*hexutil.Big)(big.NewInt(trs[4].Value)),
		amountIn:       (*hexutil.Big)(big.NewInt(0)),
		tokenOut:       TTrToToken(t, &trs[4].TestTransaction),
		tokenIn:        nil,
		symbolOut:      common.NewAndSet("USDC"),
		symbolIn:       nil,
		sender:         &trs[4].From,
		recipient:      &trs[4].To,
		chainIDOut:     &trs[4].ChainID,
		chainIDIn:      nil,
		transferType:   expectedTokenType(trs[4].Token.Address),
	}, entries[0])
	require.Equal(t, Entry{
		payloadType:    SimpleTransactionPT,
		transaction:    &transfer.TransactionIdentity{ChainID: trs[1].ChainID, Hash: trs[1].Hash, Address: trs[1].From},
		id:             0,
		timestamp:      trs[1].Timestamp,
		activityType:   SendAT,
		activityStatus: FinalizedAS,
		amountOut:      (*hexutil.Big)(big.NewInt(trs[1].Value)),
		amountIn:       (*hexutil.Big)(big.NewInt(0)),
		tokenOut:       TTrToToken(t, &trs[1].TestTransaction),
		tokenIn:        nil,
		symbolOut:      common.NewAndSet("ETH"),
		symbolIn:       nil,
		sender:         &trs[1].From,
		recipient:      &trs[1].To,
		chainIDOut:     &trs[1].ChainID,
		chainIDIn:      nil,
		transferType:   expectedTokenType(trs[1].Token.Address),
	}, entries[1])
	require.Equal(t, Entry{
		payloadType:    MultiTransactionPT,
		transaction:    nil,
		id:             td.multiTx2ID,
		timestamp:      td.multiTx2.Timestamp,
		activityType:   SendAT,
		activityStatus: PendingAS,
		amountOut:      (*hexutil.Big)(big.NewInt(td.multiTx2.FromAmount)),
		amountIn:       (*hexutil.Big)(big.NewInt(td.multiTx2.ToAmount)),
		tokenOut:       tokenFromSymbol(nil, td.multiTx2.FromToken),
		tokenIn:        tokenFromSymbol(nil, td.multiTx2.ToToken),
		symbolOut:      common.NewAndSet("USDC"),
		symbolIn:       common.NewAndSet("SNT"),
		sender:         &td.multiTx2.FromAddress,
		recipient:      &td.multiTx2.ToAddress,
		chainIDOut:     nil,
		chainIDIn:      nil,
	}, entries[2])
}

func TestGetActivityEntriesFilterByStatus(t *testing.T) {
	deps, close := setupTestActivityDB(t)
	defer close()

	// Adds 4 extractable transactions: 1 T, 1 T pending, 1 MT pending, 1 MT with 2xT finalized
	td, fromTds, toTds := fillTestData(t, deps.db)
	// Add 7 extractable transactions: 1 pending, 1 Tr failed, 1 MT failed, 4 finalized
	trs, fromTrs, toTrs := transfer.GenerateTestTransfers(t, deps.db, td.nextIndex, 7)
	multiTx := transfer.GenerateTestSendMultiTransaction(trs[6])
	failedMTID := transfer.InsertTestMultiTransaction(t, deps.db, &multiTx)
	trs[6].MultiTransactionID = failedMTID
	for i := range trs {
		if i == 1 {
			transfer.InsertTestPendingTransaction(t, deps.db, &trs[i])
		} else {
			trs[i].Success = i != 3 && i != 6
			if trs[i].Success && (i == 2 || i == 5) {
				// Finalize status depends on timestamp
				trs[i].Timestamp = mockupTime.Unix() - 10
			}
			transfer.InsertTestTransfer(t, deps.db, trs[i].To, &trs[i])
		}
	}

	allAddresses := append(append(append(fromTds, toTds...), fromTrs...), toTrs...)

	var filter Filter
	filter.Statuses = allActivityStatusesFilter()
	entries, err := getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 11, len(entries))

	filter.Statuses = []Status{PendingAS}
	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 3, len(entries))
	require.Equal(t, td.pendingTr.Hash, entries[2].transaction.Hash)
	require.Equal(t, td.multiTx2ID, entries[1].id)
	require.Equal(t, trs[1].Hash, entries[0].transaction.Hash)

	filter.Statuses = []Status{FailedAS}
	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 2, len(entries))

	filter.Statuses = []Status{CompleteAS}
	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 2, len(entries))

	filter.Statuses = []Status{FinalizedAS}
	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 4, len(entries))

	// Combined filter
	filter.Statuses = []Status{FailedAS, PendingAS}
	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 5, len(entries))
}

func TestGetActivityEntriesFilterByTokenType(t *testing.T) {
	deps, close := setupTestActivityDB(t)
	defer close()

	// Adds 4 extractable transactions 2 transactions (ETH/Goerli, ETH/Optimism), one MT USDC to DAI and another MT USDC to SNT
	td, fromTds, toTds := fillTestData(t, deps.db)
	// Add 9 transactions DAI/Goerli, ETH/Mainnet, ETH/Goerli, ETH/Optimism, USDC/Mainnet, USDC/Goerli, USDC/Optimism, SNT/Mainnet, DAI/Mainnet
	trs, fromTrs, toTrs := transfer.GenerateTestTransfers(t, deps.db, td.nextIndex, 9)
	for i := range trs {
		tokenAddr := transfer.TestTokens[i].Address
		trs[i].ChainID = common.ChainID(transfer.TestTokens[i].ChainID)
		transfer.InsertTestTransferWithOptions(t, deps.db, trs[i].To, &trs[i], &transfer.TestTransferOptions{
			TokenAddress: tokenAddr,
		})
	}

	allAddresses := append(append(append(fromTds, toTds...), fromTrs...), toTrs...)

	var filter Filter
	filter.FilterOutAssets = true
	entries, err := getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 0, len(entries))

	filter.FilterOutAssets = false
	filter.Assets = allTokensFilter()
	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 13, len(entries))

	// Native tokens are network agnostic, hence all are returned
	filter.Assets = []Token{{TokenType: Native, ChainID: common.ChainID(transfer.EthMainnet.ChainID)}}
	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 5, len(entries))

	// Test that it doesn't break the filter
	filter.Assets = []Token{{TokenType: Erc1155}}
	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 0, len(entries))

	filter.Assets = []Token{{
		TokenType: Erc20,
		ChainID:   common.ChainID(transfer.UsdcMainnet.ChainID),
		Address:   transfer.UsdcMainnet.Address,
	}}
	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	// Two MT for which ChainID is ignored and one transfer on the main net and the Goerli is ignored
	require.Equal(t, 3, len(entries))
	require.Equal(t, Erc20, entries[0].tokenIn.TokenType)
	require.Equal(t, transfer.UsdcMainnet.Address, entries[0].tokenIn.Address)
	require.Nil(t, entries[0].tokenOut)
	// MT has only symbol, the first token is lookup by symbol for both entries
	require.Equal(t, Erc20, entries[1].tokenOut.TokenType)
	require.Equal(t, transfer.UsdcMainnet.Address, entries[1].tokenOut.Address)
	require.Equal(t, Erc20, entries[1].tokenIn.TokenType)
	require.Equal(t, transfer.SntMainnet.Address, entries[1].tokenIn.Address)
	require.Equal(t, Erc20, entries[2].tokenOut.TokenType)
	require.Equal(t, transfer.UsdcMainnet.Address, entries[1].tokenOut.Address)
	require.Equal(t, Erc20, entries[2].tokenIn.TokenType)
	require.Equal(t, transfer.UsdcMainnet.Address, entries[1].tokenOut.Address)

	filter.Assets = []Token{{
		TokenType: Erc20,
		ChainID:   common.ChainID(transfer.UsdcMainnet.ChainID),
		Address:   transfer.UsdcMainnet.Address,
	}, {
		TokenType: Erc20,
		ChainID:   common.ChainID(transfer.UsdcGoerli.ChainID),
		Address:   transfer.UsdcGoerli.Address,
	}}
	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	// Two MT for which ChainID is ignored and two transfers on the main net and Goerli
	require.Equal(t, 4, len(entries))
	require.Equal(t, Erc20, entries[0].tokenIn.TokenType)
	require.Equal(t, transfer.UsdcGoerli.Address, entries[0].tokenIn.Address)
	require.Nil(t, entries[0].tokenOut)
}

func TestGetActivityEntriesFilterByCollectibles(t *testing.T) {
	deps, close := setupTestActivityDB(t)
	defer close()

	// Adds 4 extractable transactions 2 transactions (ETH/Goerli, ETH/Optimism), one MT USDC to DAI and another MT USDC to SNT
	td, fromTds, toTds := fillTestData(t, deps.db)
	// Add 4 transactions with collectibles
	trs, fromTrs, toTrs := transfer.GenerateTestTransfers(t, deps.db, td.nextIndex, 4)
	for i := range trs {
		collectibleData := transfer.TestCollectibles[i]
		trs[i].ChainID = collectibleData.ChainID
		transfer.InsertTestTransferWithOptions(t, deps.db, trs[i].To, &trs[i], &transfer.TestTransferOptions{
			TokenAddress: collectibleData.TokenAddress,
			TokenID:      collectibleData.TokenID,
		})
	}

	allAddresses := append(append(append(fromTds, toTds...), fromTrs...), toTrs...)

	var filter Filter
	filter.FilterOutCollectibles = true
	entries, err := getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 0, len(entries))

	filter.FilterOutCollectibles = false
	filter.Collectibles = allTokensFilter()
	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 8, len(entries))

	// Search for a specific collectible
	filter.Collectibles = []Token{tokenFromCollectible(&transfer.TestCollectibles[0])}
	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 1, len(entries))
	require.Equal(t, entries[0].tokenIn.Address, transfer.TestCollectibles[0].TokenAddress)
	require.Equal(t, entries[0].tokenIn.TokenID, (*hexutil.Big)(transfer.TestCollectibles[0].TokenID))

	// Search for a specific collectible
	filter.Collectibles = []Token{tokenFromCollectible(&transfer.TestCollectibles[3])}
	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 1, len(entries))
	require.Equal(t, entries[0].tokenIn.Address, transfer.TestCollectibles[3].TokenAddress)
	require.Equal(t, entries[0].tokenIn.TokenID, (*hexutil.Big)(transfer.TestCollectibles[3].TokenID))

	// Search for a multiple collectibles
	filter.Collectibles = []Token{tokenFromCollectible(&transfer.TestCollectibles[1]), tokenFromCollectible(&transfer.TestCollectibles[2])}
	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 2, len(entries))
}

func TestGetActivityEntriesFilterByToAddresses(t *testing.T) {
	deps, close := setupTestActivityDB(t)
	defer close()

	// Adds 4 extractable transactions
	td, fromTds, toTds := fillTestData(t, deps.db)
	// Add 6 extractable transactions
	trs, fromTrs, toTrs := transfer.GenerateTestTransfers(t, deps.db, td.nextIndex, 6)
	for i := range trs {
		transfer.InsertTestTransfer(t, deps.db, trs[i].To, &trs[i])
	}

	allAddresses := append(append(append(fromTds, toTds...), fromTrs...), toTrs...)

	var filter Filter
	filter.CounterpartyAddresses = allAddresses
	entries, err := getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 10, len(entries))

	filter.CounterpartyAddresses = []eth.Address{eth.HexToAddress("0x567890")}
	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 0, len(entries))

	filter.CounterpartyAddresses = []eth.Address{td.pendingTr.To, td.multiTx2.ToAddress, trs[3].To}
	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 3, len(entries))

	filter.CounterpartyAddresses = []eth.Address{td.tr1.To, td.pendingTr.From, trs[3].From, trs[5].To}
	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 2, len(entries))
}

func TestGetActivityEntriesFilterByNetworks(t *testing.T) {
	deps, close := setupTestActivityDB(t)
	defer close()

	// Adds 4 extractable transactions
	td, fromTds, toTds := fillTestData(t, deps.db)

	chainToEntryCount := make(map[common.ChainID]map[int]int)
	recordPresence := func(chainID common.ChainID, entry int) {
		if _, ok := chainToEntryCount[chainID]; !ok {
			chainToEntryCount[chainID] = make(map[int]int)
			chainToEntryCount[chainID][entry] = 1
		} else {
			if _, ok := chainToEntryCount[chainID][entry]; !ok {
				chainToEntryCount[chainID][entry] = 1
			} else {
				chainToEntryCount[chainID][entry]++
			}
		}
	}
	recordPresence(td.tr1.ChainID, 0)
	recordPresence(td.pendingTr.ChainID, 1)
	recordPresence(td.multiTx1Tr1.ChainID, 2)
	if td.multiTx1Tr2.ChainID != td.multiTx1Tr1.ChainID {
		recordPresence(td.multiTx1Tr2.ChainID, 2)
	}
	recordPresence(td.multiTx2Tr1.ChainID, 3)
	if td.multiTx2Tr2.ChainID != td.multiTx2Tr1.ChainID {
		recordPresence(td.multiTx2Tr2.ChainID, 3)
	}
	if td.multiTx2PendingTr.ChainID != td.multiTx2Tr1.ChainID && td.multiTx2PendingTr.ChainID != td.multiTx2Tr2.ChainID {
		recordPresence(td.multiTx2PendingTr.ChainID, 3)
	}

	// Add 6 extractable transactions
	trs, fromTrs, toTrs := transfer.GenerateTestTransfers(t, deps.db, td.nextIndex, 6)
	for i := range trs {
		recordPresence(trs[i].ChainID, 4+i)
		transfer.InsertTestTransfer(t, deps.db, trs[i].To, &trs[i])
	}
	allAddresses := append(append(append(fromTds, toTds...), fromTrs...), toTrs...)

	var filter Filter
	chainIDs := allNetworksFilter()
	entries, err := getActivityEntries(context.Background(), deps, allAddresses, true, chainIDs, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 10, len(entries))

	chainIDs = []common.ChainID{5674839210}
	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, chainIDs, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 0, len(entries))

	chainIDs = []common.ChainID{td.pendingTr.ChainID, td.multiTx2Tr1.ChainID, trs[3].ChainID}
	entries, err = getActivityEntries(context.Background(), deps, allAddresses, true, chainIDs, filter, 0, 15)
	require.NoError(t, err)
	expectedResults := make(map[int]int)
	for _, chainID := range chainIDs {
		for entry := range chainToEntryCount[chainID] {
			if _, ok := expectedResults[entry]; !ok {
				expectedResults[entry]++
			}
		}
	}
	require.Equal(t, len(expectedResults), len(entries))
}

func TestGetActivityEntriesFilterByNetworksOfSubTransactions(t *testing.T) {
	deps, close := setupTestActivityDB(t)
	defer close()

	// Add 6 extractable transactions
	trs, _, toTrs := transfer.GenerateTestTransfers(t, deps.db, 0, 5)
	trs[0].ChainID = 1231
	trs[1].ChainID = 1232
	trs[2].ChainID = 1233
	mt1 := transfer.GenerateTestBridgeMultiTransaction(trs[0], trs[1])
	trs[0].MultiTransactionID = transfer.InsertTestMultiTransaction(t, deps.db, &mt1)
	trs[1].MultiTransactionID = mt1.MultiTransactionID
	trs[2].MultiTransactionID = mt1.MultiTransactionID

	trs[3].ChainID = 1234
	mt2 := transfer.GenerateTestSwapMultiTransaction(trs[3], testutils.SntSymbol, 100)
	// insertMultiTransaction will insert 0 instead of NULL
	mt2.FromNetworkID = common.NewAndSet(uint64(0))
	mt2.ToNetworkID = common.NewAndSet(uint64(0))
	trs[3].MultiTransactionID = transfer.InsertTestMultiTransaction(t, deps.db, &mt2)

	for i := range trs {
		if i == 2 {
			transfer.InsertTestPendingTransaction(t, deps.db, &trs[i])
		} else {
			transfer.InsertTestTransfer(t, deps.db, trs[i].To, &trs[i])
		}
	}

	var filter Filter
	chainIDs := allNetworksFilter()
	entries, err := getActivityEntries(context.Background(), deps, toTrs, false, chainIDs, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 3, len(entries))

	chainIDs = []common.ChainID{trs[0].ChainID, trs[1].ChainID}
	entries, err = getActivityEntries(context.Background(), deps, toTrs, false, chainIDs, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 1, len(entries))
	require.Equal(t, entries[0].id, mt1.MultiTransactionID)

	// Filter by pending_transactions sub-transacitons
	chainIDs = []common.ChainID{trs[2].ChainID}
	entries, err = getActivityEntries(context.Background(), deps, toTrs, false, chainIDs, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 1, len(entries))
	require.Equal(t, entries[0].id, mt1.MultiTransactionID)

	chainIDs = []common.ChainID{trs[3].ChainID}
	entries, err = getActivityEntries(context.Background(), deps, toTrs, false, chainIDs, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 1, len(entries))
	require.Equal(t, entries[0].id, mt2.MultiTransactionID)
}

func TestGetActivityEntriesCheckToAndFrom(t *testing.T) {
	deps, close := setupTestActivityDB(t)
	defer close()

	// Adds 6 transactions from which 4 are filtered out
	td, _, _ := fillTestData(t, deps.db)

	// Add extra transactions to test To address
	trs, _, _ := transfer.GenerateTestTransfers(t, deps.db, td.nextIndex, 2)
	transfer.InsertTestTransfer(t, deps.db, trs[0].To, &trs[0])
	transfer.InsertTestPendingTransaction(t, deps.db, &trs[1])

	addresses := []eth.Address{td.tr1.To, td.pendingTr.To,
		td.multiTx1.FromAddress, td.multiTx2.FromAddress, trs[0].To, trs[1].To}

	var filter Filter
	entries, err := getActivityEntries(context.Background(), deps, addresses, false, []common.ChainID{}, filter, 0, 15)
	require.NoError(t, err)
	require.Equal(t, 6, len(entries))

	require.Equal(t, ReceiveAT, entries[5].activityType)               // td.tr1
	require.NotEqual(t, eth.Address{}, entries[5].transaction.Address) // td.tr1
	require.Equal(t, td.tr1.To, *entries[5].recipient)                 // td.tr1

	require.Equal(t, ReceiveAT, entries[4].activityType) // td.pendingTr

	// Multi-transactions are always considered as SendAT
	require.Equal(t, SendAT, entries[3].activityType) // td.multiTx1
	require.Equal(t, SendAT, entries[2].activityType) // td.multiTx2

	require.Equal(t, ReceiveAT, entries[1].activityType)               // trs[0]
	require.NotEqual(t, eth.Address{}, entries[1].transaction.Address) // trs[0]
	require.Equal(t, trs[0].To, entries[1].transaction.Address)        // trs[0]

	require.Equal(t, ReceiveAT, entries[0].activityType) // trs[1] (pending)
}

func TestGetActivityEntriesCheckContextCancellation(t *testing.T) {
	deps, close := setupTestActivityDB(t)
	defer close()

	_, fromAddresses, toAddresses := fillTestData(t, deps.db)

	cancellableCtx, cancelFn := context.WithCancel(context.Background())
	cancelFn()

	activities, err := getActivityEntries(cancellableCtx, deps, append(fromAddresses, toAddresses...), true, []common.ChainID{}, Filter{}, 0, 10)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 0, len(activities))
}

func TestGetActivityEntriesNullAddresses(t *testing.T) {
	deps, close := setupTestActivityDB(t)
	defer close()

	trs, _, _ := transfer.GenerateTestTransfers(t, deps.db, 0, 4)
	multiTx := transfer.GenerateTestBridgeMultiTransaction(trs[0], trs[1])
	multiTx.ToAddress = eth.Address{}

	trs[0].MultiTransactionID = transfer.InsertTestMultiTransaction(t, deps.db, &multiTx)
	trs[1].MultiTransactionID = trs[0].MultiTransactionID

	for i := 0; i < 3; i++ {
		transfer.InsertTestTransferWithOptions(t, deps.db, trs[i].From, &trs[i], &transfer.TestTransferOptions{
			NullifyAddresses: []eth.Address{trs[i].To},
		})
	}

	trs[3].To = eth.Address{}
	transfer.InsertTestPendingTransaction(t, deps.db, &trs[3])

	addresses := []eth.Address{trs[0].From, trs[1].From, trs[2].From, trs[3].From}

	activities, err := getActivityEntries(context.Background(), deps, addresses, false, allNetworksFilter(), Filter{}, 0, 10)
	require.NoError(t, err)
	require.Equal(t, 3, len(activities))
}

func TestGetActivityEntries_ErrorIfNoAddress(t *testing.T) {
	_, err := getActivityEntries(context.Background(), FilterDependencies{}, []eth.Address{}, true, []common.ChainID{}, Filter{}, 0, 10)
	require.EqualError(t, err, "no addresses provided")
}

func TestGetTxDetails(t *testing.T) {
	deps, close := setupTestActivityDB(t)
	defer close()

	// Adds 4 extractable transactions 2 transactions (ETH/Goerli, ETH/Optimism), one MT USDC to DAI and another MT USDC to SNT
	td, _, _ := fillTestData(t, deps.db)

	_, err := getTxDetails(context.Background(), deps.db, "")
	require.EqualError(t, err, "invalid tx id")

	details, err := getTxDetails(context.Background(), deps.db, td.tr1.Hash.String())
	require.NoError(t, err)

	require.Equal(t, td.tr1.Hash.String(), details.ID)
	require.Equal(t, 0, details.MultiTxID)
	require.Equal(t, td.tr1.Nonce, details.Nonce)
	require.Equal(t, len(details.ChainDetails), 1)
	require.Equal(t, td.tr1.ChainID, common.ChainID(details.ChainDetails[0].ChainID))
	require.Equal(t, td.tr1.BlkNumber, details.ChainDetails[0].BlockNumber)
	require.Equal(t, td.tr1.Hash, details.ChainDetails[0].Hash)
	require.Equal(t, td.tr1.Contract, *details.ChainDetails[0].Contract)
}

func TestGetMultiTxDetails(t *testing.T) {
	deps, close := setupTestActivityDB(t)
	defer close()

	// Adds 4 extractable transactions 2 transactions (ETH/Goerli, ETH/Optimism), one MT USDC to DAI and another MT USDC to SNT
	td, _, _ := fillTestData(t, deps.db)

	_, err := getMultiTxDetails(context.Background(), deps.db, 0)
	require.EqualError(t, err, "invalid tx id")

	details, err := getMultiTxDetails(context.Background(), deps.db, int(td.multiTx1.MultiTransactionID))
	require.NoError(t, err)

	require.Equal(t, "", details.ID)
	require.Equal(t, int(td.multiTx1.MultiTransactionID), details.MultiTxID)
	require.Equal(t, td.multiTx1Tr2.Nonce, details.Nonce)
	require.Equal(t, 2, len(details.ChainDetails))
	require.Equal(t, td.multiTx1Tr1.ChainID, common.ChainID(details.ChainDetails[0].ChainID))
	require.Equal(t, td.multiTx1Tr1.BlkNumber, details.ChainDetails[0].BlockNumber)
	require.Equal(t, td.multiTx1Tr1.Hash, details.ChainDetails[0].Hash)
	require.Equal(t, td.multiTx1Tr1.Contract, *details.ChainDetails[0].Contract)
	require.Equal(t, td.multiTx1Tr2.ChainID, common.ChainID(details.ChainDetails[1].ChainID))
	require.Equal(t, td.multiTx1Tr2.BlkNumber, details.ChainDetails[1].BlockNumber)
	require.Equal(t, td.multiTx1Tr2.Hash, details.ChainDetails[1].Hash)
	require.Equal(t, td.multiTx1Tr2.Contract, *details.ChainDetails[1].Contract)
}

func TestGetActivityEntriesSkipEthGasFeeOnlyTransfers(t *testing.T) {
	deps, close := setupTestActivityDB(t)
	defer close()

	to := eth.Address{0x1}
	from := eth.Address{0x2}
	hash := eth.Hash{0x3}
	blkNum := int64(1)
	chainID := common.ChainID(1)
	nonce := uint64(1)

	// Insert 0-value gas-only ETH transfer as a result of token transfer's gas fee
	transfer.InsertTestTransfer(t, deps.db, to, &transfer.TestTransfer{
		TestTransaction: transfer.TestTransaction{
			ChainID:   chainID,
			From:      from,
			Hash:      hash,
			BlkNumber: blkNum,
			Nonce:     nonce,
		},
		To:    to,
		Value: 0,
	})

	entries, err := getActivityEntries(context.Background(), deps, []eth.Address{to}, true, []common.ChainID{chainID}, Filter{}, 0, 10)
	require.NoError(t, err)
	require.Equal(t, 1, len(entries))
	require.Equal(t, hash, entries[0].transaction.Hash)

	// Insert token transfer
	transfer.InsertTestTransferWithOptions(t, deps.db, to,
		&transfer.TestTransfer{
			TestTransaction: transfer.TestTransaction{
				ChainID:   chainID,
				From:      from,
				Hash:      hash,
				BlkNumber: blkNum,
				Nonce:     nonce,
			},
			To:    to,
			Value: 1,
		},
		&transfer.TestTransferOptions{
			TokenAddress: eth.Address{0x4},
		},
	)

	// Gas-fee-only transfer should be removed, so we get only 1 transfer again
	entries, err = getActivityEntries(context.Background(), deps, []eth.Address{to}, true, []common.ChainID{chainID}, Filter{}, 0, 10)
	require.NoError(t, err)
	require.Equal(t, 1, len(entries))
	require.Equal(t, contractTypeFromDBType("erc20"), entries[0].transferType)

	// Insert real 0-value ETH transfer
	transfer.InsertTestTransfer(t, deps.db, to, &transfer.TestTransfer{
		TestTransaction: transfer.TestTransaction{
			ChainID:   chainID,
			From:      from,
			Hash:      eth.Hash{0x5}, // another hash
			BlkNumber: blkNum,
			Nonce:     nonce + 1, // another nonce
		},
		To:    to,
		Value: 0, // 0-value as well
	})

	entries, err = getActivityEntries(context.Background(), deps, []eth.Address{to}, true, []common.ChainID{chainID}, Filter{}, 0, 10)
	require.NoError(t, err)
	require.Equal(t, 2, len(entries))
}
