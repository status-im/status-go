package geth

import (
	"context"
	"encoding/json"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/les/status"
	"github.com/robertkrimen/otto"
)

const (
	// EventTransactionQueued is triggered whan send transaction request is queued
	EventTransactionQueued = "transaction.queued"

	// EventTransactionFailed is triggered when send transaction request fails
	EventTransactionFailed = "transaction.failed"

	// SendTransactionRequest is triggered on send transaction request
	SendTransactionRequest = "eth_sendTransaction"

	// MessageIDKey is a key for message ID
	// This ID is required to track from which chat a given send transaction request is coming.
	MessageIDKey = contextKey("message_id")
)

type contextKey string // in order to make sure that our context key does not collide with keys from other packages

// Send transaction response codes
const (
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
			ID:        string(queuedTx.ID),
			Args:      queuedTx.Args,
			MessageID: messageIDFromContext(queuedTx.Context),
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
			ID:           string(queuedTx.ID),
			Args:         queuedTx.Args,
			MessageID:    messageIDFromContext(queuedTx.Context),
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

// CompleteTransaction instructs backend to complete sending of a given transaction
func CompleteTransaction(id, password string) (common.Hash, error) {
	lightEthereum, err := NodeManagerInstance().LightEthereumService()
	if err != nil {
		return common.Hash{}, err
	}

	backend := lightEthereum.StatusBackend

	ctx := context.Background()
	ctx = context.WithValue(ctx, status.SelectedAccountKey, NodeManagerInstance().SelectedAccount.Hex())

	return backend.CompleteQueuedTransaction(ctx, status.QueuedTxID(id), password)
}

// CompleteTransactions instructs backend to complete sending of multiple transactions
func CompleteTransactions(ids, password string) map[string]RawCompleteTransactionResult {
	results := make(map[string]RawCompleteTransactionResult)

	parsedIDs, err := parseJSONArray(ids)
	if err != nil {
		results["none"] = RawCompleteTransactionResult{
			Error: err,
		}
		return results
	}

	for _, txID := range parsedIDs {
		txHash, txErr := CompleteTransaction(txID, password)
		results[txID] = RawCompleteTransactionResult{
			Hash:  txHash,
			Error: txErr,
		}
	}

	return results
}

// DiscardTransaction discards a given transaction from transaction queue
func DiscardTransaction(id string) error {
	lightEthereum, err := NodeManagerInstance().LightEthereumService()
	if err != nil {
		return err
	}

	backend := lightEthereum.StatusBackend

	return backend.DiscardQueuedTransaction(status.QueuedTxID(id))
}

// DiscardTransactions discards given multiple transactions from transaction queue
func DiscardTransactions(ids string) map[string]RawDiscardTransactionResult {
	var parsedIDs []string
	results := make(map[string]RawDiscardTransactionResult)

	parsedIDs, err := parseJSONArray(ids)
	if err != nil {
		results["none"] = RawDiscardTransactionResult{
			Error: err,
		}
		return results
	}

	for _, txID := range parsedIDs {
		err := DiscardTransaction(txID)
		if err != nil {
			results[txID] = RawDiscardTransactionResult{
				Error: err,
			}
		}
	}

	return results
}

func messageIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if messageID, ok := ctx.Value(MessageIDKey).(string); ok {
		return messageID
	}

	return ""
}

// JailedRequestQueue is used for allowing request pre and post processing.
// Such processing may include validation, injection of params (like message ID) etc
type JailedRequestQueue struct{}

// NewJailedRequestsQueue returns new instance of request queue
func NewJailedRequestsQueue() *JailedRequestQueue {
	return &JailedRequestQueue{}
}

// PreProcessRequest pre-processes a given RPC call to a given Otto VM
func (q *JailedRequestQueue) PreProcessRequest(vm *otto.Otto, req RPCCall) (string, error) {
	messageID := currentMessageID(vm.Context())

	return messageID, nil
}

// PostProcessRequest post-processes a given RPC call to a given Otto VM
func (q *JailedRequestQueue) PostProcessRequest(vm *otto.Otto, req RPCCall, messageID string) {
	if len(messageID) > 0 {
		vm.Call("addContext", nil, messageID, MessageIDKey, messageID) // nolint: errcheck
	}

	// set extra markers for queued transaction requests
	if req.Method == SendTransactionRequest {
		vm.Call("addContext", nil, messageID, SendTransactionRequest, true) // nolint: errcheck
	}
}

// ProcessSendTransactionRequest processes send transaction request.
// Both pre and post processing happens within this function. Pre-processing
// happens before transaction is send to backend, and post processing occurs
// when backend notifies that transaction sending is complete (either successfully
// or with error)
func (q *JailedRequestQueue) ProcessSendTransactionRequest(vm *otto.Otto, req RPCCall) (common.Hash, error) {
	// obtain status backend from LES service
	lightEthereum, err := NodeManagerInstance().LightEthereumService()
	if err != nil {
		return common.Hash{}, err
	}
	backend := lightEthereum.StatusBackend

	messageID, err := q.PreProcessRequest(vm, req)
	if err != nil {
		return common.Hash{}, err
	}
	// onSendTransactionRequest() will use context to obtain and release ticket
	ctx := context.Background()
	ctx = context.WithValue(ctx, MessageIDKey, messageID)

	//  this call blocks, up until Complete Transaction is called
	txHash, err := backend.SendTransaction(ctx, sendTxArgsFromRPCCall(req))
	if err != nil {
		return common.Hash{}, err
	}

	// invoke post processing
	q.PostProcessRequest(vm, req, messageID)

	return txHash, nil
}

// currentMessageID looks for `status.message_id` variable in current JS context
func currentMessageID(ctx otto.Context) string {
	if statusObj, ok := ctx.Symbols["status"]; ok {
		messageID, err := statusObj.Object().Get("message_id")
		if err != nil {
			return ""
		}
		if messageID, err := messageID.ToString(); err == nil {
			return messageID
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

// nolint: dupl
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

// nolint: dupl
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

// nolint: dupl
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
