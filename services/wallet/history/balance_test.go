package history

import (
	"context"
	"errors"
	"math"
	"math/big"
	"sort"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/sqlite"

	"github.com/stretchr/testify/require"
)

func setupBalanceTest(t *testing.T) (*Balance, func()) {
	db, err := appdatabase.InitializeDB(":memory:", "wallet-history-balance-tests", sqlite.ReducedKDFIterationsNumber)
	require.NoError(t, err)
	return NewBalance(NewBalanceDB(db)), func() {
		require.NoError(t, db.Close())
	}
}

type requestedBlock struct {
	time               uint64
	headerInfoRequests int
	balanceRequests    int
}

// chainClientTestSource is a test implementation of the DataSource interface
// It generates dummy consecutive blocks of data and stores them for validation
type chainClientTestSource struct {
	t                   *testing.T
	firstTimeRequest    int64
	requestedBlocks     map[int64]*requestedBlock // map of block number to block data
	lastBlockTimestamp  int64
	firstBlockTimestamp int64
	headerByNumberFn    func(ctx context.Context, number *big.Int) (*types.Header, error)
	balanceAtFn         func(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
	mockTime            int64
	timeAtMock          int64
}

const (
	testTimeLayout = "2006-01-02 15:04:05 Z07:00"
	testTime       = "2022-12-15 12:01:10 +02:00"
	oneYear        = 365 * 24 * time.Hour
)

func getTestTime(t *testing.T) time.Time {
	testTime, err := time.Parse(testTimeLayout, testTime)
	require.NoError(t, err)
	return testTime.UTC()
}

func newTestSource(t *testing.T, availableYears float64) *chainClientTestSource {
	return newTestSourceWithCurrentTime(t, availableYears, getTestTime(t).Unix())
}

func newTestSourceWithCurrentTime(t *testing.T, availableYears float64, currentTime int64) *chainClientTestSource {
	newInst := &chainClientTestSource{
		t:                   t,
		requestedBlocks:     make(map[int64]*requestedBlock),
		lastBlockTimestamp:  currentTime,
		firstBlockTimestamp: currentTime - int64(availableYears*oneYear.Seconds()),
		mockTime:            currentTime,
		timeAtMock:          time.Now().UTC().Unix(),
	}
	newInst.headerByNumberFn = newInst.HeaderByNumberMock
	newInst.balanceAtFn = newInst.BalanceAtMock
	return newInst
}

const (
	averageBlockTimeSeconds = 12.1
)

func (src *chainClientTestSource) setCurrentTime(newTime int64) {
	src.mockTime = newTime
	src.lastBlockTimestamp = newTime
}

func (src *chainClientTestSource) resetStats() {
	src.requestedBlocks = make(map[int64]*requestedBlock)
}

func (src *chainClientTestSource) availableYears() float64 {
	return float64(src.TimeNow()-src.firstBlockTimestamp) / oneYear.Seconds()
}

func (src *chainClientTestSource) blocksCount() int64 {
	return int64(math.Round(float64(src.TimeNow()-src.firstBlockTimestamp) / averageBlockTimeSeconds))
}

func (src *chainClientTestSource) blockNumberToTimestamp(number int64) int64 {
	return src.firstBlockTimestamp + int64(float64(number)*averageBlockTimeSeconds)
}

func (src *chainClientTestSource) generateBlockInfo(blockNumber int64, time uint64) *types.Header {
	return &types.Header{
		Number: big.NewInt(blockNumber),
		Time:   time,
	}
}

func (src *chainClientTestSource) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return src.headerByNumberFn(ctx, number)
}

