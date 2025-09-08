package activity

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	eth "github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	gethParams "github.com/ethereum/go-ethereum/params"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/params"
	ac "github.com/status-im/status-go/services/wallet/activity/common"
	"github.com/status-im/status-go/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/requests"
	"github.com/status-im/status-go/services/wallet/routeexecution/storage"
	"github.com/status-im/status-go/services/wallet/router/routes"
	"github.com/status-im/status-go/services/wallet/thirdparty/activity/alchemy"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/transactions"
	"github.com/status-im/status-go/walletdatabase"
)

const (
	EthDecimals      = 18
	StandardGasLimit = 21000
	DefaultNonce     = 1
	TestBlockNumber  = 16777216 // 0x1000000

	TestChainID = uint64(1)
	TestAsset   = "ETH"

	DefaultLimit  = 10
	DefaultOffset = 0
)

var (
	TestSenderAddr    = eth.HexToAddress("0x1111111111111111111111111111111111111111")
	TestReceiverAddr  = eth.HexToAddress("0x2222222222222222222222222222222222222222")
	TestExternalAddr  = eth.HexToAddress("0x3333333333333333333333333333333333333333")
	TestExternalAddr2 = eth.HexToAddress("0x4444444444444444444444444444444444444444")
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	return db, func() {
		require.NoError(t, db.Close())
	}
}

// createRouteInputParams creates route input parameters for testing
func createRouteInputParams(uuid string, fromAddr eth.Address, toAddr eth.Address, value *big.Int, asset string) *requests.RouteInputParams {
	return &requests.RouteInputParams{
		Uuid:               uuid,
		AddrFrom:           fromAddr,
		AddrTo:             toAddr,
		AmountIn:           (*hexutil.Big)(value),
		TokenID:            asset,
		SendType:           0,
		SlippagePercentage: 0,
	}
}

// createEthereumTransaction creates a test Ethereum transaction
func createEthereumTransaction(toAddr eth.Address, value *big.Int) *ethTypes.Transaction {
	gasLimit := uint64(StandardGasLimit)
	gasPrice := big.NewInt(gethParams.GWei)
	nonce := uint64(DefaultNonce)
	return ethTypes.NewTransaction(nonce, toAddr, value, gasLimit, gasPrice, nil)
}

// createTransactionData creates wallet transaction data for testing
func createTransactionData(fromAddr eth.Address, toAddr eth.Address, value *big.Int, txHash eth.Hash, tx *ethTypes.Transaction) *wallettypes.TransactionData {
	toAddress := types.Address(toAddr)
	return &wallettypes.TransactionData{
		TxArgs: &wallettypes.SendTxArgs{
			From:  types.Address(fromAddr),
			To:    &toAddress,
			Value: (*hexutil.Big)(value),
		},
		Tx:         tx,
		HashToSign: types.Hash(txHash),
		Signature:  []byte("test_signature"),
		SentHash:   types.Hash(txHash),
	}
}

// createRouterPath creates a router path for testing
func createRouterPath(chainID uint64, asset string, value *big.Int) *routes.Path {
	return &routes.Path{
		FromChain: &params.Network{
			ChainID: chainID,
		},
		FromToken: &tokenTypes.Token{
			Symbol:  asset,
			ChainID: chainID,
		},
		ToToken: &tokenTypes.Token{
			Symbol:  asset,
			ChainID: chainID,
		},
		AmountIn: (*hexutil.Big)(value),
	}
}

// saveRouteData saves route data to the database
func saveRouteData(t *testing.T, db *sql.DB, routeInputParams *requests.RouteInputParams, routerPath *routes.Path, txData *wallettypes.TransactionData) {
	routerTxDetails := &wallettypes.RouterTransactionDetails{
		RouterPath: routerPath,
		TxData:     txData,
	}

	routeData := wallettypes.NewRouteData(routeInputParams, []*wallettypes.RouterTransactionDetails{routerTxDetails})

	routeDB := storage.NewDB(db)
	err := routeDB.PutRouteData(routeData)
	require.NoError(t, err)
}

// saveTrackedTransaction saves a tracked transaction to the database
func saveTrackedTransaction(t *testing.T, db *sql.DB, chainID uint64, txHash eth.Hash, timestamp int64) {
	trackedTxDB := transactions.NewDB(db)
	err := trackedTxDB.PutTx(transactions.TrackedTx{
		ID: transactions.TxIdentity{
			ChainID: walletCommon.ChainID(chainID),
			Hash:    txHash,
		},
		Status:    ac.Success,
		Timestamp: uint64(timestamp),
	})
	require.NoError(t, err)
}

func insertTestSavedTransaction(t *testing.T, db *sql.DB, chainID uint64, txHash eth.Hash, fromAddr eth.Address, toAddr eth.Address, value *big.Int, asset string, timestamp int64, uuid string) {
	routeInputParams := createRouteInputParams(uuid, fromAddr, toAddr, value, asset)
	tx := createEthereumTransaction(toAddr, value)
	txData := createTransactionData(fromAddr, toAddr, value, txHash, tx)
	routerPath := createRouterPath(chainID, asset, value)

	saveRouteData(t, db, routeInputParams, routerPath, txData)
	saveTrackedTransaction(t, db, chainID, txHash, timestamp)
}

