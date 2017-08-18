package jail

import (
	"os"

	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/jail/console"
	"github.com/status-im/status-go/geth/node"
)

// signals
const (
	EventLocalStorageSet = "local_storage.set"
	EventSendMessage     = "jail.send_message"
	EventShowSuggestions = "jail.show_suggestions"

	// EventConsoleLog defines the event type for the console.log call.
	eventConsoleLog = "vm.console.log"
)

// registerHandlers augments and transforms a given jail cell's underlying VM,
// by adding and replacing method handlers.
func registerHandlers(jail *Jail, cell *JailCell, chatID string) error {
	jeth, err := cell.Get("jeth")
	if err != nil {
		return err
	}

	registerHandler := jeth.Object().Set

	if err = registerHandler("console", map[string]interface{}{
		"log": func(fn otto.FunctionCall) otto.Value {
			return console.Write(fn, os.Stdout, eventConsoleLog)
		},
	}); err != nil {
		return err
	}

	// register send handler
	if err = registerHandler("send", makeSendHandler(jail)); err != nil {
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
	if err = cell.Set("localStorage", struct{}{}); err != nil {
		return err
	}

	// register localStorage.set handler
	localStorage, err := cell.Get("localStorage")
	if err != nil {
		return err
	}

	if err = localStorage.Object().Set("set", makeLocalStorageSetHandler(chatID)); err != nil {
		return err
	}

	// register sendMessage/showSuggestions handlers
	if err = cell.Set("statusSignals", struct{}{}); err != nil {
		return err
	}

	statusSignals, err := cell.Get("statusSignals")
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
			res := jail.Send(call)

			// Deliver response to provided callback.
			newResultResponse(call.Otto, res)
		}()

		return otto.UndefinedValue()
	}
}

// makeSendHandler returns jeth.send() and jeth.sendAsync() handler
func makeSendHandler(jail *Jail) func(call otto.FunctionCall) (response otto.Value) {
	return jail.Send
}

// makeJethIsConnectedHandler returns jeth.isConnected() handler
func makeJethIsConnectedHandler(jail *Jail) func(call otto.FunctionCall) (response otto.Value) {
	return func(call otto.FunctionCall) otto.Value {
		client, err := jail.requestManager.RPCClient()
		if err != nil {
			return newErrorResponse(call.Otto, -32603, err.Error(), nil)
		}

		var netListeningResult bool
		if err := client.Call(&netListeningResult, "net_listening"); err != nil {
			return newErrorResponse(call.Otto, -32603, err.Error(), nil)
		}

		if !netListeningResult {
			return newErrorResponse(call.Otto, -32603, node.ErrNoRunningNode.Error(), nil)
		}

		return newResultResponse(call.Otto, true)
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

		node.SendSignal(node.SignalEnvelope{
			Type: EventLocalStorageSet,
			Event: LocalStorageSetEvent{
				ChatID: chatID,
				Data:   data,
			},
		})

		return newResultResponse(call.Otto, true)
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

		return newResultResponse(call.Otto, true)
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

		return newResultResponse(call.Otto, true)
	}
}
