//go:build use_nwaku
// +build use_nwaku

package wakuv2

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/waku-org/go-waku/waku/v2/api/publish"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
)

type nwakuPublisher struct {
	node *WakuNode
}

func newPublisher(node *WakuNode) publish.Publisher {
	return &nwakuPublisher{
		node: node,
	}
}

func (p *nwakuPublisher) RelayListPeers(pubsubTopic string) ([]peer.ID, error) {
	// TODO-nwaku
	return nil, nil
}

func (p *nwakuPublisher) RelayPublish(ctx context.Context, message *pb.WakuMessage, pubsubTopic string) (pb.MessageHash, error) {
	return p.node.RelayPublish(ctx, message, pubsubTopic)
}

// LightpushPublish publishes a message via WakuLightPush
func (p *nwakuPublisher) LightpushPublish(ctx context.Context, message *pb.WakuMessage, pubsubTopic string, maxPeers int) (pb.MessageHash, error) {
	// TODO-nwaku
	return pb.MessageHash{}, nil
}
