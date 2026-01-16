package tokenhistoricalownership

import (
	"database/sql"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/walletdatabase"
	"github.com/status-im/status-go/internal/testutils"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := testutils.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	return db, func() {
		require.NoError(t, db.Close())
	}
}

func TestStorage_MarkAsOwned(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	storage := NewStorage(db)

	address := common.HexToAddress("0x1234567890123456789012345678901234567890")
	tokenAddress := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	chainID := uint64(1)

	// First call should return true (new entry)
	isNew, err := storage.MarkAsOwned(address, chainID, tokenAddress)
	require.NoError(t, err)
	assert.True(t, isNew)

	// Second call should return false (already exists)
	isNew, err = storage.MarkAsOwned(address, chainID, tokenAddress)
	require.NoError(t, err)
	assert.False(t, isNew)

	// Different token should return true
	tokenAddress2 := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	isNew, err = storage.MarkAsOwned(address, chainID, tokenAddress2)
	require.NoError(t, err)
	assert.True(t, isNew)

	// Same token on different chain should return true
	chainID2 := uint64(5)
	isNew, err = storage.MarkAsOwned(address, chainID2, tokenAddress)
	require.NoError(t, err)
	assert.True(t, isNew)
}

func TestStorage_GetOwnedTokens(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	storage := NewStorage(db)

	address := common.HexToAddress("0x1234567890123456789012345678901234567890")
	token1 := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	token2 := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	chainID1 := uint64(1)
	chainID2 := uint64(5)

	// Initially should be empty
	tokens, err := storage.GetOwnedTokens(address)
	require.NoError(t, err)
	assert.Empty(t, tokens)

	// Mark some tokens as owned
	_, err = storage.MarkAsOwned(address, chainID1, token1)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	_, err = storage.MarkAsOwned(address, chainID2, token2)
	require.NoError(t, err)

	// Get all owned tokens
	tokens, err = storage.GetOwnedTokens(address)
	require.NoError(t, err)
	assert.Len(t, tokens, 2)

	// Verify token addresses are in the result
	addresses := []common.Address{tokens[0].TokenAddress, tokens[1].TokenAddress}
	assert.Contains(t, addresses, token1)
	assert.Contains(t, addresses, token2)

	// Verify all fields are set correctly
	for _, token := range tokens {
		assert.Equal(t, address, token.OwnerAddress)
		assert.Contains(t, []uint64{chainID1, chainID2}, token.ChainID)
		assert.Greater(t, token.Timestamp, int64(0))
	}

	// Different address should return empty
	address2 := common.HexToAddress("0x2234567890123456789012345678901234567890")
	tokens, err = storage.GetOwnedTokens(address2)
	require.NoError(t, err)
	assert.Empty(t, tokens)
}

func TestStorage_GetOwnedTokensByChain(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	storage := NewStorage(db)

	address := common.HexToAddress("0x1234567890123456789012345678901234567890")
	token1 := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	token2 := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	token3 := common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F")
	chainID1 := uint64(1)
	chainID2 := uint64(5)

	// Mark tokens as owned on different chains
	_, err := storage.MarkAsOwned(address, chainID1, token1)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	_, err = storage.MarkAsOwned(address, chainID1, token2)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	_, err = storage.MarkAsOwned(address, chainID2, token3)
	require.NoError(t, err)

	// Get tokens for chain 1
	tokens, err := storage.GetOwnedTokensByChain(address, chainID1)
	require.NoError(t, err)
	assert.Len(t, tokens, 2)

	// Verify token addresses
	addresses := []common.Address{tokens[0].TokenAddress, tokens[1].TokenAddress}
	assert.Contains(t, addresses, token1)
	assert.Contains(t, addresses, token2)

	// Verify all are on chain 1
	for _, token := range tokens {
		assert.Equal(t, chainID1, token.ChainID)
		assert.Equal(t, address, token.OwnerAddress)
	}

	// Get tokens for chain 2
	tokens, err = storage.GetOwnedTokensByChain(address, chainID2)
	require.NoError(t, err)
	assert.Len(t, tokens, 1)
	assert.Equal(t, token3, tokens[0].TokenAddress)
	assert.Equal(t, chainID2, tokens[0].ChainID)
	assert.Equal(t, address, tokens[0].OwnerAddress)

	// Get tokens for non-existent chain
	tokens, err = storage.GetOwnedTokensByChain(address, uint64(999))
	require.NoError(t, err)
	assert.Empty(t, tokens)
}

