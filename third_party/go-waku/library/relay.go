package library

import (
	"context"
	"time"

	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

// RelayEnoughPeers determines if there are enough peers to publish a message on a topic
func RelayEnoughPeers(instance *WakuInstance, topic string) (bool, error) {
	if err := validateInstance(instance, MustBeStarted); err != nil {
		return false, err
	}

	topicToCheck := protocol.DefaultPubsubTopic{}.String()
	if topic != "" {
		topicToCheck = topic
	}

	return instance.node.Relay().EnoughPeersToPublishToTopic(topicToCheck), nil
}

func relayPublish(instance *WakuInstance, msg *pb.WakuMessage, pubsubTopic string, ms int) (string, error) {
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

	hash, err := instance.node.Relay().Publish(ctx, msg, relay.WithPubSubTopic(pubsubTopic))

	return hash.String(), err
}

// RelayPublish publishes a message using waku relay and returns the message ID
func RelayPublish(instance *WakuInstance, messageJSON string, topic string, ms int) (string, error) {
	msg, err := wakuMessage(messageJSON)
	if err != nil {
		return "", err
	}

	return relayPublish(instance, msg, topic, int(ms))
}

func relaySubscribe(instance *WakuInstance, filterJSON string) error {
	cf, err := toContentFilter(filterJSON)
	if err != nil {
		return err
	}

	subscriptions, err := instance.node.Relay().Subscribe(context.Background(), cf)
	if err != nil {
		return err
	}

	for _, sub := range subscriptions {
		go func(subscription *relay.Subscription) {
			defer utils.LogOnPanic()
			for envelope := range subscription.Ch {
				send(instance, "message", toSubscriptionMessage(envelope))
			}
		}(sub)
	}

	return nil
}

// RelaySubscribe subscribes to a WakuRelay topic.
func RelaySubscribe(instance *WakuInstance, contentFilterJSON string) error {
	if err := validateInstance(instance, MustBeStarted); err != nil {
		return err
	}

	return relaySubscribe(instance, contentFilterJSON)
}

// RelayTopics returns a list of pubsub topics the node is subscribed to in WakuRelay
func RelayTopics(instance *WakuInstance) (string, error) {
	if err := validateInstance(instance, MustBeStarted); err != nil {
		return "", err
	}

	return marshalJSON(instance.node.Relay().Topics())
}

// RelayUnsubscribe closes the pubsub subscription to a pubsub topic
func RelayUnsubscribe(instance *WakuInstance, contentFilterJSON string) error {
	cf, err := toContentFilter(contentFilterJSON)
	if err != nil {
		return err
	}

	if err := validateInstance(instance, MustBeStarted); err != nil {
		return err
	}

	return instance.node.Relay().Unsubscribe(context.Background(), cf)
}
