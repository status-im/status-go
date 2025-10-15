package multistandardbalance_test

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/services/wallet/multistandardbalance"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageMemory_UpdateNativeBalance(t *testing.T) {
	storage := multistandardbalance.NewStorageMemory()
	ctx := context.Background()

	key := multistandardbalance.BalancesKey{
		Account: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		ChainID: 1,
	}

	balance := big.NewInt(1000)
	state := multistandardbalance.State{
		AtBlockNumber: big.NewInt(12345),
		AtBlockHash:   common.HexToHash("0xabcdef"),
		FetchedAt:     time.Now().Unix(),
	}

	// Test first update (initial update always reports a balance change)
	balanceChanged, oldState, err := storage.UpdateNativeBalance(ctx, key, balance, state)
	require.NoError(t, err)
	assert.True(t, balanceChanged)
	assert.Equal(t, multistandardbalance.NeverFetched, oldState.FetchedAt)

	// Test update with same block number (should not update)
	oldBalance := big.NewInt(2000)
	oldState2 := multistandardbalance.State{
		AtBlockNumber: big.NewInt(12345),
		AtBlockHash:   common.HexToHash("0xabcdef"),
		FetchedAt:     time.Now().Unix(),
	}
	balanceChanged, oldState, err = storage.UpdateNativeBalance(ctx, key, oldBalance, oldState2)
	require.NoError(t, err)
	assert.False(t, balanceChanged)

	// Test update with newer block number
	newBalance := big.NewInt(3000)
	newState := multistandardbalance.State{
		AtBlockNumber: big.NewInt(12346),
		AtBlockHash:   common.HexToHash("0xfedcba"),
		FetchedAt:     time.Now().Unix(),
	}
	balanceChanged, oldState, err = storage.UpdateNativeBalance(ctx, key, newBalance, newState)
	require.NoError(t, err)
	assert.True(t, balanceChanged)
	assert.Equal(t, state.AtBlockNumber, oldState.AtBlockNumber)
}

func TestStorageMemory_GetNativeBalance(t *testing.T) {
	storage := multistandardbalance.NewStorageMemory()
	ctx := context.Background()

	key := multistandardbalance.BalancesKey{
		Account: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		ChainID: 1,
	}

	// Test getting non-existent balance
	balance, state, err := storage.GetNativeBalance(ctx, key)
	require.NoError(t, err)
	assert.Nil(t, balance)
	assert.Equal(t, multistandardbalance.NeverFetched, state.FetchedAt)

	// Test getting existing balance
	expectedBalance := big.NewInt(1000)
	expectedState := multistandardbalance.State{
		AtBlockNumber: big.NewInt(12345),
		AtBlockHash:   common.HexToHash("0xabcdef"),
		FetchedAt:     time.Now().Unix(),
	}

	_, _, err = storage.UpdateNativeBalance(ctx, key, expectedBalance, expectedState)
	require.NoError(t, err)

	balance, state, err = storage.GetNativeBalance(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, expectedBalance, balance)
	assert.Equal(t, expectedState.AtBlockNumber, state.AtBlockNumber)
	assert.Equal(t, expectedState.AtBlockHash, state.AtBlockHash)
	assert.Equal(t, expectedState.FetchedAt, state.FetchedAt)
}

