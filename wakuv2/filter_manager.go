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
	mgr.logger.Debug("adding filter", zap.String("filterID", filterID), zap.Stringer("contentFilter", contentFilter))

	if mgr.peersAvailable {
		go mgr.subscribeAndRunLoop(filterConfig{filterID, contentFilter})
	} else {
		mgr.logger.Debug("queuing filter as not online", zap.String("filterID", filterID), zap.Stringer("contentFilter", contentFilter))
		mgr.filterQueue <- filterConfig{filterID, contentFilter}
	}
}

func (mgr *FilterManager) subscribeAndRunLoop(f filterConfig) {
	ctx, cancel := context.WithCancel(mgr.ctx)
	config := api.FilterConfig{MaxPeers: mgr.cfg.MinPeersForFilter}

	sub, err := api.Subscribe(ctx, mgr.node, f.contentFilter, config, mgr.logger)
	mgr.Lock()
	mgr.filters[f.ID] = SubDetails{cancel, sub}
	mgr.Unlock()
	if err == nil {
		mgr.logger.Debug("subscription successful, running loop", zap.String("filterID", f.ID), zap.Stringer("contentFilter", f.contentFilter))
		mgr.runFilterSubscriptionLoop(sub)
	} else {
		mgr.logger.Debug("subscription fail, queuing it", zap.String("filterID", f.ID), zap.Stringer("contentFilter", f.contentFilter))
		mgr.filterQueue <- f
	}
}

func (mgr *FilterManager) onConnectionStatusChange(pubsubTopic string, newStatus bool) {
	mgr.logger.Debug("inside onConnectionStatusChange", zap.Bool("newStatus", newStatus),
		zap.Int("filtersCount", len(mgr.filters)), zap.Int("filterQueueLen", len(mgr.filterQueue)))
	//TODO: Move this logic to a regular loop which checks if peers are available and subscribes.
	if newStatus { //Online
		if len(mgr.filterQueue) > 0 {
			//Check if any filter subs are pending and subscribe them
			for filter := range mgr.filterQueue {
				mgr.logger.Debug("subscribing from filterQueue", zap.String("filterID", filter.ID), zap.Stringer("contentFilter", filter.contentFilter))
				go mgr.subscribeAndRunLoop(filter)
				if len(mgr.filterQueue) == 0 {
					mgr.logger.Debug("filter queue empty")
					break
				}
			}
		}
	} else if !newStatus && mgr.peersAvailable { //Offline
		mgr.logger.Info("going offline, removing all filter subscriptions")
		mgr.Lock()
		for filterID, subDetails := range mgr.filters {
			mgr.logger.Debug("unsubscribing filter", zap.String("filterID", filterID), zap.Stringer("contentFilter", subDetails.sub.ContentFilter))
			subDetails.sub.Unsubscribe()
			mgr.filterQueue <- filterConfig{filterID, subDetails.sub.ContentFilter}
		}
		mgr.Unlock()
	}
	mgr.peersAvailable = newStatus
}

func (mgr *FilterManager) removeFilter(filterID string) {
	mgr.Lock()
	defer mgr.Unlock()
	mgr.logger.Debug("removing filter", zap.String("filterID", filterID))

	subDetails, ok := mgr.filters[filterID]
	if ok {
		delete(mgr.filters, filterID)
		// close goroutine running runFilterSubscriptionLoop
		// this will also close api.Sub
		subDetails.cancel()
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

func (mgr *FilterManager) runFilterSubscriptionLoop(sub *api.Sub) {
	for {
		select {
		case <-mgr.ctx.Done():
			mgr.logger.Debug("subscription loop ended", zap.Stringer("contentFilter", sub.ContentFilter))
			return
		case env, ok := <-sub.DataCh:
			if ok {
				err := (mgr.onNewEnvelopes)(env)
				if err != nil {
					mgr.logger.Error("OnNewEnvelopes error", zap.Error(err))
				}
			} else {
				mgr.logger.Debug("filter sub is closed", zap.Any("cf", sub.ContentFilter))
				return
			}
		}
	}
}
