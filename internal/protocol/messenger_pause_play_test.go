package protocol

import (
	"testing"
	"time"

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

func TestSetPausedDoesNotEnqueueForegroundCatchup(t *testing.T) {
	m := &Messenger{
		started:             true,
		logger:              zap.NewNop(),
		historicSyncTrigger: make(chan struct{}, 1),
	}

	m.SetPaused(true)
	m.SetPaused(false)

	m.historicSyncQueueMu.Lock()
	require.Empty(t, m.historicSyncQueue)
	m.historicSyncQueueMu.Unlock()

	m.SetPaused(false)
	m.historicSyncQueueMu.Lock()
	require.Empty(t, m.historicSyncQueue, "idempotent resume must not enqueue a catch-up")
	m.historicSyncQueueMu.Unlock()
}

func TestAutomaticHistoricSyncIsDeferredWhilePaused(t *testing.T) {
	m := &Messenger{}
	m.SetPaused(true)

	executed, err := m.runAutomaticHistoricSync(historicSyncRequest{
		From: time.Unix(100, 0),
		To:   time.Unix(200, 0),
	})

	require.NoError(t, err)
	require.False(t, executed)
}
