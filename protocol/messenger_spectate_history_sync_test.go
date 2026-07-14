package protocol

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestSpectatedCommunitySyncFrom verifies the spectate backfill window computation
// (issue #21470-hf). A keyless spectator's full default-period backfill of a
// community's single universal content topic ingests gigabytes of undecryptable
// payloads (measured ~2-3GB/11min at ~330% service CPU on a Samsung S21FE). The
// only byte-cutting lever is the WINDOW, because every channel rides one content
// topic so per-channel scoping is impossible. Spectate is therefore bounded to a
// scoped window matching the app's default sync period (9 days).
func TestSpectatedCommunitySyncFrom(t *testing.T) {
	require.Equal(t, 9*24*time.Hour, spectatedCommunityInitialSyncPeriod,
		"spectate window matches the 9-day default sync period — affordable since hash-first + keyless-skip")

	const now = uint32(1_700_000_000)
	require.Equal(t, now-uint32(9*24*60*60), spectatedCommunitySyncFrom(now))
}

// TestCommunityInitialHistorySync verifies the spectate-vs-join scoping decision
// (issue #21470-hf). A spectator holds no decryption keys, so deeper history is
// pure heat and is scoped to 24h. A joiner holds keys and legitimately wants the
// full default-period backfill, so joining must KEEP today's behavior (unscoped).
func TestCommunityInitialHistorySync(t *testing.T) {
	scoped, window := communityInitialHistorySync(true /* spectated */)
	require.True(t, scoped, "spectators must use the scoped window")
	require.Equal(t, spectatedCommunityInitialSyncPeriod, window)

	scoped, window = communityInitialHistorySync(false /* joined */)
	require.False(t, scoped, "joining must keep today's unscoped full-period backfill")
	require.Equal(t, time.Duration(0), window)
}

// TestSpectateShouldSkipKeylessBodies verifies the keyless body-skip gate (issue
// #21470-hf enhancement). Bodies may be skipped ONLY when the node holds no keys AND the
// description is already resolved AND every channel is key-gated. In particular a MIXED
// community (fullyEncrypted=false: an encrypted community with public channels, like
// Status) must keep full body fetch, or the public channels' readable history is dropped.
// All three inputs fail safe toward fetching.
func TestSpectateShouldSkipKeylessBodies(t *testing.T) {
	// the ONLY skip case: keyless + resolved + fully encrypted
	require.True(t, spectateShouldSkipKeylessBodies(false /* no keys */, true /* resolved */, true /* fully encrypted */),
		"keyless spectator, resolved description, all channels gated -> skip")

	// mixed community: a readable public channel exists -> must fetch
	require.False(t, spectateShouldSkipKeylessBodies(false, true, false /* NOT fully encrypted */),
		"mixed community (public channel present) must fetch bodies")

	require.False(t, spectateShouldSkipKeylessBodies(true /* holds keys */, true, true),
		"a keyed (joined) user must fetch bodies")
	require.False(t, spectateShouldSkipKeylessBodies(false, false /* description not resolved */, true),
		"an unresolved description must still be fetched")
	require.False(t, spectateShouldSkipKeylessBodies(true, false, false),
		"keyed, unresolved, mixed: fetch")
}

// TestAllChannelsEncrypted verifies the "every channel is key-gated" derivation (issue
// #21470-hf enhancement): if ANY channel is readable without keys the community is not
// fully encrypted, and an empty channel set fails safe (false -> fetch).
func TestAllChannelsEncrypted(t *testing.T) {
	// encrypted set: only "gated" channels are key-gated
	gated := map[string]bool{"gated1": true, "gated2": true, "public": false}
	enc := func(id string) bool { return gated[id] }

	require.True(t, allChannelsEncrypted([]string{"gated1", "gated2"}, enc),
		"all channels gated -> fully encrypted")
	require.False(t, allChannelsEncrypted([]string{"gated1", "public"}, enc),
		"a readable public channel -> not fully encrypted (mixed)")
	require.False(t, allChannelsEncrypted(nil, enc),
		"no known channels must fail safe toward fetching")
	require.False(t, allChannelsEncrypted([]string{}, enc),
		"empty channel set must fail safe toward fetching")
}

// TestCommunityHistorySeedTopics verifies the watermark seeding that bounds a
// spectator's FIRST backfill to the scoped window (issue #21470-hf). syncFiltersFrom
// ignores its `lastRequest` argument for topics not yet tracked — a fresh topic
// always defaults to the full sync period. So before syncing we seed each of the
// community's topics with a `now-24h` watermark. Topics already tracked must be
// SKIPPED (INSERT-OR-REPLACE would otherwise rewind a good watermark and refetch),
// and each topic is seeded at most once.
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

// TestCommunityHistoryFetchRegistry_LeaveAndBackground verifies the cancel-registry
// semantics that make the spectate backfill cancellable (issue #21470-hf, part B):
// the device-measured backfill continues headless after the UI dies, so leaving a
// community and backgrounding the app must both stop it.
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
