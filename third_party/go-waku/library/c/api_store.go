package main

/*
#include <cgo_utils.h>
*/
import "C"
import (
	"unsafe"

	"github.com/waku-org/go-waku/library"
)

// Query historic messages using waku store protocol.
// queryJSON must contain a valid json string with the following format:
//
//	{
//		"pubsubTopic": "...", // optional string
//		"startTime": 1234, // optional, unix epoch time in nanoseconds
//		"endTime": 1234, // optional, unix epoch time in nanoseconds
//		"contentTopics": [ // optional
//			"contentTopic1",
//			...
//		],
//		"pagingOptions": {// optional pagination information
//			"pageSize": 40, // number
//			"cursor": { // optional
//				"digest": ...,
//				"receiverTime": ...,
//				"senderTime": ...,
//				"pubsubTopic": ...,
//			},
//			"forward": true, // sort order
//		}
//	}
//
// If a non empty cursor is returned, this function should be executed again, setting  the `cursor` attribute with the cursor returned in the response
// peerID should contain the ID of a peer supporting the store protocol. Use NULL to automatically select a node
// If ms is greater than 0, the broadcast of the message must happen before the timeout
// (in milliseconds) is reached, or an error will be returned
//
//export waku_store_query
func waku_store_query(ctx unsafe.Pointer, queryJSON *C.char, peerID *C.char, ms C.int, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	return singleFnExec(func(instance *library.WakuInstance) (string, error) {
		return library.StoreQuery(instance, C.GoString(queryJSON), C.GoString(peerID), int(ms))
	}, ctx, cb, userData)
}

// Query historic messages stored in the localDB using waku store protocol.
// queryJSON must contain a valid json string with the following format:
//
//	{
//		"pubsubTopic": "...", // optional string
//		"startTime": 1234, // optional, unix epoch time in nanoseconds
//		"endTime": 1234, // optional, unix epoch time in nanoseconds
//		"contentTopics": [ // optional
//			"contentTopic1"
//			...
//		],
//		"pagingOptions": {// optional pagination information
//			"pageSize": 40, // number
//			"cursor": { // optional
//				"digest": ...,
//				"receiverTime": ...,
//				"senderTime": ...,
//				"pubsubTopic": ...,
//			},
//			"forward": true, // sort order
//		}
//	}
//
// If a non empty cursor is returned, this function should be executed again, setting  the `cursor` attribute with the cursor returned in the response
// Requires the `store` option to be passed when setting up the initial configuration
//
//export waku_store_local_query
func waku_store_local_query(ctx unsafe.Pointer, queryJSON *C.char, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	return singleFnExec(func(instance *library.WakuInstance) (string, error) {
		return library.StoreLocalQuery(instance, C.GoString(queryJSON))
	}, ctx, cb, userData)
}
