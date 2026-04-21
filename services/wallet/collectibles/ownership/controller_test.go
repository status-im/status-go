package ownership_test

import (
	"context"
	"math/big"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	coretypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"
	"github.com/status-im/go-wallet-sdk/pkg/contracts/erc1155"
	"github.com/status-im/go-wallet-sdk/pkg/contracts/erc721"
	"github.com/status-im/go-wallet-sdk/pkg/eventlog"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/db/walletdatabase"
	"github.com/status-im/status-go/internal/rpc/network"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/services/accounts/accountsevent"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/collectibles/ownership"
	mock_ownership "github.com/status-im/status-go/services/wallet/collectibles/ownership/mock"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/multistandardbalance"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/transferdetector"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestControllerMultipleAccountsAddedEvent(t *testing.T) {
	mockCtrl := gomock.NewController(t)

	accountsProvider := mock_ownership.NewMockAccountsProvider(mockCtrl)
	fakeAddress := types.HexToAddress("0x123")
	accountsProvider.EXPECT().GetWalletAddresses().Return([]types.Address{fakeAddress}, nil).AnyTimes()

	accountsPublisher := pubsub.NewPublisher()

	networksProvider := mock_ownership.NewMockNetworksProvider(mockCtrl)

	networksPublisher := pubsub.NewPublisher()

	networksProvider.EXPECT().GetActiveNetworks().Return([]*params.Network{
		{
			ChainID:  1,
			IsActive: true,
		},
	}, nil).AnyTimes()
	networksProvider.EXPECT().GetPublisher().Return(networksPublisher).AnyTimes()

	walletDB, err := testutils.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	emptyCollectiblesContainer := &thirdparty.CollectibleOwnershipContainer{
		Items:          []thirdparty.CollectibleIDBalance{},
		NextCursor:     "",
		PreviousCursor: "",
		Provider:       "mockProvider",
	}
	ownershipFetcher := mock_ownership.NewMockCollectibleOwnershipFetcher(mockCtrl)
	ownershipFetcher.EXPECT().FetchCollectibleOwnershipByOwner(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, chainID walletCommon.ChainID, owner common.Address, cursor string, limit int, providerID string) (*thirdparty.CollectibleOwnershipContainer, error) {
			// Simulate fetch duration
			time.Sleep(500 * time.Millisecond)
			return emptyCollectiblesContainer, nil
		}).AnyTimes()

	multistandardBalancePublisher := pubsub.NewPublisher()
	transferDetectorPublisher := pubsub.NewPublisher()
	blockChainStateProvider := mock_ownership.NewMockBlockChainStateProvider(mockCtrl)
	publisher := pubsub.NewPublisher()
	logger := zaptest.NewLogger(t).WithOptions(zap.AddCallerSkip(1))

	controller := ownership.NewController(
		ownership.NewOwnershipDB(walletDB),
		accountsProvider,
		accountsPublisher,
		networksProvider,
		multistandardBalancePublisher,
		transferDetectorPublisher,
		blockChainStateProvider,
		ownershipFetcher,
		publisher,
		logger,
	)

	controller.StartWithLoaderParams(
		ownership.PeriodicalLoaderParams{
			StartDelay:   0 * time.Second,
			LoadInterval: 10 * time.Second,
			LoaderParams: ownership.LoaderParams{
				LoadDelay:  0 * time.Second,
				FetchLimit: 50,
			},
		},
	)

	for i := 0; i < 10; i++ {
		pubsub.Publish(accountsPublisher, accountsevent.AccountsAddedEvent{
			Accounts: []common.Address{common.Address(fakeAddress)},
		})
	}

	// Check that the controller eventually reaches the updating state
	require.Eventually(t, func() bool {
		state := controller.GetLoaderState(walletCommon.ChainID(1), common.Address(fakeAddress))
		return state == ownership.LoaderStateUpdating
	}, 1*time.Second, 100*time.Millisecond)

	// Check that the controller eventually reaches the idle state
	require.Eventually(t, func() bool {
		state := controller.GetLoaderState(walletCommon.ChainID(1), common.Address(fakeAddress))
		return state == ownership.LoaderStateIdle
	}, 1*time.Second, 100*time.Millisecond)

	controller.Stop()
}

