package protocol

import (
	"time"

	"go.uber.org/zap"

	gocommon "github.com/status-im/status-go/common"
)

// startHistoryReconciliationLoop schedules a historic sync whenever the
// transport signals that reconciliation is needed — periodically while
// connectivity is not reliable, and once more when it recovers. This is the
// "unreliable" leg of the history-reconciliation policy (#7568), which the
// offline→online and app-foreground triggers do not cover: without it, a
// message that silently misses live delivery (degraded relay mesh) while the
// client stays online is never recovered (see status-app#21405). The
// transport owns the timing and the connectivity confidence; the fetch lives
// here, for now, because the per-topic lastFetched watermark is persisted by
// the Messenger in the app DB. This split is temporary until logos-delivery
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
			case <-needed:
				m.asyncRequestAllHistoricMessages()
			}
		}
	}()
}

// asyncRequestAllHistoricMessages schedules a historic-message sync on the
// historic-sync worker (startHistoricSyncWorker). Non-blocking; concurrent
// triggers coalesce into a single pending sync and are never silently
// dropped — the worker spaces, serializes and retries them.
func (m *Messenger) asyncRequestAllHistoricMessages() {
	m.logger.Debug("asyncRequestAllHistoricMessages")
	select {
	case m.historicSyncTrigger <- struct{}{}:
	default: // a sync is already pending; it will cover this trigger too
	}
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

		var lastAttempt time.Time
		for {
			select {
			case <-m.quit:
				return
			case <-m.historicSyncTrigger:
			}

			if m.isPaused() {
				// Dropped, not deferred: returning to foreground schedules its own
				// sync (SetPaused(false) → asyncRequestAllHistoricMessages).
				continue
			}

			if wait := historicSyncMinInterval - time.Since(lastAttempt); wait > 0 {
				select {
				case <-m.quit:
					return
				case <-time.After(wait):
				}
			}

			deadline := time.Now().Add(historicSyncRetryTimeout)
			for {
				lastAttempt = time.Now()
				_, err := m.requestAllHistoricMessages(false)
				if err == nil {
					break
				}
				if time.Now().After(deadline) {
					m.logger.Error("historic sync failed after retries", zap.Error(err))
					break
				}
				m.logger.Warn("historic sync failed, retrying", zap.Error(err))
				select {
				case <-m.quit:
					return
				case <-time.After(historicSyncRetryInterval):
				}
			}
		}
	}()
}
