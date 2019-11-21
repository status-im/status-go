// +build nimbus

package nimbusbridge

// https://golang.org/cmd/cgo/

/*
#cgo LDFLAGS: -Wl,-rpath,'$ORIGIN' -L${SRCDIR} -lnimbus -lpcre -lm
#include <stddef.h>
#include <stdbool.h>
#include <stdlib.h>
#include <libnimbus.h>
void onMessageHandler_cgo(received_message * msg, void* udata); // Forward declaration.
*/
import "C"

import (
	"container/list"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/ethereum/go-ethereum/crypto"
	gopointer "github.com/mattn/go-pointer"
	whispertypes "github.com/status-im/status-go/protocol/transport/whisper/types"
	protocol "github.com/status-im/status-go/protocol/types"
)

type nimbusWhisperWrapper struct {
	timesource       func() time.Time
	filters          map[string]whispertypes.Filter
	filterMessagesMu sync.Mutex
	filterMessages   map[string]*list.List
	routineQueue     *RoutineQueue
	tid              int
}

// NewNimbusWhisperWrapper returns an object that wraps Nimbus' Whisper in a whispertypes interface
func NewNimbusWhisperWrapper() whispertypes.Whisper {
	return &nimbusWhisperWrapper{
		timesource:     func() time.Time { return time.Now() },
		filters:        map[string]whispertypes.Filter{},
		filterMessages: map[string]*list.List{},
		routineQueue:   NewRoutineQueue(),
		tid:            syscall.Gettid(),
	}
}

func (w *nimbusWhisperWrapper) PublicWhisperAPI() whispertypes.PublicWhisperAPI {
	return NewNimbusPublicWhisperAPIWrapper(&w.filterMessagesMu, &w.filterMessages, w.routineQueue)
}

func (w *nimbusWhisperWrapper) Poll() {
	if syscall.Gettid() != w.tid {
		panic("Poll called from wrong thread")
	}

	C.nimbus_poll()

	w.routineQueue.HandleEvent()
}

// MinPow returns the PoW value required by this node.
func (w *nimbusWhisperWrapper) MinPow() float64 {
	return w.routineQueue.Send(func(c chan<- interface{}) {
		c <- float64(C.nimbus_get_min_pow())
	}).(float64)
}

// BloomFilter returns the aggregated bloom filter for all the topics of interest.
// The nodes are required to send only messages that match the advertised bloom filter.
// If a message does not match the bloom, it will tantamount to spam, and the peer will
// be disconnected.
func (w *nimbusWhisperWrapper) BloomFilter() []byte {
	return w.routineQueue.Send(func(c chan<- interface{}) {
		// Allocate a buffer for Nimbus to return the bloom filter on
		dataC := C.malloc(C.size_t(C.BLOOM_LEN))
		defer C.free(unsafe.Pointer(dataC))

		C.nimbus_get_bloom_filter((*C.uchar)(unsafe.Pointer(dataC)))

		// Move the returned data into a Go array
		data := make([]byte, C.BLOOM_LEN)
		copy(data, C.GoBytes(dataC, C.BLOOM_LEN))
		c <- data
	}).([]byte)
}

// GetCurrentTime returns current time.
func (w *nimbusWhisperWrapper) GetCurrentTime() time.Time {
	return w.timesource()
}

// SetTimeSource assigns a particular source of time to a whisper object.
func (w *nimbusWhisperWrapper) SetTimeSource(timesource func() time.Time) {
	w.timesource = timesource
}

func (w *nimbusWhisperWrapper) SubscribeEnvelopeEvents(eventsProxy chan<- whispertypes.EnvelopeEvent) whispertypes.Subscription {
	// TODO: when mailserver support is implemented
	panic("not implemented")
}

// SelectedKeyPairID returns the id of currently selected key pair.
// It helps distinguish between different users w/o exposing the user identity itself.
func (w *nimbusWhisperWrapper) SelectedKeyPairID() string {
	return ""
}

