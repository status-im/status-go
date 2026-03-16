package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"

	messaginglifecycle "github.com/status-im/status-go/pkg/messaging/lifecycle"
)

func TestSetPausedBackgroundUpdatesLifecycleFlag(t *testing.T) {
	t.Cleanup(func() {
		messaginglifecycle.SetPausedBackground(false)
	})

	m := &Messenger{}

	m.SetPausedBackground(true)
	require.True(t, m.isPausedBackground())
	require.True(t, messaginglifecycle.IsPausedBackground())

	m.SetPausedBackground(false)
	require.False(t, m.isPausedBackground())
	require.False(t, messaginglifecycle.IsPausedBackground())
}
