package wakuv2

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/status-im/status-go/wakuv2/common"

	node "github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
)

const (
	FilterEventAdded             int = 0
	FilterEventRemoved           int = 1
	FilterEventPingResult        int = 2
	FilterEventSubscribeResult   int = 3
	FilterEventUnsubscribeResult int = 4
	FilterEventGetStats          int = 5
)

const pingTimeout = 5 * time.Second

type FilterSubs map[string]filter.SubscriptionSet

type FilterEvent struct {
	eventType int
	filterID  string
	result    bool
	peerID    peer.ID
	tempID    string
	sub       *filter.SubscriptionDetails
	ch        chan FilterSubs
}

type FilterManager struct {
	ctx              context.Context
	filterSubs       FilterSubs
	eventChan        chan (FilterEvent)
	isFilterSubAlive func(sub *filter.SubscriptionDetails) error
	getFilter        func(string) *common.Filter
	onNewEnvelopes   func(env *protocol.Envelope) error
	peers            []peer.ID
	logger           *zap.Logger
	settings         settings
	node             *node.WakuNode
}

func newFilterManager(ctx context.Context, logger *zap.Logger, getFilterFn func(string) *common.Filter, settings settings, onNewEnvelopes func(env *protocol.Envelope) error, node *node.WakuNode) *FilterManager {
	// This fn is being mocked in test
	mgr := new(FilterManager)
	mgr.ctx = ctx
	mgr.logger = logger
	mgr.getFilter = getFilterFn
	mgr.onNewEnvelopes = onNewEnvelopes
	mgr.filterSubs = make(FilterSubs)
	mgr.eventChan = make(chan FilterEvent)
	mgr.peers = make([]peer.ID, 0)
	mgr.settings = settings
	mgr.node = node
	mgr.isFilterSubAlive = func(sub *filter.SubscriptionDetails) error {
		ctx, cancel := context.WithTimeout(ctx, pingTimeout)
		defer cancel()
		return mgr.node.FilterLightnode().IsSubscriptionAlive(ctx, sub)
	}

	return mgr
}

func (mgr *FilterManager) runFilterLoop(wg *sync.WaitGroup) {
	defer wg.Done()
	// Use it to ping filter peer(s) periodically
	ticker := time.NewTicker(pingTimeout)
	defer ticker.Stop()

	// Populate filter peers initially
	mgr.peers = mgr.findFilterPeers() // ordered list of peers to select from

	for {
		select {
		case <-mgr.ctx.Done():
			return
		case <-ticker.C:
			mgr.peers = mgr.findFilterPeers()
			mgr.pingPeers()
		case ev := <-mgr.eventChan:
			switch ev.eventType {

			case FilterEventAdded:
				mgr.filterSubs[ev.filterID] = make(filter.SubscriptionSet)
				mgr.resubscribe(ev.filterID)

			case FilterEventRemoved:
				for _, sub := range mgr.filterSubs[ev.filterID] {
					if sub == nil {
						// Skip temp subs
						continue
					}
					go mgr.unsubscribeFromFilter(ev.filterID, sub)
				}
				delete(mgr.filterSubs, ev.filterID)

			case FilterEventPingResult:
				if ev.result {
					break
				}
				if ev.filterID != "" {
					// Full resubscribe, filter has no peers
					mgr.logger.Info("filter has no subs", zap.String("filterId", ev.filterID))
					mgr.resubscribe(ev.filterID)
					break
				}
				// Remove peer from list
				for i, p := range mgr.peers {
					if ev.peerID == p {
						mgr.peers = append(mgr.peers[:i], mgr.peers[i+1:]...)
						break
					}
				}
				// Delete subs for removed peer
				for filterID, subs := range mgr.filterSubs {
					for _, sub := range subs {
						if sub == nil {
							// Skip temp subs
							continue
						}
						if sub.PeerID == ev.peerID {
							mgr.logger.Info("FILTER sub is inactive", zap.String("filterId", filterID), zap.String("subID", sub.ID))
							delete(subs, sub.ID)
							go mgr.unsubscribeFromFilter(filterID, sub)
						}
					}
					mgr.resubscribe(filterID)
				}

			case FilterEventSubscribeResult:
				if ev.result {
					mgr.filterSubs[ev.filterID][ev.sub.ID] = ev.sub
					go mgr.runFilterSubscriptionLoop(ev.sub)
				}
				// Delete temp subscription record
				delete(mgr.filterSubs[ev.filterID], ev.tempID)

			case FilterEventUnsubscribeResult:
				mgr.logger.Info("FILTER event UNSUBSCRIBE_RESULT", zap.String("filterId", ev.filterID), zap.Stringer("peerID", ev.sub.PeerID))
			case FilterEventGetStats:
				mgr.logger.Info("### getstats")
				stats := make(FilterSubs)
				for id, subs := range mgr.filterSubs {
					stats[id] = make(filter.SubscriptionSet)
					for subID, sub := range subs {
						if sub == nil {
							// Skip temp subs
							continue
						}

						stats[id][subID] = sub
					}
				}
				ev.ch <- stats
			}
		}
	}
}

