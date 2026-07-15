package wakuv2

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	wpb "github.com/waku-org/go-waku/waku/v2/protocol/pb"

	types "github.com/status-im/status-go/pkg/messaging/waku/types"
)

// TestWaitForIngestCapacity covers the pure flow-control decision that backpressures the
// hash-first body fetch against the ingest backlog (issue #21470-hf): it does not pause
// when disabled or below the high-water mark, pauses until the backlog drains, and honours
// context cancellation while paused.
func TestWaitForIngestCapacity(t *testing.T) {
	const poll = time.Millisecond

	t.Run("disabled high-water never pauses", func(t *testing.T) {
		paused, err := waitForIngestCapacity(context.Background(), func() int { return 10000 }, 0, poll)
		require.NoError(t, err)
		require.False(t, paused)
	})

	t.Run("below high-water does not pause", func(t *testing.T) {
		paused, err := waitForIngestCapacity(context.Background(), func() int { return 10 }, 100, poll)
		require.NoError(t, err)
		require.False(t, paused)
	})

	t.Run("pauses until the backlog drains", func(t *testing.T) {
		depth := 145
		calls := 0
		depthFn := func() int {
			calls++
			d := depth
			depth -= 20 // consumer drains ~20 per poll
			return d
		}
		paused, err := waitForIngestCapacity(context.Background(), depthFn, 100, poll)
		require.NoError(t, err)
		require.True(t, paused)
		// Initial over-threshold check + at least one re-check after a poll.
		require.GreaterOrEqual(t, calls, 2)
	})

	t.Run("respects context cancellation while paused", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		paused, err := waitForIngestCapacity(ctx, func() int { return 10000 }, 100, poll)
		require.ErrorIs(t, err, context.Canceled)
		require.True(t, paused)
	})
}

// hashN builds a distinct, deterministic message hash from n.
func hashN(n byte) wpb.MessageHash {
	var h wpb.MessageHash
	h[0] = n
	return h
}

func hashesN(ns ...byte) []wpb.MessageHash {
	out := make([]wpb.MessageHash, 0, len(ns))
	for _, n := range ns {
		out = append(out, hashN(n))
	}
	return out
}

// TestPartitionMessageHashes covers the body-fetch batching limit: unknown hashes must
// be split into store queries of at most maxPerRequest keys (issue #21470-hf). The
// verifier caps this at maxMsgHashesPerRequest; exceeding it would have the store node
// reject or truncate the query.
func TestPartitionMessageHashes(t *testing.T) {
	require.Equal(t, hashFirstMaxHashesPerRequest, 50, "must mirror the verifier's per-request cap")

	// empty -> no batches (nothing to fetch)
	require.Empty(t, partitionMessageHashes(nil, 50))
	require.Empty(t, partitionMessageHashes([]wpb.MessageHash{}, 50))

	// fewer than the cap -> a single batch
	got := partitionMessageHashes(hashesN(1, 2, 3), 50)
	require.Len(t, got, 1)
	require.Equal(t, hashesN(1, 2, 3), got[0])

	// exactly the cap -> a single full batch
	got = partitionMessageHashes(hashesN(1, 2, 3, 4), 4)
	require.Len(t, got, 1)
	require.Len(t, got[0], 4)

	// one over the cap -> two batches, remainder in the second
	got = partitionMessageHashes(hashesN(1, 2, 3, 4, 5), 4)
	require.Len(t, got, 2)
	require.Equal(t, hashesN(1, 2, 3, 4), got[0])
	require.Equal(t, hashesN(5), got[1])

	// several full batches plus a remainder
	got = partitionMessageHashes(hashesN(1, 2, 3, 4, 5, 6, 7), 3)
	require.Len(t, got, 3)
	require.Equal(t, hashesN(1, 2, 3), got[0])
	require.Equal(t, hashesN(4, 5, 6), got[1])
	require.Equal(t, hashesN(7), got[2])

	// defensive: a non-positive cap must not divide-by-zero or loop forever; one batch
	require.Len(t, partitionMessageHashes(hashesN(1, 2), 0), 1)
	require.Len(t, partitionMessageHashes(hashesN(1, 2), -1), 1)
}

