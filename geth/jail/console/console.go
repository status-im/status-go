package console

import (
	"fmt"
	"io"
	"strings"

	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/node"
)

const (
	// EventConsoleLog defines the event type for the console.log call.
	EventConsoleLog = "vm.console.log"

	// EventConsoleWarn defines the event type for the console.debug call.
	EventConsoleWarn = "vm.console.warn"

	// EventConsoleDebug defines the event type for the console.debug call.
	EventConsoleDebug = "vm.console.debug"

	// EventConsoleError defines the event type for the console.error call.
	EventConsoleError = "vm.console.error"
)

// Extension takes a giving extension function and writer and returns a standard otto.Function
// callable by a otto vm.
func Extension(w io.Writer, ext func(otto.FunctionCall, io.Writer) otto.Value) func(otto.FunctionCall) otto.Value {
	return func(fn otto.FunctionCall) otto.Value {
		return ext(fn, w)
	}
}

// Log provides the function caller for handling console.log
// calls as replacement for the default console.log function within a
// otto.Otto VM instance.
func Log(fn otto.FunctionCall, w io.Writer) otto.Value {
	// Record provided values into store for delviery.
	node.SendSignal(node.SignalEnvelope{
		Type:  EventConsoleLog,
		Event: convertArgs(fn.ArgumentList),
	})

	// Next print out the giving values.
	handleConsole(w, "console.log: %s", fn.ArgumentList)

	return otto.UndefinedValue()
}

// Warn provides the function caller for handling console.warn
// calls as replacement for the default console.log function within a
// otto.Otto VM instance.
func Warn(fn otto.FunctionCall, w io.Writer) otto.Value {
	// Record provided values into store for delviery.
	node.SendSignal(node.SignalEnvelope{
		Type:  EventConsoleWarn,
		Event: convertArgs(fn.ArgumentList),
	})

	// Next print out the giving values.
	handleConsole(w, "console.warn: %s", fn.ArgumentList)

	return otto.UndefinedValue()
}

// Debug provides the function caller for handling console.debug
// calls as replacement for the default console.Error function within a
// otto.Otto VM instance.
func Debug(fn otto.FunctionCall, w io.Writer) otto.Value {
	// Record provided values into store for delviery.
	node.SendSignal(node.SignalEnvelope{
		Type:  EventConsoleDebug,
		Event: convertArgs(fn.ArgumentList),
	})

	// Next print out the giving values.
	handleConsole(w, "console.debug: %s", fn.ArgumentList)

	return otto.UndefinedValue()
}

// Error provides the function caller for handling console.error
// calls as replacement for the default console.Error function within a
// otto.Otto VM instance.
func Error(fn otto.FunctionCall, w io.Writer) otto.Value {
	// Record provided values into store for delviery.
	node.SendSignal(node.SignalEnvelope{
		Type:  EventConsoleError,
		Event: convertArgs(fn.ArgumentList),
	})

	// Next print out the giving values.
	handleConsole(w, "console.error: %s", fn.ArgumentList)

	return otto.UndefinedValue()
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

// handleConsole takes the giving otto.Values and transform as
// needed into the appropriate writer.
func handleConsole(writer io.Writer, format string, args []otto.Value) {
	fmt.Fprintf(writer, format, formatForConsole(args))
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
