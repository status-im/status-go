package vm

import (
	"sync"

	"github.com/robertkrimen/otto"
)

// VM implements concurrency safe wrapper to
// otto's VM object.
type VM struct {
	sync.Mutex

	vm *otto.Otto
}

// New creates new instance of VM.
func New(vm *otto.Otto) *VM {
	return &VM{
		vm: vm,
	}
}

// Set sets the value to be keyed by the provided keyname.
func (vm *VM) Set(key string, val interface{}) error {
	vm.Lock()
	defer vm.Unlock()

	return vm.vm.Set(key, val)
}

// Get returns the giving key's otto.Value from the underline otto vm.
func (vm *VM) Get(key string) (otto.Value, error) {
	vm.Lock()
	defer vm.Unlock()

	return vm.vm.Get(key)
}

// Call attempts to call the internal call function for the giving response associated with the
// proper values.
func (vm *VM) Call(item string, this interface{}, args ...interface{}) (otto.Value, error) {
	vm.Lock()
	defer vm.Unlock()

	return vm.vm.Call(item, this, args...)
}

// Run evaluates JS source, which may be string or otto.Script variable.
func (vm *VM) Run(src interface{}) (otto.Value, error) {
	vm.Lock()
	defer vm.Unlock()

	return vm.vm.Run(src)
}

// Compile parses given source and returns otto.Script.
func (vm *VM) Compile(filename string, src interface{}) (*otto.Script, error) {
	vm.Lock()
	defer vm.Unlock()

	return vm.vm.Compile(filename, src)
}

// CompileWithSourceMap parses given source with source map and returns otto.Script.
func (vm *VM) CompileWithSourceMap(filename string, src, sm interface{}) (*otto.Script, error) {
	vm.Lock()
	defer vm.Unlock()

	return vm.vm.CompileWithSourceMap(filename, src, sm)
}

// ToValue will convert an interface{} value to a value digestible by otto/JavaScript.
func (vm *VM) ToValue(value interface{}) (otto.Value, error) {
	vm.Lock()
	defer vm.Unlock()

	return vm.vm.ToValue(value)
}
