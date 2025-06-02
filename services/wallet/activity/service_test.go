package activity

import (
	"context"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/multiaccounts/accounts"
	ethclient "github.com/status-im/status-go/rpc/chain/ethclient"
	mock_rpcclient "github.com/status-im/status-go/rpc/mock/client"
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	mock_token "github.com/status-im/status-go/services/wallet/token/mock/token"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/services/wallet/walletevent"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/transactions"
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
	pendingTracker   *transactions.PendingTxTracker
	chainClient      *transactions.MockChainClient
	rpcClient        *mock_rpcclient.MockClientInterface
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

	state.chainClient = transactions.NewMockChainClient()
	state.rpcClient = mock_rpcclient.NewMockClientInterface(mockCtrl)
	state.rpcClient.EXPECT().AbstractEthClient(gomock.Any()).DoAndReturn(func(chainID common.ChainID) (ethclient.BatchCallClient, error) {
		return state.chainClient.AbstractEthClient(chainID)
	}).AnyTimes()

	// Ensure we process pending transactions as needed, only once
	pendingCheckInterval := time.Second
	state.pendingTracker = transactions.NewPendingTxTracker(db, state.rpcClient, state.eventFeed, pendingCheckInterval)

	state.service = NewService(db, accountsDB, state.tokenMock, state.collectiblesMock, state.eventFeed)
	state.service.debounceDuration = 0
	state.close = func() {
		require.NoError(tb, state.pendingTracker.Stop())
		require.NoError(tb, db.Close())
		defer mockCtrl.Finish()
	}

	return state
}

func setupTransactions(t *testing.T, state testState, txCount int, testTxs []transactions.TestTxSummary) (allAddresses []eth.Address, pendings []transactions.PendingTransaction, ch chan walletevent.Event, cleanup func()) {
	ch = make(chan walletevent.Event, 4)
	sub := state.eventFeed.Subscribe(ch)

	pendings = transactions.MockTestTransactions(t, state.chainClient, testTxs)
	for _, p := range pendings {
		allAddresses = append(allAddresses, p.From, p.To)
	}

	txs, fromTrs, toTrs := transfer.GenerateTestTransfers(t, state.service.db, len(pendings), txCount)
	for i := range txs {
		transfer.InsertTestTransfer(t, state.service.db, txs[i].To, &txs[i])
	}

	allAddresses = append(append(allAddresses, fromTrs...), toTrs...)

	state.tokenMock.EXPECT().LookupTokenIdentity(gomock.Any(), gomock.Any(), gomock.Any()).Return(
		&tokenTypes.Token{
			ChainID: 5,
			Address: eth.Address{},
			Symbol:  "ETH",
		},
	).AnyTimes()

	state.tokenMock.EXPECT().LookupToken(gomock.Any(), gomock.Any()).Return(
		&tokenTypes.Token{
			ChainID: 5,
			Address: eth.Address{},
			Symbol:  "ETH",
		}, true,
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
			case transactions.EventPendingTransactionUpdate:
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
