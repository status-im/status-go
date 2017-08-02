package jail

import (
	"sync"

	"fknsrs.biz/p/ottoext/fetch"
	"fknsrs.biz/p/ottoext/loop"
	"fknsrs.biz/p/ottoext/timers"
	"github.com/robertkrimen/otto"
)

const (
	// JCellRequestTimeout seconds before jailed request times out.
	JCellRequestTimeout = 60
)

// JCell represents single jail cell, which is basically a JavaScript VM.
type JCell struct {
	sync.Mutex

	id string
	vm *otto.Otto
	lo *loop.Loop
}

// newJCell encapsulates what we need to create a new jailCell from the
// provided vm and eventloop instance.
func newJCell(id string, vm *otto.Otto, lo *loop.Loop) (*JCell, error) {
	// Register fetch provider from ottoext.
	if err := fetch.Define(vm, lo); err != nil {
		return nil, err
	}

	// Register event loop for timers.
	if err := timers.Define(vm, lo); err != nil {
		return nil, err
	}

	return &JCell{
		id: id,
		vm: vm,
		lo: lo,
	}, nil
}

// Fetch attempts to call the underline Fetch API added through the
// ottoext package.
func (cell *JCell) Fetch(url string, callback func(otto.Value)) (otto.Value, error) {
	cell.Lock()
	if err := cell.vm.Set("__captureFetch", callback); err != nil {
		cell.Unlock()

		return otto.UndefinedValue(), err
	}

	cell.Unlock()

	return cell.Run(`fetch("` + url + `").then(function(response){
			__captureFetch({
				"url": response.url,
				"type": response.type,
				"body": response.text(),
				"status": response.status,
				"headers": response.headers,
			});
		});
	`)
}

// Set sets the value to be keyed by the provided keyname.
func (cell *JCell) Set(key string, val interface{}) error {
	cell.Lock()
	defer cell.Unlock()

	return cell.vm.Set(key, val)
}

// Get returns the giving key's otto.Value from the underline otto vm.
func (cell *JCell) Get(key string) (otto.Value, error) {
	cell.Lock()
	defer cell.Unlock()

	return cell.vm.Get(key)
}

// RunOnLoop evaluates the giving js string on the associated vm loop returning
// an error.
func (cell *JCell) RunOnLoop(val string) (otto.Value, error) {
	cell.Lock()
	defer cell.Unlock()

	res, err := cell.vm.Run(val)
	if err != nil {
		return res, err
	}

	return res, cell.lo.Run()
}

// CallOnLoop attempts to call the internal call function for the giving response associated with the
// proper values.
func (cell *JCell) CallOnLoop(item string, this interface{}, args ...interface{}) (otto.Value, error) {
	cell.Lock()
	defer cell.Unlock()

	res, err := cell.vm.Call(item, this, args...)
	if err != nil {
		return res, err
	}

	return res, cell.lo.Run()
}

// Call attempts to call the internal call function for the giving response associated with the
// proper values.
func (cell *JCell) Call(item string, this interface{}, args ...interface{}) (otto.Value, error) {
	cell.Lock()
	defer cell.Unlock()

	return cell.vm.Call(item, this, args...)
}

// Run evaluates the giving js string on the associated vm llop.
func (cell *JCell) Run(val string) (otto.Value, error) {
	cell.Lock()
	defer cell.Unlock()

	return cell.vm.Run(val)
}
