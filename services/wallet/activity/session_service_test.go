package activity

import (
	"math/big"
	"sync"
	"testing"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/transactions"

	"github.com/stretchr/testify/require"
)

func TestService_IncrementalUpdateOnTop(t *testing.T) {
	state := setupTestService(t)
	defer state.close()

	transactionCount := 2
	allAddresses, pendings, ch, cleanup := setupTransactions(t, state, transactionCount, []transactions.TestTxSummary{{DontConfirm: true, Timestamp: transactionCount + 1}})
	defer cleanup()

	sessionID := state.service.StartFilterSession(allAddresses, allNetworksFilter(), Filter{}, 5, V1)
	require.Greater(t, sessionID, SessionID(0))
	defer state.service.StopFilterSession(sessionID)

	filterResponseCount := validateFilteringDone(t, ch, 2, nil, nil)

	exp := pendings[0]
	err := state.pendingTracker.StoreAndTrackPendingTx(&exp)
	require.NoError(t, err)

	vFn := getValidateSessionUpdateHasNewOnTopFn(t)
	pendingTransactionUpdate, sessionUpdatesCount := validateSessionUpdateEvent(t, ch, &filterResponseCount, 1, vFn)

	err = state.service.ResetFilterSession(sessionID, 5)
	require.NoError(t, err)

	// Validate the reset data
	eventActivityDoneCount := validateFilteringDone(t, ch, 3, func(payload FilterResponse) {
		require.True(t, payload.Activities[0].isNew)
		require.False(t, payload.Activities[1].isNew)
		require.False(t, payload.Activities[2].isNew)

		// Check the new transaction data
		newTx := payload.Activities[0]
		require.Equal(t, PendingTransactionPT, newTx.payloadType)
		// We don't keep type in the DB
		require.Equal(t, (*int64)(nil), newTx.transferType)
		require.Equal(t, SendAT, newTx.activityType)
		require.Equal(t, PendingAS, newTx.activityStatus)
		require.Equal(t, exp.ChainID, newTx.transaction.ChainID)
		require.Equal(t, exp.ChainID, *newTx.chainIDOut)
		require.Equal(t, (*common.ChainID)(nil), newTx.chainIDIn)
		require.Equal(t, exp.Hash, newTx.transaction.Hash)
		// Pending doesn't have address as part of identity
		require.Equal(t, eth.Address{}, newTx.transaction.Address)
		require.Equal(t, exp.From, *newTx.sender)
		require.Equal(t, exp.To, *newTx.recipient)
		require.Equal(t, 0, exp.Value.Int.Cmp((*big.Int)(newTx.amountOut)))
		require.Equal(t, exp.Timestamp, uint64(newTx.timestamp))
		require.Equal(t, exp.Symbol, *newTx.symbolOut)
		require.Equal(t, (*string)(nil), newTx.symbolIn)
		require.Equal(t, &Token{
			TokenType: Native,
			ChainID:   5,
		}, newTx.tokenOut)
		require.Equal(t, (*Token)(nil), newTx.tokenIn)
		require.Equal(t, (*eth.Address)(nil), newTx.contractAddress)

		// Check the order of the following transaction data
		require.Equal(t, SimpleTransactionPT, payload.Activities[1].payloadType)
		require.Equal(t, int64(transactionCount), payload.Activities[1].timestamp)
		require.Equal(t, SimpleTransactionPT, payload.Activities[2].payloadType)
		require.Equal(t, int64(transactionCount-1), payload.Activities[2].timestamp)
	}, nil)

	require.Equal(t, 1, pendingTransactionUpdate)
	require.Equal(t, 1, filterResponseCount)
	require.Equal(t, 1, sessionUpdatesCount)
	require.Equal(t, 1, eventActivityDoneCount)
}

func TestService_IncrementalUpdateMixed(t *testing.T) {
	state := setupTestService(t)
	defer state.close()

	transactionCount := 5
	allAddresses, pendings, ch, cleanup := setupTransactions(t, state, transactionCount,
		[]transactions.TestTxSummary{
			{DontConfirm: true, Timestamp: 2},
			{DontConfirm: true, Timestamp: 4},
			{DontConfirm: true, Timestamp: 6},
		},
	)
	defer cleanup()

	sessionID := state.service.StartFilterSession(allAddresses, allNetworksFilter(), Filter{}, 5, V1)
	require.Greater(t, sessionID, SessionID(0))
	defer state.service.StopFilterSession(sessionID)

	filterResponseCount := validateFilteringDone(t, ch, 5, nil, nil)

	for i := range pendings {
		err := state.pendingTracker.StoreAndTrackPendingTx(&pendings[i])
		require.NoError(t, err)
	}

	pendingTransactionUpdate, sessionUpdatesCount := validateSessionUpdateEvent(t, ch, &filterResponseCount, 2, func(payload SessionUpdate) bool {
		require.Nil(t, payload.HasNewOnTop)
		require.NotEmpty(t, payload.New)
		for _, update := range payload.New {
			require.True(t, update.Entry.isNew)
			foundIdx := -1
			for i, pTx := range pendings {
				if pTx.Hash == update.Entry.transaction.Hash && pTx.ChainID == update.Entry.transaction.ChainID {
					foundIdx = i
					break
				}
			}
			require.Greater(t, foundIdx, -1, "the updated transaction should be found in the pending list")
			pendings = append(pendings[:foundIdx], pendings[foundIdx+1:]...)
		}
		return len(pendings) == 1
	})

	// Validate that the last one (oldest) is out of the window
	require.Equal(t, 1, len(pendings))
	require.Equal(t, uint64(2), pendings[0].Timestamp)

	require.Equal(t, 3, pendingTransactionUpdate)
	require.LessOrEqual(t, sessionUpdatesCount, 3)
	require.Equal(t, 1, filterResponseCount)

}

