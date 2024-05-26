package transfer

import (
	"context"
	"math/big"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/wallet/bridge"
	"github.com/status-im/status-go/services/wallet/bridge/mock_bridge"
	"github.com/status-im/status-go/transactions"
	"github.com/status-im/status-go/transactions/mock_transactor"
)

func deepCopy(tx *transactions.SendTxArgs) *transactions.SendTxArgs {
	return &transactions.SendTxArgs{
		From:  tx.From,
		To:    tx.To,
		Value: tx.Value,
		Data:  tx.Data,
	}
}

func deepCopyTransactionBridgeWithTransferTx(tx *bridge.TransactionBridge) *bridge.TransactionBridge {
	return &bridge.TransactionBridge{
		BridgeName:        tx.BridgeName,
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

func setupTransactionData(_ *testing.T, transactor transactions.TransactorIface) (*MultiTransaction, []*bridge.TransactionBridge, map[string]bridge.Bridge, []*bridge.TransactionBridge) {
	SetMultiTransactionIDGenerator(StaticIDCounter())

	// Create mock data for the test
	ethTransfer := generateTestTransfer(0)
	multiTransaction := GenerateTestSendMultiTransaction(ethTransfer)

	// Initialize the bridges
	var rpcClient *rpc.Client = nil
	bridges := make(map[string]bridge.Bridge)
	transferBridge := bridge.NewTransferBridge(rpcClient, transactor)
	bridges[transferBridge.Name()] = transferBridge

	data := []*bridge.TransactionBridge{
		{
			ChainID:    1,
			BridgeName: transferBridge.Name(),
			TransferTx: &transactions.SendTxArgs{
				From:  types.Address(ethTransfer.From),
				To:    (*types.Address)(&ethTransfer.To),
				Value: (*hexutil.Big)(big.NewInt(ethTransfer.Value / 3)),
				Data:  types.HexBytes("0x0"),
				// Symbol: multiTransaction.FromAsset, // This will be set by transaction manager
				// MultiTransactionID: multiTransaction.ID, // This will be set by transaction manager
			},
		},
		{
			ChainID:    420,
			BridgeName: transferBridge.Name(),
			TransferTx: &transactions.SendTxArgs{
				From:  types.Address(ethTransfer.From),
				To:    (*types.Address)(&ethTransfer.To),
				Value: (*hexutil.Big)(big.NewInt(ethTransfer.Value * 2 / 3)),
				Data:  types.HexBytes("0x0"),
				// Symbol: multiTransaction.FromAsset, // This will be set by transaction manager
				// MultiTransactionID: multiTransaction.ID, // This will be set by transaction manager
			},
		},
	}

	expectedData := make([]*bridge.TransactionBridge, 0)
	for _, tx := range data {
		txCopy := deepCopyTransactionBridgeWithTransferTx(tx)
		updateDataFromMultiTx([]*bridge.TransactionBridge{txCopy}, &multiTransaction)
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
		transactor.EXPECT().SendTransactionWithChainID(tx.ChainID, *(tx.TransferTx), account).Return(types.Hash{}, nil)
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
	bridges := make(map[string]bridge.Bridge)
	transferBridge := mock_bridge.NewMockBridge(ctrl)

	// Set bridge name for the mock to the one used in data
	transferBridge.EXPECT().Name().Return(data[0].BridgeName).AnyTimes()
	bridges[transferBridge.Name()] = transferBridge

	expectedErr := transactions.ErrInvalidTxSender // Any error to verify
	// In case of bridge error, verify that the error is returned
	transferBridge.EXPECT().Send(gomock.Any(), gomock.Any()).Return(types.Hash{}, transactions.ErrInvalidTxSender)

	// Call the SendTransactions method
	_, err := tm.SendTransactions(context.Background(), multiTransaction, data, bridges, account)
	require.Error(t, expectedErr, err)
}

func TestSendTransactionsETHFailOnTransactor(t *testing.T) {
	tm, transactor, _ := setupTransactionManager(t)
	account := setupAccount(t, common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"))
	multiTransaction, data, bridges, expectedData := setupTransactionData(t, transactor)

	// Verify that the SendTransactionWithChainID method is called for each transaction with proper arguments
	// Return values are not checked, because they must be checked in Transactor tests. Only error propagation matters here
	expectedErr := transactions.ErrInvalidTxSender // Any error to verify
	transactor.EXPECT().SendTransactionWithChainID(expectedData[0].ChainID, *(expectedData[0].TransferTx), account).Return(types.Hash{}, nil)
	transactor.EXPECT().SendTransactionWithChainID(expectedData[1].ChainID, *(expectedData[1].TransferTx), account).Return(types.Hash{}, expectedErr)

	// Call the SendTransactions method
	_, err := tm.SendTransactions(context.Background(), multiTransaction, data, bridges, account)
	require.Error(t, expectedErr, err)
}
