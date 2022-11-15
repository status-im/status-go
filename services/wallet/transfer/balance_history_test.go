package transfer

import (
	"context"
	"database/sql"
	"math"
	"math/big"
	"sort"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/sqlite"
)

func setupTestBalanceHistoryDB(t *testing.T) (*BalanceHistory, func()) {
	db, err := appdatabase.InitializeDB(":memory:", "wallet-tests", sqlite.ReducedKDFIterationsNumber)
	require.NoError(t, err)
	walletFeed := &event.Feed{}
	return NewBalanceHistory(db, walletFeed), func() {
		require.NoError(t, db.Close())
	}
}

type requestedBlock struct {
	time              uint64
	blockInfoRequests int
	balanceRequests   int
}

// chainClientTestSource is a test implementation of the BlockInfoSource interface
// It generates dummy consecutive blocks of data and stores them for validation
type chainClientTestSource struct {
	t                   *testing.T
	firstTimeRequest    int64
	requestedBlocks     map[int64]*requestedBlock // map of block number to block data
	lastBlockTimestamp  int64
	firstBlockTimestamp int64
	blockByNumberFn     func(ctx context.Context, number *big.Int) (*types.Block, error)
	balanceAtFn         func(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
	mockTime            int64
	timeAtMock          int64
}

const (
	testTimeLayout = "2006-01-02 15:04:05 Z07:00"
	testTime       = "2022-12-15 12:01:10 +02:00"
)

func getTestTime(t *testing.T) time.Time {
	testTime, err := time.Parse(testTimeLayout, testTime)
	require.NoError(t, err)
	return testTime.UTC()
}

func newTestSource(t *testing.T, availableYears int) *chainClientTestSource {
	return newTestSourceWithCurrentTime(t, availableYears, getTestTime(t).Unix())
}

func newTestSourceWithCurrentTime(t *testing.T, availableYears int, currentTime int64) *chainClientTestSource {
	newInst := &chainClientTestSource{
		t:                   t,
		requestedBlocks:     make(map[int64]*requestedBlock),
		lastBlockTimestamp:  currentTime,
		firstBlockTimestamp: currentTime - int64(float64(availableYears)*float64(secondsInTimeInterval[BalanceHistory1Year])),
		mockTime:            currentTime,
		timeAtMock:          time.Now().UTC().Unix(),
	}
	newInst.blockByNumberFn = newInst.TestBlockByNumber
	newInst.balanceAtFn = newInst.TestBalanceAt
	return newInst
}

const (
	averageBlockTimeSeconds = 12.9
)

func (src *chainClientTestSource) setCurrentTime(newTime int64) {
	src.mockTime = newTime
	src.lastBlockTimestamp = newTime
}

func (src *chainClientTestSource) resetStats() {
	src.requestedBlocks = make(map[int64]*requestedBlock)
}

func (src *chainClientTestSource) availableYears() float64 {
	return float64(src.TimeNow()-src.firstBlockTimestamp) / float64(secondsInTimeInterval[BalanceHistory1Year])
}

func (src *chainClientTestSource) blocksCount() int64 {
	return int64(math.Round(float64(src.TimeNow()-src.firstBlockTimestamp) / averageBlockTimeSeconds))
}

func (src *chainClientTestSource) blockNumberToTimestamp(number int64) int64 {
	return src.firstBlockTimestamp + int64(float64(number)*averageBlockTimeSeconds)
}

func (src *chainClientTestSource) generateBlockInfo(blockNumber int64, time uint64) *types.Block {
	return types.NewBlockWithHeader(&types.Header{
		Number: big.NewInt(blockNumber),
		Time:   time,
	})
}

func (src *chainClientTestSource) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return src.blockByNumberFn(ctx, number)
}

func (src *chainClientTestSource) TestBlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	require.LessOrEqual(src.t, number.Int64(), src.blocksCount())
	var blockNo int64
	if number == nil {
		// Last block was requested
		blockNo = src.blocksCount()
	} else {
		blockNo = number.Int64()
	}
	timestamp := src.blockNumberToTimestamp(blockNo)

	if _, contains := src.requestedBlocks[blockNo]; contains {
		src.requestedBlocks[blockNo].blockInfoRequests++
	} else {
		src.requestedBlocks[blockNo] = &requestedBlock{
			time:              uint64(timestamp),
			blockInfoRequests: 1,
		}
	}

	return src.generateBlockInfo(blockNo, uint64(timestamp)), nil
}

func (src *chainClientTestSource) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	return src.balanceAtFn(ctx, account, blockNumber)
}

func weiInEth() *big.Int {
	res, _ := new(big.Int).SetString("1000000000000000000", 0)
	return res
}

func (src *chainClientTestSource) TestBalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	require.Greater(src.t, blockNumber.Int64(), int64(0))
	blockNo := blockNumber.Int64()
	if _, contains := src.requestedBlocks[blockNo]; contains {
		src.requestedBlocks[blockNo].balanceRequests++
	} else {
		src.requestedBlocks[blockNo] = &requestedBlock{
			time:            uint64(src.blockNumberToTimestamp(blockNo)),
			balanceRequests: 1,
		}
	}

	return new(big.Int).Mul(big.NewInt(blockNo), weiInEth()), nil
}

func (src *chainClientTestSource) ChainID() uint64 {
	return 777
}

