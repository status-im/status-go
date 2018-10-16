package rpcfilters

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAggregateLogs(t *testing.T) {
	logs := []types.Log{}
	for i := 1; i <= 15; i++ {
		logs = append(logs,
			types.Log{BlockNumber: uint64(i), BlockHash: common.Hash{byte(i)}},
			types.Log{BlockNumber: uint64(i), BlockHash: common.Hash{byte(i)}})
	}
	aggregated := aggregateLogs(logs, 10)
	start := 15 - len(aggregated) + 1
	for _, record := range aggregated {
		assert.Equal(t, start, int(record.block)) // numbers are small
		assert.Len(t, record.logs, 2)
		start++
	}
}

func TestAggregateLessThenFull(t *testing.T) {
	logs := []types.Log{}
	for i := 1; i <= 3; i++ {
		logs = append(logs,
			types.Log{BlockNumber: uint64(i), BlockHash: common.Hash{byte(i)}})
	}
	aggregated := aggregateLogs(logs, 10)
	start := 1
	for _, record := range aggregated {
		assert.Equal(t, start, int(record.block)) // numbers are small
		assert.Len(t, record.logs, 1)
		start++
	}
}

func TestMerge(t *testing.T) {
	step1Logs := []types.Log{
		{BlockNumber: 1, BlockHash: common.Hash{1}},
		{BlockNumber: 2, BlockHash: common.Hash{2}},
		{BlockNumber: 3, BlockHash: common.Hash{3}},
	}
	step2Logs := []types.Log{
		{BlockNumber: 2, BlockHash: common.Hash{2}},
		{BlockNumber: 3, BlockHash: common.Hash{3}},
		{BlockNumber: 4, BlockHash: common.Hash{4}},
	}
	reorg := []types.Log{
		{BlockNumber: 2, BlockHash: common.Hash{2, 2}},
		{BlockNumber: 3, BlockHash: common.Hash{3, 3}},
		{BlockNumber: 4, BlockHash: common.Hash{4, 4}},
		{BlockNumber: 5, BlockHash: common.Hash{5, 4}},
	}

	limit := 7
	cache := make([]cacheRecord, 0, limit)
	cache, added, replaced := merge(0, cache, aggregateLogs(step1Logs, limit))
	require.Len(t, added, 3)
	require.Empty(t, replaced)
	require.Equal(t, 3, int(cache[2].block))
	cache, added, replaced = merge(1, cache, aggregateLogs(step2Logs, limit))
	require.Len(t, added, 1)
	require.Empty(t, replaced)
	require.Equal(t, 4, int(cache[3].block))
	_, added, replaced = merge(1, cache, aggregateLogs(reorg, limit))
	require.Len(t, added, 4)
	require.Len(t, replaced, 3)
}

func TestMergeFull(t *testing.T) {
	old := []cacheRecord{
		{block: 1, hash: common.Hash{1}},
		{block: 2, hash: common.Hash{2}},
		{block: 3, hash: common.Hash{3}},
	}
	new := []cacheRecord{
		{block: 4, hash: common.Hash{4}},
		{block: 5, hash: common.Hash{5}},
	}
	old, _, _ = merge(0, old, new)
	require.Len(t, old, 5)
	require.Equal(t, int(old[2].block), 3)
	require.Equal(t, int(old[3].block), 4)
	require.Equal(t, int(old[4].block), 5)
}

func TestAddLogs(t *testing.T) {
	c := newCache(7)
	step1Logs := []types.Log{
		{BlockNumber: 1, BlockHash: common.Hash{1}},
		{BlockNumber: 2, BlockHash: common.Hash{2}},
		{BlockNumber: 3, BlockHash: common.Hash{3}},
	}
	step2Logs := []types.Log{
		{BlockNumber: 2, BlockHash: common.Hash{2}},
		{BlockNumber: 3, BlockHash: common.Hash{3}},
		{BlockNumber: 4, BlockHash: common.Hash{4}},
	}
	added, replaced, err := c.add(step1Logs)
	require.NoError(t, err)
	require.Len(t, added, 3)
	require.Empty(t, replaced)
	added, replaced, err = c.add(step2Logs)
	require.NoError(t, err)
	require.Len(t, added, 1)
	require.Empty(t, replaced)
}

func TestAddLogsNotInOrder(t *testing.T) {
	c := newCache(7)
	logs := []types.Log{{BlockNumber: 1, BlockHash: common.Hash{1}}, {BlockNumber: 3, BlockHash: common.Hash{3}}}
	_, _, err := c.add(logs)
	require.EqualError(t, err, "logs must be delivered straight in order. gaps between blocks '1' and '3'")
}
