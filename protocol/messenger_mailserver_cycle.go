package protocol

import (
	"sort"
	"time"

	"go.uber.org/zap"

	gocommon "github.com/status-im/status-go/common"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
)

const historyCursorMonitorInterval = time.Minute

// historicSyncNow returns the upper-bound timestamp for history sync requests,
// using the Messenger clock when available and wall time during initialization.
func (m *Messenger) historicSyncNow() time.Time {
	if m.messaging == nil {
		return time.Now()
	}
	return m.calculateMailserverTo()
}

// historicSyncRequest has a fixed upper bound so queueing and retries never
// expand a request into time during which delivery has become reliable.
// A zero From means "start from each topic's completeness cursor".
type historicSyncRequest struct {
	From time.Time
	To   time.Time
}

func (r historicSyncRequest) bounded() bool {
	return !r.From.IsZero()
}

func (r historicSyncRequest) valid() bool {
	return !r.To.IsZero() && (!r.bounded() || r.From.Before(r.To))
}

func historicSyncRequestsOverlap(a, b historicSyncRequest) bool {
	return a.bounded() && b.bounded() &&
		!a.To.Before(b.From) && !b.To.Before(a.From)
}

func translateHistoryReconcileWindow(window messagingtypes.HistoryReconcileWindow, observedNow, syncNow time.Time) messagingtypes.HistoryReconcileWindow {
	if window.From.IsZero() || window.To.IsZero() || !window.From.Before(window.To) {
		return messagingtypes.HistoryReconcileWindow{}
	}
	age := observedNow.Sub(window.To)
	if age < 0 {
		age = 0
	}
	to := syncNow.Add(-age)
	return messagingtypes.HistoryReconcileWindow{
		From: to.Add(-window.To.Sub(window.From)),
		To:   to,
	}
}

func coalesceHistoricSyncRequests(requests []historicSyncRequest) []historicSyncRequest {
	bounded := make([]historicSyncRequest, 0, len(requests))
	var cursorRequest historicSyncRequest
	for _, request := range requests {
		if request.bounded() {
			bounded = append(bounded, request)
			continue
		}
		if cursorRequest.To.IsZero() || request.To.After(cursorRequest.To) {
			cursorRequest = request
		}
	}

	sort.SliceStable(bounded, func(i, j int) bool {
		return bounded[i].From.Before(bounded[j].From)
	})
	merged := make([]historicSyncRequest, 0, len(bounded)+1)
	for _, request := range bounded {
		if len(merged) == 0 || !historicSyncRequestsOverlap(merged[len(merged)-1], request) {
			merged = append(merged, request)
			continue
		}
		if request.To.After(merged[len(merged)-1].To) {
			merged[len(merged)-1].To = request.To
		}
	}
	if !cursorRequest.To.IsZero() {
		kept := merged[:0]
		for _, request := range merged {
			if request.To.After(cursorRequest.To) {
				kept = append(kept, request)
			}
		}
		merged = kept
		merged = append(merged, cursorRequest)
	}
	sort.SliceStable(merged, func(i, j int) bool {
		return merged[i].To.Before(merged[j].To)
	})
	return merged
}

// startHistoryReconciliationLoop schedules a historic sync whenever the
// transport signals that reconciliation is needed — periodically while
// connectivity is not reliable, and once more when it recovers. This is the
// "unreliable" leg of the history-reconciliation policy (#7568), which the
// offline→online and app-foreground triggers do not cover: without it, a
// message that silently misses live delivery (degraded relay mesh) while the
// client stays online is never recovered (see status-app#21405). The
// transport owns the timing and the connectivity confidence; the fetch lives
// here, for now, because the per-topic completeness cursor is persisted by the
// Messenger in the app DB. This split is temporary until logos-delivery
// owns reconciliation and backfill end to end — see the note on
// Waku.OnHistoryReconcileNeeded.
func (m *Messenger) startHistoryReconciliationLoop() {
	if !m.config.codeControlFlags.AutoRequestHistoricMessages {
		return
	}

	m.shutdownWaitGroup.Add(1)
	go func() {
		defer gocommon.LogOnPanic()
		defer m.shutdownWaitGroup.Done()

		needed := m.messaging.OnHistoryReconcileNeeded()
		for {
			select {
			case <-m.quit:
				return
			case transportWindow := <-needed:
				window := translateHistoryReconcileWindow(
					transportWindow,
					time.Now(),
					m.historicSyncNow(),
				)
				if window.From.IsZero() {
					continue
				}
				m.advanceHistoryCursors(window.From)
				m.asyncRequestHistoricMessages(window)
			}
		}
	}()
}

// startHistoryCursorMonitor advances initialized topic cursors while
// a full node has a healthy relay mesh. This records reliable live delivery
// without issuing store queries and bounds catch-up after a crash.
func (m *Messenger) startHistoryCursorMonitor() {
	if !m.config.codeControlFlags.AutoRequestHistoricMessages {
		return
	}

	m.shutdownWaitGroup.Add(1)
	go func() {
		defer gocommon.LogOnPanic()
		defer m.shutdownWaitGroup.Done()

		ticker := time.NewTicker(historyCursorMonitorInterval)
		defer ticker.Stop()
		for {
			select {
			case <-m.quit:
				return
			case <-ticker.C:
				// Stay one tolerance window behind the observation so a
				// concurrent reliability transition cannot advance a cursor
				// into the newly unreliable interval.
				m.advanceHistoryCursors(
					m.historicSyncNow().Add(-time.Duration(tolerance) * time.Second),
				)
			}
		}
	}()
}