func (src *chainClientTestSource) Currency() string {
	return "eth"
}

func (src *chainClientTestSource) TimeNow() int64 {
	if src.firstTimeRequest == 0 {
		src.firstTimeRequest = time.Now().UTC().Unix()
	}
	return src.mockTime + (time.Now().UTC().Unix() - src.firstTimeRequest)
}

func TestGetOrFetchCalibrationData(t *testing.T) {
	bh, cleanDB := setupTestBalanceHistoryDB(t)
	defer cleanDB()

	dataSource := newTestSource(t, 3 /*years*/)
	balanceData, err := bh.getOrFetchCalibrationData(context.Background(), dataSource)
	require.NoError(t, err)
	require.Greater(t, len(balanceData), 0)
	require.Equal(t, int(dataSource.availableYears()), len(dataSource.requestedBlocks))
	require.Equal(t, len(balanceData), len(dataSource.requestedBlocks))
	for i := 1; i < len(balanceData); i++ {
		require.Greater(t, balanceData[i].block.Cmp(balanceData[i-1].block), 0)
	}
}

func TestGetOrFetchCalibrationDataErrorFetching(t *testing.T) {
	bh, cleanDB := setupTestBalanceHistoryDB(t)
	defer cleanDB()

	dataSource := newTestSource(t, 6 /*years*/)
	bkFn := dataSource.blockByNumberFn
	// Fail first request
	dataSource.blockByNumberFn = func(ctx context.Context, number *big.Int) (*types.Block, error) {
		return nil, errors.New("test error")
	}
	calibrationData, err := bh.getOrFetchCalibrationData(context.Background(), dataSource)
	require.Error(t, err, "test error")
	require.Nil(t, calibrationData)

	// Fail at second request
	dataSource.blockByNumberFn = func(ctx context.Context, number *big.Int) (*types.Block, error) {
		if len(dataSource.requestedBlocks) == 1 {
			return nil, errors.New("test error")
		}
		return dataSource.TestBlockByNumber(ctx, number)
	}
	calibrationData, err = bh.getOrFetchCalibrationData(context.Background(), dataSource)
	require.Error(t, err, "test error")
	require.Nil(t, calibrationData)
	require.Equal(t, 1, len(dataSource.requestedBlocks))

	// Fail at half the blocks
	dataSource.blockByNumberFn = func(ctx context.Context, number *big.Int) (*types.Block, error) {
		if number.Cmp(big.NewInt(dataSource.blocksCount()/2)) > 0 {
			return nil, errors.New("test error")
		}
		return dataSource.TestBlockByNumber(ctx, number)
	}
	calibrationData, err = bh.getOrFetchCalibrationData(context.Background(), dataSource)
	require.Error(t, err, "test error")
	require.Nil(t, calibrationData)
	require.Equal(t, 3, len(dataSource.requestedBlocks))
	require.Nil(t, calibrationData)

	dataSource.requestedBlocks = make(map[int64]*requestedBlock)
	dataSource.blockByNumberFn = bkFn
	calibrationData, err = bh.getOrFetchCalibrationData(context.Background(), dataSource)
	require.NoError(t, err)
	require.Greater(t, len(calibrationData), 0)
	require.Equal(t, 3, len(dataSource.requestedBlocks))
	require.Equal(t, 6, len(calibrationData))
	require.Equal(t, int64(1), calibrationData[0].block.Int64())
}

// extractTestData returns reqBlkNos sorted in ascending order
func extractTestData(dataSource *chainClientTestSource) (reqBlkNos []int64, infoRequests map[int64]int, balanceRequests map[int64]int) {
	reqBlkNos = make([]int64, 0, len(dataSource.requestedBlocks))
	for blockNo := range dataSource.requestedBlocks {
		reqBlkNos = append(reqBlkNos, blockNo)
	}
	sort.Slice(reqBlkNos, func(i, j int) bool {
		return reqBlkNos[i] < reqBlkNos[j]
	})

	infoRequests = make(map[int64]int)
	balanceRequests = make(map[int64]int, len(reqBlkNos))
	for i := 0; i < len(reqBlkNos); i++ {
		n := reqBlkNos[i]
		rB := dataSource.requestedBlocks[n]

		if rB.blockInfoRequests > 0 {
			infoRequests[n] = rB.blockInfoRequests
		}
		if rB.balanceRequests > 0 {
			balanceRequests[n] = rB.balanceRequests
		}
	}
	return
}

