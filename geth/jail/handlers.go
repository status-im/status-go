package jail

import (
	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth"
)

const (
	// EventLocalStorageSet is triggered when set request is sent to local storage
	EventLocalStorageSet = "local_storage.set"

	// LocalStorageMaxDataLen is maximum length of data that you can store in local storage
	LocalStorageMaxDataLen = 256
)

// registerHandlers augments and transforms a given jail cell's underlying VM,
// by adding and replacing method handlers.
func registerHandlers(jail *Jail, vm *otto.Otto, chatID string) (err error) {
	jeth, err := vm.Get("jeth")
	if err != nil {
		return err
	}
	registerHandler := jeth.Object().Set

	// register send handler
	if err = registerHandler("send", makeSendHandler(jail, chatID)); err != nil {
		return err
	}

	// register sendAsync handler
	if err = registerHandler("sendAsync", makeSendHandler(jail, chatID)); err != nil {
		return err
	}

	// register isConnected handler
	if err = registerHandler("isConnected", makeJethIsConnectedHandler(jail)); err != nil {
		return err
	}

	// define localStorage
	if err = vm.Set("localStorage", struct{}{}); err != nil {
		return
	}

	// register localStorage.set handler
	localStorage, err := vm.Get("localStorage")
	if err != nil {
		return
	}
	if err = localStorage.Object().Set("set", makeLocalStorageSetHandler(chatID)); err != nil {
		return
	}

	return nil
}

// makeSendHandler returns jeth.send() and jeth.sendAsync() handler
func makeSendHandler(jail *Jail, chatID string) func(call otto.FunctionCall) (response otto.Value) {
	return func(call otto.FunctionCall) (response otto.Value) {
		return jail.Send(chatID, call)
	}
}

// makeJethIsConnectedHandler returns jeth.isConnected() handler
func makeJethIsConnectedHandler(jail *Jail) func(call otto.FunctionCall) (response otto.Value) {
	return func(call otto.FunctionCall) otto.Value {
		client, err := jail.RPCClient()
		if err != nil {
			return newErrorResponse(call.Otto, -32603, err.Error(), nil)
		}

		var netListeningResult bool
		if err := client.Call(&netListeningResult, "net_listening"); err != nil {
			return newErrorResponse(call.Otto, -32603, err.Error(), nil)
		}

		if !netListeningResult {
			return newErrorResponse(call.Otto, -32603, geth.ErrInvalidGethNode.Error(), nil)
		}

		return newResultResponse(call.Otto, true)
	}
}

// makeLocalStorageSetHandler returns localStorage.set() handler
func makeLocalStorageSetHandler(chatID string) func(call otto.FunctionCall) (response otto.Value) {
	return func(call otto.FunctionCall) otto.Value {
		data := call.Argument(0).String()
		if len(data) > LocalStorageMaxDataLen { // cap input string
			data = data[:LocalStorageMaxDataLen]
		}

		geth.SendSignal(geth.SignalEnvelope{
			Type: EventLocalStorageSet,
			Event: geth.LocalStorageSetEvent{
				ChatID: chatID,
				Data:   data,
			},
		})

		return newResultResponse(call.Otto, true)
	}
}
