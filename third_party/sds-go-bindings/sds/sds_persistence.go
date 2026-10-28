//go:build !lint

package sds

/*
#define RET_OK 0
#define RET_ERR 1
#include <stdint.h>
#include <stdlib.h>
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"unsafe"
)

const (
	persistSaveChannelMeta = iota + 1
	persistUpdateHistory
	persistLoadChannel
	persistDropChannel
)

func setPersistenceResponse(response **C.char, responseLen *C.size_t, data []byte) {
	if len(data) == 0 {
		return
	}
	*response = (*C.char)(C.CBytes(data))
	*responseLen = C.size_t(len(data))
}

//export sdsGlobalPersistenceCallback
func sdsGlobalPersistenceCallback(
	operation C.int,
	channelID *C.char,
	payload *C.uint8_t,
	payloadLen C.size_t,
	response **C.char,
	responseLen *C.size_t,
	ctx C.uintptr_t,
) C.int {
	rm, ok := lookupReliabilityManager(uintptr(ctx))
	if !ok {
		setPersistenceResponse(response, responseLen, []byte("SDS reliability manager is unavailable"))
		return C.RET_ERR
	}
	if !rm.persistence.enabled() {
		setPersistenceResponse(response, responseLen, []byte("SDS persistence callback is not configured"))
		return C.RET_ERR
	}

	channel := C.GoString(channelID)
	var payloadBytes []byte
	if payloadLen > 0 {
		payloadBytes = C.GoBytes(unsafe.Pointer(payload), C.int(payloadLen))
	}

	var result []byte
	var err error
	switch operation {
	case persistSaveChannelMeta:
		err = rm.persistence.SaveChannelMeta(channel, payloadBytes)
	case persistUpdateHistory:
		var update HistoryUpdate
		if err = json.Unmarshal(payloadBytes, &update); err == nil {
			err = rm.persistence.UpdateHistory(channel, update)
		}
	case persistLoadChannel:
		var data ChannelData
		data, err = rm.persistence.LoadChannel(channel)
		if err == nil {
			result, err = json.Marshal(data)
		}
	case persistDropChannel:
		err = rm.persistence.DropChannel(channel)
	default:
		err = fmt.Errorf("unknown SDS persistence operation: %d", int(operation))
	}

	if err != nil {
		setPersistenceResponse(response, responseLen, []byte(err.Error()))
		return C.RET_ERR
	}

	setPersistenceResponse(response, responseLen, result)
	return C.RET_OK
}
