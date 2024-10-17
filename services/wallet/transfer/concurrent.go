package transfer

import (
	"context"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/balance"
)

const (
	NoThreadLimit         uint32 = 0
	SequentialThreadLimit uint32 = 10
)

// NewConcurrentDownloader creates ConcurrentDownloader instance.
func NewConcurrentDownloader(ctx context.Context, limit uint32) *ConcurrentDownloader {
	runner := async.NewQueuedAtomicGroup(ctx, limit)
	result := &Result{}
	return &ConcurrentDownloader{runner, result}
}

type ConcurrentDownloader struct {
	*async.QueuedAtomicGroup
	*Result
}

type Result struct {
	mu          sync.Mutex
	transfers   []Transfer
	headers     []*DBHeader
	blockRanges [][]*big.Int
}

var errDownloaderStuck = errors.New("eth downloader is stuck")

func (r *Result) Push(transfers ...Transfer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.transfers = append(r.transfers, transfers...)
}

func (r *Result) Get() []Transfer {
	r.mu.Lock()
	defer r.mu.Unlock()
	rst := make([]Transfer, len(r.transfers))
	copy(rst, r.transfers)
	return rst
}

func (r *Result) PushHeader(block *DBHeader) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.headers = append(r.headers, block)
}

func (r *Result) GetHeaders() []*DBHeader {
	r.mu.Lock()
	defer r.mu.Unlock()
	rst := make([]*DBHeader, len(r.headers))
	copy(rst, r.headers)
	return rst
}

func (r *Result) PushRange(blockRange []*big.Int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.blockRanges = append(r.blockRanges, blockRange)
}

func (r *Result) GetRanges() [][]*big.Int {
	r.mu.Lock()
	defer r.mu.Unlock()
	rst := make([][]*big.Int, len(r.blockRanges))
	copy(rst, r.blockRanges)
	r.blockRanges = [][]*big.Int{}

	return rst
}

// Downloader downloads transfers from single block using number.
type Downloader interface {
	GetTransfersByNumber(context.Context, *big.Int) ([]Transfer, error)
}

