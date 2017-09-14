package jail

import (
	"os"

	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/jail/console"
	"github.com/status-im/status-go/geth/node"
)

// signals
const (
	EventSignal          = "jail.signal"

	// EventConsoleLog defines the event type for the console.log call.
	eventConsoleLog = "vm.console.log"
)

// registerHandlers augments and transforms a given jail cell's underlying VM,
// by adding and replacing method handlers.
func registerHandlers(jail *Jail, cell common.JailCell, chatID string) error {
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
	if err = registerHandler("sendAsync", makeSendHandler(jail)); err != nil {
		return err
	}

	// register isConnected handler
	if err = registerHandler("isConnected", makeJethIsConnectedHandler(jail)); err != nil {
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
	if err = registerHandler("sendSignal", makeSignalHandler(chatID)); err != nil {
		return err
	}

	return nil
}

// makeSendHandler returns jeth.send() and jeth.sendAsync() handler
func makeSendHandler(jail *Jail) func(call otto.FunctionCall) (response otto.Value) {
	return jail.Send
}

// makeJethIsConnectedHandler returns jeth.isConnected() handler
func makeJethIsConnectedHandler(jail *Jail) func(call otto.FunctionCall) (response otto.Value) {
	return func(call otto.FunctionCall) otto.Value {
		client := jail.nodeManager.RPCClient()

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

// SignalEvent wraps Jail send signals
type SignalEvent struct {
	ChatID string `json:"chat_id"`
	Data   string `json:"data"`
}

func makeSignalHandler(chatID string) func(call otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		message := call.Argument(0).String()

		node.SendSignal(node.SignalEnvelope{
			Type: EventSignal,
			Event: SignalEvent{
				ChatID: chatID,
				Data:   message,
			},
		})

		return newResultResponse(call.Otto, true)
	}
}
