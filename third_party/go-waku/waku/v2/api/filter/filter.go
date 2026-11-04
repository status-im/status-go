package filter

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/onlinechecker"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/filtertrace"
	"github.com/waku-org/go-waku/waku/v2/protocol/subscription"
	"github.com/waku-org/go-waku/waku/v2/utils"
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

const filterSubLoopInterval = 5 * time.Second
const filterSubMaxErrCnt = 3

// filterRateLimitBackoff is how long the apiSub waits before re-issuing a
// subscribe attempt after at least one peer returned HTTP 429
// ("filter request rejected due rate limit exceeded"). The waku server uses
// 429 to ask clients to slow down; the previous implementation flattened the
// typed peer error into a plain *errors.errorString so the apiSub never saw
// the signal and kept retrying aggressively. With the typed *SubscribeError
// (see protocol/filter/subscribe_error.go) the apiSub now honors the signal
// by suppressing retries for filterRateLimitBackoff after a 429.
const filterRateLimitBackoff = 60 * time.Second

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
	errcnt                int
	// rateLimitedUntil is set when subscribe() observes a *SubscribeError whose
	// FailedPeers contain at least one HTTP 429. While time.Now().Before(rateLimitedUntil),
	// subscriptionLoop suppresses retry triggers (ticker push and checkAndResubscribe).
	// Cleared by a successful subscribe(). Read/written only from the subscriptionLoop
	// goroutine; no lock needed.
	rateLimitedUntil time.Time
	// multiplexWG tracks per-subscription goroutines that forward envelopes from
	// subDetails.C to DataCh. cleanup() must wait for them before close(DataCh)
	// to avoid "send on closed channel" panics during teardown.
	multiplexWG sync.WaitGroup
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
func Subscribe(ctx context.Context, wf *filter.WakuFilterLightNode, contentFilter protocol.ContentFilter, config FilterConfig, log *zap.Logger, params *subscribeParameters) (*Sub, error) {
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
	// filter subscription loop is to check if target subscriptions for a filter are active and if not
	// trigger resubscribe.
	go sub.subscriptionLoop(filterSubLoopInterval)
	return sub, nil
}

func (apiSub *Sub) Unsubscribe(contentFilter protocol.ContentFilter) {
	defer utils.LogOnPanic()
	_, err := apiSub.wf.Unsubscribe(apiSub.ctx, contentFilter)
	//Not reading result unless we want to do specific error handling?
	if err != nil {
		apiSub.log.Debug("failed to unsubscribe", zap.Error(err), zap.Stringer("content-filter", contentFilter))
	}
}