func (m *Messenger) advanceHistoryCursors(through time.Time) {
	if through.IsZero() || m.mailserversDatabase == nil || m.messaging == nil ||
		!m.messaging.HistoryDeliveryReliable() {
		return
	}
	m.historicSyncQueueMu.Lock()
	hasPending := len(m.historicSyncQueue) > 0
	m.historicSyncQueueMu.Unlock()
	if hasPending || m.historicSyncWorkerActive.Load() {
		return
	}
	m.historicSyncMu.Lock()
	defer m.historicSyncMu.Unlock()
	if m.historicSyncInFlight {
		return
	}
	if err := m.mailserversDatabase.AdvanceHistoryCursors(int(through.Unix())); err != nil {
		m.logger.Warn("failed to advance history cursors", zap.Error(err))
	}
}

// asyncRequestAllHistoricMessages schedules a cursor-based catch-up with a
// fixed upper bound. It is used at startup, where the preceding app-off period
// is a genuine delivery gap.
func (m *Messenger) asyncRequestAllHistoricMessages() {
	m.enqueueHistoricSync(historicSyncRequest{To: m.historicSyncNow()})
}

func (m *Messenger) asyncRequestHistoricMessages(window messagingtypes.HistoryReconcileWindow) {
	m.enqueueHistoricSync(historicSyncRequest{From: window.From, To: window.To})
}

func (m *Messenger) enqueueHistoricSync(request historicSyncRequest) {
	if !request.valid() {
		return
	}

	m.logger.Debug("enqueue historic sync",
		zap.Time("from", request.From),
		zap.Time("to", request.To))

	m.historicSyncQueueMu.Lock()
	m.historicSyncQueue = append(m.historicSyncQueue, request)
	m.historicSyncQueue = coalesceHistoricSyncRequests(m.historicSyncQueue)
	m.historicSyncQueueMu.Unlock()

	m.notifyHistoricSyncWorker()
}

func (m *Messenger) notifyHistoricSyncWorker() {
	select {
	case m.historicSyncTrigger <- struct{}{}:
	default:
	}
}

func (m *Messenger) takeHistoricSync() (historicSyncRequest, bool) {
	m.historicSyncQueueMu.Lock()
	defer m.historicSyncQueueMu.Unlock()
	if len(m.historicSyncQueue) == 0 {
		return historicSyncRequest{}, false
	}
	request := m.historicSyncQueue[0]
	m.historicSyncQueue = m.historicSyncQueue[1:]
	return request, true
}

func (m *Messenger) requeueHistoricSync(request historicSyncRequest) {
	m.historicSyncQueueMu.Lock()
	m.historicSyncQueue = append(m.historicSyncQueue, request)
	m.historicSyncQueue = coalesceHistoricSyncRequests(m.historicSyncQueue)
	m.historicSyncQueueMu.Unlock()
}

// startHistoricSyncWorker owns the execution of automatic historic-message
// syncs: it serializes them, enforces historicSyncMinInterval by waiting
// (never by dropping — triggers arriving meanwhile coalesce into the pending
// slot and are served by the next run), and retries failed syncs with backoff.
// The retry here is temporal, complementing the StoreClient's spatial
// failover: the StoreClient tries the candidate storenodes within one attempt
// and fails fast when none is reachable — which is the normal state right
// after login (freshly recreated libp2p host) or a connectivity change — while
// this loop decides when to try again.
func (m *Messenger) startHistoricSyncWorker() {
	if !m.config.codeControlFlags.AutoRequestHistoricMessages {
		return
	}

	m.shutdownWaitGroup.Add(1)
	go func() {
		defer gocommon.LogOnPanic()
		defer m.shutdownWaitGroup.Done()
		defer m.historicSyncWorkerActive.Store(false)

		var lastAttempt time.Time
		for {
			select {
			case <-m.quit:
				return
			case <-m.historicSyncTrigger:
			}

			for {
				if m.isPaused() {
					break
				}

				m.historicSyncWorkerActive.Store(true)
				request, ok := m.takeHistoricSync()
				if !ok {
					m.historicSyncWorkerActive.Store(false)
					break
				}

				if wait := historicSyncMinInterval - time.Since(lastAttempt); wait > 0 {
					select {
					case <-m.quit:
						return
					case <-time.After(wait):
					}
				}
				if m.isPaused() {
					m.requeueHistoricSync(request)
					m.historicSyncWorkerActive.Store(false)
					break
				}

				deadline := time.Now().Add(historicSyncRetryTimeout)
				deferred := false
				for {
					if m.isPaused() {
						m.requeueHistoricSync(request)
						deferred = true
						break
					}
					previousAttempt := lastAttempt
					lastAttempt = time.Now()
					executed, err := m.runAutomaticHistoricSync(request)
					if err == nil && executed {
						break
					}
					if err == nil {
						// A skipped run never touched a store node, so it must not
						// consume the rate-limit budget of the next real attempt.
						lastAttempt = previousAttempt
						// Offline and policy-blocked requests stay pending. Their
						// corresponding online/resume/setting transition wakes us.
						m.requeueHistoricSync(request)
						deferred = true
						break
					}
					if time.Now().After(deadline) {
						m.logger.Error("historic sync failed after retries", zap.Error(err))
						m.requeueHistoricSync(request)
						deferred = true
						break
					}
					m.logger.Warn("historic sync failed, retrying", zap.Error(err))
					select {
					case <-m.quit:
						return
					case <-time.After(historicSyncRetryInterval):
					}
				}

				if deferred {
					m.historicSyncWorkerActive.Store(false)
					break
				}
				m.historicSyncWorkerActive.Store(false)
			}
		}
	}()
}