func TestGetBalanceHistoryForBlocksSource(t *testing.T) {
	bh, cleanDB := setupTestBalanceHistoryDB(t)
	defer cleanDB()

	dataSource := newTestSource(t, 20 /*years*/)

	balanceData, err := bh.getBalanceHistoryFromBlocksSource(context.Background(), dataSource, common.Address{7}, BalanceHistory1Year)
	require.NoError(t, err)
	require.Greater(t, len(balanceData), 0)

	reqBlkNos, calibrations, balances := extractTestData(dataSource)
	require.Equal(t, len(balances), len(balanceData))

	// Ensure we don't request the same info twice
	for block, count := range calibrations {
		require.Equal(t, 1, count, "block %d has one info request", block)
		if balanceCount, contains := balances[block]; contains {
			require.Equal(t, 1, balanceCount, "block %d has one balance request", block)
		}
	}
	for block, count := range balances {
		require.Equal(t, 1, count, "block %d has one request", block)
	}

	resIdx := 0
	for i := 0; i < len(reqBlkNos); i++ {
		n := reqBlkNos[i]
		rB := dataSource.requestedBlocks[n]

		if _, contains := balances[n]; contains {
			// Ensure block approximation error doesn't exceed 10 blocks
			require.Less(t, math.Abs(float64(int64(rB.time)-int64(balanceData[resIdx].Timestamp))), float64(10 /*blocks*/ *averageBlockTimeSeconds))
			if resIdx > 0 {
				// Ensure result timestamps are in order
				require.Greater(t, balanceData[resIdx].Timestamp, balanceData[resIdx-1].Timestamp)
			}
			resIdx++
		}
	}
	require.Equal(t, int(dataSource.availableYears()), len(calibrations))

	timeRange := balanceData[len(balanceData)-1].Timestamp - balanceData[0].Timestamp
	secondsInYear := secondsInTimeInterval[BalanceHistory1Year]
	// We allow  +2% error due to block approximation and time snapping
	timeErrorAllowed := int64(0.02 * float64(secondsInYear))
	require.Greater(t, timeRange, uint64((1 /*year*/ *secondsInYear)-timeErrorAllowed))
	require.Less(t, timeRange, uint64((1 /*year*/ *secondsInYear)+timeErrorAllowed))
}

func TestGetBalanceHistoryForBlocksSourceBlockFetchingFailure(t *testing.T) {
	bh, cleanDB := setupTestBalanceHistoryDB(t)
	defer cleanDB()

	dataSource := newTestSource(t, 20 /*years*/)
	bkFn := dataSource.balanceAtFn
	// Fail first request
	dataSource.balanceAtFn = func(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
		return nil, errors.New("test error")
	}
	balanceData, err := bh.getBalanceHistoryFromBlocksSource(context.Background(), dataSource, common.Address{7}, BalanceHistory1Year)
	require.Error(t, err, "test error")
	require.Equal(t, len(balanceData), 0)
	_, calibrations, balances := extractTestData(dataSource)
	require.Equal(t, len(balances), 0)
	require.Greater(t, len(calibrations), 0)

	dataSource.resetStats()
	// Fail in the middle
	dataSource.balanceAtFn = func(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
		if len(dataSource.requestedBlocks) == 15 {
			return nil, errors.New("test error")
		}
		return dataSource.TestBalanceAt(ctx, account, blockNumber)
	}
	balanceData, err = bh.getBalanceHistoryFromBlocksSource(context.Background(), dataSource, common.Address{7}, BalanceHistory1Year)
	require.Error(t, err, "test error")
	require.Equal(t, len(balanceData), 0)
	_, calibrations, balances = extractTestData(dataSource)
	require.Equal(t, len(balances), 15)
	require.Equal(t, len(calibrations), 0)

	dataSource.resetStats()
	dataSource.balanceAtFn = bkFn
	balanceData, err = bh.getBalanceHistoryFromBlocksSource(context.Background(), dataSource, common.Address{7}, BalanceHistory1Year)
	require.NoError(t, err)
	require.Greater(t, len(balanceData), 0)

	_, _, balances = extractTestData(dataSource)
	require.Equal(t, len(balanceData)-15, len(balances))

	for i := 1; i < len(balanceData); i++ {
		require.Greater(t, balanceData[i].Timestamp, balanceData[i-1].Timestamp)
	}

	timeRange := balanceData[len(balanceData)-1].Timestamp - balanceData[0].Timestamp
	secondsInYear := secondsInTimeInterval[BalanceHistory1Year]
	// We allow  +/-2% error due to block approximation and time snapping
	timeErrorAllowed := int64(0.02 * float64(secondsInYear))
	require.Greater(t, timeRange, uint64((1 /*year*/ *secondsInYear)-timeErrorAllowed))
	require.Less(t, timeRange, uint64((1 /*year*/ *secondsInYear)+timeErrorAllowed))
}

func TestGetBalanceHistoryForBlocksSourceValidateBalanceValues(t *testing.T) {
	bh, cleanDB := setupTestBalanceHistoryDB(t)
	defer cleanDB()

	dataSource := newTestSource(t, 20 /*years*/)

	for currentInterval := int(BalanceHistory7Hours); currentInterval <= int(BalanceHistoryAllTime); currentInterval++ {
		dataSource.resetStats()

		requestedBalance := make(map[int64]*big.Int)
		dataSource.balanceAtFn = func(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
			balance, err := dataSource.TestBalanceAt(ctx, account, blockNumber)
			requestedBalance[blockNumber.Int64()] = new(big.Int).Set(balance)
			return balance, err
		}
		balanceData, err := bh.getBalanceHistoryFromBlocksSource(context.Background(), dataSource, common.Address{7}, BalanceHistoryTimeInterval(currentInterval))
		require.NoError(t, err)
		require.Greater(t, len(balanceData), 0)

		// Only first run is not affected by cache
		if currentInterval == int(BalanceHistory7Hours) {
			require.Equal(t, len(balanceData), len(requestedBalance))

			reqBlkNos, _, _ := extractTestData(dataSource)

			resIdx := 0
			// Check that balance values are the one requested
			for i := 0; i < len(reqBlkNos); i++ {
				n := reqBlkNos[i]

				if value, contains := requestedBalance[n]; contains {
					require.Equal(t, value.Cmp(balanceData[resIdx].Value.ToInt()), 0)
					resIdx++
				}
			}
		} else {
			require.Greater(t, len(balanceData), len(requestedBalance))
		}

		// Check that balance values are in order
		for i := 1; i < len(balanceData); i++ {
			require.Greater(t, balanceData[i].Value.ToInt().Cmp(balanceData[i-1].Value.ToInt()), 0, "expected balanceData[%d] > balanceData[%d] for interval %d", i, i-1, currentInterval)
		}
	}
}

