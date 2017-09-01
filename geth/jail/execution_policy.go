package jail

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
)

// map of command routes
var (
	//TODO(influx6): Replace this with a registry of commands to functions that
	// call appropriate op for command with ExecutionPolicy.
	rpcLocalCommandRoute = map[string]bool{
		//Whisper commands
		"shh_post":             true,
		"shh_version":          true,
		"shh_newIdentity":      true,
		"shh_hasIdentity":      true,
		"shh_newGroup":         true,
		"shh_addToGroup":       true,
		"shh_newFilter":        true,
		"shh_uninstallFilter":  true,
		"shh_getFilterChanges": true,
		"shh_getMessages":      true,

		// DB commands
		"db_putString": true,
		"db_getString": true,
		"db_putHex":    true,
		"db_getHex":    true,

		// Other commands
		"net_version":   true,
		"net_peerCount": true,
		"net_listening": true,

		// blockchain commands
		"eth_sign":            true,
		"eth_accounts":        true,
		"eth_getCompilers":    true,
		"eth_compileLLL":      true,
		"eth_compileSolidity": true,
		"eth_compileSerpent":  true,
	}
)

// ExecutionPolicy provides a central container for the executions of RPCCall requests for both
// remote/upstream processing and internal node processing.
type ExecutionPolicy struct {
	nodeManager    common.NodeManager
	accountManager common.AccountManager
}

// NewExecutionPolicy returns a new instance of ExecutionPolicy.
func NewExecutionPolicy(nodeManager common.NodeManager, accountManager common.AccountManager) *ExecutionPolicy {
	return &ExecutionPolicy{
		nodeManager:    nodeManager,
		accountManager: accountManager,
	}
}

// Execute handles the execution of a RPC request
func (ep *ExecutionPolicy) Execute(req common.RPCCall, call otto.FunctionCall) (*otto.Object, error) {
	config, err := ep.nodeManager.NodeConfig()
	if err != nil {
		return nil, err
	}

	if config.UpstreamConfig.Enabled {
		if params.SendTransactionMethodName == req.Method {
			return ep.ExecuteRemoteSendTransaction(req, call)
		}

		if !rpcLocalCommandRoute[req.Method] {
			return ep.ExecuteOnRemote(req, call)
		}

		return ep.ExecuteLocally(req, call)
	}

	if params.SendTransactionMethodName == req.Method {
		return ep.ExecuteLocalSendTransaction(req, call)
	}

	return ep.ExecuteLocally(req, call)
}

// ExecuteLocally defines a function which handles the processing of `shh_*` transaction methods
// rpc request to the internal node server.
func (ep *ExecutionPolicy) ExecuteLocally(req common.RPCCall, call otto.FunctionCall) (*otto.Object, error) {
	client, err := ep.nodeManager.RPCLocalClient()
	if err != nil {
		return nil, common.StopRPCCallError{Err: err}
	}

	return ep.executeWithClient(client, req, call)
}

// ExecuteOnRemote defines a function which handles the processing of non `eth_sendTransaction`
// rpc request to the upstream node server.
func (ep *ExecutionPolicy) ExecuteOnRemote(req common.RPCCall, call otto.FunctionCall) (*otto.Object, error) {
	client, err := ep.nodeManager.RPCUpstreamClient()
	if err != nil {
		return nil, common.StopRPCCallError{Err: err}
	}

	return ep.executeWithClient(client, req, call)
}

// ExecuteRemoteSendTransaction defines a function to execute RPC method eth_sendTransaction over the upstream server.
func (ep *ExecutionPolicy) ExecuteRemoteSendTransaction(req common.RPCCall, call otto.FunctionCall) (*otto.Object, error) {
	config, err := ep.nodeManager.NodeConfig()
	if err != nil {
		return nil, err
	}

	selectedAcct, err := ep.accountManager.SelectedAccount()
	if err != nil {
		return nil, err
	}

	client, err := ep.nodeManager.RPCUpstreamClient()
	if err != nil {
		return nil, err
	}

	fromAddr, err := req.ParseFromAddress()
	if err != nil {
		return nil, err
	}

	toAddr, err := req.ParseToAddress()
	if err != nil {
		return nil, err
	}

	// We need to request a new transaction nounce from upstream node.
	ctx, canceller := context.WithTimeout(context.Background(), time.Minute)
	defer canceller()

	var num hexutil.Uint
	if err := client.CallContext(ctx, &num, "eth_getTransactionCount", fromAddr, "pending"); err != nil {
		return nil, err
	}

	nonce := uint64(num)
	gas := (*big.Int)(req.ParseGas())
	dataVal := []byte(req.ParseData())
	priceVal := (*big.Int)(req.ParseValue())
	gasPrice := (*big.Int)(req.ParseGasPrice())
	chainID := big.NewInt(int64(config.NetworkID))

	tx := types.NewTransaction(nonce, toAddr, priceVal, gas, gasPrice, dataVal)
	txs, err := types.SignTx(tx, types.NewEIP155Signer(chainID), selectedAcct.AccountKey.PrivateKey)
	if err != nil {
		return nil, err
	}

	// Attempt to get the hex version of the transaction.
	txBytes, err := rlp.EncodeToBytes(txs)
	if err != nil {
		return nil, err
	}

	//TODO(influx6): Should we use a single context with a higher timeout, say 3-5 minutes
	// for calls to rpcClient?
	ctx2, canceler2 := context.WithTimeout(context.Background(), time.Minute)
	defer canceler2()

	var result json.RawMessage
	if err := client.CallContext(ctx2, &result, "eth_sendRawTransaction", gethcommon.ToHex(txBytes)); err != nil {
		return nil, err
	}

	resp, err := call.Otto.Object(`({"jsonrpc":"2.0"})`)
	if err != nil {
		return nil, err
	}

	resp.Set("id", req.ID)
	resp.Set("result", result)
	resp.Set("hash", txs.Hash().String())

	return resp, nil
}

// ExecuteLocalSendTransaction defines a function which handles execution of RPC method over the internal rpc server
// from the eth.LightClient. It specifically caters to process eth_sendTransaction.
func (ep *ExecutionPolicy) ExecuteLocalSendTransaction(req common.RPCCall, call otto.FunctionCall) (*otto.Object, error) {
	resp, err := call.Otto.Object(`({"jsonrpc":"2.0"})`)
	if err != nil {
		return nil, err
	}

	resp.Set("id", req.ID)

	txHash, err := processRPCCall(ep.nodeManager, req, call)
	resp.Set("result", txHash.Hex())

	if err != nil {
		resp = newErrorResponse(call.Otto, -32603, err.Error(), &req.ID).Object()
		return resp, nil
	}

	return resp, nil
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
