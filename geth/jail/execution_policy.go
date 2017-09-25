package jail

import (
	"context"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/jail/internal/vm"
)

// ExecutionPolicy provides a central container for the executions of RPCCall requests for both
// remote/upstream processing and internal node processing.
type ExecutionPolicy struct {
	nodeManager common.NodeManager
}

// NewExecutionPolicy returns a new instance of ExecutionPolicy.
func NewExecutionPolicy(
	nodeManager common.NodeManager,
) *ExecutionPolicy {
	return &ExecutionPolicy{
		nodeManager: nodeManager,
	}
}

// Execute handles the execution of a RPC request and routes appropriately to either a local or remote ethereum node.
func (ep *ExecutionPolicy) Execute(req common.RPCCall, vm *vm.VM) (map[string]interface{}, error) {
	// Arbitrary JSON-RPC response.
	var result interface{}

	resp := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      req.ID,
	}

	client := ep.nodeManager.RPCClient()

	err := client.CallContext(context.Background(), &result, req.Method, req.Params...)
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

	if result == nil {
		// Special case null because it is decoded as an empty
		// raw message for some reason.
		resp["result"] = ""
	} else {
		resp["result"] = result
	}

	return resp, nil
}