func TestGetBalanceHistoryForBlocksSourceVerifyCache(t *testing.T) {
	bh, cleanDB := setupTestBalanceHistoryDB(t)
	defer cleanDB()

	currentTime := getTestTime(t)
	oneDayAgoTime := currentTime
	oneDayAgoTime = oneDayAgoTime.Add(-time.Hour * 24)
	dataSource := newTestSourceWithCurrentTime(t, 20 /*years*/, oneDayAgoTime.Unix())

	prevBalanceData, err := bh.getBalanceHistoryFromBlocksSource(context.Background(), dataSource, common.Address{7}, BalanceHistory1Month)
	require.NoError(t, err)
	require.Greater(t, len(prevBalanceData), 0)

	timeRange := prevBalanceData[len(prevBalanceData)-1].Timestamp - prevBalanceData[0].Timestamp
	secondsInMonth := secondsInTimeInterval[BalanceHistory1Month]
	// We allow  +/-1% error due to block approximation and time snapping
	timeErrorAllowed := int64(0.01 * float64(secondsInMonth))
	require.Greater(t, timeRange, uint64((1 /*month*/ *secondsInMonth)-timeErrorAllowed))
	require.Less(t, timeRange, uint64((1 /*month*/ *secondsInMonth)+timeErrorAllowed))

	// Advance to now
	dataSource.setCurrentTime(currentTime.Unix())
	dataSource.resetStats()
	updatedBalanceData, err := bh.getBalanceHistoryFromBlocksSource(context.Background(), dataSource, common.Address{7}, BalanceHistory1Month)
	require.NoError(t, err)
	require.Greater(t, len(updatedBalanceData), 0)

	reqBlkNos, calibrations, balances := extractTestData(dataSource)
	require.Equal(t, 2, len(reqBlkNos))
	require.Equal(t, 0, len(calibrations))

	for block, count := range balances {
		require.Equal(t, 1, count, "block %d has one request", block)
	}

	resIdx := len(updatedBalanceData) - 2
	for i := 0; i < len(reqBlkNos); i++ {
		rB := dataSource.requestedBlocks[reqBlkNos[i]]

		// Ensure block approximation error doesn't exceed 10 blocks
		require.Less(t, math.Abs(float64(int64(rB.time)-int64(updatedBalanceData[resIdx].Timestamp))), float64(10 /*blocks*/ *averageBlockTimeSeconds))
		if resIdx > 0 {
			// Ensure result timestamps are in order
			require.Greater(t, updatedBalanceData[resIdx].Timestamp, updatedBalanceData[resIdx-1].Timestamp)
		}
		resIdx++
	}

	timeRange = updatedBalanceData[len(updatedBalanceData)-1].Timestamp - updatedBalanceData[0].Timestamp
	require.Greater(t, timeRange, uint64((1 /*month*/ *secondsInMonth)-timeErrorAllowed))
	require.Less(t, timeRange, uint64((1 /*month*/ *secondsInMonth)+timeErrorAllowed))
}

func TestGetBalanceHistoryForBlocksSourceFetchMultipleAccounts(t *testing.T) {
	bh, cleanDB := setupTestBalanceHistoryDB(t)
	defer cleanDB()

	sevenDataSource := newTestSource(t, 5 /*years*/)
	sevenBalanceData, err := bh.getBalanceHistoryFromBlocksSource(context.Background(), sevenDataSource, common.Address{7}, BalanceHistory1Month)
	require.NoError(t, err)
	require.Greater(t, len(sevenBalanceData), 0)

	_, sevenCalibrations, _ := extractTestData(sevenDataSource)
	require.Greater(t, len(sevenCalibrations), 0)

	nineDataSource := newTestSource(t, 5 /*years*/)
	nineBalanceData, err := bh.getBalanceHistoryFromBlocksSource(context.Background(), nineDataSource, common.Address{9}, BalanceHistory1Month)
	require.NoError(t, err)
	require.Greater(t, len(nineBalanceData), 0)

	_, nineCalibrations, _ := extractTestData(nineDataSource)
	require.Equal(t, 0, len(nineCalibrations))
}

