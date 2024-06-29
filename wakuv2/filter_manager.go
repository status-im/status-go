package wakuv2

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/status-im/status-go/wakuv2/common"

	"go.uber.org/zap"
	"golang.org/x/exp/maps"

	"github.com/waku-org/go-waku/waku/v2/api"
	"github.com/waku-org/go-waku/waku/v2/onlinechecker"
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

const filterSubBatchSize = 90

type appFilterMap map[string]filterConfig

type FilterManager struct {
	sync.Mutex
	ctx                      context.Context
	cfg                      *Config
	onlineChecker            *onlinechecker.DefaultOnlineChecker
	aggFilters               map[string]SubDetails // map of aggregated filters to apiSub details
	onNewEnvelopes           func(env *protocol.Envelope) error
	logger                   *zap.Logger
	node                     *filter.WakuFilterLightNode
	filterSubBatchDuration   time.Duration
	topicBasedAggFilterQueue map[string]filterConfig
	appFilters               appFilterMap //Map of application filterID to {aggregatedFilterID, application ContentFilter}
	filterWaitQueue          chan filterConfig
}

type SubDetails struct {
	cancel func()
	sub    *api.Sub
}

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
	mgr.aggFilters = make(map[string]SubDetails)
	mgr.node = node
	mgr.onlineChecker = onlinechecker.NewDefaultOnlineChecker(false).(*onlinechecker.DefaultOnlineChecker)
	mgr.node.SetOnlineChecker(mgr.onlineChecker)
	mgr.filterSubBatchDuration = 5 * time.Second
	mgr.topicBasedAggFilterQueue = make(map[string]filterConfig)
	mgr.appFilters = make(appFilterMap)
	mgr.filterWaitQueue = make(chan filterConfig, 100)
	go mgr.startFilterSubLoop()
	return mgr
}

func (mgr *FilterManager) startFilterSubLoop() {
	ticker := time.NewTicker(mgr.filterSubBatchDuration)
	defer ticker.Stop()
	for {
		select {
		case <-mgr.ctx.Done():
			return
		case <-ticker.C:
			//TODO: Optimization, handle case where 1st addFilter happens just before ticker expires.
			if mgr.onlineChecker.IsOnline() {
				mgr.Lock()
				for _, af := range mgr.topicBasedAggFilterQueue {
					mgr.logger.Debug("ticker hit, hence subscribing", zap.String("agg-filter-id", af.ID), zap.Int("batch-size", len(af.contentFilter.ContentTopics)),
						zap.Stringer("agg-content-filter", af.contentFilter))
					go mgr.subscribeAndRunLoop(af)
				}
				mgr.topicBasedAggFilterQueue = make(map[string]filterConfig)
				mgr.Unlock()
			}
		}
	}
}

func (mgr *FilterManager) addFilter(filterID string, f *common.Filter) {
	mgr.logger.Debug("adding filter", zap.String("filter-id", filterID)) //, zap.Strings("content-topics", maps.Keys(f.ContentTopics)))

	mgr.Lock()
	defer mgr.Unlock()

	afilter, ok := mgr.topicBasedAggFilterQueue[f.PubsubTopic]
	if !ok {
		mgr.logger.Debug("new pubsubTopic batch", zap.String("topic", f.PubsubTopic))
		cf := mgr.buildContentFilter(f.PubsubTopic, f.ContentTopics)
		afilter = filterConfig{uuid.NewString(), cf}
		mgr.topicBasedAggFilterQueue[f.PubsubTopic] = afilter
		mgr.appFilters[filterID] = filterConfig{afilter.ID, cf}
	} else {
		mgr.logger.Debug("existing pubsubTopic", zap.String("agg-filter-id", afilter.ID), zap.String("topic", f.PubsubTopic))
		if len(afilter.contentFilter.ContentTopics)+len(f.ContentTopics) > filterSubBatchSize {
			if mgr.onlineChecker.IsOnline() {
				mgr.logger.Debug("crossed batchsize, hence subscribing", zap.String("agg-filter-id", afilter.ID), zap.String("topic", f.PubsubTopic), zap.Int("batch-size", len(afilter.contentFilter.ContentTopics)+len(f.ContentTopics)))
				go mgr.subscribeAndRunLoop(afilter)
			} else {
				mgr.logger.Debug("crossed batchsize, queuing since offline", zap.String("agg-filter-id", afilter.ID), zap.String("topic", f.PubsubTopic), zap.Int("batch-size", len(afilter.contentFilter.ContentTopics)+len(f.ContentTopics)))
				//queue existing batch
				mgr.filterWaitQueue <- afilter
			}
			cf := mgr.buildContentFilter(f.PubsubTopic, f.ContentTopics)
			afilter = filterConfig{uuid.NewString(), cf}
			mgr.logger.Debug("adding to new pubsubTopic batch", zap.String("agg-filter-id", afilter.ID), zap.String("topic", f.PubsubTopic), zap.Stringer("content-filter", cf))
			mgr.topicBasedAggFilterQueue[f.PubsubTopic] = afilter
			mgr.appFilters[filterID] = filterConfig{afilter.ID, cf}
		} else {
			var contentTopics []string
			for _, ct := range maps.Keys(f.ContentTopics) {
				afilter.contentFilter.ContentTopics[ct.ContentTopic()] = struct{}{}
				contentTopics = append(contentTopics, ct.ContentTopic())
			}
			cf := protocol.NewContentFilter(f.PubsubTopic, contentTopics...)
			mgr.logger.Debug("adding to existing batch", zap.String("agg-filter-id", afilter.ID), zap.Stringer("content-filter", cf), zap.Int("batch-size", len(afilter.contentFilter.ContentTopics)))
			mgr.appFilters[filterID] = filterConfig{afilter.ID, cf}
		}
	}
}

