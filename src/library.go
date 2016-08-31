package main

import "C"
import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/whisper"
	"os"
)

var emptyError = ""

//export CreateAccount
func CreateAccount(password *C.char) *C.char {

	// This is equivalent to creating an account from the command line,
	// just modified to handle the function arg passing
	address, pubKey, mnemonic, err := createAccount(C.GoString(password))

	errString := emptyError
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := AccountInfo{
		Address:  address,
		PubKey:   pubKey,
		Mnemonic: mnemonic,
		Error:    errString,
	}
	outBytes, _ := json.Marshal(&out)

	return C.CString(string(outBytes))
}

//export RecoverAccount
func RecoverAccount(password, mnemonic *C.char) *C.char {

	address, pubKey, err := recoverAccount(C.GoString(password), C.GoString(mnemonic))

	errString := emptyError
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := AccountInfo{
		Address:  address,
		PubKey:   pubKey,
		Mnemonic: C.GoString(mnemonic),
		Error:    errString,
	}
	outBytes, _ := json.Marshal(&out)

	return C.CString(string(outBytes))
}

//export Login
func Login(address, password *C.char) *C.char {
	// loads a key file (for a given address), tries to decrypt it using the password, to verify ownership
	// if verified, purges all the previous identities from Whisper, and injects verified key as shh identity
	err := selectAccount(C.GoString(address), C.GoString(password))

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

//export Logout
func Logout() *C.char {

	// This is equivalent to clearing whisper identities
	err := logout()

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

//export CompleteTransaction
func CompleteTransaction(id, password *C.char) *C.char {
	txHash, err := completeTransaction(C.GoString(id), C.GoString(password))

	errString := emptyError
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := CompleteTransactionResult{
		Hash:  txHash.Hex(),
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

//export addWhisperFilter
func addWhisperFilter(filterJson *C.char) *C.char {

	var id int
	var filter whisper.NewFilterArgs

	err := json.Unmarshal([]byte(C.GoString(filterJson)), &filter)
	if err == nil {
		id = doAddWhisperFilter(filter)
	}

	errString := emptyError
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := AddWhisperFilterResult{
		Id:    id,
		Error: errString,
	}
	outBytes, _ := json.Marshal(&out)

	return C.CString(string(outBytes))

}

//export removeWhisperFilter
func removeWhisperFilter(idFilter int) {

	doRemoveWhisperFilter(idFilter)
}

//export clearWhisperFilters
func clearWhisperFilters() {

	doClearWhisperFilters()
}