func TestStorageMemory_UpdateERC20Balances(t *testing.T) {
	storage := multistandardbalance.NewStorageMemory()
	ctx := context.Background()

	key := multistandardbalance.BalancesKey{
		Account: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		ChainID: 1,
	}

	balances := map[multistandardbalance.ContractAddress]*big.Int{
		common.HexToAddress("0x1111111111111111111111111111111111111111"): big.NewInt(100),
		common.HexToAddress("0x2222222222222222222222222222222222222222"): big.NewInt(200),
	}

	state := multistandardbalance.State{
		AtBlockNumber: big.NewInt(12345),
		AtBlockHash:   common.HexToHash("0xabcdef"),
		FetchedAt:     time.Now().Unix(),
	}

	// Test first update (initial update always reports a balance change)
	balanceChanged, oldState, err := storage.UpdateERC20Balances(ctx, key, balances, state)
	require.NoError(t, err)
	assert.True(t, balanceChanged)
	assert.Equal(t, multistandardbalance.NeverFetched, oldState.FetchedAt)

	// Test update with same balances
	balanceChanged, oldState, err = storage.UpdateERC20Balances(ctx, key, balances, state)
	require.NoError(t, err)
	assert.False(t, balanceChanged)

	// Test update with different balances
	newBalances := map[multistandardbalance.ContractAddress]*big.Int{
		common.HexToAddress("0x1111111111111111111111111111111111111111"): big.NewInt(150),
		common.HexToAddress("0x2222222222222222222222222222222222222222"): big.NewInt(250),
	}

	newState := multistandardbalance.State{
		AtBlockNumber: big.NewInt(12346),
		AtBlockHash:   common.HexToHash("0xfedcba"),
		FetchedAt:     time.Now().Unix(),
	}

	balanceChanged, oldState, err = storage.UpdateERC20Balances(ctx, key, newBalances, newState)
	require.NoError(t, err)
	assert.True(t, balanceChanged)
}

func TestStorageMemory_GetERC20Balances(t *testing.T) {
	storage := multistandardbalance.NewStorageMemory()
	ctx := context.Background()

	key := multistandardbalance.BalancesKey{
		Account: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		ChainID: 1,
	}

	// Test getting non-existent balances
	balances, state, err := storage.GetERC20Balances(ctx, key)
	require.NoError(t, err)
	assert.Nil(t, balances)
	assert.Equal(t, multistandardbalance.NeverFetched, state.FetchedAt)

	// Test getting existing balances
	expectedBalances := map[multistandardbalance.ContractAddress]*big.Int{
		common.HexToAddress("0x1111111111111111111111111111111111111111"): big.NewInt(100),
		common.HexToAddress("0x2222222222222222222222222222222222222222"): big.NewInt(200),
	}

	expectedState := multistandardbalance.State{
		AtBlockNumber: big.NewInt(12345),
		AtBlockHash:   common.HexToHash("0xabcdef"),
		FetchedAt:     time.Now().Unix(),
	}

	_, _, err = storage.UpdateERC20Balances(ctx, key, expectedBalances, expectedState)
	require.NoError(t, err)

	balances, state, err = storage.GetERC20Balances(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, len(expectedBalances), len(balances))
	for contract, expectedBalance := range expectedBalances {
		actualBalance, exists := balances[contract]
		assert.True(t, exists)
		assert.Equal(t, expectedBalance, actualBalance)
	}
	assert.Equal(t, expectedState.AtBlockNumber, state.AtBlockNumber)
}

func TestStorageMemory_UpdateERC721Balances(t *testing.T) {
	storage := multistandardbalance.NewStorageMemory()
	ctx := context.Background()

	key := multistandardbalance.BalancesKey{
		Account: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		ChainID: 1,
	}

	balances := map[multistandardbalance.ContractAddress]*big.Int{
		common.HexToAddress("0x1111111111111111111111111111111111111111"): big.NewInt(5),
		common.HexToAddress("0x2222222222222222222222222222222222222222"): big.NewInt(3),
	}

	state := multistandardbalance.State{
		AtBlockNumber: big.NewInt(12345),
		AtBlockHash:   common.HexToHash("0xabcdef"),
		FetchedAt:     time.Now().Unix(),
	}

	// Test first update (initial update always reports a balance change)
	balanceChanged, oldState, err := storage.UpdateERC721Balances(ctx, key, balances, state)
	require.NoError(t, err)
	assert.True(t, balanceChanged)
	assert.Equal(t, multistandardbalance.NeverFetched, oldState.FetchedAt)

	// Test update with same balances
	balanceChanged, oldState, err = storage.UpdateERC721Balances(ctx, key, balances, state)
	require.NoError(t, err)
	assert.False(t, balanceChanged)
}

