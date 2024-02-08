package activity

import (
	"context"
	"database/sql"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/services/wallet/walletevent"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/transactions"
	"github.com/status-im/status-go/walletdatabase"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockCollectiblesManager implements the collectibles.ManagerInterface
type mockCollectiblesManager struct {
	mock.Mock
}

func (m *mockCollectiblesManager) FetchAssetsByCollectibleUniqueID(ctx context.Context, uniqueIDs []thirdparty.CollectibleUniqueID, asyncFetch bool) ([]thirdparty.FullCollectibleData, error) {
	args := m.Called(uniqueIDs)
	res := args.Get(0)
	if res == nil {
		return nil, args.Error(1)
	}
	return res.([]thirdparty.FullCollectibleData), args.Error(1)
}

// mockTokenManager implements the token.ManagerInterface
type mockTokenManager struct {
	mock.Mock
}

func (m *mockTokenManager) LookupTokenIdentity(chainID uint64, address eth.Address, native bool) *token.Token {
	args := m.Called(chainID, address, native)
	res := args.Get(0)
	if res == nil {
		return nil
	}
	return res.(*token.Token)
}

func (m *mockTokenManager) LookupToken(chainID *uint64, tokenSymbol string) (tkn *token.Token, isNative bool) {
	args := m.Called(chainID, tokenSymbol)
	return args.Get(0).(*token.Token), args.Bool(1)
}

type testState struct {
	service          *Service
	eventFeed        *event.Feed
	tokenMock        *mockTokenManager
	collectiblesMock *mockCollectiblesManager
	close            func()
	pendingTracker   *transactions.PendingTxTracker
	chainClient      *transactions.MockChainClient
}

func setupTestService(tb testing.TB) (state testState) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(tb, err)

	state.eventFeed = new(event.Feed)
	state.tokenMock = &mockTokenManager{}
	state.collectiblesMock = &mockCollectiblesManager{}

	state.chainClient = transactions.NewMockChainClient()

	// Ensure we process pending transactions as needed, only once
	pendingCheckInterval := time.Second
	state.pendingTracker = transactions.NewPendingTxTracker(db, state.chainClient, nil, state.eventFeed, pendingCheckInterval)

	state.service = NewService(db, state.tokenMock, state.collectiblesMock, state.eventFeed, state.pendingTracker)
	state.close = func() {
		require.NoError(tb, state.pendingTracker.Stop())
		require.NoError(tb, db.Close())
	}

	return state
}

type arg struct {
	chainID         common.ChainID
	tokenAddressStr string
	tokenIDStr      string
	tokenID         *big.Int
	tokenAddress    *eth.Address
}

// insertStubTransfersWithCollectibles will insert nil if tokenIDStr is empty
func insertStubTransfersWithCollectibles(t *testing.T, db *sql.DB, args []arg) (fromAddresses, toAddresses []eth.Address) {
	trs, fromAddresses, toAddresses := transfer.GenerateTestTransfers(t, db, 0, len(args))
	for i := range args {
		trs[i].ChainID = args[i].chainID
		if args[i].tokenIDStr == "" {
			args[i].tokenID = nil
		} else {
			args[i].tokenID = new(big.Int)
			args[i].tokenID.SetString(args[i].tokenIDStr, 0)
		}
		args[i].tokenAddress = new(eth.Address)
		*args[i].tokenAddress = eth.HexToAddress(args[i].tokenAddressStr)
		transfer.InsertTestTransferWithOptions(t, db, trs[i].To, &trs[i], &transfer.TestTransferOptions{
			TokenAddress: *args[i].tokenAddress,
			TokenID:      args[i].tokenID,
		})
	}
	return fromAddresses, toAddresses
}

