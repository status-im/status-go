package timesource

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWallClockAhead(t *testing.T) {
	require.False(t, wallClockAhead(time.Minute, time.Minute))
	require.False(t, wallClockAhead(time.Minute+time.Second, time.Minute),
		"small clock adjustments are not a suspend")
	require.True(t, wallClockAhead(time.Minute+time.Second+time.Nanosecond, time.Minute))
	require.True(t, wallClockAhead(10*time.Hour, 8*time.Second))
	require.False(t, wallClockAhead(time.Minute, 2*time.Minute),
		"a wall clock moved backwards is not a suspend")
}

func TestSuspendedBetween(t *testing.T) {
	now := time.Now()
	require.False(t, SuspendedBetween(now, now))
	require.False(t, SuspendedBetween(now, time.Now()))
	require.False(t, SuspendedBetween(now.Round(0), now.Round(0).Add(10*time.Hour)),
		"readings without a monotonic component never report a suspend")
}
