package jail

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth"
)

var (
	ErrInvalidJail = errors.New("jail environment is not properly initialized")
)

type Jail struct {
	client   *rpc.ClientRestartWrapper // lazy inited on the first call to jail.ClientRestartWrapper()
	VMs      map[string]*otto.Otto
	statusJS string
}

var jailInstance *Jail
var once sync.Once

func New() *Jail {
	once.Do(func() {
		jailInstance = &Jail{
			VMs: make(map[string]*otto.Otto),
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

func (jail *Jail) Parse(chatId string, js string) string {
	if jail == nil {
		return printError(ErrInvalidJail.Error())
	}

	vm := otto.New()
	initJjs := jail.statusJS + ";"
	jail.VMs[chatId] = vm
	_, err := vm.Run(initJjs)
	vm.Set("jeth", struct{}{})

	jethObj, _ := vm.Get("jeth")
	jethObj.Object().Set("send", jail.Send)
	jethObj.Object().Set("sendAsync", jail.Send)

	jjs := Web3_JS + `
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
	_, err := jail.ClientRestartWrapper()
	if err != nil {
		return printError(err.Error())
	}

	vm, ok := jail.VMs[chatId]
	if !ok {
		return printError(fmt.Sprintf("VM[%s] doesn't exist.", chatId))
	}

	res, err := vm.Call("call", nil, path, args)

	return printResult(res.String(), err)
}

func (jail *Jail) GetVM(chatId string) (*otto.Otto, error) {
	if jail == nil {
		return nil, ErrInvalidJail
	}

	vm, ok := jail.VMs[chatId]
	if !ok {
		return nil, fmt.Errorf("VM[%s] doesn't exist.", chatId)
	}

	return vm, nil
}

type jsonrpcCall struct {
	Id     int64
	Method string
	Params []interface{}
}

// Send will serialize the first argument, send it to the node and returns the response.
func (jail *Jail) Send(call otto.FunctionCall) (response otto.Value) {
	clientFactory, err := jail.ClientRestartWrapper()
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
		reqs   []jsonrpcCall
		batch  bool
	)
	if rawReq[0] == '[' {
		batch = true
		json.Unmarshal(rawReq, &reqs)
	} else {
		batch = false
		reqs = make([]jsonrpcCall, 1)
		json.Unmarshal(rawReq, &reqs[0])
	}

	// Execute the requests.
	resps, _ := call.Otto.Object("new Array()")
	for _, req := range reqs {
		resp, _ := call.Otto.Object(`({"jsonrpc":"2.0"})`)
		resp.Set("id", req.Id)
		var result json.RawMessage

		client := clientFactory.Client()
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

func (jail *Jail) ClientRestartWrapper() (*rpc.ClientRestartWrapper, error) {
	if jail == nil {
		return nil, ErrInvalidJail
	}

	if jail.client != nil {
		return jail.client, nil
	}

	nodeManager := geth.GetNodeManager()
	if !nodeManager.HasNode() {
		return nil, geth.ErrInvalidGethNode
	}

	// obtain RPC client from running node
	client, err := nodeManager.ClientRestartWrapper()
	if err != nil {
		return nil, err
	}
	jail.client = client

	return jail.client, nil
}

func newErrorResponse(call otto.FunctionCall, code int, msg string, id interface{}) otto.Value {
	// Bundle the error into a JSON RPC call response
	m := map[string]interface{}{"version": "2.0", "id": id, "error": map[string]interface{}{"code": code, msg: msg}}
	res, _ := json.Marshal(m)
	val, _ := call.Otto.Run("(" + string(res) + ")")
	return val
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
