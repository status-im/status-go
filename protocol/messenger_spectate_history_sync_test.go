package protocol

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSpectatedCommunitySyncFrom(t *testing.T) {
	require.Equal(t, 9*24*time.Hour, spectatedCommunityInitialSyncPeriod,
		"spectate window matches the 9-day default sync period")

	const now = uint32(1_700_000_000)
	require.Equal(t, now-uint32(9*24*60*60), spectatedCommunitySyncFrom(now))
}

// Spectate is scoped; joining must keep today's unscoped behavior.
func TestCommunityInitialHistorySync(t *testing.T) {
	scoped, window := communityInitialHistorySync(true /* spectated */)
	require.True(t, scoped, "spectators must use the scoped window")
	require.Equal(t, spectatedCommunityInitialSyncPeriod, window)

	scoped, window = communityInitialHistorySync(false /* joined */)
	require.False(t, scoped, "joining must keep today's unscoped full-period backfill")
	require.Equal(t, time.Duration(0), window)
}

// Fresh topics get a seeded watermark; already-tracked topics must be skipped
// (INSERT-OR-REPLACE would rewind a good watermark); each seeded at most once.
func TestCommunityHistorySeedTopics(t *testing.T) {
	const from = 1_699_913_600 // now - 24h

	refs := []mailserverTopicRef{
		{pubsubTopic: "pubA", contentTopic: "0x01"}, // fresh -> seed
		{pubsubTopic: "pubA", contentTopic: "0x02"}, // already tracked -> skip
		{pubsubTopic: "pubA", contentTopic: "0x01"}, // duplicate of the first -> seed once
	}
	existing := map[string]struct{}{
		mailserverTopicKey("pubA", "0x02"): {},
	}

	seeds := communityHistorySeedTopics(refs, existing, from)

	require.Len(t, seeds, 1, "only the fresh topic is seeded, exactly once")
	require.Equal(t, "pubA", seeds[0].PubsubTopic)
	require.Equal(t, "0x01", seeds[0].ContentTopic)
	require.Equal(t, from, seeds[0].LastRequest)
}

// TestCommunityHistorySeedTopics_AllTracked verifies that when every topic is already
// tracked, nothing is seeded (leaving each topic's existing watermark to drive an
// incremental, already-bounded sync).
func TestCommunityHistorySeedTopics_AllTracked(t *testing.T) {
	refs := []mailserverTopicRef{{pubsubTopic: "pubA", contentTopic: "0x01"}}
	existing := map[string]struct{}{mailserverTopicKey("pubA", "0x01"): {}}

	require.Empty(t, communityHistorySeedTopics(refs, existing, 123))
}

// --- cancel registry -------------------------------------------------------

// fakeCancel records how many times it was invoked, standing in for a
// context.CancelFunc without needing a live context.
type fakeCancel struct {
	calls *int
}

func (f fakeCancel) cancel() { *f.calls++ }

func TestCommunityHistoryFetchRegistry_LeaveAndBackground(t *testing.T) {
	r := newCommunityHistoryFetchRegistry()

	aCalls, bCalls := 0, 0
	a := fakeCancel{&aCalls}
	b := fakeCancel{&bCalls}

	// register two in-flight community fetches
	_ = r.register("communityA", a.cancel)
	_ = r.register("communityB", b.cancel)

	// leaving communityA cancels only its fetch
	r.cancel("communityA")
	require.Equal(t, 1, aCalls)
	require.Equal(t, 0, bCalls)

	// cancelling an already-cancelled / unknown id is a no-op
	r.cancel("communityA")
	r.cancel("unknown")
	require.Equal(t, 1, aCalls)

	// backgrounding cancels everything still in flight
	r.cancelAll()
	require.Equal(t, 1, bCalls)
	require.Equal(t, 1, aCalls, "A was already removed by leave, must not be re-cancelled")
}

// TestCommunityHistoryFetchRegistry_ReplaceAndFinish verifies that re-spectating the
// same community cancels the stale in-flight fetch, and that a fetch which finishes
// on its own only cleans up the map entry it still owns (never a newer one's).
func TestCommunityHistoryFetchRegistry_ReplaceAndFinish(t *testing.T) {
	r := newCommunityHistoryFetchRegistry()

	firstCalls, secondCalls := 0, 0
	first := fakeCancel{&firstCalls}
	second := fakeCancel{&secondCalls}

	tok1 := r.register("communityA", first.cancel)

	// re-spectating communityA replaces (and cancels) the stale fetch
	tok2 := r.register("communityA", second.cancel)
	require.Equal(t, 1, firstCalls, "stale fetch must be cancelled on re-register")
	require.NotEqual(t, tok1, tok2, "each registration gets a distinct token")

	// the FIRST fetch's goroutine now finishes and tries to clean up. It must NOT
	// remove the second fetch's entry (it no longer owns the slot).
	r.finish("communityA", tok1)

	// cancelling communityA must still reach the second (current) fetch
	r.cancel("communityA")
	require.Equal(t, 1, secondCalls, "current fetch must remain cancellable after a stale finish")

	// a fetch that finishes on its own removes its entry, so a later cancel is a no-op
	thirdCalls := 0
	third := fakeCancel{&thirdCalls}
	tok3 := r.register("communityA", third.cancel)
	r.finish("communityA", tok3)
	r.cancel("communityA")
	require.Equal(t, 0, thirdCalls, "finish removed the entry, so cancel must not reach the completed fetch")
}

// TestCommunityHistoryFetchRegistry_ContextCancelFunc is a light integration check
// that a real context.CancelFunc registered in the registry cancels its context.
func TestCommunityHistoryFetchRegistry_ContextCancelFunc(t *testing.T) {
	r := newCommunityHistoryFetchRegistry()
	ctx, cancel := context.WithCancel(context.Background())
	_ = r.register("communityA", cancel)

	require.NoError(t, ctx.Err())
	r.cancel("communityA")
	require.ErrorIs(t, ctx.Err(), context.Canceled)
}
