package transactions

import (
	"context"
	"database/sql"
	"encoding/json"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/rpc/chain/ethclient"
	mock_rpcclient "github.com/status-im/status-go/rpc/mock/client"

	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/walletevent"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
)

// setupTestTransactionDB will use the default pending check interval if checkInterval is nil
func setupTestTransactionDB(t *testing.T, checkInterval *time.Duration) (*PendingTxTracker, func(), *MockChainClient, *event.Feed) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	chainClient := NewMockChainClient()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	eventFeed := &event.Feed{}
	pendingCheckInterval := PendingCheckInterval
	if checkInterval != nil {
		pendingCheckInterval = *checkInterval
	}
	rpcClient := mock_rpcclient.NewMockClientInterface(ctrl)
	rpcClient.EXPECT().EthClient(common.EthereumMainnet).Return(chainClient, nil).AnyTimes()

	// Delegate the call to the fake implementation
	rpcClient.EXPECT().AbstractEthClient(gomock.Any()).DoAndReturn(func(chainID common.ChainID) (ethclient.BatchCallClient, error) {
		return chainClient.AbstractEthClient(chainID)
	}).AnyTimes()
	return NewPendingTxTracker(db, rpcClient, nil, eventFeed, pendingCheckInterval), func() {
		require.NoError(t, db.Close())
	}, chainClient, eventFeed
}

func waitForTaskToStop(pt *PendingTxTracker) {
	for pt.taskRunner.IsRunning() {
		time.Sleep(1 * time.Microsecond)
	}
}

func TestPendingTxTracker_ValidateConfirmedWithSuccessStatus(t *testing.T) {
	m, stop, chainClient, eventFeed := setupTestTransactionDB(t, nil)
	defer stop()

	txs := MockTestTransactions(t, chainClient, []TestTxSummary{{}})

	eventChan := make(chan walletevent.Event, 3)
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
				require.Equal(t, Success, p.Status)
			}
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for event")
		}
	}

	// Wait for the answer to be processed
	err = m.Stop()
	require.NoError(t, err)

	waitForTaskToStop(m)

	res, err := m.GetAllPending()
	require.NoError(t, err)
	require.Equal(t, 0, len(res))

	sub.Unsubscribe()
}

func TestPendingTxTracker_ValidateConfirmedWithFailedStatus(t *testing.T) {
	m, stop, chainClient, eventFeed := setupTestTransactionDB(t, nil)
	defer stop()

	txs := MockTestTransactions(t, chainClient, []TestTxSummary{{failStatus: true}})

	eventChan := make(chan walletevent.Event, 3)
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
				require.Equal(t, Failed, p.Status)
			}
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for event")
		}
	}

	// Wait for the answer to be processed
	err = m.Stop()
	require.NoError(t, err)

	waitForTaskToStop(m)

	res, err := m.GetAllPending()
	require.NoError(t, err)
	require.Equal(t, 0, len(res))

	sub.Unsubscribe()
}

