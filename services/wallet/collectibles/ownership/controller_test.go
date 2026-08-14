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

func TestControllerTriggerLoadUnsupportedChain(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	accountsProvider := mock_ownership.NewMockAccountsProvider(mockCtrl)
	fakeAddress := types.HexToAddress("0x123")
	accountsProvider.EXPECT().GetWalletAddresses().Return([]types.Address{fakeAddress}, nil).AnyTimes()

	accountsPublisher := pubsub.NewPublisher()
	networksProvider := mock_ownership.NewMockNetworksProvider(mockCtrl)
	networksPublisher := pubsub.NewPublisher()

	const unsupportedChainID = walletCommon.ChainID(56)

	networksProvider.EXPECT().GetActiveNetworks().Return([]*params.Network{
		{ChainID: uint64(unsupportedChainID), IsActive: true},
	}, nil).AnyTimes()
	networksProvider.EXPECT().GetPublisher().Return(networksPublisher).AnyTimes()

	walletDB, err := testutils.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	// The fetcher must never be called for a chain the predicate rejects —
	// neither by the periodical loaders nor by the on-demand TriggerLoad path.
	ownershipFetcher := mock_ownership.NewMockCollectibleOwnershipFetcher(mockCtrl)
	ownershipFetcher.EXPECT().FetchCollectibleOwnershipByOwner(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).Times(0)

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
	controller.SetChainSupportedCheck(func(chainID walletCommon.ChainID) bool {
		return chainID != unsupportedChainID
	})

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

	// No loader is created for the unsupported chain
	require.Equal(t, ownership.LoaderStateNotAvailable, controller.GetLoaderState(unsupportedChainID, common.Address(fakeAddress)))

	// On-demand load is a successful no-op, without hitting the fetcher
	err = controller.TriggerLoad(context.Background(), unsupportedChainID, common.Address(fakeAddress))
	require.NoError(t, err)
	require.Equal(t, ownership.LoaderStateNotAvailable, controller.GetLoaderState(unsupportedChainID, common.Address(fakeAddress)))

	controller.Stop()
}