func TestGetBalanceHistoryForBlocksSourceTaskCancellation(t *testing.T) {
	bh, cleanDB := setupTestBalanceHistoryDB(t)
	defer cleanDB()

	dataSource := newTestSource(t, 5 /*years*/)
	ctx, cancelFn := context.WithCancel(context.Background())
	bkFn := dataSource.blockByNumberFn
	// Fail after 15 requests
	dataSource.balanceAtFn = func(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
		if len(dataSource.requestedBlocks) == 15 {
			cancelFn()
		}
		return dataSource.TestBalanceAt(ctx, account, blockNumber)
	}
	balanceData, err := bh.getBalanceHistoryFromBlocksSource(ctx, dataSource, common.Address{7}, BalanceHistory1Year)
	require.Error(t, err, "context cancelled")
	require.Equal(t, len(balanceData), 0)

	_, calibrations, balances := extractTestData(dataSource)
	require.Equal(t, 16, len(balances)+len(calibrations))

	dataSource.blockByNumberFn = bkFn
	ctx, cancelFn = context.WithCancel(context.Background())
	balanceData, err = bh.getBalanceHistoryFromBlocksSource(ctx, dataSource, common.Address{7}, BalanceHistory1Year)
	require.NoError(t, err)
	require.Greater(t, len(balanceData), 0)
	cancelFn()
}

func TestHoursPerStepHaveCommonDivisor(t *testing.T) {
	values := make([]int64, 0, len(timeIntervalToHoursPerStep))
	for _, hours := range timeIntervalToHoursPerStep {
		values = append(values, hours)
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i] < values[j]
	})
	for i := 1; i < len(values); i++ {
		require.Equal(t, int64(0), values[i]%values[i-1], " %d value from index %d is divisible with previous %d", values[i], i, values[i-1])
	}
}

func TestTimeIntervalToBitsetFilterAreConsecutiveFlags(t *testing.T) {
	values := make([]int, 0, len(timeIntervalToBitsetFilter))
	for i := BalanceHistoryAllTime; i >= BalanceHistory7Hours; i-- {
		values = append(values, int(timeIntervalToBitsetFilter[i]))
	}
	values = append(values, int(FilterIncludeAll))

	for i := 0; i < len(values); i++ {
		// count number of bits set
		count := 0
		for j := 0; j <= 30; j++ {
			if values[i]&(1<<j) != 0 {
				count++
			}
		}
		require.Equal(t, 1, count, "%b value from index %d has only one bit set", values[i], i)

		if i > 0 {
			require.Greater(t, values[i], values[i-1], "%b value from index %d is higher then previous %d", values[i], i, values[i-1])
		}
	}
}

func TestSnapTimestamp(t *testing.T) {
	testDate := time.Date(2020, 3 /*M*/, 12 /*d*/, 3 /*H*/, 34 /*m*/, 56 /*s*/, 567 /*ms*/, time.UTC)
	snappedTimestamp := snapTimestamp(testDate.Unix(), BalanceHistory7Hours)
	expectedTimestamp := time.Date(2020, 3 /*M*/, 12 /*d*/, 2 /*H*/, 0 /*m*/, 0 /*s*/, 0 /*ms*/, time.UTC).Unix()
	require.Equal(t, expectedTimestamp, snappedTimestamp)

	testDate = testDate.Add(4 * time.Hour)
	snappedTimestamp = snapTimestamp(testDate.Unix(), BalanceHistory7Hours)
	expectedTimestamp = time.Date(2020, 3 /*M*/, 12 /*d*/, 6 /*H*/, 0 /*m*/, 0 /*s*/, 0 /*ms*/, time.UTC).Unix()
	require.Equal(t, expectedTimestamp, snappedTimestamp)

	snappedTimestamp = snapTimestamp(testDate.Unix(), BalanceHistory1Month)
	expectedTimestamp = time.Date(2020, 3 /*M*/, 12 /*d*/, 0 /*H*/, 0 /*m*/, 0 /*s*/, 0 /*ms*/, time.UTC).Unix()
	require.Equal(t, expectedTimestamp, snappedTimestamp)

	testDate = time.Date(2015, 12 /*M*/, 30 /*d*/, 23 /*H*/, 34 /*m*/, 56 /*s*/, 567 /*ms*/, time.UTC)
	snappedTimestamp = snapTimestamp(testDate.Unix(), BalanceHistory6Months)
	expectedTimestamp = time.Date(2015, 12 /*M*/, 29 /*d*/, 0 /*H*/, 0 /*m*/, 0 /*s*/, 0 /*ms*/, time.UTC).Unix()
	require.Equal(t, expectedTimestamp, snappedTimestamp)

	testDate = testDate.Add(time.Duration(-timeIntervalToHoursPerStep[BalanceHistory6Months]) * time.Hour)
	snappedTimestamp = snapTimestamp(testDate.Unix(), BalanceHistory6Months)
	expectedTimestamp = time.Date(2015, 12 /*M*/, 25 /*d*/, 12 /*H*/, 0 /*m*/, 0 /*s*/, 0 /*ms*/, time.UTC).Unix()
	require.Equal(t, expectedTimestamp, snappedTimestamp)

	testDate = time.Date(2007, 1 /*M*/, 1 /*d*/, 23 /*H*/, 34 /*m*/, 56 /*s*/, 567 /*ms*/, time.UTC)
	snappedTimestamp = snapTimestamp(testDate.Unix(), BalanceHistory1Year)
	expectedTimestamp = time.Date(2007, 1 /*M*/, 1 /*d*/, 0 /*H*/, 0 /*m*/, 0 /*s*/, 0 /*ms*/, time.UTC).Unix()
	require.Equal(t, expectedTimestamp, snappedTimestamp)

	testDate = testDate.Add(time.Duration(3*timeIntervalToHoursPerStep[BalanceHistory1Year]) * time.Hour)
	snappedTimestamp = snapTimestamp(testDate.Unix(), BalanceHistory1Year)
	expectedTimestamp = time.Date(2007, 1 /*M*/, 22 /*d*/, 0 /*H*/, 0 /*m*/, 0 /*s*/, 0 /*ms*/, time.UTC).Unix()
	require.Equal(t, expectedTimestamp, snappedTimestamp)

	testDate = time.Date(2011, 11 /*M*/, 13 /*d*/, 23 /*H*/, 34 /*m*/, 56 /*s*/, 567 /*ms*/, time.UTC)
	snappedTimestamp = snapTimestamp(testDate.Unix(), BalanceHistoryAllTime)
	expectedTimestamp = time.Date(2011, 8 /*M*/, 1 /*d*/, 0 /*H*/, 0 /*m*/, 0 /*s*/, 0 /*ms*/, time.UTC).Unix()
	require.Equal(t, expectedTimestamp, snappedTimestamp)

	testDate = time.Date(2003, 3 /*M*/, 2 /*d*/, 23 /*H*/, 34 /*m*/, 56 /*s*/, 567 /*ms*/, time.UTC)
	snappedTimestamp = snapTimestamp(testDate.Unix(), BalanceHistoryAllTime)
	expectedTimestamp = time.Date(2003, 1 /*M*/, 1 /*d*/, 0 /*H*/, 0 /*m*/, 0 /*s*/, 0 /*ms*/, time.UTC).Unix()
	require.Equal(t, expectedTimestamp, snappedTimestamp)
}

