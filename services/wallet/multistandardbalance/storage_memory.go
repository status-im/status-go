package multistandardbalance

import (
	"context"
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

func (s *StorageMemory) UpdateNativeBalance(ctx context.Context, key BalancesKey, balance *big.Int, state State) (balanceChanged bool, oldState State, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldState = defaultState()
	balanceChanged = false // If initial store for this key, balanceChanged is false
	if oldEntry, exists := s.nativeBalances[key]; exists {
		oldState = oldEntry.state
		if oldState.AtBlockNumber.Cmp(state.AtBlockNumber) >= 0 {
			// New balance is not newer than stored one, skip update
			return
		}
		balanceChanged = oldEntry.balance.Cmp(balance) != 0
	}

	s.nativeBalances[key] = balanceEntry[*big.Int]{balance: balance, state: state}
	return
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

	oldState = defaultState()
	balanceChanged = false // If initial store for this key, balanceChanged is false
	if oldEntry, exists := s.erc20Balances[key]; exists {
		oldState = oldEntry.state
		if oldState.AtBlockNumber.Cmp(state.AtBlockNumber) >= 0 {
			// New balance is not newer than stored one, skip update
			return
		}
		balanceChanged = !isBigIntMapEqual(oldEntry.balance, balances)
	}

	s.erc20Balances[key] = balanceEntry[map[ContractAddress]*big.Int]{balance: balances, state: state}
	return
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

	oldState = defaultState()
	balanceChanged = false // If initial store for this key, balanceChanged is false
	if oldEntry, exists := s.erc721Balances[key]; exists {
		oldState = oldEntry.state
		if oldState.AtBlockNumber.Cmp(state.AtBlockNumber) >= 0 {
			// New balance is not newer than stored one, skip update
			return
		}
		balanceChanged = !isBigIntMapEqual(oldEntry.balance, balances)
	}

	s.erc721Balances[key] = balanceEntry[map[ContractAddress]*big.Int]{balance: balances, state: state}
	return
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

	oldState = defaultState()
	balanceChanged = false // If initial store for this key, balanceChanged is false
	if oldEntry, exists := s.erc1155Balances[key]; exists {
		oldState = oldEntry.state
		if oldState.AtBlockNumber.Cmp(state.AtBlockNumber) >= 0 {
			// New balance is not newer than stored one, skip update
			return
		}
		balanceChanged = !isBigIntMapEqual(oldEntry.balance, balances)
	}

	s.erc1155Balances[key] = balanceEntry[map[HashableCollectibleID]*big.Int]{balance: balances, state: state}
	return
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
