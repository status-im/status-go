package history

import (
	"context"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/golang/mock/gomock"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/params"
	statusRPC "github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/wallet/market"
	"github.com/status-im/status-go/services/wallet/thirdparty/cryptocompare"
	"github.com/status-im/status-go/transactions/fake"

	"github.com/stretchr/testify/require"
)

func setupDummyServiceNoDependencies(t *testing.T) (service *Service, closeFn func()) {
	db, err := appdatabase.InitializeDB(":memory:", "wallet-history-service-tests", 1)
	require.NoError(t, err)
	cryptoCompare := cryptocompare.NewClient()

	// Creating a dummy status node to simulate what it's done in get_status_node.go
	upstreamConfig := params.UpstreamRPCConfig{
		URL:     "https://mainnet.infura.io/v3/800c641949d64d768a5070a1b0511938",
		Enabled: true,
	}

	txServiceMockCtrl := gomock.NewController(t)
	server, _ := fake.NewTestServer(txServiceMockCtrl)
	client := gethrpc.DialInProc(server)

	rpcClient, err := statusRPC.NewClient(client, 1, upstreamConfig, nil, db)
	require.NoError(t, err)

	return NewService(db, nil, rpcClient, nil, market.NewManager(cryptoCompare, cryptoCompare)), func() {
		require.NoError(t, db.Close())
	}
}

type TestDataPoint struct {
	value       int64
	timestamp   uint64
	blockNumber int64
	chainID     chainIdentity
}

// generateTestDataForElementCount generates dummy consecutive blocks of data for the same chain_id, address and currency
func prepareTestData(data []TestDataPoint) map[chainIdentity][]*DataPoint {
	res := make(map[chainIdentity][]*DataPoint)
	for i := 0; i < len(data); i++ {
		entry := data[i]
		_, found := res[entry.chainID]
		if !found {
			res[entry.chainID] = make([]*DataPoint, 0)
		}
		res[entry.chainID] = append(res[entry.chainID], &DataPoint{
			BlockNumber: (*hexutil.Big)(big.NewInt(data[i].blockNumber)),
			Timestamp:   data[i].timestamp,
			Balance:     (*hexutil.Big)(big.NewInt(data[i].value)),
		})
	}
	return res
}

// getBlockNumbers returns -1 if block number is nil
func getBlockNumbers(data []*DataPoint) []int64 {
	res := make([]int64, 0)
	for _, entry := range data {
		if entry.BlockNumber == nil {
			res = append(res, -1)
		} else {
			res = append(res, entry.BlockNumber.ToInt().Int64())
		}
	}
	return res
}