func TestService_UpdateCollectibleInfo(t *testing.T) {
	state := setupTestService(t)
	defer state.close()

	args := []arg{
		{5, "0xA2838FDA19EB6EED3F8B9EFF411D4CD7D2DE0313", "0x0D", nil, nil},
		{5, "0xA2838FDA19EB6EED3F8B9EFF411D4CD7D2DE0313", "0x762AD3E4934E687F8701F24C7274E5209213FD6208FF952ACEB325D028866949", nil, nil},
		{5, "0x3d6afaa395c31fcd391fe3d562e75fe9e8ec7e6a", "", nil, nil},
		{5, "0xA2838FDA19EB6EED3F8B9EFF411D4CD7D2DE0313", "0x0F", nil, nil},
	}
	fromAddresses, toAddresses := insertStubTransfersWithCollectibles(t, state.service.db, args)

	ch := make(chan walletevent.Event)
	sub := state.eventFeed.Subscribe(ch)

	// Expect one call for the fungible token
	state.tokenMock.On("LookupTokenIdentity", uint64(5), eth.HexToAddress("0x3d6afaa395c31fcd391fe3d562e75fe9e8ec7e6a"), false).Return(
		&token.Token{
			ChainID: 5,
			Address: eth.HexToAddress("0x3d6afaa395c31fcd391fe3d562e75fe9e8ec7e6a"),
			Symbol:  "STT",
		}, false,
	).Once()
	state.collectiblesMock.On("FetchAssetsByCollectibleUniqueID", []thirdparty.CollectibleUniqueID{
		{
			ContractID: thirdparty.ContractID{
				ChainID: args[3].chainID,
				Address: *args[3].tokenAddress},
			TokenID: &bigint.BigInt{Int: args[3].tokenID},
		}, {
			ContractID: thirdparty.ContractID{
				ChainID: args[1].chainID,
				Address: *args[1].tokenAddress},
			TokenID: &bigint.BigInt{Int: args[1].tokenID},
		},
	}).Return([]thirdparty.FullCollectibleData{
		{
			CollectibleData: thirdparty.CollectibleData{
				Name:     "Test 2",
				ImageURL: "test://url/2"},
			CollectionData: nil,
		}, {
			CollectibleData: thirdparty.CollectibleData{
				Name:     "Test 1",
				ImageURL: "test://url/1"},
			CollectionData: nil,
		},
	}, nil).Once()

	state.service.FilterActivityAsync(0, append(fromAddresses, toAddresses...), true, allNetworksFilter(), Filter{}, 0, 3)

	filterResponseCount := 0
	var updates []EntryData

	for i := 0; i < 2; i++ {
		select {
		case res := <-ch:
			switch res.Type {
			case EventActivityFilteringDone:
				var payload FilterResponse
				err := json.Unmarshal([]byte(res.Message), &payload)
				require.NoError(t, err)
				require.Equal(t, ErrorCodeSuccess, payload.ErrorCode)
				require.Equal(t, 3, len(payload.Activities))
				filterResponseCount++
			case EventActivityFilteringUpdate:
				err := json.Unmarshal([]byte(res.Message), &updates)
				require.NoError(t, err)
			}
		case <-time.NewTimer(1 * time.Second).C:
			require.Fail(t, "timeout while waiting for event")
		}
	}

	require.Equal(t, 1, filterResponseCount)
	require.Equal(t, 2, len(updates))
	require.Equal(t, "Test 2", *updates[0].NftName)
	require.Equal(t, "test://url/2", *updates[0].NftURL)
	require.Equal(t, "Test 1", *updates[1].NftName)
	require.Equal(t, "test://url/1", *updates[1].NftURL)

	sub.Unsubscribe()
}

