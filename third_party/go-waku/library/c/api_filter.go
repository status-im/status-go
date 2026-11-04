package main

/*
#include <cgo_utils.h>
*/
import "C"
import (
	"unsafe"

	"github.com/waku-org/go-waku/library"
)

// Creates a subscription to a filter full node matching a content filter.
// filterJSON must contain a JSON with this format:
//
//		{
//		  "pubsubTopic": "the pubsub topic" // optional if using autosharding, mandatory if using static or named sharding.
//	      "contentTopics": ["the content topic"] // mandatory, at least one required, with a max of 10
//		}
//
// peerID should contain the ID of a peer supporting the filter protocol. Use NULL to automatically select a node
// If ms is greater than 0, the subscription must happen before the timeout
// (in milliseconds) is reached, or an error will be returned
// It returns a json object containing the details of the subscriptions along with any errors in case of partial failures
//
//export waku_filter_subscribe
func waku_filter_subscribe(ctx unsafe.Pointer, filterJSON *C.char, peerID *C.char, ms C.int, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	return singleFnExec(func(instance *library.WakuInstance) (string, error) {
		return library.FilterSubscribe(instance, C.GoString(filterJSON), C.GoString(peerID), int(ms))
	}, ctx, cb, userData)
}

// Used to know if a service node has an active subscription for this client
// peerID should contain the ID of a peer we are subscribed to, supporting the filter protocol
// If ms is greater than 0, the subscription must happen before the timeout
// (in milliseconds) is reached, or an error will be returned
//
//export waku_filter_ping
func waku_filter_ping(ctx unsafe.Pointer, peerID *C.char, ms C.int, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	instance, err := getInstance(ctx)
	if err != nil {
		onError(err, cb, userData)
	}

	err = library.FilterPing(instance, C.GoString(peerID), int(ms))
	return onError(err, cb, userData)
}

// Sends a requests to a service node to stop pushing messages matching this filter to this client.
// It might be used to modify an existing subscription by providing a subset of the original filter
// criteria
//
//		{
//		  "pubsubTopic": "the pubsub topic" //  optional if using autosharding, mandatory if using static or named sharding.
//	      "contentTopics": ["the content topic"] // mandatory, at least one required, with a max of 10
//		}
//
// peerID should contain the ID of a peer this client is subscribed to.
// If ms is greater than 0, the subscription must happen before the timeout
// (in milliseconds) is reached, or an error will be returned
//
//export waku_filter_unsubscribe
func waku_filter_unsubscribe(ctx unsafe.Pointer, filterJSON *C.char, peerID *C.char, ms C.int, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	instance, err := getInstance(ctx)
	if err != nil {
		onError(err, cb, userData)
	}

	err = library.FilterUnsubscribe(instance, C.GoString(filterJSON), C.GoString(peerID), int(ms))
	return onError(err, cb, userData)
}

// Sends a requests to a service node (or all service nodes) to stop pushing messages
// peerID should contain the ID of a peer this client is subscribed to, or can be NULL to
// stop all active subscriptions
// If ms is greater than 0, the subscription must happen before the timeout
// (in milliseconds) is reached, or an error will be returned
//
//export waku_filter_unsubscribe_all
func waku_filter_unsubscribe_all(ctx unsafe.Pointer, peerID *C.char, ms C.int, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	return singleFnExec(func(instance *library.WakuInstance) (string, error) {
		return library.FilterUnsubscribeAll(instance, C.GoString(peerID), int(ms))
	}, ctx, cb, userData)
}
