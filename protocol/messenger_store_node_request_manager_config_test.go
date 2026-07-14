package protocol

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestCommunityFetchBackoff verifies the per-id exponential backoff that throttles
// repeated store-node community fetches which keep finalizing as not-found (issue
// #21470-hf). The dev curated directory carries "dead" community ids whose
// descriptions never arrive; any client-side loop re-issues FetchCommunity for
// them, each run pages to the cap, finalizes not-found, and the next call starts
// a fresh run — a sustained CPU storm.
//
// An earlier hotfix (b7992077f) used a FLAT 2-minute cooldown after the first
// nil finalize. On-device that over-suppressed a COLD start: a capped request
// commonly finalizes nil before the slow, loaded store node has served a
// community's large description, and the flat cooldown then blocked exactly the
// rapid retries that empirically are how descriptions land (346 attempts → 249
// suppressed → 0 descriptions resolved in 9 min). This backoff instead barely
// touches the warm-up cadence (first retries within seconds) while a genuinely
// dead id converges to one run per maxDelay.
//
// backoffFor(n) = min(baseDelay * 2^(n-1), maxDelay). Only a REAL request that
// finalizes nil increments n (recordNilFinalize); a suppressed attempt must not
// self-extend the ban. A success clears the entry entirely.
func TestCommunityFetchBackoff(t *testing.T) {
	const base = 5 * time.Second
	const max = 2 * time.Minute
	baseTime := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)

	t.Run("no entry suppresses nothing", func(t *testing.T) {
		c := newCommunityFetchBackoff(base, max)
		suppress, misses, remaining := c.suppress("0xdead", baseTime)
		require.False(t, suppress)
		require.Equal(t, 0, misses)
		require.Equal(t, time.Duration(0), remaining)
	})

	t.Run("backoff schedule doubles per miss up to the cap", func(t *testing.T) {
		c := newCommunityFetchBackoff(base, max)
		require.Equal(t, time.Duration(0), c.backoffFor(0))
		for _, tc := range []struct {
			misses int
			want   time.Duration
		}{
			{1, 5 * time.Second},
			{2, 10 * time.Second},
			{3, 20 * time.Second},
			{4, 40 * time.Second},
			{5, 80 * time.Second},
			{6, 120 * time.Second},
			{7, 120 * time.Second},
			{20, 120 * time.Second},
		} {
			require.Equalf(t, tc.want, c.backoffFor(tc.misses), "backoffFor(%d)", tc.misses)
		}
	})

	t.Run("warm-up retries are barely delayed; a dead id converges to the cap", func(t *testing.T) {
		for _, tc := range []struct {
			misses int
			window time.Duration
		}{
			{1, 5 * time.Second},
			{2, 10 * time.Second},
			{3, 20 * time.Second},
			{6, 120 * time.Second},
		} {
			c := newCommunityFetchBackoff(base, max)
			for i := 0; i < tc.misses; i++ {
				c.recordNilFinalize("0xdead", baseTime)
			}
			// Just before the window ends: suppressed, one ns of backoff left.
			suppress, misses, remaining := c.suppress("0xdead", baseTime.Add(tc.window-time.Nanosecond))
			require.Truef(t, suppress, "misses=%d", tc.misses)
			require.Equal(t, tc.misses, misses)
			require.Equal(t, time.Nanosecond, remaining)
			// Exactly at the window boundary: allowed to retry.
			suppress, _, remaining = c.suppress("0xdead", baseTime.Add(tc.window))
			require.Falsef(t, suppress, "misses=%d", tc.misses)
			require.Equal(t, time.Duration(0), remaining)
		}
	})

	t.Run("a suppressed attempt does not increment the counter (no self-extending ban)", func(t *testing.T) {
		c := newCommunityFetchBackoff(base, max)
		c.recordNilFinalize("0xdead", baseTime) // n=1, window 5s

		// Hammer suppress within the window; none of these may advance n or slide
		// the window.
		for i := 0; i < 5; i++ {
			suppress, misses, _ := c.suppress("0xdead", baseTime.Add(2*time.Second))
			require.True(t, suppress)
			require.Equal(t, 1, misses)
		}
		// Still n=1, so the retry is allowed exactly at +5s (window never moved).
		suppress, misses, _ := c.suppress("0xdead", baseTime.Add(5*time.Second))
		require.False(t, suppress)
		require.Equal(t, 1, misses)
	})

	t.Run("only a real nil finalize advances the backoff", func(t *testing.T) {
		c := newCommunityFetchBackoff(base, max)
		c.recordNilFinalize("0xdead", baseTime)                    // n=1
		c.recordNilFinalize("0xdead", baseTime.Add(5*time.Second)) // n=2, window 10s from +5s

		suppress, misses, remaining := c.suppress("0xdead", baseTime.Add(6*time.Second))
		require.True(t, suppress)
		require.Equal(t, 2, misses)
		require.Equal(t, 9*time.Second, remaining)
	})

	t.Run("an allowed retry keeps the counter so a dead id keeps climbing", func(t *testing.T) {
		c := newCommunityFetchBackoff(base, max)
		c.recordNilFinalize("0xdead", baseTime)
		c.recordNilFinalize("0xdead", baseTime) // n=2, window 10s

		// Past the window: allowed, but the n=2 entry must persist (not evicted like
		// the old TTL) so the next finalize climbs to n=3, not resets to n=1.
		suppress, misses, _ := c.suppress("0xdead", baseTime.Add(30*time.Second))
		require.False(t, suppress)
		require.Equal(t, 2, misses)

		c.recordNilFinalize("0xdead", baseTime.Add(30*time.Second)) // -> n=3, window 20s
		suppress, misses, remaining := c.suppress("0xdead", baseTime.Add(31*time.Second))
		require.True(t, suppress)
		require.Equal(t, 3, misses)
		require.Equal(t, 19*time.Second, remaining)
	})

	t.Run("a different id is independent", func(t *testing.T) {
		c := newCommunityFetchBackoff(base, max)
		c.recordNilFinalize("0xdead", baseTime)

		suppress, misses, remaining := c.suppress("0xlive", baseTime.Add(time.Second))
		require.False(t, suppress)
		require.Equal(t, 0, misses)
		require.Equal(t, time.Duration(0), remaining)
	})

	t.Run("a successful fetch clears the entry and resets the schedule", func(t *testing.T) {
		c := newCommunityFetchBackoff(base, max)
		c.recordNilFinalize("0xdead", baseTime)
		c.recordNilFinalize("0xdead", baseTime) // n=2
		c.clear("0xdead")

		suppress, misses, remaining := c.suppress("0xdead", baseTime.Add(time.Second))
		require.False(t, suppress)
		require.Equal(t, 0, misses)
		require.Equal(t, time.Duration(0), remaining)

		// After a clear, a fresh nil finalize starts back at n=1 (5s), not where the
		// prior streak left off.
		c.recordNilFinalize("0xdead", baseTime.Add(time.Minute))
		suppress, misses, remaining = c.suppress("0xdead", baseTime.Add(time.Minute+time.Second))
		require.True(t, suppress)
		require.Equal(t, 1, misses)
		require.Equal(t, 4*time.Second, remaining)
	})

	t.Run("default base/max delays keep warm-up fast and cap steady state", func(t *testing.T) {
		require.Equal(t, 5*time.Second, communityFetchBackoffBaseDelay)
		require.Equal(t, 2*time.Minute, communityFetchBackoffMaxDelay)
	})
}

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

