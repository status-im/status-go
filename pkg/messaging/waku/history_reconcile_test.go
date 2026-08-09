package wakuv2

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

func TestShouldReconcileHistory(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		reliable      bool
		wasReliable   bool
		lastReconcile time.Time
		want          bool
	}{
		{
			name:        "reliably connected: no periodic reconciliation",
			reliable:    true,
			wasReliable: true,
			want:        false,
		},
		{
			name:        "unreliable window just closed: reconcile once",
			reliable:    true,
			wasReliable: false,
			want:        true,
		},
		{
			name:          "unreliable, never reconciled: due immediately",
			reliable:      false,
			wasReliable:   false,
			lastReconcile: time.Time{},
			want:          true,
		},
		{
			name:          "unreliable, reconciled recently: not due yet",
			reliable:      false,
			wasReliable:   false,
			lastReconcile: now.Add(-historyReconcileMinInterval / 2),
			want:          false,
		},
		{
			name:          "unreliable, min interval elapsed: due again",
			reliable:      false,
			wasReliable:   false,
			lastReconcile: now.Add(-2 * historyReconcileMinInterval),
			want:          true,
		},
		{
			name:          "just turned unreliable after recent reconcile: waits for interval",
			reliable:      false,
			wasReliable:   true,
			lastReconcile: now.Add(-historyReconcileMinInterval / 2),
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldReconcileHistory(tt.reliable, tt.wasReliable, tt.lastReconcile, now, historyReconcileMinInterval)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestHistoryReconcileTrackerBoundsUnreliableWindow(t *testing.T) {
	start := time.Unix(1_000, 0)
	tracker := newHistoryReconcileTracker(true, start)

	window := tracker.observe(true, start.Add(10*time.Second), historyReconcileMinInterval)
	require.Nil(t, window)

	// The transition occurred between observations, so use the previous known
	// reliable time as the conservative lower boundary.
	window = tracker.observe(false, start.Add(20*time.Second), historyReconcileMinInterval)
	require.NotNil(t, window)
	require.Equal(t, start.Add(10*time.Second), window.From)
	require.Equal(t, start.Add(20*time.Second), window.To)

	window = tracker.observe(false, start.Add(50*time.Second), historyReconcileMinInterval)
	require.NotNil(t, window)
	require.Equal(t, start.Add(10*time.Second), window.From)
	require.Equal(t, start.Add(50*time.Second), window.To)

	window = tracker.observe(true, start.Add(60*time.Second), historyReconcileMinInterval)
	require.NotNil(t, window)
	require.Equal(t, start.Add(10*time.Second), window.From)
	require.Equal(t, start.Add(60*time.Second), window.To)

	window = tracker.observe(true, start.Add(90*time.Second), historyReconcileMinInterval)
	require.Nil(t, window)
}

func TestHistoryReconcileTrackerCapturesRapidRecovery(t *testing.T) {
	start := time.Unix(1_500, 0)
	tracker := newHistoryReconcileTracker(true, start)

	degradedAt := start.Add(time.Second)
	window := tracker.observe(false, degradedAt, historyReconcileMinInterval)
	require.NotNil(t, window)
	require.Equal(t, start, window.From)
	require.Equal(t, degradedAt, window.To)

	recoveredAt := degradedAt.Add(time.Second)
	window = tracker.observe(true, recoveredAt, historyReconcileMinInterval)
	require.NotNil(t, window)
	require.Equal(t, start, window.From)
	require.Equal(t, recoveredAt, window.To)
}

func TestHistoryReconcileTrackerPreservesDisjointWindows(t *testing.T) {
	start := time.Unix(2_000, 0)
	tracker := newHistoryReconcileTracker(true, start)

	first := tracker.observe(false, start.Add(10*time.Second), historyReconcileMinInterval)
	require.NotNil(t, first)
	window := tracker.observe(true, start.Add(20*time.Second), historyReconcileMinInterval)
	require.NotNil(t, window)

	window = tracker.observe(true, start.Add(50*time.Second), historyReconcileMinInterval)
	require.Nil(t, window)
	second := tracker.observe(false, start.Add(60*time.Second), historyReconcileMinInterval)
	require.NotNil(t, second)

	require.Equal(t, start, first.From)
	require.Equal(t, start.Add(50*time.Second), second.From)
	require.True(t, first.To.Before(second.From))
}

func TestReliablyConnected(t *testing.T) {
	core := &Waku{
		cfg:       &Config{Mode: ModeCore},
		connState: types.ConnectionStateConnected,
	}
	require.True(t, core.reliablyConnected())

	core.connState = types.ConnectionStatePartiallyConnected
	require.False(t, core.reliablyConnected())

	edge := &Waku{
		cfg:       &Config{Mode: ModeEdge},
		connState: types.ConnectionStateConnected,
	}
	require.True(t, edge.reliablyConnected())
}
