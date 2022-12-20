package transfer

import (
	"context"
	"database/sql"
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"

	"github.com/pkg/errors"

	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/chain"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

// EventBalanceHistoryUpdateStarted and EventBalanceHistoryUpdateDone are used to notify the UI that balance history is being updated
const EventBalanceHistoryUpdateStarted walletevent.EventType = "wallet-balance-history-update-started"
const EventBalanceHistoryUpdateFinished walletevent.EventType = "wallet-balance-history-update-finished"

type BalanceHistory struct {
	db *sql.DB

	asyncQueue *async.PriorityBasedAsyncQueue
	walletFeed *event.Feed
}

func NewBalanceHistory(db *sql.DB, walletFeed *event.Feed) *BalanceHistory {
	return &BalanceHistory{
		db:         db,
		asyncQueue: async.NewPriorityBasedAsyncQueue(context.Background()),
		walletFeed: walletFeed}
}

type blockInfo struct {
	block     *big.Int
	timestamp int64
}

type balanceHistoryDBEntry struct {
	chainID   uint64
	address   common.Address
	currency  string
	block     *big.Int
	timestamp int64
	balance   *big.Int
}

type BalanceState struct {
	Value     *hexutil.Big `json:"value"`
	Timestamp uint64       `json:"time"`
}

const (
	updateBalanceHistoryPriority async.TaskPriority = iota + 1
	getBalanceHistoryPriority
)

type BalanceHistoryTimeInterval int

const (
	maxAllRangeTimestamp         = math.MaxInt64
	minAllRangeTimestamp         = 0
	balanceHistoryUpdateInterval = 12 * time.Hour
	calibrationUpdateInterval    = 7 * 24 * time.Hour
)

// Specific time intervals for which balance history can be fetched
const (
	BalanceHistory7Hours BalanceHistoryTimeInterval = iota + 1
	BalanceHistory1Month
	BalanceHistory6Months
	BalanceHistory1Year
	BalanceHistoryAllTime
)

var secondsInTimeInterval = map[BalanceHistoryTimeInterval]int64{
	BalanceHistory7Hours:  7 * 24 * 60 * 60,
	BalanceHistory1Month:  30 * 24 * 60 * 60,
	BalanceHistory6Months: 6 * 30 * 24 * 60 * 60,
	BalanceHistory1Year:   365 * 24 * 60 * 60,
	BalanceHistoryAllTime: 50 * 365 * 24 * 60 * 60,
}

const secondsInHour = 60 * 60

// This defines the granularity of fetched data points for each time interval
// All must have common divisor of the smallest for the best cache hit
var timeIntervalToHoursPerStep = map[BalanceHistoryTimeInterval]int64{
	BalanceHistory7Hours:  2,
	BalanceHistory1Month:  12,
	BalanceHistory6Months: (24 * 7) / 2,
	BalanceHistory1Year:   24 * 7,
	BalanceHistoryAllTime: 24 * 7 * 4 /*weeks*/ * 4, /*months*/
}

type snapUnit int

const (
	snapUnitDay snapUnit = iota + 1
	snapUnitMonth
	snapUnitYear
)

var timeIntervalToSnapUnit = map[BalanceHistoryTimeInterval]snapUnit{
	BalanceHistory7Hours:  snapUnitDay,
	BalanceHistory1Month:  snapUnitDay,
	BalanceHistory6Months: snapUnitMonth,
	BalanceHistory1Year:   snapUnitMonth,
	BalanceHistoryAllTime: snapUnitMonth,
}

type TimeIntervalBitsetFilter int

// Bitset used to fetch relevant data points in one batch and to increase in memory cache hit instead of DB
const (
	FilterAllTime    TimeIntervalBitsetFilter = 1
	Filter1Year      TimeIntervalBitsetFilter = 1 << 3
	Filter6Months    TimeIntervalBitsetFilter = 1 << 5
	Filter1Month     TimeIntervalBitsetFilter = 1 << 7
	Filter7Hours     TimeIntervalBitsetFilter = 1 << 9
	FilterIncludeAll TimeIntervalBitsetFilter = 1 << 30
)

var timeIntervalToBitsetFilter = map[BalanceHistoryTimeInterval]TimeIntervalBitsetFilter{
	BalanceHistory7Hours:  Filter7Hours,
	BalanceHistory1Month:  Filter1Month,
	BalanceHistory6Months: Filter6Months,
	BalanceHistory1Year:   Filter1Year,
	BalanceHistoryAllTime: FilterAllTime,
}

type BlockInfoSource interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
	ChainID() uint64
	Currency() string
	TimeNow() int64
}