// TestPartitionContentTopics covers the metadata-query topic chunking that generalizes
// the hash-first walk to the general (all-filters) history sync (issue #21470-hf): the
// global sync batches MANY content topics, and a single store query may carry at most
// maxContentTopicsPerRequest (mirrors the classic HistoryRetriever's maxTopicsPerRequest
// and the verifier's maxContentTopicsPerRequest). Chunking wrong would have the store node
// reject or silently truncate the query, dropping whole topics from the walk.
func TestPartitionContentTopics(t *testing.T) {
	require.Equal(t, 10, hashFirstMaxContentTopicsPerRequest,
		"must mirror the classic HistoryRetriever maxTopicsPerRequest / verifier maxContentTopicsPerRequest")

	// empty -> no chunks
	require.Empty(t, partitionContentTopics(nil, 10))
	require.Empty(t, partitionContentTopics([]string{}, 10))

	// fewer than the cap -> a single chunk, order preserved
	got := partitionContentTopics([]string{"a", "b", "c"}, 10)
	require.Len(t, got, 1)
	require.Equal(t, []string{"a", "b", "c"}, got[0])

	// exactly the cap -> a single full chunk
	got = partitionContentTopics([]string{"a", "b", "c", "d"}, 4)
	require.Len(t, got, 1)
	require.Len(t, got[0], 4)

	// one over the cap -> two chunks, remainder in the second
	got = partitionContentTopics([]string{"a", "b", "c", "d", "e"}, 4)
	require.Len(t, got, 2)
	require.Equal(t, []string{"a", "b", "c", "d"}, got[0])
	require.Equal(t, []string{"e"}, got[1])

	// several full chunks plus a remainder
	got = partitionContentTopics([]string{"a", "b", "c", "d", "e", "f", "g"}, 3)
	require.Len(t, got, 3)
	require.Equal(t, []string{"a", "b", "c"}, got[0])
	require.Equal(t, []string{"d", "e", "f"}, got[1])
	require.Equal(t, []string{"g"}, got[2])

	// defensive: a non-positive cap must not divide-by-zero or loop forever; one chunk
	require.Len(t, partitionContentTopics([]string{"a", "b"}, 0), 1)
	require.Len(t, partitionContentTopics([]string{"a", "b"}, -1), 1)
}

// TestSplitWindowNewestFirst covers the multi-day window walk that generalizes the
// hash-first backfill to the general history sync (issue #21470-hf). The general path can
// span up to the ~9-day default sync period, but the classic HistoryRetriever caps each
// store query to 24h and walks the range backward (see history.go's exceeds24h handling);
// a single metadata query over a multi-day range would silently cover only the newest 24h
// while the watermark still advances to `to`, losing the older days. So the metadata walk
// must be split into consecutive <=24h sub-windows, ordered NEWEST-FIRST so bodies are
// fetched most-recent-first (matching aa9fa29b5).
func TestSplitWindowNewestFirst(t *testing.T) {
	const day = int64(24 * time.Hour)

	// empty / inverted range -> no windows
	require.Empty(t, splitWindowNewestFirst(1000, 1000, day))
	require.Empty(t, splitWindowNewestFirst(2000, 1000, day))

	// range <= maxWindow -> a single window equal to the input
	got := splitWindowNewestFirst(0, day, day)
	require.Equal(t, []hashFirstTimeWindow{{fromNano: 0, toNano: day}}, got)

	// sub-window range -> a single window equal to the input
	got = splitWindowNewestFirst(100, 500, day)
	require.Equal(t, []hashFirstTimeWindow{{fromNano: 100, toNano: 500}}, got)

	// exactly two days -> two full 24h windows, newest first, contiguous, no gaps/overlap
	got = splitWindowNewestFirst(0, 2*day, day)
	require.Equal(t, []hashFirstTimeWindow{
		{fromNano: day, toNano: 2 * day},
		{fromNano: 0, toNano: day},
	}, got)

	// nine-day default -> nine contiguous windows, newest-first, covering the whole range
	got = splitWindowNewestFirst(0, 9*day, day)
	require.Len(t, got, 9)
	require.Equal(t, 9*day, got[0].toNano, "first window ends at `to` (newest)")
	require.Equal(t, int64(0), got[len(got)-1].fromNano, "last window starts at `from` (oldest)")
	for i := 0; i < len(got); i++ {
		require.LessOrEqual(t, got[i].toNano-got[i].fromNano, day, "no window exceeds 24h")
		if i > 0 {
			require.Equal(t, got[i-1].fromNano, got[i].toNano, "windows are contiguous, no gap/overlap")
		}
	}

	// partial trailing window -> oldest window is clamped to `from`, never before it
	got = splitWindowNewestFirst(0, day+500, day)
	require.Equal(t, []hashFirstTimeWindow{
		{fromNano: 500, toNano: day + 500},
		{fromNano: 0, toNano: 500},
	}, got)

	// defensive: a non-positive maxWindow must not loop forever; one window spanning the range
	require.Equal(t, []hashFirstTimeWindow{{fromNano: 0, toNano: 2 * day}}, splitWindowNewestFirst(0, 2*day, 0))
	require.Equal(t, []hashFirstTimeWindow{{fromNano: 0, toNano: 2 * day}}, splitWindowNewestFirst(0, 2*day, -1))
}

