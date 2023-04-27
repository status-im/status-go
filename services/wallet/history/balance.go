package history

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

type Balance struct {
	db *BalanceDB
}

const (
	defaultChains = uint64(0)
	aDay          = time.Duration(24) * time.Hour
)

var averageBlockDurationForChain = map[uint64]time.Duration{
	defaultChains: time.Duration(12000) * time.Millisecond,
	10:            time.Duration(400) * time.Millisecond,  // Optimism
	420:           time.Duration(2000) * time.Millisecond, // Optimism Testnet
	42161:         time.Duration(300) * time.Millisecond,  // Arbitrum
	421611:        time.Duration(1500) * time.Millisecond, // Arbitrum Testnet
}

// Must have a common divisor to share common blocks and increase the cache hit
const (
	twiceADayStride time.Duration = time.Duration(12) * time.Hour
	weekStride                    = 14 * twiceADayStride
	monthsStride                  = 1 /*months*/ * 4 * weekStride
)

// bitsetFilters used to fetch relevant data points in one batch and to increase cache hit
const (
	filterAllTime   bitsetFilter = 1
	filterWeekly    bitsetFilter = 1 << 3
	filterTwiceADay bitsetFilter = 1 << 5
)

type TimeInterval int

// Specific time intervals for which balance history can be fetched
const (
	BalanceHistory7Days TimeInterval = iota + 1
	BalanceHistory1Month
	BalanceHistory6Months
	BalanceHistory1Year
	BalanceHistoryAllTime
)

var timeIntervalDuration = map[TimeInterval]time.Duration{
	BalanceHistory7Days:   time.Duration(7) * aDay,
	BalanceHistory1Month:  time.Duration(30) * aDay,
	BalanceHistory6Months: time.Duration(6*30) * aDay,
	BalanceHistory1Year:   time.Duration(365) * aDay,
}

var timeIntervalToBitsetFilter = map[TimeInterval]bitsetFilter{
	BalanceHistory7Days:   filterTwiceADay,
	BalanceHistory1Month:  filterTwiceADay,
	BalanceHistory6Months: filterWeekly,
	BalanceHistory1Year:   filterWeekly,
	BalanceHistoryAllTime: filterAllTime,
}

var timeIntervalToStrideDuration = map[TimeInterval]time.Duration{
	BalanceHistory7Days:   twiceADayStride,
	BalanceHistory1Month:  twiceADayStride,
	BalanceHistory6Months: weekStride,
	BalanceHistory1Year:   weekStride,
	BalanceHistoryAllTime: monthsStride,
}

func strideBlockCount(timeInterval TimeInterval, chainID uint64) int {
	blockDuration, found := averageBlockDurationForChain[chainID]
	if !found {
		blockDuration = averageBlockDurationForChain[defaultChains]
	}

	return int(timeIntervalToStrideDuration[timeInterval] / blockDuration)
}

func NewBalance(db *BalanceDB) *Balance {
	return &Balance{
		db: db,
	}
}

// DataSource used as an abstraction to fetch required data from a specific blockchain
type DataSource interface {
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
	ChainID() uint64
	Currency() string
	TimeNow() int64
}

type DataPoint struct {
	Balance     *hexutil.Big
	Timestamp   uint64
	BlockNumber *hexutil.Big
}

// fetchAndCache will process the last available block if blocNo is nil
// reuses previous fetched blocks timestamp to avoid fetching block headers again
func (b *Balance) fetchAndCache(ctx context.Context, source DataSource, address common.Address, blockNo *big.Int, bitset bitsetFilter) (*DataPoint, *big.Int, error) {
	var outEntry *entry
	var err error
	if blockNo != nil {
		cached, bitsetList, err := b.db.get(&assetIdentity{source.ChainID(), address, source.Currency()}, blockNo, 1, asc)
		if err != nil {
			return nil, nil, err
		}
		if len(cached) > 0 && cached[0].block.Cmp(blockNo) == 0 {
			// found a match update bitset
			err := b.db.updateBitset(&assetIdentity{source.ChainID(), address, source.Currency()}, blockNo, bitset|bitsetList[0])
			if err != nil {
				return nil, nil, err
			}
			return &DataPoint{
				Balance:     (*hexutil.Big)(cached[0].balance),
				Timestamp:   uint64(cached[0].timestamp),
				BlockNumber: (*hexutil.Big)(cached[0].block),
			}, blockNo, nil
		}

		// otherwise try fetch any to get the timestamp info
		outEntry, _, err = b.db.getFirst(source.ChainID(), blockNo)
		if err != nil {
			return nil, nil, err
		}
	}
	var timestamp int64
	if outEntry != nil {
		timestamp = outEntry.timestamp
	} else {
		header, err := source.HeaderByNumber(ctx, blockNo)
		if err != nil {
			return nil, nil, err
		}
		blockNo = new(big.Int).Set(header.Number)
		timestamp = int64(header.Time)
	}

	currentBalance, err := source.BalanceAt(ctx, address, blockNo)
	if err != nil {
		return nil, nil, err
	}

	entry := entry{
		chainID:     source.ChainID(),
		address:     address,
		tokenSymbol: source.Currency(),
		block:       new(big.Int).Set(blockNo),
		balance:     currentBalance,
		timestamp:   timestamp,
	}
	err = b.db.add(&entry, bitset)
	if err != nil {
		return nil, nil, err
	}

	var dataPoint DataPoint
	dataPoint.Balance = (*hexutil.Big)(currentBalance)
	dataPoint.Timestamp = uint64(timestamp)
	return &dataPoint, blockNo, nil
}

