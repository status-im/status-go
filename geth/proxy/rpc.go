package proxy

import (
	"errors"

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
	rpclient         *rpc.Client
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

	// if rp.defaultExecutor != nil {
	// return rp.defaultExecutor(rp.NodeManager, req, caller)
	// }

	// return nil, fmt.Errorf("RPC Method %q not supported", req.Method)

	return rp.defaultExecutor(rp.NodeManager, req, caller)
}

// RPCClient returns a client associated with the specific RPC server
// which will either be the associated NodeManager or a upstream system.
func (rp *RPCRouter) RPCClient() (*rpc.Client, error) {
	if rp.NodeManager == nil {
		return nil, errors.New("Node Manager is not initialized")
	}

	//TODO(alex): Should we check if NodeManager is started here as well?
	// if rp.NodeManager.IsNodeRunning(){
	// 	return nil, errors.New("NodeManager.Node is not started yet")
	// }

	if rp.rpclient != nil {
		return rp.rpclient, nil
	}

	config, err := rp.NodeManager.NodeConfig()
	if err != nil {
		return nil, err
	}

	// If we have no UpstreamRPCConfig set then just return normal RPClient from
	// embedded NodeManager.
	if config.UpstreamConfig == nil {
		return rp.NodeManager.RPCClient()
	}

	// If we have UpstreamRPCConfig but it's not enabled then return embedded NodeManager's
	// rpc.Client.
	if !config.UpstreamConfig.Enabled {
		return rp.NodeManager.RPCClient()
	}

	// Connect to upstream RPC server with new client and cache instance.
	rp.rpclient, err = rpc.Dial(config.UpstreamConfig.URL)
	if err != nil {
		return nil, err
	}

	return rp.rpclient, nil
}

//======================================================================================================