func (bh *BalanceHistory) fetchAndCacheBalanceForBlock(ctx context.Context, chainClient BlockInfoSource, address common.Address, blockNo *big.Int, timestamp int64, bitset TimeIntervalBitsetFilter) (*BalanceState, error) {
	currentBalance, err := chainClient.BalanceAt(ctx, address, blockNo)
	if err != nil {
		return nil, err
	}

	dataPoint := balanceHistoryDBEntry{
		chainID:   chainClient.ChainID(),
		address:   address,
		currency:  chainClient.Currency(),
		block:     new(big.Int).Set(blockNo),
		balance:   currentBalance,
		timestamp: timestamp,
	}
	err = bh.addBalanceEntryToDB(&dataPoint, int(bitset))
	if err != nil {
		return nil, err
	}

	var currentBalanceState BalanceState
	currentBalanceState.Value = (*hexutil.Big)(currentBalance)
	currentBalanceState.Timestamp = uint64(timestamp)
	return &currentBalanceState, nil
}

func averageBlockDuration(startBlock *big.Int, endBlock *big.Int, startTime int64, endTime int64) float64 {
	blockCount := new(big.Int).Sub(endBlock, startBlock).Int64()
	return float64(endTime-startTime) / float64(blockCount)
}

// calibrationAddress returns the reserved, unused, block number to filter out calibration data which is account agnostic
func calibrationAddress() common.Address {
	return common.Address{0}
}

const (
	calibrationCurrency = ""
	calibrationBitset   = 1
)

func (bh *BalanceHistory) getOrFetchCalibrationData(ctx context.Context, chainClient BlockInfoSource) ([]*blockInfo, error) {
	dbData, err := bh.getDBBalanceEntriesTimeSortedAsc(&BhIdentity{chainClient.ChainID(), calibrationAddress(), calibrationCurrency}, nil, calibrationBitset, 1000)
	if err != nil {
		return nil, err
	}

	lastBlockInfo := &blockInfo{big.NewInt(1), 0}
	// It is ok for the initial fetching to use a very long interval, as it isn't as relevant
	calibrationInterval := 365 * 24 * time.Hour // 1 year
	lastAvgBlockTime := float64(12)
	nextBlockNo := big.NewInt(1)
	var prevBlockInfo *blockInfo
	result := make([]*blockInfo, 0)
	currentTime := chainClient.TimeNow()
	if len(dbData) > 0 {
		lastBlockInfo.timestamp = dbData[len(dbData)-1].timestamp
		lastBlockInfo.block.Set(dbData[len(dbData)-1].block)
		if len(dbData) > 1 {
			prevBlockInfo = &blockInfo{new(big.Int), dbData[len(dbData)-2].timestamp}
			prevBlockInfo.block.Set(dbData[len(dbData)-2].block)
			lastAvgBlockTime = averageBlockDuration(prevBlockInfo.block, lastBlockInfo.block, prevBlockInfo.timestamp, lastBlockInfo.timestamp)
		}

		// If the last block is newer then a year, use the coarse update interval
		if currentTime-lastBlockInfo.timestamp < secondsInTimeInterval[BalanceHistory1Year] {
			calibrationInterval = calibrationUpdateInterval
		}
		nextBlockNo.Add(lastBlockInfo.block, big.NewInt(int64(calibrationInterval.Seconds()/lastAvgBlockTime)))

		for _, dbEntry := range dbData {
			result = append(result, &blockInfo{new(big.Int).Set(dbEntry.block), dbEntry.timestamp})
		}
	}

	reachedPastLastBlock := false
	for currentTime-lastBlockInfo.timestamp > int64(calibrationUpdateInterval.Seconds()) && !reachedPastLastBlock {
		// Check context for cancellation every cycle
		select {
		case <-ctx.Done():
			return nil, errors.New("context cancelled")
		default:
		}

		// Fetch data for next block
		block, err := chainClient.BlockByNumber(ctx, nextBlockNo)
		if err != nil {
			if err == ethereum.NotFound {
				// We went too far; block average decreased, fetch the last block
				block, err = chainClient.BlockByNumber(ctx, nil)
				if err != nil {
					return nil, err
				}
				reachedPastLastBlock = true
			} else {
				return nil, err
			}
		}

		dataPoint := balanceHistoryDBEntry{
			chainID:   chainClient.ChainID(),
			address:   calibrationAddress(),
			currency:  calibrationCurrency,
			block:     new(big.Int).Set(block.Number()),
			balance:   big.NewInt(0),
			timestamp: int64(block.Time()),
		}
		err = bh.addBalanceEntryToDB(&dataPoint, calibrationBitset)
		if err != nil {
			return nil, err
		}
		if prevBlockInfo != nil {
			lastAvgBlockTime = averageBlockDuration(prevBlockInfo.block, block.Number(), prevBlockInfo.timestamp, int64(block.Time()))
		} else {
			prevBlockInfo = &blockInfo{new(big.Int), int64(block.Time())}
			prevBlockInfo.block.Set(block.Number())
		}

		lastBlockInfo = &blockInfo{new(big.Int).Set(block.Number()), int64(block.Time())}
		result = append(result, lastBlockInfo)
		nextBlockNo.Add(nextBlockNo, big.NewInt(int64(calibrationInterval.Seconds()/lastAvgBlockTime)))

		prevBlockInfo = &blockInfo{new(big.Int), int64(block.Time())}
		prevBlockInfo.block.Set(block.Number())
	}

	return result, nil
}

