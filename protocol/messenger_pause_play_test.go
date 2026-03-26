package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetPausedUpdatesMessengerFlag(t *testing.T) {
	m := &Messenger{}

	m.SetPaused(true)
	require.True(t, m.isPaused())

	m.SetPaused(false)
	require.False(t, m.isPaused())
}
