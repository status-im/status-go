package pendingtxtracker_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/status-go/internal/db/walletdb"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/pkg/services/wallet/pendingtxtracker"
	mock_pendingtxtracker "github.com/status-im/status-go/pkg/services/wallet/pendingtxtracker/mock"

	ac "github.com/status-im/status-go/pkg/services/wallet/activity/common"
	"github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/walletevent"
)

type testState struct {
	db                   *sql.DB
	ctrl                 *gomock.Controller
	txStatusFetcher      *mock_pendingtxtracker.MockTxStatusFetcher
	eventFeed            *event.Feed
	pendingCheckInterval time.Duration
	pendingTxTracker     *pendingtxtracker.PendingTxTracker
	close                func()
}

// setupTestTransactionDB will use the default pending check interval if checkInterval is nil
func setupTestTransactionDB(t *testing.T, checkInterval *time.Duration) testState {
	db, err := testutils.SetupTestMemorySQLDB(walletdb.DbInitializer{})
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	txStatusFetcher := mock_pendingtxtracker.NewMockTxStatusFetcher(ctrl)
	defer ctrl.Finish()
	eventFeed := &event.Feed{}
	pendingCheckInterval := pendingtxtracker.PendingCheckInterval
	if checkInterval != nil {
		pendingCheckInterval = *checkInterval
	}

	return testState{
		db:                   db,
		ctrl:                 ctrl,
		txStatusFetcher:      txStatusFetcher,
		eventFeed:            eventFeed,
		pendingCheckInterval: pendingCheckInterval,
		pendingTxTracker:     pendingtxtracker.NewPendingTxTracker(db, txStatusFetcher, eventFeed, pendingCheckInterval),
		close: func() {
			require.NoError(t, db.Close())
			ctrl.Finish()
		},
	}
}

func waitForTaskToStop(pt *pendingtxtracker.PendingTxTracker) {
	for pt.IsRunning() {
		time.Sleep(1 * time.Microsecond)
	}
}

func unpackUpdateEvent(t *testing.T, we walletevent.Event) pendingtxtracker.PendingTxUpdatePayload {
	var p pendingtxtracker.PendingTxUpdatePayload
	err := json.Unmarshal([]byte(we.Message), &p)
	require.NoError(t, err)
	return p
}

func unpackStatusChangedEvent(t *testing.T, we walletevent.Event) pendingtxtracker.StatusChangedPayload {
	var p pendingtxtracker.StatusChangedPayload
	err := json.Unmarshal([]byte(we.Message), &p)
	require.NoError(t, err)
	return p
}

func TestPendingTxTracker_ValidateConfirmedWithSuccessStatus(t *testing.T) {
	state := setupTestTransactionDB(t, nil)
	defer state.close()

	chainID := uint64(777)
	txs := pendingtxtracker.GenerateTestPendingTransactions(0, 1, chainID)
	state.txStatusFetcher.EXPECT().FetchTxStatus(gomock.Any(), txs[0].ChainID, gomock.Any()).
		Return([]pendingtxtracker.TxStatusRes{{Status: ac.Success, Hash: txs[0].Hash}}, nil).AnyTimes()

	eventChan := make(chan walletevent.Event, 3)
	defer close(eventChan)
	sub := state.eventFeed.Subscribe(eventChan)

	err := state.pendingTxTracker.StoreAndTrackPendingTx(txs[0])
	require.NoError(t, err)

	events := make([]walletevent.Event, 0, 3)

	for i := 0; i < 3; i++ {
		select {
		case we := <-eventChan:
			events = append(events, we)
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for all events", len(events))
		}
	}

	updateCount := 0
	statusChangedCount := 0
	for _, we := range events {
		switch we.Type {
		case pendingtxtracker.EventPendingTransactionUpdate:
			updateCount++
		case pendingtxtracker.EventPendingTransactionStatusChanged:
			statusChangedCount++
			var p pendingtxtracker.StatusChangedPayload
			err := json.Unmarshal([]byte(we.Message), &p)
			require.NoError(t, err)
			require.Equal(t, txs[0].Hash, p.Hash)
			require.Equal(t, ac.Success, p.Status)
		}
	}

	// Wait for the answer to be processed
	err = state.pendingTxTracker.Stop()
	require.NoError(t, err)

	waitForTaskToStop(state.pendingTxTracker)

	res, err := state.pendingTxTracker.GetAllPending()
	require.NoError(t, err)
	require.Equal(t, 0, len(res))

	sub.Unsubscribe()
}