func (src *chainClientTestSource) HeaderByNumberMock(ctx context.Context, number *big.Int) (*types.Header, error) {
	var blockNo int64
	if number == nil {
		// Last block was requested
		blockNo = src.blocksCount()
	} else if number.Cmp(big.NewInt(src.blocksCount())) > 0 {
		return nil, ethereum.NotFound
	} else {
		require.Greater(src.t, number.Int64(), int64(0))
		blockNo = number.Int64()
	}
	timestamp := src.blockNumberToTimestamp(blockNo)

	if _, contains := src.requestedBlocks[blockNo]; contains {
		src.requestedBlocks[blockNo].headerInfoRequests++
	} else {
		src.requestedBlocks[blockNo] = &requestedBlock{
			time:               uint64(timestamp),
			headerInfoRequests: 1,
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

func (src *chainClientTestSource) BalanceAtMock(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	var blockNo int64
	if blockNumber == nil {
		// Last block was requested
		blockNo = src.blocksCount()
	} else if blockNumber.Cmp(big.NewInt(src.blocksCount())) > 0 {
		return nil, ethereum.NotFound
	} else {
		require.Greater(src.t, blockNumber.Int64(), int64(0))
		blockNo = blockNumber.Int64()
	}

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

		if rB.headerInfoRequests > 0 {
			infoRequests[n] = rB.headerInfoRequests
		}
		if rB.balanceRequests > 0 {
			balanceRequests[n] = rB.balanceRequests
		}
	}
	return
}

func minimumExpectedDataPoints(interval TimeInterval) int {
	return int(math.Ceil(float64(timeIntervalDuration[interval]) / float64(strideDuration(interval))))
}

func getTimeError(dataSource *chainClientTestSource, data []*DataPoint, interval TimeInterval) int64 {
	timeRange := int64(data[len(data)-1].Timestamp - data[0].Timestamp)
	var expectedDuration int64
	if interval != BalanceHistoryAllTime {
		expectedDuration = int64(timeIntervalDuration[interval].Seconds())
	} else {
		expectedDuration = int64((time.Duration(dataSource.availableYears()) * oneYear).Seconds())
	}
	return timeRange - expectedDuration
}

func TestBalanceHistoryGetWithoutFetch(t *testing.T) {
	bh, cleanDB := setupBalanceTest(t)
	defer cleanDB()

	dataSource := newTestSource(t, 20 /*years*/)
	currentTimestamp := dataSource.TimeNow()

	testData := []struct {
		name     string
		interval TimeInterval
	}{
		{"Week", BalanceHistory7Days},
		{"Month", BalanceHistory1Month},
		{"HalfYear", BalanceHistory6Months},
		{"Year", BalanceHistory1Year},
		{"AllTime", BalanceHistoryAllTime},
	}
	for _, testInput := range testData {
		t.Run(testInput.name, func(t *testing.T) {
			balanceData, err := bh.get(context.Background(), dataSource.ChainID(), dataSource.Currency(), common.Address{7}, currentTimestamp, testInput.interval)
			require.NoError(t, err)
			require.Equal(t, 0, len(balanceData))
		})
	}
}

func TestBalanceHistoryGetWithoutOverlappingFetch(t *testing.T) {
	testData := []struct {
		name     string
		interval TimeInterval
	}{
		{"Week", BalanceHistory7Days},
		{"Month", BalanceHistory1Month},
		{"HalfYear", BalanceHistory6Months},
		{"Year", BalanceHistory1Year},
		{"AllTime", BalanceHistoryAllTime},
	}
	for _, testInput := range testData {
		t.Run(testInput.name, func(t *testing.T) {
			bh, cleanDB := setupBalanceTest(t)
			defer cleanDB()

			dataSource := newTestSource(t, 20 /*years*/)
			currentTimestamp := dataSource.TimeNow()
			getUntilTimestamp := currentTimestamp - int64((400 /*days*/ * 24 * time.Hour).Seconds())

			fetchInterval := testInput.interval + 3
			if fetchInterval > BalanceHistoryAllTime {
				fetchInterval = BalanceHistory7Days + BalanceHistoryAllTime - testInput.interval
			}
			err := bh.update(context.Background(), dataSource, common.Address{7}, fetchInterval)
			require.NoError(t, err)

			balanceData, err := bh.get(context.Background(), dataSource.ChainID(), dataSource.Currency(), common.Address{7}, getUntilTimestamp, testInput.interval)
			require.NoError(t, err)
			require.Equal(t, 0, len(balanceData))
		})
	}
}

func TestBalanceHistoryGetWithOverlappingFetch(t *testing.T) {
	testData := []struct {
		name          string
		interval      TimeInterval
		lessDaysToGet int
	}{
		{"Week", BalanceHistory7Days, 6},
		{"Month", BalanceHistory1Month, 1},
		{"HalfYear", BalanceHistory6Months, 8},
		{"Year", BalanceHistory1Year, 16},
		{"AllTime", BalanceHistoryAllTime, 130},
	}
	for _, testInput := range testData {
		t.Run(testInput.name, func(t *testing.T) {
			bh, cleanDB := setupBalanceTest(t)
			defer cleanDB()

			dataSource := newTestSource(t, 20 /*years*/)
			currentTimestamp := dataSource.TimeNow()
			olderUntilTimestamp := currentTimestamp - int64((time.Duration(testInput.lessDaysToGet) * 24 * time.Hour).Seconds())

			err := bh.update(context.Background(), dataSource, common.Address{7}, testInput.interval)
			require.NoError(t, err)

			balanceData, err := bh.get(context.Background(), dataSource.ChainID(), dataSource.Currency(), common.Address{7}, currentTimestamp, testInput.interval)
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(balanceData), minimumExpectedDataPoints(testInput.interval))

			olderBalanceData, err := bh.get(context.Background(), dataSource.ChainID(), dataSource.Currency(), common.Address{7}, olderUntilTimestamp, testInput.interval)
			require.NoError(t, err)
			require.Less(t, len(olderBalanceData), len(balanceData))
		})
	}
}

func TestBalanceHistoryFetchFirstTime(t *testing.T) {
	testData := []struct {
		name     string
		interval TimeInterval
	}{
		{"Week", BalanceHistory7Days},
		{"Month", BalanceHistory1Month},
		{"HalfYear", BalanceHistory6Months},
		{"Year", BalanceHistory1Year},
		{"AllTime", BalanceHistoryAllTime},
	}
	for _, testInput := range testData {
		t.Run(testInput.name, func(t *testing.T) {
			bh, cleanDB := setupBalanceTest(t)
			defer cleanDB()

			dataSource := newTestSource(t, 20 /*years*/)
			currentTimestamp := dataSource.TimeNow()

			err := bh.update(context.Background(), dataSource, common.Address{7}, testInput.interval)
			require.NoError(t, err)

			balanceData, err := bh.get(context.Background(), dataSource.ChainID(), dataSource.Currency(), common.Address{7}, currentTimestamp, testInput.interval)
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(balanceData), minimumExpectedDataPoints(testInput.interval))

			reqBlkNos, headerInfos, balances := extractTestData(dataSource)
			require.Equal(t, len(balanceData), len(balances))

			// Ensure we don't request the same info twice
			for block, count := range headerInfos {
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
					require.Equal(t, rB.time, balanceData[resIdx].Timestamp)
					if resIdx > 0 {
						require.Greater(t, balanceData[resIdx].Timestamp, balanceData[resIdx-1].Timestamp, "result timestamps are in order")
					}
					resIdx++
				}
			}

			errorFromIdeal := getTimeError(dataSource, balanceData, testInput.interval)
			require.Less(t, math.Abs(float64(errorFromIdeal)), strideDuration(testInput.interval).Seconds(), "Duration error [%d s] is within 1 stride [%.f s] for interval [%#v]", errorFromIdeal, strideDuration(testInput.interval).Seconds(), testInput.interval)
		})
	}
}

