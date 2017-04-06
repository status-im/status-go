package geth

import (
	"context"
	"encoding/json"

	"github.com/teslapatrick/go-ethereum/accounts/keystore"
	"github.com/teslapatrick/go-ethereum/common"
	"github.com/teslapatrick/go-ethereum/common/hexutil"
	"github.com/teslapatrick/go-ethereum/les/status"
	"github.com/robertkrimen/otto"
)

const (
	EventTransactionQueued = "transaction.queued"
	EventTransactionFailed = "transaction.failed"
	SendTransactionRequest = "eth_sendTransaction"
	MessageIdKey           = "message_id"

	// tx error codes
	SendTransactionNoErrorCode        = "0"
	SendTransactionDefaultErrorCode   = "1"
	SendTransactionPasswordErrorCode  = "2"
	SendTransactionTimeoutErrorCode   = "3"
	SendTransactionDiscardedErrorCode = "4"
)

func onSendTransactionRequest(queuedTx status.QueuedTx) {
	SendSignal(SignalEnvelope{
		Type: EventTransactionQueued,
		Event: SendTransactionEvent{
			Id:        string(queuedTx.Id),
			Args:      queuedTx.Args,
			MessageId: messageIdFromContext(queuedTx.Context),
		},
	})
}

func onSendTransactionReturn(queuedTx *status.QueuedTx, err error) {
	if err == nil {
		return
	}

	// discard notifications with empty tx
	if queuedTx == nil {
		return
	}

	// error occurred, signal up to application
	SendSignal(SignalEnvelope{
		Type: EventTransactionFailed,
		Event: ReturnSendTransactionEvent{
			Id:           string(queuedTx.Id),
			Args:         queuedTx.Args,
			MessageId:    messageIdFromContext(queuedTx.Context),
			ErrorMessage: err.Error(),
			ErrorCode:    sendTransactionErrorCode(err),
		},
	})
}

func sendTransactionErrorCode(err error) string {
	if err == nil {
		return SendTransactionNoErrorCode
	}

	switch err {
	case keystore.ErrDecrypt:
		return SendTransactionPasswordErrorCode
	case status.ErrQueuedTxTimedOut:
		return SendTransactionTimeoutErrorCode
	case status.ErrQueuedTxDiscarded:
		return SendTransactionDiscardedErrorCode
	default:
		return SendTransactionDefaultErrorCode
	}
}

func CompleteTransaction(id, password string) (common.Hash, error) {
	lightEthereum, err := NodeManagerInstance().LightEthereumService()
	if err != nil {
		return common.Hash{}, err
	}

	backend := lightEthereum.StatusBackend

	ctx := context.Background()
	ctx = context.WithValue(ctx, status.SelectedAccountKey, NodeManagerInstance().SelectedAccount.Hex())

	return backend.CompleteQueuedTransaction(ctx, status.QueuedTxId(id), password)
}

func CompleteTransactions(ids, password string) map[string]RawCompleteTransactionResult {
	results := make(map[string]RawCompleteTransactionResult)

	parsedIds, err := parseJSONArray(ids)
	if err != nil {
		results["none"] = RawCompleteTransactionResult{
			Error: err,
		}
		return results
	}

	for _, txId := range parsedIds {
		txHash, txErr := CompleteTransaction(txId, password)
		results[txId] = RawCompleteTransactionResult{
			Hash:  txHash,
			Error: txErr,
		}
	}

	return results
}

func DiscardTransaction(id string) error {
	lightEthereum, err := NodeManagerInstance().LightEthereumService()
	if err != nil {
		return err
	}

	backend := lightEthereum.StatusBackend

	return backend.DiscardQueuedTransaction(status.QueuedTxId(id))
}

func DiscardTransactions(ids string) map[string]RawDiscardTransactionResult {
	var parsedIds []string
	results := make(map[string]RawDiscardTransactionResult)

	parsedIds, err := parseJSONArray(ids)
	if err != nil {
		results["none"] = RawDiscardTransactionResult{
			Error: err,
		}
		return results
	}

	for _, txId := range parsedIds {
		err := DiscardTransaction(txId)
		if err != nil {
			results[txId] = RawDiscardTransactionResult{
				Error: err,
			}
		}
	}

	return results
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
	lightEthereum, err := NodeManagerInstance().LightEthereumService()
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

	return status.SendTxArgs{
		From:     req.parseFromAddress(),
		To:       req.parseToAddress(),
		Value:    req.parseValue(),
		Data:     req.parseData(),
		Gas:      req.parseGas(),
		GasPrice: req.parseGasPrice(),
	}
}

func (r RPCCall) parseFromAddress() common.Address {
	params, ok := r.Params[0].(map[string]interface{})
	if !ok {
		return common.HexToAddress("0x")
	}

	from, ok := params["from"].(string)
	if !ok {
		from = "0x"
	}

	return common.HexToAddress(from)
}

func (r RPCCall) parseToAddress() *common.Address {
	params, ok := r.Params[0].(map[string]interface{})
	if !ok {
		return nil
	}

	to, ok := params["to"].(string)
	if !ok {
		return nil
	}

	address := common.HexToAddress(to)
	return &address
}

func (r RPCCall) parseData() hexutil.Bytes {
	params, ok := r.Params[0].(map[string]interface{})
	if !ok {
		return hexutil.Bytes("0x")
	}

	data, ok := params["data"].(string)
	if !ok {
		data = "0x"
	}

	byteCode, err := hexutil.Decode(data)
	if err != nil {
		byteCode = hexutil.Bytes(data)
	}

	return byteCode
}

func (r RPCCall) parseValue() *hexutil.Big {
	params, ok := r.Params[0].(map[string]interface{})
	if !ok {
		return nil
		//return (*hexutil.Big)(big.NewInt("0x0"))
	}

	inputValue, ok := params["value"].(string)
	if !ok {
		return nil
	}

	parsedValue, err := hexutil.DecodeBig(inputValue)
	if err != nil {
		return nil
	}

	return (*hexutil.Big)(parsedValue)
}

func (r RPCCall) parseGas() *hexutil.Big {
	params, ok := r.Params[0].(map[string]interface{})
	if !ok {
		return nil
	}

	inputValue, ok := params["gas"].(string)
	if !ok {
		return nil
	}

	parsedValue, err := hexutil.DecodeBig(inputValue)
	if err != nil {
		return nil
	}

	return (*hexutil.Big)(parsedValue)
}

func (r RPCCall) parseGasPrice() *hexutil.Big {
	params, ok := r.Params[0].(map[string]interface{})
	if !ok {
		return nil
	}

	inputValue, ok := params["gasPrice"].(string)
	if !ok {
		return nil
	}

	parsedValue, err := hexutil.DecodeBig(inputValue)
	if err != nil {
		return nil
	}

	return (*hexutil.Big)(parsedValue)
}

func parseJSONArray(items string) ([]string, error) {
	var parsedItems []string
	err := json.Unmarshal([]byte(items), &parsedItems)
	if err != nil {
		return nil, err
	}

	return parsedItems, nil
}
