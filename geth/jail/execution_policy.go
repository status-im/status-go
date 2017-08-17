package jail

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/common"
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

// ExecuteSendTransaction defines a function to execute RPC requests for eth_sendTransaction method only.
func (ep *ExecutionPolicy) ExecuteSendTransaction(req common.RPCCall, call otto.FunctionCall) (*otto.Object, error) {
	config, err := ep.nodeManager.NodeConfig()
	if err != nil {
		return nil, err
	}

	if config.UpstreamConfig.Enabled {
		return ep.executeRemoteSendTransaction(req, call)
	}

	return ep.executeLocalSendTransaction(req, call)
}

// executeRemoteSendTransaction defines a function to execute RPC method eth_sendTransaction over the upstream server.
func (ep *ExecutionPolicy) executeRemoteSendTransaction(req common.RPCCall, call otto.FunctionCall) (*otto.Object, error) {
	config, err := ep.nodeManager.NodeConfig()
	if err != nil {
		return nil, err
	}

	selectedAcct, err := ep.accountManager.SelectedAccount()
	if err != nil {
		return nil, err
	}

	client, err := ep.nodeManager.RPCClient()
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

// executeLocalSendTransaction defines a function which handles execution of RPC method over the internal rpc server
// from the eth.LightClient. It specifically caters to process eth_sendTransaction.
func (ep *ExecutionPolicy) executeLocalSendTransaction(req common.RPCCall, call otto.FunctionCall) (*otto.Object, error) {
	les, err := ep.nodeManager.LightEthereumService()
	if err != nil {
		return nil, err
	}

	resp, err := call.Otto.Object(`({"jsonrpc":"2.0"})`)
	if err != nil {
		return nil, err
	}
	resp.Set("id", req.ID)

	messageID, err := preProcessRequest(call.Otto, req)
	if err != nil {
		resp = newErrorResponse(call.Otto, -32603, err.Error(), &req.ID).Object()
		return resp, nil
	}

	// onSendTransactionRequest() will use context to obtain and release ticket
	ctx := context.Background()
	ctx = context.WithValue(ctx, common.MessageIDKey, messageID)

	// Marshal args to JSON string.
	rawArgs, err := json.Marshal(sendTxArgsFromRPCCall(req))
	if err != nil {
		err = fmt.Errorf("failed to marshal args: %s", err)
		resp = newErrorResponse(call.Otto, -32603, err.Error(), &req.ID).Object()
		return resp, nil
	}

	//  this call blocks, up until Complete Transaction is called
	txHash, err := les.StatusBackend.SendTransaction(ctx, rawArgs, "")
	if err != nil {
		resp = newErrorResponse(call.Otto, -32603, err.Error(), &req.ID).Object()
		return resp, nil
	}

	resp.Set("result", txHash.Hex())

	// invoke post processing
	postProcessRequest(call.Otto, req, messageID)

	return resp, nil
}

// ExecuteOtherTransaction defines a function which handles the processing of non `eth_sendTransaction`
// rpc request to the internal node server.
func (ep *ExecutionPolicy) ExecuteOtherTransaction(req common.RPCCall, call otto.FunctionCall) (*otto.Object, error) {
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
