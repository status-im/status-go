package transfer

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/chain/ethclient"
	mock_rpcclient "github.com/status-im/status-go/rpc/mock/client"
	wallet_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
	mock_pathprocessor "github.com/status-im/status-go/services/wallet/router/pathprocessor/mock"
	"github.com/status-im/status-go/services/wallet/walletevent"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/transactions"
	mock_transactor "github.com/status-im/status-go/transactions/mock"
	"github.com/status-im/status-go/walletdatabase"
)

func deepCopy(tx *wallettypes.SendTxArgs) *wallettypes.SendTxArgs {
	return &wallettypes.SendTxArgs{
		From:  tx.From,
		To:    tx.To,
		Value: tx.Value,
		Data:  tx.Data,
	}
}

func deepCopyTransactionBridgeWithTransferTx(tx *pathprocessor.MultipathProcessorTxArgs) *pathprocessor.MultipathProcessorTxArgs {
	return &pathprocessor.MultipathProcessorTxArgs{
		Name:              tx.Name,
		ChainID:           tx.ChainID,
		TransferTx:        deepCopy(tx.TransferTx),
		HopTx:             tx.HopTx,
		CbridgeTx:         tx.CbridgeTx,
		ERC721TransferTx:  tx.ERC721TransferTx,
		ERC1155TransferTx: tx.ERC1155TransferTx,
		SwapTx:            tx.SwapTx,
	}
}

func setupTransactionManager(t *testing.T) (*TransactionManager, *mock_transactor.MockTransactorIface, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock transactor
	transactor := mock_transactor.NewMockTransactorIface(ctrl)
	// Create a new instance of the TransactionManager
	tm := NewTransactionManager(NewInMemMultiTransactionStorage(), nil, transactor, nil, nil, nil, nil)

	return tm, transactor, ctrl
}

func setupAccount(_ *testing.T, address common.Address) *account.SelectedExtKey {
	// Dummy account
	return &account.SelectedExtKey{
		Address:    types.Address(address),
		AccountKey: &types.Key{},
	}
}

func setupTransactionData(_ *testing.T, transactor transactions.TransactorIface) (*MultiTransaction, []*pathprocessor.MultipathProcessorTxArgs, map[string]pathprocessor.PathProcessor, []*pathprocessor.MultipathProcessorTxArgs) {
	SetMultiTransactionIDGenerator(StaticIDCounter())

	// Create mock data for the test
	ethTransfer := generateTestTransfer(0)
	multiTransaction := GenerateTestSendMultiTransaction(ethTransfer)

	// Initialize the bridges
	var rpcClient *rpc.Client = nil
	bridges := make(map[string]pathprocessor.PathProcessor)
	transferBridge := pathprocessor.NewTransferProcessor(rpcClient, transactor)
	bridges[transferBridge.Name()] = transferBridge

	data := []*pathprocessor.MultipathProcessorTxArgs{
		{
			ChainID: 1,
			Name:    transferBridge.Name(),
			TransferTx: &wallettypes.SendTxArgs{
				From:  types.Address(ethTransfer.From),
				To:    (*types.Address)(&ethTransfer.To),
				Value: (*hexutil.Big)(big.NewInt(ethTransfer.Value / 3)),
				Data:  types.HexBytes("0x0"),
				// Symbol: multiTransaction.FromAsset, // This will be set by transaction manager
				// MultiTransactionID: multiTransaction.ID, // This will be set by transaction manager
			},
		},
		{
			ChainID: 420,
			Name:    transferBridge.Name(),
			TransferTx: &wallettypes.SendTxArgs{
				From:  types.Address(ethTransfer.From),
				To:    (*types.Address)(&ethTransfer.To),
				Value: (*hexutil.Big)(big.NewInt(ethTransfer.Value * 2 / 3)),
				Data:  types.HexBytes("0x0"),
				// Symbol: multiTransaction.FromAsset, // This will be set by transaction manager
				// MultiTransactionID: multiTransaction.ID, // This will be set by transaction manager
			},
		},
	}

	expectedData := make([]*pathprocessor.MultipathProcessorTxArgs, 0)
	for _, tx := range data {
		txCopy := deepCopyTransactionBridgeWithTransferTx(tx)
		updateDataFromMultiTx([]*pathprocessor.MultipathProcessorTxArgs{txCopy}, &multiTransaction)
		expectedData = append(expectedData, txCopy)
	}

	return &multiTransaction, data, bridges, expectedData
}

