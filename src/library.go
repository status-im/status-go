package main

import "C"
import (
	"fmt"
	"os"
)

//export doCreateAccount
func doCreateAccount(password, keydir *C.char) (*C.char, *C.char, C.int) {
	// This is equivalent to creating an account from the command line,
	// just modified to handle the function arg passing
	address, pubKey, err := createAccount(C.GoString(password), C.GoString(keydir))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return C.CString(""), C.CString(""), -1
	}
	return C.CString(address), C.CString(pubKey), 0
}

//export doUnlockAccount
func doUnlockAccount(address, password *C.char) C.int {
	// This is equivalent to unlocking an account from the command line,
	// just modified to unlock the account for the currently running geth node
	// based on the provided arguments
	if err := unlockAccount(C.GoString(address), C.GoString(password)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return -1
	}
	return 0
}

// export doStartNode
func doStartNode(datadir *C.char) C.int {
	// This starts a geth node with the given datadir
	if err := createAndStartNode(C.GoString(datadir)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return -1
	}
	return 0
}
