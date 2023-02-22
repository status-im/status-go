package lightpush

import (
	"context"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/utils"
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
// to push a waku message to. If a list of specific peers is passed, the peer will be chosen
// from that list assuming it supports the chosen protocol, otherwise it will chose a peer
// from the node peerstore
func WithAutomaticPeerSelection(fromThesePeers ...peer.ID) LightPushOption {
	return func(params *LightPushParameters) {
		p, err := utils.SelectPeer(params.host, LightPushID_v20beta1, fromThesePeers, params.log)
		if err == nil {
			params.selectedPeer = p
		} else {
			params.log.Info("selecting peer", zap.Error(err))
		}
	}
}

// WithFastestPeerSelection is an option used to select a peer from the peer store
// with the lowest ping. If a list of specific peers is passed, the peer will be chosen
// from that list assuming it supports the chosen protocol, otherwise it will chose a peer
// from the node peerstore
func WithFastestPeerSelection(ctx context.Context, fromThesePeers ...peer.ID) LightPushOption {
	return func(params *LightPushParameters) {
		p, err := utils.SelectPeerWithLowestRTT(ctx, params.host, LightPushID_v20beta1, fromThesePeers, params.log)
		if err == nil {
			params.selectedPeer = p
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
