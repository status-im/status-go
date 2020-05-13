package wallet

import (
	"context"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type balanceCache struct {
	// cache maps an address to a map of a block number and the balance of this particular address
	cache            map[common.Address]map[*big.Int]*big.Int
	requestCounter   map[common.Address]uint
	cacheHitsCounter map[common.Address]uint
	rw               sync.RWMutex
}

type BalanceCache interface {
	BalanceAt(ctx context.Context, client BalanceReader, account common.Address, blockNumber *big.Int) (*big.Int, error)
}

func (b *balanceCache) readCachedBalance(account common.Address, blockNumber *big.Int) *big.Int {
	b.rw.RLock()
	defer b.rw.RUnlock()

	return b.cache[account][blockNumber]
}

func (b *balanceCache) addBalanceToCache(account common.Address, blockNumber *big.Int, balance *big.Int) {
	b.rw.Lock()
	defer b.rw.Unlock()

	_, exists := b.cache[account]
	if !exists {
		b.cache[account] = make(map[*big.Int]*big.Int)
	}
	b.cache[account][blockNumber] = balance
}

func (b *balanceCache) incRequestsNumber(account common.Address) {
	b.rw.Lock()
	defer b.rw.Unlock()

	cnt, ok := b.requestCounter[account]
	if !ok {
		b.requestCounter[account] = 1
	}

	b.requestCounter[account] = cnt + 1
}

func (b *balanceCache) incCacheHitNumber(account common.Address) {
	b.rw.Lock()
	defer b.rw.Unlock()

	cnt, ok := b.cacheHitsCounter[account]
	if !ok {
		b.cacheHitsCounter[account] = 1
	}

	b.cacheHitsCounter[account] = cnt + 1
}

func (b *balanceCache) getStats(account common.Address) (uint, uint) {
	b.rw.RLock()
	defer b.rw.RUnlock()

	return b.requestCounter[account], b.cacheHitsCounter[account]
}

func (b *balanceCache) BalanceAt(ctx context.Context, client BalanceReader, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	b.incRequestsNumber(account)
	cachedBalance := b.readCachedBalance(account, blockNumber)
	if cachedBalance != nil {
		b.incCacheHitNumber(account)
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
		cache:            make(map[common.Address]map[*big.Int]*big.Int),
		requestCounter:   make(map[common.Address]uint),
		cacheHitsCounter: make(map[common.Address]uint),
	}
}
