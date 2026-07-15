package communities

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func hashOf(b []byte) [32]byte {
	return hashDescriptionPayload(b)
}

// keysHeld/keyless make the holdsKeys argument to shouldSkip read as intent at
// the call site.
const (
	keysHeld = true
	keyless  = false
)

// Keys-held: only a strictly-older clock or a byte-identical same-clock
// redelivery may be skipped; same-clock-different-content must proceed (key
// rotation).
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
	require.False(t, g.shouldSkip(id, 10, hashA, keysHeld), "unknown community must proceed")

	// Record clock 10 / content A as fully processed.
	g.recordProcessed(id, 10, hashA)

	// Byte-identical same-clock redelivery: skip.
	require.True(t, g.shouldSkip(id, 10, hashA, keysHeld), "byte-identical same-clock redelivery must skip")

	// Same clock, different content: MUST proceed (community.go intentionally
	// reprocesses identical clocks so a previously-encrypted description can be
	// re-evaluated once keys arrive — this is the member key-rotation path).
	require.False(t, g.shouldSkip(id, 10, hashB, keysHeld), "same-clock different-content must proceed for a keys-held community")

	// Strictly-older clock: skip (can never win the downstream clock comparison).
	require.True(t, g.shouldSkip(id, 9, hashA, keysHeld), "older clock must skip")
	require.True(t, g.shouldSkip(id, 9, hashB, keysHeld), "older clock must skip regardless of content")

	// Newer clock: proceed.
	require.False(t, g.shouldSkip(id, 11, hashA, keysHeld), "newer clock must proceed")

	// A different community is unaffected by this one's entry.
	require.False(t, g.shouldSkip("0xother", 1, hashA, keysHeld), "other community must proceed")
}

// Keyless: equal-clock republishes are skipped regardless of content hash —
// the one behaviour that differs from keys-held.
func TestDescriptionGateKeylessTruthTable(t *testing.T) {
	const id = "0xcommunity"
	hashA := hashOf([]byte("description-bytes-A"))
	hashB := hashOf([]byte("description-bytes-B"))

	g := newCommunityDescriptionGate()

	// Unknown community: never skip, even keyless (nothing recorded yet).
	require.False(t, g.shouldSkip(id, 10, hashA, keyless), "unknown community must proceed")

	g.recordProcessed(id, 10, hashA)

	// Equal clock, same content: skip (same as keys-held).
	require.True(t, g.shouldSkip(id, 10, hashA, keyless), "keyless equal-clock same-content must skip")

	// Equal clock, DIFFERENT content: skip — this is the refinement. A re-encrypted
	// republish at the same clock is useless to a keyless spectator.
	require.True(t, g.shouldSkip(id, 10, hashB, keyless), "keyless equal-clock different-content must skip")

	// Strictly-older clock: skip regardless of content.
	require.True(t, g.shouldSkip(id, 9, hashA, keyless), "keyless older clock must skip")
	require.True(t, g.shouldSkip(id, 9, hashB, keyless), "keyless older clock must skip regardless of content")

	// Newer clock: proceed — a genuinely newer description must always be processed.
	require.False(t, g.shouldSkip(id, 11, hashA, keyless), "keyless newer clock must proceed")
	require.False(t, g.shouldSkip(id, 11, hashB, keyless), "keyless newer clock must proceed regardless of content")
}

// Fail-open (holdsKeys=true on lookup error) must behave exactly like the
// keys-held branch.
func TestDescriptionGateFailOpenTreatedAsKeysHeld(t *testing.T) {
	const id = "0xcommunity"
	hashA := hashOf([]byte("A"))
	hashB := hashOf([]byte("B"))

	g := newCommunityDescriptionGate()
	g.recordProcessed(id, 10, hashA)

	// holdsKeys=true is what communityHoldsAnyDecryptionKey returns on any error.
	require.False(t, g.shouldSkip(id, 10, hashB, keysHeld), "fail-open (keys-held) equal-clock different-content must proceed")
}

func TestDescriptionGateRecordIsMonotonic(t *testing.T) {
	const id = "0xcommunity"
	hash := hashOf([]byte("x"))

	g := newCommunityDescriptionGate()
	g.recordProcessed(id, 20, hash)
	g.recordProcessed(id, 5, hash) // stale, must not overwrite the newer record

	require.True(t, g.shouldSkip(id, 20, hash, keysHeld), "recorded clock is still 20, identical redelivery skips")
	require.False(t, g.shouldSkip(id, 21, hash, keysHeld), "a genuinely newer clock still proceeds")
	require.True(t, g.shouldSkip(id, 4, hash, keysHeld), "older than recorded 20 skips")
}

func TestDescriptionGateForget(t *testing.T) {
	const id = "0xcommunity"
	hash := hashOf([]byte("x"))

	g := newCommunityDescriptionGate()
	g.recordProcessed(id, 10, hash)
	require.True(t, g.shouldSkip(id, 10, hash, keysHeld))

	g.forget(id)
	require.False(t, g.shouldSkip(id, 10, hash, keysHeld), "after forget, identical redelivery reprocesses")
}

func TestDescriptionGateForgetAll(t *testing.T) {
	hash := hashOf([]byte("x"))
	g := newCommunityDescriptionGate()
	g.recordProcessed("0xa", 10, hash)
	g.recordProcessed("0xb", 10, hash)
	require.True(t, g.shouldSkip("0xa", 10, hash, keysHeld))
	require.True(t, g.shouldSkip("0xb", 10, hash, keysHeld))

	g.forgetAll()
	require.False(t, g.shouldSkip("0xa", 10, hash, keysHeld))
	require.False(t, g.shouldSkip("0xb", 10, hash, keysHeld))
}
