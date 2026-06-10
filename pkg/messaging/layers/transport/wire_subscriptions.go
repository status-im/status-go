package transport

import (
	"context"
	"sync"

	"go.uber.org/zap"

	wakuv2 "github.com/status-im/status-go/pkg/messaging/waku"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

// wireSubscriptions is the transport's owner of the backend's wire
// subscriptions. The transport accounts every filter returned by a
// FiltersManager mutation here: installed filters are counted per (pubsub,
// content) topic pair, with MessagingAPI.Subscribe called when a pair gains
// its first filter and MessagingAPI.Unsubscribe when its last filter is
// removed. Accounting is keyed by FilterID, so re-accounting a filter the
// manager merely returned again (its Load* methods are get-or-create) is a
// no-op, and releasing a filter that was never accounted — or already
// released — is too.
type wireSubscriptions struct {
	api    MessagingAPI // nil for offline transports (tests)
	logger *zap.Logger

	mu sync.Mutex
	// known records the pair each accounted FilterID is counted under;
	// releases use it rather than the passed filter's fields, so a filter
	// whose topics were mangled on a round-trip still releases correctly.
	known  map[string]types.TopicSubscription
	counts map[types.TopicSubscription]int
}

func newWireSubscriptions(api MessagingAPI, logger *zap.Logger) *wireSubscriptions {
	return &wireSubscriptions{
		api:    api,
		logger: logger.With(zap.Namespace("wireSubscriptions")),
		known:  make(map[string]types.TopicSubscription),
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

// account ensures the filters' topic pairs are counted and subscribed. A
// filter seen before under a different pair (LoadPublic may update a filter's
// pubsub topic in place) releases the old pair and acquires the new one.
func (w *wireSubscriptions) account(filters ...*Filter) error {
	if w.api == nil {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	for _, filter := range filters {
		pair := pairOf(filter)
		old, ok := w.known[filter.FilterID]
		if ok && old == pair {
			continue
		}
		if ok {
			delete(w.known, filter.FilterID)
			w.releasePair(old)
		}
		if err := w.acquirePair(pair); err != nil {
			return err
		}
		w.known[filter.FilterID] = pair
	}
	return nil
}

// release un-counts the filters' topic pairs and unsubscribes the pairs whose
// last filter is gone. Filters never accounted, or already released, are
// skipped.
func (w *wireSubscriptions) release(filters ...*Filter) {
	if w.api == nil {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	for _, filter := range filters {
		pair, ok := w.known[filter.FilterID]
		if !ok {
			continue
		}
		delete(w.known, filter.FilterID)
		w.releasePair(pair)
	}
}

// acquirePair counts a filter against the pair and subscribes it on the wire
// when it first appears. Called with w.mu held.
func (w *wireSubscriptions) acquirePair(pair types.TopicSubscription) error {
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

// releasePair un-counts a filter from the pair and unsubscribes it on the
// wire when its last filter is gone. Called with w.mu held.
func (w *wireSubscriptions) releasePair(pair types.TopicSubscription) {
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
