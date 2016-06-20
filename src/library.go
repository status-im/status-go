package main

import "C"
import (
	"fmt"
	"os"
)

//export doCreateAccount
func doCreateAccount(password, keydir *C.char) C.int {
	// This is equivalent to creating an account from the command line,
	// just modified to handle the function arg passing
	if err := createAccount(C.GoString(password), C.GoString(keydir)); err != nil {
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
