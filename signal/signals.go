// +build !library

package signal

import (
	"encoding/json"
	"unsafe"

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

// NewEnvelope creates new envlope of given type and event payload.
func NewEnvelope(typ string, event interface{}) *Envelope {
	return &Envelope{
		Type:  typ,
		Event: event,
	}
}

// send sends application signal (in JSON) upwards to application (via default notification handler)
func send(typ string, event interface{}) {
	signal := NewEnvelope(typ, event)
	data, err := json.Marshal(&signal)
	if err != nil {
		logger.Error("Marshalling signal envelope", "error", err)
		return
	}

	notificationHandlerMutex.RLock()
	notificationHandler(string(data))
	notificationHandlerMutex.RUnlock()
}

// NodeNotificationHandler defines a handler able to process incoming node events.
// Events are encoded as JSON strings.
type NodeNotificationHandler func(jsonEvent string)

var notificationHandler NodeNotificationHandler = TriggerDefaultNodeNotificationHandler

// notificationHandlerMutex guards notificationHandler for concurrent calls
var notificationHandlerMutex sync.RWMutex

// SetDefaultNodeNotificationHandler sets notification handler to invoke on Send
func SetDefaultNodeNotificationHandler(fn NodeNotificationHandler) {
	logger.Warn("[DEBUG] Overriding notification handler")
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

//nolint: golint
func TriggerTestSignal() {
	str := `{"answer": 42}`
	notificationHandlerMutex.RLock()
	notificationHandler(str)
	notificationHandlerMutex.RUnlock()
}

//nolint: golint
func SetSignalEventCallback(cb unsafe.Pointer) {}
