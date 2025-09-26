package blockchainstate

import (
	"context"
	"errors"
	"testing"
	"time"

	mock_blockchainstate "github.com/status-im/status-go/services/wallet/blockchainstate/mock"

	"github.com/stretchr/testify/require"

	"go.uber.org/mock/gomock"
)

var mockupTime = time.Unix(946724400, 0) // 2000-01-01 12:00:00

func mockupSince(t time.Time) time.Duration {
	return mockupTime.Sub(t)
}

var mockupBlockDuration = 10 * time.Second

func mockupBlockDurationFn(chainID uint64) time.Duration {
	return mockupBlockDuration
}

func setupTestState(t *testing.T) (*BlockChainState, *mock_blockchainstate.MockEthClientGetter) {
	ctrl := gomock.NewController(t)
	ethClientGetter := mock_blockchainstate.NewMockEthClientGetter(ctrl)

	state := NewBlockChainState(ethClientGetter)
	state.sinceFn = mockupSince
	state.blockDurationFn = mockupBlockDurationFn
	return state, ethClientGetter
}

func TestGetEstimatedLatestBlockNumber_WithExistingData(t *testing.T) {
	state, _ := setupTestState(t)

	// Manually set block data (simulating what would happen after initialization)
	state.latestBlockData[1] = LatestBlockData{
		blockNumber: 100,
		timestamp:   mockupTime.Add(-31 * time.Second),
	}

	state.latestBlockData[2] = LatestBlockData{
		blockNumber: 200,
		timestamp:   mockupTime.Add(-5 * time.Second),
	}

	// Test chain 1: should estimate 3 blocks ahead (31 seconds / 10 seconds per block)
	mockupBlockDuration = 10 * time.Second
	blockNumber, err := state.GetEstimatedLatestBlockNumber(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, uint64(103), blockNumber)

	// Test chain 2: should not estimate ahead (5 seconds < 12 seconds per block)
	mockupBlockDuration = 12 * time.Second
	blockNumber, err = state.GetEstimatedLatestBlockNumber(context.Background(), 2)
	require.NoError(t, err)
	require.Equal(t, uint64(200), blockNumber)
}

func TestGetEstimatedLatestBlockNumber_WithInitialization(t *testing.T) {
	state, ethClientGetter := setupTestState(t)

	// Mock eth client to return an error (simulating network failure)
	ethClientGetter.EXPECT().EthClient(uint64(1)).Return(nil, errors.New("network error")).AnyTimes()

	// This should trigger initialization but fail
	blockNumber, err := state.GetEstimatedLatestBlockNumber(context.Background(), 1)
	require.Error(t, err)
	require.Equal(t, uint64(0), blockNumber)
	require.Contains(t, err.Error(), "network error")
}

func TestGetEstimatedBlockTime(t *testing.T) {
	state, _ := setupTestState(t)

	// Set up test data
	state.latestBlockData[1] = LatestBlockData{
		blockNumber: 100,
		timestamp:   mockupTime.Add(-10 * time.Second),
	}

	// Test block time estimation
	mockupBlockDuration = 2 * time.Second
	blockTime, err := state.GetEstimatedBlockTime(context.Background(), 1, 105)
	require.NoError(t, err)

	// Block 105 is 5 blocks ahead of 100, so 5 * 2 seconds = 10 seconds ahead
	expectedTime := mockupTime
	require.Equal(t, expectedTime, blockTime)
}

func TestSetLatestBlockNumber(t *testing.T) {
	state, _ := setupTestState(t)

	// Test setting block number
	state.SetLatestBlockNumber(1, 100)

	// Verify it was set
	blockData, exists := state.latestBlockData[1]
	require.True(t, exists)
	require.Equal(t, uint64(100), blockData.blockNumber)
	require.True(t, blockData.timestamp.After(mockupTime.Add(-time.Minute))) // Should be recent

	// Test setting a smaller block number (should not update)
	state.SetLatestBlockNumber(1, 50)
	blockData, exists = state.latestBlockData[1]
	require.True(t, exists)
	require.Equal(t, uint64(100), blockData.blockNumber) // Should still be 100

	// Test setting a larger block number (should update)
	state.SetLatestBlockNumber(1, 150)
	blockData, exists = state.latestBlockData[1]
	require.True(t, exists)
	require.Equal(t, uint64(150), blockData.blockNumber) // Should be updated to 150
}
