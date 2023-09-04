package wallet

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var mockupTime = time.Unix(946724400, 0) // 2000-01-01 12:00:00

func mockupSince(t time.Time) time.Duration {
	return mockupTime.Sub(t)
}

func setupTestState(t *testing.T) (s *BlockChainState) {
	state := NewBlockChainState(nil, nil)
	state.sinceFn = mockupSince
	return state
}

func TestEstimateLatestBlockNumber(t *testing.T) {
	state := setupTestState(t)

	state.setLatestBlockDataForChain(1, LatestBlockData{
		blockNumber:   uint64(100),
		timestamp:     mockupTime.Add(-31 * time.Second),
		blockDuration: 10 * time.Second,
	})

	state.setLatestBlockDataForChain(2, LatestBlockData{
		blockNumber:   uint64(200),
		timestamp:     mockupTime.Add(-5 * time.Second),
		blockDuration: 12 * time.Second,
	})

	val, ok := state.estimateLatestBlockNumber(1)
	require.True(t, ok)
	require.Equal(t, uint64(103), val)
	val, ok = state.estimateLatestBlockNumber(2)
	require.True(t, ok)
	require.Equal(t, uint64(200), val)
	val, ok = state.estimateLatestBlockNumber(3)
	require.False(t, ok)
	require.Equal(t, uint64(0), val)
}