func setupApproveTransactionData(_ *testing.T, transactor transactions.TransactorIface) (*MultiTransaction, []*pathprocessor.MultipathProcessorTxArgs, map[string]pathprocessor.PathProcessor, []*pathprocessor.MultipathProcessorTxArgs) {
	SetMultiTransactionIDGenerator(StaticIDCounter())

	// Create mock data for the test
	tokenTransfer := generateTestTransfer(4)
	multiTransaction := GenerateTestApproveMultiTransaction(tokenTransfer)

	// Initialize the bridges
	var rpcClient *rpc.Client = nil
	bridges := make(map[string]pathprocessor.PathProcessor)
	transferBridge := pathprocessor.NewTransferProcessor(rpcClient, transactor)
	bridges[transferBridge.Name()] = transferBridge

	data := []*pathprocessor.MultipathProcessorTxArgs{
		{
			//ChainID: 1, // This will be set by transaction manager
			Name: transferBridge.Name(),
			TransferTx: &wallettypes.SendTxArgs{
				From:  types.Address(tokenTransfer.From),
				To:    (*types.Address)(&tokenTransfer.To),
				Value: (*hexutil.Big)(big.NewInt(tokenTransfer.Value)),
				Data:  types.HexBytes("0x0"),
				// Symbol: multiTransaction.FromAsset, // This will be set by transaction manager
				// MultiTransactionID: multiTransaction.ID, // This will be set by transaction manager
			},
		},
	}

	expectedData := make([]*pathprocessor.MultipathProcessorTxArgs, 0)
	for _, tx := range data {
		txCopy := deepCopyTransactionBridgeWithTransferTx(tx)
		updateDataFromMultiTx([]*pathprocessor.MultipathProcessorTxArgs{txCopy}, &multiTransaction)
		expectedData = append(expectedData, txCopy)
	}

	return &multiTransaction, data, bridges, expectedData
}