func TestStorageMemory_GetERC721Balances(t *testing.T) {
	storage := multistandardbalance.NewStorageMemory()
	ctx := context.Background()

	key := multistandardbalance.BalancesKey{
		Account: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		ChainID: 1,
	}

	// Test getting non-existent balances
	balances, state, err := storage.GetERC721Balances(ctx, key)
	require.NoError(t, err)
	assert.Nil(t, balances)
	assert.Equal(t, multistandardbalance.NeverFetched, state.FetchedAt)

	// Test getting existing balances
	expectedBalances := map[multistandardbalance.ContractAddress]*big.Int{
		common.HexToAddress("0x1111111111111111111111111111111111111111"): big.NewInt(5),
		common.HexToAddress("0x2222222222222222222222222222222222222222"): big.NewInt(3),
	}

	expectedState := multistandardbalance.State{
		AtBlockNumber: big.NewInt(12345),
		AtBlockHash:   common.HexToHash("0xabcdef"),
		FetchedAt:     time.Now().Unix(),
	}

	_, _, err = storage.UpdateERC721Balances(ctx, key, expectedBalances, expectedState)
	require.NoError(t, err)

	balances, state, err = storage.GetERC721Balances(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, len(expectedBalances), len(balances))
	for contract, expectedBalance := range expectedBalances {
		actualBalance, exists := balances[contract]
		assert.True(t, exists)
		assert.Equal(t, expectedBalance, actualBalance)
	}
}

func TestStorageMemory_UpdateERC1155Balances(t *testing.T) {
	storage := multistandardbalance.NewStorageMemory()
	ctx := context.Background()

	key := multistandardbalance.BalancesKey{
		Account: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		ChainID: 1,
	}

	balances := map[multistandardbalance.HashableCollectibleID]*big.Int{
		multistandardbalance.HashableCollectibleID{
			ContractAddress: common.HexToAddress("0x1111111111111111111111111111111111111111"),
			TokenID:         [32]byte{1},
		}: big.NewInt(10),
		multistandardbalance.HashableCollectibleID{
			ContractAddress: common.HexToAddress("0x2222222222222222222222222222222222222222"),
			TokenID:         [32]byte{2},
		}: big.NewInt(20),
	}

	state := multistandardbalance.State{
		AtBlockNumber: big.NewInt(12345),
		AtBlockHash:   common.HexToHash("0xabcdef"),
		FetchedAt:     time.Now().Unix(),
	}

	// Test first update (initial update always reports a balance change)
	balanceChanged, oldState, err := storage.UpdateERC1155Balances(ctx, key, balances, state)
	require.NoError(t, err)
	assert.True(t, balanceChanged)
	assert.Equal(t, multistandardbalance.NeverFetched, oldState.FetchedAt)

	// Test update with same balances
	balanceChanged, oldState, err = storage.UpdateERC1155Balances(ctx, key, balances, state)
	require.NoError(t, err)
	assert.False(t, balanceChanged)
}

func TestStorageMemory_GetERC1155Balances(t *testing.T) {
	storage := multistandardbalance.NewStorageMemory()
	ctx := context.Background()

	key := multistandardbalance.BalancesKey{
		Account: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		ChainID: 1,
	}

	// Test getting non-existent balances
	balances, state, err := storage.GetERC1155Balances(ctx, key)
	require.NoError(t, err)
	assert.Nil(t, balances)
	assert.Equal(t, multistandardbalance.NeverFetched, state.FetchedAt)

	// Test getting existing balances
	expectedBalances := map[multistandardbalance.HashableCollectibleID]*big.Int{
		multistandardbalance.HashableCollectibleID{
			ContractAddress: common.HexToAddress("0x1111111111111111111111111111111111111111"),
			TokenID:         [32]byte{1},
		}: big.NewInt(10),
		multistandardbalance.HashableCollectibleID{
			ContractAddress: common.HexToAddress("0x2222222222222222222222222222222222222222"),
			TokenID:         [32]byte{2},
		}: big.NewInt(20),
	}

	expectedState := multistandardbalance.State{
		AtBlockNumber: big.NewInt(12345),
		AtBlockHash:   common.HexToHash("0xabcdef"),
		FetchedAt:     time.Now().Unix(),
	}

	_, _, err = storage.UpdateERC1155Balances(ctx, key, expectedBalances, expectedState)
	require.NoError(t, err)

	balances, state, err = storage.GetERC1155Balances(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, len(expectedBalances), len(balances))
	for collectibleID, expectedBalance := range expectedBalances {
		actualBalance, exists := balances[collectibleID]
		assert.True(t, exists)
		assert.Equal(t, expectedBalance, actualBalance)
	}
}

