package peer_exchange

import (
	"context"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

type PeerExchangeParameters struct {
	host         host.Host
	selectedPeer peer.ID
	log          *zap.Logger
}

type PeerExchangeOption func(*PeerExchangeParameters)

// WithPeer is an option used to specify the peerID to push a waku message to
func WithPeer(p peer.ID) PeerExchangeOption {
	return func(params *PeerExchangeParameters) {
		params.selectedPeer = p
	}
}

// WithAutomaticPeerSelection is an option used to randomly select a peer from the peer store
// to push a waku message to
func WithAutomaticPeerSelection() PeerExchangeOption {
	return func(params *PeerExchangeParameters) {
		p, err := utils.SelectPeer(params.host, string(PeerExchangeID_v20alpha1), params.log)
		if err == nil {
			params.selectedPeer = *p
		} else {
			params.log.Info("selecting peer", zap.Error(err))
		}
	}
}

// WithFastestPeerSelection is an option used to select a peer from the peer store
// with the lowest ping
func WithFastestPeerSelection(ctx context.Context) PeerExchangeOption {
	return func(params *PeerExchangeParameters) {
		p, err := utils.SelectPeerWithLowestRTT(ctx, params.host, string(PeerExchangeID_v20alpha1), params.log)
		if err == nil {
			params.selectedPeer = *p
		} else {
			params.log.Info("selecting peer", zap.Error(err))
		}
	}
}

// DefaultOptions are the default options to be used when using the lightpush protocol
func DefaultOptions(host host.Host) []PeerExchangeOption {
	return []PeerExchangeOption{
		WithAutomaticPeerSelection(),
	}
}
