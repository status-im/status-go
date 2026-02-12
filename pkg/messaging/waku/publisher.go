//go:build use_nwaku
// +build use_nwaku

package wakuv2

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/logos-messaging/logos-messaging-go-bindings/waku"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"

	"github.com/waku-org/go-waku/waku/v2/api/publish"
)

type nwakuPublisher struct {
	node *waku.WakuNode
}

func newPublisher(node *waku.WakuNode) publish.Publisher {
	return &nwakuPublisher{
		node: node,
	}
}

func (p *nwakuPublisher) RelayListPeers(pubsubTopic string) ([]peer.ID, error) {
	// TODO-nwaku
	return nil, nil
}

func (p *nwakuPublisher) RelayPublish(ctx context.Context, message *pb.WakuMessage, pubsubTopic string) (pb.MessageHash, error) {
	// TODO-nwaku improve this workaround to use the pb definition of the hash

	// Temporary conversion to "nwaku" WakuMessage
	nwakuWakuMessage := &pb.WakuMessage{
		Payload:    message.Payload,
		ContentTopic:   message.ContentTopic,
		Version:        message.Version,
		Timestamp:      message.Timestamp,
		Meta:           message.Meta,
		Ephemeral:      message.Ephemeral,
		RateLimitProof: message.RateLimitProof,
	}

	hexHash, err := p.node.RelayPublish(ctx, nwakuWakuMessage, pubsubTopic)
	if err != nil {
		return pb.MessageHash{}, err
	}

	// Notice the simple conversion. Same definition but different packages.
	hexHashBytes, err := hexHash.Bytes()
	if err != nil {
		return pb.MessageHash{}, err
	}
	return pb.ToMessageHash(hexHashBytes), nil
}

// LightpushPublish publishes a message via WakuLightPush
func (p *nwakuPublisher) LightpushPublish(ctx context.Context, message *pb.WakuMessage, pubsubTopic string, maxPeers int) (pb.MessageHash, error) {
	// TODO-nwaku
	return pb.MessageHash{}, nil
}
