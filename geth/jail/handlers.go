package jail

import (
	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/node"
)

const (
	EventLocalStorageSet = "local_storage.set"
	EventSendMessage = "jail.send_message"
	EventShowSuggestions = "jail.show_suggestions"
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
		client, err := jail.requestManager.RPCClient()
		if err != nil {
			return newErrorResponse(call, -32603, err.Error(), nil)
		}

		var netListeningResult bool
		if err := client.Call(&netListeningResult, "net_listening"); err != nil {
			return newErrorResponse(call, -32603, err.Error(), nil)
		}

		if !netListeningResult {
			return newErrorResponse(call, -32603, node.ErrNoRunningNode.Error(), nil)
		}

		return newResultResponse(call, true)
	}
}

// LocalStorageSetEvent is a signal sent whenever local storage Set method is called
type LocalStorageSetEvent struct {
	ChatID string `json:"chat_id"`
	Data   string `json:"data"`
}

// makeLocalStorageSetHandler returns localStorage.set() handler
func makeLocalStorageSetHandler(chatID string) func(call otto.FunctionCall) (response otto.Value) {
	return func(call otto.FunctionCall) otto.Value {
		data := call.Argument(0).String()
		if len(data) > LocalStorageMaxDataLen { // cap input string
			data = data[:LocalStorageMaxDataLen]
		}

		node.SendSignal(node.SignalEnvelope{
			Type: EventLocalStorageSet,
			Event: LocalStorageSetEvent{
				ChatID: chatID,
				Data:   data,
			},
		})

		return newResultResponse(call, true)
	}
}

// SendMessageEvent wraps Jail send signals
type SendMessageEvent struct {
	ChatID  string `json:"chat_id"`
	Message string `json:"message"`
}

func makeSendMessageHandler(chatID string) func(call otto.FunctionCall) (response otto.Value) {
	return func(call otto.FunctionCall) otto.Value {
		message := call.Argument(0).String()

		node.SendSignal(node.SignalEnvelope{
			Type: EventSendMessage,
			Event: SendMessageEvent{
				ChatID:  chatID,
				Message: message,
			},
		})

		return newResultResponse(call, true)
	}
}

// ShowSuggestionsEvent wraps Jail show suggestion signals
type ShowSuggestionsEvent struct {
	ChatID string `json:"chat_id"`
	Markup string `json:"markup"`
}

func makeShowSuggestionsHandler(chatID string) func(call otto.FunctionCall) (response otto.Value) {
	return func(call otto.FunctionCall) otto.Value {
		suggestionsMarkup := call.Argument(0).String()

		node.SendSignal(node.SignalEnvelope{
			Type: EventShowSuggestions,
			Event: ShowSuggestionsEvent{
				ChatID: chatID,
				Markup: suggestionsMarkup,
			},
		})

		return newResultResponse(call, true)
	}
}
