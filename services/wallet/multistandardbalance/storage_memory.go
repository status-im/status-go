package multistandardbalance

import (
	"context"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type StorageMemory struct {
	nativeBalances  map[BalancesKey]balanceEntry[*big.Int]
	erc20Balances   map[BalancesKey]balanceEntry[map[ContractAddress]*big.Int]
	erc721Balances  map[BalancesKey]balanceEntry[map[ContractAddress]*big.Int]
	erc1155Balances map[BalancesKey]balanceEntry[map[HashableCollectibleID]*big.Int]
	mu              sync.RWMutex
}

func NewStorageMemory() *StorageMemory {
	return &StorageMemory{
		nativeBalances:  make(map[BalancesKey]balanceEntry[*big.Int]),
		erc20Balances:   make(map[BalancesKey]balanceEntry[map[ContractAddress]*big.Int]),
		erc721Balances:  make(map[BalancesKey]balanceEntry[map[ContractAddress]*big.Int]),
		erc1155Balances: make(map[BalancesKey]balanceEntry[map[HashableCollectibleID]*big.Int]),
	}
}

type balanceEntry[T any] struct {
	balance T
	state   State
}

func updateIfNeeded[T any](entryMap map[BalancesKey]balanceEntry[T], key BalancesKey, newEntry balanceEntry[T], isEqualFn func(old T, new T) bool) (balanceChanged bool, oldState State, err error) {
	balanceChanged = false

	oldBalance := newEntry.balance
	oldState = defaultState()
	// Set oldState to the existing entry's state if it exists
	if oldEntry, exists := entryMap[key]; exists {
		oldBalance = oldEntry.balance
		oldState = oldEntry.state
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
	balanceChanged = !isEqualFn(oldBalance, newEntry.balance)

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

	return updateIfNeeded(s.erc20Balances, key, balanceEntry[map[ContractAddress]*big.Int]{balance: balances, state: state}, isBigIntMapEqual)
}

func (s *StorageMemory) GetERC20Balances(ctx context.Context, key BalancesKey) (balances map[ContractAddress]*big.Int, state State, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state = defaultState()
	if entry, exists := s.erc20Balances[key]; exists {
		balances = entry.balance
		state = entry.state
	}

	return
}

func (s *StorageMemory) UpdateERC721Balances(ctx context.Context, key BalancesKey, balances map[ContractAddress]*big.Int, state State) (balanceChanged bool, oldState State, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return updateIfNeeded(s.erc721Balances, key, balanceEntry[map[ContractAddress]*big.Int]{balance: balances, state: state}, isBigIntMapEqual)
}

func (s *StorageMemory) GetERC721Balances(ctx context.Context, key BalancesKey) (balances map[ContractAddress]*big.Int, state State, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state = defaultState()
	if entry, exists := s.erc721Balances[key]; exists {
		balances = entry.balance
		state = entry.state
	}

	return
}

func (s *StorageMemory) UpdateERC1155Balances(ctx context.Context, key BalancesKey, balances map[HashableCollectibleID]*big.Int, state State) (balanceChanged bool, oldState State, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return updateIfNeeded(s.erc1155Balances, key, balanceEntry[map[HashableCollectibleID]*big.Int]{balance: balances, state: state}, isBigIntMapEqual)
}

func (s *StorageMemory) GetERC1155Balances(ctx context.Context, key BalancesKey) (balances map[HashableCollectibleID]*big.Int, state State, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state = defaultState()
	if entry, exists := s.erc1155Balances[key]; exists {
		balances = entry.balance
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
