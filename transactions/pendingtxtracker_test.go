package transactions

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/walletevent"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
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

	clients map[common.ChainID]*MockETHClient
}

func newMockChainClient() *MockChainClient {
	return &MockChainClient{
		clients: make(map[common.ChainID]*MockETHClient),
	}
}

func (m *MockChainClient) setAvailableClients(chainIDs []common.ChainID) *MockChainClient {
	for _, chainID := range chainIDs {
		if _, ok := m.clients[chainID]; !ok {
			m.clients[chainID] = new(MockETHClient)
		}
	}
	return m
}

func (m *MockChainClient) AbstractEthClient(chainID common.ChainID) (chain.ClientInterface, error) {
	if _, ok := m.clients[chainID]; !ok {
		panic(fmt.Sprintf("no mock client for chainID %d", chainID))
	}
	return m.clients[chainID], nil
}

func setupTestTransactionDB(t *testing.T) (*PendingTxTracker, func(), *MockChainClient, *event.Feed) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	chainClient := newMockChainClient()
	eventFeed := &event.Feed{}
	return NewPendingTxTracker(db, chainClient, nil, eventFeed), func() {
		require.NoError(t, db.Close())
	}, chainClient, eventFeed
}

const (
	transactionSuccessStatus = "0x1"
	transactionFailStatus    = "0x0"
	transactionByHashRPCName = "eth_getTransactionByHash"
)

func TestPendingTxTracker_ValidateConfirmed(t *testing.T) {
	m, stop, chainClient, eventFeed := setupTestTransactionDB(t)
	defer stop()

	txs := generateTestTransactions(1)

	// Mock the first call to getTransactionByHash
	chainClient.setAvailableClients([]common.ChainID{txs[0].ChainID})
	cl := chainClient.clients[txs[0].ChainID]
	cl.On("BatchCallContext", mock.Anything, mock.MatchedBy(func(b []rpc.BatchElem) bool {
		return len(b) == 1 && b[0].Method == transactionByHashRPCName && b[0].Args[0] == txs[0].Hash
	})).Return(nil).Once().Run(func(args mock.Arguments) {
		elems := args.Get(1).([]rpc.BatchElem)
		res := elems[0].Result.(*map[string]interface{})
		(*res)["blockNumber"] = transactionSuccessStatus
	})

	eventChan := make(chan walletevent.Event, 2)
	sub := eventFeed.Subscribe(eventChan)

	err := m.StoreAndTrackPendingTx(&txs[0])
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		select {
		case we := <-eventChan:
			if i == 0 || i == 1 {
				// Check add and delete
				require.Equal(t, EventPendingTransactionUpdate, we.Type)
			} else {
				require.Equal(t, EventPendingTransactionStatusChanged, we.Type)
				var p StatusChangedPayload
				err = json.Unmarshal([]byte(we.Message), &p)
				require.NoError(t, err)
				require.Equal(t, txs[0].Hash, p.Hash)
				require.Nil(t, p.Status)
			}
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for event")
		}
	}

	// Wait for the answer to be processed
	err = m.Stop()
	require.NoError(t, err)

	res, err := m.GetAllPending()
	require.NoError(t, err)
	require.Equal(t, 0, len(res))

	sub.Unsubscribe()
}

