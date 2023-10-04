package balance

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
)

func Test_simpleCacheAll(t *testing.T) {
	cache := newSimpleCache()

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

	// Test nonce
	cache = newSimpleCache()
	nonce := int64(2)
	cache.AddNonce(account, chainID, block, &nonce)
	require.Equal(t, 1, cache.NonceSize(account, chainID))
	require.Equal(t, 0, cache.BalanceSize(account, chainID))

	nonceRes := cache.GetNonce(account, chainID, block)
	require.Equal(t, nonce, *nonceRes)

	// Test nonceRange size after adding nonce
	cache = newSimpleCache()
	cache.AddNonce(account, chainID, block, &nonce)
	require.Equal(t, 1, cache.nonceRangeCache.size(account, chainID))
	require.Equal(t, 1, len(cache.nonceRangeCache.sortedRanges))
}