// TestGateNextPageByDescriptionSeen verifies the decision that stops store-node
// pagination for a community request once a description for the target community
// has already been processed in this request (issue #21470-hf). Store-node
// history is paged newest-first, so the first description encountered is the
// newest available and every later page can only carry an older one. Continuing
// to page therefore cannot yield a newer description; it only re-unmarshals and
// re-preprocesses the (large) description on each page, driving a GC storm and
// overheating the device. This is the remaining flood after the page cap and
// validation-queue gates: the "community persisted but not newer" branch
// (messenger_store_node_request_manager.go, community.Clock() <= minimumDataClock)
// which today keeps paging to the cap and is re-issued perpetually.
func TestGateNextPageByDescriptionSeen(t *testing.T) {
	t.Run("description not seen keeps the natural decision", func(t *testing.T) {
		// No description for this community has been processed yet, so paging must
		// continue up to the cap exactly as before.
		cfg := StoreNodeRequestConfig{StopWhenDataFound: true}
		fetch, tripped := cfg.gateNextPageByDescriptionSeen(true, false)
		require.True(t, fetch)
		require.False(t, tripped)
	})

	t.Run("description seen stops paging and reports the trip", func(t *testing.T) {
		// The core #21470-hf bug: a spectated community is re-fetched,
		// minimumDataClock == its clock, so every page hits the "not newer" branch
		// and would keep paging. Once the description was processed this request,
		// older pages cannot beat it, so we stop and report the trip so the caller
		// can log it distinctly.
		cfg := StoreNodeRequestConfig{StopWhenDataFound: true}
		fetch, tripped := cfg.gateNextPageByDescriptionSeen(true, true)
		require.False(t, fetch)
		require.True(t, tripped)
	})

	t.Run("natural stop is never overridden and never reports a trip", func(t *testing.T) {
		// The uncapped decision is already "stop" (e.g. found-and-newer, or a
		// decode/validate error). Even with the description seen, this gate must
		// not claim responsibility for the stop.
		cfg := StoreNodeRequestConfig{StopWhenDataFound: true}
		fetch, tripped := cfg.gateNextPageByDescriptionSeen(false, true)
		require.False(t, fetch)
		require.False(t, tripped)
	})

	t.Run("full-window walk (StopWhenDataFound=false) is never cut short", func(t *testing.T) {
		// Mirrors the success-path `!StopWhenDataFound` return: a caller that
		// wants the entire window must keep paging even after the description is
		// seen. The gate is disabled and returns the natural decision unchanged.
		cfg := StoreNodeRequestConfig{StopWhenDataFound: false}
		fetch, tripped := cfg.gateNextPageByDescriptionSeen(true, true)
		require.True(t, fetch)
		require.False(t, tripped)
	})
}

// TestCommunityDescriptionSeen verifies the request-scoped signal that feeds the
// description-seen gate: whether the messages processed for a store-node page
// carried a description for the target community. A processed description
// surfaces the community in the MessengerResponse (handleCommunityResponse ->
// AddCommunity), including the equal-clock re-processing that drives issue
// #21470-hf, so its presence in the response is the "a description for this
// community was processed this page" signal.
func TestCommunityDescriptionSeen(t *testing.T) {
	community := createTestCommunityWithImage(t)

	t.Run("nil response is not seen", func(t *testing.T) {
		require.False(t, communityDescriptionSeen(nil, community.ID()))
	})

	t.Run("empty response is not seen", func(t *testing.T) {
		require.False(t, communityDescriptionSeen(&MessengerResponse{}, community.ID()))
	})

	t.Run("response carrying the community is seen", func(t *testing.T) {
		response := &MessengerResponse{}
		response.AddCommunity(community)
		require.True(t, communityDescriptionSeen(response, community.ID()))
	})

	t.Run("response carrying a different community is not seen", func(t *testing.T) {
		other := createTestCommunityWithImage(t)
		response := &MessengerResponse{}
		response.AddCommunity(other)
		require.False(t, communityDescriptionSeen(response, community.ID()))
	})
}
