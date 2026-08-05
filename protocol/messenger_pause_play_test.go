package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSetPausedUpdatesMessengerFlag(t *testing.T) {
	m := &Messenger{}

	m.SetPaused(true)
	require.True(t, m.isPaused())

	m.SetPaused(false)
	require.False(t, m.isPaused())
}

func TestSetPausedEnqueuesBoundedForegroundCatchup(t *testing.T) {
	m := &Messenger{
		started:             true,
		logger:              zap.NewNop(),
		historicSyncTrigger: make(chan struct{}, 1),
	}

	m.SetPaused(true)
	m.SetPaused(false)

	m.historicSyncQueueMu.Lock()
	require.Len(t, m.historicSyncQueue, 1)
	request := m.historicSyncQueue[0]
	m.historicSyncQueueMu.Unlock()
	require.True(t, request.bounded())
	require.True(t, request.From.Before(request.To))

	m.SetPaused(false)
	m.historicSyncQueueMu.Lock()
	require.Len(t, m.historicSyncQueue, 1, "idempotent resume must not enqueue another catch-up")
	m.historicSyncQueueMu.Unlock()
}