func TestBalanceHistoryFetchError(t *testing.T) {
	bh, cleanDB := setupBalanceTest(t)
	defer cleanDB()

	dataSource := newTestSource(t, 20 /*years*/)
	bkFn := dataSource.balanceAtFn
	// Fail first request
	dataSource.balanceAtFn = func(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
		return nil, errors.New("test error")
	}
	currentTimestamp := dataSource.TimeNow()
	err := bh.update(context.Background(), dataSource, common.Address{7}, BalanceHistory1Year)
	require.Error(t, err, "Expect \"test error\"")

	balanceData, err := bh.get(context.Background(), dataSource.ChainID(), dataSource.Currency(), common.Address{7}, currentTimestamp, BalanceHistory1Year)
	require.NoError(t, err)
	require.Equal(t, 0, len(balanceData))

	_, headerInfos, balances := extractTestData(dataSource)
	require.Equal(t, 0, len(balances))
	require.Equal(t, 1, len(headerInfos))

	dataSource.resetStats()
	// Fail later
	dataSource.balanceAtFn = func(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
		if len(dataSource.requestedBlocks) == 15 {
			return nil, errors.New("test error")
		}
		return dataSource.BalanceAtMock(ctx, account, blockNumber)
	}
	err = bh.update(context.Background(), dataSource, common.Address{7}, BalanceHistory1Year)
	require.Error(t, err, "Expect \"test error\"")

	balanceData, err = bh.get(context.Background(), dataSource.ChainID(), dataSource.Currency(), common.Address{7}, currentTimestamp, BalanceHistory1Year)
	require.NoError(t, err)
	require.Equal(t, 14, len(balanceData))

	reqBlkNos, headerInfos, balances := extractTestData(dataSource)
	// The request for block info is made before the balance request
	require.Equal(t, 1, dataSource.requestedBlocks[reqBlkNos[0]].headerInfoRequests)
	require.Equal(t, 0, dataSource.requestedBlocks[reqBlkNos[0]].balanceRequests)
	require.Equal(t, 14, len(balances))
	require.Equal(t, len(balances), len(headerInfos)-1)

	dataSource.resetStats()
	dataSource.balanceAtFn = bkFn
	err = bh.update(context.Background(), dataSource, common.Address{7}, BalanceHistory1Year)
	require.NoError(t, err)

	balanceData, err = bh.get(context.Background(), dataSource.ChainID(), dataSource.Currency(), common.Address{7}, currentTimestamp, BalanceHistory1Year)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(balanceData), minimumExpectedDataPoints(BalanceHistory1Year))

	_, headerInfos, balances = extractTestData(dataSource)
	// Account for cache hits
	require.Equal(t, len(balanceData)-14, len(balances))
	require.Equal(t, len(balances), len(headerInfos))

	for i := 1; i < len(balanceData); i++ {
		require.Greater(t, balanceData[i].Timestamp, balanceData[i-1].Timestamp, "result timestamps are in order")
	}

	errorFromIdeal := getTimeError(dataSource, balanceData, BalanceHistory1Year)
	require.Less(t, math.Abs(float64(errorFromIdeal)), strideDuration(BalanceHistory1Year).Seconds(), "Duration error [%d s] is within 1 stride [%.f s] for interval [%#v]", errorFromIdeal, strideDuration(BalanceHistory1Year).Seconds(), BalanceHistory1Year)
}