func TestService_UpdateCollectibleInfo_Error(t *testing.T) {
	state := setupTestService(t)
	defer state.close()

	args := []arg{
		{5, "0xA2838FDA19EB6EED3F8B9EFF411D4CD7D2DE0313", "0x762AD3E4934E687F8701F24C7274E5209213FD6208FF952ACEB325D028866949", nil, nil},
		{5, "0xA2838FDA19EB6EED3F8B9EFF411D4CD7D2DE0313", "0x0D", nil, nil},
	}

	ch := make(chan walletevent.Event, 4)
	sub := state.eventFeed.Subscribe(ch)

	fromAddresses, toAddresses := insertStubTransfersWithCollectibles(t, state.service.db, args)

	state.collectiblesMock.On("FetchAssetsByCollectibleUniqueID", mock.Anything).Return(nil, thirdparty.ErrChainIDNotSupported).Once()

	state.service.FilterActivityAsync(0, append(fromAddresses, toAddresses...), true, allNetworksFilter(), Filter{}, 0, 5)

	filterResponseCount := 0
	updatesCount := 0

	for i := 0; i < 2; i++ {
		select {
		case res := <-ch:
			switch res.Type {
			case EventActivityFilteringDone:
				var payload FilterResponse
				err := json.Unmarshal([]byte(res.Message), &payload)
				require.NoError(t, err)
				require.Equal(t, ErrorCodeSuccess, payload.ErrorCode)
				require.Equal(t, 2, len(payload.Activities))
				filterResponseCount++
			case EventActivityFilteringUpdate:
				updatesCount++
			}
		case <-time.NewTimer(20 * time.Millisecond).C:
			// We wait to ensure the EventActivityFilteringUpdate is never sent
		}
	}

	require.Equal(t, 1, filterResponseCount)
	require.Equal(t, 0, updatesCount)

	sub.Unsubscribe()
}

func setupTransactions(t *testing.T, state testState, txCount int, testTxs []transactions.TestTxSummary) (allAddresses []eth.Address, pendings []transactions.PendingTransaction, ch chan walletevent.Event, cleanup func()) {
	ch = make(chan walletevent.Event, 4)
	sub := state.eventFeed.Subscribe(ch)

	pendings = transactions.MockTestTransactions(t, state.chainClient, testTxs)

	txs, fromTrs, toTrs := transfer.GenerateTestTransfers(t, state.service.db, len(pendings), txCount)
	for i := range txs {
		transfer.InsertTestTransfer(t, state.service.db, txs[i].To, &txs[i])
	}

	allAddresses = append(append(fromTrs, toTrs...), pendings[0].From, pendings[0].To)
	state.tokenMock.On("LookupTokenIdentity", mock.Anything, mock.Anything, mock.Anything).Return(
		&token.Token{
			ChainID: 5,
			Address: eth.Address{},
			Symbol:  "ETH",
		}, true,
	).Times(0)

	state.tokenMock.On("LookupToken", mock.Anything, mock.Anything).Return(
		&token.Token{
			ChainID: 5,
			Address: eth.Address{},
			Symbol:  "ETH",
		}, true,
	).Times(0)

	return allAddresses, pendings, ch, func() {
		sub.Unsubscribe()
	}
}

func validateSessionUpdateEvent(t *testing.T, ch chan walletevent.Event, filterResponseCount *int) (pendingTransactionUpdate, sessionUpdatesCount int) {
	for sessionUpdatesCount < 1 {
		select {
		case res := <-ch:
			switch res.Type {
			case transactions.EventPendingTransactionUpdate:
				pendingTransactionUpdate++
			case EventActivitySessionUpdated:
				var payload SessionUpdate
				err := json.Unmarshal([]byte(res.Message), &payload)
				require.NoError(t, err)
				require.NotNil(t, payload.HasNewEntries)
				require.True(t, *payload.HasNewEntries)
				sessionUpdatesCount++
			case EventActivityFilteringDone:
				(*filterResponseCount)++
			}
		case <-time.NewTimer(1 * time.Second).C:
			require.Fail(t, "timeout while waiting for EventActivitySessionUpdated")
		}
	}
	return
}

type extraExpect struct {
	offset    *int
	errorCode *ErrorCode
}

func getOptionalExpectations(e *extraExpect) (expectOffset int, expectErrorCode ErrorCode) {
	expectOffset = 0
	expectErrorCode = ErrorCodeSuccess

	if e != nil {
		if e.offset != nil {
			expectOffset = *e.offset
		}
		if e.errorCode != nil {
			expectErrorCode = *e.errorCode
		}
	}
	return
}