func TestService_IncrementalUpdateFetchWindow(t *testing.T) {
	state := setupTestService(t)
	defer state.close()

	transactionCount := 5
	allAddresses, pendings, ch, cleanup := setupTransactions(t, state, transactionCount, []transactions.TestTxSummary{{DontConfirm: true, Timestamp: transactionCount + 1}})
	defer cleanup()

	sessionID := state.service.StartFilterSession(allAddresses, allNetworksFilter(), Filter{}, 2, V1)
	require.Greater(t, sessionID, SessionID(0))
	defer state.service.StopFilterSession(sessionID)

	filterResponseCount := validateFilteringDone(t, ch, 2, nil, nil)

	exp := pendings[0]
	err := state.pendingTracker.StoreAndTrackPendingTx(&exp)
	require.NoError(t, err)

	vFn := getValidateSessionUpdateHasNewOnTopFn(t)
	pendingTransactionUpdate, sessionUpdatesCount := validateSessionUpdateEvent(t, ch, &filterResponseCount, 1, vFn)

	err = state.service.ResetFilterSession(sessionID, 2)
	require.NoError(t, err)

	// Validate the reset data
	eventActivityDoneCount := validateFilteringDone(t, ch, 2, func(payload FilterResponse) {
		require.True(t, payload.Activities[0].isNew)
		require.Equal(t, int64(transactionCount+1), payload.Activities[0].timestamp)
		require.False(t, payload.Activities[1].isNew)
		require.Equal(t, int64(transactionCount), payload.Activities[1].timestamp)
	}, nil)

	require.Equal(t, 1, pendingTransactionUpdate)
	require.Equal(t, 1, filterResponseCount)
	require.Equal(t, 1, sessionUpdatesCount)
	require.Equal(t, 1, eventActivityDoneCount)

	err = state.service.GetMoreForFilterSession(sessionID, 2)
	require.NoError(t, err)

	eventActivityDoneCount = validateFilteringDone(t, ch, 2, func(payload FilterResponse) {
		require.False(t, payload.Activities[0].isNew)
		require.Equal(t, int64(transactionCount-1), payload.Activities[0].timestamp)
		require.False(t, payload.Activities[1].isNew)
		require.Equal(t, int64(transactionCount-2), payload.Activities[1].timestamp)
	}, common.NewAndSet(extraExpect{common.NewAndSet(2), nil}))
	require.Equal(t, 1, eventActivityDoneCount)
}

func TestService_IncrementalUpdateFetchWindowNoReset(t *testing.T) {
	state := setupTestService(t)
	defer state.close()

	transactionCount := 5
	allAddresses, pendings, ch, cleanup := setupTransactions(t, state, transactionCount, []transactions.TestTxSummary{{DontConfirm: true, Timestamp: transactionCount + 1}})
	defer cleanup()

	sessionID := state.service.StartFilterSession(allAddresses, allNetworksFilter(), Filter{}, 2, V1)
	require.Greater(t, sessionID, SessionID(0))
	defer state.service.StopFilterSession(sessionID)

	filterResponseCount := validateFilteringDone(t, ch, 2, func(payload FilterResponse) {
		require.Equal(t, int64(transactionCount), payload.Activities[0].timestamp)
		require.Equal(t, int64(transactionCount-1), payload.Activities[1].timestamp)
	}, nil)

	exp := pendings[0]
	err := state.pendingTracker.StoreAndTrackPendingTx(&exp)
	require.NoError(t, err)

	vFn := getValidateSessionUpdateHasNewOnTopFn(t)
	pendingTransactionUpdate, sessionUpdatesCount := validateSessionUpdateEvent(t, ch, &filterResponseCount, 1, vFn)
	require.Equal(t, 1, pendingTransactionUpdate)
	require.Equal(t, 1, filterResponseCount)
	require.Equal(t, 1, sessionUpdatesCount)

	err = state.service.GetMoreForFilterSession(sessionID, 2)
	require.NoError(t, err)

	// Validate that client continue loading the next window without being affected by the internal state of new
	eventActivityDoneCount := validateFilteringDone(t, ch, 2, func(payload FilterResponse) {
		require.False(t, payload.Activities[0].isNew)
		require.Equal(t, int64(transactionCount-2), payload.Activities[0].timestamp)
		require.False(t, payload.Activities[1].isNew)
		require.Equal(t, int64(transactionCount-3), payload.Activities[1].timestamp)
	}, common.NewAndSet(extraExpect{common.NewAndSet(2), nil}))
	require.Equal(t, 1, eventActivityDoneCount)
}

