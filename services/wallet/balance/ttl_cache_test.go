package balance

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
)

func Test_ttlCacheAll(t *testing.T) {
	const ttl = 10 * time.Millisecond
	cache := newCacheWithTTL(ttl)

	// init args
	block := big.NewInt(1)
	chainID := uint64(1)
	account := common.Address{1}
	balance := big.NewInt(1)

	// Test balance
	cache.AddBalance(account, chainID, block, balance)
	require.Equal(t, 1, cache.BalanceSize(account, chainID))
	require.Equal(t, 0, cache.NonceSize(account, chainID))

	balRes := cache.GetBalance(account, chainID, block)
	require.Equal(t, balance, balRes)

	cache.Clear()
	require.Equal(t, 0, cache.BalanceSize(account, chainID))
	require.Equal(t, 0, cache.NonceSize(account, chainID))

	// Test nonce
	nonce := int64(2)
	cache.AddNonce(account, chainID, block, &nonce)
	require.Equal(t, 1, cache.NonceSize(account, chainID))
	require.Equal(t, 0, cache.BalanceSize(account, chainID))

	nonceRes := cache.GetNonce(account, chainID, block)
	require.Equal(t, nonce, *nonceRes)
	cache.Clear()
	require.Equal(t, 0, cache.BalanceSize(account, chainID))
	require.Equal(t, 0, cache.NonceSize(account, chainID))

	// Test cache expiration
	cache.Clear()
	cache.AddBalance(account, chainID, block, balance)
	cache.AddNonce(account, chainID, block, &nonce)
	time.Sleep(ttl * 2) // wait for cache to expire
	require.Equal(t, 0, cache.BalanceSize(account, chainID))
	require.Equal(t, 0, cache.NonceSize(account, chainID))
	require.Equal(t, 1, cache.nonceRangeCache.size(account, chainID)) // not updated by ttlCache for now
	cache.Clear()

	// Test nonceRange size after adding nonce
	cache.Clear()
	cache.AddNonce(account, chainID, block, &nonce)
	require.Equal(t, 1, cache.nonceRangeCache.size(account, chainID))
	require.Equal(t, 1, len(cache.nonceRangeCache.sortedRanges))

	// Test nonceRange size after clearing
	cache.nonceRangeCache.clear()
	require.Equal(t, 0, cache.nonceRangeCache.size(account, chainID))
	require.Equal(t, 0, len(cache.nonceRangeCache.sortedRanges))
}
