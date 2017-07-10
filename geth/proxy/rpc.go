package proxy

import (
	"fmt"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/proxy/transactions"
)

//======================================================================================================

// contains all possible method name which will be
const (
	EthSendTransaction = "eth_sendTransaction"
)

//======================================================================================================

// RPCExecutor defines a function type which takes a giving RPCCall to be executed
// and returned a response.
type RPCExecutor func(common.NodeManager, common.RPCCall, otto.FunctionCall) (*otto.Object, error)

// RPCRouter defines a top-level router which sits inbetween calls from other
// to a external service or a running etherem service.
type RPCRouter struct {
	common.NodeManager
	defaultExecutor  RPCExecutor
	nonsignExecutors map[string]RPCExecutor
	signExecutors    map[string]RPCExecutor
}

// NewRPCRouter returns a new instance of a RPCRouter.
func NewRPCRouter(manager common.NodeManager) *RPCRouter {
	router := &RPCRouter{
		NodeManager:      manager,
		defaultExecutor:  transactions.ExecuteOtherTransaction,
		nonsignExecutors: make(map[string]RPCExecutor),
		signExecutors:    make(map[string]RPCExecutor),
	}

	router.RegisterExecutor(EthSendTransaction, transactions.ExecuteSendTransaction, true)

	return router
}

// RegisterExecutor adds the executor into the router map associated with the
// giving method name.
func (rp *RPCRouter) RegisterExecutor(methodName string, executor RPCExecutor, requiresSigning bool) {
	if requiresSigning {
		rp.signExecutors[methodName] = executor
		return
	}

	rp.nonsignExecutors[methodName] = executor
}

// Exec takes the giving RPCCall and caller to be executed against the appropriate caller.
// To accommodate the
func (rp *RPCRouter) Exec(req common.RPCCall, caller otto.FunctionCall) (*otto.Object, error) {
	if executor, ok := rp.signExecutors[req.Method]; ok {
		return executor(rp.NodeManager, req, caller)
	}

	if executor, ok := rp.nonsignExecutors[req.Method]; ok {
		return executor(rp, req, caller)
	}

	if rp.defaultExecutor != nil {
		return rp.defaultExecutor(rp.NodeManager, req, caller)
	}

	return nil, fmt.Errorf("RPC Method %q not supported", req.Method)
}

// RPCClient returns a client associated with the specific RPC server
// which will either be the associated NodeManager or a upstream system.
func (rp *RPCRouter) RPCClient() (*rpc.Client, error) {
	//TODO(alex): Figure out how we can return a rpc client that connects to an upstream
	// instead of the normal node based rpc client.

	return rp.NodeManager.RPCClient()
}

//======================================================================================================
