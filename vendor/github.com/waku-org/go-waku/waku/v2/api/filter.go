package api

import (
	"context"
	"encoding/json"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/subscription"
	"go.uber.org/zap"
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
	closing       chan string
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
	sub.closing = make(chan string, config.MaxPeers)
	if err != nil {
		return nil, err
	}
	sub.multiplex(subs)
	go sub.waitOnSubClose()
	return sub, nil
}

func (apiSub *Sub) Unsubscribe() {
	apiSub.cancel()
}

func (apiSub *Sub) waitOnSubClose() {
	for {
		select {
		case <-apiSub.ctx.Done():
			apiSub.log.Debug("apiSub context: Done()")
			apiSub.cleanup()
			return
		case subId := <-apiSub.closing:
			//trigger closing and resubscribe flow for subscription.
			apiSub.closeAndResubscribe(subId)
		}
	}
}

func (apiSub *Sub) closeAndResubscribe(subId string) {
	apiSub.log.Debug("sub closeAndResubscribe", zap.String("subID", subId))

	apiSub.subs[subId].Close()
	failedPeer := apiSub.subs[subId].PeerID
	delete(apiSub.subs, subId)
	apiSub.resubscribe(failedPeer)
}

func (apiSub *Sub) cleanup() {
	apiSub.log.Debug("ENTER cleanup()")
	defer func() {
		apiSub.log.Debug("EXIT cleanup()")
	}()

	for _, s := range apiSub.subs {
		close(s.Closing)
		_, err := apiSub.wf.UnsubscribeWithSubscription(apiSub.ctx, s)
		if err != nil {
			//Logging with info as this is part of cleanup
			apiSub.log.Info("failed to unsubscribe filter", zap.Error(err))
		}
	}
	close(apiSub.DataCh)

}

// Attempts to resubscribe on topics that lack subscriptions
func (apiSub *Sub) resubscribe(failedPeer peer.ID) {
	// Re-subscribe asynchronously
	existingSubCount := len(apiSub.subs)
	apiSub.log.Debug("subscribing again", zap.Stringer("contentFilter", apiSub.ContentFilter), zap.Int("numPeers", apiSub.Config.MaxPeers-existingSubCount))
	var peersToExclude peer.IDSlice
	peersToExclude = append(peersToExclude, failedPeer)
	for _, sub := range apiSub.subs {
		peersToExclude = append(peersToExclude, sub.PeerID)
	}
	subs, err := apiSub.subscribe(apiSub.ContentFilter, apiSub.Config.MaxPeers-existingSubCount, peersToExclude...)
	if err != nil {
		return
	} //Not handling scenario where all requested subs are not received as that will get handled in next cycle.

	apiSub.log.Debug("resubscribe(): before range newSubs")

	apiSub.multiplex(subs)
}

func (apiSub *Sub) subscribe(contentFilter protocol.ContentFilter, peerCount int, peersToExclude ...peer.ID) ([]*subscription.SubscriptionDetails, error) {
	// Low-level subscribe, returns a set of SubscriptionDetails
	options := make([]filter.FilterSubscribeOption, 0)
	options = append(options, filter.WithMaxPeersPerContentFilter(int(peerCount)))
	for _, p := range apiSub.Config.Peers {
		options = append(options, filter.WithPeer(p))
	}
	if len(peersToExclude) > 0 {
		apiSub.log.Debug("subscribing with peersToExclude", zap.Stringer("peersToExclude", peersToExclude[0]))
		options = append(options, filter.WithPeersToExclude(peersToExclude...))
	}
	subs, err := apiSub.wf.Subscribe(apiSub.ctx, contentFilter, options...)

	if err != nil {
		if len(subs) > 0 {
			// Partial Failure, for now proceed as we don't expect this to happen wrt specific topics.
			// Rather it can happen in case subscription with one of the peer fails.
			// This can further get automatically handled at resubscribe,
			apiSub.log.Error("partial failure in Filter subscribe", zap.Error(err), zap.Int("successCount", len(subs)))
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
		go func(subDetails *subscription.SubscriptionDetails) {
			<-subDetails.Closing
			apiSub.log.Debug("sub closing", zap.String("subID", subDetails.ID))

			apiSub.closing <- subDetails.ID
		}(subDetails)
	}
}
