package signal

/*
#include <stddef.h>
#include <stdbool.h>
extern bool StatusServiceSignalEvent(const char *jsonEvent);
*/
import "C"
import (
	"encoding/json"

	"sync"

	"github.com/ethereum/go-ethereum/log"
)

const (
	// EventNodeStarted is triggered when underlying node is started
	EventNodeStarted = "node.started"

	// EventNodeReady is triggered when underlying node is fully ready
	// (consider backend to be fully registered)
	EventNodeReady = "node.ready"

	// EventNodeStopped is triggered when underlying node is fully stopped
	EventNodeStopped = "node.stopped"

	// EventNodeCrashed is triggered when node crashes
	EventNodeCrashed = "node.crashed"

	// EventChainDataRemoved is triggered when node's chain data is removed
	EventChainDataRemoved = "chaindata.removed"
)

// Envelope is a general signal sent upward from node to RN app
type Envelope struct {
	Type  string      `json:"type"`
	Event interface{} `json:"event"`
}

// NodeCrashEvent is special kind of error, used to report node crashes
type NodeCrashEvent struct {
	Error error `json:"error"`
}

// All general log messages in this package should be routed through this logger.
var logger = log.New("package", "status-go/geth/signal")

// MarshalJSON implements the json.Marshaller interface.
func (e NodeCrashEvent) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Error string `json:"error"`
	}{
		Error: e.Error.Error(),
	})
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
	logger.Info("Notification received", "event", jsonEvent)
}

// Send sends application signal (JSON, normally) upwards to application (via default notification handler)
func Send(signal Envelope) {
	data, _ := json.Marshal(&signal)
	C.StatusServiceSignalEvent(C.CString(string(data)))
}

//export NotifyNode
//nolint: golint
func NotifyNode(jsonEvent *C.char) {
	notificationHandlerMutex.RLock()
	defer notificationHandlerMutex.RUnlock()
	notificationHandler(C.GoString(jsonEvent))
}

//export TriggerTestSignal
//nolint: golint
func TriggerTestSignal() {
	C.StatusServiceSignalEvent(C.CString(`{"answer": 42}`))
}
