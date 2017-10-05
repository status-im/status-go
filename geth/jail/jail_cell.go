package jail

import (
	"time"

	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/jail/internal/fetch"
	"github.com/status-im/status-go/geth/jail/internal/loop"
	"github.com/status-im/status-go/geth/jail/internal/timers"
	"github.com/status-im/status-go/geth/jail/internal/vm"
	"github.com/status-im/status-go/geth/log"
)

const (
	// timeout for stopping loop in milliseconds
	loopStopTimeout = 5000
)

// Cell represents a single jail cell, which is basically a JavaScript VM.
type Cell struct {
	id          string
	stopChan    chan struct{}
	stoppedChan chan struct{}
	stopped     bool
	*vm.VM
}

// Stop sends command to stop the cell and
// returns channel which will closed at finish
func (c *Cell) Stop() chan struct{} {
	//protection against double closing the channel
	c.Lock()
	defer c.Unlock()

	if !c.stopped {
		c.stopped = true
		close(c.stopChan)
	}

	return c.stoppedChan
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

	cell := &Cell{
		id:          id,
		VM:          cellVM,
		stopChan:    make(chan struct{}),
		stoppedChan: make(chan struct{}),
	}

	go cell.stopChanListener(lo)

	return cell, nil
}

// stopChanListener waits for message from stopChan
// then init stop of loop with the timeout
func (c *Cell) stopChanListener(lo *loop.Loop) {
	defer close(c.stoppedChan)

	<-c.stopChan

	select {
	case <-lo.Stop():
		return
	case <-time.After(loopStopTimeout * time.Millisecond):
		log.Warn("Loop stopping timeout")
	}
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
