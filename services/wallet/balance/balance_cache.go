package balance

import (
	"context"
	"math"
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
	NetworkID() uint64
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
	GetBalance(account common.Address, chainID uint64, blockNumber *big.Int) *big.Int
	GetNonce(account common.Address, chainID uint64, blockNumber *big.Int) *int64
	AddBalance(account common.Address, chainID uint64, blockNumber *big.Int, balance *big.Int)
	AddNonce(account common.Address, chainID uint64, blockNumber *big.Int, nonce *int64)
	Clear()
}

type balanceCacheType map[common.Address]map[uint64]map[uint64]*big.Int      // address->chainID->blockNumber->balance
type nonceCacheType map[common.Address]map[uint64]map[uint64]*int64          // address->chainID->blockNumber->nonce
type nonceRangesCacheType map[common.Address]map[uint64]map[int64]nonceRange // address->chainID->blockNumber->nonceRange
type sortedNonceRangesCacheType map[common.Address]map[uint64][]nonceRange   // address->chainID->[]nonceRange

type Cache struct {
	// balances maps an address to a map of a block number and the balance of this particular address
	balances     balanceCacheType
	nonces       nonceCacheType
	nonceRanges  nonceRangesCacheType
	sortedRanges sortedNonceRangesCacheType
	rw           sync.RWMutex
}

func NewCache() *Cache {
	return &Cache{
		balances:     make(balanceCacheType),
		nonces:       make(nonceCacheType),
		nonceRanges:  make(nonceRangesCacheType),
		sortedRanges: make(sortedNonceRangesCacheType),
	}
}

func (b *Cache) Clear() {
	b.rw.Lock()
	defer b.rw.Unlock()

	for address, chainCache := range b.balances {
		if len(chainCache) == 0 {
			continue
		}

		for chainID, cache := range chainCache {
			if len(cache) == 0 {
				continue
			}

			var maxBlock uint64 = 0
			var minBlock uint64 = math.MaxUint64
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
			b.balances[address][chainID] = newCache
		}
	}
	for address, chainCache := range b.nonces {
		if len(chainCache) == 0 {
			continue
		}

		for chainID, cache := range chainCache {
			var maxBlock uint64 = 0
			var minBlock uint64 = math.MaxUint64
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
			b.nonces[address][chainID] = newCache
		}
	}
	b.nonceRanges = make(nonceRangesCacheType)
	b.sortedRanges = make(sortedNonceRangesCacheType)
}

func (b *Cache) GetBalance(account common.Address, chainID uint64, blockNumber *big.Int) *big.Int {
	b.rw.RLock()
	defer b.rw.RUnlock()

	if b.balances[account] == nil || b.balances[account][chainID] == nil {
		return nil
	}

	return b.balances[account][chainID][blockNumber.Uint64()]
}

func (b *Cache) AddBalance(account common.Address, chainID uint64, blockNumber *big.Int, balance *big.Int) {
	b.rw.Lock()
	defer b.rw.Unlock()

	_, exists := b.balances[account]
	if !exists {
		b.balances[account] = make(map[uint64]map[uint64]*big.Int)
	}

	_, exists = b.balances[account][chainID]
	if !exists {
		b.balances[account][chainID] = make(map[uint64]*big.Int)
	}

	b.balances[account][chainID][blockNumber.Uint64()] = balance
}

func (b *Cache) BalanceAt(ctx context.Context, client Reader, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	cachedBalance := b.GetBalance(account, client.NetworkID(), blockNumber)
	if cachedBalance != nil {
		return cachedBalance, nil
	}
	balance, err := client.BalanceAt(ctx, account, blockNumber)
	if err != nil {
		return nil, err
	}
	b.AddBalance(account, client.NetworkID(), blockNumber, balance)

	return balance, nil
}

func (b *Cache) GetNonce(account common.Address, chainID uint64, blockNumber *big.Int) *int64 {
	b.rw.RLock()
	defer b.rw.RUnlock()

	if b.nonces[account] == nil || b.nonces[account][chainID] == nil {
		return nil
	}
	return b.nonces[account][chainID][blockNumber.Uint64()]
}

func (b *Cache) Cache() CacheIface {
	return b
}

func (b *Cache) sortRanges(account common.Address, chainID uint64) {
	keys := make([]int, 0, len(b.nonceRanges[account][chainID]))
	for k := range b.nonceRanges[account][chainID] {
		keys = append(keys, int(k))
	}

	sort.Ints(keys) // This will not work for keys > 2^31

	ranges := []nonceRange{}
	for _, k := range keys {
		r := b.nonceRanges[account][chainID][int64(k)]
		ranges = append(ranges, r)
	}

	_, exists := b.sortedRanges[account]
	if !exists {
		b.sortedRanges[account] = make(map[uint64][]nonceRange)
	}

	b.sortedRanges[account][chainID] = ranges
}

func (b *Cache) findNonceInRange(account common.Address, chainID uint64, block *big.Int) *int64 {
	b.rw.RLock()
	defer b.rw.RUnlock()

	for k := range b.sortedRanges[account][chainID] {
		nr := b.sortedRanges[account][chainID][k]
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

func (b *Cache) updateNonceRange(account common.Address, chainID uint64, blockNumber *big.Int, nonce *int64) {
	_, exists := b.nonceRanges[account]
	if !exists {
		b.nonceRanges[account] = make(map[uint64]map[int64]nonceRange)
	}
	_, exists = b.nonceRanges[account][chainID]
	if !exists {
		b.nonceRanges[account][chainID] = make(map[int64]nonceRange)
	}

	nr, exists := b.nonceRanges[account][chainID][*nonce]
	if !exists {
		r := nonceRange{
			max:   big.NewInt(0).Set(blockNumber),
			min:   big.NewInt(0).Set(blockNumber),
			nonce: *nonce,
		}
		b.nonceRanges[account][chainID][*nonce] = r
	} else {
		if nr.max.Cmp(blockNumber) == -1 {
			nr.max.Set(blockNumber)
		}

		if nr.min.Cmp(blockNumber) == 1 {
			nr.min.Set(blockNumber)
		}

		b.nonceRanges[account][chainID][*nonce] = nr
		b.sortRanges(account, chainID)
	}
}

func (b *Cache) AddNonce(account common.Address, chainID uint64, blockNumber *big.Int, nonce *int64) {
	b.rw.Lock()
	defer b.rw.Unlock()

	_, exists := b.nonces[account]
	if !exists {
		b.nonces[account] = make(map[uint64]map[uint64]*int64)
	}

	_, exists = b.nonces[account][chainID]
	if !exists {
		b.nonces[account][chainID] = make(map[uint64]*int64)
	}
	b.nonces[account][chainID][blockNumber.Uint64()] = nonce
	b.updateNonceRange(account, chainID, blockNumber, nonce)
}

func (b *Cache) NonceAt(ctx context.Context, client Reader, account common.Address, blockNumber *big.Int) (*int64, error) {
	cachedNonce := b.GetNonce(account, client.NetworkID(), blockNumber)
	if cachedNonce != nil {
		return cachedNonce, nil
	}
	rangeNonce := b.findNonceInRange(account, client.NetworkID(), blockNumber)
	if rangeNonce != nil {
		return rangeNonce, nil
	}

	nonce, err := client.NonceAt(ctx, account, blockNumber)
	if err != nil {
		return nil, err
	}
	int64Nonce := int64(nonce)
	b.AddNonce(account, client.NetworkID(), blockNumber, &int64Nonce)

	return &int64Nonce, nil
}