func (w *nimbusWhisperWrapper) GetPrivateKey(id string) (*ecdsa.PrivateKey, error) {
	retVal := w.routineQueue.Send(func(c chan<- interface{}) {
		idC, err := decodeHexID(id)
		if err != nil {
			c <- err
			return
		}
		defer C.free(unsafe.Pointer(idC))
		privKeyC := C.malloc(whispertypes.AesKeyLength)
		defer C.free(unsafe.Pointer(privKeyC))

		if ok := C.nimbus_get_private_key(idC, (*C.uchar)(unsafe.Pointer(privKeyC))); !ok {
			c <- errors.New("failed to get private key from Nimbus")
			return
		}

		pk, err := crypto.ToECDSA(C.GoBytes(privKeyC, C.PRIVKEY_LEN))
		if err != nil {
			c <- err
			return
		}

		c <- pk
	})
	if err, ok := retVal.(error); ok {
		return nil, err
	}

	return retVal.(*ecdsa.PrivateKey), nil
}

// AddKeyPair imports a asymmetric private key and returns a deterministic identifier.
func (w *nimbusWhisperWrapper) AddKeyPair(key *ecdsa.PrivateKey) (string, error) {
	retVal := w.routineQueue.Send(func(c chan<- interface{}) {
		privKey := crypto.FromECDSA(key)
		privKeyC := C.CBytes(privKey)
		defer C.free(unsafe.Pointer(privKeyC))

		idC := C.malloc(C.size_t(C.ID_LEN))
		defer C.free(idC)
		if !C.nimbus_add_keypair((*C.uchar)(unsafe.Pointer(privKeyC)), (*C.uchar)(idC)) {
			c <- errors.New("failed to add keypair to Nimbus")
			return
		}

		c <- protocol.EncodeHex(C.GoBytes(idC, C.ID_LEN))
	})
	if err, ok := retVal.(error); ok {
		return "", err
	}

	return retVal.(string), nil
}

// DeleteKeyPair deletes the specified key if it exists.
func (w *nimbusWhisperWrapper) DeleteKeyPair(key string) bool {
	return w.routineQueue.Send(func(c chan<- interface{}) {
		keyC, err := decodeHexID(key)
		if err != nil {
			c <- err
			return
		}
		defer C.free(unsafe.Pointer(keyC))

		c <- C.nimbus_delete_keypair(keyC)
	}).(bool)
}

// SelectKeyPair adds cryptographic identity, and makes sure
// that it is the only private key known to the node.
func (w *nimbusWhisperWrapper) SelectKeyPair(key *ecdsa.PrivateKey) error {
	return errors.New("not implemented")
}

func (w *nimbusWhisperWrapper) AddSymKeyDirect(key []byte) (string, error) {
	retVal := w.routineQueue.Send(func(c chan<- interface{}) {
		keyC := C.CBytes(key)
		defer C.free(unsafe.Pointer(keyC))

		idC := C.malloc(C.size_t(C.ID_LEN))
		defer C.free(idC)
		if !C.nimbus_add_symkey((*C.uchar)(unsafe.Pointer(keyC)), (*C.uchar)(idC)) {
			c <- errors.New("failed to add symkey to Nimbus")
			return
		}

		c <- protocol.EncodeHex(C.GoBytes(idC, C.ID_LEN))
	})
	if err, ok := retVal.(error); ok {
		return "", err
	}

	return retVal.(string), nil
}

func (w *nimbusWhisperWrapper) AddSymKeyFromPassword(password string) (string, error) {
	retVal := w.routineQueue.Send(func(c chan<- interface{}) {
		passwordC := C.CString(password)
		defer C.free(unsafe.Pointer(passwordC))

		idC := C.malloc(C.size_t(C.ID_LEN))
		defer C.free(idC)
		if C.nimbus_add_symkey_from_password(passwordC, (*C.uchar)(idC)) {
			id := C.GoBytes(idC, C.ID_LEN)
			c <- protocol.EncodeHex(id)
		} else {
			c <- errors.New("failed to add symkey to Nimbus")
		}
	})
	if err, ok := retVal.(error); ok {
		return "", err
	}

	return retVal.(string), nil
}

