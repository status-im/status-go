package process // import "fknsrs.biz/p/ottoext/process"

import (
	"os"
	"strings"

	"github.com/robertkrimen/otto"
)

func Define(vm *otto.Otto, argv []string) error {
	if v, err := vm.Get("process"); err != nil {
		return err
	} else if !v.IsUndefined() {
		return nil
	}

	env := make(map[string]string)
	for _, e := range os.Environ() {
		a := strings.SplitN(e, "=", 2)
		env[a[0]] = a[1]
	}

	return vm.Set("process", map[string]interface{}{
		"env":  env,
		"argv": argv,
	})
}