func getStepInfo(prevBlockInfo *blockInfo, blockInfo *blockInfo, timeInterval BalanceHistoryTimeInterval) (blocksInStep int64, stepDuration int64, avgBlockTime float64) {
	avgBlockTime = averageBlockDuration(prevBlockInfo.block, blockInfo.block, prevBlockInfo.timestamp, blockInfo.timestamp)
	idealStepDuration := timeIntervalToHoursPerStep[timeInterval] * secondsInHour
	blocksInStep = int64(float64(idealStepDuration) / avgBlockTime)
	stepDuration = int64(float64(blocksInStep) * avgBlockTime)
	return
}

func snapTimestamp(timestamp int64, timeInterval BalanceHistoryTimeInterval) int64 {
	original := time.Unix(timestamp, 0).In(time.UTC)

	month := original.Month()
	day := original.Day()
	hour := 0

	snapUnit := timeIntervalToSnapUnit[timeInterval]
	switch snapUnit {
	case snapUnitDay:
		d := original.Sub(time.Date(original.Year(), month, day, 0, 0, 0, 0, original.Location()))
		hour = int(int64(math.Floor(d.Hours()/float64(timeIntervalToHoursPerStep[timeInterval]))) * timeIntervalToHoursPerStep[timeInterval])
	case snapUnitMonth:
		monthSnap := time.Date(original.Year(), month, 1, 0, 0, 0, 0, original.Location())
		d := original.Sub(monthSnap)
		hours := int(int64(math.Floor(d.Hours()/float64(timeIntervalToHoursPerStep[timeInterval]))) * timeIntervalToHoursPerStep[timeInterval])
		rounded := monthSnap.Add(time.Duration(hours) * time.Hour)
		month = rounded.Month()
		day = rounded.Day()
		hour = rounded.Hour()
	case snapUnitYear:
		yearSnap := time.Date(original.Year(), 1, 1, 0, 0, 0, 0, original.Location())
		d := original.Sub(yearSnap)
		hours := int(int64(math.Floor(d.Hours()/float64(timeIntervalToHoursPerStep[timeInterval]))) * timeIntervalToHoursPerStep[timeInterval])
		rounded := yearSnap.Add(time.Duration(hours) * time.Hour)
		month = rounded.Month()
		day = 1
		hour = 0
	}

	rounded := time.Date(original.Year(), month, day, hour, 0, 0, 0, original.Location())
	return rounded.Unix()
}

