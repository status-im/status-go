package main

// #ifdef __cplusplus
// extern "C" {
// #endif
// #include <stdbool.h>
//
// extern bool GethServiceSignalEvent( const char *jsonEvent );
//
// #ifdef __cplusplus
// }
// #endif
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

//export testCallback
func testCallback(){
	C.GethServiceSignalEvent(C.CString(`{"ok":"boss"}`))
}
