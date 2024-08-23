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

// Methods on FilterManager just aggregate filters from application and subscribe to them
//
// startFilterSubLoop runs a loop where-in it waits for an interval to batch subscriptions
//
// runFilterSubscriptionLoop runs a loop for receiving messages from  underlying subscriptions and invokes onNewEnvelopes
//
// filterConfigs is the map of filer IDs to filter configs
// filterSubscriptions is the map of filter subscription IDs to subscriptions

const filterSubBatchSize = 90

type appFilterMap map[string]filterConfig

type FilterManager struct {
	sync.Mutex
	ctx                    context.Context
	cfg                    *Config
	onlineChecker          *onlinechecker.DefaultOnlineChecker
	filterSubscriptions    map[string]SubDetails // map of aggregated filters to apiSub details
	onNewEnvelopes         func(env *protocol.Envelope) error
	logger                 *zap.Logger
	node                   *filter.WakuFilterLightNode
	filterSubBatchDuration time.Duration
	incompleteFilterBatch  map[string]filterConfig
	filterConfigs          appFilterMap // map of application filterID to {aggregatedFilterID, application ContentFilter}
	waitingToSubQueue      chan filterConfig
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
	mgr.filterSubscriptions = make(map[string]SubDetails)
	mgr.node = node
	mgr.onlineChecker = onlinechecker.NewDefaultOnlineChecker(false).(*onlinechecker.DefaultOnlineChecker)
	mgr.node.SetOnlineChecker(mgr.onlineChecker)
	mgr.filterSubBatchDuration = 300 * time.Millisecond
	mgr.incompleteFilterBatch = make(map[string]filterConfig)
	mgr.filterConfigs = make(appFilterMap)
	mgr.waitingToSubQueue = make(chan filterConfig, 100)
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
			// TODO: Optimization, handle case where 1st addFilter happens just before ticker expires.
			if mgr.onlineChecker.IsOnline() {
				mgr.Lock()
				for _, af := range mgr.incompleteFilterBatch {
					mgr.logger.Debug("ticker hit, hence subscribing", zap.String("agg-filter-id", af.ID), zap.Int("batch-size", len(af.contentFilter.ContentTopics)),
						zap.Stringer("agg-content-filter", af.contentFilter))
					go mgr.subscribeAndRunLoop(af)
				}
				mgr.incompleteFilterBatch = make(map[string]filterConfig)
				mgr.Unlock()
			}
		}
	}
}

