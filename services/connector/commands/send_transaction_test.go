package commands

import (
	"encoding/json"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/eth-node/types"
	mock_client "github.com/status-im/status-go/rpc/chain/mock/client"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/signal"
)

func prepareSendTransactionRequest(dApp signal.ConnectorDApp, from types.Address) (RPCRequest, error) {
	sendArgs := wallettypes.SendTxArgs{
		From:  from,
		To:    &types.Address{0x02},
		Value: &hexutil.Big{},
		Data:  types.HexBytes("0x0"),
	}

	sendArgsJSON, err := json.Marshal(sendArgs)
	if err != nil {
		return RPCRequest{}, err
	}

	var sendArgsMap map[string]interface{}
	err = json.Unmarshal(sendArgsJSON, &sendArgsMap)
	if err != nil {
		return RPCRequest{}, err
	}

	params := []interface{}{sendArgsMap}

	return ConstructRPCRequest("eth_sendTransaction", params, &dApp)
}

func TestFailToSendTransactionWithoutPermittedDApp(t *testing.T) {
	state, close := setupCommand(t, Method_EthSendTransaction)
	t.Cleanup(close)

	// Don't save dApp in the database
	request, err := prepareSendTransactionRequest(testDAppData, types.Address{0x1})
	assert.NoError(t, err)

	_, err = state.cmd.Execute(state.ctx, request)
	assert.Equal(t, ErrDAppIsNotPermittedByUser, err)
}

func TestFailToSendTransactionWithWrongAddress(t *testing.T) {
	state, close := setupCommand(t, Method_EthSendTransaction)
	t.Cleanup(close)

	err := PersistDAppData(state.walletDb, testDAppData, types.Address{0x01}, uint64(0x1))
	assert.NoError(t, err)

	request, err := prepareSendTransactionRequest(testDAppData, types.Address{0x02})
	assert.NoError(t, err)

	_, err = state.cmd.Execute(state.ctx, request)
	assert.Equal(t, ErrParamsFromAddressIsNotShared, err)
}

func TestSendTransactionWithSignalTimout(t *testing.T) {
	state, close := setupCommand(t, Method_EthSendTransaction)
	t.Cleanup(close)

	accountAddress := types.Address{0x01}
	err := PersistDAppData(state.walletDb, testDAppData, accountAddress, uint64(0x1))
	assert.NoError(t, err)

	request, err := prepareSendTransactionRequest(testDAppData, accountAddress)
	assert.NoError(t, err)

	backupWalletResponseMaxInterval := WalletResponseMaxInterval
	WalletResponseMaxInterval = 1 * time.Millisecond

	mockedChainClient := mock_client.NewMockClientInterface(state.mockCtrl)
	state.rpcClient.EXPECT().EthClient(uint64(1)).Times(1).Return(mockedChainClient, nil)
	mockedChainClient.EXPECT().SuggestGasPrice(state.ctx).Times(1).Return(big.NewInt(1), nil)
	mockedChainClient.EXPECT().SuggestGasTipCap(state.ctx).Times(1).Return(big.NewInt(0), errors.New("EIP-1559 is not enabled"))
	state.rpcClient.EXPECT().EthClient(uint64(1)).Times(1).Return(mockedChainClient, nil)
	mockedChainClient.EXPECT().PendingNonceAt(state.ctx, common.Address(accountAddress)).Times(1).Return(uint64(10), nil)

	_, err = state.cmd.Execute(state.ctx, request)
	assert.Equal(t, ErrWalletResponseTimeout, err)
	WalletResponseMaxInterval = backupWalletResponseMaxInterval
}

func TestSendTransactionWithSignalAccepted(t *testing.T) {
	state, close := setupCommand(t, Method_EthSendTransaction)
	t.Cleanup(close)

	fakedTransactionHash := types.Hash{0x051}

	accountAddress := types.Address{0x01}
	err := PersistDAppData(state.walletDb, testDAppData, accountAddress, uint64(0x1))
	assert.NoError(t, err)

	request, err := prepareSendTransactionRequest(testDAppData, accountAddress)
	assert.NoError(t, err)

	signal.SetMobileSignalHandler(signal.MobileSignalHandler(func(s []byte) {
		var evt EventType
		err := json.Unmarshal(s, &evt)
		assert.NoError(t, err)

		switch evt.Type {
		case signal.EventConnectorSendTransaction:
			var ev signal.ConnectorSendTransactionSignal
			err := json.Unmarshal(evt.Event, &ev)
			assert.NoError(t, err)

			err = state.handler.SendTransactionAccepted(SendTransactionAcceptedArgs{
				Hash:      fakedTransactionHash,
				RequestID: ev.RequestID,
			})
			assert.NoError(t, err)
		}
	}))
	t.Cleanup(signal.ResetMobileSignalHandler)

	mockedChainClient := mock_client.NewMockClientInterface(state.mockCtrl)
	state.rpcClient.EXPECT().EthClient(uint64(1)).Times(1).Return(mockedChainClient, nil)
	mockedChainClient.EXPECT().SuggestGasPrice(state.ctx).Times(1).Return(big.NewInt(1), nil)
	mockedChainClient.EXPECT().SuggestGasTipCap(state.ctx).Times(1).Return(big.NewInt(0), errors.New("EIP-1559 is not enabled"))
	state.rpcClient.EXPECT().EthClient(uint64(1)).Times(1).Return(mockedChainClient, nil)
	mockedChainClient.EXPECT().PendingNonceAt(state.ctx, common.Address(accountAddress)).Times(1).Return(uint64(10), nil)

	response, err := state.cmd.Execute(state.ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, response, fakedTransactionHash.String())
}

func TestSendTransactionWithSignalRejected(t *testing.T) {
	state, close := setupCommand(t, Method_EthSendTransaction)
	t.Cleanup(close)

	accountAddress := types.Address{0x01}
	err := PersistDAppData(state.walletDb, testDAppData, accountAddress, uint64(0x1))
	assert.NoError(t, err)

	request, err := prepareSendTransactionRequest(testDAppData, accountAddress)
	assert.NoError(t, err)

	signal.SetMobileSignalHandler(signal.MobileSignalHandler(func(s []byte) {
		var evt EventType
		err := json.Unmarshal(s, &evt)
		assert.NoError(t, err)

		switch evt.Type {
		case signal.EventConnectorSendTransaction:
			var ev signal.ConnectorSendTransactionSignal
			err := json.Unmarshal(evt.Event, &ev)
			assert.NoError(t, err)

			err = state.handler.SendTransactionRejected(RejectedArgs{
				RequestID: ev.RequestID,
			})
			assert.NoError(t, err)
		}
	}))
	t.Cleanup(signal.ResetMobileSignalHandler)

	mockedChainClient := mock_client.NewMockClientInterface(state.mockCtrl)
	state.rpcClient.EXPECT().EthClient(uint64(1)).Times(1).Return(mockedChainClient, nil)
	mockedChainClient.EXPECT().SuggestGasPrice(state.ctx).Times(1).Return(big.NewInt(1), nil)
	mockedChainClient.EXPECT().SuggestGasTipCap(state.ctx).Times(1).Return(big.NewInt(0), errors.New("EIP-1559 is not enabled"))
	state.rpcClient.EXPECT().EthClient(uint64(1)).Times(1).Return(mockedChainClient, nil)
	mockedChainClient.EXPECT().PendingNonceAt(state.ctx, common.Address(accountAddress)).Times(1).Return(uint64(10), nil)

	_, err = state.cmd.Execute(state.ctx, request)
	assert.Equal(t, ErrSendTransactionRejectedByUser, err)
}