func TestPendingTxTracker_InterruptWatching(t *testing.T) {
	m, stop, chainClient, eventFeed := setupTestTransactionDB(t)
	defer stop()

	txs := generateTestTransactions(2)

	// Mock the first call to getTransactionByHash
	chainClient.setAvailableClients([]common.ChainID{txs[0].ChainID})
	cl := chainClient.clients[txs[0].ChainID]
	cl.On("BatchCallContext", mock.Anything, mock.MatchedBy(func(b []rpc.BatchElem) bool {
		return (len(b) == 2 && b[0].Method == transactionByHashRPCName && b[0].Args[0] == txs[0].Hash && b[1].Method == transactionByHashRPCName && b[1].Args[0] == txs[1].Hash)
	})).Return(nil).Once().Run(func(args mock.Arguments) {
		elems := args.Get(1).([]rpc.BatchElem)
		res := elems[0].Result.(*map[string]interface{})
		(*res)["blockNumber"] = nil
		res = elems[1].Result.(*map[string]interface{})
		(*res)["blockNumber"] = transactionFailStatus
	})

	eventChan := make(chan walletevent.Event, 2)
	sub := eventFeed.Subscribe(eventChan)

	for i := range txs {
		err := m.addPending(&txs[i])
		require.NoError(t, err)
	}

	// Check add
	for i := 0; i < 2; i++ {
		select {
		case we := <-eventChan:
			require.Equal(t, EventPendingTransactionUpdate, we.Type)
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for event")
		}
	}

	err := m.Start()
	require.NoError(t, err)

	for i := 0; i < 2; i++ {
		select {
		case we := <-eventChan:
			if i == 0 {
				require.Equal(t, EventPendingTransactionUpdate, we.Type)
			} else {
				require.Equal(t, EventPendingTransactionStatusChanged, we.Type)
				var p StatusChangedPayload
				err := json.Unmarshal([]byte(we.Message), &p)
				require.NoError(t, err)
				require.Equal(t, txs[1].Hash, p.Hash)
				require.Equal(t, txs[1].ChainID, p.ChainID)
				require.Nil(t, p.Status)
			}
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for event")
		}
	}

	// Stop the next timed call
	err = m.Stop()
	require.NoError(t, err)

	res, err := m.GetAllPending()
	require.NoError(t, err)
	require.Equal(t, 1, len(res), "should have only one pending tx")

	// Restart the tracker to process leftovers
	//
	cl.On("BatchCallContext", mock.Anything, mock.MatchedBy(func(b []rpc.BatchElem) bool {
		return (len(b) == 1 && b[0].Method == transactionByHashRPCName && b[0].Args[0] == txs[0].Hash)
	})).Return(nil).Once().Run(func(args mock.Arguments) {
		elems := args.Get(1).([]rpc.BatchElem)
		res := elems[0].Result.(*map[string]interface{})
		(*res)["blockNumber"] = transactionSuccessStatus
	})

	err = m.Start()
	require.NoError(t, err)

	for i := 0; i < 2; i++ {
		select {
		case we := <-eventChan:
			if i == 0 {
				require.Equal(t, EventPendingTransactionUpdate, we.Type)
			} else {
				require.Equal(t, EventPendingTransactionStatusChanged, we.Type)
				var p StatusChangedPayload
				err := json.Unmarshal([]byte(we.Message), &p)
				require.NoError(t, err)
				require.Equal(t, txs[0].ChainID, p.ChainID)
				require.Equal(t, txs[0].Hash, p.Hash)
				require.Nil(t, p.Status)
			}
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for event")
		}
	}

	err = m.Stop()
	require.NoError(t, err)

	res, err = m.GetAllPending()
	require.NoError(t, err)
	require.Equal(t, 0, len(res))

	sub.Unsubscribe()
}

func TestPendingTxTracker_MultipleClients(t *testing.T) {
	m, stop, chainClient, eventFeed := setupTestTransactionDB(t)
	defer stop()

	txs := generateTestTransactions(2)
	txs[1].ChainID++

	// Mock the both clients to be available
	chainClient.setAvailableClients([]common.ChainID{txs[0].ChainID, txs[1].ChainID})
	cl := chainClient.clients[txs[0].ChainID]
	cl.On("BatchCallContext", mock.Anything, mock.MatchedBy(func(b []rpc.BatchElem) bool {
		return (len(b) == 1 && b[0].Method == transactionByHashRPCName && b[0].Args[0] == txs[0].Hash)
	})).Return(nil).Once().Run(func(args mock.Arguments) {
		elems := args.Get(1).([]rpc.BatchElem)
		res := elems[0].Result.(*map[string]interface{})
		(*res)["blockNumber"] = transactionFailStatus
	})
	cl = chainClient.clients[txs[1].ChainID]
	cl.On("BatchCallContext", mock.Anything, mock.MatchedBy(func(b []rpc.BatchElem) bool {
		return (len(b) == 1 && b[0].Method == transactionByHashRPCName && b[0].Args[0] == txs[1].Hash)
	})).Return(nil).Once().Run(func(args mock.Arguments) {
		elems := args.Get(1).([]rpc.BatchElem)
		res := elems[0].Result.(*map[string]interface{})
		(*res)["blockNumber"] = transactionSuccessStatus
	})

	// If we call TrackPendingTransaction immediately, there is a chance that some events
	// will be emitted before we reach select, so occasionally select fails on timeout.
	go func() {
		for i := range txs {
			err := m.TrackPendingTransaction(txs[i].ChainID, txs[i].Hash, txs[i].From, txs[i].Type, true)
			require.NoError(t, err)
		}
	}()

	eventChan := make(chan walletevent.Event)
	sub := eventFeed.Subscribe(eventChan)

	// events caused by addPending
	for i := 0; i < 2; i++ {
		select {
		case we := <-eventChan:
			require.Equal(t, EventPendingTransactionUpdate, we.Type)
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for event")
		}
	}

	// events caused by tx deletion on fetch
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			select {
			case we := <-eventChan:
				if j == 0 {
					require.Equal(t, EventPendingTransactionUpdate, we.Type)
				} else {
					require.Equal(t, EventPendingTransactionStatusChanged, we.Type)
					var p StatusChangedPayload
					err := json.Unmarshal([]byte(we.Message), &p)
					require.NoError(t, err)
					require.Nil(t, p.Status)
				}
			case <-time.After(1 * time.Second):
				t.Fatal("timeout waiting for event")
			}
		}
	}

	err := m.Stop()
	require.NoError(t, err)

	res, err := m.GetAllPending()
	require.NoError(t, err)
	require.Equal(t, 0, len(res))

	sub.Unsubscribe()
}