func TestGetBalanceHistoryForBlocksSourceTestCacheHit(t *testing.T) {
	bh, cleanDB := setupTestBalanceHistoryDB(t)
	defer cleanDB()

	monthDataSource := newTestSource(t, 5 /*years*/)
	monthBalanceData, err := bh.getBalanceHistoryFromBlocksSource(context.Background(), monthDataSource, common.Address{7}, BalanceHistory1Month)
	require.NoError(t, err)
	require.Greater(t, len(monthBalanceData), 0)

	halfYearDataSource := newTestSource(t, 5 /*years*/)
	halfYearBalanceData, err := bh.getBalanceHistoryFromBlocksSource(context.Background(), halfYearDataSource, common.Address{7}, BalanceHistory1Year)
	require.NoError(t, err)
	require.Greater(t, len(halfYearBalanceData), 0)

	_, _, halfYearBalances := extractTestData(halfYearDataSource)
	// Minimal cache hit is expected for higher time interval
	require.Greater(t, len(halfYearBalanceData), len(halfYearBalances))

	yearDataSource := newTestSource(t, 5 /*years*/)
	yearBalanceData, err := bh.getBalanceHistoryFromBlocksSource(context.Background(), yearDataSource, common.Address{7}, BalanceHistory6Months)
	require.NoError(t, err)
	require.Greater(t, len(yearBalanceData), 0)

	_, _, yearBalances := extractTestData(yearDataSource)
	// More than half requests are expected to be cache hit
	require.Greater(t, int(math.Abs(float64(len(yearBalanceData))/2.0)), len(yearBalances))

	// Execute the same fetch again
	yearDataSource.resetStats()
	againYearBalanceData, err := bh.getBalanceHistoryFromBlocksSource(context.Background(), yearDataSource, common.Address{7}, BalanceHistory6Months)
	require.NoError(t, err)
	require.Greater(t, len(againYearBalanceData), 0)
	require.Equal(t, len(yearBalanceData), len(againYearBalanceData))

	_, _, againYearBalances := extractTestData(yearDataSource)
	// More than half requests are expected to be cache hit
	require.Equal(t, len(againYearBalances), 0)
}

func TestGetBalanceHistoryForBlocksSourceFetchAllTime(t *testing.T) {
	bh, cleanDB := setupTestBalanceHistoryDB(t)
	defer cleanDB()

	dataSource := newTestSource(t, 5 /*years*/)

	balanceData, err := bh.getBalanceHistoryFromBlocksSource(context.Background(), dataSource, common.Address{7}, BalanceHistoryAllTime)
	require.NoError(t, err)
	require.Greater(t, len(balanceData), 0)
	// Balance values are ETH as block numbers, so the first value should be 1
	require.Equal(t, new(big.Int).Mul(big.NewInt(1), weiInEth()), balanceData[0].Value.ToInt())

	for i := 1; i < len(balanceData); i++ {
		require.Greater(t, balanceData[i].Timestamp, balanceData[i-1].Timestamp)
		require.Greater(t, balanceData[i].Value.ToInt().Cmp(balanceData[i-1].Value.ToInt()), 0)
	}

	timeRange := balanceData[len(balanceData)-1].Timestamp - balanceData[0].Timestamp
	expectedDuration := dataSource.availableYears() * float64(secondsInTimeInterval[BalanceHistory1Year])
	// We allow error less than a step due to block approximation and time snapping
	timeErrorAllowed := time.Duration(timeIntervalToHoursPerStep[BalanceHistoryAllTime]) * time.Hour
	require.Greater(t, timeRange, uint64(expectedDuration-float64(timeErrorAllowed)))
	require.Less(t, timeRange, uint64(expectedDuration+float64(timeErrorAllowed)))

	blockInfoRequestCount := 0
	for blockNo := range dataSource.requestedBlocks {
		rB := dataSource.requestedBlocks[blockNo]
		if rB.blockInfoRequests > 0 {
			blockInfoRequestCount++
		}
	}
	require.Equal(t, int64(dataSource.availableYears()), int64(blockInfoRequestCount))
}

