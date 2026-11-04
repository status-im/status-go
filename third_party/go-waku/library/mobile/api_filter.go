package gowaku

import (
	"github.com/waku-org/go-waku/library"
)

// FilterSubscribe is used to create a subscription to a filter node to receive messages
func FilterSubscribe(instanceID uint, filterJSON string, peerID string, ms int) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	response, err := library.FilterSubscribe(instance, filterJSON, peerID, ms)
	return prepareJSONResponse(response, err)
}

// FilterPing is used to determine if a peer has an active subscription
func FilterPing(instanceID uint, peerID string, ms int) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = library.FilterPing(instance, peerID, ms)
	return makeJSONResponse(err)
}

// FilterUnsubscribe is used to remove a filter criteria from an active subscription with a filter node
func FilterUnsubscribe(instanceID uint, filterJSON string, peerID string, ms int) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = library.FilterUnsubscribe(instance, filterJSON, peerID, ms)
	return makeJSONResponse(err)
}

// FilterUnsubscribeAll is used to remove an active subscription to a peer. If no peerID is defined, it will stop all active filter subscriptions
func FilterUnsubscribeAll(instanceID uint, peerID string, ms int) string {
	instance, err := library.GetInstance(instanceID)
	if err != nil {
		return makeJSONResponse(err)
	}

	response, err := library.FilterUnsubscribeAll(instance, peerID, ms)
	return prepareJSONResponse(response, err)
}