func TestBalanceHistoryValidateBalanceValuesAndCacheHit(t *testing.T) {
	bh, cleanDB := setupBalanceTest(t)
	defer cleanDB()

	dataSource := newTestSource(t, 20 /*years*/)
	currentTimestamp := dataSource.TimeNow()
	requestedBalance := make(map[int64]*big.Int)
	dataSource.balanceAtFn = func(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
		balance, err := dataSource.BalanceAtMock(ctx, account, blockNumber)
		requestedBalance[blockNumber.Int64()] = new(big.Int).Set(balance)
		return balance, err
	}

	testData := []struct {
		name     string
		interval TimeInterval
	}{
		{"Week", BalanceHistory7Days},
		{"Month", BalanceHistory1Month},
		{"HalfYear", BalanceHistory6Months},
		{"Year", BalanceHistory1Year},
		{"AllTime", BalanceHistoryAllTime},
	}
	for _, testInput := range testData {
		t.Run(testInput.name, func(t *testing.T) {
			dataSource.resetStats()
			err := bh.update(context.Background(), dataSource, common.Address{7}, testInput.interval)
			require.NoError(t, err)

			balanceData, err := bh.get(context.Background(), dataSource.ChainID(), dataSource.Currency(), common.Address{7}, currentTimestamp, testInput.interval)
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(balanceData), minimumExpectedDataPoints(testInput.interval))

			reqBlkNos, headerInfos, _ := extractTestData(dataSource)
			// Only first run is not affected by cache
			if testInput.interval == BalanceHistory7Days {
				require.Equal(t, len(balanceData), len(requestedBalance))
				require.Equal(t, len(balanceData), len(headerInfos))
			} else {
				require.Greater(t, len(balanceData), len(requestedBalance))
				require.Greater(t, len(balanceData), len(headerInfos))
			}

			resIdx := 0
			// Check that balance values are the one requested
			for i := 0; i < len(reqBlkNos); i++ {
				n := reqBlkNos[i]

				if value, contains := requestedBalance[n]; contains {
					require.Equal(t, value.Cmp(balanceData[resIdx].Balance.ToInt()), 0)
					resIdx++
				}
				blockHeaderRequestCount := dataSource.requestedBlocks[n].headerInfoRequests
				require.Less(t, blockHeaderRequestCount, 2)
				blockBalanceRequestCount := dataSource.requestedBlocks[n].balanceRequests
				require.Less(t, blockBalanceRequestCount, 2)
			}

			// Check that balance values are in order
			for i := 1; i < len(balanceData); i++ {
				require.Greater(t, balanceData[i].Balance.ToInt().Cmp(balanceData[i-1].Balance.ToInt()), 0, "expected balanceData[%d] > balanceData[%d] for interval %d", i, i-1, testInput.interval)
			}
			requestedBalance = make(map[int64]*big.Int)
		})
	}
}

