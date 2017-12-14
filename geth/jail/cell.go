package jail

import (
	"context"
	"errors"
	"time"

	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/jail/internal/fetch"
	"github.com/status-im/status-go/geth/jail/internal/loop"
	"github.com/status-im/status-go/geth/jail/internal/loop/looptask"
	"github.com/status-im/status-go/geth/jail/internal/timers"
	"github.com/status-im/status-go/geth/jail/internal/vm"
)

const timeout = 5 * time.Second

// Cell represents a single jail cell, which is basically a JavaScript VM.
type Cell struct {
	*vm.VM
	id     string
	cancel context.CancelFunc

	loop        *loop.Loop
	loopStopped chan struct{}
	loopErr     error
}

// NewCell encapsulates what we need to create a new jailCell from the
// provided vm and eventloop instance.
func NewCell(id string) (*Cell, error) {
	vm := vm.New()
	lo := loop.New(vm)

	err := registerVMHandlers(vm, lo)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	loopStopped := make(chan struct{})
	cell := Cell{
		VM:          vm,
		id:          id,
		cancel:      cancel,
		loop:        lo,
		loopStopped: loopStopped,
	}

	// Start event loop in the background.
	go func() {
		err := lo.Run(ctx)
		if err != context.Canceled {
			cell.loopErr = err
		}

		close(loopStopped)
	}()

	return &cell, nil
}

// registerHandlers register variuous functions and handlers
// to the Otto VM, such as Fetch API callbacks or promises.
func registerVMHandlers(vm *vm.VM, lo *loop.Loop) error {
	// setTimeout/setInterval functions
	if err := timers.Define(vm, lo); err != nil {
		return err
	}

	// FetchAPI functions
	return fetch.Define(vm, lo)
}

// Stop halts event loop associated with cell.
func (c *Cell) Stop() error {
	c.cancel()

	select {
	case <-c.loopStopped:
		return c.loopErr
	case <-time.After(time.Second):
		return errors.New("stopping the cell timed out")
	}
}

// CallAsync puts otto's function with given args into
// event queue loop and schedules for immediate execution.
// Intended to be used by any cell user that want's to run
// async call, like callback.
func (c *Cell) CallAsync(fn otto.Value, args ...interface{}) error {
	task := looptask.NewCallTask(fn, args...)
	errChan := make(chan error, 1)
	defer close(errChan)

	go func() {
		err := c.loop.Add(task)
		if err != nil {
			errChan <- err
			return
		}

		err = c.loop.Ready(task)
		if err != nil {
			errChan <- err
		}
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case err, ok := <-errChan:
			if err != nil {
				return err
			}
			if !ok {
				return nil
			}
		case <-timer.C:
			return errors.New("Timeout")
		}
	}
}
