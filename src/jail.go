package main

import (
	"github.com/robertkrimen/otto"
)

var statusJs string
var vms = make(map[string]*otto.Otto)

func Init(js string) {
	statusJs = js
}

func Parse(chatId string, js string) string {
	vm := otto.New()
	jjs := statusJs + js + `
	var catalog = JSON.stringify(_status_catalog);
	`
	vms[chatId] = vm
	vm.Run(jjs)
	res, _ := vm.Get("catalog")

	return res.String()
}

func Call(chatId string, path string, args string) string {
	vm, ok := vms[chatId]
	if !ok {
		return ""
	}

	res, _ := vm.Call("call", nil, path, args)

	return res.String()
}
