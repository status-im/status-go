package console

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth"
	"github.com/status-im/status-go/geth/jail/extensions"
)

var (
	// Stdout defines the default writer for the console.log
	// delivery call.
	Stdout io.Writer = os.Stdout

	// EventConsoleLog defines the event type for the console.log call.
	EventConsoleLog = "vm.console.log"

	// EventConsoleWarn defines the event type for the console.debug call.
	EventConsoleWarn = "vm.console.warn"

	// EventConsoleDebug defines the event type for the console.debug call.
	EventConsoleDebug = "vm.console.debug"

	// EventConsoleError defines the event type for the console.error call.
	EventConsoleError = "vm.console.error"

	_ = extensions.Register(func(vm *otto.Otto) error {
		return vm.Set("console", map[string]interface{}{
			"log":   consoleLog,
			"error": consoleError,
			"debug": consoleDebug,
			"warn":  consoleWarn,
		})
	})
)

// consoleLog provides the function caller for handling console.log
// calls as replacement for the default console.log function within a
// otto.Otto VM instance.
func consoleLog(fn otto.FunctionCall) otto.Value {

	// Record provided values into store for delviery.
	geth.SendSignal(geth.SignalEnvelope{
		Type:  EventConsoleLog,
		Event: convertArgs(fn.ArgumentList),
	})

	// Next print out the giving values.
	handleConsole("console.log: %s", Stdout, fn.ArgumentList)

	return otto.UndefinedValue()
}

// consoleWarn provides the function caller for handling console.warn
// calls as replacement for the default console.log function within a
// otto.Otto VM instance.
func consoleWarn(fn otto.FunctionCall) otto.Value {

	// Record provided values into store for delviery.
	geth.SendSignal(geth.SignalEnvelope{
		Type:  EventConsoleWarn,
		Event: convertArgs(fn.ArgumentList),
	})

	// Next print out the giving values.
	handleConsole("console.warn: %s", Stdout, fn.ArgumentList)

	return otto.UndefinedValue()
}

// consoleDebug provides the function caller for handling console.debug
// calls as replacement for the default console.Error function within a
// otto.Otto VM instance.
func consoleDebug(fn otto.FunctionCall) otto.Value {

	// Record provided values into store for delviery.
	geth.SendSignal(geth.SignalEnvelope{
		Type:  EventConsoleDebug,
		Event: convertArgs(fn.ArgumentList),
	})

	// Next print out the giving values.
	handleConsole("console.debug: %s", Stdout, fn.ArgumentList)

	return otto.UndefinedValue()
}

// consoleError provides the function caller for handling console.error
// calls as replacement for the default console.Error function within a
// otto.Otto VM instance.
func consoleError(fn otto.FunctionCall) otto.Value {

	// Record provided values into store for delviery.
	geth.SendSignal(geth.SignalEnvelope{
		Type:  EventConsoleLog,
		Event: convertArgs(fn.ArgumentList),
	})

	// Next print out the giving values.
	handleConsole("console.error: %s", Stdout, fn.ArgumentList)

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
func handleConsole(tag string, writer io.Writer, args []otto.Value) {
	fmt.Fprintf(writer, tag, formatForConsole(args))
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