func TestStorageMemory_ClearMissingAccounts(t *testing.T) {
	storage := multistandardbalance.NewStorageMemory()
	ctx := context.Background()

	// Add some test data
	account1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	account2 := common.HexToAddress("0x2222222222222222222222222222222222222222")
	account3 := common.HexToAddress("0x3333333333333333333333333333333333333333")

	key1 := multistandardbalance.BalancesKey{Account: account1, ChainID: 1}
	key2 := multistandardbalance.BalancesKey{Account: account2, ChainID: 1}
	key3 := multistandardbalance.BalancesKey{Account: account3, ChainID: 1}

	state := multistandardbalance.State{
		AtBlockNumber: big.NewInt(12345),
		AtBlockHash:   common.HexToHash("0xabcdef"),
		FetchedAt:     time.Now().Unix(),
	}

	_, _, err := storage.UpdateNativeBalance(ctx, key1, big.NewInt(100), state)
	require.NoError(t, err)
	_, _, err = storage.UpdateNativeBalance(ctx, key2, big.NewInt(200), state)
	require.NoError(t, err)
	_, _, err = storage.UpdateNativeBalance(ctx, key3, big.NewInt(300), state)
	require.NoError(t, err)

	// Clear accounts not in the list (keep only account1 and account2)
	accountsToKeep := []common.Address{account1, account2}
	storage.ClearMissingAccounts(ctx, accountsToKeep)

	// Check that account1 and account2 still exist
	balance1, _, err := storage.GetNativeBalance(ctx, key1)
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(100), balance1)

	balance2, _, err := storage.GetNativeBalance(ctx, key2)
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(200), balance2)

	// Check that account3 was removed
	balance3, state3, err := storage.GetNativeBalance(ctx, key3)
	require.NoError(t, err)
	assert.Nil(t, balance3)
	assert.Equal(t, multistandardbalance.NeverFetched, state3.FetchedAt)
}

func TestStorageMemory_ClearMissingChains(t *testing.T) {
	storage := multistandardbalance.NewStorageMemory()
	ctx := context.Background()

	// Add some test data
	account := common.HexToAddress("0x1111111111111111111111111111111111111111")

	key1 := multistandardbalance.BalancesKey{Account: account, ChainID: 1}
	key2 := multistandardbalance.BalancesKey{Account: account, ChainID: 2}
	key3 := multistandardbalance.BalancesKey{Account: account, ChainID: 3}

	state := multistandardbalance.State{
		AtBlockNumber: big.NewInt(12345),
		AtBlockHash:   common.HexToHash("0xabcdef"),
		FetchedAt:     time.Now().Unix(),
	}

	_, _, err := storage.UpdateNativeBalance(ctx, key1, big.NewInt(100), state)
	require.NoError(t, err)
	_, _, err = storage.UpdateNativeBalance(ctx, key2, big.NewInt(200), state)
	require.NoError(t, err)
	_, _, err = storage.UpdateNativeBalance(ctx, key3, big.NewInt(300), state)
	require.NoError(t, err)

	// Clear chains not in the list (keep only chain 1 and 2)
	chainsToKeep := []uint64{1, 2}
	storage.ClearMissingChains(ctx, chainsToKeep)

	// Check that chain 1 and 2 still exist
	balance1, _, err := storage.GetNativeBalance(ctx, key1)
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(100), balance1)

	balance2, _, err := storage.GetNativeBalance(ctx, key2)
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(200), balance2)

	// Check that chain 3 was removed
	balance3, state3, err := storage.GetNativeBalance(ctx, key3)
	require.NoError(t, err)
	assert.Nil(t, balance3)
	assert.Equal(t, multistandardbalance.NeverFetched, state3.FetchedAt)
}

