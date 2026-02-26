package multistandardbalance_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"

	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	ac "github.com/status-im/status-go/services/wallet/activity/common"
	"github.com/status-im/status-go/services/wallet/multistandardbalance"
	mock_multistandardbalance "github.com/status-im/status-go/services/wallet/multistandardbalance/mock"
	"github.com/status-im/status-go/services/wallet/pendingtxtracker"
	"github.com/status-im/status-go/services/wallet/responses"
	"github.com/status-im/status-go/services/wallet/router/sendtype"
	"github.com/status-im/status-go/services/wallet/walletevent"

	"github.com/stretchr/testify/require"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestController_DebounceTiming(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := mock_multistandardbalance.NewMockStorage(ctrl)
	fetcher := mock_multistandardbalance.NewMockBalanceFetcher(ctrl)
	accountsProvider := mock_multistandardbalance.NewMockAccountsProvider(ctrl)
	accountsPublisher := pubsub.NewPublisher()
	networksProvider := mock_multistandardbalance.NewMockNetworksProvider(ctrl)
	tokenListProvider := mock_multistandardbalance.NewMockTokenListProvider(ctrl)
	collectibleListProvider := mock_multistandardbalance.NewMockCollectiblesListProvider(ctrl)
	lastBlockManager := mock_multistandardbalance.NewMockLastBlockManager(ctrl)
	logger := zap.NewNop()

	// Mock accounts and networks
	address1 := types.Address{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11}
	network1 := &params.Network{ChainID: 1}

	accountsProvider.EXPECT().GetWalletAddresses().Return([]types.Address{address1}, nil).AnyTimes()
	networksProvider.EXPECT().GetActiveNetworks().Return([]*params.Network{network1}, nil).AnyTimes()
	networksProvider.EXPECT().GetPublisher().Return(pubsub.NewPublisher()).AnyTimes()

	// Mock storage calls
	key := multistandardbalance.BalancesKey{
		Account: common.BytesToAddress(address1.Bytes()),
		ChainID: 1,
	}

	oldState := multistandardbalance.State{
		AtBlockNumber: big.NewInt(100),
		AtBlockHash:   common.HexToHash("0x111111"),
		FetchedAt:     time.Now().Unix() - 600, // Old state (600 seconds ago)
	}

	storage.EXPECT().GetNativeBalance(gomock.Any(), key).Return(big.NewInt(1000), oldState, nil).AnyTimes()
	storage.EXPECT().GetERC20Balances(gomock.Any(), key).Return(map[multistandardbalance.ContractAddress]*big.Int{}, oldState, nil).AnyTimes()
	storage.EXPECT().GetERC721Balances(gomock.Any(), key).Return(map[multistandardbalance.ContractAddress]*big.Int{}, oldState, nil).AnyTimes()
	storage.EXPECT().GetERC1155Balances(gomock.Any(), key).Return(map[multistandardbalance.HashableCollectibleID]*big.Int{}, oldState, nil).AnyTimes()
	storage.EXPECT().ClearMissingAccounts(gomock.Any(), gomock.Any()).AnyTimes()
	storage.EXPECT().ClearMissingChains(gomock.Any(), []uint64{1}).AnyTimes()

	// Mock token list provider
	tokenListProvider.EXPECT().GetTokenContractAddresses(uint64(1)).Return([]common.Address{}, nil).AnyTimes()

	// Mock collectible list provider
	collectibleListProvider.EXPECT().GetCollectiblesList(uint64(1), common.BytesToAddress(address1.Bytes())).Return([]multistandardbalance.CollectibleID{}, []multistandardbalance.CollectibleID{}, nil).AnyTimes()

	// Track fetch calls
	var fetchCalls int
	var fetchMutex sync.Mutex
	fetcher.EXPECT().FetchBalances(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, chainID uint64, config interface{}) (<-chan interface{}, error) {
		fetchMutex.Lock()
		fetchCalls++
		fetchMutex.Unlock()
		// Return a closed channel to simulate immediate completion
		ch := make(chan interface{})
		close(ch)
		return ch, nil
	}).AnyTimes()

	getFetchCalls := func() int {
		fetchMutex.Lock()
		defer fetchMutex.Unlock()
		return fetchCalls
	}

	resetFetchCalls := func() {
		fetchMutex.Lock()
		fetchCalls = 0
		fetchMutex.Unlock()
	}

	// Create controller with short debounce time for testing
	const debounceTime = 100 * time.Millisecond
	config := multistandardbalance.ControllerConfig{
		FetchDebounceTime: debounceTime,
		FetchPeriod:       1 * time.Hour, // Long period to avoid interference
	}

	controller := multistandardbalance.NewController(
		config,
		storage,
		fetcher,
		accountsProvider,
		accountsPublisher,
		networksProvider,
		tokenListProvider,
		collectibleListProvider,
		lastBlockManager,
		nil, // walletFeed
		logger,
	)

	resetFetchCalls()

	controller.Start()
	defer controller.Stop()

	// After calling Start, a fetch should be triggered after the debounce period
	require.Eventually(t, func() bool {
		return getFetchCalls() == 1
	}, 2*debounceTime, 10*time.Millisecond, "Expected 1 fetch call after Start")
	resetFetchCalls()

	// Test debounce behavior - multiple rapid calls should only result in one fetch
	controller.TriggerFullFetch()
	controller.TriggerFullFetch()
	controller.TriggerFullFetch()

	// Wait for debounce time
	time.Sleep(2 * debounceTime)
	require.Equal(t, 1, getFetchCalls(), "Expected 1 fetch call after multiple consecutive TriggerFullFetch calls")
	resetFetchCalls()

	// Test that another call after debounce time works
	controller.TriggerFullFetch()
	// Wait for debounce time
	time.Sleep(2 * debounceTime)
	require.Equal(t, 1, getFetchCalls(), "Expected 1 fetch call after multiple consecutive TriggerFullFetch calls")
	resetFetchCalls()
}

