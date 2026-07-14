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

// TestGateNextPageByValidationQueue verifies the decision that stops store-node
// pagination for a community request once the community's description is already
// in hand and only blocked on owner validation (issue #21470-hf). For a
// token-owned community the store-node pager's natural stop condition is
// "community persisted", but persistence is withheld until on-chain owner
// verification succeeds. A transport failure during verification is
// indistinguishable, to the pager, from "not fetched yet", so it wrongly fetches
// another page — which cannot make a dead RPC succeed and only floods the
// network. Once the description is queued for validation, more pages genuinely
// cannot help: verification retries out-of-band on the owner-verification loop.
func TestGateNextPageByValidationQueue(t *testing.T) {
	t.Run("community already in DB never stops for validation", func(t *testing.T) {
		// The found path (clock check) owns this case; the validation gate must
		// not interfere regardless of what is queued.
		require.False(t, gateNextPageByValidationQueue(true, 100, 0))
	})

	t.Run("nothing queued keeps paging (genuine not-found)", func(t *testing.T) {
		require.False(t, gateNextPageByValidationQueue(false, 0, 0))
	})

	t.Run("queued description newer than what we hold stops paging", func(t *testing.T) {
		// Description is in hand, only owner verification is pending; the
		// verification loop will retry it. More pages cannot help.
		require.True(t, gateNextPageByValidationQueue(false, 100, 0))
		require.True(t, gateNextPageByValidationQueue(false, 101, 100))
	})

	t.Run("queued description not newer than what we hold keeps paging", func(t *testing.T) {
		// The queued description is stale relative to the clock we already have;
		// a newer page could still legitimately arrive. Mirrors the found-path
		// "is this newer" test (community.Clock() <= minimumDataClock).
		require.False(t, gateNextPageByValidationQueue(false, 100, 100))
		require.False(t, gateNextPageByValidationQueue(false, 90, 100))
	})
}
