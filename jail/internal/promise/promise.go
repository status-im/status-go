package promise

import (
	"github.com/status-im/status-go/jail/internal/loop"
	"github.com/status-im/status-go/jail/internal/timers"
	"github.com/status-im/status-go/jail/internal/vm"
)

//Define jail promise
func Define(vm *vm.VM, l *loop.Loop) error {
	if v, err := vm.Get("Promise"); err != nil {
		return err
	} else if !v.IsUndefined() {
		return nil
	}

	if err := timers.Define(vm, l); err != nil {
		return err
	}

	s, err := vm.Compile("promise-bundle.js", src)
	if err != nil {
		return err
	}

	if _, err := vm.Run(s); err != nil {
		return err
	}

	return nil
}
