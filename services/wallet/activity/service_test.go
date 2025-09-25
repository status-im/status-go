package activity

import (
	"context"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/services/wallet/pendingtxtracker"
	mock_pendingtxtracker "github.com/status-im/status-go/services/wallet/pendingtxtracker/mock"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	mock_token "github.com/status-im/status-go/services/wallet/token/mock/token"
	tokentypes "github.com/status-im/status-go/services/wallet/token/types"
	"github.com/status-im/status-go/services/wallet/walletevent"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const shouldNotWaitTimeout = 19999 * time.Second

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

func (m *mockCollectiblesManager) FetchCollectionSocialsAsync(contractID thirdparty.ContractID) error {
	args := m.Called(contractID)
	res := args.Get(0)
	if res == nil {
		return args.Error(1)
	}
	return nil
}

type testState struct {
	service          *Service
	eventFeed        *event.Feed
	tokenMock        *mock_token.MockManagerInterface
	collectiblesMock *mockCollectiblesManager
	close            func()
	pendingTracker   *pendingtxtracker.PendingTxTracker
	txStatusFetcher  *mock_pendingtxtracker.MockTxStatusFetcher
}

func setupTestService(tb testing.TB) (state testState) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(tb, err)

	appDB, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(tb, err)
	accountsDB, err := accounts.NewDB(appDB)
	require.NoError(tb, err)

	state.eventFeed = new(event.Feed)
	mockCtrl := gomock.NewController(tb)
	state.tokenMock = mock_token.NewMockManagerInterface(mockCtrl)
	state.collectiblesMock = &mockCollectiblesManager{}

	state.txStatusFetcher = mock_pendingtxtracker.NewMockTxStatusFetcher(mockCtrl)
	// Ensure we process pending transactions as needed, only once
	pendingCheckInterval := time.Second
	state.pendingTracker = pendingtxtracker.NewPendingTxTracker(db, state.txStatusFetcher, state.eventFeed, pendingCheckInterval)

	state.service = NewService(db, accountsDB, state.tokenMock, state.collectiblesMock, state.eventFeed)
	state.service.debounceDuration = 0
	state.close = func() {
		require.NoError(tb, db.Close())
		defer mockCtrl.Finish()
	}

	return state
}

func setupTransactions(t *testing.T, state testState, txCount int, testTxs []pendingtxtracker.TestTxSummary) (allAddresses []eth.Address, pendings []*pendingtxtracker.PendingTransaction, ch chan walletevent.Event, cleanup func()) {
	ch = make(chan walletevent.Event, 4)
	sub := state.eventFeed.Subscribe(ch)

	pendings = pendingtxtracker.GenerateTestPendingTransactionsWithSummary(testTxs, 5)
	for _, p := range pendings {
		allAddresses = append(allAddresses, p.From, p.To)
	}

	state.tokenMock.EXPECT().GetTokenByChainAddress(gomock.Any(), gomock.Any()).Return(
		&tokentypes.Token{
			Token: &types.Token{
				ChainID: 5,
				Address: eth.Address{},
				Symbol:  "ETH",
			},
		}, nil,
	).AnyTimes()

	return allAddresses, pendings, ch, func() {
		sub.Unsubscribe()
	}
}

func getValidateSessionUpdateHasNewOnTopFn(t *testing.T) func(payload SessionUpdate) bool {
	return func(payload SessionUpdate) bool {
		require.NotNil(t, payload.HasNewOnTop)
		require.True(t, *payload.HasNewOnTop)
		return false
	}
}

// validateSessionUpdateEvent expects will give up early if checkPayloadFn return true and not wait for expectCount
func validateSessionUpdateEvent(t *testing.T, ch chan walletevent.Event, filterResponseCount *int, expectCount int, checkPayloadFn func(payload SessionUpdate) bool) (pendingTransactionUpdate, sessionUpdatesCount int) {
	for sessionUpdatesCount < expectCount {
		select {
		case res := <-ch:
			switch res.Type {
			case pendingtxtracker.EventPendingTransactionUpdate:
				pendingTransactionUpdate++
			case EventActivitySessionUpdated:
				payload, err := walletevent.GetPayload[SessionUpdate](res)
				require.NoError(t, err)

				if checkPayloadFn != nil && checkPayloadFn(*payload) {
					return
				}

				sessionUpdatesCount++
			case EventActivityFilteringDone:
				(*filterResponseCount)++
			}
		case <-time.NewTimer(shouldNotWaitTimeout).C:
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
				payload, err := walletevent.GetPayload[FilterResponse](res)
				require.NoError(t, err)

				expectOffset, expectErrorCode := getOptionalExpectations(extra)

				require.Equal(t, expectErrorCode, payload.ErrorCode)
				require.Equal(t, resCount, len(payload.Activities))

				require.Equal(t, expectOffset, payload.Offset)
				filterResponseCount++

				if checkPayloadFn != nil {
					checkPayloadFn(*payload)
				}
			}
		case <-time.NewTimer(shouldNotWaitTimeout).C:
			require.Fail(t, "timeout while waiting for EventActivityFilteringDone")
		}
	}
	return
}