func validateFilteringDone(t *testing.T, ch chan walletevent.Event, resCount int, checkPayloadFn func(payload FilterResponse), extra *extraExpect) (filterResponseCount int) {
	for filterResponseCount < 1 {
		select {
		case res := <-ch:
			switch res.Type {
			case EventActivityFilteringDone:
				var payload FilterResponse
				err := json.Unmarshal([]byte(res.Message), &payload)
				require.NoError(t, err)

				expectOffset, expectErrorCode := getOptionalExpectations(extra)

				require.Equal(t, expectErrorCode, payload.ErrorCode)
				require.Equal(t, resCount, len(payload.Activities))

				require.Equal(t, expectOffset, payload.Offset)
				filterResponseCount++

				if checkPayloadFn != nil {
					checkPayloadFn(payload)
				}
			}
		case <-time.NewTimer(1 * time.Second).C:
			require.Fail(t, "timeout while waiting for EventActivityFilteringDone")
		}
	}
	return
}

func TestService_IncrementalUpdateOnTop(t *testing.T) {
	state := setupTestService(t)
	defer state.close()

	transactionCount := 2
	allAddresses, pendings, ch, cleanup := setupTransactions(t, state, transactionCount, []transactions.TestTxSummary{{DontConfirm: true, Timestamp: transactionCount + 1}})
	defer cleanup()

	sessionID := state.service.StartFilterSession(allAddresses, true, allNetworksFilter(), Filter{}, 5)
	require.Greater(t, sessionID, SessionID(0))
	defer state.service.StopFilterSession(sessionID)

	filterResponseCount := validateFilteringDone(t, ch, 2, nil, nil)

	exp := pendings[0]
	err := state.pendingTracker.StoreAndTrackPendingTx(&exp)
	require.NoError(t, err)

	pendingTransactionUpdate, sessionUpdatesCount := validateSessionUpdateEvent(t, ch, &filterResponseCount)

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
		require.Equal(t, (*int)(nil), newTx.transferType)
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

func TestService_IncrementalUpdateFetchWindow(t *testing.T) {
	state := setupTestService(t)
	defer state.close()

	transactionCount := 5
	allAddresses, pendings, ch, cleanup := setupTransactions(t, state, transactionCount, []transactions.TestTxSummary{{DontConfirm: true, Timestamp: transactionCount + 1}})
	defer cleanup()

	sessionID := state.service.StartFilterSession(allAddresses, true, allNetworksFilter(), Filter{}, 2)
	require.Greater(t, sessionID, SessionID(0))
	defer state.service.StopFilterSession(sessionID)

	filterResponseCount := validateFilteringDone(t, ch, 2, nil, nil)

	exp := pendings[0]
	err := state.pendingTracker.StoreAndTrackPendingTx(&exp)
	require.NoError(t, err)

	pendingTransactionUpdate, sessionUpdatesCount := validateSessionUpdateEvent(t, ch, &filterResponseCount)

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

	sessionID := state.service.StartFilterSession(allAddresses, true, allNetworksFilter(), Filter{}, 2)
	require.Greater(t, sessionID, SessionID(0))
	defer state.service.StopFilterSession(sessionID)

	filterResponseCount := validateFilteringDone(t, ch, 2, func(payload FilterResponse) {
		require.Equal(t, int64(transactionCount), payload.Activities[0].timestamp)
		require.Equal(t, int64(transactionCount-1), payload.Activities[1].timestamp)
	}, nil)

	exp := pendings[0]
	err := state.pendingTracker.StoreAndTrackPendingTx(&exp)
	require.NoError(t, err)

	pendingTransactionUpdate, sessionUpdatesCount := validateSessionUpdateEvent(t, ch, &filterResponseCount)
	require.Equal(t, 1, pendingTransactionUpdate)
	require.Equal(t, 1, filterResponseCount)
	require.Equal(t, 1, sessionUpdatesCount)

	err = state.service.GetMoreForFilterSession(sessionID, 2)
	require.NoError(t, err)

	// Validate that client doesn't anything of the internal state
	eventActivityDoneCount := validateFilteringDone(t, ch, 2, func(payload FilterResponse) {
		require.False(t, payload.Activities[0].isNew)
		require.Equal(t, int64(transactionCount-2), payload.Activities[0].timestamp)
		require.False(t, payload.Activities[1].isNew)
		require.Equal(t, int64(transactionCount-3), payload.Activities[1].timestamp)
	}, common.NewAndSet(extraExpect{common.NewAndSet(2), nil}))
	require.Equal(t, 1, eventActivityDoneCount)
}