func TestGetBalanceHistoryUpdateLater(t *testing.T) {
	bh, cleanDB := setupBalanceTest(t)
	defer cleanDB()

	currentTime := getTestTime(t)
	initialTime := currentTime
	moreThanADay := 24*time.Hour + 15*time.Minute
	moreThanAMonth := 401 * moreThanADay
	initialTime = initialTime.Add(-moreThanADay - moreThanAMonth)
	dataSource := newTestSourceWithCurrentTime(t, 20 /*years*/, initialTime.Unix())

	err := bh.update(context.Background(), dataSource, common.Address{7}, BalanceHistory1Month)
	require.NoError(t, err)

	prevBalanceData, err := bh.get(context.Background(), dataSource.ChainID(), dataSource.Currency(), common.Address{7}, dataSource.TimeNow(), BalanceHistory1Month)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(prevBalanceData), minimumExpectedDataPoints(BalanceHistory1Month))

	// Advance little bit more than a day
	later := initialTime
	later = later.Add(moreThanADay)
	dataSource.setCurrentTime(later.Unix())
	dataSource.resetStats()

	err = bh.update(context.Background(), dataSource, common.Address{7}, BalanceHistory1Month)
	require.NoError(t, err)

	updatedBalanceData, err := bh.get(context.Background(), dataSource.ChainID(), dataSource.Currency(), common.Address{7}, dataSource.TimeNow(), BalanceHistory1Month)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(updatedBalanceData), minimumExpectedDataPoints(BalanceHistory1Month))

	reqBlkNos, blockInfos, balances := extractTestData(dataSource)
	require.Equal(t, 2, len(reqBlkNos))
	require.Equal(t, len(reqBlkNos), len(blockInfos))
	require.Equal(t, len(blockInfos), len(balances))

	for block, count := range balances {
		require.Equal(t, 1, count, "block %d has one request", block)
	}

	resIdx := len(updatedBalanceData) - 2
	for i := 0; i < len(reqBlkNos); i++ {
		rB := dataSource.requestedBlocks[reqBlkNos[i]]

		// Ensure block approximation error doesn't exceed 10 blocks
		require.Equal(t, 0.0, math.Abs(float64(int64(rB.time)-int64(updatedBalanceData[resIdx].Timestamp))))
		if resIdx > 0 {
			// Ensure result timestamps are in order
			require.Greater(t, updatedBalanceData[resIdx].Timestamp, updatedBalanceData[resIdx-1].Timestamp)
		}
		resIdx++
	}

	errorFromIdeal := getTimeError(dataSource, updatedBalanceData, BalanceHistory1Month)
	require.Less(t, math.Abs(float64(errorFromIdeal)), strideDuration(BalanceHistory1Month).Seconds(), "Duration error [%d s] is within 1 stride [%.f s] for interval [%#v]", errorFromIdeal, strideDuration(BalanceHistory1Month).Seconds(), BalanceHistory1Month)

	// Advance little bit more than a month
	dataSource.setCurrentTime(currentTime.Unix())
	dataSource.resetStats()

	err = bh.update(context.Background(), dataSource, common.Address{7}, BalanceHistory1Month)
	require.NoError(t, err)

	newBalanceData, err := bh.get(context.Background(), dataSource.ChainID(), dataSource.Currency(), common.Address{7}, dataSource.TimeNow(), BalanceHistory1Month)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(newBalanceData), minimumExpectedDataPoints(BalanceHistory1Month))

	_, headerInfos, balances := extractTestData(dataSource)
	require.Greater(t, len(balances), len(newBalanceData), "Expected more balance requests due to missing time catch up")

	// Ensure we don't request the same info twice
	for block, count := range headerInfos {
		require.Equal(t, 1, count, "block %d has one info request", block)
		if balanceCount, contains := balances[block]; contains {
			require.Equal(t, 1, balanceCount, "block %d has one balance request", block)
		}
	}
	for block, count := range balances {
		require.Equal(t, 1, count, "block %d has one request", block)
	}

	for i := 1; i < len(newBalanceData); i++ {
		require.Greater(t, newBalanceData[i].Timestamp, newBalanceData[i-1].Timestamp, "result timestamps are in order")
	}

	errorFromIdeal = getTimeError(dataSource, newBalanceData, BalanceHistory1Month)
	require.Less(t, math.Abs(float64(errorFromIdeal)), strideDuration(BalanceHistory1Month).Seconds(), "Duration error [%d s] is within 1 stride [%.f s] for interval [%#v]", errorFromIdeal, strideDuration(BalanceHistory1Month).Seconds(), BalanceHistory1Month)
}

