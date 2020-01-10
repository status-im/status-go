// +build nimbus

package nimbusbridge

// https://golang.org/cmd/cgo/

/*
#include <stddef.h>
#include <stdbool.h>
#include <stdlib.h>
#include <libnimbus.h>
*/
import "C"

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"github.com/status-im/status-go/eth-node/types"
)

type nimbusPublicWhisperAPIWrapper struct {
	filterMessagesMu *sync.Mutex
	filterMessages   *map[string]*list.List
	routineQueue     *RoutineQueue
}

// NewNimbusPublicWhisperAPIWrapper returns an object that wraps Nimbus's PublicWhisperAPI in a types interface
func NewNimbusPublicWhisperAPIWrapper(filterMessagesMu *sync.Mutex, filterMessages *map[string]*list.List, routineQueue *RoutineQueue) types.PublicWhisperAPI {
	return &nimbusPublicWhisperAPIWrapper{
		filterMessagesMu: filterMessagesMu,
		filterMessages:   filterMessages,
		routineQueue:     routineQueue,
	}
}

// AddPrivateKey imports the given private key.
func (w *nimbusPublicWhisperAPIWrapper) AddPrivateKey(ctx context.Context, privateKey types.HexBytes) (string, error) {
	retVal := w.routineQueue.Send(func(c chan<- callReturn) {
		privKeyC := C.CBytes(privateKey)
		defer C.free(unsafe.Pointer(privKeyC))

		idC := C.malloc(C.size_t(C.ID_LEN))
		defer C.free(idC)
		if C.nimbus_add_keypair((*C.uchar)(privKeyC), (*C.uchar)(idC)) {
			c <- callReturn{value: types.EncodeHex(C.GoBytes(idC, C.ID_LEN))}
		} else {
			c <- callReturn{err: errors.New("failed to add private key to Nimbus")}
		}
	})
	if retVal.err != nil {
		return "", retVal.err
	}

	return retVal.value.(string), nil
}

// GenerateSymKeyFromPassword derives a key from the given password, stores it, and returns its ID.
func (w *nimbusPublicWhisperAPIWrapper) GenerateSymKeyFromPassword(ctx context.Context, passwd string) (string, error) {
	retVal := w.routineQueue.Send(func(c chan<- callReturn) {
		passwordC := C.CString(passwd)
		defer C.free(unsafe.Pointer(passwordC))

		idC := C.malloc(C.size_t(C.ID_LEN))
		defer C.free(idC)
		if C.nimbus_add_symkey_from_password(passwordC, (*C.uchar)(idC)) {
			c <- callReturn{value: types.EncodeHex(C.GoBytes(idC, C.ID_LEN))}
		} else {
			c <- callReturn{err: errors.New("failed to add symkey to Nimbus")}
		}
	})
	if retVal.err != nil {
		return "", retVal.err
	}

	return retVal.value.(string), nil
}

// DeleteKeyPair removes the key with the given key if it exists.
func (w *nimbusPublicWhisperAPIWrapper) DeleteKeyPair(ctx context.Context, key string) (bool, error) {
	retVal := w.routineQueue.Send(func(c chan<- callReturn) {
		keyC, err := decodeHexID(key)
		if err != nil {
			c <- callReturn{err: err}
			return
		}
		defer C.free(unsafe.Pointer(keyC))

		c <- callReturn{value: C.nimbus_delete_keypair(keyC)}
	})
	if retVal.err != nil {
		return false, retVal.err
	}

	return retVal.value.(bool), nil
}

