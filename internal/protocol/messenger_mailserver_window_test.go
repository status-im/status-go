package protocol

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
)

func TestApplyHistoryWindowFloor(t *testing.T) {
	window := &messagingtypes.HistoryReconcileWindow{
		From: time.Unix(1_000, 0),
		To:   time.Unix(1_100, 0),
	}

	require.Equal(t, uint32(940), applyHistoryWindowFloor(100, true, window),
		"an old cursor should be bounded to the unreliable window with tolerance")
	require.Equal(t, uint32(980), applyHistoryWindowFloor(980, true, window),
		"a newer completeness cursor should remain authoritative")
	require.Equal(t, uint32(100), applyHistoryWindowFloor(100, false, window),
		"an uninitialized topic must retain its initial-history range")
	require.Equal(t, uint32(100), applyHistoryWindowFloor(100, true, nil))
}

func TestTranslateHistoryReconcileWindowUsesMessengerClock(t *testing.T) {
	window := messagingtypes.HistoryReconcileWindow{
		From: time.Unix(100, 0),
		To:   time.Unix(200, 0),
	}

	got := translateHistoryReconcileWindow(
		window,
		time.Unix(250, 0),
		time.Unix(1_000, 0),
	)
	require.Equal(t, time.Unix(850, 0), got.From)
	require.Equal(t, time.Unix(950, 0), got.To)
}

func TestTranslateHistoryReconcileWindowSpansSuspend(t *testing.T) {
	// A window spanning an overnight suspend; the transport emits wall clock
	// times, so its length must survive the translation.
	beforeSleep := time.Unix(1_000, 0)
	wokeAt := beforeSleep.Add(10 * time.Hour)
	syncNow := wokeAt.Add(2 * time.Second)

	got := translateHistoryReconcileWindow(
		messagingtypes.HistoryReconcileWindow{From: beforeSleep, To: wokeAt},
		wokeAt.Add(time.Second),
		syncNow,
	)
	require.Equal(t, syncNow.Add(-time.Second).Add(-10*time.Hour), got.From)
	require.Equal(t, syncNow.Add(-time.Second), got.To)
}

func TestTranslateHistoryReconcileWindowIgnoresMonotonicReadings(t *testing.T) {
	// Mixing a wall clock window with a monotonic "now" must give the same
	// result as using wall clock times only.
	now := time.Now()
	window := messagingtypes.HistoryReconcileWindow{
		From: now.Round(0).Add(-time.Hour),
		To:   now.Round(0).Add(-time.Minute),
	}

	got := translateHistoryReconcileWindow(window, now, now)
	want := translateHistoryReconcileWindow(window, now.Round(0), now.Round(0))
	require.True(t, want.From.Equal(got.From))
	require.True(t, want.To.Equal(got.To))
	require.Equal(t, 59*time.Minute, got.To.Sub(got.From))
}

func TestHistoryCursorMonitorThrough(t *testing.T) {
	syncNow := time.Unix(100_000, 0)
	lastTick := time.Unix(1_000, 0)
	minBehind := time.Duration(tolerance) * time.Second

	through, ok := historyCursorMonitorThrough(syncNow, lastTick, lastTick.Add(time.Minute))
	require.True(t, ok)
	require.Equal(t, syncNow.Add(-minBehind), through,
		"an on-time tick stays one tolerance window behind")

	through, ok = historyCursorMonitorThrough(syncNow, lastTick, lastTick.Add(10*time.Second))
	require.True(t, ok)
	require.Equal(t, syncNow.Add(-minBehind), through)

	// A late tick, e.g. after a suspend the monotonic clock did not see,
	// never claims history past the previous tick.
	through, ok = historyCursorMonitorThrough(syncNow, lastTick, lastTick.Add(90*time.Second))
	require.True(t, ok)
	require.Equal(t, syncNow.Add(-90*time.Second), through)

	through, ok = historyCursorMonitorThrough(syncNow, lastTick, lastTick.Add(10*time.Hour))
	require.True(t, ok)
	require.Equal(t, syncNow.Add(-10*time.Hour), through)
}

func TestCoalesceHistoricSyncRequests(t *testing.T) {
	at := func(seconds int64) time.Time { return time.Unix(seconds, 0) }
	requests := []historicSyncRequest{
		{From: at(100), To: at(200)},
		{From: at(150), To: at(250)},
		{From: at(400), To: at(450)},
		{To: at(300)},
		{To: at(350)},
	}

	got := coalesceHistoricSyncRequests(requests)
	require.Equal(t, []historicSyncRequest{
		{To: at(350)},
		{From: at(400), To: at(450)},
	}, got)
}

func TestCoalesceHistoricSyncRequestsKeepsAdjacentReliableGap(t *testing.T) {
	first := historicSyncRequest{From: time.Unix(100, 0), To: time.Unix(200, 0)}
	second := historicSyncRequest{From: time.Unix(201, 0), To: time.Unix(300, 0)}

	require.Equal(t, []historicSyncRequest{first, second},
		coalesceHistoricSyncRequests([]historicSyncRequest{first, second}))
}

func TestEnqueueHistoricSyncConcurrent(t *testing.T) {
	m := &Messenger{
		logger:              zap.NewNop(),
		historicSyncTrigger: make(chan struct{}, 1),
	}
	from := time.Unix(100, 0)

	var wg sync.WaitGroup
	for i := 1; i <= 100; i++ {
		wg.Add(1)
		go func(offset int) {
			defer wg.Done()
			m.enqueueHistoricSync(historicSyncRequest{
				From: from,
				To:   from.Add(time.Duration(offset) * time.Second),
			})
		}(i)
	}
	wg.Wait()

	m.historicSyncQueueMu.Lock()
	defer m.historicSyncQueueMu.Unlock()
	require.Equal(t, []historicSyncRequest{{
		From: from,
		To:   from.Add(100 * time.Second),
	}}, m.historicSyncQueue)
}

func TestBoundedHistoricSyncWidenedUntilSessionCatchUp(t *testing.T) {
	m := &Messenger{logger: zap.NewNop()}
	bounded := historicSyncRequest{From: time.Unix(1_000, 0), To: time.Unix(1_001, 0)}

	widened := m.prepareHistoricSyncRun(bounded)
	require.Equal(t, historicSyncRequest{To: bounded.To}, widened,
		"before the session catch-up, a bounded window must run from the persisted cursors")

	m.recordHistoricSyncSuccess(bounded)
	require.False(t, m.historicCatchUpDone.Load(),
		"a bounded run must not count as the session catch-up")

	m.recordHistoricSyncSuccess(widened)
	require.True(t, m.historicCatchUpDone.Load())
	require.Equal(t, bounded, m.prepareHistoricSyncRun(bounded),
		"after the session catch-up, bounded windows run as requested")

	cursor := historicSyncRequest{To: time.Unix(2_000, 0)}
	require.Equal(t, cursor, (&Messenger{logger: zap.NewNop()}).prepareHistoricSyncRun(cursor))
}
