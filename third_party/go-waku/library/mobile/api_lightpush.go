package gowaku

import "github.com/waku-org/go-waku/library"

// LightpushPublish is used to publish a WakuMessage in a pubsub topic using Lightpush protocol
func LightpushPublish(instanceID uint, messageJSON string, topic string, peerID string, ms int) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	response, err := library.LightpushPublish(instance, messageJSON, topic, peerID, ms)
	return prepareJSONResponse(response, err)
}
