package wakuv2

import (
	"context"
	"time"

	"github.com/status-im/status-go/wakuv2/common"

	"go.uber.org/zap"
	"golang.org/x/exp/maps"

	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/subscription"
)

const (
	FilterEventPingResult = iota
	FilterEventSubscribeResult
	FilterEventUnsubscribeResult
)

const pingTimeout = 10 * time.Second

// Methods on FilterManager maintain filter peer health
//
// runFilterLoop is the main event loop
//
// Filter Install/Uninstall events are pushed onto eventChan
// Subscribe, UnsubscribeWithSubscription, IsSubscriptionAlive calls
// are invoked from goroutines and request results pushed onto eventChan
//
// filterSubs is the map of filter IDs to subscriptions

type FilterManager struct {
	ctx              context.Context
	filters          map[string]func() // map of filters to cancel fns
	isFilterSubAlive func(sub *subscription.SubscriptionDetails) error
	onNewEnvelopes   func(env *protocol.Envelope) error
	logger           *zap.Logger
	node             *filter.WakuFilterLightNode
}

func newFilterManager(ctx context.Context, logger *zap.Logger, onNewEnvelopes func(env *protocol.Envelope) error, node *filter.WakuFilterLightNode) *FilterManager {
	// This fn is being mocked in test
	mgr := new(FilterManager)
	mgr.ctx = ctx
	mgr.logger = logger
	mgr.onNewEnvelopes = onNewEnvelopes
	mgr.filters = make(map[string]func())
	mgr.node = node
	mgr.isFilterSubAlive = func(sub *subscription.SubscriptionDetails) error {
		ctx, cancel := context.WithTimeout(ctx, pingTimeout)
		defer cancel()
		return mgr.node.IsSubscriptionAlive(ctx, sub)
	}

	return mgr
}

func (mgr *FilterManager) filterLifecycle(filterID string, contentFilter protocol.ContentFilter, ctx context.Context) {
	// Use it to ping filter peer(s) periodically
	ticker := time.NewTicker(5 * time.Second)
	logger := mgr.logger.With(zap.String("filterID", filterID), zap.Any("contentFilter", contentFilter))

	var sub *subscription.SubscriptionDetails
	defer ticker.Stop()

	for {
		// Health check
		logger.Debug("filter lifecycle")
		if sub != nil {
			logger.Debug("filter health check")
			err := mgr.isFilterSubAlive(sub)
			if err != nil {
				logger.Debug("filter ping error", zap.Error(err))
				reqCtx, _ := context.WithTimeout(ctx, pingTimeout)
				_, err := mgr.node.UnsubscribeWithSubscription(reqCtx, sub)
				sub = nil
				if err != nil {
					logger.Debug("filter unsubscribe error", zap.Error(err))
				}
			} else {
				logger.Debug("filter health ok")
			}
		} else {
			logger.Debug("filter has no sub")
		}

		if sub == nil {
			reqCtx, _ := context.WithTimeout(ctx, pingTimeout)
			logger.Debug("filter try subscribe")
			subDetails, err := mgr.node.Subscribe(reqCtx, contentFilter, filter.WithAutomaticPeerSelection())
			if subDetails != nil && len(subDetails) > 0 {
				sub = subDetails[0]
				go mgr.runFilterSubscriptionLoop(ctx, sub)
			} else {
				logger.Debug("filter subscribe error", zap.Error(err))
			}
		}

		select {
		case <-ctx.Done():
			logger.Debug("filter removed")
			reqCtx, _ := context.WithTimeout(ctx, pingTimeout)
			_, err := mgr.node.UnsubscribeWithSubscription(reqCtx, sub)
			if err != nil {
				logger.Debug("filter unsubscribe error", zap.Error(err))
			}
			return
		case <-ticker.C:
			continue
		}
	}
}

func (mgr *FilterManager) addFilter(filterID string, f *common.Filter) {
	ctx, cancel := context.WithCancel(mgr.ctx)
	mgr.filters[filterID] = cancel
	contentFilter := mgr.buildContentFilter(f.PubsubTopic, f.ContentTopics)
	go mgr.filterLifecycle(filterID, contentFilter, ctx)

}
func (mgr *FilterManager) removeFilter(filterID string) {
	cancel, ok := mgr.filters[filterID]
	if ok {
		delete(mgr.filters, filterID)
		// close goroutine running filterLifecycle
		cancel()
	}
}

func (mgr *FilterManager) buildContentFilter(pubsubTopic string, contentTopicSet common.TopicSet) protocol.ContentFilter {
	contentTopics := make([]string, len(contentTopicSet))
	for i, ct := range maps.Keys(contentTopicSet) {
		contentTopics[i] = ct.ContentTopic()
	}

	return protocol.NewContentFilter(pubsubTopic, contentTopics...)
}

func (mgr *FilterManager) runFilterSubscriptionLoop(ctx context.Context, sub *subscription.SubscriptionDetails) {
	for {
		select {
		case <-ctx.Done():
			return
		case env, ok := <-sub.C:
			if ok {
				err := (mgr.onNewEnvelopes)(env)
				if err != nil {
					mgr.logger.Error("OnNewEnvelopes error", zap.Error(err))
				}
			} else {
				mgr.logger.Debug("filter sub is closed", zap.String("id", sub.ID))
				return
			}
		}
	}
}
