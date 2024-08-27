package filter

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/onlinechecker"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/subscription"
	"go.uber.org/zap"
)

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
	ContentFilter         protocol.ContentFilter
	DataCh                chan *protocol.Envelope
	Config                FilterConfig
	subs                  subscription.SubscriptionSet
	wf                    *filter.WakuFilterLightNode
	ctx                   context.Context
	cancel                context.CancelFunc
	log                   *zap.Logger
	closing               chan string
	onlineChecker         onlinechecker.OnlineChecker
	resubscribeInProgress bool
	id                    string
}

type subscribeParameters struct {
	batchInterval          time.Duration
	multiplexChannelBuffer int
}

type SubscribeOptions func(*subscribeParameters)

func WithBatchInterval(t time.Duration) SubscribeOptions {
	return func(params *subscribeParameters) {
		params.batchInterval = t
	}
}

func WithMultiplexChannelBuffer(value int) SubscribeOptions {
	return func(params *subscribeParameters) {
		params.multiplexChannelBuffer = value
	}
}

func defaultOptions() []SubscribeOptions {
	return []SubscribeOptions{
		WithBatchInterval(5 * time.Second),
		WithMultiplexChannelBuffer(100),
	}
}

// Subscribe
func Subscribe(ctx context.Context, wf *filter.WakuFilterLightNode, contentFilter protocol.ContentFilter, config FilterConfig, log *zap.Logger, opts ...SubscribeOptions) (*Sub, error) {
	optList := append(defaultOptions(), opts...)
	params := new(subscribeParameters)
	for _, opt := range optList {
		opt(params)
	}

	sub := new(Sub)
	sub.id = uuid.NewString()
	sub.wf = wf
	sub.ctx, sub.cancel = context.WithCancel(ctx)
	sub.subs = make(subscription.SubscriptionSet)
	sub.DataCh = make(chan *protocol.Envelope, params.multiplexChannelBuffer)
	sub.ContentFilter = contentFilter
	sub.Config = config
	sub.log = log.Named("filter-api").With(zap.String("apisub-id", sub.id), zap.Stringer("content-filter", sub.ContentFilter))
	sub.log.Debug("filter subscribe params", zap.Int("max-peers", config.MaxPeers))
	sub.closing = make(chan string, config.MaxPeers)

	sub.onlineChecker = wf.OnlineChecker()
	if wf.OnlineChecker().IsOnline() {
		subs, err := sub.subscribe(contentFilter, sub.Config.MaxPeers)
		if err == nil {
			sub.multiplex(subs)
		}
	}

	go sub.subscriptionLoop(params.batchInterval)
	return sub, nil
}

func (apiSub *Sub) Unsubscribe(contentFilter protocol.ContentFilter) {
	_, err := apiSub.wf.Unsubscribe(apiSub.ctx, contentFilter)
	//Not reading result unless we want to do specific error handling?
	if err != nil {
		apiSub.log.Debug("failed to unsubscribe", zap.Error(err), zap.Stringer("content-filter", contentFilter))
	}
}

func (apiSub *Sub) subscriptionLoop(batchInterval time.Duration) {
	ticker := time.NewTicker(batchInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if apiSub.onlineChecker.IsOnline() && len(apiSub.subs) < apiSub.Config.MaxPeers &&
				!apiSub.resubscribeInProgress && len(apiSub.closing) < apiSub.Config.MaxPeers {
				apiSub.closing <- ""
			}
		case <-apiSub.ctx.Done():
			apiSub.log.Debug("apiSub context: done")
			apiSub.cleanup()
			return
		case subId := <-apiSub.closing:
			apiSub.resubscribeInProgress = true
			//trigger resubscribe flow for subscription.
			apiSub.checkAndResubscribe(subId)
		}
	}
}

func (apiSub *Sub) checkAndResubscribe(subId string) {

	var failedPeer peer.ID
	if subId != "" {
		apiSub.log.Debug("subscription close and resubscribe", zap.String("sub-id", subId), zap.Stringer("content-filter", apiSub.ContentFilter))

		apiSub.subs[subId].Close()
		failedPeer = apiSub.subs[subId].PeerID
		delete(apiSub.subs, subId)
	}
	apiSub.log.Debug("subscription status", zap.Int("sub-count", len(apiSub.subs)), zap.Stringer("content-filter", apiSub.ContentFilter))
	if apiSub.onlineChecker.IsOnline() && len(apiSub.subs) < apiSub.Config.MaxPeers {
		apiSub.resubscribe(failedPeer)
	}
	apiSub.resubscribeInProgress = false
}

func (apiSub *Sub) cleanup() {
	apiSub.log.Debug("cleaning up subscription", zap.Stringer("config", apiSub.Config))

	for _, s := range apiSub.subs {
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
	apiSub.log.Debug("subscribing again", zap.Int("num-peers", apiSub.Config.MaxPeers-existingSubCount))
	var peersToExclude peer.IDSlice
	if failedPeer != "" { //little hack, couldn't find a better way to do it
		peersToExclude = append(peersToExclude, failedPeer)
	}
	for _, sub := range apiSub.subs {
		peersToExclude = append(peersToExclude, sub.PeerID)
	}
	subs, err := apiSub.subscribe(apiSub.ContentFilter, apiSub.Config.MaxPeers-existingSubCount, peersToExclude...)
	if err != nil {
		apiSub.log.Debug("failed to resubscribe for filter", zap.Error(err))
		return
	} //Not handling scenario where all requested subs are not received as that should get handled from user of the API.

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
		apiSub.log.Debug("subscribing with peers to exclude", zap.Stringers("excluded-peers", peersToExclude))
		options = append(options, filter.WithPeersToExclude(peersToExclude...))
	}
	subs, err := apiSub.wf.Subscribe(apiSub.ctx, contentFilter, options...)

	if err != nil {
		//Inform of error, so that resubscribe can be triggered if required
		if len(apiSub.closing) < apiSub.Config.MaxPeers {
			apiSub.closing <- ""
		}
		if len(subs) > 0 {
			// Partial Failure, which means atleast 1 subscription is successful
			apiSub.log.Debug("partial failure in filter subscribe", zap.Error(err), zap.Int("success-count", len(subs)))
			return subs, nil
		}
		// TODO: Once filter error handling indicates specific error, this can be handled better.
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
			apiSub.log.Debug("new multiplex", zap.String("sub-id", subDetails.ID))
			for env := range subDetails.C {
				apiSub.DataCh <- env
			}
		}(subDetails)
		go func(subDetails *subscription.SubscriptionDetails) {
			select {
			case <-apiSub.ctx.Done():
				return
			case <-subDetails.Closing:
				apiSub.log.Debug("sub closing", zap.String("sub-id", subDetails.ID))
				apiSub.closing <- subDetails.ID
			}
		}(subDetails)
	}
}
