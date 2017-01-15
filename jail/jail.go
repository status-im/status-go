package jail

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/eapache/go-resiliency/semaphore"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth"
)

const (
	JailedRuntimeRequestTimeout = 60
)

var (
	ErrInvalidJail    = errors.New("jail environment is not properly initialized")
	ErrRequestTimeout = errors.New("long running job timed out")
)

type JailConfig struct {
	LongRunningJobTimeout time.Duration
}

type Jail struct {
	sync.RWMutex
	Config       *JailConfig
	client       *rpc.Client               // lazy inited on the first call
	cells        map[string]*JailedRuntime // jail supports running many isolated instances of jailed runtime
	statusJS     string
	requestQueue *geth.JailedRequestQueue
}

type JailedRuntime struct {
	id  string
	vm  *otto.Otto
	sem *semaphore.Semaphore
}

var jailInstance *Jail
var once sync.Once

func New() *Jail {
	once.Do(func() {
		jailInstance = &Jail{
			Config: &JailConfig{
				LongRunningJobTimeout: JailedRuntimeRequestTimeout * time.Second,
			},
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
		sem: semaphore.New(1, JailedRuntimeRequestTimeout*time.Second),
	}
}

func (jail *Jail) Parse(chatId string, js string) string {
	defer geth.HaltOnPanic()

	if jail == nil {
		return printError(ErrInvalidJail.Error())
	}

	jail.Lock()
	defer jail.Unlock()

	jail.cells[chatId] = NewJailedRuntime(chatId)
	vm := jail.cells[chatId].vm
	vm.Interrupt = make(chan func(), 1) // make sure that interrupt channel exists

	initJjs := jail.statusJS + ";"
	_, err := jail.RunUnsafe(vm, func() (otto.Value, error) {
		return vm.Run(initJjs)
	})

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

	jjs := Web3_JS + `
	var Web3 = require('web3');
	var web3 = new Web3(jeth);
	var Bignumber = require("bignumber.js");
        function bn(val){
            return new Bignumber(val);
        }
	` + js + "; var catalog = JSON.stringify(_status_catalog);"
	response, err := jail.RunUnsafe(vm, func() (otto.Value, error) {
		return vm.Run(jjs)
	})
	if err == nil {
		response, _ = vm.Get("catalog")
	}

	return printResult(response.String(), err)
}

func (jail *Jail) Call(chatId string, path string, args string) string {
	defer geth.HaltOnPanic()

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

	vm := cell.vm.Copy()                // isolate VM to allow concurrent access
	vm.Interrupt = make(chan func(), 1) // make sure that interrupt channel exists

	response, err := jail.RunUnsafe(vm, func() (response otto.Value, err error) {
		return vm.Call("call", nil, path, args)
	})

	return printResult(response.String(), err)
}

// RunUnsafe executes unsafe, 3rd party code with timeout (which halts execution, if job takes too long)
func (jail *Jail) RunUnsafe(vm *otto.Otto, fn func() (response otto.Value, err error)) (response otto.Value, err error) {
	// service timeout panics (allows to return from long running jobs)
	defer func() {
		if r := recover(); r != nil {
			if r == ErrRequestTimeout {
				response, err = otto.Value{}, ErrRequestTimeout
				return
			}

			panic(r) // re-panic
		}

	}()

	// allow long running job interruption
	cancelInterrupt := make(chan struct{}, 1)
	go func() {
		select {
		case <-cancelInterrupt:
			return
		case <-time.After(jail.Config.LongRunningJobTimeout):
			vm.Interrupt <- func() {
				panic(ErrRequestTimeout)
			}
		}
	}()

	response, err = fn()
	cancelInterrupt <- struct{}{} // not a long running job, it is ok to cancel timeout

	return
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

// Send will serialize the first argument, send it to the node and returns the response.
func (jail *Jail) Send(chatId string, call otto.FunctionCall) (response otto.Value) {
	client, err := jail.RPCClient()
	if err != nil {
		return newErrorResponse(call, -32603, err.Error(), nil)
	}

	requestQueue, err := jail.RequestQueue()
	if err != nil {
		return newErrorResponse(call, -32603, err.Error(), nil)
	}

	// Remarshal the request into a Go value.
	JSON, _ := call.Otto.Object("JSON")
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
	resps, _ := call.Otto.Object("new Array()")
	for _, req := range reqs {
		resp, _ := call.Otto.Object(`({"jsonrpc":"2.0"})`)
		resp.Set("id", req.Id)
		var result json.RawMessage

		// execute directly w/o RPC call to node
		if req.Method == geth.SendTransactionRequest {
			txHash, err := requestQueue.ProcessSendTransactionRequest(call.Otto, req)
			resp.Set("result", txHash.Hex())
			if err != nil {
				resp = newErrorResponse(call, -32603, err.Error(), &req.Id).Object()
			}
			resps.Call("push", resp)
			continue
		}

		// do extra request pre processing (persist message id)
		// within function semaphore will be acquired and released,
		// so that no more than one client (per cell) can enter
		messageId, err := requestQueue.PreProcessRequest(call.Otto, req)
		if err != nil {
			return newErrorResponse(call, -32603, err.Error(), nil)
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
				resultVal, err := JSON.Call("parse", string(result))
				if err != nil {
					resp = newErrorResponse(call, -32603, err.Error(), &req.Id).Object()
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
			resp = newErrorResponse(call, -32603, err.Error(), &req.Id).Object()
		}
		resps.Call("push", resp)

		// do extra request post processing (setting back tx context)
		requestQueue.PostProcessRequest(call.Otto, req, messageId)
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

func newErrorResponse(call otto.FunctionCall, code int, msg string, id interface{}) otto.Value {
	// Bundle the error into a JSON RPC call response
	m := map[string]interface{}{"jsonrpc": "2.0", "id": id, "error": map[string]interface{}{"code": code, msg: msg}}
	res, _ := json.Marshal(m)
	val, _ := call.Otto.Run("(" + string(res) + ")")
	return val
}

func newResultResponse(call otto.FunctionCall, result interface{}) otto.Value {
	resp, _ := call.Otto.Object(`({"jsonrpc":"2.0"})`)
	resp.Set("result", result)

	return resp.Value()
}

// throwJSException panics on an otto.Value. The Otto VM will recover from the
// Go panic and throw msg as a JavaScript error.
func throwJSException(msg interface{}) otto.Value {
	val, err := otto.ToValue(msg)
	if err != nil {
		glog.V(logger.Error).Infof("Failed to serialize JavaScript exception %v: %v", msg, err)
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
