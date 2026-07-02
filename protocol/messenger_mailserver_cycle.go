package protocol

import (
	"time"

	"go.uber.org/zap"

	gocommon "github.com/status-im/status-go/common"
)

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
