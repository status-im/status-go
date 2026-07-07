package wakuv2

import (
	"time"

	"go.uber.org/zap"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

const (
	// History reconciliation (#7568): while the node cannot be confident it is
	// receiving everything live, the consumer should periodically fetch history
	// from the store nodes so silently missed messages are recovered. The node
	// is confident only with a Connected relay mesh on every default shard.
	historyReconcileCheckInterval = 10 * time.Second
	historyReconcileMinInterval   = 30 * time.Second
)

// OnHistoryReconcileNeeded returns a channel signalled whenever history should
// be reconciled with the store nodes: periodically while connectivity is not
// reliable, and once more when it recovers (closing the unreliable window).
// It is a buffered level-trigger; consumers that are slow or paused coalesce
// pending signals instead of queueing them.
//
// Temporary: this channel exists because the Waku node cannot yet own the
// fetch itself. It already has the subscription (filter) list, but lacks the
// per-topic "when was it last fetched" watermark, which the Messenger persists
// in the app DB (mailserver_topics). Eventually logos-delivery will own
// reconciliation and history backfill entirely, including persisting
// lastFetched per topic: its Messaging API already reconciles
// (https://github.com/logos-messaging/logos-delivery/issues/3941) but does not
// yet fetch history, and neither exposes nor persists lastFetched. That gap
// should be closed in logos-delivery before integration — otherwise ownership
// is split and persistence breaks, since we cannot persist lastFetched on its
// behalf. Until then the fetch stays in the Messenger (Transport would be a
// nicer interim home, but it is a stopgap either way), and this signal bridges
// the two.
func (w *Waku) OnHistoryReconcileNeeded() <-chan struct{} {
	return w.historyReconcileNeeded
}

// startHistoryReconcileLoop runs the timing/decision half of history
// reconciliation. The fetch itself lives with the consumer (the protocol
// layer), which owns the topics and per-chat watermarks; this loop only owns
// connectivity confidence and cadence. Suspended while the node is paused
// (app backgrounded): SetPaused(false) triggers its own fetch on foreground.
func (w *Waku) startHistoryReconcileLoop() {
	w.wg.Add(1)
	go func() {
		defer gocommon.LogOnPanic()
		defer w.wg.Done()
		ticker := time.NewTicker(historyReconcileCheckInterval)
		defer ticker.Stop()
		sub := w.PauseBroadcaster.Subscribe()
		defer sub.Unsubscribe()
		paused := <-sub.C()
		var tickerC <-chan time.Time
		if !paused {
			tickerC = ticker.C
		}

		reliable := w.reliablyConnected()
		var lastReconcile time.Time

		for {
			select {
			case <-w.ctx.Done():
				return
			case pausedState, ok := <-sub.C():
				if !ok {
					return
				}
				paused = pausedState
				if paused {
					tickerC = nil
				} else {
					tickerC = ticker.C
				}
			case <-tickerC:
				wasReliable := reliable
				reliable = w.reliablyConnected()
				if !shouldReconcileHistory(reliable, wasReliable, lastReconcile, time.Now(), historyReconcileMinInterval) {
					continue
				}
				lastReconcile = time.Now()
				w.logger.Debug("history reconciliation needed", zap.Bool("reliable", reliable))
				select {
				case w.historyReconcileNeeded <- struct{}{}:
				default: // a signal is already pending; coalesce
				}
			}
		}
	}()
}

// reliablyConnected reports whether connectivity is currently good enough to be
// confident no live messages are being missed: a Connected relay mesh on every
// default shard. Note that light (Edge) nodes report Connected on any
// connectivity even though their filter-subscription health is not observable
// yet (see deriveConnectionState); whether they should instead always
// reconcile is a separate policy decision, deliberately not made here.
func (w *Waku) reliablyConnected() bool {
	return w.ConnectionState() == types.ConnectionStateConnected
}

// shouldReconcileHistory decides whether a reconciliation is due at a tick:
// when the connection just recovered (an unreliable window closed, fetch what
// it may have missed), or while it stays unreliable and minInterval has passed
// since the last reconcile. While reliably connected, never.
func shouldReconcileHistory(reliable, wasReliable bool, lastReconcile, now time.Time, minInterval time.Duration) bool {
	if reliable {
		return !wasReliable
	}
	return now.Sub(lastReconcile) >= minInterval
}