func TestPendingTxTracker_ValidateConfirmedWithFailedStatus(t *testing.T) {
	state := setupTestTransactionDB(t, nil)
	defer state.close()

	chainID := uint64(777)
	txs := pendingtxtracker.GenerateTestPendingTransactions(0, 1, chainID)
	state.txStatusFetcher.EXPECT().FetchTxStatus(gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]pendingtxtracker.TxStatusRes{{Status: ac.Failed, Hash: txs[0].Hash}}, nil).AnyTimes()

	eventChan := make(chan walletevent.Event, 3)
	sub := state.eventFeed.Subscribe(eventChan)

	err := state.pendingTxTracker.StoreAndTrackPendingTx(txs[0])
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		select {
		case we := <-eventChan:
			if i == 0 || i == 1 {
				// Check add and delete
				require.Equal(t, pendingtxtracker.EventPendingTransactionUpdate, we.Type)
			} else {
				require.Equal(t, pendingtxtracker.EventPendingTransactionStatusChanged, we.Type)
				var p pendingtxtracker.StatusChangedPayload
				err = json.Unmarshal([]byte(we.Message), &p)
				require.NoError(t, err)
				require.Equal(t, txs[0].Hash, p.Hash)
				require.Equal(t, ac.Failed, p.Status)
			}
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for event")
		}
	}

	// Wait for the answer to be processed
	err = state.pendingTxTracker.Stop()
	require.NoError(t, err)

	waitForTaskToStop(state.pendingTxTracker)

	res, err := state.pendingTxTracker.GetAllPending()
	require.NoError(t, err)
	require.Equal(t, 0, len(res))

	sub.Unsubscribe()
}

