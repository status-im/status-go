package transactions

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/common"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockETHClient struct {
	mock.Mock
}

func (m *MockETHClient) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	args := m.Called(ctx, b)
	return args.Error(0)
}

type MockChainClient struct {
	mock.Mock

	Clients map[common.ChainID]*MockETHClient
}

func NewMockChainClient() *MockChainClient {
	return &MockChainClient{
		Clients: make(map[common.ChainID]*MockETHClient),
	}
}

func (m *MockChainClient) SetAvailableClients(chainIDs []common.ChainID) *MockChainClient {
	for _, chainID := range chainIDs {
		if _, ok := m.Clients[chainID]; !ok {
			m.Clients[chainID] = new(MockETHClient)
		}
	}
	return m
}

func (m *MockChainClient) AbstractEthClient(chainID common.ChainID) (chain.BatchCallClient, error) {
	if _, ok := m.Clients[chainID]; !ok {
		panic(fmt.Sprintf("no mock client for chainID %d", chainID))
	}
	return m.Clients[chainID], nil
}

func GenerateTestPendingTransactions(count int) []PendingTransaction {
	if count > 127 {
		panic("can't generate more than 127 distinct transactions")
	}

	txs := make([]PendingTransaction, count)
	for i := 0; i < count; i++ {
		txs[i] = PendingTransaction{
			Hash:           eth.Hash{byte(i)},
			From:           eth.Address{byte(i)},
			To:             eth.Address{byte(i * 2)},
			Type:           RegisterENS,
			AdditionalData: "someuser.stateofus.eth",
			Value:          bigint.BigInt{Int: big.NewInt(int64(i))},
			GasLimit:       bigint.BigInt{Int: big.NewInt(21000)},
			GasPrice:       bigint.BigInt{Int: big.NewInt(int64(i))},
			ChainID:        777,
			Status:         new(TxStatus),
			AutoDelete:     new(bool),
		}
		*txs[i].Status = Pending  // set to pending by default
		*txs[i].AutoDelete = true // set to true by default
	}
	return txs
}

type TestTxSummary struct {
	failStatus  bool
	dontConfirm bool
}

func MockTestTransactions(t *testing.T, chainClient *MockChainClient, testTxs []TestTxSummary) []PendingTransaction {
	txs := GenerateTestPendingTransactions(len(testTxs))

	// Mock the first call to getTransactionByHash
	chainClient.SetAvailableClients([]common.ChainID{txs[0].ChainID})
	cl := chainClient.Clients[txs[0].ChainID]
	cl.On("BatchCallContext", mock.Anything, mock.MatchedBy(func(b []rpc.BatchElem) bool {
		ok := len(b) == len(testTxs)
		for i := range b {
			ok = ok && b[i].Method == GetTransactionReceiptRPCName && b[i].Args[0] == txs[0].Hash
		}
		return ok
	})).Return(nil).Once().Run(func(args mock.Arguments) {
		elems := args.Get(1).([]rpc.BatchElem)
		for i := range elems {
			receiptWrapper, ok := elems[i].Result.(*nullableReceipt)
			require.True(t, ok)
			require.NotNil(t, receiptWrapper)
			// Simulate parsing of eth_getTransactionReceipt response
			if !testTxs[i].dontConfirm {
				status := types.ReceiptStatusSuccessful
				if testTxs[i].failStatus {
					status = types.ReceiptStatusFailed
				}

				receiptWrapper.Receipt = &types.Receipt{
					BlockNumber: new(big.Int).SetUint64(1),
					Status:      status,
				}
			}
		}
	})
	return txs
}
