package lightpush

import (
	"context"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

type LightPushParameters struct {
	host         host.Host
	selectedPeer peer.ID
	requestId    []byte
	log          *zap.Logger
}

type LightPushOption func(*LightPushParameters)

// WithPeer is an option used to specify the peerID to push a waku message to
func WithPeer(p peer.ID) LightPushOption {
	return func(params *LightPushParameters) {
		params.selectedPeer = p
	}
}

// WithAutomaticPeerSelection is an option used to randomly select a peer from the peer store
// to push a waku message to
func WithAutomaticPeerSelection() LightPushOption {
	return func(params *LightPushParameters) {
		p, err := utils.SelectPeer(params.host, string(LightPushID_v20beta1), params.log)
		if err == nil {
			params.selectedPeer = *p
		} else {
			params.log.Info("selecting peer", zap.Error(err))
		}
	}
}

// WithFastestPeerSelection is an option used to select a peer from the peer store
// with the lowest ping
func WithFastestPeerSelection(ctx context.Context) LightPushOption {
	return func(params *LightPushParameters) {
		p, err := utils.SelectPeerWithLowestRTT(ctx, params.host, string(LightPushID_v20beta1), params.log)
		if err == nil {
			params.selectedPeer = *p
		} else {
			params.log.Info("selecting peer", zap.Error(err))
		}
	}
}

// WithRequestId is an option to set a specific request ID to be used when
// publishing a message
func WithRequestId(requestId []byte) LightPushOption {
	return func(params *LightPushParameters) {
		params.requestId = requestId
	}
}

// WithAutomaticRequestId is an option to automatically generate a request ID
// when publishing a message
func WithAutomaticRequestId() LightPushOption {
	return func(params *LightPushParameters) {
		params.requestId = protocol.GenerateRequestId()
	}
}

// DefaultOptions are the default options to be used when using the lightpush protocol
func DefaultOptions(host host.Host) []LightPushOption {
	return []LightPushOption{
		WithAutomaticRequestId(),
		WithAutomaticPeerSelection(),
	}
}