func TestControllerMultiStandardBalanceEvents(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	accountsProvider := mock_ownership.NewMockAccountsProvider(mockCtrl)
	fakeAddress := types.HexToAddress("0x123")
	accountsProvider.EXPECT().GetWalletAddresses().Return([]types.Address{fakeAddress}, nil).AnyTimes()

	accountsPublisher := pubsub.NewPublisher()
	networksProvider := mock_ownership.NewMockNetworksProvider(mockCtrl)
	networksPublisher := pubsub.NewPublisher()

	networksProvider.EXPECT().GetActiveNetworks().Return([]*params.Network{
		{ChainID: 1, IsActive: true},
	}, nil).AnyTimes()
	networksProvider.EXPECT().GetPublisher().Return(networksPublisher).AnyTimes()

	walletDB, err := testutils.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	ownershipDB := ownership.NewOwnershipDB(walletDB)

	emptyCollectiblesContainer := &thirdparty.CollectibleOwnershipContainer{
		Items:          []thirdparty.CollectibleIDBalance{},
		NextCursor:     "",
		PreviousCursor: "",
		Provider:       "mockProvider",
	}
	ownershipFetcher := mock_ownership.NewMockCollectibleOwnershipFetcher(mockCtrl)
	ownershipFetcher.EXPECT().FetchCollectibleOwnershipByOwner(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, chainID walletCommon.ChainID, owner common.Address, cursor string, limit int, providerID string) (*thirdparty.CollectibleOwnershipContainer, error) {
			// Simulate fetch duration
			time.Sleep(500 * time.Millisecond)
			return emptyCollectiblesContainer, nil
		}).AnyTimes()

	multistandardBalancePublisher := pubsub.NewPublisher()
	transferDetectorPublisher := pubsub.NewPublisher()
	blockChainStateProvider := mock_ownership.NewMockBlockChainStateProvider(mockCtrl)
	publisher := pubsub.NewPublisher()
	logger := zaptest.NewLogger(t).WithOptions(zap.AddCallerSkip(1))

	controller := ownership.NewController(
		ownershipDB,
		accountsProvider,
		accountsPublisher,
		networksProvider,
		multistandardBalancePublisher,
		transferDetectorPublisher,
		blockChainStateProvider,
		ownershipFetcher,
		publisher,
		logger,
	)

	params := ownership.PeriodicalLoaderParams{
		StartDelay:   1 * time.Second,
		LoadInterval: 10 * time.Second,
		LoaderParams: ownership.LoaderParams{
			LoadDelay:  0 * time.Second,
			FetchLimit: 50,
		},
	}

	controller.StartWithLoaderParams(params)

	// Check that the controller eventually reaches the updating state
	require.Eventually(t, func() bool {
		state := controller.GetLoaderState(walletCommon.ChainID(1), common.Address(fakeAddress))
		return state == ownership.LoaderStateUpdating
	}, 1*time.Second, 100*time.Millisecond)

	// Check that the controller eventually reaches the idle state
	require.Eventually(t, func() bool {
		state := controller.GetLoaderState(walletCommon.ChainID(1), common.Address(fakeAddress))
		return state == ownership.LoaderStateIdle
	}, 1*time.Second, 100*time.Millisecond)

	// Test ERC721 transfer event
	erc721Event := multistandardbalance.EventBalanceFetchFinished{
		Key: multistandardbalance.BalancesKey{
			ChainID: 1,
			Account: common.Address(fakeAddress),
		},
		ResultType:     multistandardfetcher.ResultTypeERC721,
		BalanceChanged: true,
		OldState: multistandardbalance.State{
			FetchedAt: time.Now().Unix() - 1000,
		},
		NewState: multistandardbalance.State{
			FetchedAt: time.Now().Unix(),
		},
	}
	pubsub.Publish(multistandardBalancePublisher, erc721Event)

	// Check that the controller eventually reaches the delayed state
	require.Eventually(t, func() bool {
		state := controller.GetLoaderState(walletCommon.ChainID(1), common.Address(fakeAddress))
		return state == ownership.LoaderStateDelayed
	}, 1*time.Second, 100*time.Millisecond)

	// Check that the controller eventually reaches the updating state
	require.Eventually(t, func() bool {
		state := controller.GetLoaderState(walletCommon.ChainID(1), common.Address(fakeAddress))
		return state == ownership.LoaderStateUpdating
	}, 2*params.StartDelay, 100*time.Millisecond)

	// Check that the controller eventually reaches the idle state
	require.Eventually(t, func() bool {
		state := controller.GetLoaderState(walletCommon.ChainID(1), common.Address(fakeAddress))
		return state == ownership.LoaderStateIdle
	}, 1*time.Second, 100*time.Millisecond)

	// Test ERC1155 transfer event
	erc1155Event := multistandardbalance.EventBalanceFetchFinished{
		Key: multistandardbalance.BalancesKey{
			ChainID: 1,
			Account: common.Address(fakeAddress),
		},
		ResultType:     multistandardfetcher.ResultTypeERC1155,
		BalanceChanged: true,
		OldState: multistandardbalance.State{
			FetchedAt: time.Now().Unix() - 1000,
		},
		NewState: multistandardbalance.State{
			FetchedAt: time.Now().Unix(),
		},
	}
	pubsub.Publish(multistandardBalancePublisher, erc1155Event)

	// Check that the controller eventually reaches the delayed state
	require.Eventually(t, func() bool {
		state := controller.GetLoaderState(walletCommon.ChainID(1), common.Address(fakeAddress))
		return state == ownership.LoaderStateDelayed
	}, 1*time.Second, 100*time.Millisecond)

	// Check that the controller eventually reaches the updating state
	require.Eventually(t, func() bool {
		state := controller.GetLoaderState(walletCommon.ChainID(1), common.Address(fakeAddress))
		return state == ownership.LoaderStateUpdating
	}, 2*params.StartDelay, 100*time.Millisecond)

	// Check that the controller eventually reaches the idle state
	require.Eventually(t, func() bool {
		state := controller.GetLoaderState(walletCommon.ChainID(1), common.Address(fakeAddress))
		return state == ownership.LoaderStateIdle
	}, 1*time.Second, 100*time.Millisecond)

	// Test too old transfer detected event (should not trigger refetch)
	tooOldEvent := multistandardbalance.EventBalanceFetchFinished{
		Key: multistandardbalance.BalancesKey{
			ChainID: 1,
			Account: common.Address(fakeAddress),
		},
		ResultType:     multistandardfetcher.ResultTypeERC721,
		BalanceChanged: true,
		OldState: multistandardbalance.State{
			FetchedAt: time.Now().Unix() - 2000 - 30*60,
		},
		NewState: multistandardbalance.State{
			FetchedAt: time.Now().Unix() - 1000 - 30*60,
		},
	}
	pubsub.Publish(multistandardBalancePublisher, tooOldEvent)

	time.Sleep(100 * time.Millisecond)

	require.Equal(t, ownership.LoaderStateIdle, controller.GetLoaderState(walletCommon.ChainID(1), common.Address(fakeAddress)))

	// Test non-relevant event (should not trigger refetch)
	pubsub.Publish(multistandardBalancePublisher, multistandardbalance.EventBalanceFetchFinished{
		Key: multistandardbalance.BalancesKey{
			ChainID: 1,
			Account: common.Address(fakeAddress),
		},
		ResultType:     multistandardfetcher.ResultTypeNative, // Native balance fetch
		BalanceChanged: true,
		OldState: multistandardbalance.State{
			FetchedAt: time.Now().Unix() - 1000,
		},
		NewState: multistandardbalance.State{
			FetchedAt: time.Now().Unix(),
		},
	})

	time.Sleep(100 * time.Millisecond)

	require.Equal(t, ownership.LoaderStateIdle, controller.GetLoaderState(walletCommon.ChainID(1), common.Address(fakeAddress)))

	pubsub.Publish(multistandardBalancePublisher, multistandardbalance.EventBalanceFetchFinished{
		Key: multistandardbalance.BalancesKey{
			ChainID: 1,
			Account: common.Address(fakeAddress),
		},
		ResultType:     multistandardfetcher.ResultTypeERC721,
		BalanceChanged: false, // Balance unchanged
		OldState: multistandardbalance.State{
			FetchedAt: time.Now().Unix() - 1000,
		},
		NewState: multistandardbalance.State{
			FetchedAt: time.Now().Unix(),
		},
	})

	time.Sleep(100 * time.Millisecond)

	require.Equal(t, ownership.LoaderStateIdle, controller.GetLoaderState(walletCommon.ChainID(1), common.Address(fakeAddress)))

	controller.Stop()
}

