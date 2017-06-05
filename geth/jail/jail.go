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
	JailedRuntimeRequestTimeout = time.Second * 60
)

var (
	ErrInvalidJail = errors.New("jail environment is not properly initialized")
)

type Jail struct {
	sync.RWMutex
	client       *rpc.Client               // lazy inited on the first call
	cells        map[string]*JailedRuntime // jail supports running many isolated instances of jailed runtime
	statusJS     string
	requestQueue *geth.JailedRequestQueue
}

type JailedRuntime struct {
	sync.Mutex
	id  string
	vm  *otto.Otto
	sem *semaphore.Semaphore
}

var Web3_JS = static.MustAsset("scripts/web3.js")
var jailInstance *Jail
var once sync.Once

func New() *Jail {
	once.Do(func() {
		jailInstance = &Jail{
			cells: make(map[string]*JailedRuntime),
		}
	})

	return jailInstance
}

func Init(js string) *Jail {
	jailInstance = New() // singleton, we will always get the same reference
	jailInstance.statusJS = js

	return jailInstance
}

func GetInstance() *Jail {
	return New() // singleton, we will always get the same reference
}

func NewJailedRuntime(id string) *JailedRuntime {
	return &JailedRuntime{
		id:  id,
		vm:  otto.New(),
		sem: semaphore.New(1, JailedRuntimeRequestTimeout),
	}
}

func (jail *Jail) Parse(chatId string, js string) string {
	if jail == nil {
		return printError(ErrInvalidJail.Error())
	}

	jail.Lock()
	defer jail.Unlock()

	jail.cells[chatId] = NewJailedRuntime(chatId)
	vm := jail.cells[chatId].vm

	// jeth and its handlers
	vm.Set("jeth", struct{}{})
	jethObj, _ := vm.Get("jeth")
	jethObj.Object().Set("send", makeSendHandler(jail, chatId))
	jethObj.Object().Set("sendAsync", makeSendHandler(jail, chatId))
	jethObj.Object().Set("isConnected", makeJethIsConnectedHandler(jail))

	// localStorage and its handlers
	vm.Set("localStorage", struct{}{})
	localStorage, _ := vm.Get("localStorage")
	localStorage.Object().Set("set", makeLocalStorageSetHandler(chatId))

	// sendMessage/showSuggestions handlers
	vm.Set("statusSignals", struct{}{})
	statusSignals, _ := vm.Get("statusSignals")
	statusSignals.Object().Set("sendMessage", makeSendMessageHandler(chatId))
	statusSignals.Object().Set("showSuggestions", makeShowSuggestionsHandler(chatId))

	initJjs := jail.statusJS + ";"
	_, err := vm.Run(initJjs)

	jjs := string(Web3_JS) + `
	var Web3 = require('web3');
	var web3 = new Web3(jeth);
	var Bignumber = require("bignumber.js");
        function bn(val){
            return new Bignumber(val);
        }
	` + js + "; var catalog = JSON.stringify(_status_catalog);"
	vm.Run(jjs)

	res, _ := vm.Get("catalog")

	return printResult(res.String(), err)
}

func (jail *Jail) Call(chatId string, path string, args string) string {
	_, err := jail.RPCClient()
	if err != nil {
		return printError(err.Error())
	}

	jail.RLock()
	cell, ok := jail.cells[chatId]
	if !ok {
		jail.RUnlock()
		return printError(fmt.Sprintf("Cell[%s] doesn't exist.", chatId))
	}
	jail.RUnlock()

	cell.Lock()
	defer cell.Unlock()
	res, err := cell.vm.Call("call", nil, path, args)

	return printResult(res.String(), err)
}

func (jail *Jail) GetVM(chatId string) (*otto.Otto, error) {
	if jail == nil {
		return nil, ErrInvalidJail
	}

	jail.RLock()
	defer jail.RUnlock()

	cell, ok := jail.cells[chatId]
	if !ok {
		return nil, fmt.Errorf("Cell[%s] doesn't exist.", chatId)
	}

	return cell.vm, nil
}

// GetCell returns instance of jailed runtime
func (jail *Jail) GetCell(chatId string) (*JailedRuntime, error) {
	if jail == nil {
		return nil, ErrInvalidJail
	}

	jail.RLock()
	defer jail.RUnlock()

	cell, ok := jail.cells[chatId]
	if !ok {
		return nil, fmt.Errorf("cell[%s] doesn't exist", chatId)
	}

	return cell, nil
}

// Send will serialize the first argument, send it to the node and returns the response.
// nolint: errcheck, unparam
func (jail *Jail) Send(chatId string, call otto.FunctionCall) (response otto.Value) {
	chatCell, err := jail.GetCell(chatId)
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
		resp.Set("id", req.Id)
		var result json.RawMessage

		// execute directly w/o RPC call to node
		if req.Method == geth.SendTransactionRequest {
			txHash, err := requestQueue.ProcessSendTransactionRequest(chatCell, chatCell.vm, req)
			resp.Set("result", txHash.Hex())
			if err != nil {
				resp = newErrorResponse(chatCell.vm, -32603, err.Error(), &req.Id).Object()
			}
			resps.Call("push", resp)
			continue
		}

		// do extra request pre processing (persist message id)
		// within function semaphore will be acquired and released,
		// so that no more than one client (per cell) can enter
		messageId, err := requestQueue.PreProcessRequest(chatCell.vm, req)
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
					resp = newErrorResponse(chatCell.vm, -32603, callErr.Error(), &req.Id).Object()
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
			resp = newErrorResponse(chatCell.vm, -32603, err.Error(), &req.Id).Object()
		}
		resps.Call("push", resp)

		// do extra request post processing (setting back tx context)
		requestQueue.PostProcessRequest(chatCell.vm, req, messageId)
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
	jail.client = client

	return jail.client, nil
}

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
