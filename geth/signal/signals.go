package signal

/*
#include <stddef.h>
#include <stdbool.h>
extern bool StatusServiceSignalEvent(const char *jsonEvent);
*/
import "C"
import (
	"encoding/json"

	"github.com/status-im/status-go/geth/log"
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
	Error string `json:"error"`
}

// NodeNotificationHandler defines a handler able to process incoming node events.
// Events are encoded as JSON strings.
type NodeNotificationHandler func(jsonEvent string)

var notificationHandler NodeNotificationHandler = TriggerDefaultNodeNotificationHandler

// SetDefaultNodeNotificationHandler sets notification handler to invoke on Send
func SetDefaultNodeNotificationHandler(fn NodeNotificationHandler) {
	notificationHandler = fn
}

// ResetDefaultNodeNotificationHandler sets notification handler to default one
func ResetDefaultNodeNotificationHandler() {
	notificationHandler = TriggerDefaultNodeNotificationHandler
}

// TriggerDefaultNodeNotificationHandler triggers default notification handler (helpful in tests)
func TriggerDefaultNodeNotificationHandler(jsonEvent string) {
	log.Send(log.Info("Notification received").With("event", jsonEvent))
}

// Send sends application signal (JSON, normally) upwards to application (via default notification handler)
func Send(signal Envelope) {
	data, _ := json.Marshal(&signal)
	C.StatusServiceSignalEvent(C.CString(string(data)))
}

//export NotifyNode
func NotifyNode(jsonEvent *C.char) { // nolint: golint
	notificationHandler(C.GoString(jsonEvent))
}

//export TriggerTestSignal
func TriggerTestSignal() { // nolint: golint
	C.StatusServiceSignalEvent(C.CString(`{"answer": 42}`))
}