// TestControllerTriggerLoadUnblocksOnCancelledLoad verifies that an on-demand
// load waiting on an already running periodical load doesn't hang when that load
// gets cancelled (the loader is stopped or restarted). The cancelled load
// publishes neither a finished nor an error event, so the waiter has to be woken
// up explicitly — a caller passing a context without a deadline would otherwise
// wait forever.
func TestControllerTriggerLoadUnblocksOnCancelledLoad(t *testing.T) {
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

	// The fetch stays in flight until the load is cancelled, so that the
	// on-demand load has a running load to wait on.
	fetchStarted := make(chan struct{}, 10)
	ownershipFetcher := mock_ownership.NewMockCollectibleOwnershipFetcher(mockCtrl)
	ownershipFetcher.EXPECT().FetchCollectibleOwnershipByOwner(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, chainID walletCommon.ChainID, owner common.Address, cursor string, limit int, providerID string) (*thirdparty.CollectibleOwnershipContainer, error) {
			fetchStarted <- struct{}{}
			<-ctx.Done()
			return nil, ctx.Err()
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

	select {
	case <-fetchStarted:
	case <-time.After(5 * time.Second):
		controller.Stop()
		t.Fatal("the periodical load never started fetching")
	}

	// The on-demand load finds the running load and waits for it to end.
	triggerDone := make(chan error, 1)
	go func() {
		triggerDone <- controller.TriggerLoad(context.Background(), walletCommon.ChainID(1), common.Address(fakeAddress))
	}()

	// Give TriggerLoad time to subscribe and enter the wait.
	time.Sleep(300 * time.Millisecond)

	// Cancels the running load; the waiter must not be left hanging.
	controller.Stop()

	select {
	case triggerErr := <-triggerDone:
		require.ErrorIs(t, triggerErr, context.Canceled)
	case <-time.After(3 * time.Second):
		t.Fatal("TriggerLoad did not return after the load it was waiting on got cancelled")
	}
}

// parkedFetch is a single in-flight fetch the test holds parked until it
// decides that the load owning it may unwind.
type parkedFetch struct {
	release  chan struct{}
	released bool
}

// Release lets the parked fetch return. Safe to call more than once, so that a
// cleanup can release whatever the test body left parked.
func (p *parkedFetch) Release() {
	if p.released {
		return
	}
	p.released = true
	close(p.release)
}

func waitForParkedFetch(t *testing.T, fetches <-chan *parkedFetch, what string) *parkedFetch {
	t.Helper()
	select {
	case fetch := <-fetches:
		return fetch
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
		return nil
	}
}

// TestControllerTriggerLoadIgnoresStaleCancellation verifies that an on-demand
// load waiting on the load of the current loader is not woken up by the
// cancellation of a load belonging to a loader that has already been replaced.
//
// A loader restart (a detected balance change here) cancels the running load and
// installs a replacement loader. The cancelled load unwinds asynchronously, so
// its cancellation can be published long after the restart — possibly while a
// waiter is already blocked on the replacement's load. Load events identify a
// load only by (ChainID, Account), so such a stale event looks exactly like the
// end of the load the waiter cares about, and TriggerLoad reports
// context.Canceled for a request that is in fact still running.
func TestControllerTriggerLoadIgnoresStaleCancellation(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	chainID := walletCommon.ChainID(1)
	fakeAddress := types.HexToAddress("0x123")
	account := common.Address(fakeAddress)

	accountsProvider := mock_ownership.NewMockAccountsProvider(mockCtrl)
	accountsProvider.EXPECT().GetWalletAddresses().Return([]types.Address{fakeAddress}, nil).AnyTimes()

	accountsPublisher := pubsub.NewPublisher()
	networksProvider := mock_ownership.NewMockNetworksProvider(mockCtrl)
	networksPublisher := pubsub.NewPublisher()

	networksProvider.EXPECT().GetActiveNetworks().Return([]*params.Network{
		{ChainID: uint64(chainID), IsActive: true},
	}, nil).AnyTimes()
	networksProvider.EXPECT().GetPublisher().Return(networksPublisher).AnyTimes()

	walletDB, err := testutils.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	ownershipDB := ownership.NewOwnershipDB(walletDB)

	// Record a previous successful load, so that a detected balance change is a
	// reason to refetch (and therefore to restart the loader) instead of being
	// ignored for an account whose ownership was never fetched.
	_, _, _, err = ownershipDB.Update(chainID, account, []thirdparty.CollectibleIDBalance{}, time.Now().Unix()-100)
	require.NoError(t, err)

	emptyCollectiblesContainer := &thirdparty.CollectibleOwnershipContainer{
		Items:          []thirdparty.CollectibleIDBalance{},
		NextCursor:     "",
		PreviousCursor: "",
		Provider:       "mockProvider",
	}

	// Every fetch parks until the test releases it, no matter what happens to its
	// context in the meantime: the test alone decides when each load unwinds,
	// which is what makes the ordering of the events deterministic.
	parkedFetches := make(chan *parkedFetch, 10)
	ownershipFetcher := mock_ownership.NewMockCollectibleOwnershipFetcher(mockCtrl)
	ownershipFetcher.EXPECT().FetchCollectibleOwnershipByOwner(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, chainID walletCommon.ChainID, owner common.Address, cursor string, limit int, providerID string) (*thirdparty.CollectibleOwnershipContainer, error) {
			fetch := &parkedFetch{release: make(chan struct{})}
			parkedFetches <- fetch
			<-fetch.release
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
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

	// The cancellation of the replaced loader's load is what the test needs to
	// observe, to be sure the stale event really was published before asserting
	// that the waiter ignored it. The end of the on-demand load is observed to
	// let every load quiesce before the test returns.
	cancelledCh, cancelledUnsub := pubsub.Subscribe[ownership.EventOwnedCollectiblesLoadCancelled](publisher, 10)
	defer cancelledUnsub()

	finishedCh, finishedUnsub := pubsub.Subscribe[ownership.EventOwnedCollectiblesLoadFinished](publisher, 10)
	defer finishedUnsub()

	// The start delay only applies to loads triggered by a detected change, so
	// the initial load starts fetching right away while the loader the restart
	// installs stays idle until the on-demand load drives it.
	controller.StartWithLoaderParams(
		ownership.PeriodicalLoaderParams{
			StartDelay:   1 * time.Hour,
			LoadInterval: 1 * time.Hour,
			LoaderParams: ownership.LoaderParams{
				LoadDelay:  0 * time.Second,
				FetchLimit: 50,
			},
		},
	)
	defer controller.Stop()

	// The periodical load of the first loader is now parked mid-fetch.
	firstLoadFetch := waitForParkedFetch(t, parkedFetches, "the periodical load to start fetching")
	defer firstLoadFetch.Release()

	// A detected balance change restarts the loader: the load above gets its
	// context cancelled, but it can't notice while it is parked in the fetch.
	pubsub.Publish(multistandardBalancePublisher, multistandardbalance.EventBalanceFetchFinished{
		Key: multistandardbalance.BalancesKey{
			ChainID: uint64(chainID),
			Account: account,
		},
		ResultType:     multistandardfetcher.ResultTypeERC721,
		BalanceChanged: true,
		OldState: multistandardbalance.State{
			FetchedAt: time.Now().Unix() - 1000,
		},
		NewState: multistandardbalance.State{
			FetchedAt: time.Now().Unix(),
		},
	})

	// The replacement loader is in place once the pair reports the delayed state:
	// the replaced loader was fetching, and only the replacement waits out the
	// start delay.
	require.Eventually(t, func() bool {
		return controller.GetLoaderState(chainID, account) == ownership.LoaderStateDelayed
	}, 5*time.Second, 10*time.Millisecond, "the loader was never restarted")

	// The on-demand load drives the replacement loader and waits for its load.
	triggerDone := make(chan error, 1)
	go func() {
		triggerDone <- controller.TriggerLoad(context.Background(), chainID, account)
	}()

	// The only thing that can start a new fetch here is the on-demand load, so a
	// second parked fetch proves that TriggerLoad already subscribed to the load
	// events and is waiting on the load it just started.
	secondLoadFetch := waitForParkedFetch(t, parkedFetches, "the on-demand load to start fetching")
	defer secondLoadFetch.Release()

	// Let the replaced loader's load unwind and publish its cancellation, late.
	firstLoadFetch.Release()
	select {
	case <-cancelledCh:
	case <-time.After(5 * time.Second):
		t.Fatal("the replaced loader's load never published its cancellation")
	}

	// The waiter must ignore it: the load it is waiting on is still running.
	wokenByStaleEvent := false
	select {
	case triggerErr := <-triggerDone:
		wokenByStaleEvent = true
		t.Errorf("TriggerLoad returned (%v) on the cancellation of an already replaced loader's load, while the load it was waiting on is still running", triggerErr)
	case <-time.After(500 * time.Millisecond):
	}

	// The load the waiter really is waiting on now finishes successfully.
	secondLoadFetch.Release()
	select {
	case <-finishedCh:
	case <-time.After(5 * time.Second):
		t.Fatal("the on-demand load never finished")
	}

	if wokenByStaleEvent {
		// The waiter is long gone, there's nothing left to wait for. Returning
		// only once the load it abandoned ended keeps the failure clean.
		return
	}

	select {
	case triggerErr := <-triggerDone:
		require.NoError(t, triggerErr)
	case <-time.After(5 * time.Second):
		t.Fatal("TriggerLoad did not return after the load it was waiting on finished")
	}
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
