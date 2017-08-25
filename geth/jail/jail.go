package jail

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/les/status"
	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/static"

	"fknsrs.biz/p/ottoext/loop"
)

// FIXME(tiabc): Get rid of this global variable. Move it to a constructor or initialization.
var web3JSCode = static.MustAsset("scripts/web3.js")

// errors
var (
	ErrInvalidJail = errors.New("jail environment is not properly initialized")
)

// Jail represents jailed environment inside of which we hold multiple cells.
// Each cell is a separate JavaScript VM.
type Jail struct {
	// FIXME(tiabc): This mutex handles cells field access and must be renamed appropriately: cellsMutex
	sync.RWMutex
	nodeManager    common.NodeManager
	accountManager common.AccountManager
	policy         *ExecutionPolicy
	cells          map[string]*JailCell // jail supports running many isolated instances of jailed runtime
	baseJSCode     string               // JavaScript used to initialize all new cells with
}

// New returns new Jail environment with the associated NodeManager and
// AccountManager.
func New(nodeManager common.NodeManager, accountManager common.AccountManager) *Jail {
	return &Jail{
		nodeManager:    nodeManager,
		accountManager: accountManager,
		cells:          make(map[string]*JailCell),
		policy:         NewExecutionPolicy(nodeManager, accountManager),
	}
}

// BaseJS allows to setup initial JavaScript to be loaded on each jail.Parse().
func (jail *Jail) BaseJS(js string) {
	jail.baseJSCode = js
}

// NewJailCell initializes and returns jail cell.
func (jail *Jail) NewJailCell(id string) (common.JailCell, error) {
	if jail == nil {
		return nil, ErrInvalidJail
	}

	vm := otto.New()

	newJail, err := newJailCell(id, vm, loop.New(vm))
	if err != nil {
		return nil, err
	}

	jail.Lock()
	jail.cells[id] = newJail
	jail.Unlock()

	return newJail, nil
}

// GetJailCell returns the associated *JailCell for the provided chatID.
func (jail *Jail) GetJailCell(chatID string) (common.JailCell, error) {
	return jail.GetCell(chatID)
}

// GetCell returns the associated *JailCell for the provided chatID.
func (jail *Jail) GetCell(chatID string) (*JailCell, error) {
	jail.RLock()
	defer jail.RUnlock()

	cell, ok := jail.cells[chatID]
	if !ok {
		return nil, fmt.Errorf("cell[%s] doesn't exist", chatID)
	}

	return cell, nil
}

// Parse creates a new jail cell context, with the given chatID as identifier.
// New context executes provided JavaScript code, right after the initialization.
func (jail *Jail) Parse(chatID string, js string) string {
	if jail == nil {
		return makeError(ErrInvalidJail.Error())
	}

	var err error
	var jcell *JailCell

	if jcell, err = jail.GetCell(chatID); err != nil {
		if _, mkerr := jail.NewJailCell(chatID); mkerr != nil {
			return makeError(mkerr.Error())
		}

		jcell, _ = jail.GetCell(chatID)
	}

	// init jeth and its handlers
	if err = jcell.Set("jeth", struct{}{}); err != nil {
		return makeError(err.Error())
	}

	if err = registerHandlers(jail, jcell, chatID); err != nil {
		return makeError(err.Error())
	}

	initJs := jail.baseJSCode + ";"
	if _, err = jcell.Run(initJs); err != nil {
		return makeError(err.Error())
	}

	// sendMessage/showSuggestions handlers
	jcell.Set("statusSignals", struct{}{})
	statusSignals, _ := jcell.Get("statusSignals")
	statusSignals.Object().Set("sendMessage", makeSendMessageHandler(chatID))
	statusSignals.Object().Set("showSuggestions", makeShowSuggestionsHandler(chatID))

	jjs := string(web3JSCode) + `
	var Web3 = require('web3');
	var web3 = new Web3(jeth);
	var Bignumber = require("bignumber.js");
        function bn(val){
            return new Bignumber(val);
        }
	` + js + "; var catalog = JSON.stringify(_status_catalog);"
	if _, err = jcell.Run(jjs); err != nil {
		return makeError(err.Error())
	}

	res, err := jcell.Get("catalog")
	if err != nil {
		return makeError(err.Error())
	}

	return makeResult(res.String(), err)
}

// Call executes the `call` function w/i a jail cell context identified by the chatID.
// Jail cell is clonned before call is executed i.e. all calls execute w/i their own contexts.
func (jail *Jail) Call(chatID string, path string, args string) string {
	jcell, err := jail.GetCell(chatID)
	if err != nil {
		return makeError(err.Error())
	}

	res, err := jcell.Call("call", nil, path, args)

	// WARNING(influx6): We can have go-routine leakage due to continous call to this method
	// and the call to cell.CellLoop().Run() due to improper usage, let's keep this
	// in sight if things ever go wrong here.
	// Due to the new event loop provided by ottoext.
	// We need to ensure that all possible calls to internal setIntervals/SetTimeouts/SetImmediate
	// work by lunching the loop.Run() method.
	// Needs to be done in a go-routine.
	go jcell.lo.Run()

	return makeResult(res.String(), err)
}

// Send will serialize the first argument, send it to the node and returns the response.
// nolint: errcheck, unparam
func (jail *Jail) Send(call otto.FunctionCall) (response otto.Value) {
	// Remarshal the request into a Go value.
	JSON, _ := call.Otto.Object("JSON")
	reqVal, err := JSON.Call("stringify", call.Argument(0))
	if err != nil {
		throwJSException(err.Error())
	}

	var (
		rawReq = []byte(reqVal.String())
		reqs   []common.RPCCall
		batch  bool
	)

	if rawReq[0] == '[' {
		batch = true
		json.Unmarshal(rawReq, &reqs)
	} else {
		batch = false
		reqs = make([]common.RPCCall, 1)
		json.Unmarshal(rawReq, &reqs[0])
	}

	resps, _ := call.Otto.Object("new Array()")

	// Execute the requests.
	for _, req := range reqs {

		res, resErr := jail.policy.Execute(req, call)
		if resErr != nil {
			switch resErr.(type) {
			case common.StopRPCCallError:
				return newErrorResponse(call.Otto, -32603, err.Error(), nil)
			default:
				res = newErrorResponse(call.Otto, -32603, err.Error(), &req.ID).Object()
			}
		}

		resps.Call("push", res)
	}

	// Return the responses either to the callback (if supplied)
	// or directly as the return value.
	if batch {
		response = resps.Value()
	} else {
		response, _ = resps.Get("0")
	}

	if fn := call.Argument(1); fn.Class() == "Function" {
		fn.Call(otto.NullValue(), otto.NullValue(), response)
		return otto.UndefinedValue()
	}

	return response
}

//==================================================================================================================================

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

func sendTxArgsFromRPCCall(req common.RPCCall) status.SendTxArgs {
	if req.Method != params.SendTransactionMethodName { // no need to persist extra state for other requests
		return status.SendTxArgs{}
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

	return status.SendTxArgs{
		To:       &toAddr,
		From:     fromAddr,
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

func newErrorResponse(vm *otto.Otto, code int, msg string, id interface{}) otto.Value {
	// Bundle the error into a JSON RPC call response
	m := map[string]interface{}{"jsonrpc": "2.0", "id": id, "error": map[string]interface{}{"code": code, msg: msg}}
	res, _ := json.Marshal(m)
	val, _ := vm.Run("(" + string(res) + ")")
	return val
}

func newResultResponse(vm *otto.Otto, result interface{}) otto.Value {
	resp, _ := vm.Object(`({"jsonrpc":"2.0"})`)
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
