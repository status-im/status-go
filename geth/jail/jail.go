package jail

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/eapache/go-resiliency/semaphore"
	"github.com/ethereum/go-ethereum/log"
	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/static"
)

const (
	// JailCellRequestTimeout seconds before jailed request times out
	JailCellRequestTimeout = 60
)

var web3JSCode = static.MustAsset("scripts/web3.js")

// errors
var (
	ErrInvalidJail = errors.New("jail environment is not properly initialized")
)

// JailCell represents single jail cell, which is basically a JavaScript VM.
type JailCell struct {
	id  string
	vm  *otto.Otto
	sem *semaphore.Semaphore
}

// Jail represents jailed environment inside of which we hold multiple cells.
// Each cell is a separate JavaScript VM.
type Jail struct {
	sync.RWMutex
	manager    common.RPCNodeManager
	cells      map[string]common.JailCell // jail supports running many isolated instances of jailed runtime
	baseJSCode string                     // JavaScript used to initialize all new cells with
}

func (cell *JailCell) CellVM() *otto.Otto {
	return cell.vm
}

// New returns new Jail environment
func New(manager common.RPCNodeManager) *Jail {
	return &Jail{
		manager: manager,
		cells:   make(map[string]common.JailCell),
	}
}

// BaseJS allows to setup initial JavaScript to be loaded on each jail.Parse()
func (jail *Jail) BaseJS(js string) {
	jail.baseJSCode = js
}

// NewJailCell initializes and returns jail cell
func (jail *Jail) NewJailCell(id string) common.JailCell {
	return &JailCell{
		id:  id,
		vm:  otto.New(),
		sem: semaphore.New(1, JailCellRequestTimeout*time.Second),
	}
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

// Call executes given JavaScript function w/i a jail cell context identified by the chatID.
// Jail cell is clonned before call is executed i.e. all calls execute w/i their own contexts.
func (jail *Jail) Call(chatID string, path string, args string) string {
	jail.RLock()
	cell, ok := jail.cells[chatID]
	if !ok {
		jail.RUnlock()
		return makeError(fmt.Sprintf("Cell[%s] doesn't exist.", chatID))
	}
	jail.RUnlock()

	vm := cell.CellVM().Copy() // isolate VM to allow concurrent access
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

// Send will serialize the first argument, send it to the node and returns the response.
// nolint: errcheck, unparam
func (jail *Jail) Send(chatID string, call otto.FunctionCall) (response otto.Value) {

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

	JSON, _ := call.Otto.Object("JSON")
	resps, _ := call.Otto.Object("new Array()")

	// Execute the requests.
	for _, req := range reqs {
		res, err := jail.manager.Exec(req, call)
		if err != nil {
			switch err.(type) {
			case common.StopRPCCallError:
				return newErrorResponse(call, -32603, err.Error(), nil)
			default:
				res = newErrorResponse(call, -32603, err.Error(), &req.ID).Object()
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
