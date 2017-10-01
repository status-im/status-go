package jail

import (
	"os"

	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/jail/console"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/signal"
)

// signals
const (
	EventSignal = "jail.signal"

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
	if err = registerHandler("send", makeSendHandler(jail, cell)); err != nil {
		return err
	}

	// register sendAsync handler
	if err = registerHandler("sendAsync", makeAsyncSendHandler(jail, cell)); err != nil {
		return err
	}

	// register isConnected handler
	if err = registerHandler("isConnected", makeJethIsConnectedHandler(jail, cell)); err != nil {
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

// makeAsyncSendHandler returns jeth.sendAsync() handler.
func makeAsyncSendHandler(jail *Jail, cellInt common.JailCell) func(call otto.FunctionCall) otto.Value {
	// FIXME(tiabc): Get rid of this.
	cell := cellInt.(*Cell)
	return func(call otto.FunctionCall) otto.Value {
		go func() {
			response := jail.Send(call)

			// run callback asyncronously with args (error, response)
			callback := call.Argument(1)
			err := otto.NullValue()
			cell.CallAsync(callback, err, response)
		}()
		return otto.UndefinedValue()
	}
}

// makeSendHandler returns jeth.send() and jeth.sendAsync() handler
func makeSendHandler(jail *Jail, cellInt common.JailCell) func(call otto.FunctionCall) otto.Value {
	// FIXME(tiabc): Get rid of this.
	cell := cellInt.(*Cell)
	return func(call otto.FunctionCall) otto.Value {
		// Send calls are guaranteed to be only invoked from web3 after calling the appropriate
		// method of jail.Cell and the cell is locked during that call. In order to allow jail.Send
		// to perform any operations on cell.VM and not hang, we need to unlock the mutex and return
		// it to the previous state afterwards so that the caller didn't panic doing cell.Unlock().
		cell.Unlock()
		defer cell.Lock()

		return jail.Send(call)
	}
}

// makeJethIsConnectedHandler returns jeth.isConnected() handler
func makeJethIsConnectedHandler(jail *Jail, cellInt common.JailCell) func(call otto.FunctionCall) (response otto.Value) {
	// FIXME(tiabc): Get rid of this.
	cell := cellInt.(*Cell)
	return func(call otto.FunctionCall) otto.Value {
		client := jail.nodeManager.RPCClient()

		var netListeningResult bool
		if err := client.Call(&netListeningResult, "net_listening"); err != nil {
			return newErrorResponseOtto(cell.VM, err.Error(), nil)
		}

		if !netListeningResult {
			return newErrorResponseOtto(cell.VM, node.ErrNoRunningNode.Error(), nil)
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

		signal.Send(signal.Envelope{
			Type: EventSignal,
			Event: SignalEvent{
				ChatID: chatID,
				Data:   message,
			},
		})

		return newResultResponse(call.Otto, true)
	}
}