// NewMessageFilter creates a new filter that can be used to poll for
// (new) messages that satisfy the given criteria.
func (w *nimbusPublicWhisperAPIWrapper) NewMessageFilter(req types.Criteria) (string, error) {
	// topics := make([]whisper.TopicType, len(req.Topics))
	// for index, tt := range req.Topics {
	// 	topics[index] = whisper.TopicType(tt)
	// }

	// criteria := whisper.Criteria{
	// 	SymKeyID:     req.SymKeyID,
	// 	PrivateKeyID: req.PrivateKeyID,
	// 	Sig:          req.Sig,
	// 	MinPow:       req.MinPow,
	// 	Topics:       topics,
	// 	AllowP2P:     req.AllowP2P,
	// }
	// return w.publicWhisperAPI.NewMessageFilter(criteria)
	// TODO
	return "", errors.New("not implemented")
}

// GetFilterMessages returns the messages that match the filter criteria and
// are received between the last poll and now.
func (w *nimbusPublicWhisperAPIWrapper) GetFilterMessages(id string) ([]*types.Message, error) {
	idC := C.CString(id)
	defer C.free(unsafe.Pointer(idC))

	var (
		messageList *list.List
		ok          bool
	)
	w.filterMessagesMu.Lock()
	defer w.filterMessagesMu.Unlock()
	if messageList, ok = (*w.filterMessages)[id]; !ok {
		return nil, fmt.Errorf("no filter with ID %s", id)
	}

	retVal := make([]*types.Message, messageList.Len())
	if messageList.Len() == 0 {
		return retVal, nil
	}

	elem := messageList.Front()
	index := 0
	for elem != nil {
		retVal[index] = (elem.Value).(*types.Message)
		index++
		next := elem.Next()
		messageList.Remove(elem)
		elem = next
	}
	return retVal, nil
}

// Post posts a message on the Whisper network.
// returns the hash of the message in case of success.
func (w *nimbusPublicWhisperAPIWrapper) Post(ctx context.Context, req types.NewMessage) ([]byte, error) {
	retVal := w.routineQueue.Send(func(c chan<- callReturn) {
		msg := C.post_message{
			ttl:       C.uint32_t(req.TTL),
			powTime:   C.double(req.PowTime),
			powTarget: C.double(req.PowTarget),
		}
		if req.SigID != "" {
			sourceID, err := decodeHexID(req.SigID)
			if err != nil {
				c <- callReturn{err: err}
				return
			}
			msg.sourceID = sourceID
			defer C.free(unsafe.Pointer(sourceID))
		}
		if req.SymKeyID != "" {
			symKeyID, err := decodeHexID(req.SymKeyID)
			if err != nil {
				c <- callReturn{err: err}
				return
			}
			msg.symKeyID = symKeyID
			defer C.free(unsafe.Pointer(symKeyID))
		}
		if req.PublicKey != nil && len(req.PublicKey) > 0 {
			msg.pubKey = (*C.uchar)(C.CBytes(req.PublicKey[1:]))
			defer C.free(unsafe.Pointer(msg.pubKey))
		}
		msg.payloadLen = C.size_t(len(req.Payload))
		msg.payload = (*C.uchar)(C.CBytes(req.Payload))
		defer C.free(unsafe.Pointer(msg.payload))
		msg.paddingLen = C.size_t(len(req.Padding))
		msg.padding = (*C.uchar)(C.CBytes(req.Padding))
		defer C.free(unsafe.Pointer(msg.padding))
		copyTopicToCBuffer(&msg.topic[0], req.Topic[:])

		// TODO: return envelope hash once nimbus_post is improved to return it
		if C.nimbus_post(&msg) {
			c <- callReturn{value: make([]byte, 0)}
			return
		}
		c <- callReturn{err: fmt.Errorf("failed to post message symkeyid=%s pubkey=%#x topic=%#x", req.SymKeyID, req.PublicKey, req.Topic[:])}
		// hashC := C.nimbus_post(&msg)
		// if hashC == nil {
		// 	return nil, errors.New("Nimbus failed to post message")
		// }
		// return hex.DecodeString(C.GoString(hashC))
	})
	if retVal.err != nil {
		return nil, retVal.err
	}

	return retVal.value.([]byte), nil
}
