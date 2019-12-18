package wallet

import (
	"context"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type BalanceCache struct {
	cache map[common.Address]map[*big.Int]*big.Int
	lock  sync.RWMutex
}

func (b *BalanceCache) readCachedBalance(account common.Address, blockNumber *big.Int) (*big.Int, bool) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	balance, exists := b.cache[account][blockNumber]
	return balance, exists
}

func (b *BalanceCache) addBalanceToCache(account common.Address, blockNumber *big.Int, balance *big.Int) {
	b.lock.Lock()
	defer b.lock.Unlock()

	_, exists := b.cache[account]
	if !exists {
		b.cache[account] = make(map[*big.Int]*big.Int)
	}
	b.cache[account][blockNumber] = balance
}

func (b *BalanceCache) BalanceAt(client BalanceReader, ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	cachedBalance, exists := b.readCachedBalance(account, blockNumber)
	if exists {
		return cachedBalance, nil
	} else {
		balance, err := client.BalanceAt(ctx, account, blockNumber)
		if err != nil {
			return nil, err
		}
		b.addBalanceToCache(account, blockNumber, balance)

		return balance, nil
	}
}

func NewBalanceCache() *BalanceCache {
	return &BalanceCache{
		cache: make(map[common.Address]map[*big.Int]*big.Int),
		lock:  sync.RWMutex{},
	}
}
