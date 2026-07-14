package processor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShouldDropUndecryptableHashRatchetMessage(t *testing.T) {
	// Disabled: never drop, regardless of key material (default-off behaviour).
	require.False(t, shouldDropUndecryptableHashRatchetMessage(false, false))
	require.False(t, shouldDropUndecryptableHashRatchetMessage(false, true))

	// Enabled: drop only when the node holds no key material for the group
	// (keyless spectator). A member missing this rotation's key still holds
	// other keys, so the message is kept for retroactive replay.
	require.True(t, shouldDropUndecryptableHashRatchetMessage(true, false))
	require.False(t, shouldDropUndecryptableHashRatchetMessage(true, true))
}