func (w *nimbusWhisperWrapper) DeleteSymKey(id string) bool {
	return w.routineQueue.Send(func(c chan<- interface{}) {
		idC, err := decodeHexID(id)
		if err != nil {
			c <- err
			return
		}
		defer C.free(unsafe.Pointer(idC))

		c <- C.nimbus_delete_symkey(idC)
	}).(bool)
}

func (w *nimbusWhisperWrapper) GetSymKey(id string) ([]byte, error) {
	retVal := w.routineQueue.Send(func(c chan<- interface{}) {
		idC, err := decodeHexID(id)
		if err != nil {
			c <- err
			return
		}
		defer C.free(unsafe.Pointer(idC))

		// Allocate a buffer for Nimbus to return the symkey on
		dataC := C.malloc(C.size_t(C.SYMKEY_LEN))
		defer C.free(unsafe.Pointer(dataC))
		if ok := C.nimbus_get_symkey(idC, (*C.uchar)(unsafe.Pointer(dataC))); !ok {
			c <- errors.New("symkey not found")
			return
		}

		c <- C.GoBytes(dataC, C.SYMKEY_LEN)
	})
	if err, ok := retVal.(error); ok {
		return nil, err
	}

	return retVal.([]byte), nil
}

//export onMessageHandler
func onMessageHandler(msg *C.received_message, udata unsafe.Pointer) {
	messageList := (gopointer.Restore(udata)).(*list.List)

	topic := whispertypes.TopicType{}
	copy(topic[:], C.GoBytes(unsafe.Pointer(&msg.topic[0]), whispertypes.TopicLength)[:whispertypes.TopicLength])
	wrappedMsg := &whispertypes.Message{
		TTL:       uint32(msg.ttl),
		Timestamp: uint32(msg.timestamp),
		Topic:     topic,
		Payload:   C.GoBytes(unsafe.Pointer(msg.decoded), C.int(msg.decodedLen)),
		PoW:       float64(msg.pow),
		Hash:      C.GoBytes(unsafe.Pointer(&msg.hash[0]), protocol.HashLength),
		P2P:       true,
	}
	if msg.source != nil {
		wrappedMsg.Sig = append([]byte{0x04}, C.GoBytes(unsafe.Pointer(msg.source), whispertypes.PubKeyLength)...)
	}
	if msg.recipientPublicKey != nil {
		wrappedMsg.Dst = append([]byte{0x04}, C.GoBytes(unsafe.Pointer(msg.recipientPublicKey), whispertypes.PubKeyLength)...)
	}

	messageList.PushBack(wrappedMsg)
}

func (w *nimbusWhisperWrapper) Subscribe(opts *whispertypes.SubscriptionOptions) (string, error) {
	f, err := w.createFilterWrapper("", opts)
	if err != nil {
		return "", err
	}

	retVal := w.routineQueue.Send(func(c chan<- interface{}) {
		// Create a message store for this filter, so we can add new messages to it from the nimbus_subscribe_filter callback
		messageList := list.New()
		idC := C.malloc(C.size_t(C.ID_LEN))
		defer C.free(idC)
		if !C.nimbus_subscribe_filter(
			GetNimbusFilterFrom(f),
			(C.received_msg_handler)(unsafe.Pointer(C.onMessageHandler_cgo)), gopointer.Save(messageList),
			(*C.uchar)(idC)) {
			c <- errors.New("failed to subscribe to filter in Nimbus")
			return
		}
		filterID := C.GoString((*C.char)(idC))

		w.filterMessagesMu.Lock()
		w.filterMessages[filterID] = messageList // TODO: Check if this is done too late (race condition with onMessageHandler)
		w.filterMessagesMu.Unlock()

		f.(*nimbusFilterWrapper).id = filterID

		c <- filterID
	})
	if err, ok := retVal.(error); ok {
		return "", err
	}

	return retVal.(string), nil
}

