package jail

import (
	"context"
	"errors"
	"time"

	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/jail/internal/fetch"
	"github.com/status-im/status-go/jail/internal/loop"
	"github.com/status-im/status-go/jail/internal/loop/looptask"
	"github.com/status-im/status-go/jail/internal/timers"
	"github.com/status-im/status-go/jail/internal/vm"
)

const timeout = 5 * time.Second

// Manager defines methods for managing jailed environments
type Manager interface {
	// Call executes given JavaScript function w/i a jail cell context identified by the chatID.
	Call(chatID, this, args string) string

	// CreateCell creates a new jail cell.
	CreateCell(chatID string) (JSCell, error)

	// Parse creates a new jail cell context, with the given chatID as identifier.
	// New context executes provided JavaScript code, right after the initialization.
	// DEPRECATED in favour of CreateAndInitCell.
	Parse(chatID, js string) string

	// CreateAndInitCell creates a new jail cell and initialize it
	// with web3 and other handlers.
	CreateAndInitCell(chatID string, code ...string) string

	// Cell returns an existing instance of JSCell.
	Cell(chatID string) (JSCell, error)

	// Execute allows to run arbitrary JS code within a cell.
	Execute(chatID, code string) string

	// SetBaseJS allows to setup initial JavaScript to be loaded on each jail.CreateAndInitCell().
	SetBaseJS(js string)

	// Stop stops all background activity of jail
	Stop()
}

// JSValue is a wrapper around an otto.Value.
type JSValue struct {
	value otto.Value
}

// Value returns the underlying otto.Value from a JSValue. This value IS NOT THREADSAFE.
func (v *JSValue) Value() otto.Value {
	return v.value
}

// JSCell represents single jail cell, which is basically a JavaScript VM.
// It's designed to be a transparent wrapper around otto.VM's methods.
type JSCell interface {
	// Set a value inside VM.
	Set(string, interface{}) error
	// Get a value from VM.
	Get(string) (JSValue, error)
	// Run an arbitrary JS code. Input maybe string or otto.Script.
	Run(interface{}) (JSValue, error)
	// Call an arbitrary JS function by name and args.
	Call(item string, this interface{}, args ...interface{}) (JSValue, error)
	// Stop stops background execution of cell.
	Stop() error
}

// Cell represents a single jail cell, which is basically a JavaScript VM.
type Cell struct {
	jsvm   *vm.VM
	id     string
	cancel context.CancelFunc

	loop        *loop.Loop
	loopStopped chan struct{}
	loopErr     error
}

// NewCell encapsulates what we need to create a new jailCell from the
// provided vm and eventloop instance.
func NewCell(id string) (*Cell, error) {
	newVM := vm.New()
	lo := loop.New(newVM)

	err := registerVMHandlers(newVM, lo)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	loopStopped := make(chan struct{})
	cell := Cell{
		jsvm:        newVM,
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
	errChan := make(chan error)

	go func() {
		defer close(errChan)
		err := c.loop.AddAndExecute(task)
		if err != nil {
			errChan <- err
		}
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case err := <-errChan:
		return err

	case <-timer.C:
		return errors.New("Timeout")
	}

}

// Set calls Set on the underlying JavaScript VM.
func (c *Cell) Set(key string, val interface{}) error {
	return c.jsvm.Set(key, val)
}

// Get calls Get on the underlying JavaScript VM and returns
// a wrapper around the otto.Value.
func (c *Cell) Get(key string) (JSValue, error) {
	v, err := c.jsvm.Get(key)
	if err != nil {
		return JSValue{}, err
	}
	value := JSValue{value: v}
	return value, nil
}

// GetObjectValue calls GetObjectValue on the underlying JavaScript VM and returns
// a wrapper around the otto.Value.
func (c *Cell) GetObjectValue(v otto.Value, name string) (JSValue, error) {
	v, err := c.jsvm.GetObjectValue(v, name)
	if err != nil {
		return JSValue{}, err
	}
	value := JSValue{value: v}
	return value, nil
}

// Run calls Run on the underlying JavaScript VM and returns
// a wrapper around the otto.Value.
func (c *Cell) Run(src interface{}) (JSValue, error) {
	v, err := c.jsvm.Run(src)
	if err != nil {
		return JSValue{}, err
	}
	value := JSValue{value: v}
	return value, nil
}

// Call calls Call on the underlying JavaScript VM and returns
// a wrapper around the otto.Value.
func (c *Cell) Call(item string, this interface{}, args ...interface{}) (JSValue, error) {
	v, err := c.jsvm.Call(item, this, args...)
	if err != nil {
		return JSValue{}, err
	}
	value := JSValue{value: v}
	return value, nil
}