func (apiSub *Sub) subscriptionLoop(loopInterval time.Duration) {
	defer utils.LogOnPanic()
	ticker := time.NewTicker(loopInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			apiSub.errcnt = 0 //reset errorCount
			if shouldHonourRateLimitBackoff(apiSub.rateLimitedUntil, time.Now()) {
				apiSub.log.Debug("ticker push suppressed by rate-limit backoff",
					zap.Time("rate-limited-until", apiSub.rateLimitedUntil),
				)
				continue
			}
			if apiSub.onlineChecker.IsOnline() && len(apiSub.subs) < apiSub.Config.MaxPeers &&
				!apiSub.resubscribeInProgress && len(apiSub.closing) < apiSub.Config.MaxPeers {
				apiSub.closing <- ""
			}
		case <-apiSub.ctx.Done():
			apiSub.log.Debug("apiSub context: done")
			apiSub.cleanup()
			return
		case subId := <-apiSub.closing:
			if shouldHonourRateLimitBackoff(apiSub.rateLimitedUntil, time.Now()) {
				apiSub.log.Debug("checkAndResubscribe suppressed by rate-limit backoff",
					zap.Time("rate-limited-until", apiSub.rateLimitedUntil),
				)
				continue
			}
			if apiSub.errcnt < filterSubMaxErrCnt {
				apiSub.resubscribeInProgress = true
				//trigger resubscribe flow for subscription.
				apiSub.checkAndResubscribe(subId)
			} else {
				apiSub.log.Debug("retry suppressed by errcnt bound",
					zap.Int("errcnt", apiSub.errcnt),
					zap.Int("filter-sub-max-err-cnt", filterSubMaxErrCnt),
				)
			}
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
	// Wait for in-flight multiplex goroutines to exit before closing DataCh,
	// otherwise they may panic sending to a closed channel.
	apiSub.multiplexWG.Wait()
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

// shouldHonourRateLimitBackoff reports whether the apiSub is currently within
// a rate-limit backoff window and should skip retry triggers. now == rateLimitedUntil
// is treated as "window has just elapsed" → false (allow retry), so a zero-value
// rateLimitedUntil (never set) is always false.
func shouldHonourRateLimitBackoff(rateLimitedUntil, now time.Time) bool {
	return now.Before(rateLimitedUntil)
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
		apiSub.log.Warn("subscribe error",
			zap.Error(err),
			zap.Int("errcnt-before-inc", apiSub.errcnt),
		)

		// If any peer responded HTTP 429, enter a rate-limit backoff window.
		// subscriptionLoop's gates will suppress retry triggers for
		// filterRateLimitBackoff. The typed *SubscribeError comes from
		// protocol/filter/client.go.
		var subErr *filter.SubscribeError
		if errors.As(err, &subErr) && subErr.HasRateLimitError() {
			apiSub.rateLimitedUntil = time.Now().Add(filterRateLimitBackoff)
			apiSub.log.Warn("rate-limited by peer, backing off",
				zap.Duration("backoff", filterRateLimitBackoff),
				zap.Time("until", apiSub.rateLimitedUntil),
				zap.Int("failed-peers", len(subErr.FailedPeers)),
			)
		}

		apiSub.errcnt++
		apiSub.log.Debug("errcnt incremented",
			zap.Int("new-errcnt", apiSub.errcnt),
			zap.Error(err),
		)

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
	// On full success, clear any prior rate-limit backoff so retries can resume
	// normally if a fresh failure occurs later.
	apiSub.rateLimitedUntil = time.Time{}
	apiSub.log.Debug("subscribe success", zap.Int("subs-count", len(subs)))
	return subs, nil
}

func (apiSub *Sub) multiplex(subs []*subscription.SubscriptionDetails) {
	observer := apiSub.wf.TraceObserver()
	// Multiplex onto single channel
	// Goroutines exit when subDetails.C is closed or apiSub.ctx is done.
	// cleanup() waits on multiplexWG before close(DataCh) to avoid a race.
	for _, subDetails := range subs {
		apiSub.subs[subDetails.ID] = subDetails
		apiSub.multiplexWG.Add(1)
		go func(subDetails *subscription.SubscriptionDetails) {
			defer utils.LogOnPanic()
			defer apiSub.multiplexWG.Done()
			apiSub.log.Debug("new multiplex", zap.String("sub-id", subDetails.ID))
			// Both the receive and the send must be cancelable via apiSub.ctx:
			// during node teardown, UnsubscribeWithSubscription may early-return
			// from ErrOnNotRunning() without calling sub.Close(), leaving
			// subDetails.C open forever. A bare `for env := range subDetails.C`
			// would then block here, multiplexWG.Wait() in cleanup() would block
			// on it, and the whole filter shutdown would deadlock.
			for {
				select {
				case env, ok := <-subDetails.C:
					if !ok {
						return
					}
					emit := func(stage, reason string) {
						if observer == nil {
							return
						}
						observer(filtertrace.Event{Stage: stage, Reason: reason, RemotePeerID: subDetails.PeerID.String(),
							EnvelopeHash: env.Hash().String(), PubsubTopic: env.PubsubTopic(), ContentTopic: env.Message().ContentTopic,
							SubscriptionID: subDetails.ID, QueueDepth: len(apiSub.DataCh), QueueCapacity: cap(apiSub.DataCh)})
					}
					emit("filter_multiplex_dequeued", "")
					select {
					case apiSub.DataCh <- env:
						emit("filter_multiplex_queued", "")
					case <-apiSub.ctx.Done():
						emit("filter_queue_rejected", "multiplex_context_canceled")
						return
					}
				case <-apiSub.ctx.Done():
					return
				}
			}
		}(subDetails)
		go func(subDetails *subscription.SubscriptionDetails) {
			defer utils.LogOnPanic()
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