func (w *nimbusWhisperWrapper) GetFilter(id string) whispertypes.Filter {
	idC := C.CString(id)
	defer C.free(unsafe.Pointer(idC))

	panic("GetFilter not implemented")
	// pFilter := C.nimbus_get_filter(idC)
	// return NewNimbusFilterWrapper(pFilter, id, false)
}

func (w *nimbusWhisperWrapper) Unsubscribe(id string) error {
	return w.routineQueue.Send(func(c chan<- interface{}) {
		idC, err := decodeHexID(id)
		if err != nil {
			c <- err
			return
		}
		defer C.free(unsafe.Pointer(idC))

		if ok := C.nimbus_unsubscribe_filter(idC); !ok {
			c <- errors.New("filter not found")
			return
		}

		w.filterMessagesMu.Lock()
		if messageList, ok := w.filterMessages[id]; ok {
			gopointer.Unref(gopointer.Save(messageList))
			delete(w.filterMessages, id)
		}
		w.filterMessagesMu.Unlock()

		if f, ok := w.filters[id]; ok {
			f.(*nimbusFilterWrapper).Free()
			delete(w.filters, id)
		}

		c <- nil
	}).(error)
}

func decodeHexID(id string) (*C.uint8_t, error) {
	idBytes, err := protocol.DecodeHex(id)
	if err == nil && len(idBytes) != C.ID_LEN {
		err = fmt.Errorf("ID length must be %v bytes, actual length is %v", C.ID_LEN, len(idBytes))
	}
	if err != nil {
		return nil, err
	}

	return (*C.uint8_t)(C.CBytes(idBytes)), nil
}

// copyTopicToCBuffer copies a Go topic buffer to a C topic buffer without allocating new memory
func copyTopicToCBuffer(dst *C.uchar, topic []byte) {
	if len(topic) != whispertypes.TopicLength {
		panic("invalid Whisper topic buffer size")
	}

	p := (*[whispertypes.TopicLength]C.uchar)(unsafe.Pointer(dst))
	for index, b := range topic {
		p[index] = C.uchar(b)
	}
}

func (w *nimbusWhisperWrapper) createFilterWrapper(id string, opts *whispertypes.SubscriptionOptions) (whispertypes.Filter, error) {
	if len(opts.Topics) != 1 {
		return nil, errors.New("currently only 1 topic is supported by the Nimbus bridge")
	}

	filter := C.filter_options{
		minPow:   C.double(opts.PoW),
		allowP2P: C.int(1),
	}
	copyTopicToCBuffer(&filter.topic[0], opts.Topics[0])
	if opts.PrivateKeyID != "" {
		if idC, err := decodeHexID(opts.PrivateKeyID); err == nil {
			filter.privateKeyID = idC
		} else {
			return nil, err
		}
	}
	if opts.SymKeyID != "" {
		if idC, err := decodeHexID(opts.SymKeyID); err == nil {
			filter.symKeyID = idC
		} else {
			return nil, err
		}
	}

	return NewNimbusFilterWrapper(&filter, id, true), nil
}

func (w *nimbusWhisperWrapper) SendMessagesRequest(peerID []byte, r whispertypes.MessagesRequest) error {
	return errors.New("not implemented")
}

// RequestHistoricMessages sends a message with p2pRequestCode to a specific peer,
// which is known to implement MailServer interface, and is supposed to process this
// request and respond with a number of peer-to-peer messages (possibly expired),
// which are not supposed to be forwarded any further.
// The whisper protocol is agnostic of the format and contents of envelope.
func (w *nimbusWhisperWrapper) RequestHistoricMessagesWithTimeout(peerID []byte, envelope whispertypes.Envelope, timeout time.Duration) error {
	return errors.New("not implemented")
}

// SyncMessages can be sent between two Mail Servers and syncs envelopes between them.
func (w *nimbusWhisperWrapper) SyncMessages(peerID []byte, req whispertypes.SyncMailRequest) error {
	return errors.New("not implemented")
}