func insertTestAlchemyTransfer(t *testing.T, db *sql.DB, chainID uint64, address eth.Address, txHash eth.Hash, timestamp int64, value float64, asset string, fromAddr eth.Address, toAddr eth.Address) {
	blockNum := big.NewInt(TestBlockNumber)

	ethFloat := big.NewFloat(value)
	weiFloat := new(big.Float).Mul(ethFloat, big.NewFloat(gethParams.Ether))
	weiValue, _ := weiFloat.Int(nil)

	transfer := alchemy.Transfer{
		Category:    alchemy.TransferCategoryExternal,
		BlockNum:    &bigint.VarHexBigInt{Int: blockNum},
		FromAddress: fromAddr,
		ToAddress:   &toAddr,
		Value:       value,
		Asset:       asset,
		UniqueID:    fmt.Sprintf("%s:external", txHash.Hex()),
		Hash:        txHash,
		RawContract: alchemy.RawContract{
			Value:   &bigint.VarHexBigInt{Int: weiValue},
			Address: nil, // nil for ETH transfers, would need token address for ERC20
			Decimal: &bigint.VarHexBigInt{Int: big.NewInt(EthDecimals)},
		},
		Metadata: alchemy.Metadata{
			BlockTimestamp: time.Unix(timestamp, 0),
		},
	}

	persistence := alchemy.NewPersistence(db)
	err := persistence.SaveTransfers([]alchemy.Transfer{transfer}, chainID, address)
	require.NoError(t, err)
}

// Creates internal transfer where sender account sends to receiver account (both in our wallet)
func createInternalTransferScenario(t *testing.T, db *sql.DB, senderAddr eth.Address, receiverAddr eth.Address, txHash eth.Hash, timestamp int64, value float64, asset string) {
	chainID := TestChainID
	uuid := fmt.Sprintf("uuid-%d", timestamp)

	ethFloat := big.NewFloat(value)
	weiFloat := new(big.Float).Mul(ethFloat, big.NewFloat(gethParams.Ether))
	weiValue, _ := weiFloat.Int(nil)

	insertTestSavedTransaction(t, db, chainID, txHash, senderAddr, receiverAddr, weiValue, asset, timestamp, uuid)
	insertTestAlchemyTransfer(t, db, chainID, senderAddr, txHash, timestamp, value, asset, senderAddr, receiverAddr)
	insertTestAlchemyTransfer(t, db, chainID, receiverAddr, txHash, timestamp, value, asset, senderAddr, receiverAddr)
}

// insertTestPendingTransaction inserts a pending transaction for testing
func insertTestPendingTransaction(t *testing.T, db *sql.DB, chainID uint64, txHash eth.Hash, fromAddr eth.Address, toAddr eth.Address, value *big.Int, asset string, timestamp int64, uuid string) {
	routeInputParams := createRouteInputParams(uuid, fromAddr, toAddr, value, asset)
	tx := createEthereumTransaction(toAddr, value)
	txData := createTransactionData(fromAddr, toAddr, value, txHash, tx)
	routerPath := createRouterPath(chainID, asset, value)

	saveRouteData(t, db, routeInputParams, routerPath, txData)

	trackedTxDB := transactions.NewDB(db)
	err := trackedTxDB.PutTx(transactions.TrackedTx{
		ID: transactions.TxIdentity{
			ChainID: walletCommon.ChainID(chainID),
			Hash:    txHash,
		},
		Status:    ac.Pending,
		Timestamp: uint64(timestamp),
	})
	require.NoError(t, err)
}

// TestGetTransactionsOrder_SenderView tests transaction filtering from the sender's perspective
// in an individual account view scenario.
//
// Test Scenario:
// - Sender account sends 0.001 ETH to receiver account (internal transfer within our wallet)
// - Sender account receives 0.002 ETH from external address (external incoming transfer)
// - Query transactions for sender address only
//
// Expected Behavior:
// - Internal transfer should appear as "sent" source (because sender initiated it)
// - External transfer should appear as "fetched" source (from blockchain data)
// - Both transactions should be associated with sender address
// - Results should be ordered by timestamp (newest first)
// - Total count should be 2 transactions
func TestGetTransactionsOrder_SenderView(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	chainID := TestChainID
	senderAddr := TestSenderAddr
	receiverAddr := TestReceiverAddr
	externalAddr := TestExternalAddr
	txHash := eth.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	timestamp := int64(1757333323)

	createInternalTransferScenario(t, db, senderAddr, receiverAddr, txHash, timestamp, 0.001, TestAsset)

	otherTxHash := eth.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	insertTestAlchemyTransfer(t, db, chainID, senderAddr, otherTxHash, timestamp-1000, 0.002, TestAsset,
		externalAddr, senderAddr)

	addresses := []eth.Address{senderAddr}
	chainIDs := []walletCommon.ChainID{walletCommon.ChainID(chainID)}

	results, totalCount, err := getTransactionsOrder(context.Background(), db, addresses, chainIDs, Filter{}, DefaultOffset, DefaultLimit)

	require.NoError(t, err)
	require.Equal(t, int64(2), totalCount, "Should have 2 transactions for sender")
	require.Len(t, results, 2)

	foundSent := false
	foundOther := false
	for _, tx := range results {
		if tx.Hash == txHash {
			require.Equal(t, SentTransaction, tx.Source, "Internal transfer should be 'sent' for sender")
			require.Equal(t, senderAddr, tx.Address, "Address should be sender")
			foundSent = true
		}
		if tx.Hash == otherTxHash {
			require.Equal(t, FetchedTransaction, tx.Source, "Other transaction should be 'fetched'")
			require.Equal(t, senderAddr, tx.Address, "Address should be sender")
			foundOther = true
		}
	}

	require.True(t, foundSent, "Should find sent internal transfer")
	require.True(t, foundOther, "Should find other fetched transaction")
	require.Equal(t, txHash, results[0].Hash, "Newer transaction should be first")
	require.Equal(t, timestamp, results[0].Timestamp, "Timestamp should match")
}