func TestStorageMemory_UpdateNativeBalance_WithNilBlockNumber(t *testing.T) {
	storage := multistandardbalance.NewStorageMemory()
	ctx := context.Background()

	key := multistandardbalance.BalancesKey{
		Account: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		ChainID: 1,
	}

	// First, create an initial entry with a valid state
	initialBalance := big.NewInt(1000)
	initialState := multistandardbalance.State{
		AtBlockNumber: big.NewInt(12345),
		AtBlockHash:   common.HexToHash("0xabcdef"),
		FetchedAt:     time.Now().Unix(),
	}

	_, _, err := storage.UpdateNativeBalance(ctx, key, initialBalance, initialState)
	require.NoError(t, err)

	// Now try to update with a nil block number - this should cause an error
	newBalance := big.NewInt(2000)
	stateWithNilBlockNumber := multistandardbalance.State{
		AtBlockNumber: nil, // This should cause an error
		AtBlockHash:   common.HexToHash("0xfedcba"),
		FetchedAt:     time.Now().Unix(),
	}

	balanceChanged, oldState, err := storage.UpdateNativeBalance(ctx, key, newBalance, stateWithNilBlockNumber)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "new state at block number is nil")
	assert.False(t, balanceChanged)
	assert.Equal(t, initialState.AtBlockNumber, oldState.AtBlockNumber)

	// Verify that the original balance is still there (not updated)
	retrievedBalance, retrievedState, err := storage.GetNativeBalance(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, initialBalance, retrievedBalance)
	assert.Equal(t, initialState.AtBlockNumber, retrievedState.AtBlockNumber)
}

func TestStorageMemory_UpdateERC20Balances_WithNilBlockNumber(t *testing.T) {
	storage := multistandardbalance.NewStorageMemory()
	ctx := context.Background()

	key := multistandardbalance.BalancesKey{
		Account: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		ChainID: 1,
	}

	// First, create an initial entry with a valid state
	initialBalances := map[multistandardbalance.ContractAddress]*big.Int{
		common.HexToAddress("0x1111111111111111111111111111111111111111"): big.NewInt(100),
	}
	initialState := multistandardbalance.State{
		AtBlockNumber: big.NewInt(12345),
		AtBlockHash:   common.HexToHash("0xabcdef"),
		FetchedAt:     time.Now().Unix(),
	}

	_, _, err := storage.UpdateERC20Balances(ctx, key, initialBalances, initialState)
	require.NoError(t, err)

	// Now try to update with a nil block number - this should cause an error
	newBalances := map[multistandardbalance.ContractAddress]*big.Int{
		common.HexToAddress("0x1111111111111111111111111111111111111111"): big.NewInt(200),
	}
	stateWithNilBlockNumber := multistandardbalance.State{
		AtBlockNumber: nil, // This should cause an error
		AtBlockHash:   common.HexToHash("0xfedcba"),
		FetchedAt:     time.Now().Unix(),
	}

	balanceChanged, oldState, err := storage.UpdateERC20Balances(ctx, key, newBalances, stateWithNilBlockNumber)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "new state at block number is nil")
	assert.False(t, balanceChanged)
	assert.Equal(t, initialState.AtBlockNumber, oldState.AtBlockNumber)

	// Verify that the original balances are still there (not updated)
	retrievedBalances, retrievedState, err := storage.GetERC20Balances(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, len(initialBalances), len(retrievedBalances))
	for contract, expectedBalance := range initialBalances {
		actualBalance, exists := retrievedBalances[contract]
		assert.True(t, exists)
		assert.Equal(t, expectedBalance, actualBalance)
	}
	assert.Equal(t, initialState.AtBlockNumber, retrievedState.AtBlockNumber)
}