// Returns new block ranges that contain transfers and found block headers that contain transfers, and a block where
// beginning of trasfers history detected
func checkRangesWithStartBlock(parent context.Context, client balance.Reader, cache balance.Cacher,
	account common.Address, ranges [][]*big.Int, threadLimit uint32, startBlock *big.Int) (
	resRanges [][]*big.Int, headers []*DBHeader, newStartBlock *big.Int, err error) {

	logutils.ZapLogger().Debug("start checkRanges",
		zap.Stringer("account", account),
		zap.Int("ranges len", len(ranges)),
		zap.Stringer("startBlock", startBlock),
	)

	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()

	c := NewConcurrentDownloader(ctx, threadLimit)

	newStartBlock = startBlock

	for _, blocksRange := range ranges {
		from := blocksRange[0]
		to := blocksRange[1]

		logutils.ZapLogger().Debug("check block range",
			zap.Stringer("from", from),
			zap.Stringer("to", to),
		)

		if startBlock != nil {
			if to.Cmp(newStartBlock) <= 0 {
				logutils.ZapLogger().Debug("'to' block is less than 'start' block",
					zap.Stringer("to", to),
					zap.Stringer("startBlock", startBlock),
				)
				continue
			}
		}

		c.Add(func(ctx context.Context) error {
			if from.Cmp(to) >= 0 {
				logutils.ZapLogger().Debug("'from' block is greater than or equal to 'to' block",
					zap.Stringer("from", from),
					zap.Stringer("to", to),
				)
				return nil
			}
			logutils.ZapLogger().Debug("eth transfers comparing blocks",
				zap.Stringer("from", from),
				zap.Stringer("to", to),
			)

			if startBlock != nil {
				if to.Cmp(startBlock) <= 0 {
					logutils.ZapLogger().Debug("'to' block is less than 'start' block",
						zap.Stringer("to", to),
						zap.Stringer("startBlock", startBlock),
					)
					return nil
				}
			}

			lb, err := cache.BalanceAt(ctx, client, account, from)
			if err != nil {
				return err
			}
			hb, err := cache.BalanceAt(ctx, client, account, to)
			if err != nil {
				return err
			}
			if lb.Cmp(hb) == 0 {
				logutils.ZapLogger().Debug("balances are equal",
					zap.Stringer("from", from),
					zap.Stringer("to", to),
					zap.Stringer("lb", lb),
					zap.Stringer("hb", hb),
				)

				hn, err := cache.NonceAt(ctx, client, account, to)
				if err != nil {
					return err
				}
				// if nonce is zero in a newer block then there is no need to check an older one
				if *hn == 0 {
					logutils.ZapLogger().Debug("zero nonce", zap.Stringer("to", to))

					if hb.Cmp(big.NewInt(0)) == 0 { // balance is 0, nonce is 0, we stop checking further, that will be the start block (even though the real one can be a later one)
						if startBlock != nil {
							if to.Cmp(newStartBlock) > 0 {
								logutils.ZapLogger().Debug("found possible start block, we should not search back", zap.Stringer("block", to))
								newStartBlock = to // increase newStartBlock if we found a new higher block
							}
						} else {
							newStartBlock = to
						}
					}

					return nil
				}

				ln, err := cache.NonceAt(ctx, client, account, from)
				if err != nil {
					return err
				}
				if *ln == *hn {
					logutils.ZapLogger().Debug("transaction count is also equal",
						zap.Stringer("from", from),
						zap.Stringer("to", to),
						zap.Int64p("ln", ln),
						zap.Int64p("hn", hn),
					)
					return nil
				}
			}
			if new(big.Int).Sub(to, from).Cmp(one) == 0 {
				header, err := client.HeaderByNumber(ctx, to)
				if err != nil {
					return err
				}
				c.PushHeader(toDBHeader(header, account))
				return nil
			}
			mid := new(big.Int).Add(from, to)
			mid = mid.Div(mid, two)
			_, err = cache.BalanceAt(ctx, client, account, mid)
			if err != nil {
				return err
			}
			logutils.ZapLogger().Debug("balances are not equal",
				zap.Stringer("from", from),
				zap.Stringer("mid", mid),
				zap.Stringer("to", to),
			)

			c.PushRange([]*big.Int{mid, to})
			c.PushRange([]*big.Int{from, mid})
			return nil
		})
	}

	select {
	case <-c.WaitAsync():
	case <-ctx.Done():
		return nil, nil, nil, errDownloaderStuck
	}

	if c.Error() != nil {
		return nil, nil, nil, errors.Wrap(c.Error(), "failed to dowload transfers using concurrent downloader")
	}

	logutils.ZapLogger().Debug("end checkRanges",
		zap.Stringer("account", account),
		zap.Stringer("newStartBlock", newStartBlock),
	)
	return c.GetRanges(), c.GetHeaders(), newStartBlock, nil
}

func findBlocksWithEthTransfers(parent context.Context, client balance.Reader, cache balance.Cacher,
	account common.Address, low, high *big.Int, noLimit bool, threadLimit uint32) (
	from *big.Int, headers []*DBHeader, resStartBlock *big.Int, err error) {

	ranges := [][]*big.Int{{low, high}}
	from = big.NewInt(low.Int64())
	headers = []*DBHeader{}
	var lvl = 1

	for len(ranges) > 0 && lvl <= 30 {
		logutils.ZapLogger().Debug("check blocks ranges",
			zap.Int("lvl", lvl),
			zap.Int("ranges len", len(ranges)),
		)
		lvl++
		// Check if there are transfers in blocks in ranges. To do that, nonce and balance is checked
		// the block ranges that have transfers are returned
		newRanges, newHeaders, strtBlock, err := checkRangesWithStartBlock(parent, client, cache,
			account, ranges, threadLimit, resStartBlock)
		resStartBlock = strtBlock
		if err != nil {
			return nil, nil, nil, err
		}

		headers = append(headers, newHeaders...)

		if len(newRanges) > 0 {
			logutils.ZapLogger().Debug("found new ranges",
				zap.Stringer("account", account),
				zap.Int("lvl", lvl),
				zap.Int("new ranges len", len(newRanges)),
			)
		}
		if len(newRanges) > 60 && !noLimit {
			sort.SliceStable(newRanges, func(i, j int) bool {
				return newRanges[i][0].Cmp(newRanges[j][0]) == 1
			})

			newRanges = newRanges[:60]
			from = newRanges[len(newRanges)-1][0]
		}

		ranges = newRanges
	}

	return
}
