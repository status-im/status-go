package main

import (
	"github.com/robertkrimen/otto"
	"fmt"
	"encoding/json"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rpc"
)

var statusJs string
var vms = make(map[string]*otto.Otto)

func Init(js string) {
	statusJs = js
}

func printError(error string) string {
	str := JSONError{
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
			res = "null";
		}
		out = fmt.Sprintf(`{"result": %s}`, res)
	}

	return out
}

func Parse(chatId string, js string) string {
	vm := otto.New()
	initJjs := statusJs + ";"
	vms[chatId] = vm
	_, err := vm.Run(initJjs)
	vm.Set("jeth", struct{}{})

	jethObj, _ := vm.Get("jeth")
	jethObj.Object().Set("send", Send)
	jethObj.Object().Set("sendAsync", Send)

	jjs :=  Web3_JS + `
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

func Call(chatId string, path string, args string) string {
	vm, ok := vms[chatId]
	if !ok {
		return printError(fmt.Sprintf("Vm[%s] doesn't exist.", chatId))
	}

	res, err := vm.Call("call", nil, path, args)

	return printResult(res.String(), err);
}

// Send will serialize the first argument, send it to the node and returns the response.
func Send(call otto.FunctionCall) (response otto.Value) {
	// Ensure that we've got a batch request (array) or a single request (object)
	arg := call.Argument(0).Object()
	if arg == nil || (arg.Class() != "Array" && arg.Class() != "Object") {
		throwJSException("request must be an object or array")
	}
	// Convert the otto VM arguments to Go values
	data, err := call.Otto.Call("JSON.stringify", nil, arg)
	if err != nil {
		throwJSException(err.Error())
	}
	reqjson, err := data.ToString()

	if err != nil {
		throwJSException(err.Error())
	}

	var (
		reqs  []rpc.JSONRequest
		batch = true
	)
	if err = json.Unmarshal([]byte(reqjson), &reqs); err != nil {
		// single request?
		reqs = make([]rpc.JSONRequest, 1)
		if err = json.Unmarshal([]byte(reqjson), &reqs[0]); err != nil {
			throwJSException("invalid request")
		}
		batch = false
	}
	// Iteratively execute the requests
	call.Otto.Set("response_len", len(reqs))
	call.Otto.Run("var ret_response = new Array(response_len);")

	for i, req := range reqs {
		// Execute the RPC request and parse the reply
		if err = client.Send(&req); err != nil {
			return newErrorResponse(call, -32603, err.Error(), req.Id)
		}
		result := make(map[string]interface{})
		if err = client.Recv(&result); err != nil {
			return newErrorResponse(call, -32603, err.Error(), req.Id)
		}
		// Feed the reply back into the JavaScript runtime environment
		id, _ := result["id"]
		jsonver, _ := result["jsonrpc"]

		call.Otto.Set("ret_id", id)
		call.Otto.Set("ret_jsonrpc", jsonver)
		call.Otto.Set("response_idx", i)

		if res, ok := result["result"]; ok {
			payload, _ := json.Marshal(res)
			call.Otto.Set("ret_result", string(payload))
			response, err = call.Otto.Run(`
				ret_response[response_idx] = { jsonrpc: ret_jsonrpc, id: ret_id, result: JSON.parse(ret_result) };
			`)
			continue
		}
		if res, ok := result["error"]; ok {
			payload, _ := json.Marshal(res)
			call.Otto.Set("ret_result", string(payload))
			response, err = call.Otto.Run(`
				ret_response[response_idx] = { jsonrpc: ret_jsonrpc, id: ret_id, error: JSON.parse(ret_result) };
			`)
			continue
		}
		return newErrorResponse(call, -32603, fmt.Sprintf("Invalid response"), new(int64))
	}
	// Convert single requests back from batch ones
	if !batch {
		call.Otto.Run("ret_response = ret_response[0];")
	}
	// Execute any registered callbacks
	if call.Argument(1).IsObject() {
		call.Otto.Set("callback", call.Argument(1))
		call.Otto.Run(`
		if (Object.prototype.toString.call(callback) == '[object Function]') {
			callback(null, ret_response);
		}
		`)
	}
	return
}


// newErrorResponse creates a JSON RPC error response for a specific request id,
// containing the specified error code and error message. Beside returning the
// error to the caller, it also sets the ret_error and ret_response JavaScript
// variables.
func newErrorResponse(call otto.FunctionCall, code int, msg string, id interface{}) (response otto.Value) {
	// Bundle the error into a JSON RPC call response
	res := rpc.JSONErrResponse{
		Version: "2.0",
		Id:      id,
		Error: rpc.JSONError{
			Code:    code,
			Message: msg,
		},
	}
	// Serialize the error response into JavaScript variables
	errObj, err := json.Marshal(res.Error)
	if err != nil {
		glog.V(logger.Error).Infof("Failed to serialize JSON RPC error: %v", err)
	}
	resObj, err := json.Marshal(res)
	if err != nil {
		glog.V(logger.Error).Infof("Failed to serialize JSON RPC error response: %v", err)
	}

	if _, err = call.Otto.Run("ret_error = " + string(errObj)); err != nil {
		glog.V(logger.Error).Infof("Failed to set `ret_error` to the occurred error: %v", err)
	}
	resVal, err := call.Otto.Run("ret_response = " + string(resObj))
	if err != nil {
		glog.V(logger.Error).Infof("Failed to set `ret_response` to the JSON RPC response: %v", err)
	}
	return resVal
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
