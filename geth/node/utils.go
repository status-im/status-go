package node

import (
	"fmt"

	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/signal"
)

// HaltOnPanic recovers from panic, logs issue, sends upward notification, and exits
func HaltOnPanic() {
	if r := recover(); r != nil {
		strErr := fmt.Sprintf("%v: %v", ErrNodeRunFailure, r)

		// send signal up to native app
		signal.Send(signal.Envelope{
			Type: signal.EventNodeCrashed,
			Event: signal.NodeCrashEvent{
				Error: strErr,
			},
		})

		common.Fatalf(ErrNodeRunFailure, r) // os.exit(1) is called internally
	}
}
