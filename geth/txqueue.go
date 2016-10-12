package geth

/*
#include <stddef.h>
#include <stdbool.h>
extern bool StatusServiceSignalEvent( const char *jsonEvent );
*/
import "C"

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/cnf/structhash"
	"github.com/eapache/go-resiliency/semaphore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/les/status"
	"github.com/robertkrimen/otto"
)

const (
	EventTransactionQueued = "transaction.queued"
	SendTransactionRequest = "eth_sendTransaction"
	MessageIdKey           = "message_id"
	CellTicketKey          = "cell_ticket"
)

func onSendTransactionRequest(queuedTx status.QueuedTx) {
	requestCtx := context.Background()
	requestQueue, err := GetNodeManager().JailedRequestQueue()
	if err == nil {
		requestCtx = requestQueue.PopQueuedTxContext(&queuedTx)
	}

	// request context obtained (if exists), safe to release the ticket
	if ticket := cellTicketFromContext(requestCtx); ticket != nil {
		ticket.Release()
	}

	event := GethEvent{
		Type: EventTransactionQueued,
		Event: SendTransactionEvent{
			Id:        string(queuedTx.Id),
			Args:      queuedTx.Args,
			MessageId: messageIdFromContext(requestCtx),
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

func cellTicketFromContext(ctx context.Context) *semaphore.Semaphore {
	if ctx == nil {
		return nil
	}
	if sem, ok := ctx.Value(CellTicketKey).(*semaphore.Semaphore); ok {
		return sem
	}

	return nil
}

type JailedRequest struct {
	method string
	ctx    context.Context
	vm     *otto.Otto
}

type JailedRequestQueue struct {
	requests map[string]*JailedRequest
}

func NewJailedRequestsQueue() *JailedRequestQueue {
	return &JailedRequestQueue{
		requests: make(map[string]*JailedRequest),
	}
}

func (q *JailedRequestQueue) PreProcessRequest(ticket *semaphore.Semaphore, vm *otto.Otto, req RPCCall) error {
	// serialize access
	err := ticket.Acquire()
	if err != nil {
		return err
	}

	messageId := currentMessageId(vm.Context())

	// save request context for reuse (by request handlers, such as queued transaction signal sender)
	ctx := context.Background()
	ctx = context.WithValue(ctx, "method", req.Method)
	if len(messageId) > 0 {
		ctx = context.WithValue(ctx, MessageIdKey, messageId)
	}

	// onSendTransactionRequest() will use context to obtain and release ticket
	if req.Method == SendTransactionRequest {
		ctx = context.WithValue(ctx, CellTicketKey, ticket)
	} else {
		ticket.Release()
	}
	q.saveRequestContext(vm, ctx, req)

	return nil
}

func (q *JailedRequestQueue) PostProcessRequest(vm *otto.Otto, req RPCCall) {
	// set message id (if present in context)
	messageId := currentMessageId(vm.Context())
	if len(messageId) > 0 {
		vm.Call("addContext", nil, messageId, MessageIdKey, messageId)
	}

	// set extra markers for queued transaction requests
	if req.Method == SendTransactionRequest {
		vm.Call("addContext", nil, messageId, SendTransactionRequest, true)
	}
}

func (q *JailedRequestQueue) saveRequestContext(vm *otto.Otto, ctx context.Context, req RPCCall) {
	hash := hashFromRPCCall(req)

	if len(hash) == 0 { // no need to persist empty hash
		return
	}

	q.requests[hash] = &JailedRequest{
		method: req.Method,
		ctx:    ctx,
		vm:     vm,
	}
}

func (q *JailedRequestQueue) GetQueuedTxContext(queuedTx *status.QueuedTx) context.Context {
	hash := hashFromQueuedTx(queuedTx)

	req, ok := q.requests[hash]
	if ok {
		return req.ctx
	}

	return context.Background()
}

func (q *JailedRequestQueue) PopQueuedTxContext(queuedTx *status.QueuedTx) context.Context {
	hash := hashFromQueuedTx(queuedTx)

	req, ok := q.requests[hash]
	if ok {
		delete(q.requests, hash)
		return req.ctx
	}

	return context.Background()
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

type HashableSendRequest struct {
	method string
	from   string
	to     string
	value  string
	data   string
}

func hashFromRPCCall(req RPCCall) string {
	if req.Method != SendTransactionRequest { // no need to persist extra state for other requests
		return ""
	}

	params, ok := req.Params[0].(map[string]interface{})
	if !ok {
		return ""
	}

	from, ok := params["from"].(string)
	if !ok {
		from = ""
	}

	to, ok := params["to"].(string)
	if !ok {
		to = ""
	}

	value, ok := params["value"].(string)
	if !ok {
		value = ""
	}

	data, ok := params["data"].(string)
	if !ok {
		data = ""
	}

	s := HashableSendRequest{
		method: req.Method,
		from:   from,
		to:     to,
		value:  value,
		data:   data,
	}

	return fmt.Sprintf("%x", structhash.Sha1(s, 1))
}

func hashFromQueuedTx(queuedTx *status.QueuedTx) string {
	value, err := queuedTx.Args.Value.MarshalJSON()
	if err != nil {
		return ""
	}

	s := HashableSendRequest{
		method: SendTransactionRequest,
		from:   queuedTx.Args.From.Hex(),
		to:     queuedTx.Args.To.Hex(),
		value:  string(bytes.Replace(value, []byte(`"`), []byte(""), 2)),
		data:   queuedTx.Args.Data,
	}

	return fmt.Sprintf("%x", structhash.Sha1(s, 1))
}
