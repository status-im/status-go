package transactions

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/les/status"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/common"
)

// contains sets of possible defines the name for a giving transaction.
const (
	SendTransactionName = "eth_sendTransaction"
)

// ExecuteRemoteSendTransaction defines a funct
func ExecuteRemoteSendTransaction(manager common.RPCNodeManager, req common.RPCCall, call otto.FunctionCall) (*otto.Object, error) {
	resp, _ := call.Otto.Object(`({"jsonrpc":"2.0"})`)
	resp.Set("id", req.ID)

	config, err := manager.NodeConfig()
	if err != nil {
		return nil, err
	}

	selectedAcct, err := manager.Account().SelectedAccount()
	if err != nil {
		return nil, err
	}

	client, err := manager.RPCClient()
	if err != nil {
		return nil, err
	}

	fromAddr := req.ParseFromAddress()

	// We need to request a new transaction nounce from upstream node.
	ctx, canceler := context.WithDeadline(context.Background(), time.Now().Add(1*time.Minute))

	defer canceler()

	var num hexutil.Uint
	if err := client.CallContext(ctx, &num, "eth_getBlockTransactionCountByHash", fromAddr.Hash()); err != nil {
		return nil, err
	}

	nonce := uint64(num)
	toAddr := req.ParseToAddress()
	gas := (*big.Int)(req.ParseGas())
	dataVal := []byte(req.ParseData())
	priceVal := (*big.Int)(req.ParseValue())
	gasPrice := (*big.Int)(req.ParseGasPrice())
	chainID := big.NewInt(int64(config.NetworkID))

	tx := types.NewTransaction(nonce, *toAddr, priceVal, gas, gasPrice, dataVal)
	txs, err := types.SignTx(tx, types.NewEIP155Signer(chainID), selectedAcct.AccountKey.PrivateKey)
	if err != nil {
		return nil, err
	}

	// Attempt to get the hex version of the transaction.
	txBytes, err := rlp.EncodeToBytes(txs)
	if err != nil {
		return nil, err
	}

	ctx2, canceler2 := context.WithDeadline(context.Background(), time.Now().Add(1*time.Minute))
	defer canceler2()

	var result json.RawMessage
	if err := client.CallContext(ctx2, &result, "eth_sendRawTransaction", gethcommon.ToHex(txBytes)); err != nil {
		return nil, err
	}

	resp.Set("result", result)
	resp.Set("hash", txs.Hash().String())

	return resp, nil
}

// ExecuteSendTransaction defines a function which handles the procedure called for the dealing with
// RPCCalls with "eth_sendTransaction" Methods.
func ExecuteSendTransaction(manager common.NodeManager, req common.RPCCall, call otto.FunctionCall) (*otto.Object, error) {
	resp, _ := call.Otto.Object(`({"jsonrpc":"2.0"})`)
	resp.Set("id", req.ID)

	txHash, err := processRPCCall(manager, req, call)
	resp.Set("result", txHash.Hex())

	if err != nil {
		resp = newErrorResponse(call, -32603, err.Error(), &req.ID).Object()
		return resp, nil
	}

	return resp, nil
}

func processRPCCall(manager common.NodeManager, req common.RPCCall, call otto.FunctionCall) (gethcommon.Hash, error) {
	lightEthereum, err := manager.LightEthereumService()
	if err != nil {
		return gethcommon.Hash{}, err
	}

	backend := lightEthereum.StatusBackend

	messageID, err := preProcessRequest(call.Otto, req)
	if err != nil {
		return gethcommon.Hash{}, err
	}

	// onSendTransactionRequest() will use context to obtain and release ticket
	ctx := context.Background()
	ctx = context.WithValue(ctx, common.MessageIDKey, messageID)

	//  this call blocks, up until Complete Transaction is called
	txHash, err := backend.SendTransaction(ctx, sendTxArgsFromRPCCall(req))
	if err != nil {
		return gethcommon.Hash{}, err
	}

	// invoke post processing
	postProcessRequest(call.Otto, req, messageID)

	return txHash, nil
}

//==========================================================================================================

// ExecuteOtherTransaction defines a function which handles the processing of non `eth_sendTransaction`
// requests. It is expected that this method does not require signing the transaction.
func ExecuteOtherTransaction(manager common.NodeManager, req common.RPCCall, call otto.FunctionCall) (*otto.Object, error) {
	client, err := manager.RPCClient()
	if err != nil {
		return nil, common.StopRPCCallError{Err: err}
	}

	JSON, _ := call.Otto.Object("JSON")

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

	errc := make(chan error, 1)
	errc2 := make(chan error)

	go func() {
		errc2 <- <-errc
	}()

	errc <- client.Call(&result, req.Method, req.Params...)
	err = <-errc2

	switch err := err.(type) {
	case nil:
		if result == nil {

			// Special case null because it is decoded as an empty
			// raw message for some reason.
			resp.Set("result", otto.NullValue())

		} else {

			resultVal, callErr := JSON.Call("parse", string(result))

			if callErr != nil {
				resp = newErrorResponse(call, -32603, callErr.Error(), &req.ID).Object()
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

		resp = newErrorResponse(call, -32603, err.Error(), &req.ID).Object()
	}

	// do extra request post processing (setting back tx context)
	postProcessRequest(call.Otto, req, messageID)

	return resp, nil
}

//==========================================================================================================

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
	if req.Method == SendTransactionName {
		vm.Call("addContext", nil, messageID, SendTransactionName, true) // nolint: errcheck
	}
}

func sendTxArgsFromRPCCall(req common.RPCCall) status.SendTxArgs {
	if req.Method != SendTransactionName { // no need to persist extra state for other requests
		return status.SendTxArgs{}
	}

	return status.SendTxArgs{
		From:     req.ParseFromAddress(),
		To:       req.ParseToAddress(),
		Value:    req.ParseValue(),
		Data:     req.ParseData(),
		Gas:      req.ParseGas(),
		GasPrice: req.ParseGasPrice(),
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

//==========================================================================================================

func newErrorResponse(call otto.FunctionCall, code int, msg string, id interface{}) otto.Value {
	// Bundle the error into a JSON RPC call response
	m := map[string]interface{}{"jsonrpc": "2.0", "id": id, "error": map[string]interface{}{"code": code, msg: msg}}
	res, _ := json.Marshal(m)
	val, _ := call.Otto.Run("(" + string(res) + ")")
	return val
}

func newResultResponse(call otto.FunctionCall, result interface{}) otto.Value {
	resp, _ := call.Otto.Object(`({"jsonrpc":"2.0"})`)
	resp.Set("result", result) // nolint: errcheck

	return resp.Value()
}

// throwJSException panics on an otto.Value. The Otto VM will recover from the
// Go panic and throw msg as a JavaScript error.
func throwJSException(msg interface{}) otto.Value {
	val, err := otto.ToValue(msg)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to serialize JavaScript exception %v: %v", msg, err))
	}
	panic(val)
}

// JSONError is wrapper around errors, that are sent upwards
type JSONError struct {
	Error string `json:"error"`
}

func makeError(error string) string {
	str := JSONError{
		Error: error,
	}
	outBytes, _ := json.Marshal(&str)
	return string(outBytes)
}

func makeResult(res string, err error) string {
	var out string
	if err != nil {
		out = makeError(err.Error())
	} else {
		if "undefined" == res {
			res = "null"
		}
		out = fmt.Sprintf(`{"result": %s}`, res)
	}

	return out
}

//==========================================================================================================