// update retrieves the balance history for a specified asset from the database initially
// and supplements any missing information from the blockchain to minimize the number of RPC calls.
// if context is cancelled it will return with error
func (b *Balance) update(ctx context.Context, source DataSource, address common.Address, timeInterval TimeInterval) error {
	startTimestamp := int64(0)
	fetchTimestamp := int64(0)
	endTime := source.TimeNow()
	if timeInterval != BalanceHistoryAllTime {
		// Ensure we always get the complete range by fetching the next block also
		startTimestamp = endTime - int64(timeIntervalDuration[timeInterval].Seconds())
		fetchTimestamp = startTimestamp - int64(timeIntervalToStrideDuration[timeInterval].Seconds())
	}
	identity := &assetIdentity{source.ChainID(), address, source.Currency()}
	firstCached, err := b.firstCachedStartingAt(identity, fetchTimestamp, timeInterval)
	if err != nil {
		return err
	}

	var oldestCached *big.Int
	var oldestTimestamp int64
	var newestCached *big.Int
	if firstCached != nil {
		oldestCached = new(big.Int).Set(firstCached.block)
		oldestTimestamp = firstCached.timestamp
		lastCached, err := b.lastCached(identity, timeInterval)
		if err != nil {
			return err
		}
		newestCached = new(big.Int).Set(lastCached.block)
	} else {
		var fetchBlock *big.Int
		lastEntry, _, err := b.db.getLastEntryForChain(source.ChainID())
		if err != nil {
			return err
		}
		if lastEntry != nil {
			fetchBlock = new(big.Int).Set(lastEntry.block)
		}
		mostRecentDataPoint, mostRecentBlock, err := b.fetchAndCache(ctx, source, address, fetchBlock, timeIntervalToBitsetFilter[timeInterval])
		if err != nil {
			return err
		}

		oldestCached = new(big.Int).Set(mostRecentBlock)
		oldestTimestamp = int64(mostRecentDataPoint.Timestamp)
		newestCached = new(big.Int).Set(mostRecentBlock)
	}

	if oldestTimestamp > startTimestamp {
		err := b.fetchBackwardAndCache(ctx, source, address, oldestCached, startTimestamp, timeInterval)
		if err != nil {
			return err
		}
	}

	// Fetch forward if didn't update in a stride duration
	err = b.fetchForwardAndCache(ctx, source, address, newestCached, timeInterval)
	if err != nil {
		return err
	}

	return nil
}