func TestPendingTxTracker_InterruptWatching(t *testing.T) {
	m, stop, chainClient, eventFeed := setupTestTransactionDB(t, nil)
	defer stop()

	txs := GenerateTestPendingTransactions(0, 2)

	// Mock the first call to getTransactionByHash
	chainClient.SetAvailableClients([]common.ChainID{txs[0].ChainID})
	cl := chainClient.Clients[txs[0].ChainID]
	cl.On("BatchCallContext", mock.Anything, mock.MatchedBy(func(b []rpc.BatchElem) bool {
		return (len(b) == 2 && b[0].Method == GetTransactionReceiptRPCName && b[0].Args[0] == txs[0].Hash && b[1].Method == GetTransactionReceiptRPCName && b[1].Args[0] == txs[1].Hash)
	})).Return(nil).Once().Run(func(args mock.Arguments) {
		elems := args.Get(1).([]rpc.BatchElem)

		// Simulate still pending due to "null" return from eth_getTransactionReceipt
		elems[0].Result.(*nullableReceipt).Receipt = nil

		// Simulate parsing of eth_getTransactionReceipt response
		elems[1].Result.(*nullableReceipt).Receipt = &types.Receipt{
			BlockNumber: new(big.Int).SetUint64(1),
			Status:      1,
		}
	})

	eventChan := make(chan walletevent.Event, 2)
	sub := eventFeed.Subscribe(eventChan)

	for i := range txs {
		err := m.addPendingAndNotify(&txs[i])
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
				require.Equal(t, Success, p.Status)
			}
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for event")
		}
	}

	// Stop the next timed call
	err = m.Stop()
	require.NoError(t, err)

	waitForTaskToStop(m)

	res, err := m.GetAllPending()
	require.NoError(t, err)
	require.Equal(t, 1, len(res), "should have only one pending tx")

	// Restart the tracker to process leftovers
	//
	cl.On("BatchCallContext", mock.Anything, mock.MatchedBy(func(b []rpc.BatchElem) bool {
		return (len(b) == 1 && b[0].Method == GetTransactionReceiptRPCName && b[0].Args[0] == txs[0].Hash)
	})).Return(nil).Once().Run(func(args mock.Arguments) {
		elems := args.Get(1).([]rpc.BatchElem)
		// Simulate parsing of eth_getTransactionReceipt response
		elems[0].Result.(*nullableReceipt).Receipt = &types.Receipt{
			BlockNumber: new(big.Int).SetUint64(1),
			Status:      1,
		}
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
				require.Equal(t, Success, p.Status)
			}
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for event")
		}
	}

	err = m.Stop()
	require.NoError(t, err)

	waitForTaskToStop(m)

	res, err = m.GetAllPending()
	require.NoError(t, err)
	require.Equal(t, 0, len(res))

	sub.Unsubscribe()
}

func TestPendingTxTracker_MultipleClients(t *testing.T) {
	m, stop, chainClient, eventFeed := setupTestTransactionDB(t, nil)
	defer stop()

	txs := GenerateTestPendingTransactions(0, 2)
	txs[1].ChainID++

	// Mock the both clients to be available
	chainClient.SetAvailableClients([]common.ChainID{txs[0].ChainID, txs[1].ChainID})
	cl := chainClient.Clients[txs[0].ChainID]
	cl.On("BatchCallContext", mock.Anything, mock.MatchedBy(func(b []rpc.BatchElem) bool {
		return (len(b) == 1 && b[0].Method == GetTransactionReceiptRPCName && b[0].Args[0] == txs[0].Hash)
	})).Return(nil).Once().Run(func(args mock.Arguments) {
		elems := args.Get(1).([]rpc.BatchElem)
		// Simulate parsing of eth_getTransactionReceipt response
		elems[0].Result.(*nullableReceipt).Receipt = &types.Receipt{
			BlockNumber: new(big.Int).SetUint64(1),
			Status:      1,
		}
	})
	cl = chainClient.Clients[txs[1].ChainID]
	cl.On("BatchCallContext", mock.Anything, mock.MatchedBy(func(b []rpc.BatchElem) bool {
		return (len(b) == 1 && b[0].Method == GetTransactionReceiptRPCName && b[0].Args[0] == txs[1].Hash)
	})).Return(nil).Once().Run(func(args mock.Arguments) {
		elems := args.Get(1).([]rpc.BatchElem)
		// Simulate parsing of eth_getTransactionReceipt response
		elems[0].Result.(*nullableReceipt).Receipt = &types.Receipt{
			BlockNumber: new(big.Int).SetUint64(1),
			Status:      1,
		}
	})

	eventChan := make(chan walletevent.Event, 6)
	sub := eventFeed.Subscribe(eventChan)

	for i := range txs {
		err := m.TrackPendingTransaction(txs[i].ChainID, txs[i].Hash, txs[i].From, txs[i].To, txs[i].Type, AutoDelete, "")
		require.NoError(t, err)
	}

	err := m.Start()
	require.NoError(t, err)

	storeEventCount := 0
	statusEventCount := 0

	validateStatusChange := func(we *walletevent.Event) {
		if we.Type == EventPendingTransactionUpdate {
			storeEventCount++
		} else if we.Type == EventPendingTransactionStatusChanged {
			statusEventCount++
			require.Equal(t, EventPendingTransactionStatusChanged, we.Type)
			var p StatusChangedPayload
			err := json.Unmarshal([]byte(we.Message), &p)
			require.NoError(t, err)
			require.Equal(t, Success, p.Status)
		}
	}

	for i := 0; i < 2; i++ {
		for j := 0; j < 3; j++ {
			select {
			case we := <-eventChan:
				validateStatusChange(&we)
			case <-time.After(1 * time.Second):
				t.Fatal("timeout waiting for event", i, j, storeEventCount, statusEventCount)
			}
		}
	}

	require.Equal(t, 4, storeEventCount)
	require.Equal(t, 2, statusEventCount)

	err = m.Stop()
	require.NoError(t, err)

	waitForTaskToStop(m)

	res, err := m.GetAllPending()
	require.NoError(t, err)
	require.Equal(t, 0, len(res))

	sub.Unsubscribe()
}

