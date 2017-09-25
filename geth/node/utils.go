package node

import (
	"fmt"
	"os"
	osSignal "os/signal"

	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/log"
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

// HaltOnInterruptSignal stops node and panics if you press Ctrl-C enough times
func HaltOnInterruptSignal(nodeManager *NodeManager) {
	sigc := make(chan os.Signal, 1)
	osSignal.Notify(sigc, os.Interrupt)
	defer osSignal.Stop(sigc)
	<-sigc
	if nodeManager.node == nil {
		return
	}
	log.Info("Got interrupt, shutting down...")
	go nodeManager.node.Stop() // nolint: errcheck
	for i := 3; i > 0; i-- {
		<-sigc
		if i > 1 {
			log.Info(fmt.Sprintf("Already shutting down, interrupt %d more times for panic.", i-1))
		}
	}
	panic("interrupted!")
}
