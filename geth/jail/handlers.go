package jail

import (
	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth"
)

const (
	EventLocalStorageSet   = "local_storage.set"
	LocalStorageMaxDataLen = 256
)

// makeSendHandler returns jeth.send() and jeth.sendAsync() handler
func makeSendHandler(jail *Jail, chatId string) func(call otto.FunctionCall) (response otto.Value) {
	return func(call otto.FunctionCall) (response otto.Value) {
		return jail.Send(chatId, call)
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
func makeLocalStorageSetHandler(chatId string) func(call otto.FunctionCall) (response otto.Value) {
	return func(call otto.FunctionCall) otto.Value {
		data := call.Argument(0).String()
		if len(data) > LocalStorageMaxDataLen { // cap input string
			data = data[:LocalStorageMaxDataLen]
		}

		geth.SendSignal(geth.SignalEnvelope{
			Type: EventLocalStorageSet,
			Event: geth.LocalStorageSetEvent{
				ChatId: chatId,
				Data:   data,
			},
		})

		return newResultResponse(call.Otto, true)
	}
}
