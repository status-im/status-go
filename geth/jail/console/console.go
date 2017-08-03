package console

import (
	"fmt"
	"io"
	"strings"

	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/node"
)

// Write provides the baselevel function to writes data to the underline writer
// for the underline otto vm.
func Write(fn otto.FunctionCall, w io.Writer, ntype string) otto.Value {
	node.SendSignal(node.SignalEnvelope{
		Type:  ntype,
		Event: convertArgs(fn.ArgumentList),
	})

	// Next print out the giving values.
	writeConsole(w, "%s: %s", ntype, fn.ArgumentList)

	return otto.UndefinedValue()
}

// writeArgument takes the giving otto.Values and transform as
// needed into the appropriate writer.
func writeConsole(writer io.Writer, format string, ntype string, args []otto.Value) {
	fmt.Fprintf(writer, format, ntype, formatForConsole(args))
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
