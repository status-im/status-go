package main

import "C"
import (
	"encoding/json"
	"fmt"
	"os"
)

var emptyError = ""

//export CreateAccount
func CreateAccount(password *C.char) *C.char {

	// This is equivalent to creating an account from the command line,
	// just modified to handle the function arg passing
	address, pubKey, err := createAccount(C.GoString(password))

	errString := emptyError
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := AccountInfo{
		Address: address,
		PubKey:  pubKey,
		Error:   errString,
	}
	outBytes, _ := json.Marshal(&out)

	return C.CString(string(outBytes))
}

//export Login
func Login(address, password *C.char) *C.char {
	// Equivalent to unlocking an account briefly, to inject a whisper identity,
	// then locking the account again
	out := UnlockAccount(address, password, 1)
	return out
}

//export UnlockAccount
func UnlockAccount(address, password *C.char, seconds int) *C.char {

	// This is equivalent to unlocking an account from the command line,
	// just modified to unlock the account for the currently running geth node
	// based on the provided arguments
	err := unlockAccount(C.GoString(address), C.GoString(password), seconds)

	errString := emptyError
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := JSONError{
		Error: errString,
	}
	outBytes, _ := json.Marshal(&out)

	return C.CString(string(outBytes))
}

//export StartNode
func StartNode(datadir *C.char) *C.char {

	// This starts a geth node with the given datadir
	err := createAndStartNode(C.GoString(datadir))

	errString := emptyError
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := JSONError{
		Error: errString,
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

//export addPeer
func addPeer(url *C.char) *C.char {
	success, err := doAddPeer(C.GoString(url))
	errString := emptyError
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := AddPeerResult{
		Success: success,
		Error:   errString,
	}
	outBytes, _ := json.Marshal(&out)

	return C.CString(string(outBytes))
}
