package communities

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShouldSkipPrivateDataDecryption(t *testing.T) {
	// Skip only when there is encrypted private data AND the node holds no keys.
	require.True(t, shouldSkipPrivateDataDecryption(true, false), "keyless node with private data must skip")
	require.False(t, shouldSkipPrivateDataDecryption(true, true), "node holding keys must attempt")
	require.False(t, shouldSkipPrivateDataDecryption(false, false), "no private data means nothing to skip")
	require.False(t, shouldSkipPrivateDataDecryption(false, true), "no private data means nothing to skip")
}

func TestCommunityKeyEntitlementCache(t *testing.T) {
	c := newCommunityKeyEntitlement()

	// Unknown until remembered.
	_, ok := c.known("comm-a")
	require.False(t, ok)

	// Remembered values are returned.
	c.remember("comm-a", false)
	holds, ok := c.known("comm-a")
	require.True(t, ok)
	require.False(t, holds)

	c.remember("comm-b", true)
	holds, ok = c.known("comm-b")
	require.True(t, ok)
	require.True(t, holds)

	// forget invalidates only the named id, forcing a fresh lookup (gate lift on key arrival).
	c.forget("comm-a")
	_, ok = c.known("comm-a")
	require.False(t, ok)
	_, ok = c.known("comm-b")
	require.True(t, ok, "forget must not touch other communities")

	// forgetAll clears every cached entitlement (used when any new key arrives).
	c.remember("comm-a", false)
	c.forgetAll()
	_, ok = c.known("comm-a")
	require.False(t, ok)
	_, ok = c.known("comm-b")
	require.False(t, ok)
}

func TestCommunityKeyEntitlementDropLogging(t *testing.T) {
	c := newCommunityKeyEntitlement()

	// First drop for a community => INFO, count 1.
	d := c.recordDrop("comm-a")
	require.True(t, d.logInfo)
	require.False(t, d.logDebug)
	require.Equal(t, uint64(1), d.count)

	// Subsequent drops below the interval => silent, counting up.
	for i := 2; i < dropLogInterval; i++ {
		d = c.recordDrop("comm-a")
		require.False(t, d.logInfo, "only the first drop logs INFO")
		require.False(t, d.logDebug)
	}

	// Every dropLogInterval-th drop => DEBUG checkpoint, never a second INFO.
	d = c.recordDrop("comm-a")
	require.False(t, d.logInfo)
	require.True(t, d.logDebug)
	require.Equal(t, uint64(dropLogInterval), d.count)

	// A different community gets its own first-drop INFO and independent count.
	d = c.recordDrop("comm-b")
	require.True(t, d.logInfo)
	require.Equal(t, uint64(1), d.count)
}
