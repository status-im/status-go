package jail

import (
	"context"

	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/jail/internal/fetch"
	"github.com/status-im/status-go/geth/jail/internal/loop"
	"github.com/status-im/status-go/geth/jail/internal/loop/looptask"
	"github.com/status-im/status-go/geth/jail/internal/timers"
	"github.com/status-im/status-go/geth/jail/internal/vm"
)

// Cell represents a single jail cell, which is basically a JavaScript VM.
type Cell struct {
	*vm.VM
	id     string
	cancel context.CancelFunc
	lo     *loop.Loop
}

// newCell encapsulates what we need to create a new jailCell from the
// provided vm and eventloop instance.
func newCell(id string, ottoVM *otto.Otto) (*Cell, error) {
	cellVM := vm.New(ottoVM)

	lo := loop.New(cellVM)

	registerVMHandlers(cellVM, lo)

	ctx, cancel := context.WithCancel(context.Background())

	// start event loop in background
	go lo.Run(ctx)

	return &Cell{
		VM:     cellVM,
		id:     id,
		cancel: cancel,
		lo:     lo,
	}, nil
}

// registerHandlers register variuous functions and handlers
// to the Otto VM, such as Fetch API callbacks or promises.
func registerVMHandlers(v *vm.VM, lo *loop.Loop) error {
	// setTimeout/setInterval functions
	if err := timers.Define(v, lo); err != nil {
		return err
	}

	// FetchAPI functions
	if err := fetch.Define(v, lo); err != nil {
		return err
	}

	return nil
}

// Stop halts event loop associated with cell.
func (c *Cell) Stop() {
	c.cancel()
}

// CallAsync puts otto's function with given args into
// event queue loop and schedules for immediate execution.
// Intended to be used by any cell user that want's to run
// async call, like callback.
func (c *Cell) CallAsync(fn otto.Value, args ...interface{}) {
	task := looptask.NewCallTask(fn, args...)
	c.lo.Add(task)
	// TODO(divan): review API of `loop` package, it's contrintuitive
	go c.lo.Ready(task)
}