func (mgr *FilterManager) subscribeAndRunLoop(f filterConfig) {
	ctx, cancel := context.WithCancel(mgr.ctx)
	config := api.FilterConfig{MaxPeers: mgr.cfg.MinPeersForFilter}
	sub, err := api.Subscribe(ctx, mgr.node, f.contentFilter, config, mgr.logger)
	mgr.Lock()
	mgr.aggFilters[f.ID] = SubDetails{cancel, sub}
	mgr.Unlock()
	if err == nil {
		mgr.logger.Debug("subscription successful, running loop", zap.String("agg-filter-id", f.ID), zap.Stringer("content-filter", f.contentFilter))
		mgr.runFilterSubscriptionLoop(sub)
	} else {
		mgr.logger.Error("subscription fail, need to debug issue", zap.String("agg-filter-id", f.ID), zap.Stringer("content-filter", f.contentFilter), zap.Error(err))
	}
}

func (mgr *FilterManager) onConnectionStatusChange(pubsubTopic string, newStatus bool) {
	mgr.logger.Debug("inside on connection status change", zap.Bool("new-status", newStatus),
		zap.Int("agg filters count", len(mgr.aggFilters)))
	if newStatus && !mgr.onlineChecker.IsOnline() { //switched from offline to Online
		mgr.logger.Debug("switching from offline to online")
		mgr.Lock()
		if len(mgr.filterWaitQueue) > 0 {
			for af := range mgr.filterWaitQueue {
				if pubsubTopic == "" || pubsubTopic == af.contentFilter.PubsubTopic {
					//Check if any filter subs are pending and subscribe them
					mgr.logger.Debug("subscribing from filter queue", zap.String("filter-id", af.ID), zap.Stringer("content-filter", af.contentFilter))
					go mgr.subscribeAndRunLoop(af)
				} else {
					//TODO: Can this cause issues?
					mgr.filterWaitQueue <- af
				}
				if len(mgr.filterWaitQueue) == 0 {
					mgr.logger.Debug("no pending subscriptions")
					break
				}
			}
		}
		mgr.Unlock()
	}

	mgr.onlineChecker.SetOnline(newStatus)
}

func (mgr *FilterManager) removeFilter(filterID string) {
	mgr.Lock()
	defer mgr.Unlock()
	mgr.logger.Debug("removing filter", zap.String("filter-id", filterID))
	filterConfig, ok := mgr.appFilters[filterID]
	if !ok {
		mgr.logger.Debug("filter removal: filter not found", zap.String("filter-id", filterID))
		return
	}
	af, ok := mgr.aggFilters[filterConfig.ID]
	if ok {
		delete(mgr.appFilters, filterID)
		for ct := range filterConfig.contentFilter.ContentTopics {
			delete(af.sub.ContentFilter.ContentTopics, ct)
		}
		if len(af.sub.ContentFilter.ContentTopics) == 0 {
			mgr.aggFilters[filterConfig.ID].cancel()
		} else {
			go mgr.aggFilters[filterConfig.ID].sub.Unsubscribe(filterConfig.contentFilter)
		}
	} else {
		mgr.logger.Debug("filter removal: aggregated filter not found", zap.String("filter-id", filterID), zap.String("agg-filter-id", filterConfig.ID))
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
