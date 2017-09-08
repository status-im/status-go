package jail

import (
	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/jail/internal/fetch"
	"github.com/status-im/status-go/geth/jail/internal/loop"
	"github.com/status-im/status-go/geth/jail/internal/timers"
	"github.com/status-im/status-go/geth/jail/internal/vm"
)

// Cell represents a single jail cell, which is basically a JavaScript VM.
type Cell struct {
	id string
	*vm.VM
}

// newCell encapsulates what we need to create a new jailCell from the
// provided vm and eventloop instance.
func newCell(id string, ottoVM *otto.Otto) (*Cell, error) {
	cellVM := vm.New(ottoVM)

	lo := loop.New(cellVM)

	registerVMHandlers(cellVM, lo)

	// start loop in a goroutine
	// Cell is currently immortal, so the loop
	go lo.Run()

	return &Cell{
		id: id,
		VM: cellVM,
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
