package wallet

import (
	"context"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// NewConcurrentDownloader creates ConcurrentDownloader instance.
func NewConcurrentDownloader(ctx context.Context) *ConcurrentDownloader {
	runner := NewAtomicGroup(ctx)
	result := &Result{}
	return &ConcurrentDownloader{runner, result}
}

type ConcurrentDownloader struct {
	*AtomicGroup
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

// TransferDownloader downloads transfers from single block using number.
type TransferDownloader interface {
	GetTransfersByNumber(context.Context, *big.Int) ([]Transfer, error)
}

func checkRanges(parent context.Context, client reactorClient, cache BalanceCache, downloader TransferDownloader, account common.Address, ranges [][]*big.Int) ([][]*big.Int, []*DBHeader, error) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()

	c := NewConcurrentDownloader(ctx)

	for _, blocksRange := range ranges {
		from := blocksRange[0]
		to := blocksRange[1]

		c.Add(func(ctx context.Context) error {
			if from.Cmp(to) >= 0 {
				return nil
			}
			log.Debug("eth transfers comparing blocks", "from", from, "to", to)
			lb, err := cache.BalanceAt(ctx, client, account, from)
			if err != nil {
				return err
			}
			hb, err := cache.BalanceAt(ctx, client, account, to)
			if err != nil {
				return err
			}
			if lb.Cmp(hb) == 0 {
				log.Debug("balances are equal", "from", from, "to", to)
				// In case if balances are equal but non zero we want to check if
				// eth_getTransactionCount return different values, because there
				// still might be transactions
				if lb.Cmp(zero) != 0 {
					return nil
				}

				ln, err := client.NonceAt(ctx, account, from)
				if err != nil {
					return err
				}
				hn, err := client.NonceAt(ctx, account, to)
				if err != nil {
					return err
				}
				if ln == hn {
					log.Debug("transaction count is also equal", "from", from, "to", to)
					return nil
				}
			}
			if new(big.Int).Sub(to, from).Cmp(one) == 0 {
				header, err := client.HeaderByNumber(ctx, to)
				if err != nil {
					return err
				}
				c.PushHeader(toDBHeader(header))
				return nil
			}
			mid := new(big.Int).Add(from, to)
			mid = mid.Div(mid, two)
			_, err = cache.BalanceAt(ctx, client, account, mid)
			if err != nil {
				return err
			}
			log.Debug("balances are not equal", "from", from, "mid", mid, "to", to)

			c.PushRange([]*big.Int{from, mid})
			c.PushRange([]*big.Int{mid, to})
			return nil
		})

	}

	select {
	case <-c.WaitAsync():
	case <-ctx.Done():
		return nil, nil, errDownloaderStuck
	}

	if c.Error() != nil {
		return nil, nil, errors.Wrap(c.Error(), "failed to dowload transfers using concurrent downloader")
	}

	return c.GetRanges(), c.GetHeaders(), nil
}

func findBlocksWithEthTransfers(parent context.Context, client reactorClient, cache BalanceCache, downloader TransferDownloader, account common.Address, low, high *big.Int, noLimit bool) (from *big.Int, headers []*DBHeader, err error) {
	ranges := [][]*big.Int{{low, high}}
	minBlock := big.NewInt(low.Int64())
	headers = []*DBHeader{}
	var lvl = 1
	for len(ranges) > 0 && lvl <= 30 {
		log.Debug("check blocks ranges", "lvl", lvl, "ranges len", len(ranges))
		lvl++
		newRanges, newHeaders, err := checkRanges(parent, client, cache, downloader, account, ranges)
		if err != nil {
			return nil, nil, err
		}

		headers = append(headers, newHeaders...)

		if len(newRanges) > 0 {
			log.Debug("found new ranges", "account", account, "lvl", lvl, "new ranges len", len(newRanges))
		}
		if len(newRanges) > 60 && !noLimit {
			sort.SliceStable(newRanges, func(i, j int) bool {
				return newRanges[i][0].Cmp(newRanges[j][0]) == 1
			})

			newRanges = newRanges[:60]
			minBlock = newRanges[len(newRanges)-1][0]
		}

		ranges = newRanges
	}

	return minBlock, headers, err
}
