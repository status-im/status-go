package filter

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

type (
	FilterSubscribeParameters struct {
		host         host.Host
		selectedPeer peer.ID
		log          *zap.Logger
	}

	FilterSubscribeOption func(*FilterSubscribeParameters)

	FilterParameters struct {
		Timeout time.Duration
	}

	Option func(*FilterParameters)
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
		p, err := utils.SelectPeer(params.host, FilterID_v20beta1, fromThesePeers, params.log)
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
		p, err := utils.SelectPeerWithLowestRTT(ctx, params.host, FilterID_v20beta1, fromThesePeers, params.log)
		if err == nil {
			params.selectedPeer = p
		} else {
			params.log.Info("selecting peer", zap.Error(err))
		}
	}
}

func DefaultOptions() []Option {
	return []Option{
		WithTimeout(24 * time.Hour),
	}
}

func DefaultSubscribtionOptions() []FilterSubscribeOption {
	return []FilterSubscribeOption{
		WithAutomaticPeerSelection(),
	}
}
