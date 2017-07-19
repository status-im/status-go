package jail

import (
	"time"

	"fknsrs.biz/p/ottoext/fetch"
	"fknsrs.biz/p/ottoext/loop"
	"fknsrs.biz/p/ottoext/timers"
	"github.com/eapache/go-resiliency/semaphore"
	"github.com/robertkrimen/otto"

	"github.com/status-im/status-go/geth/common"
)

const (
	// JailCellRequestTimeout seconds before jailed request times out.
	JailCellRequestTimeout = 60
)

// JailCell represents single jail cell, which is basically a JavaScript VM.
type JailCell struct {
	// FIXME(tiabc): It's never called. Double-check and delete.
	//sync.Mutex

	id string
	vm *otto.Otto
	lo *loop.Loop

	// FIXME(tiabc): It's never used. Is it a mistake?
	sem *semaphore.Semaphore
}

// newJailCell encapsulates what we need to create a new jailCell from the
// provided vm and eventloop instance.
func newJailCell(id string, vm *otto.Otto, lo *loop.Loop) (*JailCell, error) {
	// Register fetch provider from ottoext.
	if err := fetch.Define(vm, lo); err != nil {
		return nil, err
	}

	// Register event loop for timers.
	if err := timers.Define(vm, lo); err != nil {
		return nil, err
	}

	return &JailCell{
		id:  id,
		vm:  vm,
		lo:  lo,
		sem: semaphore.New(1, JailCellRequestTimeout*time.Second),
	}, nil
}

// Copy returns a new JailCell instance with a new eventloop runtime associated with
// the given cell.
func (cell *JailCell) Copy() (common.JailCell, error) {
	vmCopy := cell.vm.Copy()
	return newJailCell(cell.id, vmCopy, loop.New(vmCopy))
}

// Fetch attempts to call the underline Fetch API added through the
// ottoext package.
func (cell *JailCell) Fetch(url string, callback func(otto.Value)) (otto.Value, error) {
	if err := cell.vm.Set("__captureFetch", callback); err != nil {
		return otto.UndefinedValue(), err
	}

	return cell.Exec(`fetch("` + url + `").then(function(response){
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

// Exec evaluates the giving js string on the associated vm loop returning
// an error.
func (cell *JailCell) Exec(val string) (otto.Value, error) {
	res, err := cell.vm.Run(val)
	if err != nil {
		return res, err
	}

	return res, cell.lo.Run()
}

// Run evaluates the giving js string on the associated vm llop.
func (cell *JailCell) Run(val string) (otto.Value, error) {
	return cell.vm.Run(val)
}

// CellLoop returns the ottoext.Loop instance which provides underline timeout/setInternval
// event runtime for the Jail vm.
func (cell *JailCell) CellLoop() *loop.Loop {
	return cell.lo
}

// Executor returns a structure which implements the common.JailExecutor.
func (cell *JailCell) Executor() common.JailExecutor {
	return cell
}

// CellVM returns the associated otto.Vm connect to the giving cell.
func (cell *JailCell) CellVM() *otto.Otto {
	return cell.vm
}