func TestStorageMemory_UpdateERC721Balances_WithNilBlockNumber(t *testing.T) {
	storage := multistandardbalance.NewStorageMemory()
	ctx := context.Background()

	key := multistandardbalance.BalancesKey{
		Account: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		ChainID: 1,
	}

	// First, create an initial entry with a valid state
	initialBalances := map[multistandardbalance.ContractAddress]*big.Int{
		common.HexToAddress("0x1111111111111111111111111111111111111111"): big.NewInt(5),
	}
	initialState := multistandardbalance.State{
		AtBlockNumber: big.NewInt(12345),
		AtBlockHash:   common.HexToHash("0xabcdef"),
		FetchedAt:     time.Now().Unix(),
	}

	_, _, err := storage.UpdateERC721Balances(ctx, key, initialBalances, initialState)
	require.NoError(t, err)

	// Now try to update with a nil block number - this should cause an error
	newBalances := map[multistandardbalance.ContractAddress]*big.Int{
		common.HexToAddress("0x1111111111111111111111111111111111111111"): big.NewInt(10),
	}
	stateWithNilBlockNumber := multistandardbalance.State{
		AtBlockNumber: nil, // This should cause an error
		AtBlockHash:   common.HexToHash("0xfedcba"),
		FetchedAt:     time.Now().Unix(),
	}

	balanceChanged, oldState, err := storage.UpdateERC721Balances(ctx, key, newBalances, stateWithNilBlockNumber)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "new state at block number is nil")
	assert.False(t, balanceChanged)
	assert.Equal(t, initialState.AtBlockNumber, oldState.AtBlockNumber)

	// Verify that the original balances are still there (not updated)
	retrievedBalances, retrievedState, err := storage.GetERC721Balances(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, len(initialBalances), len(retrievedBalances))
	for contract, expectedBalance := range initialBalances {
		actualBalance, exists := retrievedBalances[contract]
		assert.True(t, exists)
		assert.Equal(t, expectedBalance, actualBalance)
	}
	assert.Equal(t, initialState.AtBlockNumber, retrievedState.AtBlockNumber)
}

func TestStorageMemory_UpdateERC1155Balances_WithNilBlockNumber(t *testing.T) {
	storage := multistandardbalance.NewStorageMemory()
	ctx := context.Background()

	key := multistandardbalance.BalancesKey{
		Account: common.HexToAddress("0x1234567890123456789012345678901234567890"),
		ChainID: 1,
	}

	// First, create an initial entry with a valid state
	initialBalances := map[multistandardbalance.HashableCollectibleID]*big.Int{
		multistandardbalance.HashableCollectibleID{
			ContractAddress: common.HexToAddress("0x1111111111111111111111111111111111111111"),
			TokenID:         [32]byte{1},
		}: big.NewInt(10),
	}
	initialState := multistandardbalance.State{
		AtBlockNumber: big.NewInt(12345),
		AtBlockHash:   common.HexToHash("0xabcdef"),
		FetchedAt:     time.Now().Unix(),
	}

	_, _, err := storage.UpdateERC1155Balances(ctx, key, initialBalances, initialState)
	require.NoError(t, err)

	// Now try to update with a nil block number - this should cause an error
	newBalances := map[multistandardbalance.HashableCollectibleID]*big.Int{
		multistandardbalance.HashableCollectibleID{
			ContractAddress: common.HexToAddress("0x1111111111111111111111111111111111111111"),
			TokenID:         [32]byte{1},
		}: big.NewInt(20),
	}
	stateWithNilBlockNumber := multistandardbalance.State{
		AtBlockNumber: nil, // This should cause an error
		AtBlockHash:   common.HexToHash("0xfedcba"),
		FetchedAt:     time.Now().Unix(),
	}

	balanceChanged, oldState, err := storage.UpdateERC1155Balances(ctx, key, newBalances, stateWithNilBlockNumber)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "new state at block number is nil")
	assert.False(t, balanceChanged)
	assert.Equal(t, initialState.AtBlockNumber, oldState.AtBlockNumber)

	// Verify that the original balances are still there (not updated)
	retrievedBalances, retrievedState, err := storage.GetERC1155Balances(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, len(initialBalances), len(retrievedBalances))
	for collectibleID, expectedBalance := range initialBalances {
		actualBalance, exists := retrievedBalances[collectibleID]
		assert.True(t, exists)
		assert.Equal(t, expectedBalance, actualBalance)
	}
	assert.Equal(t, initialState.AtBlockNumber, retrievedState.AtBlockNumber)
}
