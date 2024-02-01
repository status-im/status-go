package api

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/subscription"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
)

const FilterPingTimeout = 5 * time.Second
const MultiplexChannelBuffer = 100

type FilterConfig struct {
	MaxPeers int       `json:"maxPeers"`
	Peers    []peer.ID `json:"peers"`
}

func (fc FilterConfig) String() string {
	jsonStr, err := json.Marshal(fc)
	if err != nil {
		return ""
	}
	return string(jsonStr)
}

type Sub struct {
	ContentFilter protocol.ContentFilter
	DataCh        chan *protocol.Envelope
	Config        FilterConfig
	subs          subscription.SubscriptionSet
	wf            *filter.WakuFilterLightNode
	ctx           context.Context
	cancel        context.CancelFunc
	log           *zap.Logger
}

// Subscribe
func Subscribe(ctx context.Context, wf *filter.WakuFilterLightNode, contentFilter protocol.ContentFilter, config FilterConfig, log *zap.Logger) (*Sub, error) {
	sub := new(Sub)
	sub.wf = wf
	sub.ctx, sub.cancel = context.WithCancel(ctx)
	sub.subs = make(subscription.SubscriptionSet)
	sub.DataCh = make(chan *protocol.Envelope, MultiplexChannelBuffer)
	sub.ContentFilter = contentFilter
	sub.Config = config
	sub.log = log.Named("filter-api")
	sub.log.Debug("filter subscribe params", zap.Int("maxPeers", config.MaxPeers), zap.Stringer("contentFilter", contentFilter))
	subs, err := sub.subscribe(contentFilter, sub.Config.MaxPeers)

	if err != nil {
		return nil, err
	}
	sub.multiplex(subs)
	go sub.healthCheckLoop()
	return sub, nil
}

func (apiSub *Sub) Unsubscribe() {
	apiSub.cancel()

}

func (apiSub *Sub) healthCheckLoop() {
	// Health checks
	ticker := time.NewTicker(FilterPingTimeout)
	defer ticker.Stop()
	for {
		select {
		case <-apiSub.ctx.Done():
			apiSub.log.Debug("healthCheckLoop: Done()")
			apiSub.cleanup()
			return
		case <-ticker.C:
			apiSub.log.Debug("healthCheckLoop: checkAliveness()")
			topicCounts := apiSub.getTopicCounts()
			apiSub.resubscribe(topicCounts)
		}
	}

}

func (apiSub *Sub) cleanup() {
	apiSub.log.Debug("ENTER cleanup()")
	defer func() {
		apiSub.log.Debug("EXIT cleanup()")
	}()

	for _, s := range apiSub.subs {
		_, err := apiSub.wf.UnsubscribeWithSubscription(apiSub.ctx, s)
		if err != nil {
			//Logging with info as this is part of cleanup
			apiSub.log.Info("failed to unsubscribe filter", zap.Error(err))
		}
	}
	close(apiSub.DataCh)

}

// Returns active sub counts for each pubsub topic
func (apiSub *Sub) getTopicCounts() map[string]int {
	// Buffered chan for sub aliveness results
	type CheckResult struct {
		sub   *subscription.SubscriptionDetails
		alive bool
	}
	checkResults := make(chan CheckResult, len(apiSub.subs))
	var wg sync.WaitGroup

	// Run pings asynchronously
	for _, s := range apiSub.subs {
		wg.Add(1)
		go func(sub *subscription.SubscriptionDetails) {
			defer wg.Done()
			ctx, cancelFunc := context.WithTimeout(apiSub.ctx, FilterPingTimeout)
			defer cancelFunc()
			err := apiSub.wf.IsSubscriptionAlive(ctx, sub)

			apiSub.log.Debug("Check result:", zap.Any("subID", sub.ID), zap.Bool("result", err == nil))
			checkResults <- CheckResult{sub, err == nil}
		}(s)
	}

	// Collect healthy topic counts
	topicCounts := make(map[string]int)

	topicMap, _ := protocol.ContentFilterToPubSubTopicMap(apiSub.ContentFilter)
	for _, t := range maps.Keys(topicMap) {
		topicCounts[t] = 0
	}
	wg.Wait()
	close(checkResults)
	for s := range checkResults {
		if !s.alive {
			// Close inactive subs
			s.sub.Close()
			delete(apiSub.subs, s.sub.ID)
		} else {
			topicCounts[s.sub.ContentFilter.PubsubTopic]++
		}
	}

	return topicCounts
}

// Attempts to resubscribe on topics that lack subscriptions
func (apiSub *Sub) resubscribe(topicCounts map[string]int) {

	// Delete healthy topics
	for t, cnt := range topicCounts {
		if cnt == apiSub.Config.MaxPeers {
			delete(topicCounts, t)
		}
	}

	if len(topicCounts) == 0 {
		// All topics healthy, return
		return
	}
	var wg sync.WaitGroup

	// Re-subscribe asynchronously
	newSubs := make(chan []*subscription.SubscriptionDetails)

	for t, cnt := range topicCounts {
		cFilter := protocol.ContentFilter{PubsubTopic: t, ContentTopics: apiSub.ContentFilter.ContentTopics}
		wg.Add(1)
		go func(count int) {
			defer wg.Done()
			subs, err := apiSub.subscribe(cFilter, apiSub.Config.MaxPeers-count)
			if err != nil {
				return
			} //Not handling scenario where all requested subs are not received as that will get handled in next cycle.
			newSubs <- subs
		}(cnt)
	}
	wg.Wait()
	close(newSubs)
	apiSub.log.Debug("resubscribe(): before range newSubs")
	for subs := range newSubs {
		if subs != nil {
			apiSub.multiplex(subs)
		}
	}
	apiSub.log.Debug("checkAliveness(): close(newSubs)")
	//close(newSubs)
}

func (apiSub *Sub) subscribe(contentFilter protocol.ContentFilter, peerCount int) ([]*subscription.SubscriptionDetails, error) {
	// Low-level subscribe, returns a set of SubscriptionDetails
	options := make([]filter.FilterSubscribeOption, 0)
	options = append(options, filter.WithMaxPeersPerContentFilter(int(peerCount)))
	for _, p := range apiSub.Config.Peers {
		options = append(options, filter.WithPeer(p))
	}
	subs, err := apiSub.wf.Subscribe(apiSub.ctx, contentFilter, options...)

	if err != nil {
		if len(subs) > 0 {
			// Partial Failure, for now proceed as we don't expect this to happen wrt specific topics.
			// Rather it can happen in case subscription with one of the peer fails.
			// This can further get automatically handled at resubscribe,
			apiSub.log.Error("partial failure in Filter subscribe", zap.Error(err))
			return subs, nil
		}
		// In case of complete subscription failure, application or user needs to handle and probably retry based on error
		// TODO: Once filter error handling indicates specific error, this can be addressed based on the error at this layer.
		return nil, err
	}

	return subs, nil
}

func (apiSub *Sub) multiplex(subs []*subscription.SubscriptionDetails) {

	// Multiplex onto single channel
	// Goroutines will exit once sub channels are closed
	for _, subDetails := range subs {
		apiSub.subs[subDetails.ID] = subDetails
		go func(subDetails *subscription.SubscriptionDetails) {
			apiSub.log.Debug("New multiplex", zap.String("subID", subDetails.ID))
			for env := range subDetails.C {
				apiSub.DataCh <- env
			}
		}(subDetails)
	}
}
