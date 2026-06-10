package transport

import (
	"context"
	"sync"

	"go.uber.org/zap"

	wakuv2 "github.com/status-im/status-go/pkg/messaging/waku"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

// FilterSubscriptions is the transport's owner of the backend's wire
// subscriptions, which exist for light clients (the Waku Filter protocol) —
// relay backends treat the Subscribe/Unsubscribe calls as bookkeeping. The
// transport acquires every filter returned by a FiltersManager mutation here
// and releases every removed one: installed filters are counted per (pubsub,
// content) topic pair, with MessagingAPI.Subscribe called when a pair gains
// its first filter and MessagingAPI.Unsubscribe when its last filter is
// removed. Tracking is keyed by FilterID, so re-acquiring a filter the
// manager merely returned again (its Load* methods are get-or-create) is a
// no-op, and releasing a filter that was never acquired — or already
// released — is too.
type FilterSubscriptions struct {
	api    MessagingAPI // nil for offline transports (tests)
	logger *zap.Logger

	mu sync.Mutex
	// known records the pair each acquired FilterID is counted under;
	// releases use it rather than the passed filter's fields, so a filter
	// whose topics were mangled on a round-trip still releases correctly.
	known  map[string]types.TopicSubscription
	counts map[types.TopicSubscription]int
}

func newFilterSubscriptions(api MessagingAPI, logger *zap.Logger) *FilterSubscriptions {
	return &FilterSubscriptions{
		api:    api,
		logger: logger.With(zap.Namespace("FilterSubscriptions")),
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

// Acquire ensures the filters' topic pairs are counted and subscribed. A
// filter seen before under a different pair (LoadPublic may update a filter's
// pubsub topic in place) releases the old pair and acquires the new one.
func (s *FilterSubscriptions) Acquire(filters ...*Filter) error {
	if s.api == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, filter := range filters {
		pair := pairOf(filter)
		old, ok := s.known[filter.FilterID]
		if ok && old == pair {
			continue
		}
		if ok {
			delete(s.known, filter.FilterID)
			s.release(old)
		}
		if err := s.acquire(pair); err != nil {
			return err
		}
		s.known[filter.FilterID] = pair
	}
	return nil
}

// Release un-counts the filters' topic pairs and unsubscribes the pairs whose
// last filter is gone. Filters never acquired, or already released, are
// skipped.
func (s *FilterSubscriptions) Release(filters ...*Filter) {
	if s.api == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, filter := range filters {
		pair, ok := s.known[filter.FilterID]
		if !ok {
			continue
		}
		delete(s.known, filter.FilterID)
		s.release(pair)
	}
}

// acquire counts a filter against the pair and subscribes it on the wire
// when it first appears. Called with s.mu held.
func (s *FilterSubscriptions) acquire(pair types.TopicSubscription) error {
	s.counts[pair]++
	if s.counts[pair] > 1 {
		return nil
	}
	if err := s.api.Subscribe(context.Background(), pair); err != nil {
		delete(s.counts, pair)
		return err
	}
	return nil
}

// release un-counts a filter from the pair and unsubscribes it on the
// wire when its last filter is gone. Called with s.mu held.
func (s *FilterSubscriptions) release(pair types.TopicSubscription) {
	if s.counts[pair] == 0 {
		return
	}
	s.counts[pair]--
	if s.counts[pair] > 0 {
		return
	}
	delete(s.counts, pair)

	if err := s.api.Unsubscribe(context.Background(), pair); err != nil {
		s.logger.Warn("failed to remove wire subscription",
			zap.String("pubsubTopic", pair.PubsubTopic),
			zap.String("contentTopic", pair.ContentTopic.String()),
			zap.Error(err))
	}
}