func TestControllerNetworkEventsWatcher(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	accountsProvider := mock_ownership.NewMockAccountsProvider(mockCtrl)
	accountsPublisher := pubsub.NewPublisher()
	fakeAddress := types.HexToAddress("0x123")
	accountsProvider.EXPECT().GetWalletAddresses().Return([]types.Address{fakeAddress}, nil).AnyTimes()

	networksProvider := mock_ownership.NewMockNetworksProvider(mockCtrl)
	networksPublisher := pubsub.NewPublisher()
	chain1 := &params.Network{ChainID: 1, IsActive: true}
	chain2 := &params.Network{ChainID: 2, IsActive: true}
	networksProvider.EXPECT().GetPublisher().Return(networksPublisher).AnyTimes()

	walletDB, err := testutils.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	emptyCollectiblesContainer := &thirdparty.CollectibleOwnershipContainer{
		Items:          []thirdparty.CollectibleIDBalance{},
		NextCursor:     "",
		PreviousCursor: "",
		Provider:       "mockProvider",
	}
	ownershipFetcher := mock_ownership.NewMockCollectibleOwnershipFetcher(mockCtrl)
	ownershipFetcher.EXPECT().FetchCollectibleOwnershipByOwner(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, chainID walletCommon.ChainID, owner common.Address, cursor string, limit int, providerID string) (*thirdparty.CollectibleOwnershipContainer, error) {
			// Simulate fetch duration
			time.Sleep(500 * time.Millisecond)
			return emptyCollectiblesContainer, nil
		}).AnyTimes()

	multistandardBalancePublisher := pubsub.NewPublisher()
	transferDetectorPublisher := pubsub.NewPublisher()
	blockChainStateProvider := mock_ownership.NewMockBlockChainStateProvider(mockCtrl)
	publisher := pubsub.NewPublisher()
	logger := zaptest.NewLogger(t).WithOptions(zap.AddCallerSkip(1))

	controller := ownership.NewController(
		ownership.NewOwnershipDB(walletDB),
		accountsProvider,
		accountsPublisher,
		networksProvider,
		multistandardBalancePublisher,
		transferDetectorPublisher,
		blockChainStateProvider,
		ownershipFetcher,
		publisher,
		logger,
	)

	networksProvider.EXPECT().GetActiveNetworks().Return([]*params.Network{chain1}, nil).Times(1)

	controller.StartWithLoaderParams(
		ownership.PeriodicalLoaderParams{
			StartDelay:   0 * time.Second,
			LoadInterval: 10 * time.Second,
			LoaderParams: ownership.LoaderParams{
				LoadDelay:  0 * time.Second,
				FetchLimit: 50,
			},
		},
	)

	// Check that the controller eventually reaches the updating state
	require.Eventually(t, func() bool {
		state1 := controller.GetLoaderState(walletCommon.ChainID(chain1.ChainID), common.Address(fakeAddress))
		state2 := controller.GetLoaderState(walletCommon.ChainID(chain2.ChainID), common.Address(fakeAddress))
		return state1 == ownership.LoaderStateUpdating && state2 == ownership.LoaderStateNotAvailable
	}, 1*time.Second, 100*time.Millisecond)

	// Check that the controller eventually reaches the idle state
	require.Eventually(t, func() bool {
		state1 := controller.GetLoaderState(walletCommon.ChainID(chain1.ChainID), common.Address(fakeAddress))
		state2 := controller.GetLoaderState(walletCommon.ChainID(chain2.ChainID), common.Address(fakeAddress))
		return state1 == ownership.LoaderStateIdle && state2 == ownership.LoaderStateNotAvailable
	}, 1*time.Second, 100*time.Millisecond)

	// Add a new chain
	networksProvider.EXPECT().GetActiveNetworks().Return([]*params.Network{chain1, chain2}, nil).Times(1)

	// Publish network change event
	pubsub.Publish(networksPublisher, network.EventActiveNetworksChanged{})

	// Check the old chain stays idle and the new chain reaches the updating state
	require.Eventually(t, func() bool {
		state1 := controller.GetLoaderState(walletCommon.ChainID(chain1.ChainID), common.Address(fakeAddress))
		state2 := controller.GetLoaderState(walletCommon.ChainID(chain2.ChainID), common.Address(fakeAddress))
		return state1 == ownership.LoaderStateIdle && state2 == ownership.LoaderStateUpdating
	}, 1*time.Second, 100*time.Millisecond)

	// Check that the new chain eventually reaches the idle state
	require.Eventually(t, func() bool {
		state1 := controller.GetLoaderState(walletCommon.ChainID(chain1.ChainID), common.Address(fakeAddress))
		state2 := controller.GetLoaderState(walletCommon.ChainID(chain2.ChainID), common.Address(fakeAddress))
		return state1 == ownership.LoaderStateIdle && state2 == ownership.LoaderStateIdle
	}, 1*time.Second, 100*time.Millisecond)

	// Remove chain
	networksProvider.EXPECT().GetActiveNetworks().Return([]*params.Network{chain1}, nil).Times(1)

	// Publish network change event
	pubsub.Publish(networksPublisher, network.EventActiveNetworksChanged{})

	// Check that the remaining chain stays idle and the removed chain reaches the not available state
	require.Eventually(t, func() bool {
		state1 := controller.GetLoaderState(walletCommon.ChainID(chain1.ChainID), common.Address(fakeAddress))
		state2 := controller.GetLoaderState(walletCommon.ChainID(chain2.ChainID), common.Address(fakeAddress))
		return state1 == ownership.LoaderStateIdle && state2 == ownership.LoaderStateNotAvailable
	}, 1*time.Second, 100*time.Millisecond)

	controller.Stop()
}