// generateTestDataForElementCount generates dummy consecutive blocks of data for the same chain_id, address and currency
func generateTestDataForElementCount(count int) (result []*balanceHistoryDBEntry) {
	baseDataPoint := balanceHistoryDBEntry{
		chainID:   777,
		address:   common.Address{7},
		currency:  "ETH",
		block:     big.NewInt(11),
		balance:   big.NewInt(101),
		timestamp: 11,
	}

	result = make([]*balanceHistoryDBEntry, 0, count)
	for i := 0; i < count; i++ {
		newDataPoint := baseDataPoint
		newDataPoint.block = big.NewInt(0).Add(baseDataPoint.block, big.NewInt(int64(i)))
		newDataPoint.timestamp += int64(i)
		result = append(result, &newDataPoint)
	}
	return result
}

func TestBalanceHistoryAddDataPoint(t *testing.T) {
	bh, cleanDB := setupTestBalanceHistoryDB(t)
	defer cleanDB()

	testDataPoint := generateTestDataForElementCount(1)[0]

	err := bh.addBalanceEntryToDB(testDataPoint, 1)
	require.NoError(t, err)

	outDataPoint := balanceHistoryDBEntry{
		chainID: 0,
		block:   big.NewInt(0),
		balance: big.NewInt(0),
	}
	rows, err := bh.db.Query("SELECT * FROM balance_history")
	require.NoError(t, err)

	ok := rows.Next()
	require.True(t, ok)

	bitset := 0
	err = rows.Scan(&outDataPoint.chainID, &outDataPoint.address, &outDataPoint.currency, (*bigint.SQLBigInt)(outDataPoint.block), &outDataPoint.timestamp, &bitset, (*bigint.SQLBigIntBytes)(outDataPoint.balance))
	require.NoError(t, err)
	require.NotEqual(t, err, sql.ErrNoRows)
	require.Equal(t, testDataPoint, &outDataPoint)

	ok = rows.Next()
	require.False(t, ok)
}

func TestBalanceHistoryUpdateDataPoint(t *testing.T) {
	bh, cleanDB := setupTestBalanceHistoryDB(t)
	defer cleanDB()

	testDataPoints := generateTestDataForElementCount(2)

	err := bh.addBalanceEntryToDB(testDataPoints[0], 1)
	require.NoError(t, err)
	testDataPoints[1].block.Set(testDataPoints[0].block)
	err = bh.addBalanceEntryToDB(testDataPoints[1], 1)
	require.Error(t, err)
	err = bh.upsertBalanceEntryToDB(testDataPoints[1], 1)
	require.NoError(t, err)

	outDataPoint := balanceHistoryDBEntry{
		chainID: 0,
		block:   big.NewInt(0),
		balance: big.NewInt(0),
	}
	rows, err := bh.db.Query("SELECT * FROM balance_history")
	require.NoError(t, err)

	ok := rows.Next()
	require.True(t, ok)

	bitset := 0
	err = rows.Scan(&outDataPoint.chainID, &outDataPoint.address, &outDataPoint.currency, (*bigint.SQLBigInt)(outDataPoint.block), &outDataPoint.timestamp, &bitset, (*bigint.SQLBigIntBytes)(outDataPoint.balance))
	require.NoError(t, err)
	require.NotEqual(t, err, sql.ErrNoRows)
	require.Equal(t, testDataPoints[1], &outDataPoint)
	require.Equal(t, 1, bitset)

	ok = rows.Next()
	require.False(t, ok)
}

func TestBalanceHistoryGetDataPoint(t *testing.T) {
	bh, cleanDB := setupTestBalanceHistoryDB(t)
	defer cleanDB()

	testDataPoints := generateTestDataForElementCount(2)

	err := bh.addBalanceEntryToDB(testDataPoints[0], 2)
	require.NoError(t, err)

	outDataPoint, bitset, err := bh.getBalanceEntryFromDB(testDataPoints[0].chainID, testDataPoints[0].address, testDataPoints[0].currency, testDataPoints[0].block)
	require.NoError(t, err)
	require.NotEqual(t, outDataPoint, nil)
	require.Equal(t, testDataPoints[0], outDataPoint)
	require.Equal(t, 2, bitset)
}

func TestBalanceHistoryCheckMissingDataPoint(t *testing.T) {
	bh, cleanDB := setupTestBalanceHistoryDB(t)
	defer cleanDB()

	testDataPoint := generateTestDataForElementCount(1)[0]

	err := bh.addBalanceEntryToDB(testDataPoint, 20)
	require.NoError(t, err)

	missingDataPoint := testDataPoint
	missingDataPoint.block = big.NewInt(12)

	outDataPoint, bitset, err := bh.getBalanceEntryFromDB(missingDataPoint.chainID, missingDataPoint.address, missingDataPoint.currency, missingDataPoint.block)
	require.NoError(t, err)
	require.Nil(t, outDataPoint)
	require.Equal(t, 0, bitset)
}

