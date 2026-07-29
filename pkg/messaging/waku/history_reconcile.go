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

type historyReconcileTracker struct {
	reliable       bool
	checkedAt      time.Time
	unreliableFrom time.Time
	lastReconcile  time.Time
}

func newHistoryReconcileTracker(reliable bool, now time.Time) historyReconcileTracker {
	tracker := historyReconcileTracker{reliable: reliable, checkedAt: now}
	if !reliable {
		tracker.unreliableFrom = now
	}
	return tracker
}

func (t *historyReconcileTracker) observe(reliable bool, now time.Time, minInterval time.Duration) (types.HistoryReconcileWindow, bool) {
	wasReliable := t.reliable
	t.reliable = reliable
	if wasReliable && !reliable {
		t.unreliableFrom = t.checkedAt
	}
	t.checkedAt = now

	if !shouldReconcileHistory(reliable, wasReliable, t.lastReconcile, now, minInterval) {
		return types.HistoryReconcileWindow{}, false
	}
	if t.unreliableFrom.IsZero() {
		t.unreliableFrom = now
	}
	t.lastReconcile = now
	window := types.HistoryReconcileWindow{From: t.unreliableFrom, To: now}
	if reliable {
		t.unreliableFrom = time.Time{}
	}
	return window, true
}

// OnHistoryReconcileNeeded returns unreliable delivery windows that should be
// reconciled with the store nodes: periodically while connectivity is not
// reliable, and once more when it recovers (closing the unreliable window).
//
// Temporary: this channel exists because the Waku node cannot yet own the
// fetch itself. It already has the subscription (filter) list, but lacks the
// per-topic "known complete through" cursor, which the Messenger persists in
// the app DB (mailserver_topics). Eventually logos-delivery will own
// reconciliation and history backfill entirely, including persisting that
// cursor per topic: its Messaging API already reconciles
// (https://github.com/logos-messaging/logos-delivery/issues/3941) but does not
// yet fetch history, and neither exposes nor persists a completeness cursor.
// That gap
// should be closed in logos-delivery before integration — otherwise ownership
// is split and persistence breaks, since we cannot persist the cursor on its
// behalf. Until then the fetch stays in the Messenger (Transport would be a
// nicer interim home, but it is a stopgap either way), and this signal bridges
// the two.
func (w *Waku) OnHistoryReconcileNeeded() <-chan types.HistoryReconcileWindow {
	return w.historyReconcileNeeded
}

// startHistoryReconcileLoop runs the timing/decision half of history
// reconciliation. The fetch itself lives with the consumer (the protocol
// layer), which owns the topics and per-chat watermarks; this loop only owns
// connectivity confidence and cadence. Suspended while the node is paused
// (app backgrounded, no ticker armed): SetPaused(false) triggers its own
// fetch on foreground.
func (w *Waku) startHistoryReconcileLoop() {
	w.wg.Add(1)
	go func() {
		defer gocommon.LogOnPanic()
		defer w.wg.Done()

		sub := w.PauseBroadcaster.Subscribe()
		defer sub.Unsubscribe()

		tracker := newHistoryReconcileTracker(w.reliablyConnected(), time.Now())

		pt := gocommon.NewPausableTicker(gocommon.PausableTickerConfig{
			Interval: historyReconcileCheckInterval,
			OnTick: func() {
				now := time.Now()
				reliable := w.reliablyConnected()
				window, needed := tracker.observe(reliable, now, historyReconcileMinInterval)
				if !needed {
					return
				}
				w.logger.Debug("history reconciliation needed",
					zap.Bool("reliable", reliable),
					zap.Time("from", window.From),
					zap.Time("to", window.To))
				select {
				case w.historyReconcileNeeded <- window:
				case <-w.ctx.Done():
				}
			},
		}, sub.C())
		pt.Run(w.ctx.Done())
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

// HistoryDeliveryReliable is stricter than reliablyConnected for light nodes:
// their filter-subscription health is not observable, so Connected alone must
// not advance persisted history completeness cursors.
func (w *Waku) HistoryDeliveryReliable() bool {
	return !w.cfg.IsLightClient() && w.reliablyConnected()
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
