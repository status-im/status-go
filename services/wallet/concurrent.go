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
	runner := NewAtomicGroup(ctx)
	result := &Result{}
	return &ConcurrentDownloader{runner, result}
}

type ConcurrentDownloader struct {
	*AtomicGroup
	*Result
}

type Result struct {
	mu        sync.Mutex
	transfers []Transfer
}

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

// TransferDownloader downloads transfers from single block using number.
type TransferDownloader interface {
	GetTransfersByNumber(context.Context, *big.Int) ([]Transfer, error)
}

func downloadEthConcurrently(c *ConcurrentDownloader, client BalanceReader, balanceCache *BalanceCache, downloader TransferDownloader, account common.Address, low, high *big.Int) {
	c.Add(func(ctx context.Context) error {
		if low.Cmp(high) >= 0 {
			return nil
		}
		log.Debug("eth transfers comparing blocks", "low", low, "high", high)
		lb, err := balanceCache.BalanceAt(client, ctx, account, low)
		//lb, err := client.BalanceAt(ctx, account, low)
		if err != nil {
			return err
		}
		hb, err := balanceCache.BalanceAt(client, ctx, account, high)
		//hb, err := client.BalanceAt(ctx, account, high)
		if err != nil {
			return err
		}
		if lb.Cmp(hb) == 0 {
			log.Debug("balances are equal", "low", low, "high", high)
			// In case if balances are equal but non zero we want to check if
			// eth_getTransactionCount return different values, because there
			// still might be transactions
			if lb.Cmp(zero) != 0 {
				return nil
			}

			ln, err := client.NonceAt(ctx, account, low)
			if err != nil {
				return err
			}
			hn, err := client.NonceAt(ctx, account, high)
			if err != nil {
				return err
			}
			if ln == hn {
				log.Debug("transaction count is also equal", "low", low, "high", high)
				return nil
			}
		}
		if new(big.Int).Sub(high, low).Cmp(one) == 0 {
			transfers, err := downloader.GetTransfersByNumber(ctx, high)
			if err != nil {
				return err
			}
			c.Push(transfers...)
			return nil
		}
		mid := new(big.Int).Add(low, high)
		mid = mid.Div(mid, two)
		balanceCache.BalanceAt(client, ctx, account, mid)
		log.Debug("balances are not equal. spawn two concurrent downloaders", "low", low, "mid", mid, "high", high)
		downloadEthConcurrently(c, client, balanceCache, downloader, account, low, mid)
		downloadEthConcurrently(c, client, balanceCache, downloader, account, mid, high)
		return nil
	})
}