func (mgr *FilterManager) subscribeToFilter(filterID string, peer peer.ID, tempID string) {

	f := mgr.getFilter(filterID)
	if f == nil {
		mgr.logger.Error("FILTER subscribeToFilter: No filter found", zap.String("id", filterID))
		mgr.eventChan <- FilterEvent{eventType: FilterEventSubscribeResult, filterID: filterID, tempID: tempID, result: false}
		return
	}
	contentFilter := mgr.buildContentFilter(f.PubsubTopic, f.ContentTopics)
	mgr.logger.Info("FILTER subscribe to filter node", zap.Stringer("peer", peer), zap.String("pubsubTopic", contentFilter.PubsubTopic), zap.Strings("contentTopics", contentFilter.ContentTopicsList()))
	ctx, cancel := context.WithTimeout(mgr.ctx, pingTimeout)
	defer cancel()

	subDetails, err := mgr.node.FilterLightnode().Subscribe(ctx, contentFilter, filter.WithPeer(peer))
	success := err == nil
	var sub *filter.SubscriptionDetails
	if err != nil {
		mgr.logger.Warn("FILTER could not add wakuv2 filter for peer", zap.Stringer("peer", peer), zap.Error(err))
	} else {
		mgr.logger.Info("FILTER subscription success", zap.Stringer("peer", peer), zap.String("pubsubTopic", contentFilter.PubsubTopic), zap.Strings("contentTopics", contentFilter.ContentTopicsList()))
		sub = subDetails[0]

	}

	mgr.eventChan <- FilterEvent{eventType: FilterEventSubscribeResult, filterID: filterID, tempID: tempID, sub: sub, result: success}
}

func (mgr *FilterManager) unsubscribeFromFilter(filterID string, sub *filter.SubscriptionDetails) {
	mgr.logger.Info("FILTER unsubscribe from filter node", zap.String("filterId", filterID), zap.String("subId", sub.ID), zap.Stringer("peer", sub.PeerID))
	// Unsubscribe on light node
	ctx, cancel := context.WithTimeout(mgr.ctx, pingTimeout)
	defer cancel()
	_, err := mgr.node.FilterLightnode().UnsubscribeWithSubscription(ctx, sub)
	success := err == nil

	if err != nil {
		mgr.logger.Warn("could not unsubscribe wakuv2 filter for peer", zap.String("subId", sub.ID), zap.Error(err))
	}

	mgr.eventChan <- FilterEvent{eventType: FilterEventUnsubscribeResult, filterID: filterID, result: success, sub: sub}
}

