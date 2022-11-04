package filter

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/status-im/go-waku/waku/v2/utils"
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
		timeout time.Duration
	}

	Option func(*FilterParameters)
)

func WithTimeout(timeout time.Duration) Option {
	return func(params *FilterParameters) {
		params.timeout = timeout
	}
}

func WithPeer(p peer.ID) FilterSubscribeOption {
	return func(params *FilterSubscribeParameters) {
		params.selectedPeer = p
	}
}

func WithAutomaticPeerSelection() FilterSubscribeOption {
	return func(params *FilterSubscribeParameters) {
		p, err := utils.SelectPeer(params.host, string(FilterID_v20beta1), params.log)
		if err == nil {
			params.selectedPeer = *p
		} else {
			params.log.Info("selecting peer", zap.Error(err))
		}
	}
}

func WithFastestPeerSelection(ctx context.Context) FilterSubscribeOption {
	return func(params *FilterSubscribeParameters) {
		p, err := utils.SelectPeerWithLowestRTT(ctx, params.host, string(FilterID_v20beta1), params.log)
		if err == nil {
			params.selectedPeer = *p
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
