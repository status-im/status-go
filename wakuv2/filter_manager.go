package wakuv2

import (
	"context"
	"errors"
	"time"

	"github.com/status-im/status-go/wakuv2/common"

	"go.uber.org/zap"
	"golang.org/x/exp/maps"

	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/subscription"
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
	ctx            context.Context
	filters        map[string]func() // map of filters to cancel fns
	onNewEnvelopes func(env *protocol.Envelope) error
	logger         *zap.Logger
	node           *filter.WakuFilterLightNode
}

func newFilterManager(ctx context.Context, logger *zap.Logger, onNewEnvelopes func(env *protocol.Envelope) error, node *filter.WakuFilterLightNode) *FilterManager {
	// This fn is being mocked in test
	mgr := new(FilterManager)
	mgr.ctx = ctx
	mgr.logger = logger
	mgr.onNewEnvelopes = onNewEnvelopes
	mgr.filters = make(map[string]func())
	mgr.node = node
	return mgr
}
func (mgr *FilterManager) isFilterSubAlive(ctx context.Context, sub *subscription.SubscriptionDetails) error {
	if sub.Closed {
		return errors.New("sub closed")
	}
	ctx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	return mgr.node.IsSubscriptionAlive(ctx, sub)
}

func (mgr *FilterManager) filterLifecycle(ctx context.Context, filterID string, contentFilter protocol.ContentFilter) {
	logger := mgr.logger.With(zap.String("filterID", filterID), zap.Any("contentFilter", contentFilter))

	// Use it to ping filter peer(s) periodically
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// A single sub for this filter
	var sub *subscription.SubscriptionDetails

	for {
		logger.Debug("filter lifecycle started")
		if sub != nil {
			logger.Debug("filter health check")
			err := mgr.isFilterSubAlive(ctx, sub)
			if err != nil {
				logger.Debug("filter ping error", zap.Error(err))
				mgr.unsubscribe(ctx, logger, sub)
				sub = nil
			} else {
				logger.Debug("filter health ok")
			}
		} else {
			logger.Debug("filter has no sub")
		}

		if sub == nil {
			// We are here either when the filter has just been added,
			// or when ping failed
			logger.Debug("filter try subscribe")
			sub = mgr.subscribe(ctx, logger, contentFilter)
		}

		select {
		case <-ctx.Done():
			logger.Debug("filter lifecycle completed")
			if sub != nil {
				// Note use of parent FilterManager's context,
				// as in cases of filter removal the filter context will be Done
				mgr.unsubscribe(mgr.ctx, logger, sub)
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
	go mgr.filterLifecycle(ctx, filterID, contentFilter)

}
func (mgr *FilterManager) removeFilter(filterID string) {
	cancel, ok := mgr.filters[filterID]
	if ok {
		delete(mgr.filters, filterID)
		// close goroutine running filterLifecycle
		cancel()
	} else {
		mgr.logger.Debug("filter removal: lifecycle goroutine not found", zap.String("filterID", filterID))
	}
}

func (mgr *FilterManager) buildContentFilter(pubsubTopic string, contentTopicSet common.TopicSet) protocol.ContentFilter {
	contentTopics := make([]string, len(contentTopicSet))
	for i, ct := range maps.Keys(contentTopicSet) {
		contentTopics[i] = ct.ContentTopic()
	}

	return protocol.NewContentFilter(pubsubTopic, contentTopics...)
}

func (mgr *FilterManager) subscribe(ctx context.Context, logger *zap.Logger, contentFilter protocol.ContentFilter) *subscription.SubscriptionDetails {
	reqCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	subDetails, err := mgr.node.Subscribe(reqCtx, contentFilter, filter.WithAutomaticPeerSelection())
	if len(subDetails) > 0 {
		sub := subDetails[0]
		go mgr.runFilterSubscriptionLoop(ctx, sub)
		return sub
	}
	if err != nil {
		logger.Debug("filter subscribe error", zap.Error(err))
	}
	return nil
}

func (mgr *FilterManager) unsubscribe(ctx context.Context, logger *zap.Logger, sub *subscription.SubscriptionDetails) {
	reqCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	_, err := mgr.node.UnsubscribeWithSubscription(reqCtx, sub)
	if err != nil {
		logger.Debug("filter unsubscribe error", zap.Error(err))
	}
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
