package protocol

import (
	"time"

	"go.uber.org/zap"

	gocommon "github.com/status-im/status-go/common"
)

// startHistoryReconciliationLoop fetches history from the store nodes whenever
// the transport signals that reconciliation is needed — periodically while
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

		// No isPaused check needed: the waku-side loop producing the signal is
		// itself suspended while the transport is paused.
		needed := m.messaging.OnHistoryReconcileNeeded()
		for {
			select {
			case <-m.quit:
				return
			case <-needed:
				m.logger.Debug("reconciling history with store nodes")
				if _, err := m.requestAllHistoricMessages(true, false); err != nil {
					m.logger.Warn("history reconciliation fetch failed", zap.Error(err))
				}
			}
		}
	}()
}

func (m *Messenger) asyncRequestAllHistoricMessages() {
	if !m.config.codeControlFlags.AutoRequestHistoricMessages {
		return
	}

	m.logger.Debug("asyncRequestAllHistoricMessages")

	go func() {
		defer gocommon.LogOnPanic()
		// On login the libp2p host is freshly (re)created and no storenode is
		// connected yet, so the first attempt can fail with "no store node
		// reachable" before a storenode is dialable. The go-waku storenode cycle
		// that used to retrigger the fetch is gone, so retry with backoff until a
		// storenode serves the query (or we give up). Once one attempt succeeds
		// it arms the throttle, so concurrent/later triggers no-op.
		deadline := time.Now().Add(historicSyncRetryTimeout)
		for {
			_, err := m.requestAllHistoricMessages(true, false)
			if err == nil {
				return
			}
			if time.Now().After(deadline) {
				m.logger.Error("failed to request historic messages after retries", zap.Error(err))
				return
			}
			m.logger.Warn("failed to request historic messages, retrying", zap.Error(err))
			select {
			case <-m.quit:
				return
			case <-time.After(historicSyncRetryInterval):
			}
		}
	}()
}
