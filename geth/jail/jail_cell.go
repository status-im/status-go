package jail

import (
	"sync"

	"fknsrs.biz/p/ottoext/loop"
	"fknsrs.biz/p/ottoext/timers"
	"github.com/robertkrimen/otto"
)

const (
	// JailCellRequestTimeout seconds before jailed request times out.
	JailCellRequestTimeout = 60
)

// JailCell represents single jail cell, which is basically a JavaScript VM.
// TODO(influx6): Rename JailCell to Cell in next refactoring phase.
type JailCell struct {
	sync.Mutex

	id string
	vm *otto.Otto
}

// newJailCell encapsulates what we need to create a new jailCell from the
// provided vm and eventloop instance.
func newJailCell(id string, vm *otto.Otto) (*JailCell, error) {
	// create new event loop for the new cell.
	// this loop is handling 'setTimeout/setInterval'
	// calls and is running endlessly in a separate goroutine
	lo := loop.New(vm)

	// register handlers for setTimeout/setInterval
	// functions
	if err := timers.Define(vm, lo); err != nil {
		return nil, err
	}

	// finally, start loop in a goroutine
	// JailCell is currently immortal, so the loop
	go lo.Run()

	return &JailCell{
		id: id,
		vm: vm,
	}, nil
}

// Set sets the value to be keyed by the provided keyname.
func (cell *JailCell) Set(key string, val interface{}) error {
	cell.Lock()
	defer cell.Unlock()

	return cell.vm.Set(key, val)
}

// Get returns the giving key's otto.Value from the underline otto vm.
func (cell *JailCell) Get(key string) (otto.Value, error) {
	cell.Lock()
	defer cell.Unlock()

	return cell.vm.Get(key)
}

// Call attempts to call the internal call function for the giving response associated with the
// proper values.
func (cell *JailCell) Call(item string, this interface{}, args ...interface{}) (otto.Value, error) {
	cell.Lock()
	defer cell.Unlock()

	return cell.vm.Call(item, this, args...)
}

// Run evaluates the giving js string on the associated vm llop.
func (cell *JailCell) Run(val string) (otto.Value, error) {
	cell.Lock()
	defer cell.Unlock()

	return cell.vm.Run(val)
}
