package main

import (
	"github.com/robertkrimen/otto"
	"fmt"
	"encoding/json"
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
	jjs := statusJs + js + `
	var catalog = JSON.stringify(_status_catalog);
	`
	vms[chatId] = vm
	_, err := vm.Run(jjs)
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
