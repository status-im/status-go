package backend

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/pkg/services/wallet/walletevent"
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
	r.spawn = func(f func()) { f() }
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

func TestMemoryReleaserReleaseNowRunsOffCaller(t *testing.T) {
	r, calls, _, _ := newTestReleaser(t)
	spawned := 0
	r.spawn = func(func()) { spawned++ }

	r.releaseNow()
	require.Equal(t, 1, spawned, "the release is handed to spawn")
	require.Equal(t, 0, *calls, "nothing runs on the caller")

	r.releaseNow()
	require.Equal(t, 1, spawned, "the throttle is recorded before the release runs")
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

func TestMemoryReleaserStaleCallbackDoesNotRelease(t *testing.T) {
	r, calls, _, pending := newTestReleaser(t)

	r.scheduleAfterLogin(time.Minute)
	stopped := *pending
	r.stop()
	stopped() // fired before stop() took the lock
	require.Equal(t, 0, *calls)

	r.scheduleAfterLogin(time.Minute)
	replaced := *pending
	r.scheduleAfterLogin(time.Minute)
	replaced()
	require.Equal(t, 0, *calls)
	require.NotNil(t, r.timer, "the replacing timer stays armed")

	(*pending)()
	require.Equal(t, 1, *calls)
}

// Real timer, zero delay: the callback can start before scheduleAfterLogin returns, which is
// the interleaving the race detector checks.
func TestMemoryReleaserTimerCallbackRacesWithArming(t *testing.T) {
	released := make(chan struct{}, 1)
	r := newMemoryReleaser()
	r.release = func() { released <- struct{}{} }

	r.scheduleAfterLogin(0)
	select {
	case <-released:
	case <-time.After(5 * time.Second):
		t.Fatal("scheduled release did not run")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	require.Nil(t, r.timer)
}

func TestMemoryReleaserReleasesGoHeapThenNativeHeap(t *testing.T) {
	var order []string
	r := newMemoryReleaser()
	r.freeOSMemory = func() { order = append(order, "go") }
	r.purgeNativeHeap = func() { order = append(order, "native") }

	r.release()
	require.Equal(t, []string{"go", "native"}, order, "finalizers run by the GC free native memory before the purge")
}

type fakeTimer struct {
	delay time.Duration
	fire  func()
}

// newTrailingTestReleaser records every armed timer so a test can fire the one it means.
func newTrailingTestReleaser(t *testing.T) (*memoryReleaser, *int, *time.Time, *[]*fakeTimer) {
	r, calls, now, _ := newTestReleaser(t)
	var timers []*fakeTimer
	r.afterFn = func(d time.Duration, f func()) *time.Timer {
		timers = append(timers, &fakeTimer{delay: d, fire: f})
		return time.NewTimer(time.Hour)
	}
	return r, calls, now, &timers
}

func tokenListsUpdated() walletevent.Event {
	return walletevent.Event{Type: walletevent.EventTokenListsUpdated}
}

func TestMemoryReleaserTokenListsUpdateReleasesAfterSettling(t *testing.T) {
	r, calls, now, timers := newTrailingTestReleaser(t)

	r.onWalletEvent(walletevent.Event{Type: walletevent.EventType("other")})
	require.Empty(t, *timers, "other wallet events are ignored")

	r.onWalletEvent(tokenListsUpdated())
	require.Len(t, *timers, 1)
	require.Equal(t, tokenListsReleaseDelay, (*timers)[0].delay, "the release waits for the rebuild the update triggers")
	require.Equal(t, 0, *calls)

	*now = now.Add(tokenListsReleaseDelay)
	(*timers)[0].fire()
	require.Equal(t, 1, *calls)
}

func TestMemoryReleaserTokenListsUpdateTrailsThrottledRelease(t *testing.T) {
	r, calls, now, timers := newTrailingTestReleaser(t)

	r.releaseNow() // the post-login release just ran
	require.Equal(t, 1, *calls)

	*now = now.Add(time.Second)
	r.onWalletEvent(tokenListsUpdated())
	r.onWalletEvent(tokenListsUpdated())
	r.onWalletEvent(tokenListsUpdated())
	require.Len(t, *timers, 1, "a burst of updates collapses to one trailing release")
	require.Equal(t, r.minGap-time.Second, (*timers)[0].delay, "it lands when the gap ends")

	*now = now.Add((*timers)[0].delay)
	(*timers)[0].fire()
	require.Equal(t, 2, *calls)

	r.onWalletEvent(tokenListsUpdated())
	require.Len(t, *timers, 2, "a fired trailing release can be re-armed")
}

func TestMemoryReleaserTokenListsUpdateDuringBurstIsCovered(t *testing.T) {
	r, calls, _, timers := newTrailingTestReleaser(t)

	r.scheduleAfterLogin(time.Minute)
	r.onWalletEvent(tokenListsUpdated())
	require.Len(t, *timers, 1, "the pending post-login release covers an update inside the burst")

	(*timers)[0].fire()
	require.Equal(t, 1, *calls)
	require.Nil(t, r.timer)
}

func TestMemoryReleaserTrailingReleaseSkipsAfterPauseRelease(t *testing.T) {
	r, calls, now, timers := newTrailingTestReleaser(t)

	r.onWalletEvent(tokenListsUpdated())
	*now = now.Add(time.Second)
	r.releaseNow() // the app went to background before the trailing release
	require.Equal(t, 1, *calls)

	*now = now.Add(tokenListsReleaseDelay)
	(*timers)[0].fire()
	require.Equal(t, 1, *calls, "the pause release already returned the refresh's garbage")
}

func TestMemoryReleaserStopCancelsTrailingRelease(t *testing.T) {
	r, calls, now, timers := newTrailingTestReleaser(t)

	r.onWalletEvent(tokenListsUpdated())
	require.NotNil(t, r.trailing)
	r.stop()
	require.Nil(t, r.trailing)

	*now = now.Add(tokenListsReleaseDelay)
	(*timers)[0].fire() // fired before stop() took the lock
	require.Equal(t, 0, *calls)
}

func TestMemoryReleaserWatchesTokenListsUntilStop(t *testing.T) {
	r, _, _, _ := newTestReleaser(t)
	released := make(chan struct{}, 1)
	r.release = func() { released <- struct{}{} }
	r.afterFn = func(_ time.Duration, f func()) *time.Timer { return time.AfterFunc(0, f) }
	var feed event.Feed
	update := walletevent.Event{Type: walletevent.EventTokenListsUpdated}

	r.watchTokenLists(&feed)
	r.watchTokenLists(&feed) // a second login replaces the watch
	require.Eventually(t, func() bool { return feed.Send(update) > 0 }, 5*time.Second, time.Millisecond)
	require.Equal(t, 1, feed.Send(walletevent.Event{Type: walletevent.EventType("other")}), "one subscriber after re-arming")
	select {
	case <-released:
	case <-time.After(5 * time.Second):
		t.Fatal("token-list update did not release")
	}

	r.stop()
	require.Equal(t, 0, feed.Send(update), "stop unsubscribes from the wallet feed")
}

// The watch holds the logged-out node's wallet feed; logout must drop it with the pending release.
func TestLogoutStopsMemoryReleaser(t *testing.T) {
	b := NewStatusBackend(testutils.MustCreateTestLogger())
	b.UpdateRootDataDir(t.TempDir())
	var feed event.Feed
	b.memRelease.scheduleAfterLogin(time.Hour)
	b.memRelease.watchTokenLists(&feed)
	require.Eventually(t, func() bool { return feed.Send(tokenListsUpdated()) > 0 }, 5*time.Second, time.Millisecond)

	require.NoError(t, b.Logout())

	require.Equal(t, 0, feed.Send(tokenListsUpdated()), "logout unsubscribes from the wallet feed")
	b.memRelease.mu.Lock()
	defer b.memRelease.mu.Unlock()
	require.Nil(t, b.memRelease.timer)
	require.Nil(t, b.memRelease.trailing)
}