func TestPendingTxTracker_InterruptWatching(t *testing.T) {
	state := setupTestTransactionDB(t, nil)
	defer state.close()

	chainID := uint64(777)
	txs := pendingtxtracker.GenerateTestPendingTransactions(0, 2, chainID)

	state.txStatusFetcher.EXPECT().FetchTxStatus(gomock.Any(), common.ChainID(chainID), gomock.Any()).DoAndReturn(
		func(ctx context.Context, chainID common.ChainID, hashes []eth.Hash) ([]pendingtxtracker.TxStatusRes, error) {
			require.Equal(t, 2, len(hashes))
			require.Contains(t, hashes, txs[0].Hash)
			require.Contains(t, hashes, txs[1].Hash)
			return []pendingtxtracker.TxStatusRes{
				{Status: ac.Success, Hash: txs[1].Hash},
			}, nil
		}).Times(1)

	eventChan := make(chan walletevent.Event, 10)
	sub := state.eventFeed.Subscribe(eventChan)

	err := state.pendingTxTracker.StoreAndTrackPendingTxs(txs)
	require.NoError(t, err)

	tx0Updates := make([]pendingtxtracker.PendingTxUpdatePayload, 0, 2)
	tx1Updates := make([]pendingtxtracker.PendingTxUpdatePayload, 0, 2)
	tx0StatusChanges := make([]pendingtxtracker.StatusChangedPayload, 0, 2)
	tx1StatusChanges := make([]pendingtxtracker.StatusChangedPayload, 0, 2)

	require.Eventually(t, func() bool {
		select {
		case we := <-eventChan:
			switch we.Type {
			case pendingtxtracker.EventPendingTransactionUpdate:
				updatePayload := unpackUpdateEvent(t, we)
				if updatePayload.TxIdentity == txs[0].Identity() {
					tx0Updates = append(tx0Updates, updatePayload)
				} else if updatePayload.TxIdentity == txs[1].Identity() {
					tx1Updates = append(tx1Updates, updatePayload)
				} else {
					t.Fatal("unexpected tx identity", updatePayload.TxIdentity)
				}
			case pendingtxtracker.EventPendingTransactionStatusChanged:
				statusChangedPayload := unpackStatusChangedEvent(t, we)
				if statusChangedPayload.TxIdentity == txs[0].Identity() {
					tx0StatusChanges = append(tx0StatusChanges, statusChangedPayload)
				} else if statusChangedPayload.TxIdentity == txs[1].Identity() {
					tx1StatusChanges = append(tx1StatusChanges, statusChangedPayload)
				} else {
					t.Fatal("unexpected tx identity", statusChangedPayload.TxIdentity)
				}
			}
		default:
			break
		}

		// We should have 1 update for tx0 (added) and 2 for tx1 (added and removed).
		// We should have 0 status changes for tx0 and 1 for tx1 (success).
		return len(tx0Updates) == 1 && len(tx1Updates) == 2 && len(tx0StatusChanges) == 0 && len(tx1StatusChanges) == 1
	}, 1*time.Second, 10*time.Millisecond)

	require.False(t, tx0Updates[0].Deleted)
	require.NotEqual(t, tx1Updates[0].Deleted, tx1Updates[1].Deleted)
	require.Equal(t, ac.Success, tx1StatusChanges[0].Status)

	// Stop the next timed call
	err = state.pendingTxTracker.Stop()
	require.NoError(t, err)

	waitForTaskToStop(state.pendingTxTracker)

	res, err := state.pendingTxTracker.GetAllPending()
	require.NoError(t, err)
	require.Equal(t, 1, len(res), "should have only one pending tx")

	// Restart the tracker to process leftovers
	state.txStatusFetcher.EXPECT().FetchTxStatus(gomock.Any(), common.ChainID(chainID), gomock.Any()).DoAndReturn(
		func(ctx context.Context, chainID common.ChainID, hashes []eth.Hash) ([]pendingtxtracker.TxStatusRes, error) {
			require.Equal(t, 1, len(hashes))
			require.Contains(t, hashes, txs[0].Hash)
			return []pendingtxtracker.TxStatusRes{
				{Status: ac.Success, Hash: txs[0].Hash},
			}, nil
		}).Times(1)

	err = state.pendingTxTracker.Start()
	require.NoError(t, err)

	tx0Updates = make([]pendingtxtracker.PendingTxUpdatePayload, 0, 2)
	tx0StatusChanges = make([]pendingtxtracker.StatusChangedPayload, 0, 2)

	require.Eventually(t, func() bool {
		select {
		case we := <-eventChan:
			switch we.Type {
			case pendingtxtracker.EventPendingTransactionUpdate:
				updatePayload := unpackUpdateEvent(t, we)
				if updatePayload.TxIdentity == txs[0].Identity() {
					tx0Updates = append(tx0Updates, updatePayload)
				} else {
					t.Fatal("unexpected tx identity", updatePayload.TxIdentity)
				}
			case pendingtxtracker.EventPendingTransactionStatusChanged:
				statusChangedPayload := unpackStatusChangedEvent(t, we)
				if statusChangedPayload.TxIdentity == txs[0].Identity() {
					tx0StatusChanges = append(tx0StatusChanges, statusChangedPayload)
				} else {
					t.Fatal("unexpected tx identity", statusChangedPayload.TxIdentity)
				}
			}
		default:
			break
		}

		// We should have 1 update for tx0 (removed).
		// We should have 1 status changes for tx0.
		return len(tx0Updates) == 1 && len(tx0StatusChanges) == 1
	}, 1*time.Second, 10*time.Millisecond)
	require.True(t, tx0Updates[0].Deleted)
	require.Equal(t, ac.Success, tx0StatusChanges[0].Status)

	err = state.pendingTxTracker.Stop()
	require.NoError(t, err)

	waitForTaskToStop(state.pendingTxTracker)

	res, err = state.pendingTxTracker.GetAllPending()
	require.NoError(t, err)
	require.Equal(t, 0, len(res))

	sub.Unsubscribe()
}