func TestBalanceHistoryDataPointUniquenessConstraint(t *testing.T) {
	bh, cleanDB := setupTestBalanceHistoryDB(t)
	defer cleanDB()

	testDataPoint := generateTestDataForElementCount(1)[0]

	err := bh.addBalanceEntryToDB(testDataPoint, 1)
	require.NoError(t, err)

	testDataPointSame := testDataPoint
	testDataPointSame.balance = big.NewInt(102)
	testDataPointSame.timestamp = 12

	err = bh.addBalanceEntryToDB(testDataPointSame, 1)
	require.ErrorContains(t, err, "UNIQUE constraint failed", "should fail because of uniqueness constraint")

	rows, err := bh.db.Query("SELECT * FROM balance_history")
	require.NoError(t, err)

	ok := rows.Next()
	require.True(t, ok)
	ok = rows.Next()
	require.False(t, ok)

	testDataPointNew := testDataPointSame
	testDataPointNew.block = big.NewInt(21)

	err = bh.addBalanceEntryToDB(testDataPointNew, 277)
	require.NoError(t, err)

	rows, err = bh.db.Query("SELECT * FROM balance_history")
	require.NoError(t, err)

	ok = rows.Next()
	require.True(t, ok)
	ok = rows.Next()
	require.True(t, ok)
	ok = rows.Next()
	require.False(t, ok)

	outDataPoint, bitset, err := bh.getBalanceEntryFromDB(testDataPointNew.chainID, testDataPointNew.address, testDataPointNew.currency, testDataPointNew.block)
	require.NoError(t, err)
	require.NotEqual(t, outDataPoint, nil)
	require.Equal(t, testDataPointNew, outDataPoint)
	require.Equal(t, 277, bitset)
}

func TestBalanceHistoryGetOldestDataPoint(t *testing.T) {
	bh, cleanDB := setupTestBalanceHistoryDB(t)
	defer cleanDB()

	testDataPoints := generateTestDataForElementCount(5)
	for i := len(testDataPoints) - 1; i >= 0; i-- {
		err := bh.addBalanceEntryToDB(testDataPoints[i], 1)
		require.NoError(t, err)
	}

	outDataPoints, err := bh.getDBBalanceEntriesTimeSortedAsc(&bhIdentity{testDataPoints[0].chainID, testDataPoints[0].address, testDataPoints[0].currency}, nil, 1, 1)
	require.NoError(t, err)
	require.NotEqual(t, outDataPoints, nil)
	require.Equal(t, outDataPoints[0], testDataPoints[0])
}

func TestBalanceHistoryGetLatestDataPoint(t *testing.T) {
	bh, cleanDB := setupTestBalanceHistoryDB(t)
	defer cleanDB()

	testDataPoints := generateTestDataForElementCount(5)
	for i := 0; i < len(testDataPoints); i++ {
		err := bh.addBalanceEntryToDB(testDataPoints[i], 1)
		require.NoError(t, err)
	}

	outDataPoints, err := bh.getDBBalanceEntriesTimeSortedDesc(&bhIdentity{testDataPoints[0].chainID, testDataPoints[0].address, testDataPoints[0].currency}, nil, 1, 1)
	require.NoError(t, err)
	require.NotEqual(t, outDataPoints, nil)
	require.Equal(t, outDataPoints[0], testDataPoints[len(testDataPoints)-1])
}

func TestBalanceHistoryGetClosestDataPointToTimestamp(t *testing.T) {
	bh, cleanDB := setupTestBalanceHistoryDB(t)
	defer cleanDB()

	testDataPoints := generateTestDataForElementCount(5)
	for i := 0; i < len(testDataPoints); i++ {
		err := bh.addBalanceEntryToDB(testDataPoints[i], 1)
		require.NoError(t, err)
	}

	itemToGetIndex := 2
	outDataPoints, err := bh.getDBBalanceEntriesByTimeIntervalAndSortedAsc(&bhIdentity{testDataPoints[0].chainID, testDataPoints[0].address, testDataPoints[0].currency}, nil, &bhFilter{testDataPoints[itemToGetIndex].timestamp, maxAllRangeTimestamp, 1}, 1)
	require.NoError(t, err)
	require.NotEqual(t, outDataPoints, nil)
	require.Equal(t, len(outDataPoints), 1)
	require.Equal(t, outDataPoints[0], testDataPoints[itemToGetIndex])
}

func TestBalanceHistoryGetDataPointsInTimeRange(t *testing.T) {
	bh, cleanDB := setupTestBalanceHistoryDB(t)
	defer cleanDB()

	testDataPoints := generateTestDataForElementCount(5)
	for i := 0; i < len(testDataPoints); i++ {
		err := bh.addBalanceEntryToDB(testDataPoints[i], 1)
		require.NoError(t, err)
	}

	startIndex := 1
	endIndex := 3
	outDataPoints, err := bh.getDBBalanceEntriesByTimeIntervalAndSortedAsc(&bhIdentity{testDataPoints[0].chainID, testDataPoints[0].address, testDataPoints[0].currency}, nil, &bhFilter{testDataPoints[startIndex].timestamp, testDataPoints[endIndex].timestamp, 1}, 100)
	require.NoError(t, err)
	require.NotEqual(t, outDataPoints, nil)
	require.Equal(t, len(outDataPoints), endIndex-startIndex+1)
	for i := startIndex; i <= endIndex; i++ {
		require.Equal(t, outDataPoints[i-startIndex], testDataPoints[i])
	}
}