func computeBlockInfoForTimestamp(timestamp int64, startCalibIdx int, calib []*blockInfo, timeInterval BalanceHistoryTimeInterval) (block *big.Int, blockTimestamp int64, calibIdx int) {
	if startCalibIdx == 0 {
		startCalibIdx = 1
	}
	calibIdx = startCalibIdx
	for i := startCalibIdx; i < len(calib) && calib[i-1].timestamp < timestamp; i++ {
		calibIdx = i
	}
	_, _, currentAvgBlockTime := getStepInfo(calib[calibIdx-1], calib[calibIdx], timeInterval)
	timeToPrevBlock := timestamp - calib[calibIdx-1].timestamp
	timeToNextBlock := timestamp - calib[calibIdx].timestamp
	// Compute relative to the closest calibration block to minimize the error
	if math.Abs(float64(timeToPrevBlock)) <= math.Abs(float64(timeToNextBlock)) {
		blocksCount := int64(math.Floor(float64(timeToPrevBlock) / currentAvgBlockTime))
		block = new(big.Int).Add(calib[calibIdx-1].block, big.NewInt(blocksCount))
		blockTimestamp = calib[calibIdx-1].timestamp + int64(float64(blocksCount)*currentAvgBlockTime)
	} else {
		blocksCount := int64(math.Floor(float64(timeToNextBlock) / currentAvgBlockTime))
		block = new(big.Int).Add(calib[calibIdx].block, big.NewInt(blocksCount))
		blockTimestamp = calib[calibIdx].timestamp + int64(float64(blocksCount)*currentAvgBlockTime)
	}
	return
}

// getDBBalanceEntriesByTimeIntervalAndSortedDesc returns nil if no entries are found
func (bh *BalanceHistory) mostRecentBalanceEntry(ctx context.Context, chainClient BlockInfoSource, address common.Address) (*BalanceState, error) {
	outDataPoints, err := bh.getDBBalanceEntriesByTimeIntervalAndSortedDesc(&BhIdentity{chainClient.ChainID(), address, chainClient.Currency()}, nil, &bhFilter{minAllRangeTimestamp, maxAllRangeTimestamp, expandFlag(int(FilterIncludeAll))}, 1)
	if err != nil {
		return nil, err
	}
	if len(outDataPoints) > 0 {
		return &BalanceState{
			Value:     (*hexutil.Big)(outDataPoints[0].balance),
			Timestamp: uint64(outDataPoints[0].timestamp),
		}, nil
	}
	return nil, nil
}

// expandFlag expand a flag to match all lower value flags (fills the less significant bits of the flag with 1; e.g. 0b1000 -> 0b1111)
func expandFlag(flag int) int {
	return (flag << 1) - 1
}

