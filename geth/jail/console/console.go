package console

import (
	"fmt"
	"io"
	"strings"

	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/signal"
)

// Write provides the base function to write data to the underline writer
// for the underline otto vm.
func Write(fn otto.FunctionCall, w io.Writer) otto.Value {
	args := convertArgs(fn.ArgumentList)
	signal.SendConsole(args)

	// Next print out the giving values.
	fmt.Fprintf(w, "%s: %s", signal.EventVMConsole, formatForConsole(fn.ArgumentList))

	return otto.UndefinedValue()
}

// formatForConsole handles conversion of giving otto.Values into
// string counter part.
func formatForConsole(argumentList []otto.Value) string {
	output := []string{}
	for _, argument := range argumentList {
		output = append(output, fmt.Sprintf("%v", argument))
	}
	return strings.Join(output, " ")
}

// convertArgs attempts to convert otto.Values into proper go types else
// uses original.
func convertArgs(argumentList []otto.Value) []interface{} {
	var items []interface{}

	for _, arg := range argumentList {
		realArg, err := arg.Export()
		if err != nil {
			items = append(items, arg)
			continue
		}

		items = append(items, realArg)
	}

	return items
}