func TestPendingTxTracker_Watch(t *testing.T) {
	m, stop, chainClient, eventFeed := setupTestTransactionDB(t, nil)
	defer stop()

	txs := GenerateTestPendingTransactions(0, 2)
	// Make the second already confirmed
	*txs[0].Status = Success

	// Mock the first call to getTransactionByHash
	chainClient.SetAvailableClients([]common.ChainID{txs[0].ChainID})
	cl := chainClient.Clients[txs[0].ChainID]
	cl.On("BatchCallContext", mock.Anything, mock.MatchedBy(func(b []rpc.BatchElem) bool {
		return len(b) == 1 && b[0].Method == GetTransactionReceiptRPCName && b[0].Args[0] == txs[1].Hash
	})).Return(nil).Once().Run(func(args mock.Arguments) {
		elems := args.Get(1).([]rpc.BatchElem)
		// Simulate parsing of eth_getTransactionReceipt response
		elems[0].Result.(*nullableReceipt).Receipt = &types.Receipt{
			BlockNumber: new(big.Int).SetUint64(1),
			Status:      1,
		}
	})

	eventChan := make(chan walletevent.Event, 3)
	sub := eventFeed.Subscribe(eventChan)

	// Track the first transaction
	err := m.TrackPendingTransaction(txs[1].ChainID, txs[1].Hash, txs[1].From, txs[1].To, txs[1].Type, Keep, "")
	require.NoError(t, err)

	// Store the confirmed already
	err = m.StoreAndTrackPendingTx(&txs[0])
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
				require.Equal(t, txs[1].ChainID, p.ChainID)
				require.Equal(t, txs[1].Hash, p.Hash)
				require.Equal(t, Success, p.Status)
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

	waitForTaskToStop(m)

	res, err := m.GetAllPending()
	require.NoError(t, err)
	require.Equal(t, 0, len(res), "should have no pending tx")

	status, err := m.Watch(context.Background(), txs[1].ChainID, txs[1].Hash)
	require.NoError(t, err)
	require.NotEqual(t, Pending, status)

	err = m.Delete(context.Background(), txs[1].ChainID, txs[1].Hash)
	require.NoError(t, err)

	select {
	case we := <-eventChan:
		require.Equal(t, EventPendingTransactionUpdate, we.Type)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for the delete event")
	}

	sub.Unsubscribe()
}