// getBalanceHistory expect a time precision of +/- average block time (~12s)
// implementation relies that a block has relative constant time length to save block header requests
func (bh *BalanceHistory) getBalanceHistoryFromBlocksSource(ctx context.Context, chainClient BlockInfoSource, address common.Address, timeInterval BalanceHistoryTimeInterval) ([]*BalanceState, error) {
	calib, err := bh.getOrFetchCalibrationData(ctx, chainClient)
	if err != nil {
		return nil, err
	} else if len(calib) < 2 {
		return nil, errors.New("not enough calibration data to compute average block time")
	}

	bitsetFilter := timeIntervalToBitsetFilter[timeInterval]
	currentTimestamp := chainClient.TimeNow()
	var startTimestamp int64
	var roundedTimestamp int64
	if timeInterval != BalanceHistoryAllTime {
		startTimestamp = currentTimestamp - secondsInTimeInterval[timeInterval]
		roundedTimestamp = snapTimestamp(startTimestamp, timeInterval)
	} else {
		startTimestamp = calib[0].timestamp
		roundedTimestamp = startTimestamp
	}

	var currentBlockNumber *big.Int
	var currentBlockTimestamp int64
	nextBlockTimestamp := startTimestamp
	calibIdx := 1
	if timeInterval != BalanceHistoryAllTime {
		currentBlockNumber, currentBlockTimestamp, calibIdx = computeBlockInfoForTimestamp(roundedTimestamp, 1, calib, timeInterval)
	} else {
		currentBlockNumber = new(big.Int).Set(calib[0].block)
		currentBlockTimestamp = calib[0].timestamp
	}

	outDataPoints, err := bh.getDBBalanceEntriesByTimeIntervalAndSortedAsc(&BhIdentity{chainClient.ChainID(), address, chainClient.Currency()}, nil, &bhFilter{currentBlockTimestamp, maxAllRangeTimestamp, expandFlag(int(bitsetFilter))}, 1000)
	if err != nil {
		return nil, err
	}

	cachedData := make(map[string]*BalanceState)
	for _, dataPoint := range outDataPoints {
		cachedData[dataPoint.block.String()] = &BalanceState{
			Value:     (*hexutil.Big)(dataPoint.balance),
			Timestamp: uint64(dataPoint.timestamp),
		}
	}

	prevCalibIdx := 1
	var stepDuration int64
	var currentBalanceState *BalanceState
	points := make([]*BalanceState, 0)
	for nextBlockTimestamp < currentTimestamp {
		// Check context for cancellation every cycle
		select {
		case <-ctx.Done():
			return nil, errors.New("context cancelled")
		default:
		}

		cachedDataEntry, found := cachedData[currentBlockNumber.String()]
		if found {
			points = append(points, cachedDataEntry)
		} else {
			outDataPoint, bitset, err := bh.getBalanceEntryFromDB(chainClient.ChainID(), address, chainClient.Currency(), currentBlockNumber)
			if err != nil {
				return nil, err
			}
			if outDataPoint != nil {
				err = bh.upsertBalanceEntryToDB(outDataPoint, bitset|int(bitsetFilter))
				if err != nil {
					return nil, err
				}
				currentBalanceState = &BalanceState{
					Value:     (*hexutil.Big)(outDataPoint.balance),
					Timestamp: uint64(outDataPoint.timestamp),
				}
			} else {
				currentBalanceState, err = bh.fetchAndCacheBalanceForBlock(ctx, chainClient, address, currentBlockNumber, currentBlockTimestamp, bitsetFilter)
				if err != nil {
					return nil, err
				}
			}
			points = append(points, currentBalanceState)
		}

		if calibIdx != prevCalibIdx || stepDuration == 0 {
			_, stepDuration, _ = getStepInfo(calib[calibIdx-1], calib[calibIdx], timeInterval)
			prevCalibIdx = calibIdx
		}

		prevTimestamp := roundedTimestamp
		// For steps smaller than rounding interval, we need to make sure that we don't request the same block twice
		for roundedTimestamp == prevTimestamp {
			nextBlockTimestamp += stepDuration
			roundedTimestamp = snapTimestamp(nextBlockTimestamp, timeInterval)
		}
		currentBlockNumber, currentBlockTimestamp, calibIdx = computeBlockInfoForTimestamp(roundedTimestamp, calibIdx, calib, timeInterval)
	}

	return points, nil
}

// Native token implementation of BlockInfoSource interface
type chainClientSource struct {
	chainClient *chain.Client
	currency    string
}

func (src *chainClientSource) BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return src.chainClient.BlockByNumber(ctx, number)
}

func (src *chainClientSource) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	return src.chainClient.BalanceAt(ctx, account, blockNumber)
}

func (src *chainClientSource) ChainID() uint64 {
	return src.chainClient.ChainID
}

func (src *chainClientSource) Currency() string {
	return src.currency
}

func (src *chainClientSource) TimeNow() int64 {
	return time.Now().UTC().Unix()
}

type tokenChainClientSource struct {
	chainClientSource
	TokenManager   *token.Manager
	NetworkManager *network.Manager
}

func (src *tokenChainClientSource) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	network := src.NetworkManager.Find(src.chainClient.ChainID)
	if network == nil {
		return nil, errors.New("network not found")
	}
	token := src.TokenManager.FindToken(network, src.currency)
	if token == nil {
		return nil, errors.New("token not found")
	}
	balance, err := src.TokenManager.GetTokenBalanceAt(ctx, src.chainClient, account, token.Address, blockNumber)
	if err != nil {
		if err.Error() == "no contract code at given address" {
			// Ignore requests before contract deployment
			balance = big.NewInt(0)
			err = nil
		} else {
			return nil, err
		}
	}
	return balance, err
}

type BalanceHistoryRequirements struct {
	NetworkManager *network.Manager
	TokenManager   *token.Manager
}

