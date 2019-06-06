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
	runner := NewConcurrentRunner(ctx)
	result := &Result{}
	return &ConcurrentDownloader{runner, result}
}

type ConcurrentDownloader struct {
	*ConcurrentRunner
	*Result
}

type Result struct {
	mu        sync.Mutex
	transfers []Transfer
}

func (r *Result) Add(transfers ...Transfer) {
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

func NewConcurrentRunner(ctx context.Context) *ConcurrentRunner {
	// TODO(dshulyak) rename to atomic group and keep interface consistent with regular Group.
	ctx, cancel := context.WithCancel(ctx)
	return &ConcurrentRunner{
		ctx:    ctx,
		cancel: cancel,
	}
}

// ConcurrentRunner runs group atomically.
type ConcurrentRunner struct {
	ctx    context.Context
	cancel func()
	wg     sync.WaitGroup

	mu    sync.Mutex
	error error
}

// Go spawns function in a goroutine and stores results or errors.
func (d *ConcurrentRunner) Go(f func(context.Context) error) {
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		err := f(d.ctx)
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
	}()
}

// Wait for all downloaders to finish.
func (d *ConcurrentRunner) Wait() {
	d.wg.Wait()
	if d.Error() == nil {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.cancel()
	}
}

func (d *ConcurrentRunner) WaitAsync() <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		d.Wait()
		close(ch)
	}()
	return ch
}

// Error stores an error that was reported by any of the downloader. Should be called after Wait.
func (d *ConcurrentRunner) Error() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.error
}

// TransferDownloader downloads transfers from single block using number.
type TransferDownloader interface {
	GetTransfersByNumber(context.Context, *big.Int) ([]Transfer, error)
}

func downloadEthConcurrently(c *ConcurrentDownloader, client BalanceReader, downloader TransferDownloader, account common.Address, low, high *big.Int) {
	c.Go(func(ctx context.Context) error {
		log.Debug("eth transfers comparing blocks", "low", low, "high", high)
		lb, err := client.BalanceAt(ctx, account, low)
		if err != nil {
			return err
		}
		hb, err := client.BalanceAt(ctx, account, high)
		if err != nil {
			return err
		}
		if lb.Cmp(hb) == 0 {
			log.Debug("balances are equal", "low", low, "high", high)
			return nil
		}
		if new(big.Int).Sub(high, low).Cmp(one) == 0 {
			log.Debug("higher block is a parent", "low", low, "high", high)
			transfers, err := downloader.GetTransfersByNumber(ctx, high)
			if err != nil {
				return err
			}
			c.Add(transfers...)
			return nil
		}
		mid := new(big.Int).Add(low, high)
		mid = mid.Div(mid, two)
		log.Debug("balances are not equal spawn two concurrent downloaders", "low", low, "mid", mid, "high", high)
		downloadEthConcurrently(c, client, downloader, account, low, mid)
		downloadEthConcurrently(c, client, downloader, account, mid, high)
		return nil
	})
}
