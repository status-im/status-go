package main

/*
#include <cgo_utils.h>
*/
import "C"

import (
	"unsafe"

	"github.com/waku-org/go-waku/library"
)

// Returns a list of objects containing the peerID, enr and multiaddresses for each node found
//
//	given a url to a DNS discoverable ENR tree
//
// The nameserver can optionally be specified to resolve the enrtree url. Otherwise NULL or
// empty to automatically use the default system dns.
// If ms is greater than 0, the subscription must happen before the timeout
// (in milliseconds) is reached, or an error will be returned
//
//export waku_dns_discovery
func waku_dns_discovery(ctx unsafe.Pointer, url *C.char, nameserver *C.char, ms C.int, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	return singleFnExec(func(instance *library.WakuInstance) (string, error) {
		return library.DNSDiscovery(C.GoString(url), C.GoString(nameserver), int(ms))
	}, ctx, cb, userData)
}

// Update the bootnode list used for discovering new peers via DiscoveryV5
// The bootnodes param should contain a JSON array containing the bootnode ENRs i.e. `["enr:...", "enr:..."]`
//
//export waku_discv5_update_bootnodes
func waku_discv5_update_bootnodes(ctx unsafe.Pointer, bootnodes *C.char, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	instance, err := getInstance(ctx)
	if err != nil {
		onError(err, cb, userData)
	}

	err = library.SetBootnodes(instance, C.GoString(bootnodes))
	return onError(err, cb, userData)
}