/*
addFilter method checks if there are existing waiting filters for the pubsubTopic to be subscribed and adds the new filter to the same batch
once batchlimit is hit, all filters are subscribed to and new batch is created.
if node is not online, then batch is pushed to a queue to be picked up later for subscription and new batch is created
*/
func (mgr *FilterManager) addFilter(filterID string, f *common.Filter) {
	mgr.logger.Debug("adding filter", zap.String("filter-id", filterID))

	mgr.Lock()
	defer mgr.Unlock()

	afilter, ok := mgr.incompleteFilterBatch[f.PubsubTopic]
	if !ok {
		//no existing batch for pubsubTopic
		mgr.logger.Debug("new pubsubTopic batch", zap.String("topic", f.PubsubTopic))
		cf := mgr.buildContentFilter(f.PubsubTopic, f.ContentTopics)
		afilter = filterConfig{uuid.NewString(), cf}
		mgr.incompleteFilterBatch[f.PubsubTopic] = afilter
		mgr.filterConfigs[filterID] = filterConfig{afilter.ID, cf}
	} else {
		mgr.logger.Debug("existing pubsubTopic batch", zap.String("agg-filter-id", afilter.ID), zap.String("topic", f.PubsubTopic))
		if len(afilter.contentFilter.ContentTopics)+len(f.ContentTopics) > filterSubBatchSize {
			//filter batch limit is hit
			if mgr.onlineChecker.IsOnline() {
				//node is online, go ahead and subscribe the batch
				mgr.logger.Debug("crossed pubsubTopic batchsize and online, subscribing to filters", zap.String("agg-filter-id", afilter.ID), zap.String("topic", f.PubsubTopic), zap.Int("batch-size", len(afilter.contentFilter.ContentTopics)+len(f.ContentTopics)))
				go mgr.subscribeAndRunLoop(afilter)
			} else {
				mgr.logger.Debug("crossed pubsubTopic batchsize and offline, queuing filters", zap.String("agg-filter-id", afilter.ID), zap.String("topic", f.PubsubTopic), zap.Int("batch-size", len(afilter.contentFilter.ContentTopics)+len(f.ContentTopics)))
				// queue existing batch as node is not online
				mgr.waitingToSubQueue <- afilter
			}
			cf := mgr.buildContentFilter(f.PubsubTopic, f.ContentTopics)
			afilter = filterConfig{uuid.NewString(), cf}
			mgr.logger.Debug("creating a new pubsubTopic batch", zap.String("agg-filter-id", afilter.ID), zap.String("topic", f.PubsubTopic), zap.Stringer("content-filter", cf))
			mgr.incompleteFilterBatch[f.PubsubTopic] = afilter
			mgr.filterConfigs[filterID] = filterConfig{afilter.ID, cf}
		} else {
			//add to existing batch as batch limit not reached
			var contentTopics []string
			for _, ct := range maps.Keys(f.ContentTopics) {
				afilter.contentFilter.ContentTopics[ct.ContentTopic()] = struct{}{}
				contentTopics = append(contentTopics, ct.ContentTopic())
			}
			cf := protocol.NewContentFilter(f.PubsubTopic, contentTopics...)
			mgr.logger.Debug("adding to existing pubsubTopic batch", zap.String("agg-filter-id", afilter.ID), zap.Stringer("content-filter", cf), zap.Int("batch-size", len(afilter.contentFilter.ContentTopics)))
			mgr.filterConfigs[filterID] = filterConfig{afilter.ID, cf}
		}
	}
}

func (mgr *FilterManager) subscribeAndRunLoop(f filterConfig) {
	ctx, cancel := context.WithCancel(mgr.ctx)
	config := api.FilterConfig{MaxPeers: mgr.cfg.MinPeersForFilter}
	sub, err := api.Subscribe(ctx, mgr.node, f.contentFilter, config, mgr.logger)
	mgr.Lock()
	mgr.filterSubscriptions[f.ID] = SubDetails{cancel, sub}
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
		zap.Int("agg filters count", len(mgr.filterSubscriptions)))
	if newStatus && !mgr.onlineChecker.IsOnline() { //switched from offline to Online
		mgr.logger.Debug("switching from offline to online")
		mgr.Lock()
		if len(mgr.waitingToSubQueue) > 0 {
			for af := range mgr.waitingToSubQueue {
				// TODO: change the below logic once topic specific health is implemented for lightClients
				if pubsubTopic == "" || pubsubTopic == af.contentFilter.PubsubTopic {
					// Check if any filter subs are pending and subscribe them
					mgr.logger.Debug("subscribing from filter queue", zap.String("filter-id", af.ID), zap.Stringer("content-filter", af.contentFilter))
					go mgr.subscribeAndRunLoop(af)
				} else {
					// TODO: Can this cause issues?
					mgr.waitingToSubQueue <- af
				}
				if len(mgr.waitingToSubQueue) == 0 {
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
	filterConfig, ok := mgr.filterConfigs[filterID]
	if !ok {
		mgr.logger.Debug("filter removal: filter not found", zap.String("filter-id", filterID))
		return
	}
	af, ok := mgr.filterSubscriptions[filterConfig.ID]
	if ok {
		delete(mgr.filterConfigs, filterID)
		for ct := range filterConfig.contentFilter.ContentTopics {
			delete(af.sub.ContentFilter.ContentTopics, ct)
		}
		if len(af.sub.ContentFilter.ContentTopics) == 0 {
			af.cancel()
		} else {
			go af.sub.Unsubscribe(filterConfig.contentFilter)
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