func TestPendingTxTracker_Watch(t *testing.T) {
	m, stop, chainClient, eventFeed := setupTestTransactionDB(t)
	defer stop()

	txs := generateTestTransactions(2)
	// Make the second already confirmed
	*txs[1].Status = Done

	// Mock the first call to getTransactionByHash
	chainClient.setAvailableClients([]common.ChainID{txs[0].ChainID})
	cl := chainClient.clients[txs[0].ChainID]
	cl.On("BatchCallContext", mock.Anything, mock.MatchedBy(func(b []rpc.BatchElem) bool {
		return len(b) == 1 && b[0].Method == transactionByHashRPCName && b[0].Args[0] == txs[0].Hash
	})).Return(nil).Once().Run(func(args mock.Arguments) {
		elems := args.Get(1).([]rpc.BatchElem)
		res := elems[0].Result.(*map[string]interface{})
		(*res)["blockNumber"] = transactionFailStatus
	})

	eventChan := make(chan walletevent.Event, 2)
	sub := eventFeed.Subscribe(eventChan)

	// Track the first transaction
	err := m.TrackPendingTransaction(txs[0].ChainID, txs[0].Hash, txs[0].From, txs[0].Type, false)
	require.NoError(t, err)

	// Store the confirmed already
	err = m.StoreAndTrackPendingTx(&txs[1])
	require.NoError(t, err)

	storeEventCount := 0
	statusEventCount := 0
	for j := 0; j < 3; j++ {
		select {
		case we := <-eventChan:
			if EventPendingTransactionUpdate == we.Type {
				storeEventCount++
			} else if EventPendingTransactionStatusChanged == we.Type {
				statusEventCount++
				var p StatusChangedPayload
				err := json.Unmarshal([]byte(we.Message), &p)
				require.NoError(t, err)
				require.Equal(t, txs[0].ChainID, p.ChainID)
				require.Equal(t, txs[0].Hash, p.Hash)
				require.Nil(t, p.Status)
			}
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for the status update event")
		}
	}
	require.Equal(t, 2, storeEventCount)
	require.Equal(t, 1, statusEventCount)

	// Stop the next timed call
	err = m.Stop()
	require.NoError(t, err)

	res, err := m.GetAllPending()
	require.NoError(t, err)
	require.Equal(t, 0, len(res), "should have only one pending tx")

	status, err := m.Watch(context.Background(), txs[0].ChainID, txs[0].Hash)
	require.NoError(t, err)
	require.NotEqual(t, Pending, status)

	err = m.Delete(context.Background(), txs[0].ChainID, txs[0].Hash)
	require.NoError(t, err)

	select {
	case we := <-eventChan:
		require.Equal(t, EventPendingTransactionUpdate, we.Type)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for the delete event")
	}

	sub.Unsubscribe()
}

func TestPendingTransactions(t *testing.T) {
	manager, stop, _, _ := setupTestTransactionDB(t)
	defer stop()

	tx := generateTestTransactions(1)[0]

	rst, err := manager.GetAllPending()
	require.NoError(t, err)
	require.Nil(t, rst)

	rst, err = manager.GetPendingByAddress([]uint64{777}, tx.From)
	require.NoError(t, err)
	require.Nil(t, rst)

	err = manager.addPending(&tx)
	require.NoError(t, err)

	rst, err = manager.GetPendingByAddress([]uint64{777}, tx.From)
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
	require.Equal(t, tx, *rst[0])

	rst, err = manager.GetAllPending()
	require.NoError(t, err)
	require.Equal(t, 1, len(rst))
	require.Equal(t, tx, *rst[0])

	rst, err = manager.GetPendingByAddress([]uint64{777}, eth.Address{2})
	require.NoError(t, err)
	require.Nil(t, rst)

	err = manager.Delete(context.Background(), common.ChainID(777), tx.Hash)
	require.Error(t, err, ErrStillPending)

	rst, err = manager.GetPendingByAddress([]uint64{777}, tx.From)
	require.NoError(t, err)
	require.Equal(t, 0, len(rst))

	rst, err = manager.GetAllPending()
	require.NoError(t, err)
	require.Equal(t, 0, len(rst))
}

func generateTestTransactions(count int) []PendingTransaction {
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
