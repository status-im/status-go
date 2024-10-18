package history

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/logutils"
)

const genesisTimestamp = 1438269988

// Specific time intervals for which balance history can be fetched
type TimeInterval int

const (
	BalanceHistory7Days TimeInterval = iota + 1
	BalanceHistory1Month
	BalanceHistory6Months
	BalanceHistory1Year
	BalanceHistoryAllTime
)

const aDay = time.Duration(24) * time.Hour

var timeIntervalDuration = map[TimeInterval]time.Duration{
	BalanceHistory7Days:   time.Duration(7) * aDay,
	BalanceHistory1Month:  time.Duration(30) * aDay,
	BalanceHistory6Months: time.Duration(6*30) * aDay,
	BalanceHistory1Year:   time.Duration(365) * aDay,
}

func TimeIntervalDurationSecs(timeInterval TimeInterval) uint64 {
	return uint64(timeIntervalDuration[timeInterval].Seconds())
}

type DataPoint struct {
	Balance     *hexutil.Big
	Timestamp   uint64
	BlockNumber *hexutil.Big
}

// String returns a string representation of the data point
func (d *DataPoint) String() string {
	return fmt.Sprintf("timestamp: %d balance: %v block: %v", d.Timestamp, d.Balance.ToInt(), d.BlockNumber.ToInt())
}

type Balance struct {
	db *BalanceDB
}

func NewBalance(db *BalanceDB) *Balance {
	return &Balance{db}
}

// get returns the balance history for the given address from the given timestamp till now
func (b *Balance) get(ctx context.Context, chainID uint64, currency string, addresses []common.Address, fromTimestamp uint64) ([]*entry, error) {
	logutils.ZapLogger().Debug("Getting balance history",
		zap.Uint64("chainID", chainID),
		zap.String("currency", currency),
		zap.Stringers("address", addresses),
		zap.Uint64("fromTimestamp", fromTimestamp),
	)

	cached, err := b.db.getNewerThan(&assetIdentity{chainID, addresses, currency}, fromTimestamp)
	if err != nil {
		return nil, err
	}

	return cached, nil
}

func (b *Balance) addEdgePoints(chainID uint64, currency string, addresses []common.Address, fromTimestamp, toTimestamp uint64, data []*entry) (res []*entry, err error) {
	logutils.ZapLogger().Debug("Adding edge points",
		zap.Uint64("chainID", chainID),
		zap.String("currency", currency),
		zap.Stringers("address", addresses),
		zap.Uint64("fromTimestamp", fromTimestamp),
	)

	if len(addresses) == 0 {
		return nil, errors.New("addresses must not be empty")
	}

	res = data

	var firstEntry *entry

	if len(data) > 0 {
		firstEntry = data[0]
	} else {
		firstEntry = &entry{
			chainID:     chainID,
			address:     addresses[0],
			tokenSymbol: currency,
			timestamp:   int64(fromTimestamp),
		}
	}

	previous, err := b.db.getEntryPreviousTo(firstEntry)
	if err != nil {
		return nil, err
	}

	firstTimestamp, lastTimestamp := timestampBoundaries(fromTimestamp, toTimestamp, data)

	if previous != nil {
		previous.timestamp = int64(firstTimestamp) // We might need to use another minimal offset respecting the time interval
		previous.block = nil
		res = append([]*entry{previous}, res...)
	} else {
		// Add a zero point at the beginning to draw a line from
		res = append([]*entry{
			{
				chainID:     chainID,
				address:     addresses[0],
				tokenSymbol: currency,
				timestamp:   int64(firstTimestamp),
				balance:     big.NewInt(0),
			},
		}, res...)
	}

	lastPoint := res[len(res)-1]
	if lastPoint.timestamp < int64(lastTimestamp) {
		// Add a last point to draw a line to
		res = append(res, &entry{
			chainID:     chainID,
			address:     lastPoint.address,
			tokenSymbol: currency,
			timestamp:   int64(lastTimestamp),
			balance:     lastPoint.balance,
		})
	}

	return res, nil
}

func timestampBoundaries(fromTimestamp, toTimestamp uint64, data []*entry) (firstTimestamp, lastTimestamp uint64) {
	firstTimestamp = fromTimestamp
	if fromTimestamp == 0 {
		if len(data) > 0 {
			if data[0].timestamp == 0 {
				panic("data[0].timestamp must never be 0")
			}
			firstTimestamp = uint64(data[0].timestamp) - 1
		} else {
			firstTimestamp = genesisTimestamp
		}
	}

	if toTimestamp < firstTimestamp {
		panic("toTimestamp < fromTimestamp")
	}

	lastTimestamp = toTimestamp

	return firstTimestamp, lastTimestamp
}

func addPaddingPoints(currency string, addresses []common.Address, toTimestamp uint64, data []*entry, limit int) (res []*entry, err error) {
	logutils.ZapLogger().Debug("addPaddingPoints start",
		zap.String("currency", currency),
		zap.Stringers("address", addresses),
		zap.Int("len(data)", len(data)),
		zap.Any("data", data),
		zap.Int("limit", limit),
	)

	if len(data) < 2 { // Edge points must be added separately during the previous step
		return nil, errors.New("slice is empty")
	}

	if limit <= len(data) {
		return data, nil
	}

	fromTimestamp := uint64(data[0].timestamp)
	delta := (toTimestamp - fromTimestamp) / uint64(limit-1)

	res = make([]*entry, len(data))
	copy(res, data)

	var address common.Address
	if len(addresses) > 0 {
		address = addresses[0]
	}

	for i, j, index := 1, 0, 0; len(res) < limit; index++ {
		// Add a last point to draw a line to. For some cases we might not need it,
		// but when merging with points from other chains, we might get wrong balance if we don't have it.
		paddingTimestamp := int64(fromTimestamp + delta*uint64(i))

		if paddingTimestamp < data[j].timestamp {
			// make a room for a new point
			res = append(res[:index+1], res[index:]...)
			// insert a new point
			entry := &entry{
				address:     address,
				tokenSymbol: currency,
				timestamp:   paddingTimestamp,
				balance:     data[j-1].balance, // take the previous balance
				chainID:     data[j-1].chainID,
			}
			res[index] = entry

			logutils.ZapLogger().Debug("Added padding point",
				zap.Stringer("entry", entry),
				zap.Int64("timestamp", paddingTimestamp),
				zap.Int("i", i),
				zap.Int("j", j),
				zap.Int("index", index),
			)
			i++
		} else if paddingTimestamp >= data[j].timestamp {
			logutils.ZapLogger().Debug("Kept real point",
				zap.Any("entry", data[j]),
				zap.Int64("timestamp", paddingTimestamp),
				zap.Int("i", i),
				zap.Int("j", j),
				zap.Int("index", index),
			)
			j++
		}
	}

	logutils.ZapLogger().Debug("addPaddingPoints end", zap.Int("len(res)", len(res)))

	return res, nil
}
