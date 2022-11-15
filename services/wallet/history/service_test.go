package history

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/stretchr/testify/require"
)

type TestDataPoint struct {
	value       int64
	timestamp   uint64
	blockNumber int64
	chainID     uint64
}

// generateTestDataForElementCount generates dummy consecutive blocks of data for the same chain_id, address and currency
func prepareTestData(data []TestDataPoint) map[uint64][]*DataPoint {
	res := make(map[uint64][]*DataPoint)
	for i := 0; i < len(data); i++ {
		entry := data[i]
		_, found := res[entry.chainID]
		if !found {
			res[entry.chainID] = make([]*DataPoint, 0)
		}
		res[entry.chainID] = append(res[entry.chainID], &DataPoint{
			BlockNumber: (*hexutil.Big)(big.NewInt(data[i].blockNumber)),
			Timestamp:   data[i].timestamp,
			Value:       (*hexutil.Big)(big.NewInt(data[i].value)),
		})
	}
	return res
}

func getBlockNumbers(data []*DataPoint) []int64 {
	res := make([]int64, 0)
	for _, entry := range data {
		res = append(res, entry.BlockNumber.ToInt().Int64())
	}
	return res
}

func getValues(data []*DataPoint) []int64 {
	res := make([]int64, 0)
	for _, entry := range data {
		res = append(res, entry.Value.ToInt().Int64())
	}
	return res
}

func getTimestamps(data []*DataPoint) []int64 {
	res := make([]int64, 0)
	for _, entry := range data {
		res = append(res, int64(entry.Timestamp))
	}
	return res
}

func TestServiceGetBalanceHistory(t *testing.T) {
	testData := prepareTestData([]TestDataPoint{
		// Drop 100
		{value: 1, timestamp: 100, blockNumber: 100, chainID: 1},
		{value: 1, timestamp: 100, blockNumber: 100, chainID: 2},
		// Keep 105
		{value: 1, timestamp: 105, blockNumber: 105, chainID: 1},
		{value: 1, timestamp: 105, blockNumber: 105, chainID: 2},
		{value: 1, timestamp: 105, blockNumber: 105, chainID: 3},
		// Drop 110
		{value: 1, timestamp: 105, blockNumber: 105, chainID: 2},
		{value: 1, timestamp: 105, blockNumber: 105, chainID: 3},
		// Keep 115
		{value: 2, timestamp: 115, blockNumber: 115, chainID: 1},
		{value: 2, timestamp: 115, blockNumber: 115, chainID: 2},
		{value: 2, timestamp: 115, blockNumber: 115, chainID: 3},
		// Drop 120
		{value: 1, timestamp: 120, blockNumber: 120, chainID: 3},
		// Keep 125
		{value: 3, timestamp: 125, blockNumber: 125, chainID: 1},
		{value: 3, timestamp: 125, blockNumber: 125, chainID: 2},
		{value: 3, timestamp: 125, blockNumber: 125, chainID: 3},
		// Keep 130
		{value: 4, timestamp: 130, blockNumber: 130, chainID: 1},
		{value: 4, timestamp: 130, blockNumber: 130, chainID: 2},
		{value: 4, timestamp: 130, blockNumber: 130, chainID: 3},
		// Drop 135
		{value: 1, timestamp: 135, blockNumber: 135, chainID: 1},
	})

	res, err := mergeDataPoints(testData)
	require.NoError(t, err)
	require.Equal(t, 4, len(res))
	require.Equal(t, []int64{105, 115, 125, 130}, getBlockNumbers(res))
	require.Equal(t, []int64{3, 3 * 2, 3 * 3, 3 * 4}, getValues(res))
	require.Equal(t, []int64{105, 115, 125, 130}, getTimestamps(res))
}

func TestServiceGetBalanceHistoryAllMatch(t *testing.T) {
	testData := prepareTestData([]TestDataPoint{
		// Keep 105
		{value: 1, timestamp: 105, blockNumber: 105, chainID: 1},
		{value: 1, timestamp: 105, blockNumber: 105, chainID: 2},
		{value: 1, timestamp: 105, blockNumber: 105, chainID: 3},
		// Keep 115
		{value: 2, timestamp: 115, blockNumber: 115, chainID: 1},
		{value: 2, timestamp: 115, blockNumber: 115, chainID: 2},
		{value: 2, timestamp: 115, blockNumber: 115, chainID: 3},
		// Keep 125
		{value: 3, timestamp: 125, blockNumber: 125, chainID: 1},
		{value: 3, timestamp: 125, blockNumber: 125, chainID: 2},
		{value: 3, timestamp: 125, blockNumber: 125, chainID: 3},
		// Keep 135
		{value: 4, timestamp: 135, blockNumber: 135, chainID: 1},
		{value: 4, timestamp: 135, blockNumber: 135, chainID: 2},
		{value: 4, timestamp: 135, blockNumber: 135, chainID: 3},
	})

	res, err := mergeDataPoints(testData)
	require.NoError(t, err)
	require.Equal(t, 4, len(res))
	require.Equal(t, []int64{105, 115, 125, 135}, getBlockNumbers(res))
	require.Equal(t, []int64{3, 3 * 2, 3 * 3, 3 * 4}, getValues(res))
	require.Equal(t, []int64{105, 115, 125, 135}, getTimestamps(res))
}

func TestServiceGetBalanceHistoryOneChain(t *testing.T) {
	testData := prepareTestData([]TestDataPoint{
		// Keep 105
		{value: 1, timestamp: 105, blockNumber: 105, chainID: 1},
		// Keep 115
		{value: 2, timestamp: 115, blockNumber: 115, chainID: 1},
		// Keep 125
		{value: 3, timestamp: 125, blockNumber: 125, chainID: 1},
	})

	res, err := mergeDataPoints(testData)
	require.NoError(t, err)
	require.Equal(t, 3, len(res))
	require.Equal(t, []int64{105, 115, 125}, getBlockNumbers(res))
	require.Equal(t, []int64{1, 2, 3}, getValues(res))
	require.Equal(t, []int64{105, 115, 125}, getTimestamps(res))
}

func TestServiceGetBalanceHistoryDropAll(t *testing.T) {
	testData := prepareTestData([]TestDataPoint{
		{value: 1, timestamp: 100, blockNumber: 100, chainID: 1},
		{value: 1, timestamp: 100, blockNumber: 101, chainID: 2},
		{value: 1, timestamp: 100, blockNumber: 102, chainID: 3},
		{value: 1, timestamp: 100, blockNumber: 103, chainID: 4},
	})

	res, err := mergeDataPoints(testData)
	require.NoError(t, err)
	require.Equal(t, 0, len(res))
}

func TestServiceGetBalanceHistoryEmptyDB(t *testing.T) {
	testData := prepareTestData([]TestDataPoint{})

	res, err := mergeDataPoints(testData)
	require.NoError(t, err)
	require.Equal(t, 0, len(res))
}