func TestGetBalanceHistoryFetchMultipleAccounts(t *testing.T) {
	bh, cleanDB := setupBalanceTest(t)
	defer cleanDB()

	sevenDataSource := newTestSource(t, 5 /*years*/)

	err := bh.update(context.Background(), sevenDataSource, common.Address{7}, BalanceHistory1Month)
	require.NoError(t, err)

	sevenBalanceData, err := bh.get(context.Background(), sevenDataSource.ChainID(), sevenDataSource.Currency(), common.Address{7}, sevenDataSource.TimeNow(), BalanceHistory1Month)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(sevenBalanceData), minimumExpectedDataPoints(BalanceHistory1Month))

	_, sevenBlockInfos, _ := extractTestData(sevenDataSource)
	require.Greater(t, len(sevenBlockInfos), 0)

	nineDataSource := newTestSource(t, 5 /*years*/)
	err = bh.update(context.Background(), nineDataSource, common.Address{9}, BalanceHistory1Month)
	require.NoError(t, err)

	nineBalanceData, err := bh.get(context.Background(), nineDataSource.ChainID(), nineDataSource.Currency(), common.Address{7}, nineDataSource.TimeNow(), BalanceHistory1Month)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(nineBalanceData), minimumExpectedDataPoints(BalanceHistory1Month))

	_, nineBlockInfos, nineBalances := extractTestData(nineDataSource)
	require.Equal(t, 0, len(nineBlockInfos))
	require.Equal(t, len(nineBalanceData), len(nineBalances))
}

