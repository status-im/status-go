package transactions

import (
	"context"
	"fmt"
	"math/big"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/stretchr/testify/mock"
)

type MockETHClient struct {
	mock.Mock
}

func (m *MockETHClient) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	args := m.Called(ctx, b)
	return args.Error(0)
}

const (
	GetTransactionReceiptRPCName = "eth_getTransactionReceipt"
)

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
