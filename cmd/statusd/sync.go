package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/status-im/status-go/geth/common"
)

func createContextFromTimeout(timeout int) (context.Context, context.CancelFunc) {
	if timeout == 0 {
		return context.WithCancel(context.Background())
	}

	return context.WithTimeout(context.Background(), time.Duration(timeout)*time.Minute)
}

// syncAndStopNode tries to sync the blockchain and stop the node.
// It returns an exit code (`0` if successful or `1` in case of error)
// that can be used in `os.Exit` to exit immediately when the function returns.
// The special exit code `-1` is used if execution was interrupted.
func syncAndStopNode(interruptCh <-chan struct{}, nodeManager common.NodeManager, timeout int) (exitCode int) {
	log.Printf("syncAndStopNode: node will synchronize the chain and exit (timeout %d mins)", timeout)

	ctx, cancel := createContextFromTimeout(timeout)
	defer cancel()

	doneSync := make(chan struct{})
	errSync := make(chan error)
	go func() {
		if err := nodeManager.EnsureSync(ctx); err != nil {
			errSync <- err
		}
		close(doneSync)
	}()

	select {
	case err := <-errSync:
		fmt.Printf("syncAndStopNode: failed to sync the chain: %v", err)
		exitCode = 1
	case <-doneSync:
	case <-interruptCh:
		// cancel context and return immediately if interrupted
		// `-1` is used as a special exit code to denote interruption
		return -1
	}

	done, err := nodeManager.StopNode()
	if err != nil {
		log.Printf("syncAndStopNode: failed to stop the node: %v", err)
		return 1
	}
	<-done

	return
}