func TestSendTransactionsETHSuccess(t *testing.T) {
	tm, transactor, _ := setupTransactionManager(t)
	account := setupAccount(t, common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"))
	multiTransaction, data, bridges, expectedData := setupTransactionData(t, transactor)

	// Verify that the SendTransactionWithChainID method is called for each transaction with proper arguments
	// Return values are not checked, because they must be checked in Transactor tests
	for _, tx := range expectedData {
		transactor.EXPECT().SendTransactionWithChainID(tx.ChainID, *(tx.TransferTx), int64(-1), account).Return(types.Hash{}, uint64(0), nil)
	}

	// Call the SendTransactions method
	_, err := tm.SendTransactions(context.Background(), multiTransaction, data, bridges, account)
	require.NoError(t, err)
}

func TestSendTransactionsApproveSuccess(t *testing.T) {
	tm, transactor, _ := setupTransactionManager(t)
	account := setupAccount(t, common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"))
	multiTransaction, data, bridges, expectedData := setupApproveTransactionData(t, transactor)

	// Verify that the SendTransactionWithChainID method is called for each transaction with proper arguments
	// Return values are not checked, because they must be checked in Transactor tests
	for _, tx := range expectedData {
		transactor.EXPECT().SendTransactionWithChainID(tx.ChainID, *(tx.TransferTx), int64(-1), account).Return(types.Hash{}, uint64(0), nil)
	}

	// Call the SendTransactions method
	_, err := tm.SendTransactions(context.Background(), multiTransaction, data, bridges, account)
	require.NoError(t, err)
}

func TestSendTransactionsETHFailOnBridge(t *testing.T) {
	tm, transactor, ctrl := setupTransactionManager(t)
	account := setupAccount(t, common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"))
	multiTransaction, data, _, _ := setupTransactionData(t, transactor)

	// Initialize the bridges
	bridges := make(map[string]pathprocessor.PathProcessor)
	transferBridge := mock_pathprocessor.NewMockPathProcessor(ctrl)

	// Set bridge name for the mock to the one used in data
	transferBridge.EXPECT().Name().Return(data[0].Name).AnyTimes()
	bridges[transferBridge.Name()] = transferBridge

	expectedErr := wallettypes.ErrInvalidTxSender // Any error to verify
	// In case of bridge error, verify that the error is returned
	transferBridge.EXPECT().Send(gomock.Any(), int64(-1), gomock.Any()).Return(types.Hash{}, uint64(0), wallettypes.ErrInvalidTxSender)

	// Call the SendTransactions method
	_, err := tm.SendTransactions(context.Background(), multiTransaction, data, bridges, account)
	require.ErrorIs(t, expectedErr, err)
}

func TestSendTransactionsETHFailOnTransactor(t *testing.T) {
	tm, transactor, _ := setupTransactionManager(t)
	account := setupAccount(t, common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"))
	multiTransaction, data, bridges, expectedData := setupTransactionData(t, transactor)

	// Verify that the SendTransactionWithChainID method is called for each transaction with proper arguments
	// Return values are not checked, because they must be checked in Transactor tests. Only error propagation matters here
	expectedErr := wallettypes.ErrInvalidTxSender // Any error to verify
	transactor.EXPECT().SendTransactionWithChainID(expectedData[0].ChainID, *(expectedData[0].TransferTx), int64(-1), account).Return(types.Hash{}, uint64(0), nil)
	transactor.EXPECT().SendTransactionWithChainID(expectedData[1].ChainID, *(expectedData[1].TransferTx), int64(-1), account).Return(types.Hash{}, uint64(0), expectedErr)

	// Call the SendTransactions method
	_, err := tm.SendTransactions(context.Background(), multiTransaction, data, bridges, account)
	require.ErrorIs(t, expectedErr, err)
}

func TestWatchTransaction(t *testing.T) {
	tm, _, _ := setupTransactionManager(t)
	chainID := uint64(777) // GeneratePendingTransaction uses this chainID
	pendingTxTimeout = 2 * time.Millisecond

	walletDB, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	chainClient := transactions.NewMockChainClient()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	rpcClient := mock_rpcclient.NewMockClientInterface(ctrl)
	rpcClient.EXPECT().AbstractEthClient(wallet_common.ChainID(chainID)).DoAndReturn(func(chainID wallet_common.ChainID) (ethclient.BatchCallClient, error) {
		return chainClient.AbstractEthClient(chainID)
	}).AnyTimes()
	eventFeed := &event.Feed{}
	// For now, pending tracker is not interface, so we have to use a real one
	tm.pendingTracker = transactions.NewPendingTxTracker(walletDB, rpcClient, nil, eventFeed, pendingTxTimeout)
	tm.eventFeed = eventFeed

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*pendingTxTimeout)
	defer cancel()

	// Insert a pending transaction
	txs := transactions.MockTestTransactions(t, chainClient, []transactions.TestTxSummary{{}})
	err = tm.pendingTracker.StoreAndTrackPendingTx(&txs[0]) // We dont need to track it, but no other way to insert it
	require.NoError(t, err)

	txEventPayload := transactions.StatusChangedPayload{
		TxIdentity: transactions.TxIdentity{
			Hash:    txs[0].Hash,
			ChainID: wallet_common.ChainID(chainID),
		},
		Status: transactions.Pending,
	}
	jsonPayload, err := json.Marshal(txEventPayload)
	require.NoError(t, err)

	go func() {
		time.Sleep(pendingTxTimeout / 2)
		eventFeed.Send(walletevent.Event{
			Type:    transactions.EventPendingTransactionStatusChanged,
			Message: string(jsonPayload),
		})
	}()

	// Call the WatchTransaction method
	err = tm.WatchTransaction(ctx, chainID, txs[0].Hash)
	require.NoError(t, err)
}

func TestWatchTransaction_Timeout(t *testing.T) {
	tm, _, _ := setupTransactionManager(t)
	chainID := uint64(777) // GeneratePendingTransaction uses this chainID
	transactionHash := common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	pendingTxTimeout = 2 * time.Millisecond

	walletDB, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	chainClient := transactions.NewMockChainClient()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	rpcClient := mock_rpcclient.NewMockClientInterface(gomock.NewController(t))
	rpcClient.EXPECT().AbstractEthClient(wallet_common.ChainID(chainID)).DoAndReturn(func(chainID wallet_common.ChainID) (ethclient.BatchCallClient, error) {
		return chainClient.AbstractEthClient(chainID)
	}).AnyTimes()
	eventFeed := &event.Feed{}
	// For now, pending tracker is not interface, so we have to use a real one
	tm.pendingTracker = transactions.NewPendingTxTracker(walletDB, rpcClient, nil, eventFeed, pendingTxTimeout)
	tm.eventFeed = eventFeed

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Microsecond)
	defer cancel()

	// Insert a pending transaction
	txs := transactions.MockTestTransactions(t, chainClient, []transactions.TestTxSummary{{}})
	err = tm.pendingTracker.StoreAndTrackPendingTx(&txs[0]) // We dont need to track it, but no other way to insert it
	require.NoError(t, err)

	// Call the WatchTransaction method
	err = tm.WatchTransaction(ctx, chainID, transactionHash)
	require.ErrorIs(t, err, ErrWatchPendingTxTimeout)
}

func TestCreateMultiTransactionFromCommand(t *testing.T) {
	tm, _, _ := setupTransactionManager(t)

	var command *MultiTransactionCommand

	// Test types that should get chainID from the data
	mtTypes := []MultiTransactionType{MultiTransactionSend, MultiTransactionApprove, MultiTransactionSwap, MultiTransactionBridge, MultiTransactionType(7)}

	for _, mtType := range mtTypes {
		fromAmount := hexutil.Big(*big.NewInt(1000000000000000000))
		toAmount := hexutil.Big(*big.NewInt(123))
		command = &MultiTransactionCommand{
			Type:        mtType,
			FromAddress: common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
			ToAddress:   common.HexToAddress("0xabcdef1234567890abcdef1234567890abcdef12"),
			FromAsset:   "DAI",
			ToAsset:     "USDT",
			FromAmount:  &fromAmount,
			ToAmount:    &toAmount,
		}

		data := make([]*pathprocessor.MultipathProcessorTxArgs, 0)
		data = append(data, &pathprocessor.MultipathProcessorTxArgs{
			ChainID: 1,
		})

		if mtType == MultiTransactionBridge {
			data[0].HopTx = &pathprocessor.HopBridgeTxArgs{
				ChainID:   1,
				ChainIDTo: 2,
			}
		}

		multiTransaction, err := tm.CreateMultiTransactionFromCommand(command, data)
		if mtType > MultiTransactionApprove {
			// Unsupported type
			require.Error(t, err)
			break
		}
		require.NoError(t, err)
		require.NotNil(t, multiTransaction)
		require.Equal(t, command.FromAddress, multiTransaction.FromAddress)
		require.Equal(t, command.ToAddress, multiTransaction.ToAddress)
		require.Equal(t, command.FromAsset, multiTransaction.FromAsset)
		require.Equal(t, command.ToAsset, multiTransaction.ToAsset)
		require.Equal(t, command.FromAmount, multiTransaction.FromAmount)
		require.Equal(t, command.ToAmount, multiTransaction.ToAmount)
		require.Equal(t, command.Type, multiTransaction.Type)
		require.Equal(t, data[0].ChainID, multiTransaction.FromNetworkID)
	}
}