func getValues(data []*DataPoint) []int64 {
	res := make([]int64, 0)
	for _, entry := range data {
		res = append(res, entry.Balance.ToInt().Int64())
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

func TestServiceMergeDataPoints(t *testing.T) {
	strideDuration := 5 * time.Second
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

	res, err := mergeDataPoints(testData, strideDuration)
	require.NoError(t, err)
	require.Equal(t, 4, len(res))
	require.Equal(t, []int64{105, 115, 125, 130}, getBlockNumbers(res))
	require.Equal(t, []int64{3, 3 * 2, 3 * 3, 3 * 4}, getValues(res))
	require.Equal(t, []int64{110, 120, 130, 135}, getTimestamps(res))
}

func TestServiceMergeDataPointsAllMatch(t *testing.T) {
	strideDuration := 10 * time.Second
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

	res, err := mergeDataPoints(testData, strideDuration)
	require.NoError(t, err)
	require.Equal(t, 4, len(res))
	require.Equal(t, []int64{105, 115, 125, 135}, getBlockNumbers(res))
	require.Equal(t, []int64{3, 3 * 2, 3 * 3, 3 * 4}, getValues(res))
	require.Equal(t, []int64{115, 125, 135, 145}, getTimestamps(res))
}

func TestServiceMergeDataPointsOneChain(t *testing.T) {
	strideDuration := 10 * time.Second
	testData := prepareTestData([]TestDataPoint{
		// Keep 105
		{value: 1, timestamp: 105, blockNumber: 105, chainID: 1},
		// Keep 115
		{value: 2, timestamp: 115, blockNumber: 115, chainID: 1},
		// Keep 125
		{value: 3, timestamp: 125, blockNumber: 125, chainID: 1},
	})

	res, err := mergeDataPoints(testData, strideDuration)
	require.NoError(t, err)
	require.Equal(t, 3, len(res))
	require.Equal(t, []int64{105, 115, 125}, getBlockNumbers(res))
	require.Equal(t, []int64{1, 2, 3}, getValues(res))
	require.Equal(t, []int64{105, 115, 125}, getTimestamps(res), "Expect no merging for one chain")
}

func TestServiceMergeDataPointsDropAll(t *testing.T) {
	strideDuration := 10 * time.Second
	testData := prepareTestData([]TestDataPoint{
		{value: 1, timestamp: 100, blockNumber: 100, chainID: 1},
		{value: 1, timestamp: 110, blockNumber: 110, chainID: 2},
		{value: 1, timestamp: 120, blockNumber: 120, chainID: 3},
		{value: 1, timestamp: 130, blockNumber: 130, chainID: 4},
	})

	res, err := mergeDataPoints(testData, strideDuration)
	require.NoError(t, err)
	require.Equal(t, 0, len(res))
}

func TestServiceMergeDataPointsEmptyDB(t *testing.T) {
	testData := prepareTestData([]TestDataPoint{})

	strideDuration := 10 * time.Second

	res, err := mergeDataPoints(testData, strideDuration)
	require.NoError(t, err)
	require.Equal(t, 0, len(res))
}

func TestServiceFindFirstStrideWindowFirstForAllChainInOneStride(t *testing.T) {
	strideDuration := 10 * time.Second
	testData := prepareTestData([]TestDataPoint{
		{value: 1, timestamp: 103, blockNumber: 101, chainID: 2},
		{value: 1, timestamp: 106, blockNumber: 102, chainID: 3},
		{value: 1, timestamp: 100, blockNumber: 100, chainID: 1},
		{value: 1, timestamp: 110, blockNumber: 103, chainID: 1},
		{value: 1, timestamp: 110, blockNumber: 103, chainID: 2},
	})

	startTimestamp, pos := findFirstStrideWindow(testData, strideDuration)
	require.Equal(t, testData[1][0].Timestamp, uint64(startTimestamp))
	require.Equal(t, map[chainIdentity]int{1: 0, 2: 0, 3: 0}, pos)
}

func TestServiceSortTimeAsc(t *testing.T) {
	testData := prepareTestData([]TestDataPoint{
		{value: 3, timestamp: 103, blockNumber: 103, chainID: 3},
		{value: 4, timestamp: 104, blockNumber: 104, chainID: 4},
		{value: 2, timestamp: 102, blockNumber: 102, chainID: 2},
		{value: 1, timestamp: 101, blockNumber: 101, chainID: 1},
	})

	sorted := sortTimeAsc(testData, map[chainIdentity]int{4: 0, 3: 0, 2: 0, 1: 0})
	require.Equal(t, []timeIdentity{{1, 0}, {2, 0}, {3, 0}, {4, 0}}, sorted)
}

func TestServiceAtEnd(t *testing.T) {
	testData := prepareTestData([]TestDataPoint{
		{value: 1, timestamp: 101, blockNumber: 101, chainID: 1},
		{value: 1, timestamp: 103, blockNumber: 103, chainID: 2},
		{value: 1, timestamp: 105, blockNumber: 105, chainID: 1},
	})

	sorted := sortTimeAsc(testData, map[chainIdentity]int{1: 0, 2: 0})
	require.False(t, sorted[0].atEnd(testData))
	require.True(t, sorted[1].atEnd(testData))
	sorted = sortTimeAsc(testData, map[chainIdentity]int{1: 1, 2: 0})
	require.True(t, sorted[1].atEnd(testData))
}

func TestServiceTokenToValue(t *testing.T) {
	weisInOneMain := big.NewFloat(math.Pow(10, 18.0))
	res := tokenToValue(big.NewInt(12345), 1000, weisInOneMain)
	require.Equal(t, 0.000000000012345, res)

	in, ok := new(big.Int).SetString("1234567890000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000", 10)
	require.True(t, ok)
	res = tokenToValue(in, 10000, weisInOneMain)
	require.Equal(t, 1.23456789e+112, res)

	res = tokenToValue(big.NewInt(1000000000000000000), 1.0, weisInOneMain)
	require.Equal(t, 1.0, res)

	res = tokenToValue(big.NewInt(1), 1.23456789, weisInOneMain)
	require.InEpsilonf(t, 1.23456789e-18, res, 1.0e-8, "Expects error for handling such low values")

	res = tokenToValue(new(big.Int).Exp(big.NewInt(10), big.NewInt(254), nil), 100000, weisInOneMain)
	require.Equal(t, 1e+241, res, "Expect exponent 254-18+5")
}

func TestServiceGetBalanceHistoryNoData(t *testing.T) {
	service, closeFn := setupDummyServiceNoDependencies(t)
	defer closeFn()

	res, err := service.GetBalanceHistory(context.Background(), []uint64{777}, common.HexToAddress(`0x1`), "ETH", "EUR", time.Now().Unix(), BalanceHistory1Year)
	require.NoError(t, err)
	require.Equal(t, 0, len(res))
}
