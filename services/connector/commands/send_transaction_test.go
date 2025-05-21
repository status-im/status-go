package commands

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	gethTrie "github.com/ethereum/go-ethereum/trie"
	"github.com/status-im/status-go/eth-node/types"
	mock_client "github.com/status-im/status-go/rpc/chain/mock/client"
	"github.com/status-im/status-go/services/wallet/router/fees"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/signal"
)

const (
	blocksToCheck = 5
	blockNumber   = uint64(10)
)

var blockToReturn = gethTypes.NewBlock(&gethTypes.Header{
	Number: big.NewInt(10),
	Time:   uint64(time.Now().Unix()),
},
	[]*gethTypes.Transaction{
		gethTypes.NewTransaction(0, common.HexToAddress(""), big.NewInt(1), 100000, big.NewInt(1), nil),
		gethTypes.NewTransaction(0, common.HexToAddress(""), big.NewInt(1), 100000, big.NewInt(2), nil),
		gethTypes.NewTransaction(0, common.HexToAddress(""), big.NewInt(1), 100000, big.NewInt(3), nil),
		gethTypes.NewTransaction(0, common.HexToAddress(""), big.NewInt(1), 100000, big.NewInt(4), nil),
		gethTypes.NewTransaction(0, common.HexToAddress(""), big.NewInt(1), 100000, big.NewInt(5), nil),
		gethTypes.NewTransaction(0, common.HexToAddress(""), big.NewInt(1), 100000, big.NewInt(6), nil),
		gethTypes.NewTransaction(0, common.HexToAddress(""), big.NewInt(1), 100000, big.NewInt(7), nil),
		gethTypes.NewTransaction(0, common.HexToAddress(""), big.NewInt(1), 100000, big.NewInt(8), nil),
		gethTypes.NewTransaction(0, common.HexToAddress(""), big.NewInt(1), 100000, big.NewInt(9), nil),
		gethTypes.NewTransaction(0, common.HexToAddress(""), big.NewInt(1), 100000, big.NewInt(10), nil),
	},
	nil,
	nil,
	gethTrie.NewStackTrie(nil),
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
	feeHistory := &fees.FeeHistory{}
	percentiles := []int{fees.RewardPercentiles1, fees.RewardPercentiles2, fees.RewardPercentiles3}
	state.rpcClient.EXPECT().Call(feeHistory, uint64(1), "eth_feeHistory", uint64(10), "latest", percentiles).Times(1).Return(nil)
	state.rpcClient.EXPECT().EthClient(uint64(1)).Times(2).Return(mockedChainClient, nil)
	mockedChainClient.EXPECT().BlockNumber(state.ctx).Times(1).Return(blockNumber, nil)
	for i := uint64(0); i < uint64(blocksToCheck); i++ {
		blockNum := big.NewInt(0).SetUint64(blockNumber - i)
		mockedChainClient.EXPECT().BlockByNumber(state.ctx, blockNum).Times(1).Return(blockToReturn, nil)
	}
	mockedChainClient.EXPECT().SuggestGasPrice(state.ctx).Times(1).Return(big.NewInt(1), nil)
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
	feeHistory := &fees.FeeHistory{}
	percentiles := []int{fees.RewardPercentiles1, fees.RewardPercentiles2, fees.RewardPercentiles3}
	state.rpcClient.EXPECT().Call(feeHistory, uint64(1), "eth_feeHistory", uint64(10), "latest", percentiles).Times(1).Return(nil)
	state.rpcClient.EXPECT().EthClient(uint64(1)).Times(2).Return(mockedChainClient, nil)
	mockedChainClient.EXPECT().BlockNumber(state.ctx).Times(1).Return(blockNumber, nil)
	for i := uint64(0); i < uint64(blocksToCheck); i++ {
		blockNum := big.NewInt(0).SetUint64(blockNumber - i)
		mockedChainClient.EXPECT().BlockByNumber(state.ctx, blockNum).Times(1).Return(blockToReturn, nil)
	}
	mockedChainClient.EXPECT().SuggestGasPrice(state.ctx).Times(1).Return(big.NewInt(1), nil)
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
	feeHistory := &fees.FeeHistory{}
	percentiles := []int{fees.RewardPercentiles1, fees.RewardPercentiles2, fees.RewardPercentiles3}
	state.rpcClient.EXPECT().Call(feeHistory, uint64(1), "eth_feeHistory", uint64(10), "latest", percentiles).Times(1).Return(nil)
	state.rpcClient.EXPECT().EthClient(uint64(1)).Times(2).Return(mockedChainClient, nil)
	mockedChainClient.EXPECT().BlockNumber(state.ctx).Times(1).Return(blockNumber, nil)
	for i := uint64(0); i < uint64(blocksToCheck); i++ {
		blockNum := big.NewInt(0).SetUint64(blockNumber - i)
		mockedChainClient.EXPECT().BlockByNumber(state.ctx, blockNum).Times(1).Return(blockToReturn, nil)
	}
	mockedChainClient.EXPECT().SuggestGasPrice(state.ctx).Times(1).Return(big.NewInt(1), nil)
	state.rpcClient.EXPECT().EthClient(uint64(1)).Times(1).Return(mockedChainClient, nil)
	mockedChainClient.EXPECT().PendingNonceAt(state.ctx, common.Address(accountAddress)).Times(1).Return(uint64(10), nil)

	_, err = state.cmd.Execute(state.ctx, request)
	assert.Equal(t, ErrSendTransactionRejectedByUser, err)
}
