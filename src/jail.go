package main

import (
	"github.com/robertkrimen/otto"
	"fmt"
)

var statusJs string
var vms = make(map[string]*otto.Otto)

func Init(js string) {
	statusJs = js
}

func printResult(res string, err error) string {
	var out string
	if err != nil {
		out = fmt.Sprintf(`{
		        "error": %s
		        }`,
			err.Error())
	} else {
		out = fmt.Sprintf(`{
		        "result": %s
		        }`,
			res)
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
	err.Error()

	return printResult(res.String(), err)
}

func Call(chatId string, path string, args string) string {
	vm, ok := vms[chatId]
	if !ok {
		return fmt.Sprintf(
			"{\"error\":\"Vm[%s] doesn't exist.\"}",
			chatId)
	}

	res, err := vm.Call("call", nil, path, args)

	return printResult(res.String(), err);
}
