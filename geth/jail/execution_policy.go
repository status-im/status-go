package jail

import (
	"context"
	"encoding/json"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
)

// ExecutionPolicy provides a central container for the executions of RPCCall requests for both
// remote/upstream processing and internal node processing.
type ExecutionPolicy struct {
	nodeManager    common.NodeManager
	accountManager common.AccountManager
	txQueueManager common.TxQueueManager
}

// NewExecutionPolicy returns a new instance of ExecutionPolicy.
func NewExecutionPolicy(
	nodeManager common.NodeManager, accountManager common.AccountManager, txQueueManager common.TxQueueManager,
) *ExecutionPolicy {
	return &ExecutionPolicy{
		nodeManager:    nodeManager,
		accountManager: accountManager,
		txQueueManager: txQueueManager,
	}
}

func (ep *ExecutionPolicy) Execute(req common.RPCCall, call otto.FunctionCall) (*otto.Object, error) {
	switch req.Method {
	case params.SendTransactionMethodName:
		return ep.executeSendTransaction(req, call)
	default:
		return ep.executeOtherTransaction(req, call)
	}
}

// ExecuteSendTransaction defines a function to execute RPC requests for eth_sendTransaction method only.
func (ep *ExecutionPolicy) executeSendTransaction(req common.RPCCall, call otto.FunctionCall) (*otto.Object, error) {
	res, err := call.Otto.Object(`({"jsonrpc":"2.0"})`)
	if err != nil {
		return nil, err
	}

	res.Set("id", req.ID)

	messageID, err := preProcessRequest(call.Otto, req)
	if err != nil {
		return nil, err
	}

	// TODO(adam): check if context is used
	ctx := context.WithValue(context.Background(), common.MessageIDKey, messageID)
	args := sendTxArgsFromRPCCall(req)
	tx := ep.txQueueManager.CreateTransaction(ctx, args)

	if err := ep.txQueueManager.QueueTransaction(tx); err != nil {
		return nil, err
	}

	if err := ep.txQueueManager.WaitForTransaction(tx); err != nil {
		return nil, err
	}

	// invoke post processing
	postProcessRequest(call.Otto, req, messageID)

	// @TODO(adam): which one is actually used?
	res.Set("result", tx.Hash.Hex())
	res.Set("hash", tx.Hash.Hex())

	return res, nil
}

// ExecuteOtherTransaction defines a function which handles the processing of non `eth_sendTransaction`
// rpc request to the internal node server.
func (ep *ExecutionPolicy) executeOtherTransaction(req common.RPCCall, call otto.FunctionCall) (*otto.Object, error) {
	client, err := ep.nodeManager.RPCClient()
	if err != nil {
		return nil, common.StopRPCCallError{Err: err}
	}

	JSON, err := call.Otto.Object("JSON")
	if err != nil {
		return nil, err
	}

	var result json.RawMessage

	resp, _ := call.Otto.Object(`({"jsonrpc":"2.0"})`)
	resp.Set("id", req.ID)

	// do extra request pre processing (persist message id)
	// within function semaphore will be acquired and released,
	// so that no more than one client (per cell) can enter
	messageID, err := preProcessRequest(call.Otto, req)
	if err != nil {
		return nil, common.StopRPCCallError{Err: err}
	}

	err = client.Call(&result, req.Method, req.Params...)

	switch err := err.(type) {
	case nil:
		if result == nil {

			// Special case null because it is decoded as an empty
			// raw message for some reason.
			resp.Set("result", otto.NullValue())

		} else {

			resultVal, callErr := JSON.Call("parse", string(result))

			if callErr != nil {
				resp = newErrorResponse(call.Otto, -32603, callErr.Error(), &req.ID).Object()
			} else {
				resp.Set("result", resultVal)
			}

		}

	case rpc.Error:

		resp.Set("error", map[string]interface{}{
			"code":    err.ErrorCode(),
			"message": err.Error(),
		})

	default:

		resp = newErrorResponse(call.Otto, -32603, err.Error(), &req.ID).Object()
	}

	// do extra request post processing (setting back tx context)
	postProcessRequest(call.Otto, req, messageID)

	return resp, nil
}

// preProcessRequest pre-processes a given RPC call to a given Otto VM
func preProcessRequest(vm *otto.Otto, req common.RPCCall) (string, error) {
	messageID := currentMessageID(vm.Context())

	return messageID, nil
}

// postProcessRequest post-processes a given RPC call to a given Otto VM
func postProcessRequest(vm *otto.Otto, req common.RPCCall, messageID string) {
	if len(messageID) > 0 {
		vm.Call("addContext", nil, messageID, common.MessageIDKey, messageID) // nolint: errcheck
	}

	// set extra markers for queued transaction requests
	if req.Method == params.SendTransactionMethodName {
		vm.Call("addContext", nil, messageID, params.SendTransactionMethodName, true) // nolint: errcheck
	}
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

func sendTxArgsFromRPCCall(req common.RPCCall) common.SendTxArgs {
	// no need to persist extra state for other requests
	if req.Method != params.SendTransactionMethodName {
		return common.SendTxArgs{}
	}

	var err error
	var fromAddr, toAddr gethcommon.Address

	fromAddr, err = req.ParseFromAddress()
	if err != nil {
		fromAddr = gethcommon.HexToAddress("0x0")
	}

	toAddr, err = req.ParseToAddress()
	if err != nil {
		toAddr = gethcommon.HexToAddress("0x0")
	}

	return common.SendTxArgs{
		To:       &toAddr,
		From:     fromAddr,
		Value:    req.ParseValue(),
		Data:     req.ParseData(),
		Gas:      req.ParseGas(),
		GasPrice: req.ParseGasPrice(),
	}
}
