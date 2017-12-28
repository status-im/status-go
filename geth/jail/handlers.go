package jail

import (
	"os"

	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/jail/console"
	"github.com/status-im/status-go/geth/signal"
)

const (
	// EventSignal is a signal from jail.
	EventSignal = "jail.signal"
	// eventConsoleLog defines the event type for the console.log call.
	eventConsoleLog = "vm.console.log"
)

// registerWeb3Provider creates an object called "jeth",
// which is a web3.js provider.
func registerWeb3Provider(jail *Jail, cell *Cell) error {
	jeth := map[string]interface{}{
		"console": map[string]interface{}{
			"log": func(fn otto.FunctionCall) otto.Value {
				return console.Write(fn, os.Stdout, eventConsoleLog)
			},
		},
		"send":        createSendHandler(jail, cell),
		"sendAsync":   createSendAsyncHandler(jail, cell),
		"isConnected": createIsConnectedHandler(jail),
	}

	return cell.Set("jeth", jeth)
}

// registerStatusSignals creates an object called "statusSignals".
// TODO(adam): describe what it is and when it's used.
func registerStatusSignals(cell *Cell) error {
	statusSignals := map[string]interface{}{
		"sendSignal": createSendSignalHandler(cell),
	}

	return cell.Set("statusSignals", statusSignals)
}

// createSendHandler returns jeth.send().
func createSendHandler(jail *Jail, cell *Cell) func(call otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		// As it's a sync call, it's called already from a thread-safe context,
		// thus using otto.Otto directly. Otherwise, it would try to acquire a lock again
		// and result in a deadlock.
		vm := cell.VM.UnsafeVM()

		request, err := vm.Call("JSON.stringify", nil, call.Argument(0))
		if err != nil {
			throwJSError(err)
		}

		response, err := jail.sendRPCCall(request.String())
		if err != nil {
			throwJSError(err)
		}

		value, err := vm.ToValue(response)
		if err != nil {
			throwJSError(err)
		}

		return value
	}
}

// createSendAsyncHandler returns jeth.sendAsync() handler.
func createSendAsyncHandler(jail *Jail, cell *Cell) func(call otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		// As it's a sync call, it's called already from a thread-safe context,
		// thus using otto.Otto directly. Otherwise, it would try to acquire a lock again
		// and result in a deadlock.
		unsafeVM := cell.VM.UnsafeVM()

		request, err := unsafeVM.Call("JSON.stringify", nil, call.Argument(0))
		if err != nil {
			throwJSError(err)
		}

		go func() {
			// As it's an async call, it's not called from a thread-safe context,
			// thus using a thread-safe vm.VM.
			vm := cell.VM
			callback := call.Argument(1)
			response, err := jail.sendRPCCall(request.String())

			// If provided callback argument is not a function, don't call it.
			if callback.Class() != "Function" {
				return
			}

			// nolint: errcheck
			if err != nil {
				cell.CallAsync(callback, vm.MakeCustomError("Error", err.Error()))
			} else {
				cell.CallAsync(callback, nil, response)
			}
		}()

		return otto.UndefinedValue()
	}
}

// createIsConnectedHandler returns jeth.isConnected() handler.
// This handler returns `true` if client is actively listening for network connections.
func createIsConnectedHandler(jail RPCClientProvider) func(call otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		client := jail.RPCClient()
		if client == nil {
			throwJSError(ErrNoRPCClient)
		}

		var netListeningResult bool
		if err := client.Call(&netListeningResult, "net_listening"); err != nil {
			throwJSError(err)
		}

		if netListeningResult {
			return otto.TrueValue()
		}

		return otto.FalseValue()
	}
}

func createSendSignalHandler(cell *Cell) func(otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		message := call.Argument(0).String()

		signal.Send(signal.Envelope{
			Type: EventSignal,
			Event: struct {
				ChatID string `json:"chat_id"`
				Data   string `json:"data"`
			}{
				ChatID: cell.id,
				Data:   message,
			},
		})

		// As it's a sync call, it's called already from a thread-safe context,
		// thus using otto.Otto directly. Otherwise, it would try to acquire a lock again
		// and result in a deadlock.
		vm := cell.VM.UnsafeVM()

		value, err := wrapResultInValue(vm, otto.TrueValue())
		if err != nil {
			throwJSError(err)
		}

		return value
	}
}

// throwJSError calls panic with an error string. It should be called
// only in a context that handles panics like otto.Otto.
func throwJSError(err error) {
	value, err := otto.ToValue(err.Error())
	if err != nil {
		panic(err.Error())
	}

	panic(value)
}

func wrapResultInValue(vm *otto.Otto, result interface{}) (value otto.Value, err error) {
	value, err = vm.Run(`({})`)
	if err != nil {
		return
	}

	err = value.Object().Set("result", result)
	if err != nil {
		return
	}

	return
}
