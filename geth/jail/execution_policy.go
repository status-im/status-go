package jail

import (
	"context"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/jail/internal/vm"
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
func (ep *ExecutionPolicy) Execute(req common.RPCCall, vm *vm.VM) (map[string]interface{}, error) {
	if params.SendTransactionMethodName == req.Method {
		return ep.executeSendTransaction(vm, req)
	}

	client := ep.nodeManager.RPCClient()

	return ep.executeWithClient(client, vm, req)
}

// executeRemoteSendTransaction defines a function to execute RPC method eth_sendTransaction over the upstream server.
func (ep *ExecutionPolicy) executeSendTransaction(vm *vm.VM, req common.RPCCall) (map[string]interface{}, error) {
	messageID, err := preProcessRequest(vm)
	if err != nil {
		return nil, err
	}

	// TODO(adam): check if context is used
	ctx := context.WithValue(context.Background(), common.MessageIDKey, messageID)
	args := req.ToSendTxArgs()

	tx := ep.txQueueManager.CreateTransaction(ctx, args)

	if err := ep.txQueueManager.QueueTransaction(tx); err != nil {
		return nil, err
	}

	if err := ep.txQueueManager.WaitForTransaction(tx); err != nil {
		return nil, err
	}

	// invoke post processing
	postProcessRequest(vm, req, messageID)

	res := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      req.ID,
		// @TODO(adam): which one is actually used?
		"result": tx.Hash.Hex(),
		"hash":   tx.Hash.Hex(),
	}

	return res, nil
}

func (ep *ExecutionPolicy) executeWithClient(client *rpc.Client, vm *vm.VM, req common.RPCCall) (map[string]interface{}, error) {
	// Arbitrary JSON-RPC response.
	var result interface{}

	resp := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      req.ID,
	}

	// do extra request pre processing (persist message id)
	// within function semaphore will be acquired and released,
	// so that no more than one client (per cell) can enter
	messageID, err := preProcessRequest(vm)
	if err != nil {
		return nil, common.StopRPCCallError{Err: err}
	}

	if client == nil {
		resp = newErrorResponse("RPC client is not available. Node is stopped?", &req.ID)
	} else {
		err = client.Call(&result, req.Method, req.Params...)
		if err != nil {
			if err2, ok := err.(gethrpc.Error); ok {
				resp["error"] = map[string]interface{}{
					"code":    err2.ErrorCode(),
					"message": err2.Error(),
				}
			} else {
				resp = newErrorResponse(err.Error(), &req.ID)
			}
		}
	}

	if result == nil {
		// Special case null because it is decoded as an empty
		// raw message for some reason.
		resp["result"] = ""
	} else {
		resp["result"] = result
	}

	// do extra request post processing (setting back tx context)
	postProcessRequest(vm, req, messageID)

	return resp, nil
}

// preProcessRequest pre-processes a given RPC call to a given Otto VM
func preProcessRequest(vm *vm.VM) (string, error) {
	messageID := currentMessageID(vm)

	return messageID, nil
}

// postProcessRequest post-processes a given RPC call to a given Otto VM
func postProcessRequest(vm *vm.VM, req common.RPCCall, messageID string) {
	if len(messageID) > 0 {
		vm.Call("addContext", nil, messageID, common.MessageIDKey, messageID) // nolint: errcheck
	}

	// set extra markers for queued transaction requests
	if req.Method == params.SendTransactionMethodName {
		vm.Call("addContext", nil, messageID, params.SendTransactionMethodName, true) // nolint: errcheck
	}
}

// currentMessageID looks for `status.message_id` variable in current JS context
func currentMessageID(vm *vm.VM) string {
	msgID, err := vm.Run("status.message_id")
	if err != nil {
		return ""
	}
	return msgID.String()
}
