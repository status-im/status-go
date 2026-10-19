package multistandardbalance

import (
	"context"
	"fmt"
	"maps"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type StorageMemory struct {
	nativeBalances  map[BalancesKey]balanceEntry[*big.Int]
	erc20Balances   map[BalancesKey]tokensEntry[ContractAddress]
	erc721Balances  map[BalancesKey]tokensEntry[ContractAddress]
	erc1155Balances map[BalancesKey]tokensEntry[HashableCollectibleID]
	mu              sync.RWMutex
}

func NewStorageMemory() *StorageMemory {
	return &StorageMemory{
		nativeBalances:  make(map[BalancesKey]balanceEntry[*big.Int]),
		erc20Balances:   make(map[BalancesKey]tokensEntry[ContractAddress]),
		erc721Balances:  make(map[BalancesKey]tokensEntry[ContractAddress]),
		erc1155Balances: make(map[BalancesKey]tokensEntry[HashableCollectibleID]),
	}
}

type balanceEntry[T any] struct {
	balance T
	state   State
}

// tokensEntry keeps the block each token was last read at. Its maps are
// replaced, never changed, so the maps Get returns stay as they were.
type tokensEntry[K comparable] struct {
	balances map[K]*big.Int
	blocks   map[K]*big.Int
	state    State
}

// mergeTokens takes each token of a read unless it was already read at the
// same or a newer block, and keeps the tokens the read did not cover. The
// state is that of the newest read.
func mergeTokens[K comparable](entryMap map[BalancesKey]tokensEntry[K], key BalancesKey, read map[K]*big.Int, state State) (balanceChanged bool, oldState State, err error) {
	oldState = defaultState()
	old, exists := entryMap[key]
	if exists {
		oldState = old.state
	}

	if state.AtBlockNumber == nil {
		err = fmt.Errorf("new state at block number is nil")
		return
	}

	merged := tokensEntry[K]{
		balances: make(map[K]*big.Int, max(len(old.balances), len(read))),
		blocks:   make(map[K]*big.Int, max(len(old.blocks), len(read))),
		state:    oldState,
	}
	maps.Copy(merged.balances, old.balances)
	maps.Copy(merged.blocks, old.blocks)

	updated := false
	for token, balance := range read {
		if block, wasRead := merged.blocks[token]; wasRead && state.AtBlockNumber.Cmp(block) <= 0 {
			continue
		}
		if previous, known := merged.balances[token]; !known || previous.Cmp(balance) != 0 {
			balanceChanged = true
		}
		merged.balances[token] = balance
		merged.blocks[token] = state.AtBlockNumber
		updated = true
	}
	if oldState.AtBlockNumber == nil || state.AtBlockNumber.Cmp(oldState.AtBlockNumber) > 0 {
		merged.state = state
		updated = true
	}
	// The first read of a key reports a change even when it holds no token.
	balanceChanged = balanceChanged || !exists

	if updated {
		entryMap[key] = merged
	}
	return
}

func updateIfNeeded[T any](entryMap map[BalancesKey]balanceEntry[T], key BalancesKey, newEntry balanceEntry[T], isEqualFn func(old T, new T) bool) (balanceChanged bool, oldState State, err error) {
	balanceChanged = false

	oldBalance := newEntry.balance
	oldState = defaultState()
	oldEntryExists := false
	// Set oldState to the existing entry's state if it exists
	if oldEntry, exists := entryMap[key]; exists {
		oldBalance = oldEntry.balance
		oldState = oldEntry.state
		oldEntryExists = true
	}

	// Check if new state has a valid block number
	if newEntry.state.AtBlockNumber == nil {
		err = fmt.Errorf("new state at block number is nil")
		return
	}

	// If old state has a block number, check if new state is newer
	if oldState.AtBlockNumber != nil && newEntry.state.AtBlockNumber.Cmp(oldState.AtBlockNumber) <= 0 {
		return
	}
	// Report a change if the entry doesn't exist or the balance is different
	balanceChanged = !oldEntryExists || !isEqualFn(oldBalance, newEntry.balance)

	entryMap[key] = newEntry
	return
}

func (s *StorageMemory) UpdateNativeBalance(ctx context.Context, key BalancesKey, balance *big.Int, state State) (balanceChanged bool, oldState State, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return updateIfNeeded(s.nativeBalances, key, balanceEntry[*big.Int]{balance: balance, state: state}, func(old *big.Int, new *big.Int) bool {
		return old.Cmp(new) == 0
	})
}

func (s *StorageMemory) GetNativeBalance(ctx context.Context, key BalancesKey) (balance *big.Int, state State, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state = defaultState()

	if entry, exists := s.nativeBalances[key]; exists {
		balance = entry.balance
		state = entry.state
	}

	return
}

func (s *StorageMemory) UpdateERC20Balances(ctx context.Context, key BalancesKey, balances map[ContractAddress]*big.Int, state State) (balanceChanged bool, oldState State, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return mergeTokens(s.erc20Balances, key, balances, state)
}

func (s *StorageMemory) GetERC20Balances(ctx context.Context, key BalancesKey) (balances map[ContractAddress]*big.Int, state State, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state = defaultState()
	if entry, exists := s.erc20Balances[key]; exists {
		balances = entry.balances
		state = entry.state
	}

	return
}

func (s *StorageMemory) UpdateERC721Balances(ctx context.Context, key BalancesKey, balances map[ContractAddress]*big.Int, state State) (balanceChanged bool, oldState State, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return mergeTokens(s.erc721Balances, key, balances, state)
}

func (s *StorageMemory) GetERC721Balances(ctx context.Context, key BalancesKey) (balances map[ContractAddress]*big.Int, state State, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state = defaultState()
	if entry, exists := s.erc721Balances[key]; exists {
		balances = entry.balances
		state = entry.state
	}

	return
}

func (s *StorageMemory) UpdateERC1155Balances(ctx context.Context, key BalancesKey, balances map[HashableCollectibleID]*big.Int, state State) (balanceChanged bool, oldState State, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return mergeTokens(s.erc1155Balances, key, balances, state)
}

func (s *StorageMemory) GetERC1155Balances(ctx context.Context, key BalancesKey) (balances map[HashableCollectibleID]*big.Int, state State, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state = defaultState()
	if entry, exists := s.erc1155Balances[key]; exists {
		balances = entry.balances
		state = entry.state
	}

	return
}

func (s *StorageMemory) ClearMissingAccounts(ctx context.Context, accounts []common.Address) {
	s.mu.Lock()
	defer s.mu.Unlock()

	deleteAccountsNotInList(s.nativeBalances, accounts)
	deleteAccountsNotInList(s.erc20Balances, accounts)
	deleteAccountsNotInList(s.erc721Balances, accounts)
	deleteAccountsNotInList(s.erc1155Balances, accounts)
}

func (s *StorageMemory) ClearMissingChains(ctx context.Context, chains []uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	deleteChainsNotInList(s.nativeBalances, chains)
	deleteChainsNotInList(s.erc20Balances, chains)
	deleteChainsNotInList(s.erc721Balances, chains)
	deleteChainsNotInList(s.erc1155Balances, chains)
}