// TestControllerTriggerLoad tests the TriggerLoad method with realistic data and storage verification
func TestControllerTriggerLoad(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	accountsProvider := mock_ownership.NewMockAccountsProvider(mockCtrl)
	fakeAddress := types.HexToAddress("0x123")
	accountsProvider.EXPECT().GetWalletAddresses().Return([]types.Address{fakeAddress}, nil).AnyTimes()

	accountsPublisher := pubsub.NewPublisher()
	networksProvider := mock_ownership.NewMockNetworksProvider(mockCtrl)
	networksPublisher := pubsub.NewPublisher()

	networksProvider.EXPECT().GetActiveNetworks().Return([]*params.Network{
		{ChainID: 1, IsActive: true},
	}, nil).AnyTimes()
	networksProvider.EXPECT().GetPublisher().Return(networksPublisher).AnyTimes()

	walletDB, err := testutils.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	// Create a realistic collectibles container with actual data for TriggerLoad test
	balance1a := thirdparty.CollectibleIDBalance{
		ID: thirdparty.CollectibleUniqueID{
			ContractID: thirdparty.ContractID{
				Address: common.HexToAddress("0x1234567890123456789012345678901234567890"),
				ChainID: 1,
			},
			TokenID: &bigint.BigInt{Int: big.NewInt(1)},
		},
		Balance: &bigint.BigInt{Int: big.NewInt(5)},
	}
	balance1b := thirdparty.CollectibleIDBalance{
		ID: thirdparty.CollectibleUniqueID{
			ContractID: thirdparty.ContractID{
				Address: common.HexToAddress("0x0987654321098765432109876543210987654321"),
				ChainID: 1,
			},
			TokenID: &bigint.BigInt{Int: big.NewInt(2)},
		},
		Balance: &bigint.BigInt{Int: big.NewInt(1)},
	}
	collectiblesContainer1 := &thirdparty.CollectibleOwnershipContainer{
		Items: []thirdparty.CollectibleIDBalance{
			balance1a,
			balance1b,
		},
		NextCursor:     "",
		PreviousCursor: "",
		Provider:       "mockProvider",
	}

	collectiblesContainer999 := &thirdparty.CollectibleOwnershipContainer{
		Items: []thirdparty.CollectibleIDBalance{
			{
				ID: thirdparty.CollectibleUniqueID{
					ContractID: thirdparty.ContractID{
						Address: common.HexToAddress("0x0987654321098765432109876543210987654321"),
						ChainID: 999,
					},
					TokenID: &bigint.BigInt{Int: big.NewInt(1)},
				},
				Balance: &bigint.BigInt{Int: big.NewInt(5)},
			},
		},
		NextCursor:     "",
		PreviousCursor: "",
		Provider:       "mockProvider",
	}

	ownershipFetcher := mock_ownership.NewMockCollectibleOwnershipFetcher(mockCtrl)
	// Expect specific calls for the existing account/chain
	ownershipFetcher.EXPECT().FetchCollectibleOwnershipByOwner(
		gomock.Any(),
		walletCommon.ChainID(1),
		common.Address(fakeAddress),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(collectiblesContainer1, nil).AnyTimes()

	// Expect call for the non-existing account/chain
	ownershipFetcher.EXPECT().FetchCollectibleOwnershipByOwner(
		gomock.Any(),
		walletCommon.ChainID(999),
		common.Address(types.HexToAddress("0x999")),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(collectiblesContainer999, nil).Times(1)

	multistandardBalancePublisher := pubsub.NewPublisher()
	transferDetectorPublisher := pubsub.NewPublisher()
	blockChainStateProvider := mock_ownership.NewMockBlockChainStateProvider(mockCtrl)
	publisher := pubsub.NewPublisher()
	logger := zaptest.NewLogger(t).WithOptions(zap.AddCallerSkip(1))

	controller := ownership.NewController(
		ownership.NewOwnershipDB(walletDB),
		accountsProvider,
		accountsPublisher,
		networksProvider,
		multistandardBalancePublisher,
		transferDetectorPublisher,
		blockChainStateProvider,
		ownershipFetcher,
		publisher,
		logger,
	)

	controller.StartWithLoaderParams(
		ownership.PeriodicalLoaderParams{
			StartDelay:   0 * time.Second,
			LoadInterval: 10 * time.Second,
			LoaderParams: ownership.LoaderParams{
				LoadDelay:  0 * time.Second,
				FetchLimit: 50,
			},
		},
	)

	// Wait for controller to start and any initial loads to complete
	require.Eventually(t, func() bool {
		// Check if controller is ready by verifying it can respond to state queries
		state := controller.GetLoaderState(walletCommon.ChainID(1), common.Address(fakeAddress))
		return state >= ownership.LoaderStateIdle && state <= ownership.LoaderStateError
	}, 300*time.Millisecond, 50*time.Millisecond)

	// Test TriggerLoad for existing account/chain
	// It's being periodically loaded already, existing loader should be used
	ctx := context.Background()
	err = controller.TriggerLoad(ctx, walletCommon.ChainID(1), common.Address(fakeAddress))
	require.NoError(t, err)

	// Test TriggerLoad for non-existing account/chain
	// Use a chain ID that's not in the active networks and a different address
	nonExistingAddress := types.HexToAddress("0x999") // Different address to avoid conflicts
	err = controller.TriggerLoad(ctx, walletCommon.ChainID(999), common.Address(nonExistingAddress))
	require.NoError(t, err)

	// Verify that the collectibles were properly stored in the database
	ownershipDB := ownership.NewOwnershipDB(walletDB)

	// Check storage for the existing account/chain
	balances, err := ownershipDB.GetOwnedCollectibles([]walletCommon.ChainID{walletCommon.ChainID(1)}, []common.Address{common.Address(fakeAddress)}, 0, 100)
	require.NoError(t, err)
	require.Len(t, balances, 2, "Should have 2 collectibles stored")

	// Check storage for the non-existing account/chain
	balances2, err := ownershipDB.GetOwnedCollectibles([]walletCommon.ChainID{walletCommon.ChainID(999)}, []common.Address{common.Address(nonExistingAddress)}, 0, 100)
	require.NoError(t, err)
	require.Len(t, balances2, 1, "Should have 1 collectible stored")

	controller.Stop()
}

func TestControllerAccountsEvents(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	accountsProvider := mock_ownership.NewMockAccountsProvider(mockCtrl)
	accountsPublisher := pubsub.NewPublisher()
	fakeAddress := types.HexToAddress("0x123")
	newAddress := types.HexToAddress("0x456")

	networksProvider := mock_ownership.NewMockNetworksProvider(mockCtrl)
	networksPublisher := pubsub.NewPublisher()
	networksProvider.EXPECT().GetPublisher().Return(networksPublisher).AnyTimes()
	networksProvider.EXPECT().GetActiveNetworks().Return([]*params.Network{
		{ChainID: 1, IsActive: true},
	}, nil).AnyTimes()

	walletDB, err := testutils.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	emptyCollectiblesContainer := &thirdparty.CollectibleOwnershipContainer{
		Items:          []thirdparty.CollectibleIDBalance{},
		NextCursor:     "",
		PreviousCursor: "",
		Provider:       "mockProvider",
	}
	ownershipFetcher := mock_ownership.NewMockCollectibleOwnershipFetcher(mockCtrl)
	ownershipFetcher.EXPECT().FetchCollectibleOwnershipByOwner(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, chainID walletCommon.ChainID, owner common.Address, cursor string, limit int, providerID string) (*thirdparty.CollectibleOwnershipContainer, error) {
			time.Sleep(500 * time.Millisecond)
			return emptyCollectiblesContainer, nil
		}).AnyTimes()

	multistandardBalancePublisher := pubsub.NewPublisher()
	transferDetectorPublisher := pubsub.NewPublisher()
	blockChainStateProvider := mock_ownership.NewMockBlockChainStateProvider(mockCtrl)
	publisher := pubsub.NewPublisher()
	logger := zaptest.NewLogger(t).WithOptions(zap.AddCallerSkip(1))

	controller := ownership.NewController(
		ownership.NewOwnershipDB(walletDB),
		accountsProvider,
		accountsPublisher,
		networksProvider,
		multistandardBalancePublisher,
		transferDetectorPublisher,
		blockChainStateProvider,
		ownershipFetcher,
		publisher,
		logger,
	)

	accountsProvider.EXPECT().GetWalletAddresses().Return([]types.Address{fakeAddress}, nil).Times(1)
	controller.StartWithLoaderParams(
		ownership.PeriodicalLoaderParams{
			StartDelay:   0 * time.Second,
			LoadInterval: 100 * time.Second,
			LoaderParams: ownership.LoaderParams{
				LoadDelay:  0 * time.Second,
				FetchLimit: 50,
			},
		},
	)

	// Wait for event processing
	require.Eventually(t, func() bool {
		state := controller.GetLoaderState(walletCommon.ChainID(1), common.Address(fakeAddress))
		return state == ownership.LoaderStateUpdating
	}, 200*time.Millisecond, 50*time.Millisecond)

	require.Eventually(t, func() bool {
		state := controller.GetLoaderState(walletCommon.ChainID(1), common.Address(fakeAddress))
		return state == ownership.LoaderStateIdle
	}, 1000*time.Millisecond, 50*time.Millisecond)

	// Verify that loader doesn't exist for new account
	removedAccountState := controller.GetLoaderState(walletCommon.ChainID(1), common.Address(newAddress))
	require.Equal(t, ownership.LoaderStateNotAvailable, removedAccountState)

	// Test AccountsAddedEvent
	accountsProvider.EXPECT().GetWalletAddresses().Return([]types.Address{fakeAddress, newAddress}, nil).Times(1)
	pubsub.Publish(accountsPublisher, accountsevent.AccountsAddedEvent{
		Accounts: []common.Address{common.Address(newAddress)},
	})

	// Wait for event processing
	require.Eventually(t, func() bool {
		state := controller.GetLoaderState(walletCommon.ChainID(1), common.Address(newAddress))
		return state == ownership.LoaderStateUpdating
	}, 200*time.Millisecond, 50*time.Millisecond)

	// Verify no state change for existing account
	existingAccountState := controller.GetLoaderState(walletCommon.ChainID(1), common.Address(fakeAddress))
	require.Equal(t, ownership.LoaderStateIdle, existingAccountState)

	// Wait for new account to be loaded
	require.Eventually(t, func() bool {
		state := controller.GetLoaderState(walletCommon.ChainID(1), common.Address(newAddress))
		return state == ownership.LoaderStateIdle
	}, 1000*time.Millisecond, 50*time.Millisecond)

	// Test AccountsRemovedEvent
	accountsProvider.EXPECT().GetWalletAddresses().Return([]types.Address{newAddress}, nil).Times(1)
	pubsub.Publish(accountsPublisher, accountsevent.AccountsRemovedEvent{
		Accounts: []common.Address{common.Address(fakeAddress)},
	})

	// Verify that loader was stopped for removed account
	require.Eventually(t, func() bool {
		state := controller.GetLoaderState(walletCommon.ChainID(1), common.Address(fakeAddress))
		return state == ownership.LoaderStateNotAvailable
	}, 1000*time.Millisecond, 50*time.Millisecond)

	// State unchanged for new account
	newAccountState := controller.GetLoaderState(walletCommon.ChainID(1), common.Address(newAddress))
	require.Equal(t, ownership.LoaderStateIdle, newAccountState)

	controller.Stop()
}

func TestControllerPeriodicalLoads(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	accountsProvider := mock_ownership.NewMockAccountsProvider(mockCtrl)
	fakeAddress := types.HexToAddress("0x123")
	accountsProvider.EXPECT().GetWalletAddresses().Return([]types.Address{fakeAddress}, nil).AnyTimes()

	accountsPublisher := pubsub.NewPublisher()
	networksProvider := mock_ownership.NewMockNetworksProvider(mockCtrl)
	networksPublisher := pubsub.NewPublisher()

	networksProvider.EXPECT().GetActiveNetworks().Return([]*params.Network{
		{ChainID: 1, IsActive: true},
	}, nil).AnyTimes()
	networksProvider.EXPECT().GetPublisher().Return(networksPublisher).AnyTimes()

	walletDB, err := testutils.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	emptyCollectiblesContainer := &thirdparty.CollectibleOwnershipContainer{
		Items:          []thirdparty.CollectibleIDBalance{},
		NextCursor:     "",
		PreviousCursor: "",
		Provider:       "mockProvider",
	}
	ownershipFetcher := mock_ownership.NewMockCollectibleOwnershipFetcher(mockCtrl)

	// Expect multiple calls due to periodic loading
	callCount := 0
	ownershipFetcher.EXPECT().FetchCollectibleOwnershipByOwner(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, chainID walletCommon.ChainID, owner common.Address, cursor string, limit int, providerID string) (*thirdparty.CollectibleOwnershipContainer, error) {
			callCount++
			return emptyCollectiblesContainer, nil
		}).AnyTimes()

	multistandardBalancePublisher := pubsub.NewPublisher()
	transferDetectorPublisher := pubsub.NewPublisher()
	blockChainStateProvider := mock_ownership.NewMockBlockChainStateProvider(mockCtrl)
	publisher := pubsub.NewPublisher()
	logger := zaptest.NewLogger(t).WithOptions(zap.AddCallerSkip(1))

	controller := ownership.NewController(
		ownership.NewOwnershipDB(walletDB),
		accountsProvider,
		accountsPublisher,
		networksProvider,
		multistandardBalancePublisher,
		transferDetectorPublisher,
		blockChainStateProvider,
		ownershipFetcher,
		publisher,
		logger,
	)

	// Make sure no loads have been made yet
	require.Equal(t, callCount, 0)

	// Use a short interval for testing
	controller.StartWithLoaderParams(
		ownership.PeriodicalLoaderParams{
			StartDelay:   0 * time.Second,
			LoadInterval: 500 * time.Millisecond, // Short interval for testing
			LoaderParams: ownership.LoaderParams{
				LoadDelay:  0 * time.Second,
				FetchLimit: 50,
			},
		},
	)

	// Wait for initial load (should happen immediately due to waitForInterval = false)
	require.Eventually(t, func() bool {
		return callCount > 0
	}, 300*time.Millisecond, 100*time.Millisecond)
	initialCallCount := callCount

	// Wait for at least one periodic load
	// The LoadInterval is 500ms, so we need to wait longer than that
	// The periodic loader will trigger after the full interval has passed
	require.Eventually(t, func() bool {
		return callCount > initialCallCount
	}, 1000*time.Millisecond, 100*time.Millisecond)

	controller.Stop()
}