// Check whether each of the installed filters
// has enough alive subscriptions to peers
func (mgr *FilterManager) pingPeers() {
	mgr.logger.Info("FILTER pingPeers")

	distinctPeers := make(map[peer.ID]struct{})
	for filterID, subs := range mgr.filterSubs {
		if len(subs) == 0 { // TODO !!!!!!!
			// No subs found, trigger full resubscribe
			mgr.logger.Info("FILTER ping peer no subs", zap.String("filterId", filterID))
			go func() {
				mgr.eventChan <- FilterEvent{eventType: FilterEventPingResult, filterID: filterID, result: false}
			}()
			continue
		}
		for _, sub := range subs {
			if sub == nil {
				// Skip temp subs
				continue
			}
			_, found := distinctPeers[sub.PeerID]
			if found {
				continue
			}
			distinctPeers[sub.PeerID] = struct{}{}
			mgr.logger.Info("FILTER ping peer", zap.Stringer("peerId", sub.PeerID))
			go func(sub *filter.SubscriptionDetails) {
				err := mgr.isFilterSubAlive(sub)
				alive := err == nil

				if alive {
					mgr.logger.Info("FILTER aliveness check succeeded", zap.Stringer("peerId", sub.PeerID))
				} else {
					mgr.logger.Info("FILTER aliveness check failed", zap.Stringer("peerId", sub.PeerID), zap.Error(err))
				}
				mgr.eventChan <- FilterEvent{eventType: FilterEventPingResult, peerID: sub.PeerID, result: alive}
			}(sub)
		}
	}
}

func (mgr *FilterManager) buildContentFilter(pubsubTopic string, contentTopicSet common.TopicSet) filter.ContentFilter {
	contentTopics := make([]string, len(contentTopicSet))
	for i, ct := range maps.Keys(contentTopicSet) {
		contentTopics[i] = ct.ContentTopic()
	}

	return filter.ContentFilter{
		PubsubTopic:   pubsubTopic,
		ContentTopics: filter.NewContentTopicSet(contentTopics...),
	}
}

// Find suitable peer(s)
func (mgr *FilterManager) findFilterPeers() []peer.ID {
	allPeers := mgr.node.Host().Peerstore().Peers()
	//mgr.logger.Info("Peerstore peers", zap.Stringers("peers", allPeers))

	peers := make([]peer.ID, 0)
	for _, peer := range allPeers {
		protocols, err := mgr.node.Host().Peerstore().SupportsProtocols(peer, filter.FilterSubscribeID_v20beta1, relay.WakuRelayID_v200)
		if err != nil {
			mgr.logger.Info("SupportsProtocols error", zap.Error(err))
			continue
		}

		if len(protocols) == 2 {
			peers = append(peers, peer)
		}
	}

	mgr.logger.Info("Filtered peers", zap.Int("cnt", len(peers)))
	return peers
}

func (mgr *FilterManager) findPeerCandidate() (peer.ID, error) {
	if len(mgr.peers) == 0 {
		return "", errors.New("FILTER could not select a suitable peer")
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(mgr.peers))))
	return mgr.peers[n.Int64()], nil
}

func (mgr *FilterManager) resubscribe(filterID string) {
	subs, found := mgr.filterSubs[filterID]
	if !found {
		mgr.logger.Error("resubscribe filter not found", zap.String("filterId", filterID))
		return
	}
	mgr.logger.Info("FILTER active subscriptions count:", zap.String("filterId", filterID), zap.Int("len", len(subs)))
	for i := len(subs); i < mgr.settings.MinPeersForFilter; i++ {
		mgr.logger.Info("FILTER check not passed, try subscribing to peers", zap.String("filterId", filterID))
		peer, err := mgr.findPeerCandidate()

		if err == nil {
			// Create sub placeholder in order to avoid potentially too many subs
			tempID := uuid.NewString()
			subs[tempID] = nil
			go mgr.subscribeToFilter(filterID, peer, tempID)
		} else {
			mgr.logger.Error("FILTER resubscribe findPeer error", zap.Error(err))
		}
	}
}

func (mgr *FilterManager) runFilterSubscriptionLoop(sub *filter.SubscriptionDetails) {
	for {
		select {
		case <-mgr.ctx.Done():
			return
		case env, ok := <-sub.C:
			if ok {
				err := (mgr.onNewEnvelopes)(env)
				if err != nil {
					mgr.logger.Error("OnNewEnvelopes error", zap.Error(err))
				}
			} else {
				mgr.logger.Info("FILTER sub is closed", zap.String("id", sub.ID))
				return
			}
		}
	}
}