// TestFilterUnknownHashes covers known-hash filtering: bodies are fetched ONLY for
// hashes not already held locally, so a hash the node already processed must be dropped
// (issue #21470-hf). This is what turns off the redundant re-download of the
// heartbeat-republished community description.
func TestFilterUnknownHashes(t *testing.T) {
	// known set: hashes 2 and 4 are already held locally
	knownSet := map[wpb.MessageHash]bool{hashN(2): true, hashN(4): true}
	exists := func(h wpb.MessageHash) (bool, error) { return knownSet[h], nil }

	// mixed: 1,3,5 unknown (kept, in order); 2,4 known (dropped, counted)
	unknown, known, err := filterUnknownHashes(hashesN(1, 2, 3, 4, 5), exists)
	require.NoError(t, err)
	require.Equal(t, hashesN(1, 3, 5), unknown)
	require.Equal(t, 2, known)

	// all known -> nothing to fetch
	unknown, known, err = filterUnknownHashes(hashesN(2, 4), exists)
	require.NoError(t, err)
	require.Empty(t, unknown)
	require.Equal(t, 2, known)

	// all unknown -> all fetched
	unknown, known, err = filterUnknownHashes(hashesN(7, 8, 9), exists)
	require.NoError(t, err)
	require.Equal(t, hashesN(7, 8, 9), unknown)
	require.Equal(t, 0, known)
}

// TestFilterUnknownHashes_Dedup verifies a hash repeated in the input is fetched at most
// once (the store node can return the same hash across pages; fetching its 1.4MB body
// twice is exactly the waste we are removing).
func TestFilterUnknownHashes_Dedup(t *testing.T) {
	exists := func(wpb.MessageHash) (bool, error) { return false, nil }

	unknown, known, err := filterUnknownHashes(hashesN(1, 1, 2, 1, 2), exists)
	require.NoError(t, err)
	require.Equal(t, hashesN(1, 2), unknown, "each distinct unknown hash fetched once, first-seen order")
	require.Equal(t, 0, known)
}

// TestFilterUnknownHashes_ExistsError verifies an existence-check error aborts (we must
// not silently treat an unreadable local state as "unknown" and refetch everything).
func TestFilterUnknownHashes_ExistsError(t *testing.T) {
	boom := errors.New("boom")
	exists := func(wpb.MessageHash) (bool, error) { return false, boom }

	_, _, err := filterUnknownHashes(hashesN(1), exists)
	require.ErrorIs(t, err, boom)
}

// TestBuildMetadataQuery covers the two behavioral contracts of the hash-walk query
// (issue #21470-hf): it must request NO bodies (the whole point — metadata only) and it
// must page NEWEST-FIRST so bodies are fetched most-recent-first.
func TestBuildMetadataQuery(t *testing.T) {
	q := buildMetadataQuery("req-1", "pub-a", []string{"0x01", "0x02"}, 1000, 2000)

	require.False(t, q.IncludeData, "metadata walk must NOT fetch bodies")
	require.False(t, q.PaginationForward, "must page newest-first so bodies are fetched most-recent-first")
	require.Equal(t, uint64(hashFirstMetadataPageSize), q.GetPaginationLimit())
	require.Equal(t, "pub-a", q.GetPubsubTopic())
	require.Equal(t, []string{"0x01", "0x02"}, q.ContentTopics)
	require.Equal(t, int64(1000), q.GetTimeStart())
	require.Equal(t, int64(2000), q.GetTimeEnd())
}

// TestPlanBodyFetch covers the keyless body-skip decision (issue #21470-hf
// enhancement): a keyless spectator whose community description is already resolved must
// fetch NO bodies (every body on the shared community topic is an undecryptable channel
// message or a description already held), while a keyed user fetches everything.
func TestPlanBodyFetch(t *testing.T) {
	unknown := hashesN(1, 2, 3)

	// skip: fetch nothing, count all as keyless-skipped
	toFetch, skipped := planBodyFetch(unknown, true)
	require.Empty(t, toFetch, "keyless + resolved description fetches no bodies")
	require.Equal(t, 3, skipped)

	// no-skip: fetch everything, skip nothing
	toFetch, skipped = planBodyFetch(unknown, false)
	require.Equal(t, unknown, toFetch)
	require.Equal(t, 0, skipped)

	// skip with nothing unknown: no bodies, no skips
	toFetch, skipped = planBodyFetch(nil, true)
	require.Empty(t, toFetch)
	require.Equal(t, 0, skipped)
}

// TestHashFirstStatsAdd covers the window bookkeeping: a backfill spans several batches,
// and the once-per-backfill INFO log needs their sum (issue #21470-hf).
func TestHashFirstStatsAdd(t *testing.T) {
	var total types.HashFirstStats
	total.Add(types.HashFirstStats{HashesSeen: 100, HashesKnown: 90, BodiesFetched: 10, BytesEstimate: 1000, BodiesSkippedKeyless: 0})
	total.Add(types.HashFirstStats{HashesSeen: 40, HashesKnown: 9, BodiesFetched: 0, BytesEstimate: 0, BodiesSkippedKeyless: 31})

	require.Equal(t, types.HashFirstStats{
		HashesSeen:           140,
		HashesKnown:          99,
		BodiesFetched:        10,
		BytesEstimate:        1000,
		BodiesSkippedKeyless: 31,
	}, total)
}
