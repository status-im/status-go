package main

import "C"
import (
	"encoding/json"
	"fmt"
	"os"
)

//export doCreateAccount
func doCreateAccount(password, keydir *C.char) *C.char {
	// This is equivalent to creating an account from the command line,
	// just modified to handle the function arg passing
	address, pubKey, err := createAccount(C.GoString(password), C.GoString(keydir))
	out := AccountInfo{
		Address: address,
		PubKey:  pubKey,
		Error:   err.Error(),
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	outBytes, _ := json.Marshal(&out)
	return C.CString(string(outBytes))

}

//export doUnlockAccount
func doUnlockAccount(address, password *C.char, seconds int) *C.char {
	// This is equivalent to unlocking an account from the command line,
	// just modified to unlock the account for the currently running geth node
	// based on the provided arguments
	err := unlockAccount(C.GoString(address), C.GoString(password), seconds)
	out := JSONError{
		Error: err.Error(),
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	outBytes, _ := json.Marshal(&out)
	return C.CString(string(outBytes))
}

//export doStartNode
func doStartNode(datadir *C.char) *C.char {
	// This starts a geth node with the given datadir
	err := createAndStartNode(C.GoString(datadir))
	out := JSONError{
		Error: err.Error(),
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	outBytes, _ := json.Marshal(&out)
	return C.CString(string(outBytes))
}

//export parse
func parse(chatId *C.char, js *C.char) *C.char {
	res := Parse(C.GoString(chatId), C.GoString(js))
	return C.CString(res)
}

//export call
func call(chatId *C.char, path *C.char, params *C.char) *C.char {
	res := Call(C.GoString(chatId), C.GoString(path), C.GoString(params))
	return C.CString(res)
}

//export initJail
func initJail(js *C.char) {
	Init(C.GoString(js))
}
