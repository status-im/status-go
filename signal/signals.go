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

// All general log messages in this package should be routed through this logger.
var logger = log.New("package", "status-go/signal")

// Envelope is a general signal sent upward from node to RN app
type Envelope struct {
	Type  string      `json:"type"`
	Event interface{} `json:"event"`
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

// TODO: remove this after refactoring
func Send(e Envelope) { sendSignal(e) }

// sendSignal sends application signal (JSON, normally) upwards to application (via default notification handler)
func sendSignal(signal Envelope) {
	data, err := json.Marshal(&signal)
	if err != nil {
		logger.Error("Marshalling signal envelope", "error", err)
	}
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
