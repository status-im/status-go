package jail

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/log"
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
	sync.RWMutex
	requestManager *RequestManager
	cells          map[string]common.JailCell // jail supports running many isolated instances of jailed runtime
	baseJSCode     string                     // JavaScript used to initialize all new cells with
}

// New returns new Jail environment
func New(nodeManager common.NodeManager) *Jail {
	return &Jail{
		requestManager: NewRequestManager(nodeManager),
		cells:          make(map[string]common.JailCell),
	}
}

// BaseJS allows to setup initial JavaScript to be loaded on each jail.Parse()
func (jail *Jail) BaseJS(js string) {
	jail.baseJSCode = js
}

// NewJailCell initializes and returns jail cell
func (jail *Jail) NewJailCell(id string) common.JailCell {
	vm := otto.New()

	newJail, err := newJailCell(id, vm, loop.New(vm))
	if err != nil {
		//TODO(alex): Should we really panic here, his there
		// a better way. Think on it.
		panic(err)
	}

	return newJail
}

// Parse creates a new jail cell context, with the given chatID as identifier.
// New context executes provided JavaScript code, right after the initialization.
func (jail *Jail) Parse(chatID string, js string) string {
	var err error
	if jail == nil {
		return makeError(ErrInvalidJail.Error())
	}

	jail.Lock()
	defer jail.Unlock()

	jail.cells[chatID] = jail.NewJailCell(chatID)
	vm := jail.cells[chatID].CellVM()

	initJjs := jail.baseJSCode + ";"
	if _, err = vm.Run(initJjs); err != nil {
		return makeError(err.Error())
	}

	// init jeth and its handlers
	if err = vm.Set("jeth", struct{}{}); err != nil {
		return makeError(err.Error())
	}
	if err = registerHandlers(jail, vm, chatID); err != nil {
		return makeError(err.Error())
	}

	// sendMessage/showSuggestions handlers
	vm.Set("statusSignals", struct{}{})
	statusSignals, _ := vm.Get("statusSignals")
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
	if _, err = vm.Run(jjs); err != nil {
		return makeError(err.Error())
	}

	res, err := vm.Get("catalog")
	if err != nil {
		return makeError(err.Error())
	}

	return makeResult(res.String(), err)
}

// Call executes the `call` function w/i a jail cell context identified by the chatID.
// Jail cell is clonned before call is executed i.e. all calls execute w/i their own contexts.
func (jail *Jail) Call(chatID string, path string, args string) string {
	jail.RLock()
	cell, ok := jail.cells[chatID]
	if !ok {
		jail.RUnlock()
		return makeError(fmt.Sprintf("Cell[%s] doesn't exist.", chatID))
	}
	jail.RUnlock()

	// Due to the new timer assigned we need to clone existing cell to allow
	// unique cell runtime and eventloop context.
	cellCopy, err := cell.Copy()
	if err != nil {
		return makeError(err.Error())
	}

	// isolate VM to allow concurrent access
	vm := cellCopy.CellVM()
	res, err := vm.Call("call", nil, path, args)

	return makeResult(res.String(), err)
}

// JailCellVM returns instance of Otto VM (which is persisted w/i jail cell) by chatID
func (jail *Jail) JailCellVM(chatID string) (*otto.Otto, error) {
	if jail == nil {
		return nil, ErrInvalidJail
	}

	jail.RLock()
	defer jail.RUnlock()

	cell, ok := jail.cells[chatID]
	if !ok {
		return nil, fmt.Errorf("cell[%s] doesn't exist", chatID)
	}

	return cell.CellVM(), nil
}

// GetCell returns instance of jailed runtime
func (jail *Jail) GetCell(chatID string) (common.JailCell, error) {
	if jail == nil {
		return nil, ErrInvalidJail
	}

	jail.RLock()
	defer jail.RUnlock()

	cell, ok := jail.cells[chatID]
	if !ok {
		return nil, fmt.Errorf("cell[%s] doesn't exist", chatID)
	}

	return cell, nil
}

// Send will serialize the first argument, send it to the node and returns the response.
// nolint: errcheck, unparam
func (jail *Jail) Send(call otto.FunctionCall) (response otto.Value) {
	client, err := jail.requestManager.RPCClient()
	if err != nil {
		return newErrorResponse(call.Otto, -32603, err.Error(), nil)
	}

	// Remarshal the request into a Go value.
	JSON, _ := call.Otto.Object("JSON")
	reqVal, err := JSON.Call("stringify", call.Argument(0))
	if err != nil {
		throwJSException(err.Error())
	}
	var (
		rawReq = []byte(reqVal.String())
		reqs   []RPCCall
		batch  bool
	)
	if rawReq[0] == '[' {
		batch = true
		json.Unmarshal(rawReq, &reqs)
	} else {
		batch = false
		reqs = make([]RPCCall, 1)
		json.Unmarshal(rawReq, &reqs[0])
	}

	// Execute the requests.
	resps, _ := call.Otto.Object("new Array()")
	for _, req := range reqs {
		resp, _ := call.Otto.Object(`({"jsonrpc":"2.0"})`)
		resp.Set("id", req.ID)
		var result json.RawMessage

		// execute directly w/o RPC call to node
		if req.Method == SendTransactionRequest {
			txHash, err := jail.requestManager.ProcessSendTransactionRequest(call.Otto, req)
			resp.Set("result", txHash.Hex())
			if err != nil {
				resp = newErrorResponse(call.Otto, -32603, err.Error(), &req.ID).Object()
			}
			resps.Call("push", resp)
			continue
		}

		// do extra request pre processing (persist message id)
		// within function semaphore will be acquired and released,
		// so that no more than one client (per cell) can enter
		messageID, err := jail.requestManager.PreProcessRequest(call.Otto, req)
		if err != nil {
			return newErrorResponse(call.Otto, -32603, err.Error(), nil)
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
		resps.Call("push", resp)

		// do extra request post processing (setting back tx context)
		jail.requestManager.PostProcessRequest(call.Otto, req, messageID)
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

func newErrorResponse(otto *otto.Otto, code int, msg string, id interface{}) otto.Value {
	// Bundle the error into a JSON RPC call response
	m := map[string]interface{}{"jsonrpc": "2.0", "id": id, "error": map[string]interface{}{"code": code, msg: msg}}
	res, _ := json.Marshal(m)
	val, _ := otto.Run("(" + string(res) + ")")
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
