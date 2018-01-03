package node

import (
	"fmt"

	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/signal"
)

// HaltOnPanic recovers from panic, logs issue, sends upward notification, and exits
func HaltOnPanic() {
	if r := recover(); r != nil {
		err := fmt.Errorf("%v: %v", ErrNodeRunFailure, r)

		// send signal up to native app
		signal.Send(signal.Envelope{
			Type: signal.EventNodeCrashed,
			Event: signal.NodeCrashEvent{
				Error: err.Error(),
			},
		})

		common.Fatalf(err) // os.exit(1) is called internally
	}
}
