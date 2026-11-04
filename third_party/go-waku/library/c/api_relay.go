package main

/*
#include <cgo_utils.h>
*/
import "C"
import (
	"unsafe"

	"github.com/waku-org/go-waku/library"
)

// Determine if there are enough peers to publish a message on a topic. Use NULL
// to verify the number of peers in the default pubsub topic
//
//export waku_relay_enough_peers
func waku_relay_enough_peers(ctx unsafe.Pointer, topic *C.char, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	return singleFnExec(func(instance *library.WakuInstance) (string, error) {
		result, err := library.RelayEnoughPeers(instance, C.GoString(topic))
		if result {
			return "true", err
		}
		return "false", err
	}, ctx, cb, userData)
}

// Publish a message using waku relay and returns the message ID. Use NULL for topic to derive the pubsub topic from the contentTopic.
// If ms is greater than 0, the broadcast of the message must happen before the timeout
// (in milliseconds) is reached, or an error will be returned.
//
//export waku_relay_publish
func waku_relay_publish(ctx unsafe.Pointer, messageJSON *C.char, topic *C.char, ms C.int, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	return singleFnExec(func(instance *library.WakuInstance) (string, error) {
		return library.RelayPublish(instance, C.GoString(messageJSON), C.GoString(topic), int(ms))
	}, ctx, cb, userData)
}

// Subscribe to WakuRelay to receive messages matching a content filter.
// filterJSON must contain a JSON with this format:
//
//		{
//		  "pubsubTopic": "the pubsub topic" // optional if using autosharding, mandatory if using static or named sharding.
//	      "contentTopics": ["the content topic"] // optional
//		}
//
// When a message is received, a "message" event is emitted containing the message and pubsub topic in which
// the message was received
//
//export waku_relay_subscribe
func waku_relay_subscribe(ctx unsafe.Pointer, filterJSON *C.char, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	instance, err := getInstance(ctx)
	if err != nil {
		onError(err, cb, userData)
	}

	err = library.RelaySubscribe(instance, C.GoString(filterJSON))
	return onError(err, cb, userData)
}

// Returns a json response with the list of pubsub topics the node
// is subscribed to in WakuRelay
//
//export waku_relay_topics
func waku_relay_topics(ctx unsafe.Pointer, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	return singleFnExec(func(instance *library.WakuInstance) (string, error) {
		return library.RelayTopics(instance)
	}, ctx, cb, userData)
}

// Closes the pubsub subscription to stop receiving messages matching a content filter
// filterJSON must contain a JSON with this format:
//
//		{
//		  "pubsubTopic": "the pubsub topic" // optional if using autosharding, mandatory if using static or named sharding.
//	      "contentTopics": ["the content topic"] // optional
//		}
//
//export waku_relay_unsubscribe
func waku_relay_unsubscribe(ctx unsafe.Pointer, filterJSON *C.char, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	instance, err := getInstance(ctx)
	if err != nil {
		onError(err, cb, userData)
	}

	err = library.RelayUnsubscribe(instance, C.GoString(filterJSON))
	return onError(err, cb, userData)
}
