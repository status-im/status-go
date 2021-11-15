package lightpush

import (
	"context"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/status-im/go-waku/waku/v2/protocol"
	"github.com/status-im/go-waku/waku/v2/utils"
)

type LightPushParameters struct {
	host         host.Host
	selectedPeer peer.ID
	requestId    []byte
}

type LightPushOption func(*LightPushParameters)

func WithPeer(p peer.ID) LightPushOption {
	return func(params *LightPushParameters) {
		params.selectedPeer = p
	}
}

func WithAutomaticPeerSelection(host host.Host) LightPushOption {
	return func(params *LightPushParameters) {
		p, err := utils.SelectPeer(host, string(LightPushID_v20beta1))
		if err == nil {
			params.selectedPeer = *p
		} else {
			log.Info("Error selecting peer: ", err)
		}
	}
}

func WithFastestPeerSelection(ctx context.Context) LightPushOption {
	return func(params *LightPushParameters) {
		p, err := utils.SelectPeerWithLowestRTT(ctx, params.host, string(LightPushID_v20beta1))
		if err == nil {
			params.selectedPeer = *p
		} else {
			log.Info("Error selecting peer: ", err)
		}
	}
}

func WithRequestId(requestId []byte) LightPushOption {
	return func(params *LightPushParameters) {
		params.requestId = requestId
	}
}

func WithAutomaticRequestId() LightPushOption {
	return func(params *LightPushParameters) {
		params.requestId = protocol.GenerateRequestId()
	}
}

func DefaultOptions(host host.Host) []LightPushOption {
	return []LightPushOption{
		WithAutomaticRequestId(),
		WithAutomaticPeerSelection(host),
	}
}
