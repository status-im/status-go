package transport

import (
	"context"
	"sync"

	"go.uber.org/zap"

	wakuv2 "github.com/status-im/status-go/pkg/messaging/waku"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

// wireSubscriptions is the transport's owner of the backend's wire
// subscriptions. It observes the FiltersManager's filter lifecycle and counts
// installed filters per (pubsub, content) topic pair, calling
// MessagingAPI.Subscribe when a pair gains its first filter and
// MessagingAPI.Unsubscribe when its last filter is removed. This is the only
// place wire subscriptions are driven from — the FiltersManager itself knows
// nothing about the wire.
type wireSubscriptions struct {
	api    MessagingAPI // nil for offline transports (tests)
	logger *zap.Logger

	mu     sync.Mutex
	counts map[types.TopicSubscription]int
}

func newWireSubscriptions(api MessagingAPI, logger *zap.Logger) *wireSubscriptions {
	return &wireSubscriptions{
		api:    api,
		logger: logger.With(zap.Namespace("wireSubscriptions")),
		counts: make(map[types.TopicSubscription]int),
	}
}

// pairOf returns the wire-subscription pair a filter listens on, normalized
// the same way received messages are routed (WatchersByTopic): an empty
// pubsub topic means the default shard.
func pairOf(filter *Filter) types.TopicSubscription {
	pubsubTopic := filter.PubsubTopic
	if pubsubTopic == "" {
		pubsubTopic = wakuv2.DefaultShardPubsubTopic()
	}
	return types.TopicSubscription{PubsubTopic: pubsubTopic, ContentTopic: filter.ContentTopic}
}

// OnFilterAdded counts an installed filter against its topic pair and
// subscribes the pair on the wire when it first appears.
func (w *wireSubscriptions) OnFilterAdded(filter *Filter) error {
	if w.api == nil {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	pair := pairOf(filter)
	w.counts[pair]++
	if w.counts[pair] > 1 {
		return nil
	}

	if err := w.api.Subscribe(context.Background(), pair); err != nil {
		delete(w.counts, pair)
		return err
	}
	return nil
}

// OnFilterRemoved un-counts a removed filter from its topic pair and
// unsubscribes the pair on the wire when its last filter is gone.
func (w *wireSubscriptions) OnFilterRemoved(filter *Filter) {
	if w.api == nil {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	pair := pairOf(filter)
	if w.counts[pair] == 0 {
		return
	}
	w.counts[pair]--
	if w.counts[pair] > 0 {
		return
	}
	delete(w.counts, pair)

	if err := w.api.Unsubscribe(context.Background(), pair); err != nil {
		w.logger.Warn("failed to remove wire subscription",
			zap.String("pubsubTopic", pair.PubsubTopic),
			zap.String("contentTopic", pair.ContentTopic.String()),
			zap.Error(err))
	}
}
