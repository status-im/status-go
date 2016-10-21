package geth

/*
#include <stddef.h>
#include <stdbool.h>
extern bool StatusServiceSignalEvent( const char *jsonEvent );
*/
import "C"

import (
	"context"
	"encoding/json"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/les/status"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/robertkrimen/otto"
)

const (
	EventTransactionQueued = "transaction.queued"
	SendTransactionRequest = "eth_sendTransaction"
	MessageIdKey           = "message_id"
)

func onSendTransactionRequest(queuedTx status.QueuedTx) {
	event := GethEvent{
		Type: EventTransactionQueued,
		Event: SendTransactionEvent{
			Id:        string(queuedTx.Id),
			Args:      queuedTx.Args,
			MessageId: messageIdFromContext(queuedTx.Context),
		},
	}

	body, _ := json.Marshal(&event)
	C.StatusServiceSignalEvent(C.CString(string(body)))
}

func CompleteTransaction(id, password string) (common.Hash, error) {
	lightEthereum, err := GetNodeManager().LightEthereumService()
	if err != nil {
		return common.Hash{}, err
	}

	backend := lightEthereum.StatusBackend

	return backend.CompleteQueuedTransaction(status.QueuedTxId(id), password)
}

func messageIdFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if messageId, ok := ctx.Value(MessageIdKey).(string); ok {
		return messageId
	}

	return ""
}

type JailedRequestQueue struct{}

func NewJailedRequestsQueue() *JailedRequestQueue {
	return &JailedRequestQueue{}
}

func (q *JailedRequestQueue) PreProcessRequest(vm *otto.Otto, req RPCCall) (string, error) {
	messageId := currentMessageId(vm.Context())

	return messageId, nil
}

func (q *JailedRequestQueue) PostProcessRequest(vm *otto.Otto, req RPCCall, messageId string) {
	if len(messageId) > 0 {
		vm.Call("addContext", nil, messageId, MessageIdKey, messageId)
	}

	// set extra markers for queued transaction requests
	if req.Method == SendTransactionRequest {
		vm.Call("addContext", nil, messageId, SendTransactionRequest, true)
	}
}

func (q *JailedRequestQueue) ProcessSendTransactionRequest(vm *otto.Otto, req RPCCall) (common.Hash, error) {
	// obtain status backend from LES service
	lightEthereum, err := GetNodeManager().LightEthereumService()
	if err != nil {
		return common.Hash{}, err
	}
	backend := lightEthereum.StatusBackend

	messageId, err := q.PreProcessRequest(vm, req)
	if err != nil {
		return common.Hash{}, err
	}
	// onSendTransactionRequest() will use context to obtain and release ticket
	ctx := context.Background()
	ctx = context.WithValue(ctx, MessageIdKey, messageId)

	//  this call blocks, up until Complete Transaction is called
	txHash, err := backend.SendTransaction(ctx, sendTxArgsFromRPCCall(req))
	if err != nil {
		return common.Hash{}, err
	}

	// invoke post processing
	q.PostProcessRequest(vm, req, messageId)

	return txHash, nil
}

// currentMessageId looks for `status.message_id` variable in current JS context
func currentMessageId(ctx otto.Context) string {
	if statusObj, ok := ctx.Symbols["status"]; ok {
		messageId, err := statusObj.Object().Get("message_id")
		if err != nil {
			return ""
		}
		if messageId, err := messageId.ToString(); err == nil {
			return messageId
		}
	}

	return ""
}

func sendTxArgsFromRPCCall(req RPCCall) status.SendTxArgs {
	if req.Method != SendTransactionRequest { // no need to persist extra state for other requests
		return status.SendTxArgs{}
	}

	params, ok := req.Params[0].(map[string]interface{})
	if !ok {
		return status.SendTxArgs{}
	}

	from, ok := params["from"].(string)
	if !ok {
		from = ""
	}

	to, ok := params["to"].(string)
	if !ok {
		to = ""
	}

	param, ok := params["value"].(string)
	if !ok {
		param = "0x0"
	}
	value, err := strconv.ParseInt(param, 0, 64)
	if err != nil {
		return status.SendTxArgs{}
	}

	data, ok := params["data"].(string)
	if !ok {
		data = ""
	}

	toAddress := common.HexToAddress(to)
	return status.SendTxArgs{
		From:  common.HexToAddress(from),
		To:    &toAddress,
		Value: rpc.NewHexNumber(big.NewInt(value)),
		Data:  data,
	}
}
