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
	}}, m.historicSyncPending)
}