func TestGetBalanceHistoryUpdateCancellation(t *testing.T) {
	bh, cleanDB := setupBalanceTest(t)
	defer cleanDB()

	dataSource := newTestSource(t, 5 /*years*/)
	ctx, cancelFn := context.WithCancel(context.Background())
	bkFn := dataSource.balanceAtFn
	// Fail after 15 requests
	dataSource.balanceAtFn = func(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
		if len(dataSource.requestedBlocks) == 15 {
			cancelFn()
		}
		return dataSource.BalanceAtMock(ctx, account, blockNumber)
	}
	err := bh.update(ctx, dataSource, common.Address{7}, BalanceHistory1Year)
	require.Error(t, ctx.Err(), "Service canceled")
	require.Error(t, err, "context cancelled")

	balanceData, err := bh.get(context.Background(), dataSource.ChainID(), dataSource.Currency(), common.Address{7}, dataSource.TimeNow(), BalanceHistory1Year)
	require.NoError(t, err)
	require.Equal(t, 15, len(balanceData))

	_, blockInfos, balances := extractTestData(dataSource)
	// The request for block info is made before the balance fails
	require.Equal(t, 15, len(balances))
	require.Equal(t, 15, len(blockInfos))

	dataSource.balanceAtFn = bkFn
	ctx, cancelFn = context.WithCancel(context.Background())

	err = bh.update(ctx, dataSource, common.Address{7}, BalanceHistory1Year)
	require.NoError(t, ctx.Err())
	require.NoError(t, err)

	balanceData, err = bh.get(context.Background(), dataSource.ChainID(), dataSource.Currency(), common.Address{7}, dataSource.TimeNow(), BalanceHistory1Year)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(balanceData), minimumExpectedDataPoints(BalanceHistory1Year))
	cancelFn()
}

func TestBlockStrideHaveCommonDivisor(t *testing.T) {
	values := make([]blocksStride, 0, len(timeIntervalToStride))
	for _, blockCount := range timeIntervalToStride {
		values = append(values, blockCount)
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i] < values[j]
	})
	for i := 1; i < len(values); i++ {
		require.Equal(t, blocksStride(0), values[i]%values[i-1], " %d value from index %d is divisible with previous %d", values[i], i, values[i-1])
	}
}

func TestBlockStrideMatchesBitsetFilter(t *testing.T) {
	filterToStrideEquivalence := map[bitsetFilter]blocksStride{
		filterAllTime:   fourMonthsStride,
		filterWeekly:    weekStride,
		filterTwiceADay: twiceADayStride,
	}

	for interval, bitsetFiler := range timeIntervalToBitsetFilter {
		stride, found := timeIntervalToStride[interval]
		require.True(t, found)
		require.Equal(t, stride, filterToStrideEquivalence[bitsetFiler])
	}
}

func TestTimeIntervalToBitsetFilterAreConsecutiveFlags(t *testing.T) {
	values := make([]int, 0, len(timeIntervalToBitsetFilter))
	for i := BalanceHistoryAllTime; i >= BalanceHistory7Days; i-- {
		values = append(values, int(timeIntervalToBitsetFilter[i]))
	}

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
			require.GreaterOrEqual(t, values[i], values[i-1], "%b value from index %d is higher then previous %d", values[i], i, values[i-1])
		}
	}
}
