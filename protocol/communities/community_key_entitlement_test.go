package communities

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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

	// forgetAll clears every cached entitlement (used when any new key arrives).
	c.forgetAll()
	_, ok = c.known("comm-a")
	require.False(t, ok)
	_, ok = c.known("comm-b")
	require.False(t, ok)
}

func TestCommunityKeyEntitlementDropLogging(t *testing.T) {
	c := newCommunityKeyEntitlement()

	// One log per community per session.
	require.True(t, c.shouldLogDrop("comm-a"))
	require.False(t, c.shouldLogDrop("comm-a"))
	require.True(t, c.shouldLogDrop("comm-b"))
	require.False(t, c.shouldLogDrop("comm-a"))
}
