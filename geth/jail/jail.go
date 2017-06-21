package jail

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/eapache/go-resiliency/semaphore"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth"
	"github.com/status-im/status-go/static"
)

const (
	// JailedRuntimeRequestTimeout seconds before jailed request times out
	JailedRuntimeRequestTimeout = time.Second * 60
)

// errors
var (
	ErrInvalidJail = errors.New("jail environment is not properly initialized")
)

// Jail represents jailed environment inside of which we hold
// multiple cells. Each cell is separate JavaScript VM.
type Jail struct {
	sync.RWMutex
	client       *rpc.Client               // lazy inited on the first call
	cells        map[string]*JailedRuntime // jail supports running many isolated instances of jailed runtime
	statusJS     string
	requestQueue *geth.JailedRequestQueue
}

// JailedRuntime represents single jail cell, which is JavaScript VM.
type JailedRuntime struct {
	sync.Mutex
	id  string
	vm  *otto.Otto
	sem *semaphore.Semaphore
}

var web3JS = static.MustAsset("scripts/web3.js")
var jailInstance *Jail
var once sync.Once

// New returns singleton jail environment
func New() *Jail {
	once.Do(func() {
		jailInstance = &Jail{
			cells: make(map[string]*JailedRuntime),
		}
	})

	return jailInstance
}

// Init allows to setup initial JavaScript to be loaded on each jail.Parse()
func Init(js string) *Jail {
	jailInstance = New() // singleton, we will always get the same reference
	jailInstance.statusJS = js

	return jailInstance
}

// GetInstance returns singleton jail environment instance
func GetInstance() *Jail {
	return New() // singleton, we will always get the same reference
}

// NewJailedRuntime initializes and returns jail cell
func NewJailedRuntime(id string) *JailedRuntime {
	return &JailedRuntime{
		id:  id,
		vm:  otto.New(),
		sem: semaphore.New(1, JailedRuntimeRequestTimeout),
	}
}

// Parse creates a new jail cell context, with the given chatID as identifier
// New context executes provided JavaScript code, right after the initialization.
func (jail *Jail) Parse(chatID string, js string) string {
	var err error
	if jail == nil {
		return printError(ErrInvalidJail.Error())
	}

	jail.Lock()
	defer jail.Unlock()

	jail.cells[chatID] = NewJailedRuntime(chatID)
	vm := jail.cells[chatID].vm

	// init jeth and its handlers
	if err = vm.Set("jeth", struct{}{}); err != nil {
		return printError(err.Error())
	}
	if err = registerHandlers(jail, vm, chatID); err != nil {
		return printError(err.Error())
	}

	initJjs := jail.statusJS + ";"
	if _, err = vm.Run(initJjs); err != nil {
		return printError(err.Error())
	}

	jjs := string(web3JS) + `
	var Web3 = require('web3');
	var web3 = new Web3(jeth);
	var Bignumber = require("bignumber.js");
        function bn(val){
            return new Bignumber(val);
        }
	` + js + "; var catalog = JSON.stringify(_status_catalog);"
	if _, err = vm.Run(jjs); err != nil {
		return printError(err.Error())
	}

	res, err := vm.Get("catalog")
	if err != nil {
		return printError(err.Error())
	}

	return printResult(res.String(), err)
}

// Call executes given JavaScript function w/i a jail cell context identified by the chatID
func (jail *Jail) Call(chatID string, path string, args string) string {
	_, err := jail.RPCClient()
	if err != nil {
		return printError(err.Error())
	}

	jail.RLock()
	cell, ok := jail.cells[chatID]
	if !ok {
		jail.RUnlock()
		return printError(fmt.Sprintf("Cell[%s] doesn't exist.", chatID))
	}
	jail.RUnlock()

	cell.Lock()
	defer cell.Unlock()
	res, err := cell.vm.Call("call", nil, path, args)

	return printResult(res.String(), err)
}

// GetVM returns instance of Otto VM (which is persisted w/i jail cell) by chatID
func (jail *Jail) GetVM(chatID string) (*otto.Otto, error) {
	if jail == nil {
		return nil, ErrInvalidJail
	}

	jail.RLock()
	defer jail.RUnlock()

	cell, ok := jail.cells[chatID]
	if !ok {
		return nil, fmt.Errorf("cell[%s] doesn't exist", chatID)
	}

	return cell.vm, nil
}

