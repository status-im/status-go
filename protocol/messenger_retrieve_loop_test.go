package protocol

import (
	"testing"
	"time"
)

// TestNextFlushDeadline verifies the scheduling decision that bounds the
// retrieve-loop debounce latency: normally the flush fires after a quiet
// window (debounce) following the last match, but under a sustained match
// flood the ceiling (maxLatency measured from the first unflushed match)
// caps how long the flush can be delayed. See issue #21470.
func TestNextFlushDeadline(t *testing.T) {
	base := time.Unix(1000, 0)
	debounce := time.Second
	maxLatency := 3 * time.Second

	tests := []struct {
		name         string
		pendingSince time.Time
		now          time.Time
		want         time.Time
	}{
		{
			// First/isolated match: quiet window still fits under the ceiling,
			// so behaviour is unchanged from the pure debounce.
			name:         "quiet path debounce wins",
			pendingSince: base,
			now:          base,
			want:         base.Add(debounce),
		},
		{
			// Matches have been arriving for 2.5s without a quiet gap; the
			// ceiling (pendingSince+3s) is sooner than now+1s and must win.
			name:         "flood ceiling caps latency",
			pendingSince: base.Add(-2500 * time.Millisecond),
			now:          base,
			want:         base.Add(500 * time.Millisecond),
		},
		{
			// Exactly at the crossover: both deadlines land on now+1s.
			name:         "boundary deadlines equal",
			pendingSince: base.Add(-2 * time.Second),
			now:          base,
			want:         base.Add(debounce),
		},
		{
			// Just past the crossover: ceiling wins by 1ms.
			name:         "just past boundary ceiling wins",
			pendingSince: base.Add(-2001 * time.Millisecond),
			now:          base,
			want:         base.Add(999 * time.Millisecond),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := nextFlushDeadline(tc.pendingSince, tc.now, debounce, maxLatency)
			if !got.Equal(tc.want) {
				t.Fatalf("nextFlushDeadline(%v, %v, %v, %v) = %v, want %v",
					tc.pendingSince, tc.now, debounce, maxLatency, got, tc.want)
			}
		})
	}
}
