package transferdetector_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/go-wallet-sdk/pkg/eventfilter"
	"github.com/status-im/go-wallet-sdk/pkg/eventlog"

	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/pkg/services/wallet/transferdetector"
	mock_transferdetector "github.com/status-im/status-go/pkg/services/wallet/transferdetector/mock"

	"github.com/stretchr/testify/require"

	"go.uber.org/atomic"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestController_DebounceTiming(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fetcher := mock_transferdetector.NewMockFilterProvider(ctrl)
	accountsProvider := mock_transferdetector.NewMockAccountsProvider(ctrl)
	accountsPublisher := pubsub.NewPublisher()
	networksProvider := mock_transferdetector.NewMockNetworksProvider(ctrl)
	lastBlockProvider := mock_transferdetector.NewMockLastBlockProvider(ctrl)
	logger := zap.NewNop()

	// Mock accounts and networks
	address1 := types.Address{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11}
	network1 := &params.Network{ChainID: 1}

	accountsProvider.EXPECT().GetWalletAddresses().Return([]types.Address{address1}, nil).AnyTimes()
	networksProvider.EXPECT().GetActiveNetworks().Return([]*params.Network{network1}, nil).AnyTimes()
	networksProvider.EXPECT().GetPublisher().Return(pubsub.NewPublisher()).AnyTimes()
	lastReturnedBlockNumber := atomic.NewUint64(100)
	lastBlockProvider.EXPECT().GetReadBlocks(uint64(1)).DoAndReturn(func(chainID uint64) (uint64, uint64, bool) {
		lastReturnedBlockNumber.Add(100)
		return 200, lastReturnedBlockNumber.Load(), true
	}).AnyTimes()

	// Track fetch calls
	var fetchCalls int
	var fetchMutex sync.Mutex
	fetcher.EXPECT().FilterTransfers(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, chainID uint64, config eventfilter.TransferQueryConfig) ([]eventlog.Event, error) {
		fetchMutex.Lock()
		fetchCalls++
		fetchMutex.Unlock()
		return []eventlog.Event{}, nil
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
	config := transferdetector.ControllerConfig{
		FetchDebounceTime: 100 * time.Millisecond,
	}

	controller := transferdetector.NewController(
		config,
		fetcher,
		accountsProvider,
		accountsPublisher,
		networksProvider,
		lastBlockProvider,
		logger,
	)

	resetFetchCalls()

	controller.Start()
	defer controller.Stop()

	// After calling Start, the initial block should be set but no actual fetch should occur
	// since lastFetchedBlockNumber starts at 0
	time.Sleep(100 * time.Millisecond)
	require.Equal(t, 0, getFetchCalls(), "Expected no fetch calls after Start (initial block setting only)")
	resetFetchCalls()

	// Test debounce behavior - multiple rapid calls should only result in one fetch
	// The block difference is already set up above (100 -> 150)

	controller.TriggerFetch()
	controller.TriggerFetch()
	controller.TriggerFetch()

	// Wait for debounce time (100ms)
	time.Sleep(200 * time.Millisecond)
	require.Equal(t, 1, getFetchCalls(), "Expected 1 fetch call after multiple consecutive TriggerFetch calls")
	resetFetchCalls()

	// Test that another call after debounce time works
	controller.TriggerFetch()
	// Wait for debounce time
	time.Sleep(200 * time.Millisecond)
	require.Equal(t, 1, getFetchCalls(), "Expected 1 fetch call after TriggerFetch")
	resetFetchCalls()
}

func TestController_TransferDetection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fetcher := mock_transferdetector.NewMockFilterProvider(ctrl)
	accountsProvider := mock_transferdetector.NewMockAccountsProvider(ctrl)
	accountsPublisher := pubsub.NewPublisher()
	networksProvider := mock_transferdetector.NewMockNetworksProvider(ctrl)
	lastBlockProvider := mock_transferdetector.NewMockLastBlockProvider(ctrl)
	logger := zap.NewNop()

	// Mock accounts and networks
	address1 := types.Address{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11}
	address2 := types.Address{0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22}
	network1 := &params.Network{ChainID: 1}
	network2 := &params.Network{ChainID: 2}

	accountsProvider.EXPECT().GetWalletAddresses().Return([]types.Address{address1, address2}, nil).AnyTimes()
	networksProvider.EXPECT().GetActiveNetworks().Return([]*params.Network{network1, network2}, nil).AnyTimes()
	networksProvider.EXPECT().GetPublisher().Return(pubsub.NewPublisher()).AnyTimes()
	lastReturnedBlockNumber1 := atomic.NewUint64(100)
	lastReturnedBlockNumber2 := atomic.NewUint64(200)

	lastBlockProvider.EXPECT().GetReadBlocks(uint64(1)).DoAndReturn(func(chainID uint64) (uint64, uint64, bool) {
		lastReturnedBlockNumber1.Add(100)
		return 200, lastReturnedBlockNumber1.Load(), true
	}).AnyTimes()
	lastBlockProvider.EXPECT().GetReadBlocks(uint64(2)).DoAndReturn(func(chainID uint64) (uint64, uint64, bool) {
		lastReturnedBlockNumber2.Add(100)
		return 300, lastReturnedBlockNumber2.Load(), true
	}).AnyTimes()

	// Mock transfer events
	mockEvents := []eventlog.Event{
		{
			ContractKey: "0x1234567890123456789012345678901234567890",
			EventKey:    "Transfer",
			Unpacked: map[string]interface{}{
				"from":  common.BytesToAddress(address1.Bytes()),
				"to":    common.BytesToAddress(address2.Bytes()),
				"value": "1000000000000000000", // 1 ETH
			},
		},
	}

	// Track fetch calls and verify config
	var fetchCalls []struct {
		ChainID uint64
		Config  eventfilter.TransferQueryConfig
	}
	var fetchMutex sync.Mutex
	fetcher.EXPECT().FilterTransfers(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, chainID uint64, config eventfilter.TransferQueryConfig) ([]eventlog.Event, error) {
		fetchMutex.Lock()
		fetchCalls = append(fetchCalls, struct {
			ChainID uint64
			Config  eventfilter.TransferQueryConfig
		}{ChainID: chainID, Config: config})
		fetchMutex.Unlock()
		return mockEvents, nil
	}).AnyTimes()

	getFetchCalls := func() []struct {
		ChainID uint64
		Config  eventfilter.TransferQueryConfig
	} {
		fetchMutex.Lock()
		defer fetchMutex.Unlock()
		result := make([]struct {
			ChainID uint64
			Config  eventfilter.TransferQueryConfig
		}, len(fetchCalls))
		copy(result, fetchCalls)
		return result
	}

	resetFetchCalls := func() {
		fetchMutex.Lock()
		fetchCalls = nil
		fetchMutex.Unlock()
	}

	// Create controller with short debounce time for testing
	config := transferdetector.ControllerConfig{
		FetchDebounceTime: 100 * time.Millisecond,
	}

	controller := transferdetector.NewController(
		config,
		fetcher,
		accountsProvider,
		accountsPublisher,
		networksProvider,
		lastBlockProvider,
		logger,
	)

	startedCh, startedUnsubFn := pubsub.Subscribe[transferdetector.EventTransferDetectionStarted](controller.GetPublisher(), 10)
	defer startedUnsubFn()

	finishedCh, finishedUnsubFn := pubsub.Subscribe[transferdetector.EventTransferDetectionFinished](controller.GetPublisher(), 10)
	defer finishedUnsubFn()

	errorCh, errorUnsubFn := pubsub.Subscribe[transferdetector.EventTransferDetectionError](controller.GetPublisher(), 10)
	defer errorUnsubFn()

	resetFetchCalls()

	controller.Start()
	defer controller.Stop()

	// After calling Start, the initial blocks should be set but no actual fetch should occur
	// since lastFetchedBlockNumber starts at 0
	time.Sleep(100 * time.Millisecond)
	require.Equal(t, 0, len(getFetchCalls()), "Expected no fetch calls after Start (initial block setting only)")

	// Now trigger a fetch to test the actual functionality
	controller.TriggerFetch()

	// Wait for the fetch to complete
	require.Eventually(t, func() bool {
		calls := getFetchCalls()
		return len(calls) == 2 // Should fetch both chains
	}, 2*time.Second, 50*time.Millisecond, "Expected 2 fetch calls after TriggerFetch")

	// Verify that both chains were fetched
	actualCalls := getFetchCalls()
	require.Len(t, actualCalls, 2, "Expected 2 fetch calls to be made")

	// Verify the configs contain the expected accounts
	chain1Call := actualCalls[0]
	chain2Call := actualCalls[1]

	// One of the calls should be for chain 1, the other for chain 2
	chainIDs := []uint64{chain1Call.ChainID, chain2Call.ChainID}
	require.Contains(t, chainIDs, uint64(1), "Expected chain 1 to be fetched")
	require.Contains(t, chainIDs, uint64(2), "Expected chain 2 to be fetched")

	// Verify that both accounts are included in the config
	for _, call := range actualCalls {
		require.Len(t, call.Config.Accounts, 2, "Expected 2 accounts in config")
		require.Contains(t, call.Config.Accounts, common.BytesToAddress(address1.Bytes()), "Expected address1 in config")
		require.Contains(t, call.Config.Accounts, common.BytesToAddress(address2.Bytes()), "Expected address2 in config")

		// Verify transfer types
		expectedTypes := []eventfilter.TransferType{
			eventfilter.TransferTypeERC20,
			eventfilter.TransferTypeERC721,
			eventfilter.TransferTypeERC1155,
		}
		require.Equal(t, expectedTypes, call.Config.TransferTypes, "Expected correct transfer types")
		require.Equal(t, eventfilter.Both, call.Config.Direction, "Expected Both direction")
	}

	// Verify events are published with correct values
	// Collect events
	var startedEvents []transferdetector.EventTransferDetectionStarted
	var finishedEvents []transferdetector.EventTransferDetectionFinished
	var errorEvents []transferdetector.EventTransferDetectionError
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

	// Collect error events
	go func() {
		for ev := range errorCh {
			eventsMutex.Lock()
			errorEvents = append(errorEvents, ev)
			eventsMutex.Unlock()
		}
	}()

	// Wait for events to be collected
	require.Eventually(t, func() bool {
		eventsMutex.Lock()
		defer eventsMutex.Unlock()
		return len(startedEvents) >= 2 && len(finishedEvents) >= 2
	}, 2*time.Second, 50*time.Millisecond, "Expected started and finished events")

	// Verify started events
	eventsMutex.Lock()
	require.Len(t, startedEvents, 2, "Expected 2 started events (one per chain)")
	require.Len(t, errorEvents, 0, "Expected no error events")

	// Verify started events contain correct values
	startedChainIDs := make(map[uint64]bool)
	for _, ev := range startedEvents {
		startedChainIDs[ev.ChainID] = true
		require.Contains(t, []uint64{1, 2}, ev.ChainID, "Started event should be for chain 1 or 2")
		require.Len(t, ev.Accounts, 2, "Started event should contain 2 accounts")
		require.Contains(t, ev.Accounts, common.BytesToAddress(address1.Bytes()), "Started event should contain address1")
		require.Contains(t, ev.Accounts, common.BytesToAddress(address2.Bytes()), "Started event should contain address2")
		require.True(t, ev.FromBlock > 0, "FromBlock should be positive")
		require.True(t, ev.ToBlock > ev.FromBlock, "ToBlock should be greater than FromBlock")
	}
	require.True(t, startedChainIDs[1], "Expected started event for chain 1")
	require.True(t, startedChainIDs[2], "Expected started event for chain 2")

	// Verify finished events contain correct values
	require.Len(t, finishedEvents, 2, "Expected 2 finished events (one per chain)")
	finishedChainIDs := make(map[uint64]bool)
	for _, ev := range finishedEvents {
		finishedChainIDs[ev.ChainID] = true
		require.Contains(t, []uint64{1, 2}, ev.ChainID, "Finished event should be for chain 1 or 2")
		require.Len(t, ev.Accounts, 2, "Finished event should contain 2 accounts")
		require.Contains(t, ev.Accounts, common.BytesToAddress(address1.Bytes()), "Finished event should contain address1")
		require.Contains(t, ev.Accounts, common.BytesToAddress(address2.Bytes()), "Finished event should contain address2")
		require.True(t, ev.FromBlock > 0, "FromBlock should be positive")
		require.True(t, ev.ToBlock > ev.FromBlock, "ToBlock should be greater than FromBlock")
		require.Len(t, ev.Events, 1, "Finished event should contain 1 mock event")

		// Verify the mock event content
		if len(ev.Events) > 0 {
			event := ev.Events[0]
			require.Equal(t, eventlog.ContractKey("0x1234567890123456789012345678901234567890"), event.ContractKey, "Event should have correct contract key")
			require.Equal(t, eventlog.EventKey("Transfer"), event.EventKey, "Event should have correct event key")
			require.NotNil(t, event.Unpacked, "Event should have unpacked data")

			// Verify unpacked data
			unpacked, ok := event.Unpacked.(map[string]interface{})
			require.True(t, ok, "Unpacked data should be a map")
			require.Equal(t, common.BytesToAddress(address1.Bytes()), unpacked["from"], "Event 'from' should be address1")
			require.Equal(t, common.BytesToAddress(address2.Bytes()), unpacked["to"], "Event 'to' should be address2")
			require.Equal(t, "1000000000000000000", unpacked["value"], "Event 'value' should be 1 ETH")
		}
	}
	require.True(t, finishedChainIDs[1], "Expected finished event for chain 1")
	require.True(t, finishedChainIDs[2], "Expected finished event for chain 2")
	eventsMutex.Unlock()
}
