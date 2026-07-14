package wakuv2

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	wpb "github.com/waku-org/go-waku/waku/v2/protocol/pb"

	types "github.com/status-im/status-go/pkg/messaging/waku/types"
)

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
