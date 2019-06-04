package wallet

import (
	"context"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// NewConcurrentDownloader creates ConcurrentDownloader instance.
func NewConcurrentDownloader(ctx context.Context) *ConcurrentDownloader {
	ctx, cancel := context.WithCancel(ctx)
	return &ConcurrentDownloader{
		ctx:    ctx,
		cancel: cancel,
	}
}

// ConcurrentDownloader manages downloaders life cycle.
type ConcurrentDownloader struct {
	ctx    context.Context
	cancel func()
	wg     sync.WaitGroup

	mu      sync.Mutex
	results []Transfer
	error   error
}

// Go spawns function in a goroutine and stores results or errors.
func (d *ConcurrentDownloader) Go(f func(context.Context) ([]Transfer, error)) {
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		transfers, err := f(d.ctx)
		d.mu.Lock()
		defer d.mu.Unlock()
		if err != nil {
			// do not overwrite original error by context errors
			if d.error != nil {
				return
			}
			d.error = err
			d.cancel()
			return
		}
		d.results = append(d.results, transfers...)
	}()
}

// Transfers returns collected transfers. To get all results should be called after Wait.
func (d *ConcurrentDownloader) Transfers() []Transfer {
	d.mu.Lock()
	defer d.mu.Unlock()
	rst := make([]Transfer, len(d.results))
	copy(rst, d.results)
	return rst
}

// Wait for all downloaders to finish.
func (d *ConcurrentDownloader) Wait() {
	d.wg.Wait()
	if d.Error() == nil {
		d.cancel()
	}
}

// Error stores an error that was reported by any of the downloader. Should be called after Wait.
func (d *ConcurrentDownloader) Error() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.error
}

func downloadEthConcurrently(c *ConcurrentDownloader, client BalanceReader, batch BatchDownloader, account common.Address, low, high *big.Int) {
	c.Go(func(ctx context.Context) ([]Transfer, error) {
		log.Debug("eth transfers comparing blocks", "low", low, "high", high)
		lb, err := client.BalanceAt(ctx, account, low)
		if err != nil {
			return nil, err
		}
		hb, err := client.BalanceAt(ctx, account, high)
		if err != nil {
			return nil, err
		}
		if lb.Cmp(hb) == 0 {
			log.Debug("balances are equal", "low", low, "high", high)
			return nil, nil
		}
		if new(big.Int).Sub(high, low).Cmp(one) == 0 {
			log.Debug("higher block is a parent", "low", low, "high", high)
			return batch.GetTransfersInRange(ctx, high, high)
		}
		mid := new(big.Int).Add(low, high)
		mid = mid.Div(mid, two)
		log.Debug("balances are not equal spawn two concurrent downloaders", "low", low, "mid", mid, "high", high)
		downloadEthConcurrently(c, client, batch, account, low, mid)
		downloadEthConcurrently(c, client, batch, account, mid, high)
		return nil, nil
	})
}