func (bh *BalanceHistory) StartBalanceHistory(req *BalanceHistoryRequirements, rpcClient *rpc.Client) {
	go func() {
		bh.updateBalanceHistoryRunTask(rpcClient, req.NetworkManager, req.TokenManager)
		timer := time.NewTimer(balanceHistoryUpdateInterval)
		for range timer.C {
			timer.Reset(balanceHistoryUpdateInterval)

			bh.updateBalanceHistoryRunTask(rpcClient, req.NetworkManager, req.TokenManager)
		}
	}()
}

func (bh *BalanceHistory) updateBalanceHistoryRunTask(rpcClient *rpc.Client, networkManager *network.Manager, tokenManager *token.Manager) {
	bh.asyncQueue.RunTask(func(ctx context.Context) {
		bh.walletFeed.Send(walletevent.Event{
			Type: EventBalanceHistoryUpdateStarted,
		})
		defer bh.walletFeed.Send(walletevent.Event{
			Type: EventBalanceHistoryUpdateFinished,
		})
		retryCount := 0
		var err error
		for retryCount < 3 {
			err = bh.UpdateBalanceHistoryForAllEnabledNetworks(ctx, rpcClient, networkManager, tokenManager)
			// If done or context was cancelled, we don't want to retry
			if err == nil || ctx.Err() != nil {
				return
			}
			retryCount++
			select {
			case <-ctx.Done(): //context cancelled
				return
			case <-time.After(time.Duration(retryCount*5) * time.Minute):
			}
		}
	}, updateBalanceHistoryPriority)
}

