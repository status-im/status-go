package main

import "C"
import (
	"fmt"
	"os"
)

//export doCreateAccount
func doCreateAccount(password, keydir *C.char) *C.char {
	// This is equivalent to creating an account from the command line,
	// just modified to handle the function arg passing
	address, pubKey, err := createAccount(C.GoString(password), C.GoString(keydir))
	out := fmt.Sprintf(`{
		"address": %s,
		"pubkey": %s,
		"error": %s
	}`, address, pubKey, err.Error())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return C.CString(out)
	}
	return C.CString(out)
}

//export doUnlockAccount
func doUnlockAccount(address, password *C.char) *C.char {
	// This is equivalent to unlocking an account from the command line,
	// just modified to unlock the account for the currently running geth node
	// based on the provided arguments
	err := unlockAccount(C.GoString(address), C.GoString(password))
	out := fmt.Sprintf("{\"error\": %s}", err.Error())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return C.CString(out)
	}
	return C.CString(out)
}

//export doStartNode
func doStartNode(datadir *C.char) *C.char {
	// This starts a geth node with the given datadir
	err := createAndStartNode(C.GoString(datadir))
	out := fmt.Sprintf("{\"error\": %s}", err.Error())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return C.CString(out)
	}
	return C.CString(out)
}
