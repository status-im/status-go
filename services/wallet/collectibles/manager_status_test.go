package collectibles

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	mock_client "github.com/status-im/status-go/internal/rpc/chain/mock/client"
	mock_rpcclient "github.com/status-im/status-go/internal/rpc/mock/client"
	mock_collectibles "github.com/status-im/status-go/services/wallet/collectibles/mock"
	mock_ownership "github.com/status-im/status-go/services/wallet/collectibles/ownership/mock"
	mock_community "github.com/status-im/status-go/services/wallet/community/mock"
	mock_thirdparty "github.com/status-im/status-go/services/wallet/thirdparty/mock"

	"github.com/status-im/status-go/internal/circuitbreaker"
	"github.com/status-im/status-go/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/connection"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

// TestCollectiblesStatus_SuccessfulFetchDoesNotRelyOnProviderIsConnected
// Repro: a successful provider round-trip must mark the chain as connected even when
// third-party health IsConnected() is false (async / out of sync with real calls).
// Old code defers checkConnectionStatus which only looks at IsConnected and overwrites
// a successful request with Disconnected.
func TestCollectiblesStatus_SuccessfulFetchDoesNotRelyOnProviderIsConnected(t *testing.T) {
	t.Parallel()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()
	chainID := walletCommon.ChainID(1)
	owner := common.HexToAddress("0x1234567890abcdef")
	providerID := "test_provider"

	chainClient := &CopyableMockChainClient{MockClientInterface: mock_client.NewMockClientInterface(mockCtrl)}
	chainClient.EXPECT().CallContract(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	chainClient.EXPECT().CodeAt(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	rpcClient := mock_rpcclient.NewMockClientInterface(mockCtrl)
	rpcClient.EXPECT().EthClient(gomock.Any()).Return(chainClient, nil).AnyTimes()

	mockProvider := mock_thirdparty.NewMockCollectibleAccountOwnershipProvider(mockCtrl)
	mockProvider.EXPECT().IsChainSupported(chainID).Return(true).AnyTimes()
	// Intentionally false: simulates desync with actual fetch success
	mockProvider.EXPECT().IsConnected().Return(false).AnyTimes()
	mockProvider.EXPECT().ID().Return(providerID).AnyTimes()

	mockAssetContainer := &thirdparty.FullCollectibleDataContainer{
		Items: []thirdparty.FullCollectibleData{
			{
				CollectibleData: thirdparty.CollectibleData{
					ID: thirdparty.CollectibleUniqueID{
						ContractID: thirdparty.ContractID{
							Address: common.HexToAddress("0x0000000000000000000000000000000000000001"),
						},
						TokenID: &bigint.BigInt{Int: big.NewInt(1)},
					},
				},
			},
		},
	}
	mockProvider.EXPECT().FetchAllAssetsByOwner(gomock.Any(), chainID, owner, "", 1).Return(mockAssetContainer, nil)

	mockProviders := thirdparty.CollectibleProviders{
		AccountOwnershipProviders: []thirdparty.CollectibleAccountOwnershipProvider{mockProvider},
	}

	manager := NewManager(nil, rpcClient, nil, mockProviders, nil, new(event.Feed))

	chainKey := chainID.String()
	statuses := &sync.Map{}
	statuses.Store(chainKey, connection.NewStatus())
	manager.statuses = statuses
	manager.statusNotifier = createStatusNotifier(statuses, manager.feed)

	collectiblesDataDB := mock_collectibles.NewMockCollectibleDataStorage(mockCtrl)
	collectiblesDataDB.EXPECT().SetData(gomock.Any(), gomock.Any()).Return(nil)
	collectiblesDataDB.EXPECT().GetData(gomock.Any()).DoAndReturn(func(ids []thirdparty.CollectibleUniqueID) (map[string]thirdparty.CollectibleData, error) {
		ret := make(map[string]thirdparty.CollectibleData)
		for _, id := range ids {
			ret[id.HashKey()] = thirdparty.CollectibleData{ID: id}
		}
		return ret, nil
	})
	collectiblesDataDB.EXPECT().GetCommunityInfo(gomock.Any()).Return(&thirdparty.CollectibleCommunityInfo{}, nil).AnyTimes()
	collectionsDataDB := mock_collectibles.NewMockCollectionDataStorage(mockCtrl)
	collectionsDataDB.EXPECT().SetData(gomock.Any(), gomock.Any()).Return(nil)
	collectionsDataDB.EXPECT().GetData(gomock.Any()).DoAndReturn(func(ids []thirdparty.ContractID) (map[string]thirdparty.CollectionData, error) {
		ret := make(map[string]thirdparty.CollectionData)
		for _, id := range ids {
			ret[id.HashKey()] = thirdparty.CollectionData{ID: id}
		}
		return ret, nil
	})
	communityManager := mock_community.NewMockCommunityManagerInterface(mockCtrl)
	communityManager.EXPECT().GetCommunityID(gomock.Any()).Return(providerID).AnyTimes()
	communityManager.EXPECT().FillCollectiblesMetadata(gomock.Any(), gomock.Any()).Return(true, nil).AnyTimes()
	communityManager.EXPECT().GetCommunityInfo(gomock.Any()).Return(&thirdparty.CommunityInfo{}, nil, nil).AnyTimes()
	ownershipDB := mock_ownership.NewMockOwnershipStorage(mockCtrl)
	ownershipDB.EXPECT().GetLatestOwnershipUpdateTimestamp(gomock.Any()).Return(int64(0), nil).AnyTimes()
	ownershipDB.EXPECT().GetOwnership(gomock.Any()).Return([]thirdparty.AccountBalance{}, nil).AnyTimes()
	manager.collectiblesDataDB = collectiblesDataDB
	manager.collectionsDataDB = collectionsDataDB
	manager.communityManager = communityManager
	manager.ownershipDB = ownershipDB

	_, err := manager.FetchAllAssetsByOwner(ctx, chainID, owner, "", 1, thirdparty.FetchFromAnyProvider)
	require.NoError(t, err)

	loaded, ok := manager.statuses.Load(chainKey)
	require.True(t, ok, "status for chain should exist")
	st := loaded.(*connection.Status)
	// After fix, successful functor must mark the chain as connected; old code only looked at
	// IsConnected() and would leave/force Disconnected.
	assert.Equal(t, connection.StateValueConnected, st.GetStateValue(), "state should follow successful collectibles call, not provider IsConnected()")
}

// TestCollectiblesStatus_CanceledErrorDoesNotFlipToDisconnected
// When the circuit breaker call fails with context.Canceled, we must not treat that as
// a hard "providers down" signal (old checkConnectionStatus still forced Disconnected
// if no provider was IsConnected() at defer time).
func TestCollectiblesStatus_CanceledErrorDoesNotFlipToDisconnected(t *testing.T) {
	t.Parallel()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // all RPC-like calls are canceled
	chainID := walletCommon.ChainID(1)
	owner := common.HexToAddress("0x1234567890abcdef")
	providerID := "test_provider_canceled"

	chainClient := &CopyableMockChainClient{MockClientInterface: mock_client.NewMockClientInterface(mockCtrl)}
	chainClient.EXPECT().CodeAt(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	rpcClient := mock_rpcclient.NewMockClientInterface(mockCtrl)
	rpcClient.EXPECT().EthClient(gomock.Any()).Return(chainClient, nil).AnyTimes()

	mockProvider := mock_thirdparty.NewMockCollectibleAccountOwnershipProvider(mockCtrl)
	mockProvider.EXPECT().IsChainSupported(chainID).Return(true).AnyTimes()
	mockProvider.EXPECT().IsConnected().Return(false).AnyTimes()
	mockProvider.EXPECT().ID().Return(providerID).AnyTimes()
	mockProvider.EXPECT().FetchAllAssetsByOwner(gomock.Any(), chainID, owner, "", 1).Return(
		(*thirdparty.FullCollectibleDataContainer)(nil), context.Canceled)

	mockProviders := thirdparty.CollectibleProviders{
		AccountOwnershipProviders: []thirdparty.CollectibleAccountOwnershipProvider{mockProvider},
	}
	manager := NewManager(nil, rpcClient, nil, mockProviders, nil, new(event.Feed))
	chainKey := chainID.String()
	statuses := &sync.Map{}
	statuses.Store(chainKey, connection.NewStatus())
	manager.statuses = statuses
	manager.statusNotifier = createStatusNotifier(statuses, manager.feed)
	manager.collectiblesDataDB = mock_collectibles.NewMockCollectibleDataStorage(mockCtrl)
	manager.collectionsDataDB = mock_collectibles.NewMockCollectionDataStorage(mockCtrl)
	communityManager := mock_community.NewMockCommunityManagerInterface(mockCtrl)
	manager.communityManager = communityManager
	ownershipDB := mock_ownership.NewMockOwnershipStorage(mockCtrl)
	ownershipDB.EXPECT().GetLatestOwnershipUpdateTimestamp(gomock.Any()).Return(int64(0), nil).AnyTimes()
	ownershipDB.EXPECT().GetOwnership(gomock.Any()).Return([]thirdparty.AccountBalance{}, nil).AnyTimes()
	manager.ownershipDB = ownershipDB

	_, err := manager.FetchAllAssetsByOwner(ctx, chainID, owner, "", 1, thirdparty.FetchFromAnyProvider)
	assert.Error(t, err)

	loaded, ok := manager.statuses.Load(chainKey)
	require.True(t, ok)
	st := loaded.(*connection.Status)
	assert.Equal(t, connection.StateValueUnknown, st.GetStateValue(), "canceled/ignored outcome must not mark chain as disconnected for collectibles")
}

// TestCollectiblesStatus_MultipleSuccessfulFetchesNoDisconnectedSpike
// Two successful fetches in a row should not oscillate: second SetIsConnected(true) is a no-op.
func TestCollectiblesStatus_MultipleSuccessfulFetchesNoDisconnectedSpike(t *testing.T) {
	t.Parallel()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()
	chainID := walletCommon.ChainID(1)
	owner := common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd")

	chainClient := &CopyableMockChainClient{MockClientInterface: mock_client.NewMockClientInterface(mockCtrl)}
	chainClient.EXPECT().CallContract(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	chainClient.EXPECT().CodeAt(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	rpcClient := mock_rpcclient.NewMockClientInterface(mockCtrl)
	rpcClient.EXPECT().EthClient(gomock.Any()).Return(chainClient, nil).AnyTimes()

	timestamp := time.Now().UnixNano()
	mockProvider := mock_thirdparty.NewMockCollectibleAccountOwnershipProvider(mockCtrl)
	mockProvider.EXPECT().IsChainSupported(chainID).Return(true).AnyTimes()
	// Stale: still false while requests succeed; old defer path overwrites to disconnected each time
	mockProvider.EXPECT().IsConnected().Return(false).AnyTimes()
	mockProvider.EXPECT().ID().Return(fmt.Sprintf("m_%d", timestamp)).AnyTimes()

	item := thirdparty.FullCollectibleData{
		CollectibleData: thirdparty.CollectibleData{
			ID: thirdparty.CollectibleUniqueID{
				ContractID: thirdparty.ContractID{
					Address: common.HexToAddress("0x0000000000000000000000000000000000000001"),
					ChainID: chainID,
				},
				TokenID: &bigint.BigInt{Int: big.NewInt(1)},
			},
		},
	}
	mockContainer := &thirdparty.FullCollectibleDataContainer{Items: []thirdparty.FullCollectibleData{item}}
	mockProvider.EXPECT().FetchAllAssetsByOwner(gomock.Any(), chainID, owner, "", 1).Return(mockContainer, nil).Times(2)

	manager := NewManager(nil, rpcClient, nil, thirdparty.CollectibleProviders{
		AccountOwnershipProviders: []thirdparty.CollectibleAccountOwnershipProvider{mockProvider},
	}, nil, new(event.Feed))
	chainKey := chainID.String()
	statuses := &sync.Map{}
	statuses.Store(chainKey, connection.NewStatus())
	manager.statuses = statuses
	manager.statusNotifier = createStatusNotifier(statuses, manager.feed)

	// Databases + community
	collectiblesDataDB := mock_collectibles.NewMockCollectibleDataStorage(mockCtrl)
	collectiblesDataDB.EXPECT().SetData(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	collectiblesDataDB.EXPECT().GetData(gomock.Any()).DoAndReturn(func(ids []thirdparty.CollectibleUniqueID) (map[string]thirdparty.CollectibleData, error) {
		m := make(map[string]thirdparty.CollectibleData)
		for _, id := range ids {
			m[id.HashKey()] = thirdparty.CollectibleData{ID: id}
		}
		return m, nil
	}).AnyTimes()
	collectiblesDataDB.EXPECT().GetCommunityInfo(gomock.Any()).Return(&thirdparty.CollectibleCommunityInfo{}, nil).AnyTimes()
	collectionsDataDB := mock_collectibles.NewMockCollectionDataStorage(mockCtrl)
	collectionsDataDB.EXPECT().SetData(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	collectionsDataDB.EXPECT().GetData(gomock.Any()).Return(map[string]thirdparty.CollectionData{}, nil).AnyTimes()
	communityManager := mock_community.NewMockCommunityManagerInterface(mockCtrl)
	communityManager.EXPECT().GetCommunityID(gomock.Any()).Return("c").AnyTimes()
	communityManager.EXPECT().FillCollectiblesMetadata(gomock.Any(), gomock.Any()).Return(true, nil).AnyTimes()
	communityManager.EXPECT().GetCommunityInfo(gomock.Any()).Return(&thirdparty.CommunityInfo{}, nil, nil).AnyTimes()
	ownershipDB := mock_ownership.NewMockOwnershipStorage(mockCtrl)
	ownershipDB.EXPECT().GetLatestOwnershipUpdateTimestamp(gomock.Any()).Return(int64(0), nil).AnyTimes()
	ownershipDB.EXPECT().GetOwnership(gomock.Any()).Return([]thirdparty.AccountBalance{}, nil).AnyTimes()
	manager.collectiblesDataDB = collectiblesDataDB
	manager.collectionsDataDB = collectionsDataDB
	manager.communityManager = communityManager
	manager.ownershipDB = ownershipDB

	_, err := manager.FetchAllAssetsByOwner(ctx, chainID, owner, "", 1, thirdparty.FetchFromAnyProvider)
	require.NoError(t, err)
	_, err = manager.FetchAllAssetsByOwner(ctx, chainID, owner, "", 1, thirdparty.FetchFromAnyProvider)
	require.NoError(t, err)
	loaded, _ := manager.statuses.Load(chainKey)
	st := loaded.(*connection.Status)
	assert.Equal(t, connection.StateValueConnected, st.GetStateValue())
}

func TestIsCollectiblesIgnorableError(t *testing.T) {
	t.Parallel()
	wrapped := fmt.Errorf("wrap: %w", context.Canceled)
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"canceled", context.Canceled, true},
		{"wrapped_canceled", wrapped, true},
		{"other", errors.New("fail"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isCollectiblesIgnorableError(tc.err))
		})
	}
}

func TestApplyCallStatuses_DoesNotChangeStateWhenAllIgnored(t *testing.T) {
	t.Parallel()
	chainID := walletCommon.ChainID(1)
	m := &Manager{statuses: &sync.Map{}, feed: new(event.Feed)}
	m.statuses.Store(chainID.String(), connection.NewStatus())
	m.statusNotifier = createStatusNotifier(m.statuses, m.feed)

	now := time.Now()
	m.applyCallStatuses(chainID, []circuitbreaker.FunctorCallStatus{
		{Name: "a", Err: context.Canceled, Timestamp: now, StartTime: now},
		{Name: "b", Err: context.Canceled, Timestamp: now, StartTime: now},
	})
	st := mustConnStatus(t, m.statuses, chainID)
	assert.Equal(t, connection.StateValueUnknown, st.GetStateValue())
}

func TestApplyCallStatuses_AllFailuresSetDisconnected(t *testing.T) {
	t.Parallel()
	chainID := walletCommon.ChainID(1)
	m := &Manager{statuses: &sync.Map{}, feed: new(event.Feed)}
	m.statuses.Store(chainID.String(), connection.NewStatus())
	m.statusNotifier = createStatusNotifier(m.statuses, m.feed)
	now := time.Now()
	m.applyCallStatuses(chainID, []circuitbreaker.FunctorCallStatus{
		{Name: "a", Err: errors.New("e1"), Timestamp: now, StartTime: now},
	})
	st := mustConnStatus(t, m.statuses, chainID)
	assert.Equal(t, connection.StateValueDisconnected, st.GetStateValue())
}

func TestApplyCallStatuses_SuccessWinsInMixedList(t *testing.T) {
	t.Parallel()
	chainID := walletCommon.ChainID(1)
	m := &Manager{statuses: &sync.Map{}, feed: new(event.Feed)}
	m.statuses.Store(chainID.String(), connection.NewStatus())
	m.statusNotifier = createStatusNotifier(m.statuses, m.feed)
	now := time.Now()
	m.applyCallStatuses(chainID, []circuitbreaker.FunctorCallStatus{
		{Name: "a", Err: errors.New("failed"), Timestamp: now, StartTime: now},
		{Name: "b", Err: nil, Timestamp: now, StartTime: now},
	})
	st := mustConnStatus(t, m.statuses, chainID)
	assert.Equal(t, connection.StateValueConnected, st.GetStateValue())
}

func mustConnStatus(t *testing.T, statuses *sync.Map, chainID walletCommon.ChainID) *connection.Status {
	t.Helper()
	v, ok := statuses.Load(chainID.String())
	require.True(t, ok)
	return v.(*connection.Status)
}

// testSearchProvider implements thirdparty.CollectibleSearchProvider for search status tests.
type testSearchProvider struct {
	id      string
	chainID walletCommon.ChainID
	scRes   *thirdparty.FullCollectibleDataContainer
	scErr   error
	scolRes *thirdparty.CollectionDataContainer
	scolErr error
}

func (p *testSearchProvider) ID() string                                   { return p.id }
func (p *testSearchProvider) IsConnected() bool                            { return true }
func (p *testSearchProvider) IsChainSupported(c walletCommon.ChainID) bool { return c == p.chainID }
func (p *testSearchProvider) SearchCollectibles(ctx context.Context, chainID walletCommon.ChainID, _ []common.Address, _, _ string, _ int) (*thirdparty.FullCollectibleDataContainer, error) {
	_ = ctx
	_ = chainID
	if p.scRes != nil || p.scErr != nil {
		return p.scRes, p.scErr
	}
	return &thirdparty.FullCollectibleDataContainer{}, nil
}
func (p *testSearchProvider) SearchCollections(ctx context.Context, chainID walletCommon.ChainID, _, _ string, _ int) (*thirdparty.CollectionDataContainer, error) {
	_ = ctx
	_ = chainID
	if p.scolRes != nil || p.scolErr != nil {
		return p.scolRes, p.scolErr
	}
	return &thirdparty.CollectionDataContainer{}, nil
}

// TestSearchCollectibles_OnlyIgnorableProviderErrors_DoesNotDisconnect
// Repro: only context.Canceled (or other ignorable) perrs would leave the final
// return as ErrAllProvidersFailedForChainID; that sentinel must not drive
// connection state if every provider error was ignorable.
func TestSearchCollectibles_OnlyIgnorableProviderErrors_DoesNotDisconnect(t *testing.T) {
	t.Parallel()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	chainID := walletCommon.ChainID(1)
	sp := &testSearchProvider{id: "sp1", chainID: chainID, scErr: context.Canceled}
	manager := NewManager(nil, nil, nil, thirdparty.CollectibleProviders{SearchProviders: []thirdparty.CollectibleSearchProvider{sp}}, nil, new(event.Feed))

	chainKey := chainID.String()
	statuses := &sync.Map{}
	statuses.Store(chainKey, connection.NewStatus())
	manager.statuses = statuses
	manager.statusNotifier = createStatusNotifier(statuses, manager.feed)
	cdb := mock_collectibles.NewMockCollectibleDataStorage(mockCtrl)
	coldb := mock_collectibles.NewMockCollectionDataStorage(mockCtrl)
	manager.collectiblesDataDB = cdb
	manager.collectionsDataDB = coldb
	manager.communityManager = mock_community.NewMockCommunityManagerInterface(mockCtrl)
	odb := mock_ownership.NewMockOwnershipStorage(mockCtrl)
	odb.EXPECT().GetLatestOwnershipUpdateTimestamp(gomock.Any()).Return(int64(0), nil).AnyTimes()
	odb.EXPECT().GetOwnership(gomock.Any()).Return([]thirdparty.AccountBalance{}, nil).AnyTimes()
	manager.ownershipDB = odb

	_, err := manager.SearchCollectibles(context.Background(), chainID, "q", "", 10, thirdparty.FetchFromAnyProvider)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAllProvidersFailedForChainID)

	st := mustConnStatus(t, manager.statuses, chainID)
	assert.Equal(t, connection.StateValueUnknown, st.GetStateValue(), "all-ignorable provider search failures must not mark disconnected")
}

// TestSearchCollections_ProviderSuccessSetDataError_StaysConnected
// A successful provider response must set connected before processCollectionData;
// a DB error there is not a provider connectivity issue.
func TestSearchCollections_ProviderSuccessSetDataError_StaysConnected(t *testing.T) {
	t.Parallel()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	chainID := walletCommon.ChainID(1)
	sp := &testSearchProvider{
		id:      "sp1",
		chainID: chainID,
		scolRes: &thirdparty.CollectionDataContainer{
			Items: []thirdparty.CollectionData{
				{ID: thirdparty.ContractID{ChainID: chainID, Address: common.HexToAddress("0x1")}, Name: "a"},
			},
		},
	}
	manager := NewManager(nil, nil, nil, thirdparty.CollectibleProviders{SearchProviders: []thirdparty.CollectibleSearchProvider{sp}}, nil, new(event.Feed))
	chainKey := chainID.String()
	statuses := &sync.Map{}
	statuses.Store(chainKey, connection.NewStatus())
	manager.statuses = statuses
	manager.statusNotifier = createStatusNotifier(statuses, manager.feed)
	manager.collectiblesDataDB = mock_collectibles.NewMockCollectibleDataStorage(mockCtrl)
	coldb := mock_collectibles.NewMockCollectionDataStorage(mockCtrl)
	dbErr := errors.New("set failed")
	coldb.EXPECT().SetData(gomock.Any(), gomock.Any()).Return(dbErr)
	manager.collectionsDataDB = coldb
	odb := mock_ownership.NewMockOwnershipStorage(mockCtrl)
	odb.EXPECT().GetLatestOwnershipUpdateTimestamp(gomock.Any()).Return(int64(0), nil).AnyTimes()
	odb.EXPECT().GetOwnership(gomock.Any()).Return([]thirdparty.AccountBalance{}, nil).AnyTimes()
	manager.ownershipDB = odb
	manager.communityManager = mock_community.NewMockCommunityManagerInterface(mockCtrl)

	_, err := manager.SearchCollections(context.Background(), chainID, "q", "", 10, thirdparty.FetchFromAnyProvider)
	require.ErrorIs(t, err, dbErr)
	st := mustConnStatus(t, manager.statuses, chainID)
	assert.Equal(t, connection.StateValueConnected, st.GetStateValue(), "local DB error after a successful search provider response must not flip to disconnected")
}
