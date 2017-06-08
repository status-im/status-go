package extensions

import (
	"github.com/robertkrimen/otto"
)

// ExtensionFunction types a function type which is used to define
// given extensions to be registered.
type ExtensionFunction func(*otto.Otto) error

// exts defines a package level variable to hold registered otto extensions.
var exts = struct {
	extensions []ExtensionFunction
}{
	extensions: make([]ExtensionFunction, 0),
}

// Register adds the giving extension into the appropriate extension store.
// Overrides previous key if available.
// Panics if the giving extension is not a valid convertible
// type for otto.ToValue.
func Register(extension ExtensionFunction) bool {
	exts.extensions = append(exts.extensions, extension)
	return true
}

// ActivateExtensions adds all the registered extensions to the Otto instance.
// It will immediately return an error if any of the extensions fails to
// register.
func ActivateExtensions(vm *otto.Otto) error {
	for _, extension := range exts.extensions {
		if err := extension(vm); err != nil {
			return err
		}
	}

	return nil
}
