package main

/*
#include <cgo_utils.h>
*/
import "C"
import (
	"unsafe"

	"github.com/waku-org/go-waku/library"
)

// Publish a message using waku lightpush. Use NULL for topic to derive the pubsub topic from the contentTopic.
// peerID should contain the ID of a peer supporting the lightpush protocol. Use NULL to automatically select a node
// If ms is greater than 0, the broadcast of the message must happen before the timeout
// (in milliseconds) is reached, or an error will be returned
//
//export waku_lightpush_publish
func waku_lightpush_publish(ctx unsafe.Pointer, messageJSON *C.char, topic *C.char, peerID *C.char, ms C.int, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	return singleFnExec(func(instance *library.WakuInstance) (string, error) {
		return library.LightpushPublish(instance, C.GoString(messageJSON), C.GoString(topic), C.GoString(peerID), int(ms))
	}, ctx, cb, userData)
}