func TestController_FetchPeriod(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := mock_multistandardbalance.NewMockStorage(ctrl)
	fetcher := mock_multistandardbalance.NewMockBalanceFetcher(ctrl)
	accountsProvider := mock_multistandardbalance.NewMockAccountsProvider(ctrl)
	accountsPublisher := pubsub.NewPublisher()
	networksProvider := mock_multistandardbalance.NewMockNetworksProvider(ctrl)
	tokenListProvider := mock_multistandardbalance.NewMockTokenListProvider(ctrl)
	collectibleListProvider := mock_multistandardbalance.NewMockCollectiblesListProvider(ctrl)
	lastBlockManager := mock_multistandardbalance.NewMockLastBlockManager(ctrl)
	logger := zap.NewNop()

	// Mock accounts and networks
	address1 := types.Address{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11}
	network1 := &params.Network{ChainID: 1}

	accountsProvider.EXPECT().GetWalletAddresses().Return([]types.Address{address1}, nil).AnyTimes()
	networksProvider.EXPECT().GetActiveNetworks().Return([]*params.Network{network1}, nil).AnyTimes()
	networksProvider.EXPECT().GetPublisher().Return(pubsub.NewPublisher()).AnyTimes()
	lastBlockManager.EXPECT().SetLatestBlockNumber(uint64(1), uint64(100)).AnyTimes()

	// Mock storage calls to return recently fetched state
	key := multistandardbalance.BalancesKey{
		Account: common.BytesToAddress(address1.Bytes()),
		ChainID: 1,
	}

	oldState := multistandardbalance.State{
		AtBlockNumber: big.NewInt(100),
		AtBlockHash:   common.HexToHash("0x111111"),
		FetchedAt:     time.Now().Unix() - 600, // Old state (600 seconds ago)
	}

	storage.EXPECT().GetNativeBalance(gomock.Any(), key).Return(big.NewInt(1000), oldState, nil).AnyTimes()
	storage.EXPECT().GetERC20Balances(gomock.Any(), key).Return(map[multistandardbalance.ContractAddress]*big.Int{}, oldState, nil).AnyTimes()
	storage.EXPECT().GetERC721Balances(gomock.Any(), key).Return(map[multistandardbalance.ContractAddress]*big.Int{}, oldState, nil).AnyTimes()
	storage.EXPECT().GetERC1155Balances(gomock.Any(), key).Return(map[multistandardbalance.HashableCollectibleID]*big.Int{}, oldState, nil).AnyTimes()
	storage.EXPECT().ClearMissingAccounts(gomock.Any(), gomock.Any()).AnyTimes()
	storage.EXPECT().ClearMissingChains(gomock.Any(), []uint64{1}).AnyTimes()

	// Mock token list provider
	tokenListProvider.EXPECT().GetTokenContractAddresses(uint64(1)).Return([]common.Address{}, nil).AnyTimes()

	// Mock collectible list provider
	collectibleListProvider.EXPECT().GetCollectiblesList(uint64(1), common.BytesToAddress(address1.Bytes())).Return([]multistandardbalance.CollectibleID{}, []multistandardbalance.CollectibleID{}, nil).AnyTimes()

	// Track fetch calls
	var fetchCalls int
	var fetchMutex sync.Mutex
	fetcher.EXPECT().FetchBalances(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, chainID uint64, config interface{}) (<-chan interface{}, error) {
		fetchMutex.Lock()
		fetchCalls++
		fetchMutex.Unlock()
		// Return a closed channel to simulate immediate completion
		ch := make(chan interface{})
		close(ch)
		return ch, nil
	}).AnyTimes()

	getFetchCalls := func() int {
		fetchMutex.Lock()
		defer fetchMutex.Unlock()
		return fetchCalls
	}

	resetFetchCalls := func() {
		fetchMutex.Lock()
		fetchCalls = 0
		fetchMutex.Unlock()
	}

	// Create controller with short fetch period for testing
	const fetchPeriod = 500 * time.Millisecond
	config := multistandardbalance.ControllerConfig{
		FetchDebounceTime: 0 * time.Millisecond,
		FetchPeriod:       fetchPeriod,
	}

	controller := multistandardbalance.NewController(
		config,
		storage,
		fetcher,
		accountsProvider,
		accountsPublisher,
		networksProvider,
		tokenListProvider,
		collectibleListProvider,
		lastBlockManager,
		nil, // walletFeed
		logger,
	)

	resetFetchCalls()

	controller.Start()
	defer controller.Stop()

	// After calling Start, a fetch should be triggered immediately
	require.Eventually(t, func() bool {
		return getFetchCalls() == 1
	}, 100*time.Millisecond, 10*time.Millisecond, "Expected 1 fetch call after Start")
	resetFetchCalls()

	// Wait for period time
	time.Sleep(config.FetchPeriod)
	require.Eventually(t, func() bool {
		return getFetchCalls() == 1
	}, 100*time.Millisecond, 10*time.Millisecond, "Expected 1 fetch call after fetch period")
	resetFetchCalls()

	// Wait a bit less than the period time
	time.Sleep(config.FetchPeriod - 100*time.Millisecond)
	controller.TriggerFullFetch()
	require.Eventually(t, func() bool {
		return getFetchCalls() == 1
	}, 100*time.Millisecond, 10*time.Millisecond, "Expected 1 fetch call after TriggerFullFetch()")
	resetFetchCalls()

	// The periodic ticker continues independently, so after ~100ms more (to complete the period from last periodic fetch),
	// another periodic fetch should happen
	time.Sleep(200 * time.Millisecond)
	require.Eventually(t, func() bool {
		return getFetchCalls() >= 1
	}, 100*time.Millisecond, 10*time.Millisecond, "Expected at least 1 fetch call from periodic ticker")
	resetFetchCalls()
}

