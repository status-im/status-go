package transport

import (
	"context"
	"time"

	"go.uber.org/zap"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

// missingMessageReconcileInterval is how often the reconciler checks for messages
// the store has but we don't.
var missingMessageReconcileInterval = 30 * time.Second

// missingMessageReconcileDelay holds the reconcile window's upper bound back from
// now, so we don't query for messages that may still be propagating to the store.
const missingMessageReconcileDelay = 20 * time.Second

// reconcileMissingMessagesLoop periodically asks the store for messages on the
// listening topics that we don't already have, back-filling gaps that live
// delivery missed. It reads the listening filters directly (so there is no
// criteria setter) and tracks the reconciled window per pubsub topic; the
// StoreClient selects and pins a storenode per query, so this loop is unaware of
// storenode selection.
func (t *Transport) reconcileMissingMessagesLoop() {
	defer gocommon.LogOnPanic()

	ticker := time.NewTicker(missingMessageReconcileInterval)
	defer ticker.Stop()

	// lastReconciled records, per pubsub topic, the upper bound of the last
	// successfully reconciled window, so each tick only covers new time.
	lastReconciled := make(map[string]time.Time)

	for {
		select {
		case <-t.quit:
			return
		case <-ticker.C:
		}
		t.reconcileMissingMessages(lastReconciled)
	}
}

func (t *Transport) reconcileMissingMessages(lastReconciled map[string]time.Time) {
	// Group the listening, non-ephemeral filters by pubsub topic.
	contentTopicsByPubsub := make(map[string][]types.TopicType)
	for _, f := range t.filters.Filters() {
		if !f.Listen || f.Ephemeral {
			continue
		}
		contentTopicsByPubsub[f.PubsubTopic] = append(contentTopicsByPubsub[f.PubsubTopic], f.ContentTopic)
	}

	to := time.Now().Add(-missingMessageReconcileDelay)
	for pubsubTopic, contentTopics := range contentTopicsByPubsub {
		from, seen := lastReconciled[pubsubTopic]
		if !seen {
			// First sighting of this topic: start from now. Older history is the
			// job of explicit history sync, not the reconciler.
			from = to
		}
		if !to.After(from) {
			continue
		}

		err := t.waku.FetchMissingMessages(context.Background(), types.MailserverBatch{
			From:        from,
			To:          to,
			PubsubTopic: pubsubTopic,
			Topics:      contentTopics,
		})
		if err != nil {
			t.logger.Debug("failed to reconcile missing messages",
				zap.String("pubsubTopic", pubsubTopic), zap.Error(err))
			continue // don't advance the window — retry it next tick
		}
		lastReconciled[pubsubTopic] = to
	}
}
