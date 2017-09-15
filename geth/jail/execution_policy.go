package jail

import (
	"context"
	"encoding/json"

	gethcommon "github.com/ethereum/go-ethereum/common"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/rpc"
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

// Execute handles the execution of a RPC request and routes appropriately to either a local or remote ethereum node.
func (ep *ExecutionPolicy) Execute(req common.RPCCall, call otto.FunctionCall) (*otto.Object, error) {
	if params.SendTransactionMethodName == req.Method {
		return ep.executeSendTransaction(req, call)
	}

	client := ep.nodeManager.RPCClient()

	return ep.executeWithClient(client, req, call)
}

// executeRemoteSendTransaction defines a function to execute RPC method eth_sendTransaction over the upstream server.
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

func (ep *ExecutionPolicy) executeWithClient(client *rpc.Client, req common.RPCCall, call otto.FunctionCall) (*otto.Object, error) {
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

	if client == nil {
		resp = newErrorResponse(call.Otto, -32603, "RPC client is not available. Node is stopped?", &req.ID).Object()
	} else {
		err = client.Call(&result, req.Method, req.Params...)
		if err != nil {
			if err2, ok := err.(gethrpc.Error); ok {
				resp.Set("error", map[string]interface{}{
					"code":    err2.ErrorCode(),
					"message": err2.Error(),
				})
			} else {
				resp = newErrorResponse(call.Otto, -32603, err.Error(), &req.ID).Object()
			}
		}
	}

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
