package communities

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func hashOf(b []byte) [32]byte {
	return hashDescriptionPayload(b)
}

// TestDescriptionGateTruthTable exercises the redelivery gate decision across the
// whole truth table (issue #21470-hf): unknown community, strictly-older clock,
// byte-identical same-clock redelivery, same-clock-different-content, and a newer
// clock. Only a strictly-older clock or a byte-identical same-clock redelivery may
// be skipped; everything else must proceed to full processing.
func TestDescriptionGateTruthTable(t *testing.T) {
	const id = "0xcommunity"
	payloadA := []byte("description-bytes-A")
	payloadB := []byte("description-bytes-B")
	hashA := hashOf(payloadA)
	hashB := hashOf(payloadB)

	g := newCommunityDescriptionGate()

	// Unknown community: never skip (fail open — nothing recorded yet). This is
	// also the queued-validation guarantee: a community that has only ever been
	// queued (never fully processed) has no gate entry, so validateCommunity's
	// re-entry is never blocked.
	require.False(t, g.shouldSkip(id, 10, hashA), "unknown community must proceed")

	// Record clock 10 / content A as fully processed.
	g.recordProcessed(id, 10, hashA)

	// Byte-identical same-clock redelivery: skip.
	require.True(t, g.shouldSkip(id, 10, hashA), "byte-identical same-clock redelivery must skip")

	// Same clock, different content: MUST proceed (community.go intentionally
	// reprocesses identical clocks so a previously-encrypted description can be
	// re-evaluated once keys arrive).
	require.False(t, g.shouldSkip(id, 10, hashB), "same-clock different-content must proceed")

	// Strictly-older clock: skip (can never win the downstream clock comparison).
	require.True(t, g.shouldSkip(id, 9, hashA), "older clock must skip")
	require.True(t, g.shouldSkip(id, 9, hashB), "older clock must skip regardless of content")

	// Newer clock: proceed.
	require.False(t, g.shouldSkip(id, 11, hashA), "newer clock must proceed")

	// A different community is unaffected by this one's entry.
	require.False(t, g.shouldSkip("0xother", 1, hashA), "other community must proceed")
}

// TestDescriptionGateRecordIsMonotonic verifies recordProcessed never moves the
// recorded clock backwards, so an out-of-order older success cannot shadow a newer
// one and cause a newer redelivery to be wrongly skipped.
func TestDescriptionGateRecordIsMonotonic(t *testing.T) {
	const id = "0xcommunity"
	hash := hashOf([]byte("x"))

	g := newCommunityDescriptionGate()
	g.recordProcessed(id, 20, hash)
	g.recordProcessed(id, 5, hash) // stale, must not overwrite the newer record

	require.True(t, g.shouldSkip(id, 20, hash), "recorded clock is still 20, identical redelivery skips")
	require.False(t, g.shouldSkip(id, 21, hash), "a genuinely newer clock still proceeds")
	require.True(t, g.shouldSkip(id, 4, hash), "older than recorded 20 skips")
}

// TestDescriptionGateForget verifies a single-community invalidation (community
// delete) forces the next description to be fully reprocessed.
func TestDescriptionGateForget(t *testing.T) {
	const id = "0xcommunity"
	hash := hashOf([]byte("x"))

	g := newCommunityDescriptionGate()
	g.recordProcessed(id, 10, hash)
	require.True(t, g.shouldSkip(id, 10, hash))

	g.forget(id)
	require.False(t, g.shouldSkip(id, 10, hash), "after forget, identical redelivery reprocesses")
}

// TestDescriptionGateForgetAll verifies key-arrival invalidation lifts the gate for
// every community so a byte-identical description can be reprocessed and decrypt
// now-available private data.
func TestDescriptionGateForgetAll(t *testing.T) {
	hash := hashOf([]byte("x"))
	g := newCommunityDescriptionGate()
	g.recordProcessed("0xa", 10, hash)
	g.recordProcessed("0xb", 10, hash)
	require.True(t, g.shouldSkip("0xa", 10, hash))
	require.True(t, g.shouldSkip("0xb", 10, hash))

	g.forgetAll()
	require.False(t, g.shouldSkip("0xa", 10, hash))
	require.False(t, g.shouldSkip("0xb", 10, hash))
}
