package backend

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newTestReleaser(t *testing.T) (*memoryReleaser, *int, *time.Time, *func()) {
	t.Helper()
	calls := 0
	now := time.Unix(1_000_000, 0)
	var pending func()
	r := newMemoryReleaser()
	r.release = func() { calls++ }
	r.now = func() time.Time { return now }
	r.afterFn = func(_ time.Duration, f func()) *time.Timer {
		pending = f
		return time.NewTimer(time.Hour)
	}
	return r, &calls, &now, &pending
}

func TestMemoryReleaserThrottlesWithinMinGap(t *testing.T) {
	r, calls, now, _ := newTestReleaser(t)

	r.releaseNow()
	require.Equal(t, 1, *calls)

	*now = now.Add(r.minGap / 2)
	r.releaseNow()
	require.Equal(t, 1, *calls, "a second release inside the gap is skipped")

	*now = now.Add(r.minGap)
	r.releaseNow()
	require.Equal(t, 2, *calls)
}

func TestMemoryReleaserPauseWaitsForScheduledRelease(t *testing.T) {
	r, calls, _, pending := newTestReleaser(t)

	r.scheduleAfterLogin(time.Minute)
	require.NotNil(t, *pending)
	r.releaseNow() // the UI went to background during the burst: wait for the scheduled release
	require.Equal(t, 0, *calls)
	(*pending)()
	require.Equal(t, 1, *calls, "the post-login release runs")
	require.Nil(t, r.timer)
	r.releaseNow()
	require.Equal(t, 1, *calls, "pause releases stay throttled inside the gap")
}

func TestMemoryReleaserStopCancelsPendingTimer(t *testing.T) {
	r, _, _, _ := newTestReleaser(t)

	r.scheduleAfterLogin(time.Minute)
	require.NotNil(t, r.timer)
	r.stop()
	require.Nil(t, r.timer)
	r.stop() // idempotent
}
