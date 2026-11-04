package library

import (
	"context"
	"time"

	"github.com/waku-org/go-waku/waku/v2/protocol/pb"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
)

func lightpushPublish(instance *WakuInstance, msg *pb.WakuMessage, pubsubTopic string, peerID string, ms int) (string, error) {
	if err := validateInstance(instance, MustBeStarted); err != nil {
		return "", err
	}

	var ctx context.Context
	var cancel context.CancelFunc

	if ms > 0 {
		ctx, cancel = context.WithTimeout(instance.ctx, time.Duration(int(ms))*time.Millisecond)
		defer cancel()
	} else {
		ctx = instance.ctx
	}

	var lpOptions []lightpush.RequestOption
	if peerID != "" {
		p, err := peer.Decode(peerID)
		if err != nil {
			return "", err
		}
		lpOptions = append(lpOptions, lightpush.WithPeer(p))
	} else {
		lpOptions = append(lpOptions, lightpush.WithAutomaticPeerSelection())
	}

	if pubsubTopic != "" {
		lpOptions = append(lpOptions, lightpush.WithPubSubTopic(pubsubTopic))
	}

	hash, err := instance.node.Lightpush().Publish(ctx, msg, lpOptions...)

	return hash.String(), err
}

// LightpushPublish is used to publish a WakuMessage in a pubsub topic using Lightpush protocol
func LightpushPublish(instance *WakuInstance, messageJSON string, pubsubTopic string, peerID string, ms int) (string, error) {
	msg, err := wakuMessage(messageJSON)
	if err != nil {
		return "", err
	}

	return lightpushPublish(instance, msg, pubsubTopic, peerID, ms)
}
