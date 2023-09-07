package filter

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/peermanager"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

type (
	FilterSubscribeParameters struct {
		host         host.Host
		selectedPeer peer.ID
		pm           *peermanager.PeerManager
		requestID    []byte
		log          *zap.Logger
	}

	FilterUnsubscribeParameters struct {
		unsubscribeAll bool
		selectedPeer   peer.ID
		requestID      []byte
		log            *zap.Logger
		wg             *sync.WaitGroup
	}

	FilterParameters struct {
		Timeout        time.Duration
		MaxSubscribers int
	}

	Option func(*FilterParameters)

	FilterSubscribeOption   func(*FilterSubscribeParameters)
	FilterUnsubscribeOption func(*FilterUnsubscribeParameters)
)

func WithTimeout(timeout time.Duration) Option {
	return func(params *FilterParameters) {
		params.Timeout = timeout
	}
}

func WithPeer(p peer.ID) FilterSubscribeOption {
	return func(params *FilterSubscribeParameters) {
		params.selectedPeer = p
	}
}

// WithAutomaticPeerSelection is an option used to randomly select a peer from the peer store.
// If a list of specific peers is passed, the peer will be chosen from that list assuming it
// supports the chosen protocol, otherwise it will chose a peer from the node peerstore
func WithAutomaticPeerSelection(fromThesePeers ...peer.ID) FilterSubscribeOption {
	return func(params *FilterSubscribeParameters) {
		var p peer.ID
		var err error
		if params.pm == nil {
			p, err = utils.SelectPeer(params.host, FilterSubscribeID_v20beta1, fromThesePeers, params.log)
		} else {
			p, err = params.pm.SelectPeer(FilterSubscribeID_v20beta1, fromThesePeers, params.log)
		}
		if err == nil {
			params.selectedPeer = p
		} else {
			params.log.Info("selecting peer", zap.Error(err))
		}
	}
}

// WithFastestPeerSelection is an option used to select a peer from the peer store
// with the lowest ping If a list of specific peers is passed, the peer will be chosen
// from that list assuming it supports the chosen protocol, otherwise it will chose a
// peer from the node peerstore
func WithFastestPeerSelection(ctx context.Context, fromThesePeers ...peer.ID) FilterSubscribeOption {
	return func(params *FilterSubscribeParameters) {
		p, err := utils.SelectPeerWithLowestRTT(ctx, params.host, FilterSubscribeID_v20beta1, fromThesePeers, params.log)
		if err == nil {
			params.selectedPeer = p
		} else {
			params.log.Info("selecting peer", zap.Error(err))
		}
	}
}

// WithRequestID is an option to set a specific request ID to be used when
// creating a filter subscription
func WithRequestID(requestID []byte) FilterSubscribeOption {
	return func(params *FilterSubscribeParameters) {
		params.requestID = requestID
	}
}

// WithAutomaticRequestID is an option to automatically generate a request ID
// when creating a filter subscription
func WithAutomaticRequestID() FilterSubscribeOption {
	return func(params *FilterSubscribeParameters) {
		params.requestID = protocol.GenerateRequestId()
	}
}

func DefaultSubscriptionOptions() []FilterSubscribeOption {
	return []FilterSubscribeOption{
		WithAutomaticPeerSelection(),
		WithAutomaticRequestID(),
	}
}

func UnsubscribeAll() FilterUnsubscribeOption {
	return func(params *FilterUnsubscribeParameters) {
		params.unsubscribeAll = true
	}
}

func Peer(p peer.ID) FilterUnsubscribeOption {
	return func(params *FilterUnsubscribeParameters) {
		params.selectedPeer = p
	}
}

// RequestID is an option to set a specific request ID to be used when
// removing a subscription from a filter node
func RequestID(requestID []byte) FilterUnsubscribeOption {
	return func(params *FilterUnsubscribeParameters) {
		params.requestID = requestID
	}
}

func AutomaticRequestId() FilterUnsubscribeOption {
	return func(params *FilterUnsubscribeParameters) {
		params.requestID = protocol.GenerateRequestId()
	}
}

// WithWaitGroup allos specigying a waitgroup to wait until all
// unsubscribe requests are complete before the function is complete
func WithWaitGroup(wg *sync.WaitGroup) FilterUnsubscribeOption {
	return func(params *FilterUnsubscribeParameters) {
		params.wg = wg
	}
}

// DontWait is used to fire and forget an unsubscription, and don't
// care about the results of it
func DontWait() FilterUnsubscribeOption {
	return func(params *FilterUnsubscribeParameters) {
		params.wg = nil
	}
}

func DefaultUnsubscribeOptions() []FilterUnsubscribeOption {
	return []FilterUnsubscribeOption{
		AutomaticRequestId(),
		WithWaitGroup(&sync.WaitGroup{}),
	}
}

func WithMaxSubscribers(maxSubscribers int) Option {
	return func(params *FilterParameters) {
		params.MaxSubscribers = maxSubscribers
	}
}

func DefaultOptions() []Option {
	return []Option{
		WithTimeout(24 * time.Hour),
		WithMaxSubscribers(DefaultMaxSubscriptions),
	}
}
