package commands

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/internal/crypto/types"
	mock_client "github.com/status-im/status-go/internal/rpc/chain/mock/client"
	"github.com/status-im/status-go/internal/signal"
	"github.com/status-im/status-go/pkg/services/wallet/router/fees"
	"github.com/status-im/status-go/pkg/services/wallet/wallettypes"
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
	mockedChainClient.EXPECT().PendingNonceAt(gomock.Any(), common.Address(accountAddress)).Times(1).Return(uint64(10), nil)
	state.ethClientGetter.EXPECT().EthClient(uint64(1)).AnyTimes().Return(mockedChainClient, nil)
	state.feeManager.EXPECT().SuggestedFees(gomock.Any(), uint64(1), common.Address(accountAddress)).Times(1).Return(
		&fees.SuggestedFees{
			GasPrice:             big.NewInt(1),
			BaseFee:              big.NewInt(1),
			MaxPriorityFeePerGas: big.NewInt(1),
			MaxFeesLevels: &fees.MaxFeesLevels{
				Low:                 (*hexutil.Big)(big.NewInt(1)),
				LowPriority:         (*hexutil.Big)(big.NewInt(1)),
				LowEstimatedTime:    1,
				Medium:              (*hexutil.Big)(big.NewInt(1)),
				MediumPriority:      (*hexutil.Big)(big.NewInt(1)),
				MediumEstimatedTime: 1,
				High:                (*hexutil.Big)(big.NewInt(1)),
				HighPriority:        (*hexutil.Big)(big.NewInt(1)),
				HighEstimatedTime:   1,
			},
			EIP1559Enabled: true,
		}, false, false, nil)

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

	signal.SetHandler(signal.Handler(func(s []byte) {
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
	t.Cleanup(signal.ResetHandler)

	mockedChainClient := mock_client.NewMockClientInterface(state.mockCtrl)
	mockedChainClient.EXPECT().PendingNonceAt(gomock.Any(), common.Address(accountAddress)).Times(1).Return(uint64(10), nil)
	state.ethClientGetter.EXPECT().EthClient(uint64(1)).AnyTimes().Return(mockedChainClient, nil)
	state.feeManager.EXPECT().SuggestedFees(gomock.Any(), uint64(1), common.Address(accountAddress)).Times(1).Return(
		&fees.SuggestedFees{
			GasPrice:             big.NewInt(1),
			BaseFee:              big.NewInt(1),
			MaxPriorityFeePerGas: big.NewInt(1),
			MaxFeesLevels: &fees.MaxFeesLevels{
				Low:                 (*hexutil.Big)(big.NewInt(1)),
				LowPriority:         (*hexutil.Big)(big.NewInt(1)),
				LowEstimatedTime:    1,
				Medium:              (*hexutil.Big)(big.NewInt(1)),
				MediumPriority:      (*hexutil.Big)(big.NewInt(1)),
				MediumEstimatedTime: 1,
				High:                (*hexutil.Big)(big.NewInt(1)),
				HighPriority:        (*hexutil.Big)(big.NewInt(1)),
				HighEstimatedTime:   1,
			},
			EIP1559Enabled: true,
		}, false, false, nil)

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

	signal.SetHandler(signal.Handler(func(s []byte) {
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
	t.Cleanup(signal.ResetHandler)

	mockedChainClient := mock_client.NewMockClientInterface(state.mockCtrl)
	mockedChainClient.EXPECT().PendingNonceAt(gomock.Any(), common.Address(accountAddress)).Times(1).Return(uint64(10), nil)
	state.ethClientGetter.EXPECT().EthClient(uint64(1)).AnyTimes().Return(mockedChainClient, nil)
	state.feeManager.EXPECT().SuggestedFees(gomock.Any(), uint64(1), common.Address(accountAddress)).Times(1).Return(
		&fees.SuggestedFees{
			GasPrice:             big.NewInt(1),
			BaseFee:              big.NewInt(1),
			MaxPriorityFeePerGas: big.NewInt(1),
			MaxFeesLevels: &fees.MaxFeesLevels{
				Low:                 (*hexutil.Big)(big.NewInt(1)),
				LowPriority:         (*hexutil.Big)(big.NewInt(1)),
				LowEstimatedTime:    1,
				Medium:              (*hexutil.Big)(big.NewInt(1)),
				MediumPriority:      (*hexutil.Big)(big.NewInt(1)),
				MediumEstimatedTime: 1,
				High:                (*hexutil.Big)(big.NewInt(1)),
				HighPriority:        (*hexutil.Big)(big.NewInt(1)),
				HighEstimatedTime:   1,
			},
			EIP1559Enabled: true,
		}, false, false, nil)

	_, err = state.cmd.Execute(state.ctx, request)
	assert.Equal(t, ErrSendTransactionRejectedByUser, err)
}