func TestControllerTransferDetectionEvents(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	accountsProvider := mock_ownership.NewMockAccountsProvider(mockCtrl)
	fakeAddress := types.HexToAddress("0x123")
	accountsProvider.EXPECT().GetWalletAddresses().Return([]types.Address{fakeAddress}, nil).AnyTimes()

	accountsPublisher := pubsub.NewPublisher()
	networksProvider := mock_ownership.NewMockNetworksProvider(mockCtrl)
	networksPublisher := pubsub.NewPublisher()

	networksProvider.EXPECT().GetActiveNetworks().Return([]*params.Network{
		{ChainID: 1, IsActive: true},
	}, nil).AnyTimes()
	networksProvider.EXPECT().GetPublisher().Return(networksPublisher).AnyTimes()

	walletDB, err := testutils.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	emptyCollectiblesContainer := &thirdparty.CollectibleOwnershipContainer{
		Items:          []thirdparty.CollectibleIDBalance{},
		NextCursor:     "",
		PreviousCursor: "",
		Provider:       "mockProvider",
	}
	ownershipFetcher := mock_ownership.NewMockCollectibleOwnershipFetcher(mockCtrl)
	ownershipFetcher.EXPECT().FetchCollectibleOwnershipByOwner(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, chainID walletCommon.ChainID, owner common.Address, cursor string, limit int, providerID string) (*thirdparty.CollectibleOwnershipContainer, error) {
			return emptyCollectiblesContainer, nil
		}).AnyTimes()

	multistandardBalancePublisher := pubsub.NewPublisher()
	transferDetectorPublisher := pubsub.NewPublisher()
	blockChainStateProvider := mock_ownership.NewMockBlockChainStateProvider(mockCtrl)
	var estimatedBlockCalls atomic.Int32
	blockChainStateProvider.EXPECT().GetEstimatedBlockTime(gomock.Any(), uint64(1), gomock.Any()).DoAndReturn(
		func(ctx context.Context, chainID uint64, blockNumber uint64) (time.Time, error) {
			estimatedBlockCalls.Add(1)
			return time.Now(), nil
		},
	).AnyTimes()

	publisher := pubsub.NewPublisher()
	logger := zaptest.NewLogger(t).WithOptions(zap.AddCallerSkip(1))

	controller := ownership.NewController(
		ownership.NewOwnershipDB(walletDB),
		accountsProvider,
		accountsPublisher,
		networksProvider,
		multistandardBalancePublisher,
		transferDetectorPublisher,
		blockChainStateProvider,
		ownershipFetcher,
		publisher,
		logger,
	)

	params := ownership.PeriodicalLoaderParams{
		StartDelay:   1 * time.Second,
		LoadInterval: 10 * time.Second,
		LoaderParams: ownership.LoaderParams{
			LoadDelay:  0 * time.Second,
			FetchLimit: 50,
		},
	}
	controller.StartWithLoaderParams(params)
	defer controller.Stop()

	// Ensure initial load completed so ownership timestamp exists.
	require.Eventually(t, func() bool {
		return controller.GetLoaderState(walletCommon.ChainID(1), common.Address(fakeAddress)) == ownership.LoaderStateIdle
	}, 2*time.Second, 50*time.Millisecond)

	otherAddress := common.HexToAddress("0x456")
	msg := transferdetector.EventTransferDetectionFinished{
		ChainID:   1,
		Accounts:  []common.Address{common.Address(fakeAddress)},
		FromBlock: 100,
		ToBlock:   200,
		Events: []eventlog.Event{
			{
				EventKey: eventlog.ERC721Transfer,
				Unpacked: erc721.Erc721Transfer{
					From: common.Address(fakeAddress),
					To:   otherAddress,
					Raw: coretypes.Log{
						BlockNumber: 123,
					},
				},
			},
			{
				EventKey: eventlog.ERC1155TransferSingle,
				Unpacked: erc1155.Erc1155TransferSingle{
					From: common.Address(fakeAddress),
					To:   otherAddress,
					Raw: coretypes.Log{
						BlockNumber: 124,
					},
				},
			},
			{
				EventKey: eventlog.ERC1155TransferBatch,
				Unpacked: erc1155.Erc1155TransferBatch{
					From: common.Address(fakeAddress),
					To:   otherAddress,
					Raw: coretypes.Log{
						BlockNumber: 125,
					},
				},
			},
			{
				EventKey: eventlog.ERC721Transfer,
				Unpacked: struct{}{},
			},
		},
	}
	pubsub.Publish(transferDetectorPublisher, msg)

	require.Eventually(t, func() bool {
		return estimatedBlockCalls.Load() >= 3
	}, 2*time.Second, 50*time.Millisecond)
}
