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
	filters        map[string]func() // map of filters to cancel funcs
	onNewEnvelopes func(env *protocol.Envelope) error
	logger         *zap.Logger
	node           *filter.WakuFilterLightNode
	peersAvailable bool
	filterQueue    chan filterDetails
}

const filterQueueSize = 1000

type filterDetails struct {
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
	mgr.filters = make(map[string]func())
	mgr.node = node
	mgr.peersAvailable = false
	mgr.filterQueue = make(chan filterDetails, filterQueueSize)
	return mgr
}

func (mgr *FilterManager) addFilter(filterID string, f *common.Filter) {
	mgr.Lock()
	defer mgr.Unlock()
	contentFilter := mgr.buildContentFilter(f.PubsubTopic, f.ContentTopics)
	if mgr.peersAvailable {
		go mgr.subscribeAndRunLoop(filterDetails{filterID, contentFilter})
	} else {
		mgr.filterQueue <- filterDetails{filterID, contentFilter}
	}
}

func (mgr *FilterManager) subscribeAndRunLoop(f filterDetails) {
	ctx, cancel := context.WithCancel(mgr.ctx)
	mgr.Lock()
	mgr.filters[f.ID] = cancel
	mgr.Unlock()
	config := api.FilterConfig{MaxPeers: mgr.cfg.MinPeersForFilter}

	sub, err := api.Subscribe(ctx, mgr.node, f.contentFilter, config, mgr.logger)
	if err == nil {
		mgr.runFilterSubscriptionLoop(sub)
	}
}

func (mgr *FilterManager) onConnectionStatusChange(pubsubTopic string, newStatus bool) {
	//As of now nothing to be done in case of loss of connection as Filter subscription API would take care of it.
	if newStatus == true {
		mgr.peersAvailable = true
		//Check if any filter subs are pending and subscribe them
		for filter := range mgr.filterQueue {
			go mgr.subscribeAndRunLoop(filter)
			if len(mgr.filterQueue) == 0 {
				break
			}
		}
	}
}

func (mgr *FilterManager) removeFilter(filterID string) {
	mgr.Lock()
	defer mgr.Unlock()
	cancel, ok := mgr.filters[filterID]
	if ok {
		delete(mgr.filters, filterID)
		// close goroutine running runFilterSubscriptionLoop
		// this will also close api.Sub
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

func (mgr *FilterManager) runFilterSubscriptionLoop(sub *api.Sub) {
	for {
		select {
		case <-mgr.ctx.Done():
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
