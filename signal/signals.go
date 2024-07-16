package signal

/*
#include <stddef.h>
#include <stdbool.h>
#include <stdlib.h>
extern bool StatusServiceSignalEvent(const char *jsonEvent);
extern void SetEventCallback(void *cb);
*/
import "C"
import (
	"encoding/json"
	"unsafe"

	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

// MobileSignalHandler is a simple callback function that gets called when any signal is received
type MobileSignalHandler func([]byte)

// storing the current signal handler here
var mobileSignalHandler MobileSignalHandler

// SignalBlocklist is an optional set of signals that should be blocklisted,
// i.e. not sent.
type SignalBlocklist map[string]struct{}

var signalBlocklist SignalBlocklist

// All general log messages in this package should be routed through this logger.
var logger = log.New("package", "status-go/signal")

// Envelope is a general signal sent upward from node to RN app
type Envelope struct {
	Type  string      `json:"type"`
	Event interface{} `json:"event"`
}

// NewEnvelope creates new envlope of given type and event payload.
func NewEnvelope(typ string, event interface{}) *Envelope {
	return &Envelope{
		Type:  typ,
		Event: event,
	}
}

// isSignalBlocklisted returns true when a signal type is present in the
// blocklist, or when not found, if the signal type and event type are present.
//
// The convention is that clients can specify a particular event type from any
// signal by concatenating a forward slash.
//
// Example for signal "wallet" and event type
// "wallet-collectible-status-changed":
//
//	"wallet/wallet-collectible-status-changed"
func isSignalBlocklisted(signal *Envelope, event any) bool {
	if len(signalBlocklist) > 0 {
		// If the signal type is in the blocklist, then "send" is a nop.
		if _, ok := signalBlocklist[signal.Type]; ok {
			logger.Info("imotta - Signal blocklisted", "signal", signal.Type)
			return true
		}

		// A signal may encompass an event, in which case we need to concretely
		// check the type of the event.
		//
		// THIS DOES NOT WORK because it creates a circular dependency and the
		// compiler will complaint.
		if e, ok := event.(walletevent.Event); ok {
			name := signal.Type + "/" + string(e.Type)
			logger.Info("imotta - Signal blocklisted", "signal", name)
			if _, ok := signalBlocklist[name]; ok {
				return true
			}
		}
	}

	return false
}

// send sends application signal (in JSON) upwards to application (via default notification handler)
func send(typ string, event interface{}) {
	signal := NewEnvelope(typ, event)
	data, err := json.Marshal(&signal)
	if err != nil {
		logger.Error("Marshalling signal envelope", "error", err)
		return
	}

	if isSignalBlocklisted(signal, event) {
		return
	}

	// If a Go implementation of signal handler is set, let's use it.
	if mobileSignalHandler != nil {
		mobileSignalHandler(data)
	} else {
		// ...and fallback to C implementation otherwise.
		str := C.CString(string(data))
		C.StatusServiceSignalEvent(str)
		C.free(unsafe.Pointer(str))
	}
}

// NodeNotificationHandler defines a handler able to process incoming node events.
// Events are encoded as JSON strings.
type NodeNotificationHandler func(jsonEvent string)

var notificationHandler NodeNotificationHandler = TriggerDefaultNodeNotificationHandler

// notificationHandlerMutex guards notificationHandler for concurrent calls
var notificationHandlerMutex sync.RWMutex

// SetDefaultNodeNotificationHandler sets notification handler to invoke on Send
func SetDefaultNodeNotificationHandler(fn NodeNotificationHandler) {
	notificationHandlerMutex.Lock()
	notificationHandler = fn
	notificationHandlerMutex.Unlock()
}

// ResetDefaultNodeNotificationHandler sets notification handler to default one
func ResetDefaultNodeNotificationHandler() {
	notificationHandlerMutex.Lock()
	notificationHandler = TriggerDefaultNodeNotificationHandler
	notificationHandlerMutex.Unlock()
}

// TriggerDefaultNodeNotificationHandler triggers default notification handler (helpful in tests)
func TriggerDefaultNodeNotificationHandler(jsonEvent string) {
	logger.Trace("Notification received", "event", jsonEvent)
}

// nolint: golint
//
//export NotifyNode
func NotifyNode(jsonEvent *C.char) {
	notificationHandlerMutex.RLock()
	defer notificationHandlerMutex.RUnlock()
	notificationHandler(C.GoString(jsonEvent))
}

// nolint: golint
//
//export TriggerTestSignal
func TriggerTestSignal() {
	str := C.CString(`{"answer": 42}`)
	C.StatusServiceSignalEvent(str)
	C.free(unsafe.Pointer(str))
}

// SetMobileSignalHandler sets new handler for geth events
// this function uses pure go implementation
func SetMobileSignalHandler(handler MobileSignalHandler) {
	mobileSignalHandler = handler
}

func SetSignalBlocklist(blocklist SignalBlocklist) {
	signalBlocklist = blocklist
}

// SetSignalEventCallback set callback
// this function uses C implementation (see `signals.c` file)
func SetSignalEventCallback(cb unsafe.Pointer) {
	C.SetEventCallback(cb)
}
