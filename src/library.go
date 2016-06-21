package main

import "C"
import (
	"fmt"
	"os"
)

//export doCreateAccount
func doCreateAccount(password, keydir *C.char) (*C.char, C.int) {
	// This is equivalent to creating an account from the command line,
	// just modified to handle the function arg passing
	address, err := createAccount(C.GoString(password), C.GoString(keydir))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return C.CString(""), -1
	}
	return C.CString(address), 0
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
