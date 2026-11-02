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
	"sync/atomic"
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

// All general log messages in this package should be routed through this logger.
var logger = logutils.ZapLogger().Named("signal")

// epoch identifies the backend session that produced a signal. It starts at 1 so
// a client can treat a missing or zero epoch as invalid, and only advances when a
// client explicitly logs out (see AdvanceEpoch).
var epoch atomic.Uint64

func init() {
	epoch.Store(1)
}

// Epoch returns the epoch stamped on signals sent from now on.
func Epoch() uint64 {
	return epoch.Load()
}

// AdvanceEpoch starts a new signal epoch and returns it. Clients that log out and
// back in within the same process use the epoch to reject signals the previous
// session queued before the logout but which are delivered after it.
func AdvanceEpoch() uint64 {
	return epoch.Add(1)
}

// Envelope is a general signal sent upward from node to RN app
type Envelope struct {
	Type      string      `json:"type"`
	Epoch     uint64      `json:"epoch"`
	Event     interface{} `json:"event"`
	Timestamp int64       `json:"timestamp"`
}

// newEnvelope creates new envelope of given type and event payload.
func newEnvelope(typ string, event interface{}) *Envelope {
	return &Envelope{
		Type:      typ,
		Epoch:     Epoch(),
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
	if signalHandler != nil {
		signalHandler(data)
	} else {
		// ...and fallback to C implementation otherwise.
		str := C.CString(string(data))
		C.SignalEvent(str)
		C.free(unsafe.Pointer(str))
	}
}

// SetHandler sets new handler for events.
func SetHandler(handler Handler) {
	signalHandler = handler
}

func ResetHandler() {
	signalHandler = nil
}

// SetSignalEventCallback sets a C callback provided by client,
// see `signals.c` file.
func SetSignalEventCallback(cb unsafe.Pointer) {
	C.SetEventCallback(cb)
}
