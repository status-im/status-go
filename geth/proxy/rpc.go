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
	accountManager common.AccountManager
	rpclient       *rpc.Client
}

// NewRPCRouter returns a new instance of a RPCRouter.
func NewRPCRouter(manager common.NodeManager, acct common.AccountManager) *RPCRouter {
	router := &RPCRouter{
		NodeManager:    manager,
		accountManager: acct,
	}

	return router
}

// Account returns the associated common.AccountManager suited for the
// giving rpc router.
func (rp *RPCRouter) Account() common.AccountManager {
	return rp.accountManager
}

// Exec takes the giving RPCCall and caller to be executed against the appropriate caller.
// To accommodate the
func (rp *RPCRouter) Exec(req common.RPCCall, caller otto.FunctionCall) (*otto.Object, error) {
	config, err := rp.NodeManager.NodeConfig()
	if err != nil {
		return nil, err
	}

	switch req.Method {
	case transactions.SendTransactionName:
		if config.UpstreamConfig.Enabled {
			return transactions.ExecuteRemoteSendTransaction(rp, req, caller)
		}

		return transactions.ExecuteSendTransaction(rp.NodeManager, req, caller)
	default:
		return transactions.ExecuteOtherTransaction(rp, req, caller)
	}
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
