package gowaku

import (
	"github.com/waku-org/go-waku/library"
)

// RelayEnoughPeers determines if there are enough peers to publish a message on a topic
func RelayEnoughPeers(instanceID uint, topic string) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	response, err := library.RelayEnoughPeers(instance, topic)
	return prepareJSONResponse(response, err)
}

// RelayPublish publishes a message using waku relay and returns the message ID
func RelayPublish(instanceID uint, messageJSON string, topic string, ms int) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	hash, err := library.RelayPublish(instance, messageJSON, topic, ms)
	return prepareJSONResponse(hash, err)
}

// RelaySubscribe subscribes to a WakuRelay topic.
func RelaySubscribe(instanceID uint, topic string) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = library.RelaySubscribe(instance, topic)
	return makeJSONResponse(err)
}

// RelayTopics returns a list of pubsub topics the node is subscribed to in WakuRelay
func RelayTopics(instanceID uint) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	topics, err := library.RelayTopics(instance)
	return prepareJSONResponse(topics, err)
}

// RelayUnsubscribe closes the pubsub subscription to a pubsub topic
func RelayUnsubscribe(instanceID uint, topic string) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = library.RelayUnsubscribe(instance, topic)
	return makeJSONResponse(err)
}
