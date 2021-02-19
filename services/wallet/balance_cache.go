package wallet

import (
	"context"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type balanceCache struct {
	// balances maps an address to a map of a block number and the balance of this particular address
	balances map[common.Address]map[*big.Int]*big.Int
	nonces   map[common.Address]map[*big.Int]*int64
	rw       sync.RWMutex
}

type BalanceCache interface {
	BalanceAt(ctx context.Context, client BalanceReader, account common.Address, blockNumber *big.Int) (*big.Int, error)
	NonceAt(ctx context.Context, client BalanceReader, account common.Address, blockNumber *big.Int) (*int64, error)
}

func (b *balanceCache) ReadCachedBalance(account common.Address, blockNumber *big.Int) *big.Int {
	b.rw.RLock()
	defer b.rw.RUnlock()

	return b.balances[account][blockNumber]
}

func (b *balanceCache) addBalanceToCache(account common.Address, blockNumber *big.Int, balance *big.Int) {
	b.rw.Lock()
	defer b.rw.Unlock()

	_, exists := b.balances[account]
	if !exists {
		b.balances[account] = make(map[*big.Int]*big.Int)
	}
	b.balances[account][blockNumber] = balance
}

func (b *balanceCache) BalanceAt(ctx context.Context, client BalanceReader, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	cachedBalance := b.ReadCachedBalance(account, blockNumber)
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

func (b *balanceCache) ReadCachedNonce(account common.Address, blockNumber *big.Int) *int64 {
	b.rw.RLock()
	defer b.rw.RUnlock()

	return b.nonces[account][blockNumber]
}

func (b *balanceCache) addNonceToCache(account common.Address, blockNumber *big.Int, nonce *int64) {
	b.rw.Lock()
	defer b.rw.Unlock()

	_, exists := b.nonces[account]
	if !exists {
		b.nonces[account] = make(map[*big.Int]*int64)
	}
	b.nonces[account][blockNumber] = nonce
}

func (b *balanceCache) NonceAt(ctx context.Context, client BalanceReader, account common.Address, blockNumber *big.Int) (*int64, error) {
	cachedNonce := b.ReadCachedNonce(account, blockNumber)
	if cachedNonce != nil {
		return cachedNonce, nil
	}
	nonce, err := client.NonceAt(ctx, account, blockNumber)
	if err != nil {
		return nil, err
	}
	int64Nonce := int64(nonce)
	b.addNonceToCache(account, blockNumber, &int64Nonce)

	return &int64Nonce, nil
}

func newBalanceCache() *balanceCache {
	return &balanceCache{
		balances: make(map[common.Address]map[*big.Int]*big.Int),
		nonces:   make(map[common.Address]map[*big.Int]*int64),
	}
}