func TestPendingTxTracker_MultipleClients(t *testing.T) {
	state := setupTestTransactionDB(t, nil)
	defer state.close()

	chainID0 := uint64(777)
	chainID1 := uint64(778)
	txs := pendingtxtracker.GenerateTestPendingTransactions(0, 1, chainID0)
	txs = append(txs, pendingtxtracker.GenerateTestPendingTransactions(1, 1, chainID1)...)

	// Mock the both clients to be available
	state.txStatusFetcher.EXPECT().FetchTxStatus(gomock.Any(), common.ChainID(chainID0), gomock.Any()).DoAndReturn(
		func(ctx context.Context, chainID common.ChainID, hashes []eth.Hash) ([]pendingtxtracker.TxStatusRes, error) {
			require.Equal(t, common.ChainID(chainID0), chainID)
			require.Equal(t, 1, len(hashes))
			require.Contains(t, hashes, txs[0].Hash)
			return []pendingtxtracker.TxStatusRes{
				{Status: ac.Success, Hash: txs[0].Hash},
			}, nil
		}).Times(1)

	state.txStatusFetcher.EXPECT().FetchTxStatus(gomock.Any(), common.ChainID(chainID1), gomock.Any()).DoAndReturn(
		func(ctx context.Context, chainID common.ChainID, hashes []eth.Hash) ([]pendingtxtracker.TxStatusRes, error) {
			require.Equal(t, common.ChainID(chainID1), chainID)
			require.Equal(t, 1, len(hashes))
			require.Contains(t, hashes, txs[1].Hash)
			return []pendingtxtracker.TxStatusRes{
				{Status: ac.Success, Hash: txs[1].Hash},
			}, nil
		}).Times(1)

	eventChan := make(chan walletevent.Event, 10)
	sub := state.eventFeed.Subscribe(eventChan)

	err := state.pendingTxTracker.StoreAndTrackPendingTxs(txs)
	require.NoError(t, err)

	tx0Updates := make([]pendingtxtracker.PendingTxUpdatePayload, 0, 2)
	tx1Updates := make([]pendingtxtracker.PendingTxUpdatePayload, 0, 2)
	tx0StatusChanges := make([]pendingtxtracker.StatusChangedPayload, 0, 2)
	tx1StatusChanges := make([]pendingtxtracker.StatusChangedPayload, 0, 2)

	require.Eventually(t, func() bool {
		select {
		case we := <-eventChan:
			switch we.Type {
			case pendingtxtracker.EventPendingTransactionUpdate:
				updatePayload := unpackUpdateEvent(t, we)
				if updatePayload.TxIdentity == txs[0].Identity() {
					tx0Updates = append(tx0Updates, updatePayload)
				} else if updatePayload.TxIdentity == txs[1].Identity() {
					tx1Updates = append(tx1Updates, updatePayload)
				} else {
					t.Fatal("unexpected tx identity", updatePayload.TxIdentity)
				}
			case pendingtxtracker.EventPendingTransactionStatusChanged:
				statusChangedPayload := unpackStatusChangedEvent(t, we)
				if statusChangedPayload.TxIdentity == txs[0].Identity() {
					tx0StatusChanges = append(tx0StatusChanges, statusChangedPayload)
				} else if statusChangedPayload.TxIdentity == txs[1].Identity() {
					tx1StatusChanges = append(tx1StatusChanges, statusChangedPayload)
				} else {
					t.Fatal("unexpected tx identity", statusChangedPayload.TxIdentity)
				}
			}
		default:
			break
		}

		// We should have 2 updates for each tx (added and removed).
		// We should have 1 status change for each tx (success).
		return len(tx0Updates) == 2 && len(tx1Updates) == 2 && len(tx0StatusChanges) == 1 && len(tx1StatusChanges) == 1
	}, 2*time.Second, 10*time.Millisecond)

	require.Equal(t, ac.Success, tx0StatusChanges[0].Status)
	require.Equal(t, ac.Success, tx1StatusChanges[0].Status)

	err = state.pendingTxTracker.Stop()
	require.NoError(t, err)

	waitForTaskToStop(state.pendingTxTracker)

	res, err := state.pendingTxTracker.GetAllPending()
	require.NoError(t, err)
	require.Equal(t, 0, len(res))

	sub.Unsubscribe()
}
