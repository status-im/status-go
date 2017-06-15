package jail

import (
	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth"
)

// signals
const (
	EventLocalStorageSet = "local_storage.set"
	EventSendMessage     = "jail.send_message"
	EventShowSuggestions = "jail.show_suggestions"
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
	if err = registerHandler("sendAsync", makeAsyncSendHandler(jail, chatID)); err != nil {
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

	// register sendMessage/showSuggestions handlers
	if err = vm.Set("statusSignals", struct{}{}); err != nil {
		return err
	}
	statusSignals, err := vm.Get("statusSignals")
	if err != nil {
		return err
	}
	registerHandler = statusSignals.Object().Set
	if err = registerHandler("sendMessage", makeSendMessageHandler(chatID)); err != nil {
		return err
	}
	if err = registerHandler("showSuggestions", makeShowSuggestionsHandler(chatID)); err != nil {
		return err
	}

	return nil
}

// makeAsyncSendHandler returns jeth.sendAsync() handler.
func makeAsyncSendHandler(jail *Jail, chatID string) func(call otto.FunctionCall) (response otto.Value) {
	return func(call otto.FunctionCall) (response otto.Value) {
		go func() {
			res := jail.Send(chatID, call)

			// Deliver response if callback is provided.
			if call.Otto != nil {
				newResultResponse(call, res)
			}
		}()

		return otto.UndefinedValue()
	}
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

func makeSendMessageHandler(chatID string) func(call otto.FunctionCall) (response otto.Value) {
	return func(call otto.FunctionCall) otto.Value {
		message := call.Argument(0).String()

		geth.SendSignal(geth.SignalEnvelope{
			Type: EventSendMessage,
			Event: geth.SendMessageEvent{
				ChatID:  chatID,
				Message: message,
			},
		})

		return newResultResponse(call.Otto, true)
	}
}

func makeShowSuggestionsHandler(chatID string) func(call otto.FunctionCall) (response otto.Value) {
	return func(call otto.FunctionCall) otto.Value {
		suggestionsMarkup := call.Argument(0).String()

		geth.SendSignal(geth.SignalEnvelope{
			Type: EventShowSuggestions,
			Event: geth.ShowSuggestionsEvent{
				ChatID: chatID,
				Markup: suggestionsMarkup,
			},
		})

		return newResultResponse(call.Otto, true)
	}
}