// Simulate and validate a multi-step user flow that was also a regression in the original implementation
func TestService_FilteredIncrementalUpdateResetAndClear(t *testing.T) {
	state := setupTestService(t)
	defer state.close()

	transactionCount := 5
	allAddresses, pendings, ch, cleanup := setupTransactions(t, state, transactionCount, []transactions.TestTxSummary{{DontConfirm: true, Timestamp: transactionCount + 1}})
	defer cleanup()

	// Generate new transaction for step 5
	newOffset := transactionCount + 2
	newTxs, newFromTrs, newToTrs := transfer.GenerateTestTransfers(t, state.service.db, newOffset, 1)
	allAddresses = append(append(allAddresses, newFromTrs...), newToTrs...)

	// 1. User visualizes transactions for the first time
	sessionID := state.service.StartFilterSession(allAddresses, allNetworksFilter(), Filter{}, 4, V1)
	require.Greater(t, sessionID, SessionID(0))
	defer state.service.StopFilterSession(sessionID)

	validateFilteringDone(t, ch, 4, nil, nil)

	// 2. User applies a filter for pending transactions
	err := state.service.UpdateFilterForSession(sessionID, Filter{Statuses: []Status{PendingAS}}, 4)
	require.NoError(t, err)

	filterResponseCount := validateFilteringDone(t, ch, 0, nil, nil)

	// 3. A pending transaction is added
	exp := pendings[0]
	err = state.pendingTracker.StoreAndTrackPendingTx(&exp)
	require.NoError(t, err)

	vFn := getValidateSessionUpdateHasNewOnTopFn(t)
	pendingTransactionUpdate, sessionUpdatesCount := validateSessionUpdateEvent(t, ch, &filterResponseCount, 1, vFn)

	// 4. User resets the view and the new pending transaction has the new flag
	err = state.service.ResetFilterSession(sessionID, 2)
	require.NoError(t, err)

	// Validate the reset data
	eventActivityDoneCount := validateFilteringDone(t, ch, 1, func(payload FilterResponse) {
		require.True(t, payload.Activities[0].isNew)
		require.Equal(t, int64(transactionCount+1), payload.Activities[0].timestamp)
	}, nil)

	require.Equal(t, 1, pendingTransactionUpdate)
	require.Equal(t, 1, filterResponseCount)
	require.Equal(t, 1, sessionUpdatesCount)
	require.Equal(t, 1, eventActivityDoneCount)

	// 5. A new transaction is downloaded
	transfer.InsertTestTransfer(t, state.service.db, newTxs[0].To, &newTxs[0])

	// 6. User clears the filter and only the new transaction should have the new flag
	err = state.service.UpdateFilterForSession(sessionID, Filter{}, 4)
	require.NoError(t, err)

	eventActivityDoneCount = validateFilteringDone(t, ch, 4, func(payload FilterResponse) {
		require.True(t, payload.Activities[0].isNew)
		require.Equal(t, int64(newOffset), payload.Activities[0].timestamp)
		require.False(t, payload.Activities[1].isNew)
		require.Equal(t, int64(newOffset-1), payload.Activities[1].timestamp)
		require.False(t, payload.Activities[2].isNew)
		require.Equal(t, int64(newOffset-2), payload.Activities[2].timestamp)
		require.False(t, payload.Activities[3].isNew)
		require.Equal(t, int64(newOffset-3), payload.Activities[3].timestamp)
	}, nil)
	require.Equal(t, 1, eventActivityDoneCount)
}

// Test the different session-related endpoints in a multi-threaded environment
func TestService_MultiThread(t *testing.T) {
	state := setupTestService(t)
	defer state.close()

	const n = 3
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			sessionID := state.service.StartFilterSession([]eth.Address{}, allNetworksFilter(), Filter{}, 5, V2)
			require.Greater(t, sessionID, SessionID(0))

			transactionCount := 5
			_, _, _, cleanup := setupTransactions(t, state, transactionCount, []transactions.TestTxSummary{{DontConfirm: true, Timestamp: transactionCount + 1}})
			defer cleanup()

			const m = 10
			var subwg sync.WaitGroup
			subwg.Add(m)
			for j := 0; j < m; j++ {
				go func() {
					defer subwg.Done()
					var suberr error

					suberr = state.service.UpdateFilterForSession(sessionID, Filter{}, 5)
					require.NoError(t, suberr)

					suberr = state.service.ResetFilterSession(sessionID, 5)
					require.NoError(t, suberr)

					suberr = state.service.GetMoreForFilterSession(sessionID, 5)
					require.NoError(t, suberr)
				}()
			}
			subwg.Wait()

			state.service.StopFilterSession(sessionID)
		}()
	}
	wg.Wait()
}