// TestGetTransactionsOrder_ReceiverView tests transaction filtering from the receiver's perspective
// in an individual account view scenario.
//
// Test Scenario:
// - Sender account sends 0.001 ETH to receiver account (internal transfer within our wallet)
// - Receiver account receives 0.003 ETH from external address (external incoming transfer)
// - Query transactions for receiver address only
//
// Expected Behavior:
// - Internal transfer should appear as "fetched" source (receiver didn't initiate it)
// - External transfer should appear as "fetched" source (from blockchain data)
// - Both transactions should be associated with receiver address
// - Results should be ordered by timestamp (newest first)
// - Total count should be 2 transactions
// - CRITICAL: Receiver should NOT see the "sent" version of the internal transfer
func TestGetTransactionsOrder_ReceiverView(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	chainID := TestChainID
	senderAddr := TestSenderAddr
	receiverAddr := TestReceiverAddr
	externalAddr := TestExternalAddr2
	txHash := eth.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	timestamp := int64(1757333323)

	createInternalTransferScenario(t, db, senderAddr, receiverAddr, txHash, timestamp, 0.001, TestAsset)

	otherTxHash := eth.HexToHash("0xcccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
	insertTestAlchemyTransfer(t, db, chainID, receiverAddr, otherTxHash, timestamp-2000, 0.003, TestAsset,
		externalAddr, receiverAddr)

	addresses := []eth.Address{receiverAddr}
	chainIDs := []walletCommon.ChainID{walletCommon.ChainID(chainID)}

	results, totalCount, err := getTransactionsOrder(context.Background(), db, addresses, chainIDs, Filter{}, DefaultOffset, DefaultLimit)

	require.NoError(t, err)
	require.Equal(t, int64(2), totalCount, "Should have 2 transactions for receiver")
	require.Len(t, results, 2)

	foundReceived := false
	foundOther := false
	for _, tx := range results {
		if tx.Hash == txHash {
			require.Equal(t, FetchedTransaction, tx.Source, "Internal transfer should be 'fetched' for receiver")
			require.Equal(t, receiverAddr, tx.Address, "Address should be receiver")
			foundReceived = true
		}
		if tx.Hash == otherTxHash {
			require.Equal(t, FetchedTransaction, tx.Source, "Other transaction should be 'fetched'")
			require.Equal(t, receiverAddr, tx.Address, "Address should be receiver")
			foundOther = true
		}
	}

	require.True(t, foundReceived, "Should find received internal transfer")
	require.True(t, foundOther, "Should find other fetched transaction")
	require.Equal(t, txHash, results[0].Hash, "Newer transaction should be first")

	for _, tx := range results {
		if tx.Hash == txHash {
			require.NotEqual(t, SentTransaction, tx.Source, "Receiver should NOT see 'sent' version of internal transfer")
		}
	}
}

// TestGetTransactionsOrder_AggregatedView tests the core deduplication logic when querying
// multiple addresses simultaneously (aggregated wallet view).
//
// Test Scenario:
// - Internal transfer: sender -> receiver (0.001 ETH)
// - External transfer to sender: external -> sender (0.002 ETH)
// - External transfer to receiver: external -> receiver (0.003 ETH)
// - Query both sender AND receiver addresses together
//
// Expected Behavior:
// - Internal transfer should appear TWICE (once per address with correct perspective)
// - Internal transfer for sender should show as "sent" source (sender initiated it)
// - Internal transfer for receiver should show as "fetched" source (receiver received it)
// - External transfers should appear correctly for their respective addresses
// - Total count should be 4 transactions (2 perspectives of internal + 2 external)
// - All transactions ordered by timestamp (newest first)
func TestGetTransactionsOrder_AggregatedView(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	chainID := TestChainID
	senderAddr := TestSenderAddr
	receiverAddr := TestReceiverAddr
	externalAddr := TestExternalAddr
	externalAddr2 := TestExternalAddr2

	internalTxHash := eth.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	externalTxHash1 := eth.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	externalTxHash2 := eth.HexToHash("0xcccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")

	baseTimestamp := int64(1757333323)

	createInternalTransferScenario(t, db, senderAddr, receiverAddr, internalTxHash, baseTimestamp, 0.001, TestAsset)

	insertTestAlchemyTransfer(t, db, chainID, senderAddr, externalTxHash1, baseTimestamp-1000, 0.002, TestAsset,
		externalAddr, senderAddr)

	insertTestAlchemyTransfer(t, db, chainID, receiverAddr, externalTxHash2, baseTimestamp-2000, 0.003, TestAsset,
		externalAddr2, receiverAddr)

	addresses := []eth.Address{senderAddr, receiverAddr}
	chainIDs := []walletCommon.ChainID{walletCommon.ChainID(chainID)}

	results, totalCount, err := getTransactionsOrder(context.Background(), db, addresses, chainIDs, Filter{}, DefaultOffset, DefaultLimit)

	require.NoError(t, err)
	require.Equal(t, int64(4), totalCount, "Should have exactly 4 transactions (2 perspectives of internal + 2 external)")
	require.Len(t, results, 4)

	foundInternalSender := false
	foundInternalReceiver := false
	foundExternal1 := false
	foundExternal2 := false

	for _, tx := range results {
		if tx.Hash == internalTxHash && tx.Address == senderAddr {
			require.Equal(t, SentTransaction, tx.Source, "Internal transfer should be 'sent' for sender")
			foundInternalSender = true
		}
		if tx.Hash == internalTxHash && tx.Address == receiverAddr {
			require.Equal(t, FetchedTransaction, tx.Source, "Internal transfer should be 'fetched' for receiver")
			foundInternalReceiver = true
		}
		if tx.Hash == externalTxHash1 {
			require.Equal(t, FetchedTransaction, tx.Source, "External transfer should be 'fetched'")
			require.Equal(t, senderAddr, tx.Address, "External transfer should be associated with sender")
			foundExternal1 = true
		}
		if tx.Hash == externalTxHash2 {
			require.Equal(t, FetchedTransaction, tx.Source, "External transfer should be 'fetched'")
			require.Equal(t, receiverAddr, tx.Address, "External transfer should be associated with receiver")
			foundExternal2 = true
		}
	}

	require.True(t, foundInternalSender, "Should find internal transfer from sender perspective")
	require.True(t, foundInternalReceiver, "Should find internal transfer from receiver perspective")
	require.True(t, foundExternal1, "Should find first external transfer")
	require.True(t, foundExternal2, "Should find second external transfer")

	require.Equal(t, internalTxHash, results[0].Hash, "Internal transfer (newest) should be first")
	require.Equal(t, internalTxHash, results[1].Hash, "Internal transfer (same timestamp) should also be second")
	require.Equal(t, externalTxHash1, results[2].Hash, "First external transfer should be third")
	require.Equal(t, externalTxHash2, results[3].Hash, "Second external transfer should be fourth")
}

// TestGetTransactionsOrder_PendingPriority tests that pending transactions remain visible
// even when newer fetched transactions exist.
//
// Test Scenario:
// - User sends pending transaction at timestamp T1 (older) - exists only in "sent", NOT in "fetched"
// - Other transactions get fetched from blockchain with timestamps T2, T3 (newer)
// - Query for sender address
//
// Expected Behavior:
// - Pending transaction should appear as "sent" source (exists only in route data)
// - Pending transaction should use its original timestamp T1
// - All transactions should be ordered chronologically (newest first)
// - Pending transaction should remain visible despite being older than fetched transactions
func TestGetTransactionsOrder_PendingPriority(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	chainID := TestChainID
	senderAddr := TestSenderAddr
	receiverAddr := TestReceiverAddr
	externalAddr := TestExternalAddr

	pendingTxHash := eth.HexToHash("0xdddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd")
	otherTxHash1 := eth.HexToHash("0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
	otherTxHash2 := eth.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

	baseTimestamp := int64(1757333323)
	pendingTimestamp := baseTimestamp - 2000 // T1 (oldest - pending)
	otherTimestamp1 := baseTimestamp - 1000  // T2 (newer - fetched)
	otherTimestamp2 := baseTimestamp         // T3 (newest - fetched)

	weiValue := new(big.Int).Mul(big.NewInt(1), big.NewInt(gethParams.Ether/1000)) // 0.001 ETH
	uuid := "pending-test-uuid"

	insertTestPendingTransaction(t, db, chainID, pendingTxHash, senderAddr, receiverAddr, weiValue, TestAsset, pendingTimestamp, uuid)

	insertTestAlchemyTransfer(t, db, chainID, senderAddr, otherTxHash1, otherTimestamp1, 0.002, TestAsset,
		externalAddr, senderAddr)

	insertTestAlchemyTransfer(t, db, chainID, senderAddr, otherTxHash2, otherTimestamp2, 0.003, TestAsset,
		externalAddr, senderAddr)

	addresses := []eth.Address{senderAddr}
	chainIDs := []walletCommon.ChainID{walletCommon.ChainID(chainID)}

	results, totalCount, err := getTransactionsOrder(context.Background(), db, addresses, chainIDs, Filter{}, DefaultOffset, DefaultLimit)

	require.NoError(t, err)
	require.Equal(t, int64(3), totalCount, "Should have 3 transactions (1 pending + 2 fetched)")
	require.Len(t, results, 3)

	foundPending := false
	foundOther1 := false
	foundOther2 := false

	for _, tx := range results {
		if tx.Hash == pendingTxHash {
			require.Equal(t, SentTransaction, tx.Source, "Pending transaction should be 'sent' (exists only in route data)")
			require.Equal(t, senderAddr, tx.Address, "Address should be sender")
			require.Equal(t, pendingTimestamp, tx.Timestamp, "Should use original pending timestamp")
			foundPending = true
		}
		if tx.Hash == otherTxHash1 {
			require.Equal(t, FetchedTransaction, tx.Source, "First other transaction should be 'fetched'")
			require.Equal(t, senderAddr, tx.Address, "Address should be sender")
			foundOther1 = true
		}
		if tx.Hash == otherTxHash2 {
			require.Equal(t, FetchedTransaction, tx.Source, "Second other transaction should be 'fetched'")
			require.Equal(t, senderAddr, tx.Address, "Address should be sender")
			foundOther2 = true
		}
	}

	require.True(t, foundPending, "Should find pending transaction")
	require.True(t, foundOther1, "Should find first other transaction")
	require.True(t, foundOther2, "Should find second other transaction")

	require.Equal(t, otherTxHash2, results[0].Hash, "Newest fetched transaction should be first")
	require.Equal(t, otherTxHash1, results[1].Hash, "Second newest fetched transaction should be second")
	require.Equal(t, pendingTxHash, results[2].Hash, "Pending transaction should be third (oldest timestamp)")
}

// TestGetTransactionsOrder_MixedStatusOrdering tests transaction ordering with mixed
// transaction statuses and timestamps.
//
// Test Scenario:
// - Pending transaction (oldest timestamp)
// - Confirmed transaction (middle timestamp)
// - Fetched transaction (newest timestamp)
// - All for the same address
//
// Expected Behavior:
// - All transactions should appear in timestamp order (newest first)
// - Transaction sources should be preserved ("sent" for pending/confirmed, "fetched" for external)
// - No transactions should be dropped or duplicated
// - Status should not affect ordering (only timestamps matter)
func TestGetTransactionsOrder_MixedStatusOrdering(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	chainID := TestChainID
	senderAddr := TestSenderAddr
	receiverAddr := TestReceiverAddr
	externalAddr := TestExternalAddr

	pendingTxHash := eth.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	confirmedTxHash := eth.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")
	fetchedTxHash := eth.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333")

	baseTimestamp := int64(1757333323)
	pendingTimestamp := baseTimestamp - 2000   // Oldest
	confirmedTimestamp := baseTimestamp - 1000 // Middle
	fetchedTimestamp := baseTimestamp          // Newest

	weiValue := new(big.Int).Mul(big.NewInt(1), big.NewInt(gethParams.Ether/1000)) // 0.001 ETH

	insertTestPendingTransaction(t, db, chainID, pendingTxHash, senderAddr, receiverAddr, weiValue, TestAsset, pendingTimestamp, "pending-uuid")

	insertTestSavedTransaction(t, db, chainID, confirmedTxHash, senderAddr, receiverAddr, weiValue, TestAsset, confirmedTimestamp, "confirmed-uuid")

	insertTestAlchemyTransfer(t, db, chainID, senderAddr, fetchedTxHash, fetchedTimestamp, 0.001, TestAsset,
		externalAddr, senderAddr)

	addresses := []eth.Address{senderAddr}
	chainIDs := []walletCommon.ChainID{walletCommon.ChainID(chainID)}

	results, totalCount, err := getTransactionsOrder(context.Background(), db, addresses, chainIDs, Filter{}, DefaultOffset, DefaultLimit)

	require.NoError(t, err)
	require.Equal(t, int64(3), totalCount, "Should have 3 transactions")
	require.Len(t, results, 3)

	require.Equal(t, fetchedTxHash, results[0].Hash, "Fetched transaction (newest) should be first")
	require.Equal(t, FetchedTransaction, results[0].Source, "Should be fetched")
	require.Equal(t, fetchedTimestamp, results[0].Timestamp, "Should have newest timestamp")

	require.Equal(t, confirmedTxHash, results[1].Hash, "Confirmed transaction (middle) should be second")
	require.Equal(t, SentTransaction, results[1].Source, "Should be sent")
	require.Equal(t, confirmedTimestamp, results[1].Timestamp, "Should have middle timestamp")

	require.Equal(t, pendingTxHash, results[2].Hash, "Pending transaction (oldest) should be third")
	require.Equal(t, SentTransaction, results[2].Source, "Should be sent")
	require.Equal(t, pendingTimestamp, results[2].Timestamp, "Should have oldest timestamp")
}

// TestGetTransactionsOrder_EmptyResults tests behavior when no transactions exist
// for the queried addresses.
//
// Test Scenario:
// - Query addresses that have no transaction history
// - No data in either "sent" or "fetched" sources
//
// Expected Behavior:
// - Should return empty results array
// - Total count should be 0
// - Should not return error
func TestGetTransactionsOrder_EmptyResults(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	chainID := TestChainID
	addresses := []eth.Address{TestSenderAddr, TestReceiverAddr}
	chainIDs := []walletCommon.ChainID{walletCommon.ChainID(chainID)}

	results, totalCount, err := getTransactionsOrder(context.Background(), db, addresses, chainIDs, Filter{}, DefaultOffset, DefaultLimit)

	require.NoError(t, err)
	require.Equal(t, int64(0), totalCount, "Should have 0 transactions")
	require.Len(t, results, 0, "Results array should be empty")
}

// TestGetTransactionsOrder_EmptyAddressList tests behavior when empty address
// list is provided.
//
// Test Scenario:
// - Pass empty address array to function
// - Database contains transaction data
//
// Expected Behavior:
// - Should return error for empty address list (validation)
// - This is expected behavior - function requires at least one address
func TestGetTransactionsOrder_EmptyAddressList(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	chainID := TestChainID
	senderAddr := TestSenderAddr
	receiverAddr := TestReceiverAddr
	txHash := eth.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	timestamp := int64(1757333323)

	createInternalTransferScenario(t, db, senderAddr, receiverAddr, txHash, timestamp, 0.001, TestAsset)

	addresses := []eth.Address{} // Empty address list
	chainIDs := []walletCommon.ChainID{walletCommon.ChainID(chainID)}

	results, totalCount, err := getTransactionsOrder(context.Background(), db, addresses, chainIDs, Filter{}, DefaultOffset, DefaultLimit)

	require.Error(t, err, "Should return error for empty address list")
	require.Contains(t, err.Error(), "no addresses provided", "Error should mention no addresses provided")
	require.Equal(t, int64(0), totalCount, "Total count should be 0 when error occurs")
	require.Len(t, results, 0, "Results array should be empty when error occurs")
}

// TestGetTransactionsOrder_Pagination tests pagination functionality with deduplication.
//
// Test Scenario:
// - Create 5 transactions (2 internal transfers + 3 external) = 7 total result rows
// - Test various offset/limit combinations
// - Verify total count remains consistent across pagination
//
// Expected Behavior:
// - Total count should always be the same regardless of pagination
// - Results should be properly sliced based on offset/limit
// - Deduplication should work correctly across page boundaries
// - Ordering should be maintained across pages
func TestGetTransactionsOrder_Pagination(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	chainID := TestChainID
	senderAddr := TestSenderAddr
	receiverAddr := TestReceiverAddr
	externalAddr := TestExternalAddr

	baseTimestamp := int64(1757333323)

	internalTx1Hash := eth.HexToHash("0x1000000000000000000000000000000000000000000000000000000000000000")
	createInternalTransferScenario(t, db, senderAddr, receiverAddr, internalTx1Hash, baseTimestamp, 0.001, TestAsset)

	internalTx2Hash := eth.HexToHash("0x2000000000000000000000000000000000000000000000000000000000000000")
	createInternalTransferScenario(t, db, receiverAddr, senderAddr, internalTx2Hash, baseTimestamp-1000, 0.002, TestAsset)

	externalTx1Hash := eth.HexToHash("0x3000000000000000000000000000000000000000000000000000000000000000")
	insertTestAlchemyTransfer(t, db, chainID, senderAddr, externalTx1Hash, baseTimestamp-2000, 0.003, TestAsset,
		externalAddr, senderAddr)

	externalTx2Hash := eth.HexToHash("0x4000000000000000000000000000000000000000000000000000000000000000")
	insertTestAlchemyTransfer(t, db, chainID, receiverAddr, externalTx2Hash, baseTimestamp-3000, 0.004, TestAsset,
		externalAddr, receiverAddr)

	externalTx3Hash := eth.HexToHash("0x5000000000000000000000000000000000000000000000000000000000000000")
	insertTestAlchemyTransfer(t, db, chainID, senderAddr, externalTx3Hash, baseTimestamp-4000, 0.005, TestAsset,
		externalAddr, senderAddr)

	addresses := []eth.Address{senderAddr, receiverAddr}
	chainIDs := []walletCommon.ChainID{walletCommon.ChainID(chainID)}

	allResults, totalCount, err := getTransactionsOrder(context.Background(), db, addresses, chainIDs, Filter{}, 0, 10)
	require.NoError(t, err)
	expectedTotal := totalCount

	firstPageResults, firstPageCount, err := getTransactionsOrder(context.Background(), db, addresses, chainIDs, Filter{}, 0, 3)
	require.NoError(t, err)
	require.Equal(t, expectedTotal, firstPageCount, "Total count should be consistent")
	require.Len(t, firstPageResults, 3, "Should return 3 results for first page")

	secondPageResults, secondPageCount, err := getTransactionsOrder(context.Background(), db, addresses, chainIDs, Filter{}, 3, 3)
	require.NoError(t, err)
	require.Equal(t, expectedTotal, secondPageCount, "Total count should be consistent")

	thirdPageResults, thirdPageCount, err := getTransactionsOrder(context.Background(), db, addresses, chainIDs, Filter{}, 6, 3)
	require.NoError(t, err)
	require.Equal(t, expectedTotal, thirdPageCount, "Total count should be consistent")

	combinedResults := append(firstPageResults, secondPageResults...)
	combinedResults = append(combinedResults, thirdPageResults...)

	require.Equal(t, len(allResults), len(combinedResults), "Paginated results should equal full results")

	for i := 0; i < len(allResults) && i < len(combinedResults); i++ {
		require.Equal(t, allResults[i].Hash, combinedResults[i].Hash, "Transaction order should be maintained across pages")
		require.Equal(t, allResults[i].Timestamp, combinedResults[i].Timestamp, "Timestamps should match")
	}
}

// TestGetTransactionsOrder_PaginationEdgeCases tests pagination boundary conditions.
//
// Test Scenario:
// - Test offset greater than total results
// - Test NoLimit constant (should return all results)
// - Test very large limit
// - Test negative values (if function handles them)
//
// Expected Behavior:
// - Offset > total should return empty results (total count behavior may vary)
// - NoLimit should return all results (treated as "no limit")
// - Large limit should return all available results
// - Function should handle edge cases gracefully
func TestGetTransactionsOrder_PaginationEdgeCases(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	chainID := TestChainID
	senderAddr := TestSenderAddr
	receiverAddr := TestReceiverAddr
	txHash := eth.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	timestamp := int64(1757333323)

	createInternalTransferScenario(t, db, senderAddr, receiverAddr, txHash, timestamp, 0.001, TestAsset)

	addresses := []eth.Address{senderAddr, receiverAddr}
	chainIDs := []walletCommon.ChainID{walletCommon.ChainID(chainID)}

	normalResults, expectedTotal, err := getTransactionsOrder(context.Background(), db, addresses, chainIDs, Filter{}, 0, 10)
	require.NoError(t, err)
	require.Greater(t, expectedTotal, int64(0), "Should have some transactions for edge case testing")

	offsetTooLarge, _, err := getTransactionsOrder(context.Background(), db, addresses, chainIDs, Filter{}, 1000, 10)
	require.NoError(t, err)
	require.Len(t, offsetTooLarge, 0, "Should return empty results when offset > total")

	limitZero, totalCount, err := getTransactionsOrder(context.Background(), db, addresses, chainIDs, Filter{}, 0, ac.NoLimit)
	require.NoError(t, err)
	require.Equal(t, expectedTotal, totalCount, "Total count should be consistent even with NoLimit")
	require.Len(t, limitZero, int(expectedTotal), "NoLimit should return all results (treated as no limit)")

	largeLimitResults, totalCount, err := getTransactionsOrder(context.Background(), db, addresses, chainIDs, Filter{}, 0, 1000)
	require.NoError(t, err)
	require.Equal(t, expectedTotal, totalCount, "Total count should be consistent with large limit")
	require.Len(t, largeLimitResults, len(normalResults), "Large limit should return all available results")
}

// TestGetTransactionsOrder_MultipleChains tests transaction filtering across different chain IDs.
//
// Test Scenario:
// - Same address has transactions on multiple chains (Ethereum mainnet, Polygon)
// - Query specific chains vs all chains
// - Verify chain ID filtering works correctly
//
// Expected Behavior:
// - Transactions should be properly filtered by chain ID
// - Same address on different chains should be treated independently
// - Chain ID filtering should work correctly
// - Results should include correct chain ID for each transaction
func TestGetTransactionsOrder_MultipleChains(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ethereumChainID := uint64(1)  // Ethereum mainnet
	polygonChainID := uint64(137) // Polygon mainnet
	senderAddr := TestSenderAddr

	ethTxHash := eth.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	polygonTxHash := eth.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")

	baseTimestamp := int64(1757333323)

	insertTestAlchemyTransfer(t, db, ethereumChainID, senderAddr, ethTxHash, baseTimestamp, 0.001, TestAsset,
		TestExternalAddr, senderAddr)
	insertTestAlchemyTransfer(t, db, polygonChainID, senderAddr, polygonTxHash, baseTimestamp-1000, 0.002, "MATIC",
		TestExternalAddr, senderAddr)

	addresses := []eth.Address{senderAddr}

	ethOnlyResults, ethCount, err := getTransactionsOrder(context.Background(), db, addresses, []walletCommon.ChainID{walletCommon.ChainID(ethereumChainID)}, Filter{}, DefaultOffset, DefaultLimit)
	require.NoError(t, err)
	require.Equal(t, int64(1), ethCount, "Should have 1 transaction on Ethereum")
	require.Len(t, ethOnlyResults, 1)
	require.Equal(t, ethTxHash, ethOnlyResults[0].Hash, "Should be Ethereum transaction")
	require.Equal(t, walletCommon.ChainID(ethereumChainID), ethOnlyResults[0].ChainID, "Should have correct chain ID")

	polygonOnlyResults, polygonCount, err := getTransactionsOrder(context.Background(), db, addresses, []walletCommon.ChainID{walletCommon.ChainID(polygonChainID)}, Filter{}, DefaultOffset, DefaultLimit)
	require.NoError(t, err)
	require.Equal(t, int64(1), polygonCount, "Should have 1 transaction on Polygon")
	require.Len(t, polygonOnlyResults, 1)
	require.Equal(t, polygonTxHash, polygonOnlyResults[0].Hash, "Should be Polygon transaction")
	require.Equal(t, walletCommon.ChainID(polygonChainID), polygonOnlyResults[0].ChainID, "Should have correct chain ID")

	allChainsResults, allCount, err := getTransactionsOrder(context.Background(), db, addresses, []walletCommon.ChainID{walletCommon.ChainID(ethereumChainID), walletCommon.ChainID(polygonChainID)}, Filter{}, DefaultOffset, DefaultLimit)
	require.NoError(t, err)
	require.Equal(t, int64(2), allCount, "Should have 2 transactions across all chains")
	require.Len(t, allChainsResults, 2)

	require.Equal(t, ethTxHash, allChainsResults[0].Hash, "Ethereum transaction should be first (newer timestamp)")
	require.Equal(t, polygonTxHash, allChainsResults[1].Hash, "Polygon transaction should be second")
}

// TestGetTransactionsOrder_SameHashDifferentChains tests handling of identical
// transaction hashes on different chains.
//
// Test Scenario:
// - Same transaction hash exists on multiple chains (possible in real scenarios)
// - Query both chains to ensure proper separation
//
// Expected Behavior:
// - Same hash on different chains should be treated as separate transactions
// - Each transaction should have correct chain ID
// - Both transactions should appear in results when querying multiple chains
// - Chain ID filtering should work correctly
func TestGetTransactionsOrder_SameHashDifferentChains(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ethereumChainID := uint64(1)
	polygonChainID := uint64(137)
	senderAddr := TestSenderAddr

	sameHash := eth.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	baseTimestamp := int64(1757333323)

	insertTestAlchemyTransfer(t, db, ethereumChainID, senderAddr, sameHash, baseTimestamp, 0.001, TestAsset,
		TestExternalAddr, senderAddr)
	insertTestAlchemyTransfer(t, db, polygonChainID, senderAddr, sameHash, baseTimestamp-1000, 0.002, "MATIC",
		TestExternalAddr, senderAddr)

	addresses := []eth.Address{senderAddr}
	chainIDs := []walletCommon.ChainID{walletCommon.ChainID(ethereumChainID), walletCommon.ChainID(polygonChainID)}

	results, totalCount, err := getTransactionsOrder(context.Background(), db, addresses, chainIDs, Filter{}, DefaultOffset, DefaultLimit)

	require.NoError(t, err)
	require.Equal(t, int64(2), totalCount, "Should have 2 transactions with same hash on different chains")
	require.Len(t, results, 2)

	foundEthereumTx := false
	foundPolygonTx := false
	for _, tx := range results {
		require.Equal(t, sameHash, tx.Hash, "Both transactions should have same hash")
		if tx.ChainID == walletCommon.ChainID(ethereumChainID) {
			foundEthereumTx = true
		}
		if tx.ChainID == walletCommon.ChainID(polygonChainID) {
			foundPolygonTx = true
		}
	}

	require.True(t, foundEthereumTx, "Should find Ethereum transaction")
	require.True(t, foundPolygonTx, "Should find Polygon transaction")
}

// TestGetTransactionsOrder_EmptyChainList tests validation when no chain IDs are provided.
//
// Test Scenario:
// - Pass empty chain ID array to function
// - Database contains transaction data
//
// Expected Behavior:
// - Should return error for empty chain ID list (validation)
// - This is expected behavior - function requires at least one chain ID
func TestGetTransactionsOrder_EmptyChainList(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	senderAddr := TestSenderAddr
	receiverAddr := TestReceiverAddr
	txHash := eth.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	timestamp := int64(1757333323)

	createInternalTransferScenario(t, db, senderAddr, receiverAddr, txHash, timestamp, 0.001, TestAsset)

	addresses := []eth.Address{senderAddr}
	chainIDs := []walletCommon.ChainID{} // Empty chain ID list

	results, totalCount, err := getTransactionsOrder(context.Background(), db, addresses, chainIDs, Filter{}, DefaultOffset, DefaultLimit)

	require.Error(t, err, "Should return error for empty chain ID list")
	require.Contains(t, err.Error(), "no chainIDs provided", "Error should mention no chainIDs provided")
	require.Equal(t, int64(0), totalCount, "Total count should be 0 when error occurs")
	require.Len(t, results, 0, "Results array should be empty when error occurs")
}