// UpdateBalanceHistoryForAllEnabledNetworks return true if the balance history was updated or false if any step failed due to temporary error (e.g. network error)
// Expects ctx to have cancellation support and processing to be cancelled by the caller
func (bh *BalanceHistory) UpdateBalanceHistoryForAllEnabledNetworks(ctx context.Context, rpcClient *rpc.Client, networkManager *network.Manager, tokenManager *token.Manager) error {
	accountsDB, err := accounts.NewDB(bh.db)
	if err != nil {
		return err
	}

	addresses, err := accountsDB.GetWalletAddresses()
	if err != nil {
		return err
	}

	networks, err := networkManager.Get(true)
	if err != nil {
		return err
	}

	networkIds := make([]uint64, 0)
	for _, network := range networks {
		networkIds = append(networkIds, network.ChainID)
	}

	tokens, err := tokenManager.GetVisible(networkIds)
	if err != nil {
		return err
	}

	for chainID, tokens := range tokens {
		for _, token := range tokens {
			var dataSource BlockInfoSource
			chainClient, err := chain.NewClient(rpcClient, chainID)
			if err != nil {
				return err
			}
			if token.IsNative() {
				dataSource = &chainClientSource{chainClient, token.Symbol}
			} else {
				dataSource = &tokenChainClientSource{
					chainClientSource: chainClientSource{
						chainClient: chainClient,
						currency:    token.Symbol,
					},
					TokenManager:   tokenManager,
					NetworkManager: networkManager,
				}
			}

			for _, address := range addresses {
				for currentInterval := int(BalanceHistory7Hours); currentInterval <= int(BalanceHistoryAllTime); currentInterval++ {
					// Check context for cancellation every fetch attempt
					select {
					case <-ctx.Done():
						return errors.New("context cancelled")
					default:
					}
					_, err = bh.getBalanceHistoryFromBlocksSource(ctx, dataSource, common.Address(address), BalanceHistoryTimeInterval(currentInterval))
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (c *Controller) GetBalanceHistoryAndInterruptUpdate(ctx context.Context, req *BalanceHistoryRequirements, identity *BhIdentity, timeInterval BalanceHistoryTimeInterval) ([]*BalanceState, error) {
	resChannel := make(chan []*BalanceState)
	errChannel := make(chan error)
	c.balanceHistory.asyncQueue.RunTask(func(ctx context.Context) {
		var dataSource BlockInfoSource
		chainClient, err := chain.NewClient(c.rpcClient, identity.ChainID)
		if err != nil {
			errChannel <- err
			return
		}

		network := req.NetworkManager.Find(identity.ChainID)
		if network == nil {
			errChannel <- errors.New("network not found")
			return
		}
		token := req.TokenManager.FindToken(network, identity.Currency)
		if token == nil {
			errChannel <- errors.New("token not found")
			return
		}
		if token.IsNative() {
			dataSource = &chainClientSource{chainClient, identity.Currency}
		} else {
			dataSource = &tokenChainClientSource{
				chainClientSource: chainClientSource{
					chainClient: chainClient,
					currency:    identity.Currency,
				},
				TokenManager:   req.TokenManager,
				NetworkManager: req.NetworkManager,
			}
		}
		res, err := c.GetBalanceHistory(ctx, dataSource, identity.Address, timeInterval)
		if err != nil {
			errChannel <- err
			return
		}
		resChannel <- res
	}, getBalanceHistoryPriority)
	select {
	case res := <-resChannel:
		return res, nil
	case err := <-errChannel:
		return nil, err
	}
}

func (c *Controller) GetBalanceHistory(ctx context.Context, dataSource BlockInfoSource, address common.Address, timeInterval BalanceHistoryTimeInterval) ([]*BalanceState, error) {
	entries, err := c.balanceHistory.getBalanceHistoryFromBlocksSource(ctx, dataSource, address, timeInterval)
	if err != nil {
		return nil, err
	}

	// Complete the last point with the latest available balance
	lastAvailable, err := c.balanceHistory.mostRecentBalanceEntry(ctx, dataSource, address)
	if err != nil {
		return nil, err
	}
	if lastAvailable != nil && lastAvailable.Timestamp > entries[len(entries)-1].Timestamp {
		entries = append(entries, lastAvailable)
	}
	return entries, nil
}

func (bh *BalanceHistory) addBalanceEntryToDB(entry *balanceHistoryDBEntry, bitset int) error {
	_, err := bh.db.Exec("INSERT INTO balance_history (chain_id, address, currency, block, timestamp, bitset, balance) VALUES (?, ?, ?, ?, ?, ?, ?)", entry.chainID, entry.address, entry.currency, (*bigint.SQLBigInt)(entry.block), entry.timestamp, bitset, (*bigint.SQLBigIntBytes)(entry.balance))
	return err
}

func (bh *BalanceHistory) upsertBalanceEntryToDB(entry *balanceHistoryDBEntry, bitset int) error {
	// Updating in place doesn't work. Tried "INSERT INTO balance_history ... ON CONFLICT(chain_id, address, currency, block) DO UPDATE SET timestamp=excluded.timestamp, bitset=(bitset | excluded.bitset), balance=excluded.balance"
	_, err := bh.db.Exec("INSERT OR REPLACE INTO balance_history (chain_id, address, currency, block, timestamp, bitset, balance) VALUES (?, ?, ?, ?, ?, ?, ?)", entry.chainID, entry.address, entry.currency, (*bigint.SQLBigInt)(entry.block), entry.timestamp, bitset, (*bigint.SQLBigIntBytes)(entry.balance))
	return err
}

// getBalanceEntryFromDB returns nil if no entry is found
func (bh *BalanceHistory) getBalanceEntryFromDB(chainID uint64, address common.Address, currency string, block *big.Int) (res *balanceHistoryDBEntry, bitset int, err error) {
	res = &balanceHistoryDBEntry{
		chainID:  chainID,
		address:  address,
		currency: currency,
		block:    new(big.Int),
		balance:  new(big.Int),
	}
	queryStr := "SELECT timestamp, balance, bitset FROM balance_history WHERE chain_id = ? AND address = ? AND currency = ? AND block = ?"
	row := bh.db.QueryRow(queryStr, chainID, address, currency, (*bigint.SQLBigInt)(block))
	err = row.Scan(&res.timestamp, (*bigint.SQLBigIntBytes)(res.balance), &bitset)
	if err == sql.ErrNoRows {
		return nil, 0, nil
	} else if err != nil {
		return nil, 0, err
	}
	res.block.Set(block)
	return
}

type BhIdentity struct {
	ChainID  uint64
	Address  common.Address
	Currency string
}

type bhFilter struct {
	minTimestamp int64
	maxTimestamp int64
	bitsetFilter int
}

// getBalanceEntriesFromDBTimeSorted returns a sorted list of entries or empty array if none is found for the given input
// If startingAtBlock is provided, the result will include the provided block number if available or the next available one
// If startingAtBlock is NOT provided the result will begin from the first available block
// minTimestamp and maxTimestamp interval filter the results by timestamp.
// bitsetFilter filters the results by bitset. This way higher values can include lower values to simulate time interval levels
// asc defines the order of the result by block number (which correlates also with time). If true, the result will be sorted by ascending, otherwise by descending timestamp
func (bh *BalanceHistory) getDBBalanceEntriesTimeSorted(identify *BhIdentity, startingAtBlock *big.Int, filter *bhFilter, maxEntries int, asc bool) ([]*balanceHistoryDBEntry, error) {
	// Start from the first block in case a specific one was not provided
	if startingAtBlock == nil {
		startingAtBlock = big.NewInt(0)
	}
	// We are interested in order by timestamp, but we request by block number that correlates to the order of timestamp and it is indexed
	var queryStr string
	if asc {
		queryStr = "SELECT block, timestamp, balance FROM balance_history WHERE chain_id = ? AND address = ? AND currency = ? AND block >= ? AND timestamp BETWEEN ? AND ? AND (bitset & ?) > 0 ORDER BY block ASC LIMIT ?"
	} else {
		queryStr = "SELECT block, timestamp, balance FROM balance_history WHERE chain_id = ? AND address = ? AND currency = ? AND block >= ? AND timestamp BETWEEN ? AND ? AND (bitset & ?) > 0 ORDER BY block DESC LIMIT ?"
	}
	rows, err := bh.db.Query(queryStr, identify.ChainID, identify.Address, identify.Currency, (*bigint.SQLBigInt)(startingAtBlock), filter.minTimestamp, filter.maxTimestamp, filter.bitsetFilter, maxEntries)
	if err != nil {
		return make([]*balanceHistoryDBEntry, 0), err
	}
	defer rows.Close()

	currentEntry := 0
	result := make([]*balanceHistoryDBEntry, 0)
	for rows.Next() && currentEntry < maxEntries {
		entry := &balanceHistoryDBEntry{
			chainID:  0,
			address:  identify.Address,
			currency: identify.Currency,
			block:    new(big.Int),
			balance:  new(big.Int),
		}
		err := rows.Scan((*bigint.SQLBigInt)(entry.block), &entry.timestamp, (*bigint.SQLBigIntBytes)(entry.balance))
		if err != nil {
			return make([]*balanceHistoryDBEntry, 0), err
		}
		entry.chainID = identify.ChainID
		result = append(result, entry)
		currentEntry++
	}
	return result, nil
}

func (bh *BalanceHistory) getDBBalanceEntriesByTimeIntervalAndSortedAsc(identify *BhIdentity, startingAtBlock *big.Int, filter *bhFilter, maxEntries int) ([]*balanceHistoryDBEntry, error) {
	return bh.getDBBalanceEntriesTimeSorted(identify, startingAtBlock, filter, maxEntries, true)
}

func (bh *BalanceHistory) getDBBalanceEntriesByTimeIntervalAndSortedDesc(identify *BhIdentity, startingAtBlock *big.Int, filter *bhFilter, maxEntries int) ([]*balanceHistoryDBEntry, error) {
	return bh.getDBBalanceEntriesTimeSorted(identify, startingAtBlock, filter, maxEntries, false)
}

func (bh *BalanceHistory) getDBBalanceEntriesTimeSortedAsc(identify *BhIdentity, startingAtBlock *big.Int, bitsetFilter int, maxEntries int) ([]*balanceHistoryDBEntry, error) {
	return bh.getDBBalanceEntriesTimeSorted(identify, startingAtBlock, &bhFilter{minAllRangeTimestamp, maxAllRangeTimestamp, bitsetFilter}, maxEntries, true)
}

func (bh *BalanceHistory) getDBBalanceEntriesTimeSortedDesc(identify *BhIdentity, startingAtBlock *big.Int, bitsetFilter int, maxEntries int) ([]*balanceHistoryDBEntry, error) {
	return bh.getDBBalanceEntriesTimeSorted(identify, startingAtBlock, &bhFilter{minAllRangeTimestamp, maxAllRangeTimestamp, bitsetFilter}, maxEntries, false)
}
