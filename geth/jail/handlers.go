package jail

import (
	"os"

	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/jail/console"
	"github.com/status-im/status-go/geth/node"
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
		"isConnected": createIsConnectedHandler(jail, cell),
	}

	if err := cell.Set("jeth", jeth); err != nil {
		return err
	}

	return nil
}

// registerStatusSignals creates an object called "statusSignals".
// TODO(adam): describe what it is and when it's used.
func registerStatusSignals(jail *Jail, cell *Cell) error {
	statusSignals := map[string]interface{}{
		"sendSignal": createSendSignalHandler(jail, cell),
	}

	if err := cell.Set("statusSignals", statusSignals); err != nil {
		return err
	}

	return nil
}

// createSendHandler returns jeth.send() and jeth.sendAsync() handler
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

		response, err := jail.sendRPCCall(cell, request.String())
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
		go func() {
			// As it's an async call, it's not called from a thread-safe context,
			// thus using vm.VM.
			vm := cell.VM
			callback := call.Argument(1)
			isFunction := callback.Class() == "Function"

			request, err := vm.Call("JSON.stringify", nil, call.Argument(0))
			if err != nil && isFunction {
				cell.CallAsync(callback, vm.MakeCustomError("Error", err.Error()))
				return
			}

			response, err := jail.sendRPCCall(cell, request.String())
			if err != nil && isFunction {
				cell.CallAsync(callback, vm.MakeCustomError("Error", err.Error()))
				return
			}

			if isFunction {
				cell.CallAsync(callback, nil, response)
			}
		}()

		return otto.UndefinedValue()
	}
}

// createIsConnectedHandler returns jeth.isConnected() handler
// TODO(adam): according to https://github.com/ethereum/wiki/wiki/JavaScript-API#web3isconnected
// this callback should return Boolean instead of an object `{"result": Boolean}`.
// TODO(adam): remove error wrapping as it should be a custom Error object.
func createIsConnectedHandler(jail *Jail, cell *Cell) func(call otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		client := jail.rpcClientProvider()
		if client == nil {
			throwJSError(ErrNoRPCClient)
		}

		// As it's a sync call, it's called already from a thread-safe context,
		// thus using otto.Otto directly. Otherwise, it would try to acquire a lock again
		// and result in a deadlock.
		vm := cell.VM.UnsafeVM()

		var netListeningResult bool
		if err := client.Call(&netListeningResult, "net_listening"); err != nil {
			value, err := wrapErrorInValue(vm, err)
			if err != nil {
				throwJSError(err)
			}

			return value
		}

		if !netListeningResult {
			value, err := wrapErrorInValue(vm, node.ErrNoRunningNode)
			if err != nil {
				throwJSError(err)
			}

			return value
		}

		value, err := wrapResultInValue(vm, netListeningResult)
		if err != nil {
			throwJSError(err)
		}

		return value
	}
}

func createSendSignalHandler(jail *Jail, cell *Cell) func(otto.FunctionCall) otto.Value {
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

func wrapErrorInValue(vm *otto.Otto, anErr error) (value otto.Value, err error) {
	return vm.Run(`({"error":"` + anErr.Error() + `"})`)
}
