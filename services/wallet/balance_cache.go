package wallet

import (
	"context"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type balanceCache struct {
	// cache maps an address to a map of a block number and the balance of this particular address
	cache map[common.Address]map[*big.Int]*big.Int
	lock  sync.RWMutex
}

func (b *balanceCache) readCachedBalance(account common.Address, blockNumber *big.Int) *big.Int {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.cache[account][blockNumber]
}

func (b *balanceCache) addBalanceToCache(account common.Address, blockNumber *big.Int, balance *big.Int) {
	b.lock.Lock()
	defer b.lock.Unlock()

	_, exists := b.cache[account]
	if !exists {
		b.cache[account] = make(map[*big.Int]*big.Int)
	}
	b.cache[account][blockNumber] = balance
}

func (b *balanceCache) BalanceAt(client BalanceReader, ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	cachedBalance := b.readCachedBalance(account, blockNumber)
	if cachedBalance != nil {
		return cachedBalance, nil
	}
	balance, err := client.BalanceAt(ctx, account, blockNumber)
	if err != nil {
		return nil, err
	}
	b.addBalanceToCache(account, blockNumber, balance)

	return balance, nil
}

func newBalanceCache() *balanceCache {
	return &balanceCache{
		cache: make(map[common.Address]map[*big.Int]*big.Int),
	}
}