func TestController_BasicFlow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := mock_multistandardbalance.NewMockStorage(ctrl)
	fetcher := mock_multistandardbalance.NewMockBalanceFetcher(ctrl)
	accountsProvider := mock_multistandardbalance.NewMockAccountsProvider(ctrl)
	accountsPublisher := pubsub.NewPublisher()
	networksProvider := mock_multistandardbalance.NewMockNetworksProvider(ctrl)
	tokenListProvider := mock_multistandardbalance.NewMockTokenListProvider(ctrl)
	collectibleListProvider := mock_multistandardbalance.NewMockCollectiblesListProvider(ctrl)
	lastBlockManager := mock_multistandardbalance.NewMockLastBlockManager(ctrl)
	logger := zap.NewNop()

	// Mock accounts and networks
	address1 := types.Address{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11}
	address2 := types.Address{0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22}
	network1 := &params.Network{ChainID: 1}
	network2 := &params.Network{ChainID: 2}

	accountsProvider.EXPECT().GetWalletAddresses().Return([]types.Address{address1, address2}, nil).AnyTimes()
	networksProvider.EXPECT().GetActiveNetworks().Return([]*params.Network{network1, network2}, nil).AnyTimes()
	networksProvider.EXPECT().GetPublisher().Return(pubsub.NewPublisher()).AnyTimes()

	// Create keys for different account/chain combinations
	key1_1 := multistandardbalance.BalancesKey{
		Account: common.BytesToAddress(address1.Bytes()),
		ChainID: 1,
	}
	key1_2 := multistandardbalance.BalancesKey{
		Account: common.BytesToAddress(address1.Bytes()),
		ChainID: 2,
	}
	key2_1 := multistandardbalance.BalancesKey{
		Account: common.BytesToAddress(address2.Bytes()),
		ChainID: 1,
	}
	key2_2 := multistandardbalance.BalancesKey{
		Account: common.BytesToAddress(address2.Bytes()),
		ChainID: 2,
	}

	// Create different states: recent, old, and never fetched
	recentState := multistandardbalance.State{
		AtBlockNumber: big.NewInt(100),
		AtBlockHash:   common.HexToHash("0x111111"),
		FetchedAt:     time.Now().Unix(), // Recent state
	}

	oldState := multistandardbalance.State{
		AtBlockNumber: big.NewInt(200),
		AtBlockHash:   common.HexToHash("0x222222"),
		FetchedAt:     time.Now().Unix() - 600, // Old state (600 seconds ago)
	}

	neverFetchedState := multistandardbalance.State{
		AtBlockNumber: big.NewInt(0),
		AtBlockHash:   common.Hash{},
		FetchedAt:     multistandardbalance.NeverFetched,
	}

	// Mock storage calls with different states for different keys
	// key1_1: recent state (should NOT fetch)
	storage.EXPECT().GetNativeBalance(gomock.Any(), key1_1).Return(big.NewInt(1000), recentState, nil).AnyTimes()
	storage.EXPECT().GetERC20Balances(gomock.Any(), key1_1).Return(map[multistandardbalance.ContractAddress]*big.Int{}, recentState, nil).AnyTimes()
	storage.EXPECT().GetERC721Balances(gomock.Any(), key1_1).Return(map[multistandardbalance.ContractAddress]*big.Int{}, recentState, nil).AnyTimes()
	storage.EXPECT().GetERC1155Balances(gomock.Any(), key1_1).Return(map[multistandardbalance.HashableCollectibleID]*big.Int{}, recentState, nil).AnyTimes()

	// key1_2: old state (should fetch)
	storage.EXPECT().GetNativeBalance(gomock.Any(), key1_2).Return(big.NewInt(2000), oldState, nil).AnyTimes()
	storage.EXPECT().GetERC20Balances(gomock.Any(), key1_2).Return(map[multistandardbalance.ContractAddress]*big.Int{}, oldState, nil).AnyTimes()
	storage.EXPECT().GetERC721Balances(gomock.Any(), key1_2).Return(map[multistandardbalance.ContractAddress]*big.Int{}, oldState, nil).AnyTimes()
	storage.EXPECT().GetERC1155Balances(gomock.Any(), key1_2).Return(map[multistandardbalance.HashableCollectibleID]*big.Int{}, oldState, nil).AnyTimes()

	// key2_1: never fetched (should fetch)
	storage.EXPECT().GetNativeBalance(gomock.Any(), key2_1).Return((*big.Int)(nil), neverFetchedState, nil).AnyTimes()
	storage.EXPECT().GetERC20Balances(gomock.Any(), key2_1).Return((map[multistandardbalance.ContractAddress]*big.Int)(nil), neverFetchedState, nil).AnyTimes()
	storage.EXPECT().GetERC721Balances(gomock.Any(), key2_1).Return((map[multistandardbalance.ContractAddress]*big.Int)(nil), neverFetchedState, nil).AnyTimes()
	storage.EXPECT().GetERC1155Balances(gomock.Any(), key2_1).Return((map[multistandardbalance.HashableCollectibleID]*big.Int)(nil), neverFetchedState, nil).AnyTimes()

	// key2_2: recent state (should NOT fetch)
	storage.EXPECT().GetNativeBalance(gomock.Any(), key2_2).Return(big.NewInt(4000), recentState, nil).AnyTimes()
	storage.EXPECT().GetERC20Balances(gomock.Any(), key2_2).Return(map[multistandardbalance.ContractAddress]*big.Int{}, recentState, nil).AnyTimes()
	storage.EXPECT().GetERC721Balances(gomock.Any(), key2_2).Return(map[multistandardbalance.ContractAddress]*big.Int{}, recentState, nil).AnyTimes()
	storage.EXPECT().GetERC1155Balances(gomock.Any(), key2_2).Return(map[multistandardbalance.HashableCollectibleID]*big.Int{}, recentState, nil).AnyTimes()

	// Mock UpdateNativeBalance to return success (needed for finished events)
	storage.EXPECT().UpdateNativeBalance(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(false, multistandardbalance.State{}, nil).AnyTimes()

	// Mock ERC20, ERC721, and ERC1155 storage update methods
	storage.EXPECT().UpdateERC20Balances(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(false, multistandardbalance.State{}, nil).AnyTimes()
	storage.EXPECT().UpdateERC721Balances(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(false, multistandardbalance.State{}, nil).AnyTimes()
	storage.EXPECT().UpdateERC1155Balances(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(false, multistandardbalance.State{}, nil).AnyTimes()

	// Mock LastBlockManager
	lastBlockManager.EXPECT().SetLatestBlockNumber(gomock.Any(), gomock.Any()).AnyTimes()

	storage.EXPECT().ClearMissingAccounts(gomock.Any(), gomock.Any()).AnyTimes()
	storage.EXPECT().ClearMissingChains(gomock.Any(), []uint64{1, 2}).AnyTimes()

	// Create some test contracts for ERC20, ERC721, and ERC1155
	erc20Contract1 := common.HexToAddress("0x1234567890123456789012345678901234567890")
	erc20Contract2 := common.HexToAddress("0x2345678901234567890123456789012345678901")
	erc721Contract1 := common.HexToAddress("0x3456789012345678901234567890123456789012")
	erc721Contract2 := common.HexToAddress("0x4567890123456789012345678901234567890123")
	erc1155Contract1 := common.HexToAddress("0x5678901234567890123456789012345678901234")
	erc1155Contract2 := common.HexToAddress("0x6789012345678901234567890123456789012345")

	// Mock token list provider to return ERC20 contracts
	tokenListProvider.EXPECT().GetTokenContractAddresses(uint64(1)).Return([]common.Address{erc20Contract1, erc20Contract2}, nil).AnyTimes()
	tokenListProvider.EXPECT().GetTokenContractAddresses(uint64(2)).Return([]common.Address{erc20Contract1, erc20Contract2}, nil).AnyTimes()

	// Mock collectible list provider to return ERC721 and ERC1155 collectibles
	erc721Collectibles := []multistandardbalance.CollectibleID{
		{ContractAddress: erc721Contract1, TokenID: big.NewInt(1)},
		{ContractAddress: erc721Contract2, TokenID: big.NewInt(2)},
	}
	erc1155Collectibles := []multistandardbalance.CollectibleID{
		{ContractAddress: erc1155Contract1, TokenID: big.NewInt(100)},
		{ContractAddress: erc1155Contract2, TokenID: big.NewInt(200)},
	}

	collectibleListProvider.EXPECT().GetCollectiblesList(uint64(1), common.BytesToAddress(address1.Bytes())).Return(erc721Collectibles, erc1155Collectibles, nil).AnyTimes()
	collectibleListProvider.EXPECT().GetCollectiblesList(uint64(2), common.BytesToAddress(address1.Bytes())).Return(erc721Collectibles, erc1155Collectibles, nil).AnyTimes()
	collectibleListProvider.EXPECT().GetCollectiblesList(uint64(1), common.BytesToAddress(address2.Bytes())).Return(erc721Collectibles, erc1155Collectibles, nil).AnyTimes()
	collectibleListProvider.EXPECT().GetCollectiblesList(uint64(2), common.BytesToAddress(address2.Bytes())).Return(erc721Collectibles, erc1155Collectibles, nil).AnyTimes()

	// Define a struct to track fetch calls with all their information
	type FetchCall struct {
		ChainID uint64
		Config  multistandardfetcher.FetchConfig
	}

	// Track fetch calls and verify which chains are fetched
	var fetchCalls []FetchCall
	var fetchMutex sync.Mutex
	fetcher.EXPECT().FetchBalances(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, chainID uint64, config multistandardfetcher.FetchConfig) (<-chan multistandardfetcher.FetchResult, error) {
		fetchMutex.Lock()
		fetchCalls = append(fetchCalls, FetchCall{
			ChainID: chainID,
			Config:  config,
		})
		fetchMutex.Unlock()

		// Return a channel with some mock results to trigger finished events
		ch := make(chan multistandardfetcher.FetchResult, 50)
		go func() {
			defer close(ch)

			// Send native balance results
			for _, addr := range config.Native {
				ch <- multistandardfetcher.FetchResult{
					ResultType: multistandardfetcher.ResultTypeNative,
					Result: multistandardfetcher.NativeResult{
						Account:       addr,
						Result:        big.NewInt(1000000000000000000), // 1 ETH
						Err:           nil,
						AtBlockNumber: big.NewInt(12345),
						AtBlockHash:   common.HexToHash("0x123"),
					},
				}
			}

			// Send ERC20 balance results
			for addr, contracts := range config.ERC20 {
				erc20Results := make(map[multistandardfetcher.ContractAddress]*big.Int)
				for _, contract := range contracts {
					erc20Results[contract] = big.NewInt(500000000000000000) // 0.5 tokens
				}
				ch <- multistandardfetcher.FetchResult{
					ResultType: multistandardfetcher.ResultTypeERC20,
					Result: multistandardfetcher.ERC20Result{
						Account:       addr,
						Results:       erc20Results,
						Err:           nil,
						AtBlockNumber: big.NewInt(12345),
						AtBlockHash:   common.HexToHash("0x123"),
					},
				}
			}

			// Send ERC721 balance results
			for addr, contracts := range config.ERC721 {
				erc721Results := make(map[multistandardfetcher.ContractAddress]*big.Int)
				for _, contract := range contracts {
					erc721Results[contract] = big.NewInt(1) // 1 NFT
				}
				ch <- multistandardfetcher.FetchResult{
					ResultType: multistandardfetcher.ResultTypeERC721,
					Result: multistandardfetcher.ERC721Result{
						Account:       addr,
						Results:       erc721Results,
						Err:           nil,
						AtBlockNumber: big.NewInt(12345),
						AtBlockHash:   common.HexToHash("0x123"),
					},
				}
			}

			// Send ERC1155 balance results
			for addr, collectibles := range config.ERC1155 {
				erc1155Results := make(map[multistandardfetcher.HashableCollectibleID]*big.Int)
				for _, collectible := range collectibles {
					erc1155Results[collectible.ToHashableCollectibleID()] = big.NewInt(10) // 10 tokens
				}
				ch <- multistandardfetcher.FetchResult{
					ResultType: multistandardfetcher.ResultTypeERC1155,
					Result: multistandardfetcher.ERC1155Result{
						Account:       addr,
						Results:       erc1155Results,
						Err:           nil,
						AtBlockNumber: big.NewInt(12345),
						AtBlockHash:   common.HexToHash("0x123"),
					},
				}
			}
		}()
		return ch, nil
	}).AnyTimes()

	getFetchCalls := func() []FetchCall {
		fetchMutex.Lock()
		defer fetchMutex.Unlock()
		result := make([]FetchCall, len(fetchCalls))
		copy(result, fetchCalls)
		return result
	}

	getChainIDs := func() []uint64 {
		fetchMutex.Lock()
		defer fetchMutex.Unlock()
		result := make([]uint64, len(fetchCalls))
		for i, call := range fetchCalls {
			result[i] = call.ChainID
		}
		return result
	}

	resetFetchCalls := func() {
		fetchMutex.Lock()
		fetchCalls = nil
		fetchMutex.Unlock()
	}

	// Create controller with a short fetch period to make old state detection work
	config := multistandardbalance.ControllerConfig{
		FetchDebounceTime: 0 * time.Millisecond,
		FetchPeriod:       1 * time.Second, // Short period so old state (600s ago) is considered old
	}

	controller := multistandardbalance.NewController(
		config,
		storage,
		fetcher,
		accountsProvider,
		accountsPublisher,
		networksProvider,
		tokenListProvider,
		collectibleListProvider,
		lastBlockManager,
		nil, // walletFeed
		logger,
	)

	startedCh, startedUnsubFn := pubsub.Subscribe[multistandardbalance.EventBalanceFetchStarted](controller.GetPublisher(), 10)
	finishedCh, finishedUnsubFn := pubsub.Subscribe[multistandardbalance.EventBalanceFetchFinished](controller.GetPublisher(), 10)

	defer func() {
		startedUnsubFn()
		finishedUnsubFn()
	}()

	resetFetchCalls()

	// Start the controller to enable the trigger channel
	controller.Start()
	defer controller.Stop()

	// Track events with thread-safe collection
	var startedEvents []multistandardbalance.EventBalanceFetchStarted
	var finishedEvents []multistandardbalance.EventBalanceFetchFinished
	var eventsMutex sync.Mutex

	// Collect started events
	go func() {
		for ev := range startedCh {
			eventsMutex.Lock()
			startedEvents = append(startedEvents, ev)
			eventsMutex.Unlock()
		}
	}()

	// Collect finished events
	go func() {
		for ev := range finishedCh {
			eventsMutex.Lock()
			finishedEvents = append(finishedEvents, ev)
			eventsMutex.Unlock()
		}
	}()

	// Wait for initial fetch to complete
	require.Eventually(t, func() bool {
		eventsMutex.Lock()
		defer eventsMutex.Unlock()
		return len(startedEvents) >= 2 // Should have 2 started events (one per chain)
	}, 1*time.Second, 10*time.Millisecond, "Expected at least 2 EventBalanceFetchStarted events")

	// Wait for finished events (2 accounts × 2 chains × 4 balance types = 16 events)
	require.Eventually(t, func() bool {
		eventsMutex.Lock()
		defer eventsMutex.Unlock()
		return len(finishedEvents) >= 16 // Should have 16 finished events (2 accounts × 2 chains × 4 balance types)
	}, 1*time.Second, 10*time.Millisecond, "Expected at least 16 EventBalanceFetchFinished events")

	// Verify the started events contain the expected chains
	eventsMutex.Lock()
	require.Len(t, startedEvents, 2, "Expected exactly 2 started events")
	chainIDs := make([]uint64, len(startedEvents))
	for i, ev := range startedEvents {
		chainIDs[i] = ev.ChainID
	}
	eventsMutex.Unlock()

	require.Contains(t, chainIDs, uint64(1), "Expected chain 1 in started events")
	require.Contains(t, chainIDs, uint64(2), "Expected chain 2 in started events")

	// Verify finished events have the expected keys and result types
	eventsMutex.Lock()
	require.Len(t, finishedEvents, 16, "Expected exactly 16 finished events (2 accounts × 2 chains × 4 balance types)")

	// Check that we have finished events for both chains and verify specific values
	finishedChainIDs := make(map[uint64]bool)
	resultTypes := make(map[multistandardfetcher.ResultType]int)
	accountChainPairs := make(map[string]bool) // "account:chain" -> bool
	expectedBlockNumber := big.NewInt(12345)
	expectedBlockHash := common.HexToHash("0x123")

	for _, ev := range finishedEvents {
		finishedChainIDs[ev.Key.ChainID] = true
		resultTypes[ev.ResultType]++

		// Track account:chain pairs
		accountChainKey := fmt.Sprintf("%s:%d", ev.Key.Account.Hex(), ev.Key.ChainID)
		accountChainPairs[accountChainKey] = true

		// Verify the event contains expected values
		require.Contains(t, []uint64{1, 2}, ev.Key.ChainID, "Event should be for chain 1 or 2")
		require.Contains(t, []common.Address{common.BytesToAddress(address1.Bytes()), common.BytesToAddress(address2.Bytes())}, ev.Key.Account, "Event should be for address1 or address2")
		require.False(t, ev.BalanceChanged, "BalanceChanged should be false (no change from mock storage)")
		require.NotNil(t, ev.NewState, "NewState should not be nil")
		require.Equal(t, expectedBlockNumber, ev.NewState.AtBlockNumber, "NewState.AtBlockNumber should match expected value")
		require.Equal(t, expectedBlockHash, ev.NewState.AtBlockHash, "NewState.AtBlockHash should match expected value")
		require.True(t, ev.NewState.FetchedAt > 0, "NewState.FetchedAt should be positive timestamp")
	}
	eventsMutex.Unlock()

	require.True(t, finishedChainIDs[1], "Expected finished events for chain 1")
	require.True(t, finishedChainIDs[2], "Expected finished events for chain 2")

	// Verify we have events for all balance types (4 events per balance type: 2 accounts × 2 chains)
	require.Equal(t, 4, resultTypes[multistandardfetcher.ResultTypeNative], "Expected 4 native balance events")
	require.Equal(t, 4, resultTypes[multistandardfetcher.ResultTypeERC20], "Expected 4 ERC20 balance events")
	require.Equal(t, 4, resultTypes[multistandardfetcher.ResultTypeERC721], "Expected 4 ERC721 balance events")
	require.Equal(t, 4, resultTypes[multistandardfetcher.ResultTypeERC1155], "Expected 4 ERC1155 balance events")

	// Verify we have events for all expected account:chain combinations
	expectedAccountChainPairs := []string{
		fmt.Sprintf("%s:1", address1.Hex()),
		fmt.Sprintf("%s:2", address1.Hex()),
		fmt.Sprintf("%s:1", address2.Hex()),
		fmt.Sprintf("%s:2", address2.Hex()),
	}
	for _, expectedPair := range expectedAccountChainPairs {
		require.True(t, accountChainPairs[expectedPair], "Expected events for account:chain pair %s", expectedPair)
	}

	// Verify fetch configs contain the expected contracts and collectibles
	fetchCalls = getFetchCalls()
	require.Len(t, fetchCalls, 2, "Expected 2 fetch calls (one per chain)")

	// Verify each fetch call has the correct structure
	for _, call := range fetchCalls {
		require.Contains(t, []uint64{1, 2}, call.ChainID, "Fetch call should be for chain 1 or 2")

		config := call.Config

		// Verify native addresses (should be both addresses for both chains)
		require.Len(t, config.Native, 2, "Config should contain 2 native addresses")
		require.Contains(t, config.Native, common.BytesToAddress(address1.Bytes()), "Config should contain address1")
		require.Contains(t, config.Native, common.BytesToAddress(address2.Bytes()), "Config should contain address2")

		// Verify ERC20 contracts (should be 2 contracts per address)
		require.Len(t, config.ERC20, 2, "Config should contain ERC20 for 2 addresses")
		for _, addr := range config.Native {
			erc20Contracts, exists := config.ERC20[addr]
			require.True(t, exists, "ERC20 config should exist for address %s", addr.Hex())
			require.Len(t, erc20Contracts, 2, "Should have 2 ERC20 contracts for address %s", addr.Hex())
			require.Contains(t, erc20Contracts, erc20Contract1, "Should contain erc20Contract1")
			require.Contains(t, erc20Contracts, erc20Contract2, "Should contain erc20Contract2")
		}

		// Verify ERC721 contracts (should be 2 contracts per address)
		require.Len(t, config.ERC721, 2, "Config should contain ERC721 for 2 addresses")
		for _, addr := range config.Native {
			erc721Contracts, exists := config.ERC721[addr]
			require.True(t, exists, "ERC721 config should exist for address %s", addr.Hex())
			require.Len(t, erc721Contracts, 2, "Should have 2 ERC721 contracts for address %s", addr.Hex())
			require.Contains(t, erc721Contracts, erc721Contract1, "Should contain erc721Contract1")
			require.Contains(t, erc721Contracts, erc721Contract2, "Should contain erc721Contract2")
		}

		// Verify ERC1155 collectibles (should be 2 collectibles per address)
		require.Len(t, config.ERC1155, 2, "Config should contain ERC1155 for 2 addresses")
		for _, addr := range config.Native {
			erc1155Collectibles, exists := config.ERC1155[addr]
			require.True(t, exists, "ERC1155 config should exist for address %s", addr.Hex())
			require.Len(t, erc1155Collectibles, 2, "Should have 2 ERC1155 collectibles for address %s", addr.Hex())

			// Convert expected collectibles to HashableCollectibleID for comparison
			var tokenID100, tokenID200 [32]byte
			big.NewInt(100).FillBytes(tokenID100[:])
			big.NewInt(200).FillBytes(tokenID200[:])

			expectedCollectibles := []multistandardfetcher.HashableCollectibleID{
				{ContractAddress: erc1155Contract1, TokenID: tokenID100},
				{ContractAddress: erc1155Contract2, TokenID: tokenID200},
			}
			for _, expected := range expectedCollectibles {
				found := false
				for _, actual := range erc1155Collectibles {
					actualHashable := actual.ToHashableCollectibleID()
					if actualHashable.ContractAddress == expected.ContractAddress && actualHashable.TokenID == expected.TokenID {
						found = true
						break
					}
				}
				require.True(t, found, "Should contain ERC1155 collectible %+v", expected)
			}
		}
	}

	// After calling Start, a fetch should be triggered immediately
	require.Eventually(t, func() bool {
		chains := getChainIDs()
		return len(chains) == 2 // Should fetch both chains initially
	}, 100*time.Millisecond, 10*time.Millisecond, "Expected 2 chains to be fetched after Start")
	resetFetchCalls()

	// Trigger fetch with empty config - should only fetch chains with old/never fetched states
	controller.TriggerFetchWithConfig(make(multistandardbalance.FetchConfig))

	// Wait for the fetch to complete
	require.Eventually(t, func() bool {
		chains := getChainIDs()
		return len(chains) == 2 // Should fetch chains 1 and 2
	}, 100*time.Millisecond, 10*time.Millisecond, "Expected 2 chains to be fetched with TriggerFetchWithConfig")

	// Verify that only chains with old/never fetched states were fetched
	actualCalls := getFetchCalls()
	actualChains := getChainIDs()
	// Should have fetched chains 1 and 2 (for key1_2 and key2_1 which have old/never fetched states)
	// Chain 1: key2_1 has never fetched state
	// Chain 2: key1_2 has old state
	require.Len(t, actualChains, 2, "Expected 2 chains to be fetched")
	require.Contains(t, actualChains, uint64(1), "Expected chain 1 to be fetched (key2_1 has never fetched state)")
	require.Contains(t, actualChains, uint64(2), "Expected chain 2 to be fetched (key1_2 has old state)")

	// Verify the configs contain only the expected accounts
	require.Len(t, actualCalls, 2, "Expected 2 fetch calls to be made")

	// Find configs for each chain
	var config1, config2 multistandardfetcher.FetchConfig
	for _, call := range actualCalls {
		if call.ChainID == 1 {
			config1 = call.Config
		} else if call.ChainID == 2 {
			config2 = call.Config
		}
	}

	// Chain 1 should only have address2 (key2_1 has never fetched state)
	require.Len(t, config1.Native, 1, "Chain 1 should have 1 native account")
	require.Contains(t, config1.Native, common.BytesToAddress(address2.Bytes()), "Chain 1 should include address2")

	// Chain 2 should only have address1 (key1_2 has old state)
	require.Len(t, config2.Native, 1, "Chain 2 should have 1 native account")
	require.Contains(t, config2.Native, common.BytesToAddress(address1.Bytes()), "Chain 2 should include address1")

	// Reset and test with full fetch - should fetch all chains
	resetFetchCalls()

	controller.TriggerFullFetch()

	// Wait for the fetch to complete
	require.Eventually(t, func() bool {
		chains := getChainIDs()
		return len(chains) == 2 // Should fetch both chains when full fetch
	}, 100*time.Millisecond, 10*time.Millisecond, "Expected 2 chains to be fetched with TriggerFullFetch")

	// Verify that all chains were fetched with TriggerFullFetch
	actualCallsForced := getFetchCalls()
	actualChainsForced := getChainIDs()
	// Should have fetched both chains with TriggerFullFetch
	require.Len(t, actualChainsForced, 2, "Expected 2 chains to be fetched with TriggerFullFetch")
	require.Contains(t, actualChainsForced, uint64(1), "Expected chain 1 to be fetched with TriggerFullFetch")
	require.Contains(t, actualChainsForced, uint64(2), "Expected chain 2 to be fetched with TriggerFullFetch")

	// Verify the configs contain all accounts with TriggerFullFetch
	require.Len(t, actualCallsForced, 2, "Expected 2 fetch calls to be made with TriggerFullFetch")

	// Find configs for each chain
	var config1Forced, config2Forced multistandardfetcher.FetchConfig
	for _, call := range actualCallsForced {
		if call.ChainID == 1 {
			config1Forced = call.Config
		} else if call.ChainID == 2 {
			config2Forced = call.Config
		}
	}

	// With TriggerFullFetch, both chains should have both accounts
	require.Len(t, config1Forced.Native, 2, "Chain 1 should have 2 native accounts with TriggerFullFetch")
	require.Contains(t, config1Forced.Native, common.BytesToAddress(address1.Bytes()), "Chain 1 should include address1 with TriggerFullFetch")
	require.Contains(t, config1Forced.Native, common.BytesToAddress(address2.Bytes()), "Chain 1 should include address2 with TriggerFullFetch")

	require.Len(t, config2Forced.Native, 2, "Chain 2 should have 2 native accounts with TriggerFullFetch")
	require.Contains(t, config2Forced.Native, common.BytesToAddress(address1.Bytes()), "Chain 2 should include address1 with TriggerFullFetch")
	require.Contains(t, config2Forced.Native, common.BytesToAddress(address2.Bytes()), "Chain 2 should include address2 with TriggerFullFetch")
}

func newTestController(
	t *testing.T,
	walletFeed *event.Feed,
	cfg multistandardbalance.ControllerConfig,
) (
	controller *multistandardbalance.Controller,
	fetchCalls *[]FetchCallRecord,
	fetchMutex *sync.Mutex,
	fromAddr types.Address,
	toAddr types.Address,
) {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	storage := mock_multistandardbalance.NewMockStorage(ctrl)
	balanceFetcher := mock_multistandardbalance.NewMockBalanceFetcher(ctrl)
	accountsProvider := mock_multistandardbalance.NewMockAccountsProvider(ctrl)
	accountsPublisher := pubsub.NewPublisher()
	networksProvider := mock_multistandardbalance.NewMockNetworksProvider(ctrl)
	tokenListProvider := mock_multistandardbalance.NewMockTokenListProvider(ctrl)
	collectibleListProvider := mock_multistandardbalance.NewMockCollectiblesListProvider(ctrl)
	lastBlockManager := mock_multistandardbalance.NewMockLastBlockManager(ctrl)

	fromAddr = types.Address{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11}
	toAddr = types.Address{0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22}

	accountsProvider.EXPECT().GetWalletAddresses().Return([]types.Address{fromAddr, toAddr}, nil).AnyTimes()
	networksProvider.EXPECT().GetActiveNetworks().Return([]*params.Network{{ChainID: 1}, {ChainID: 10}}, nil).AnyTimes()
	networksProvider.EXPECT().GetPublisher().Return(pubsub.NewPublisher()).AnyTimes()

	recentState := multistandardbalance.State{FetchedAt: time.Now().Unix()}
	storage.EXPECT().GetNativeBalance(gomock.Any(), gomock.Any()).Return(big.NewInt(0), recentState, nil).AnyTimes()
	storage.EXPECT().GetERC20Balances(gomock.Any(), gomock.Any()).Return(map[multistandardbalance.ContractAddress]*big.Int{}, recentState, nil).AnyTimes()
	storage.EXPECT().GetERC721Balances(gomock.Any(), gomock.Any()).Return(map[multistandardbalance.ContractAddress]*big.Int{}, recentState, nil).AnyTimes()
	storage.EXPECT().GetERC1155Balances(gomock.Any(), gomock.Any()).Return(map[multistandardbalance.HashableCollectibleID]*big.Int{}, recentState, nil).AnyTimes()
	storage.EXPECT().ClearMissingAccounts(gomock.Any(), gomock.Any()).AnyTimes()
	storage.EXPECT().ClearMissingChains(gomock.Any(), gomock.Any()).AnyTimes()

	tokenListProvider.EXPECT().GetTokenContractAddresses(gomock.Any()).Return([]common.Address{}, nil).AnyTimes()
	collectibleListProvider.EXPECT().GetCollectiblesList(gomock.Any(), gomock.Any()).Return([]multistandardbalance.CollectibleID{}, []multistandardbalance.CollectibleID{}, nil).AnyTimes()
	lastBlockManager.EXPECT().SetLatestBlockNumber(gomock.Any(), gomock.Any()).AnyTimes()

	calls := make([]FetchCallRecord, 0)
	mu := &sync.Mutex{}
	balanceFetcher.EXPECT().FetchBalances(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, chainID uint64, fetchCfg multistandardfetcher.FetchConfig) (<-chan multistandardfetcher.FetchResult, error) {
			mu.Lock()
			calls = append(calls, FetchCallRecord{ChainID: chainID, Config: fetchCfg})
			mu.Unlock()
			ch := make(chan multistandardfetcher.FetchResult)
			close(ch)
			return ch, nil
		},
	).AnyTimes()

	controller = multistandardbalance.NewController(
		cfg, storage, balanceFetcher, accountsProvider, accountsPublisher,
		networksProvider, tokenListProvider, collectibleListProvider,
		lastBlockManager, walletFeed, zap.NewNop(),
	)

	return controller, &calls, mu, fromAddr, toAddr
}

type FetchCallRecord struct {
	ChainID uint64
	Config  multistandardfetcher.FetchConfig
}

func getFetchCallsCopy(calls *[]FetchCallRecord, mu *sync.Mutex) []FetchCallRecord {
	mu.Lock()
	defer mu.Unlock()
	result := make([]FetchCallRecord, len(*calls))
	copy(result, *calls)
	return result
}

func resetFetchCallsShared(calls *[]FetchCallRecord, mu *sync.Mutex) {
	mu.Lock()
	*calls = nil
	mu.Unlock()
}

// TestController_FetchImmediatelyWithConfig verifies that a successful EventPendingTransactionStatusChanged event
// triggers an immediate balance fetch for the fromAddress/fromChain and toAddress/toChain in SendDetails,
// bypassing the debounce, and only fetches Native + ERC20 for a regular transfer.
func TestController_FetchImmediatelyWithConfig(t *testing.T) {
	const fromChain = uint64(1)
	const toChain = uint64(10)

	walletFeed := new(event.Feed)
	// 0ms debounce lets the initial TriggerFullFetch fire immediately.
	// 1-hour period prevents periodic fetches from interfering.
	cfg := multistandardbalance.ControllerConfig{
		FetchDebounceTime: 0,
		FetchPeriod:       1 * time.Hour,
	}
	controller, fetchCalls, fetchMutex, fromAddr, toAddr := newTestController(t, walletFeed, cfg)

	controller.Start()
	defer controller.Stop()

	// Wait for the initial full fetch (2 active chains) to complete.
	require.Eventually(t, func() bool {
		return len(getFetchCallsCopy(fetchCalls, fetchMutex)) >= 2
	}, 500*time.Millisecond, 10*time.Millisecond, "initial fetch should complete")
	resetFetchCallsShared(fetchCalls, fetchMutex)

	// Send a success event for a regular token transfer.
	payload := pendingtxtracker.StatusChangedPayload{
		Status: ac.Success,
		TxDetails: pendingtxtracker.TxDetails{
			SendDetails: &responses.SendDetails{
				FromAddress: fromAddr,
				FromChain:   fromChain,
				ToAddress:   toAddr,
				ToChain:     toChain,
				SendType:    int(sendtype.Transfer), // not a collectibles transfer
			},
		},
	}
	payloadJSON, err := json.Marshal(payload)
	require.NoError(t, err)
	walletFeed.Send(walletevent.Event{
		Type:    pendingtxtracker.EventPendingTransactionStatusChanged,
		Message: string(payloadJSON),
	})

	// Fetch must happen immediately — no debounce involved.
	require.Eventually(t, func() bool {
		return len(getFetchCallsCopy(fetchCalls, fetchMutex)) >= 2
	}, 500*time.Millisecond, 10*time.Millisecond, "immediate fetch should happen for both chains")

	calls := getFetchCallsCopy(fetchCalls, fetchMutex)
	require.Len(t, calls, 2, "expected exactly one fetch call per chain")

	callsByChain := make(map[uint64]multistandardfetcher.FetchConfig, 2)
	for _, c := range calls {
		callsByChain[c.ChainID] = c.Config
	}

	// fromChain: only fromAddress, Native + ERC20, no collectibles.
	fromCfg, ok := callsByChain[fromChain]
	require.True(t, ok, "expected a fetch call for fromChain %d", fromChain)
	require.Contains(t, fromCfg.Native, common.Address(fromAddr), "fromChain should fetch fromAddress natively")
	require.NotContains(t, fromCfg.Native, common.Address(toAddr), "fromChain should NOT fetch toAddress")
	require.Empty(t, fromCfg.ERC721, "no ERC721 for a regular transfer")
	require.Empty(t, fromCfg.ERC1155, "no ERC1155 for a regular transfer")

	// toChain: only toAddress, Native + ERC20, no collectibles.
	toCfg, ok := callsByChain[toChain]
	require.True(t, ok, "expected a fetch call for toChain %d", toChain)
	require.Contains(t, toCfg.Native, common.Address(toAddr), "toChain should fetch toAddress natively")
	require.NotContains(t, toCfg.Native, common.Address(fromAddr), "toChain should NOT fetch fromAddress")
	require.Empty(t, toCfg.ERC721, "no ERC721 for a regular transfer")
	require.Empty(t, toCfg.ERC1155, "no ERC1155 for a regular transfer")
}

// TestController_FetchImmediatelyWithConfig_IgnoresNonSuccessStatus verifies that
// Pending and Failed events do NOT trigger an immediate balance fetch.
func TestController_FetchImmediatelyWithConfig_IgnoresNonSuccessStatus(t *testing.T) {
	const fromChain = uint64(1)
	const toChain = uint64(10)

	walletFeed := new(event.Feed)
	cfg := multistandardbalance.ControllerConfig{
		FetchDebounceTime: 0,
		FetchPeriod:       1 * time.Hour,
	}
	controller, fetchCalls, fetchMutex, fromAddr, toAddr := newTestController(t, walletFeed, cfg)

	controller.Start()
	defer controller.Stop()

	// Wait for the initial full fetch to complete, then reset.
	require.Eventually(t, func() bool {
		return len(getFetchCallsCopy(fetchCalls, fetchMutex)) >= 2
	}, 500*time.Millisecond, 10*time.Millisecond, "initial fetch should complete")
	resetFetchCallsShared(fetchCalls, fetchMutex)

	for _, status := range []ac.TxStatus{ac.Pending, ac.Failed} {
		resetFetchCallsShared(fetchCalls, fetchMutex)

		payload := pendingtxtracker.StatusChangedPayload{
			Status: status,
			TxDetails: pendingtxtracker.TxDetails{
				SendDetails: &responses.SendDetails{
					FromAddress: fromAddr,
					FromChain:   fromChain,
					ToAddress:   toAddr,
					ToChain:     toChain,
					SendType:    int(sendtype.Transfer),
				},
			},
		}
		payloadJSON, err := json.Marshal(payload)
		require.NoError(t, err)
		walletFeed.Send(walletevent.Event{
			Type:    pendingtxtracker.EventPendingTransactionStatusChanged,
			Message: string(payloadJSON),
		})

		// Give the watcher goroutine time to process the event.
		time.Sleep(100 * time.Millisecond)
		require.Empty(t, getFetchCallsCopy(fetchCalls, fetchMutex),
			"status %q should NOT trigger an immediate fetch", status)
	}
}
