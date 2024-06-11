package wakuv2

import (
	"context"
	"sync"

	"github.com/status-im/status-go/wakuv2/common"

	"go.uber.org/zap"
	"golang.org/x/exp/maps"

	"github.com/waku-org/go-waku/waku/v2/api"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
)

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
	sync.Mutex
	ctx            context.Context
	cfg            *Config
	filters        map[string]SubDetails // map of filters to apiSub details
	onNewEnvelopes func(env *protocol.Envelope) error
	logger         *zap.Logger
	node           *filter.WakuFilterLightNode
	peersAvailable bool
	filterQueue    chan filterConfig
}
type SubDetails struct {
	cancel func()
	sub    *api.Sub
}

const filterQueueSize = 1000

type filterConfig struct {
	ID            string
	contentFilter protocol.ContentFilter
}

func newFilterManager(ctx context.Context, logger *zap.Logger, cfg *Config, onNewEnvelopes func(env *protocol.Envelope) error, node *filter.WakuFilterLightNode) *FilterManager {
	// This fn is being mocked in test
	mgr := new(FilterManager)
	mgr.ctx = ctx
	mgr.logger = logger
	mgr.cfg = cfg
	mgr.onNewEnvelopes = onNewEnvelopes
	mgr.filters = make(map[string]SubDetails)
	mgr.node = node
	mgr.peersAvailable = false
	mgr.filterQueue = make(chan filterConfig, filterQueueSize)

	return mgr
}

func (mgr *FilterManager) addFilter(filterID string, f *common.Filter) {
	mgr.Lock()
	defer mgr.Unlock()
	contentFilter := mgr.buildContentFilter(f.PubsubTopic, f.ContentTopics)
	mgr.logger.Debug("adding filter", zap.String("filter-id", filterID), zap.Stringer("content-filter", contentFilter))

	if mgr.peersAvailable {
		go mgr.subscribeAndRunLoop(filterConfig{filterID, contentFilter})
	} else {
		mgr.logger.Debug("queuing filter as not online", zap.String("filter-id", filterID), zap.Stringer("content-filter", contentFilter))
		mgr.filterQueue <- filterConfig{filterID, contentFilter}
	}
}

func (mgr *FilterManager) subscribeAndRunLoop(f filterConfig) {
	ctx, cancel := context.WithCancel(mgr.ctx)
	config := api.FilterConfig{MaxPeers: mgr.cfg.MinPeersForFilter}

	sub, err := api.Subscribe(ctx, mgr.node, f.contentFilter, config, mgr.logger, mgr.peersAvailable)
	mgr.Lock()
	mgr.filters[f.ID] = SubDetails{cancel, sub}
	mgr.Unlock()
	if err == nil {
		mgr.logger.Debug("subscription successful, running loop", zap.String("filter-id", f.ID), zap.Stringer("content-filter", f.contentFilter))
		mgr.runFilterSubscriptionLoop(sub)
	} else {
		mgr.logger.Error("subscription fail, need to debug issue", zap.String("filter-id", f.ID), zap.Stringer("content-filter", f.contentFilter), zap.Error(err))
	}
}

func (mgr *FilterManager) onConnectionStatusChange(pubsubTopic string, newStatus bool) {
	mgr.logger.Debug("inside on connection status change", zap.Bool("new-status", newStatus),
		zap.Int("filters count", len(mgr.filters)), zap.Int("filter-queue-len", len(mgr.filterQueue)))
	//TODO: Needs optimization because only on transition from offline to online should trigger this logic.
	if newStatus { //Online
		if len(mgr.filterQueue) > 0 {
			//Check if any filter subs are pending and subscribe them
			for filter := range mgr.filterQueue {
				mgr.logger.Debug("subscribing from filter queue", zap.String("filter-id", filter.ID), zap.Stringer("content-filter", filter.contentFilter))
				go mgr.subscribeAndRunLoop(filter)
				if len(mgr.filterQueue) == 0 {
					mgr.logger.Debug("filter queue empty")
					break
				}
			}
		}
	}
	//if !newStatus && mgr.peersAvailable || newStatus && !mgr.peersAvailable { //Offline or online
	//mgr.logger.Info("node status change, notifying all filter subscriptions", zap.Bool("new-status", newStatus))
	mgr.Lock()
	for _, subDetails := range mgr.filters {
		subDetails.sub.SetNodeState(newStatus)
	}
	mgr.Unlock()
	//}
	mgr.peersAvailable = newStatus
}

func (mgr *FilterManager) removeFilter(filterID string) {
	mgr.Lock()
	defer mgr.Unlock()
	mgr.logger.Debug("removing filter", zap.String("filter-id", filterID))

	subDetails, ok := mgr.filters[filterID]
	if ok {
		delete(mgr.filters, filterID)
		// close goroutine running runFilterSubscriptionLoop
		// this will also close api.Sub
		subDetails.cancel()
	} else {
		mgr.logger.Debug("filter removal: filter not found", zap.String("filter-id", filterID))
	}
}

func (mgr *FilterManager) buildContentFilter(pubsubTopic string, contentTopicSet common.TopicSet) protocol.ContentFilter {
	contentTopics := make([]string, len(contentTopicSet))
	for i, ct := range maps.Keys(contentTopicSet) {
		contentTopics[i] = ct.ContentTopic()
	}

	return protocol.NewContentFilter(pubsubTopic, contentTopics...)
}

func (mgr *FilterManager) runFilterSubscriptionLoop(sub *api.Sub) {
	for {
		select {
		case <-mgr.ctx.Done():
			mgr.logger.Debug("subscription loop ended", zap.Stringer("content-filter", sub.ContentFilter))
			return
		case env, ok := <-sub.DataCh:
			if ok {
				err := (mgr.onNewEnvelopes)(env)
				if err != nil {
					mgr.logger.Error("invoking onNewEnvelopes error", zap.Error(err))
				}
			} else {
				mgr.logger.Debug("filter sub is closed", zap.Any("content-filter", sub.ContentFilter))
				return
			}
		}
	}
}
