package publish

import (
	"context"
	"errors"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
)

var ErrRelayNotAvailable = errors.New("relay is not available")
var ErrLightpushNotAvailable = errors.New("lightpush is not available")

func NewDefaultPublisher(lightpush *lightpush.WakuLightPush, relay *relay.WakuRelay) Publisher {
	return &defaultPublisher{
		lightpush: lightpush,
		relay:     relay,
	}
}

type defaultPublisher struct {
	lightpush *lightpush.WakuLightPush
	relay     *relay.WakuRelay
}

func (d *defaultPublisher) RelayListPeers(pubsubTopic string) ([]peer.ID, error) {
	if d.relay == nil {
		return nil, ErrRelayNotAvailable
	}

	return d.relay.PubSub().ListPeers(pubsubTopic), nil
}

func (d *defaultPublisher) RelayPublish(ctx context.Context, message *pb.WakuMessage, pubsubTopic string) (pb.MessageHash, error) {
	if d.relay == nil {
		return pb.MessageHash{}, ErrRelayNotAvailable
	}

	return d.relay.Publish(ctx, message, relay.WithPubSubTopic(pubsubTopic))
}

func (d *defaultPublisher) LightpushPublish(ctx context.Context, message *pb.WakuMessage, pubsubTopic string, maxPeers int) (pb.MessageHash, error) {
	if d.lightpush == nil {
		return pb.MessageHash{}, ErrLightpushNotAvailable
	}

	return d.lightpush.Publish(ctx, message, lightpush.WithPubSubTopic(pubsubTopic), lightpush.WithMaxPeers(maxPeers))
}
