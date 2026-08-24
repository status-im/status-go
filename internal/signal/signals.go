package signal

/*
#include <stddef.h>
#include <stdbool.h>
#include <stdlib.h>
extern void SignalEvent(const char *jsonEvent);
extern void SetEventCallback(void *cb);
*/
import "C"
import (
	"encoding/json"
	"sync"
	"time"
	"unsafe"

	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/internal/logutils/callog"
	"github.com/status-im/status-go/internal/logutils/requestlog"
)

// Handler is a simple callback function that gets called when any signal is emitted.
type Handler func([]byte)

// storing the current signal handler here
var signalHandler Handler

// signalHandlerMutex guards the process-wide Go signal handler.
var signalHandlerMutex sync.RWMutex

// All general log messages in this package should be routed through this logger.
var logger = logutils.ZapLogger().Named("signal")

// Envelope is a general signal sent upward from node to RN app
type Envelope struct {
	Type      string      `json:"type"`
	Event     interface{} `json:"event"`
	Timestamp int64       `json:"timestamp"`
}

// newEnvelope creates new envelope of given type and event payload.
func newEnvelope(typ string, event interface{}) *Envelope {
	return &Envelope{
		Type:      typ,
		Event:     event,
		Timestamp: time.Now().Unix(),
	}
}

// send sends application signal (in JSON) upwards to application via go or C callback.
func send(typ string, event interface{}) {
	signal := newEnvelope(typ, event)
	data, err := json.Marshal(&signal)
	if err != nil {
		logger.Error("Marshalling signal envelope", zap.Error(err))
		return
	}
	callog.LogSignal(requestlog.GetRequestLogger(), typ, event)

	// If a Go implementation of signal handler is set, let's use it.
	signalHandlerMutex.RLock()
	handler := signalHandler
	signalHandlerMutex.RUnlock()
	if handler != nil {
		handler(data)
	} else {
		// ...and fallback to C implementation otherwise.
		str := C.CString(string(data))
		C.SignalEvent(str)
		C.free(unsafe.Pointer(str))
	}
}

// SetHandler sets new handler for events.
func SetHandler(handler Handler) {
	signalHandlerMutex.Lock()
	defer signalHandlerMutex.Unlock()
	signalHandler = handler
}

func ResetHandler() {
	signalHandlerMutex.Lock()
	defer signalHandlerMutex.Unlock()
	signalHandler = nil
}

// SetSignalEventCallback sets a C callback provided by client,
// see `signals.c` file.
func SetSignalEventCallback(cb unsafe.Pointer) {
	C.SetEventCallback(cb)
}
