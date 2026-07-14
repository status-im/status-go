package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestStoreNodeRequestPageCap verifies the per-request page ceiling that bounds
// a runaway store-node history fetch (issue #21470-hf). The cap must only ever
// turn a "fetch next page" decision into a stop; it must never force extra
// paging, and it must be a no-op when the natural decision is already to stop
// (e.g. StopWhenDataFound with the data found) or when the cap is disabled.
func TestStoreNodeRequestPageCap(t *testing.T) {
	const cap = 30

	t.Run("disabled cap (MaxPageCount==0) never trips", func(t *testing.T) {
		cfg := StoreNodeRequestConfig{MaxPageCount: 0, FurtherPageSize: 50}
		fetch, size, tripped := cfg.gateNextPageByCap(true, cfg.FurtherPageSize, 1_000_000)
		require.True(t, fetch)
		require.Equal(t, uint64(50), size)
		require.False(t, tripped)
	})

	t.Run("below cap keeps paging", func(t *testing.T) {
		cfg := StoreNodeRequestConfig{MaxPageCount: cap, FurtherPageSize: 50}
		fetch, size, tripped := cfg.gateNextPageByCap(true, cfg.FurtherPageSize, cap-1)
		require.True(t, fetch)
		require.Equal(t, uint64(50), size)
		require.False(t, tripped)
	})

	t.Run("exactly at cap trips and stops", func(t *testing.T) {
		cfg := StoreNodeRequestConfig{MaxPageCount: cap, FurtherPageSize: 50}
		fetch, size, tripped := cfg.gateNextPageByCap(true, cfg.FurtherPageSize, cap)
		require.False(t, fetch)
		require.Equal(t, uint64(0), size)
		require.True(t, tripped)
	})

	t.Run("above cap trips and stops", func(t *testing.T) {
		cfg := StoreNodeRequestConfig{MaxPageCount: cap, FurtherPageSize: 50}
		fetch, size, tripped := cfg.gateNextPageByCap(true, cfg.FurtherPageSize, cap+5)
		require.False(t, fetch)
		require.Equal(t, uint64(0), size)
		require.True(t, tripped)
	})

	t.Run("natural stop is never overridden and never reports a trip", func(t *testing.T) {
		// Models StopWhenDataFound with the community found, or the decode/validate
		// error branches: the uncapped decision is already "stop". Even past the
		// cap, the cap must not claim responsibility for the stop.
		cfg := StoreNodeRequestConfig{MaxPageCount: cap, FurtherPageSize: 50}
		fetch, size, tripped := cfg.gateNextPageByCap(false, 0, cap+100)
		require.False(t, fetch)
		require.Equal(t, uint64(0), size)
		require.False(t, tripped)
	})

	t.Run("default config ships a generous, enabled cap", func(t *testing.T) {
		cfg := defaultStoreNodeRequestConfig()
		require.Greater(t, cfg.MaxPageCount, 1, "cap must be enabled by default")
		require.GreaterOrEqual(t, cfg.MaxPageCount, 20, "cap must be generous vs the 1-2 page normal fetch")
	})
}