func TestPendingTxTracker_Watch_StatusChangeIncrementally(t *testing.T) {
	m, stop, chainClient, eventFeed := setupTestTransactionDB(t, common.NewAndSet(1*time.Nanosecond))
	defer stop()

	txs := GenerateTestPendingTransactions(0, 2)

	var firsDoneWG sync.WaitGroup
	firsDoneWG.Add(1)

	// Mock the first call to getTransactionByHash
	chainClient.SetAvailableClients([]common.ChainID{txs[0].ChainID})
	cl := chainClient.Clients[txs[0].ChainID]

	cl.On("BatchCallContext", mock.Anything, mock.MatchedBy(func(b []rpc.BatchElem) bool {
		if len(cl.Calls) == 0 {
			res := len(b) > 0 && b[0].Method == GetTransactionReceiptRPCName && b[0].Args[0] == txs[0].Hash
			// If the first processing call picked up the second validate this case also
			if len(b) == 2 {
				res = res && b[1].Method == GetTransactionReceiptRPCName && b[1].Args[0] == txs[1].Hash
			}
			return res
		}
		// Second call we expect only one left
		return len(b) == 1 && (b[0].Method == GetTransactionReceiptRPCName && b[0].Args[0] == txs[1].Hash)
	})).Return(nil).Twice().Run(func(args mock.Arguments) {
		elems := args.Get(1).([]rpc.BatchElem)
		if len(cl.Calls) == 2 {
			firsDoneWG.Wait()
		}
		// Only first item is processed, second is left pending
		// Simulate parsing of eth_getTransactionReceipt response
		elems[0].Result.(*nullableReceipt).Receipt = &types.Receipt{
			BlockNumber: new(big.Int).SetUint64(1),
			Status:      1,
		}
	})

	eventChan := make(chan walletevent.Event, 6)
	sub := eventFeed.Subscribe(eventChan)

	for i := range txs {
		// Track the first transaction
		err := m.TrackPendingTransaction(txs[i].ChainID, txs[i].Hash, txs[i].From, txs[i].To, txs[i].Type, Keep, "")
		require.NoError(t, err)
	}

	storeEventCount := 0
	statusEventCount := 0

	validateStatusChange := func(we *walletevent.Event) {
		var p StatusChangedPayload
		err := json.Unmarshal([]byte(we.Message), &p)
		require.NoError(t, err)

		if statusEventCount == 0 {
			require.Equal(t, txs[0].ChainID, p.ChainID)
			require.Equal(t, txs[0].Hash, p.Hash)
			require.Equal(t, Success, p.Status)

			status, err := m.Watch(context.Background(), txs[0].ChainID, txs[0].Hash)
			require.NoError(t, err)
			require.Equal(t, Success, *status)
			err = m.Delete(context.Background(), txs[0].ChainID, txs[0].Hash)
			require.NoError(t, err)

			status, err = m.Watch(context.Background(), txs[1].ChainID, txs[1].Hash)
			require.NoError(t, err)
			require.Equal(t, Pending, *status)
			firsDoneWG.Done()
		} else {
			_, err := m.Watch(context.Background(), txs[0].ChainID, txs[0].Hash)
			require.Equal(t, err, sql.ErrNoRows)

			status, err := m.Watch(context.Background(), txs[1].ChainID, txs[1].Hash)
			require.NoError(t, err)
			require.Equal(t, Success, *status)
			err = m.Delete(context.Background(), txs[1].ChainID, txs[1].Hash)
			require.NoError(t, err)
		}

		statusEventCount++
	}

	for j := 0; j < 6; j++ {
		select {
		case we := <-eventChan:
			if EventPendingTransactionUpdate == we.Type {
				storeEventCount++
			} else if EventPendingTransactionStatusChanged == we.Type {
				validateStatusChange(&we)
			}
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for the status update event")
		}
	}

	_, err := m.Watch(context.Background(), txs[1].ChainID, txs[1].Hash)
	require.Equal(t, err, sql.ErrNoRows)

	// One for add and one for delete
	require.Equal(t, 4, storeEventCount)
	require.Equal(t, 2, statusEventCount)

	err = m.Stop()
	require.NoError(t, err)

	waitForTaskToStop(m)

	res, err := m.GetAllPending()
	require.NoError(t, err)
	require.Equal(t, 0, len(res), "should have no pending tx")

	sub.Unsubscribe()
}

func TestPendingTransactions(t *testing.T) {
	manager, stop, _, _ := setupTestTransactionDB(t, nil)
	defer stop()

	tx := GenerateTestPendingTransactions(0, 1)[0]

	rst, err := manager.GetAllPending()
	require.NoError(t, err)
	require.Nil(t, rst)

	rst, err = manager.GetPendingByAddress([]uint64{777}, tx.From)
	require.NoError(t, err)
	require.Nil(t, rst)

	err = manager.addPendingAndNotify(&tx)
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
