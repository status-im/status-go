package wakuv2

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