// GetCell returns instance of jailed runtime
func (jail *Jail) GetCell(chatID string) (*JailedRuntime, error) {
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
func (jail *Jail) Send(chatID string, call otto.FunctionCall) (response otto.Value) {
	chatCell, err := jail.GetCell(chatID)
	if err != nil {
		return newErrorResponse(chatCell.vm, -32603, err.Error(), nil)
	}
	client, err := jail.RPCClient()
	if err != nil {
		return newErrorResponse(chatCell.vm, -32603, err.Error(), nil)
	}

	requestQueue, err := jail.RequestQueue()
	if err != nil {
		return newErrorResponse(chatCell.vm, -32603, err.Error(), nil)
	}

	// Remarshal the request into a Go value.
	JSON, _ := chatCell.vm.Object("JSON")
	reqVal, err := JSON.Call("stringify", call.Argument(0))
	if err != nil {
		throwJSException(err.Error())
	}
	var (
		rawReq = []byte(reqVal.String())
		reqs   []geth.RPCCall
		batch  bool
	)
	if rawReq[0] == '[' {
		batch = true
		json.Unmarshal(rawReq, &reqs)
	} else {
		batch = false
		reqs = make([]geth.RPCCall, 1)
		json.Unmarshal(rawReq, &reqs[0])
	}

	// Execute the requests.
	resps, _ := chatCell.vm.Object("new Array()")
	for _, req := range reqs {
		resp, _ := chatCell.vm.Object(`({"jsonrpc":"2.0"})`)
		resp.Set("id", req.ID)
		var result json.RawMessage

		// execute directly w/o RPC call to node
		if req.Method == geth.SendTransactionRequest {
			txHash, err := requestQueue.ProcessSendTransactionRequest(chatCell, chatCell.vm, req)
			resp.Set("result", txHash.Hex())
			if err != nil {
				resp = newErrorResponse(chatCell.vm, -32603, err.Error(), &req.ID).Object()
			}
			resps.Call("push", resp)
			continue
		}

		// do extra request pre processing (persist message id)
		// within function semaphore will be acquired and released,
		// so that no more than one client (per cell) can enter
		messageID, err := requestQueue.PreProcessRequest(chatCell.vm, req)
		if err != nil {
			return newErrorResponse(chatCell.vm, -32603, err.Error(), nil)
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
					resp = newErrorResponse(chatCell.vm, -32603, callErr.Error(), &req.ID).Object()
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
			resp = newErrorResponse(chatCell.vm, -32603, err.Error(), &req.ID).Object()
		}
		resps.Call("push", resp)

		// do extra request post processing (setting back tx context)
		requestQueue.PostProcessRequest(chatCell.vm, req, messageID)
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

// RPCClient returns RPC client instance, creating it if necessary.
// Returned instance is cached, so successive calls receive the same one.
// nolint: dupl
func (jail *Jail) RPCClient() (*rpc.Client, error) {
	if jail == nil {
		return nil, ErrInvalidJail
	}

	if jail.client != nil {
		return jail.client, nil
	}

	nodeManager := geth.NodeManagerInstance()
	if !nodeManager.NodeInited() {
		return nil, geth.ErrInvalidGethNode
	}

	// obtain RPC client from running node
	client, err := nodeManager.RPCClient()
	if err != nil {
		return nil, err
	}

	jail.Lock()
	jail.client = client
	jail.Unlock()

	return jail.client, nil
}

// RequestQueue returns request queue instance, creating it if necessary.
// Returned instance is cached, so successive calls receive the same one.
// nolint: dupl
func (jail *Jail) RequestQueue() (*geth.JailedRequestQueue, error) {
	if jail == nil {
		return nil, ErrInvalidJail
	}

	if jail.requestQueue != nil {
		return jail.requestQueue, nil
	}

	nodeManager := geth.NodeManagerInstance()
	if !nodeManager.NodeInited() {
		return nil, geth.ErrInvalidGethNode
	}

	requestQueue, err := nodeManager.JailedRequestQueue()
	if err != nil {
		return nil, err
	}
	jail.requestQueue = requestQueue

	return jail.requestQueue, nil
}

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

func printError(error string) string {
	str := geth.JSONError{
		Error: error,
	}
	outBytes, _ := json.Marshal(&str)
	return string(outBytes)
}

func printResult(res string, err error) string {
	var out string
	if err != nil {
		out = printError(err.Error())
	} else {
		if "undefined" == res {
			res = "null"
		}
		out = fmt.Sprintf(`{"result": %s}`, res)
	}

	return out
}