// get returns the balance history for the given address and time interval until endTimestamp
func (b *Balance) get(ctx context.Context, chainID uint64, currency string, address common.Address, endTimestamp int64, timeInterval TimeInterval) ([]*DataPoint, error) {
	startTimestamp := int64(0)
	fetchTimestamp := int64(0)
	if timeInterval != BalanceHistoryAllTime {
		// Ensure we always get the complete range by fetching the next block also
		startTimestamp = endTimestamp - int64(timeIntervalDuration[timeInterval].Seconds())
		fetchTimestamp = startTimestamp - int64(timeIntervalToStrideDuration[timeInterval].Seconds())
	}
	cached, _, err := b.db.filter(&assetIdentity{chainID, address, currency}, nil, &balanceFilter{fetchTimestamp, endTimestamp, expandFlag(timeIntervalToBitsetFilter[timeInterval])}, 800, asc)
	if err != nil {
		return nil, err
	}

	points := make([]*DataPoint, 0, len(cached)+1)
	for _, entry := range cached {
		dataPoint := DataPoint{
			Balance:     (*hexutil.Big)(entry.balance),
			Timestamp:   uint64(entry.timestamp),
			BlockNumber: (*hexutil.Big)(entry.block),
		}
		points = append(points, &dataPoint)
	}

	lastCached, _, err := b.db.get(&assetIdentity{chainID, address, currency}, nil, 1, desc)
	if err != nil {
		return nil, err
	}
	if len(lastCached) > 0 && len(cached) > 0 && lastCached[0].block.Cmp(cached[len(cached)-1].block) > 0 {
		points = append(points, &DataPoint{
			Balance:     (*hexutil.Big)(lastCached[0].balance),
			Timestamp:   uint64(lastCached[0].timestamp),
			BlockNumber: (*hexutil.Big)(lastCached[0].block),
		})
	}

	return points, nil
}

// fetchBackwardAndCache fetches and adds to DB balance entries starting one stride before the endBlock and stops
// when reaching a block timestamp older than startTimestamp or genesis block
// relies on the approximation of a block length to match averageBlockDurationForChain for sampling the data
func (b *Balance) fetchBackwardAndCache(ctx context.Context, source DataSource, address common.Address, endBlock *big.Int, startTimestamp int64, timeInterval TimeInterval) error {
	stride := strideBlockCount(timeInterval, source.ChainID())
	nextBlock := new(big.Int).Set(endBlock)
	for nextBlock.Cmp(big.NewInt(1)) > 0 {
		if shouldCancel(ctx) {
			return errors.New("context cancelled")
		}

		nextBlock.Sub(nextBlock, big.NewInt(int64(stride)))
		if nextBlock.Cmp(big.NewInt(0)) <= 0 {
			// we reached the genesis block which doesn't have a usable timestamp, fetch next
			nextBlock.SetUint64(1)
		}

		dataPoint, _, err := b.fetchAndCache(ctx, source, address, nextBlock, timeIntervalToBitsetFilter[timeInterval])
		if err != nil {
			return err
		}

		// Allow to go back one stride to match the requested interval
		if int64(dataPoint.Timestamp) < startTimestamp {
			return nil
		}
	}

	return nil
}

// fetchForwardAndCache fetches and adds to DB balance entries starting one stride before the startBlock and stops
// when block not found
// relies on the approximation of a block length to match averageBlockDurationForChain
func (b *Balance) fetchForwardAndCache(ctx context.Context, source DataSource, address common.Address, startBlock *big.Int, timeInterval TimeInterval) error {
	stride := strideBlockCount(timeInterval, source.ChainID())
	nextBlock := new(big.Int).Set(startBlock)
	for {
		if shouldCancel(ctx) {
			return errors.New("context cancelled")
		}

		nextBlock.Add(nextBlock, big.NewInt(int64(stride)))
		_, _, err := b.fetchAndCache(ctx, source, address, nextBlock, timeIntervalToBitsetFilter[timeInterval])
		if err != nil {
			if err == ethereum.NotFound {
				// We overshoot, stop and return what we have
				return nil
			}
			return err
		}
	}
}

// firstCachedStartingAt returns first cached entry for the given identity and time interval starting at fetchTimestamp or nil if none found
func (b *Balance) firstCachedStartingAt(identity *assetIdentity, startTimestamp int64, timeInterval TimeInterval) (first *entry, err error) {
	entries, _, err := b.db.filter(identity, nil, &balanceFilter{startTimestamp, maxAllRangeTimestamp, expandFlag(timeIntervalToBitsetFilter[timeInterval])}, 1, desc)
	if err != nil {
		return nil, err
	} else if len(entries) == 0 {
		return nil, nil
	}
	return entries[0], nil
}

// lastCached returns last cached entry for the given identity and time interval or nil if none found
func (b *Balance) lastCached(identity *assetIdentity, timeInterval TimeInterval) (first *entry, err error) {
	entries, _, err := b.db.filter(identity, nil, &balanceFilter{minAllRangeTimestamp, maxAllRangeTimestamp, expandFlag(timeIntervalToBitsetFilter[timeInterval])}, 1, desc)
	if err != nil {
		return nil, err
	} else if len(entries) == 0 {
		return nil, nil
	}
	return entries[0], nil
}

// shouldCancel returns true if the context has been cancelled and task should be aborted
func shouldCancel(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
	}
	return false
}