func TestStorage_IsOwned(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	storage := NewStorage(db)

	address := common.HexToAddress("0x1234567890123456789012345678901234567890")
	tokenAddress := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	chainID := uint64(1)

	// Initially should not be owned
	isOwned, err := storage.IsOwned(address, chainID, tokenAddress)
	require.NoError(t, err)
	assert.False(t, isOwned)

	// Mark as owned
	_, err = storage.MarkAsOwned(address, chainID, tokenAddress)
	require.NoError(t, err)

	// Now should be owned
	isOwned, err = storage.IsOwned(address, chainID, tokenAddress)
	require.NoError(t, err)
	assert.True(t, isOwned)

	// Different token should not be owned
	tokenAddress2 := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	isOwned, err = storage.IsOwned(address, chainID, tokenAddress2)
	require.NoError(t, err)
	assert.False(t, isOwned)

	// Same token on different chain should not be owned
	chainID2 := uint64(5)
	isOwned, err = storage.IsOwned(address, chainID2, tokenAddress)
	require.NoError(t, err)
	assert.False(t, isOwned)

	// Mark it on different chain
	_, err = storage.MarkAsOwned(address, chainID2, tokenAddress)
	require.NoError(t, err)

	// Now should be owned on both chains
	isOwned, err = storage.IsOwned(address, chainID, tokenAddress)
	require.NoError(t, err)
	assert.True(t, isOwned)

	isOwned, err = storage.IsOwned(address, chainID2, tokenAddress)
	require.NoError(t, err)
	assert.True(t, isOwned)
}

func TestStorage_RemoveOwnerRecords(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	storage := NewStorage(db)

	address1 := common.HexToAddress("0x1234567890123456789012345678901234567890")
	address2 := common.HexToAddress("0x2234567890123456789012345678901234567890")
	tokenAddress := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	chainID1 := uint64(1)
	chainID2 := uint64(5)

	// Mark tokens as owned for both addresses
	_, err := storage.MarkAsOwned(address1, chainID1, tokenAddress)
	require.NoError(t, err)

	_, err = storage.MarkAsOwned(address1, chainID2, tokenAddress)
	require.NoError(t, err)

	_, err = storage.MarkAsOwned(address2, chainID1, tokenAddress)
	require.NoError(t, err)

	// Verify both are owned
	isOwned, err := storage.IsOwned(address1, chainID1, tokenAddress)
	require.NoError(t, err)
	assert.True(t, isOwned)

	isOwned, err = storage.IsOwned(address1, chainID2, tokenAddress)
	require.NoError(t, err)
	assert.True(t, isOwned)

	isOwned, err = storage.IsOwned(address2, chainID1, tokenAddress)
	require.NoError(t, err)
	assert.True(t, isOwned)

	// Remove records for address1
	err = storage.RemoveOwnerRecords(address1)
	require.NoError(t, err)

	// Verify address1 tokens were removed
	isOwned, err = storage.IsOwned(address1, chainID1, tokenAddress)
	require.NoError(t, err)
	assert.False(t, isOwned)

	isOwned, err = storage.IsOwned(address1, chainID2, tokenAddress)
	require.NoError(t, err)
	assert.False(t, isOwned)

	// Verify address2 tokens still exist
	isOwned, err = storage.IsOwned(address2, chainID1, tokenAddress)
	require.NoError(t, err)
	assert.True(t, isOwned)

	// Verify GetOwnedTokens returns empty for address1
	tokens, err := storage.GetOwnedTokens(address1)
	require.NoError(t, err)
	assert.Empty(t, tokens)

	// Verify GetOwnedTokens still returns tokens for address2
	tokens, err = storage.GetOwnedTokens(address2)
	require.NoError(t, err)
	assert.Len(t, tokens, 1)
	assert.Equal(t, tokenAddress, tokens[0].TokenAddress)
}

func TestStorage_GetOwnedTokens_Ordering(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	storage := NewStorage(db)

	address := common.HexToAddress("0x1234567890123456789012345678901234567890")
	token1 := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	token2 := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	token3 := common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F")
	chainID := uint64(1)

	// Mark tokens with delays to ensure different timestamps
	// Use 1 second delays since time.Now().Unix() has second precision
	_, err := storage.MarkAsOwned(address, chainID, token1)
	require.NoError(t, err)

	time.Sleep(1100 * time.Millisecond) // 1.1 seconds to ensure different timestamp

	_, err = storage.MarkAsOwned(address, chainID, token2)
	require.NoError(t, err)

	time.Sleep(1100 * time.Millisecond) // 1.1 seconds to ensure different timestamp

	_, err = storage.MarkAsOwned(address, chainID, token3)
	require.NoError(t, err)

	// Get tokens - should be ordered by timestamp DESC (newest first)
	tokens, err := storage.GetOwnedTokens(address)
	require.NoError(t, err)
	assert.Len(t, tokens, 3)

	// Verify ordering (newest first) - token3 was last, so should be first
	assert.Equal(t, token3, tokens[0].TokenAddress)
	assert.Equal(t, token2, tokens[1].TokenAddress)
	assert.Equal(t, token1, tokens[2].TokenAddress)

	// Verify timestamps are in descending order
	assert.GreaterOrEqual(t, tokens[0].Timestamp, tokens[1].Timestamp)
	assert.GreaterOrEqual(t, tokens[1].Timestamp, tokens[2].Timestamp)
}
