package balance

import (
	"context"
	"math/big"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/rpc/chain"
)

type nonceRange struct {
	nonce int64
	max   *big.Int
	min   *big.Int
}

// Reader interface for reading balance at a specified address.
type Reader interface {
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
	NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	FullTransactionByBlockNumberAndIndex(ctx context.Context, blockNumber *big.Int, index uint) (*chain.FullTransaction, error)
}

// Cacher interface for caching balance to BalanceCache. Requires BalanceReader to fetch balance.
type Cacher interface {
	BalanceAt(ctx context.Context, client Reader, account common.Address, blockNumber *big.Int) (*big.Int, error)
	NonceAt(ctx context.Context, client Reader, account common.Address, blockNumber *big.Int) (*int64, error)
	Clear()
	Cache() CacheIface
}

// Interface for cache of balances.
type CacheIface interface {
	GetBalance(account common.Address, blockNumber *big.Int) *big.Int
	GetNonce(account common.Address, blockNumber *big.Int) *int64
	AddBalance(account common.Address, blockNumber *big.Int, balance *big.Int)
	AddNonce(account common.Address, blockNumber *big.Int, nonce *int64)
	Clear()
}

type Cache struct {
	// balances maps an address to a map of a block number and the balance of this particular address
	balances     map[common.Address]map[uint64]*big.Int // we don't care about block number overflow as we use cache only for comparing balances when fetching, not for UI
	nonces       map[common.Address]map[uint64]*int64   // we don't care about block number overflow as we use cache only for comparing balances when fetching, not for UI
	nonceRanges  map[common.Address]map[int64]nonceRange
	sortedRanges map[common.Address][]nonceRange
	rw           sync.RWMutex
}

func NewCache() *Cache {
	return &Cache{
		balances:     make(map[common.Address]map[uint64]*big.Int),
		nonces:       make(map[common.Address]map[uint64]*int64),
		nonceRanges:  make(map[common.Address]map[int64]nonceRange),
		sortedRanges: make(map[common.Address][]nonceRange),
	}
}

func (b *Cache) Clear() {
	b.rw.Lock()
	defer b.rw.Unlock()

	for address, cache := range b.balances {
		if len(cache) == 0 {
			continue
		}

		var maxBlock uint64 = 0
		var minBlock uint64 = 18446744073709551615
		for key := range cache {
			if key > maxBlock {
				maxBlock = key
			}
			if key < minBlock {
				minBlock = key
			}
		}
		newCache := make(map[uint64]*big.Int)
		newCache[maxBlock] = cache[maxBlock]
		newCache[minBlock] = cache[minBlock]
		b.balances[address] = newCache
	}
	for address, cache := range b.nonces {
		if len(cache) == 0 {
			continue
		}

		var maxBlock uint64 = 0
		var minBlock uint64 = 18446744073709551615
		for key := range cache {
			if key > maxBlock {
				maxBlock = key
			}
			if key < minBlock {
				minBlock = key
			}
		}
		newCache := make(map[uint64]*int64)
		newCache[maxBlock] = cache[maxBlock]
		newCache[minBlock] = cache[minBlock]
		b.nonces[address] = newCache
	}
	b.nonceRanges = make(map[common.Address]map[int64]nonceRange)
	b.sortedRanges = make(map[common.Address][]nonceRange)
}

func (b *Cache) GetBalance(account common.Address, blockNumber *big.Int) *big.Int {
	b.rw.RLock()
	defer b.rw.RUnlock()

	return b.balances[account][blockNumber.Uint64()]
}

func (b *Cache) AddBalance(account common.Address, blockNumber *big.Int, balance *big.Int) {
	b.rw.Lock()
	defer b.rw.Unlock()

	_, exists := b.balances[account]
	if !exists {
		b.balances[account] = make(map[uint64]*big.Int)
	}
	b.balances[account][blockNumber.Uint64()] = balance
}

func (b *Cache) BalanceAt(ctx context.Context, client Reader, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	cachedBalance := b.GetBalance(account, blockNumber)
	if cachedBalance != nil {
		return cachedBalance, nil
	}
	balance, err := client.BalanceAt(ctx, account, blockNumber)
	if err != nil {
		return nil, err
	}
	b.AddBalance(account, blockNumber, balance)

	return balance, nil
}

func (b *Cache) GetNonce(account common.Address, blockNumber *big.Int) *int64 {
	b.rw.RLock()
	defer b.rw.RUnlock()

	return b.nonces[account][blockNumber.Uint64()]
}

func (b *Cache) Cache() CacheIface {
	return b
}

func (b *Cache) sortRanges(account common.Address) {
	keys := make([]int, 0, len(b.nonceRanges[account]))
	for k := range b.nonceRanges[account] {
		keys = append(keys, int(k))
	}

	sort.Ints(keys) // This will not work for keys > 2^31

	ranges := []nonceRange{}
	for _, k := range keys {
		r := b.nonceRanges[account][int64(k)]
		ranges = append(ranges, r)
	}

	b.sortedRanges[account] = ranges
}

func (b *Cache) findNonceInRange(account common.Address, block *big.Int) *int64 {
	b.rw.RLock()
	defer b.rw.RUnlock()

	for k := range b.sortedRanges[account] {
		nr := b.sortedRanges[account][k]
		cmpMin := nr.min.Cmp(block)
		if cmpMin == 1 {
			return nil
		} else if cmpMin == 0 {
			return &nr.nonce
		} else {
			cmpMax := nr.max.Cmp(block)
			if cmpMax >= 0 {
				return &nr.nonce
			}
		}
	}

	return nil
}

func (b *Cache) updateNonceRange(account common.Address, blockNumber *big.Int, nonce *int64) {
	_, exists := b.nonceRanges[account]
	if !exists {
		b.nonceRanges[account] = make(map[int64]nonceRange)
	}
	nr, exists := b.nonceRanges[account][*nonce]
	if !exists {
		r := nonceRange{
			max:   big.NewInt(0).Set(blockNumber),
			min:   big.NewInt(0).Set(blockNumber),
			nonce: *nonce,
		}
		b.nonceRanges[account][*nonce] = r
	} else {
		if nr.max.Cmp(blockNumber) == -1 {
			nr.max.Set(blockNumber)
		}

		if nr.min.Cmp(blockNumber) == 1 {
			nr.min.Set(blockNumber)
		}

		b.nonceRanges[account][*nonce] = nr
		b.sortRanges(account)
	}
}

func (b *Cache) AddNonce(account common.Address, blockNumber *big.Int, nonce *int64) {
	b.rw.Lock()
	defer b.rw.Unlock()

	_, exists := b.nonces[account]
	if !exists {
		b.nonces[account] = make(map[uint64]*int64)
	}
	b.nonces[account][blockNumber.Uint64()] = nonce
	b.updateNonceRange(account, blockNumber, nonce)
}

func (b *Cache) NonceAt(ctx context.Context, client Reader, account common.Address, blockNumber *big.Int) (*int64, error) {
	cachedNonce := b.GetNonce(account, blockNumber)
	if cachedNonce != nil {
		return cachedNonce, nil
	}
	rangeNonce := b.findNonceInRange(account, blockNumber)
	if rangeNonce != nil {
		return rangeNonce, nil
	}

	nonce, err := client.NonceAt(ctx, account, blockNumber)
	if err != nil {
		return nil, err
	}
	int64Nonce := int64(nonce)
	b.AddNonce(account, blockNumber, &int64Nonce)

	return &int64Nonce, nil
}
