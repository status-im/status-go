package main

import "C"
import (
	"encoding/json"
	"fmt"
	"os"

	whisper "github.com/ethereum/go-ethereum/whisper/whisperv2"
	"github.com/status-im/status-go/geth"
	"github.com/status-im/status-go/geth/jail"
	"github.com/status-im/status-go/geth/params"
)

//export CreateAccount
func CreateAccount(password *C.char) *C.char {
	// This is equivalent to creating an account from the command line,
	// just modified to handle the function arg passing
	address, pubKey, mnemonic, err := geth.CreateAccount(C.GoString(password))

	errString := ""
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := geth.AccountInfo{
		Address:  address,
		PubKey:   pubKey,
		Mnemonic: mnemonic,
		Error:    errString,
	}
	outBytes, _ := json.Marshal(&out)

	return C.CString(string(outBytes))
}

//export CreateChildAccount
func CreateChildAccount(parentAddress, password *C.char) *C.char {

	address, pubKey, err := geth.CreateChildAccount(C.GoString(parentAddress), C.GoString(password))

	errString := ""
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := geth.AccountInfo{
		Address: address,
		PubKey:  pubKey,
		Error:   errString,
	}
	outBytes, _ := json.Marshal(&out)

	return C.CString(string(outBytes))
}

//export RecoverAccount
func RecoverAccount(password, mnemonic *C.char) *C.char {

	address, pubKey, err := geth.RecoverAccount(C.GoString(password), C.GoString(mnemonic))

	errString := ""
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := geth.AccountInfo{
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
	err := geth.SelectAccount(C.GoString(address), C.GoString(password))

	errString := ""
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := geth.JSONError{
		Error: errString,
	}
	outBytes, _ := json.Marshal(&out)

	return C.CString(string(outBytes))
}

//export Logout
func Logout() *C.char {

	// This is equivalent to clearing whisper identities
	err := geth.Logout()

	errString := ""
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := geth.JSONError{
		Error: errString,
	}
	outBytes, _ := json.Marshal(&out)

	return C.CString(string(outBytes))
}

//export CompleteTransaction
func CompleteTransaction(id, password *C.char) *C.char {
	txHash, err := geth.CompleteTransaction(C.GoString(id), C.GoString(password))

	errString := ""
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := geth.CompleteTransactionResult{
		Id:    C.GoString(id),
		Hash:  txHash.Hex(),
		Error: errString,
	}
	outBytes, _ := json.Marshal(&out)

	return C.CString(string(outBytes))
}

//export CompleteTransactions
func CompleteTransactions(ids, password *C.char) *C.char {
	out := geth.CompleteTransactionsResult{}
	out.Results = make(map[string]geth.CompleteTransactionResult)

	results := geth.CompleteTransactions(C.GoString(ids), C.GoString(password))
	for txId, result := range results {
		txResult := geth.CompleteTransactionResult{
			Id:   txId,
			Hash: result.Hash.Hex(),
		}
		if result.Error != nil {
			txResult.Error = result.Error.Error()
		}
		out.Results[txId] = txResult
	}
	outBytes, _ := json.Marshal(&out)

	return C.CString(string(outBytes))
}

//export DiscardTransaction
func DiscardTransaction(id *C.char) *C.char {
	err := geth.DiscardTransaction(C.GoString(id))

	errString := ""
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := geth.DiscardTransactionResult{
		Id:    C.GoString(id),
		Error: errString,
	}
	outBytes, _ := json.Marshal(&out)

	return C.CString(string(outBytes))
}

//export DiscardTransactions
func DiscardTransactions(ids *C.char) *C.char {
	out := geth.DiscardTransactionsResult{}
	out.Results = make(map[string]geth.DiscardTransactionResult)

	results := geth.DiscardTransactions(C.GoString(ids))
	for txId, result := range results {
		txResult := geth.DiscardTransactionResult{
			Id: txId,
		}
		if result.Error != nil {
			txResult.Error = result.Error.Error()
		}
		out.Results[txId] = txResult
	}
	outBytes, _ := json.Marshal(&out)

	return C.CString(string(outBytes))
}

//export GenerateConfig
func GenerateConfig(datadir *C.char, networkId C.int) *C.char {
	config, err := params.NewNodeConfig(C.GoString(datadir), int(networkId))
	if err != nil {
		return makeJSONErrorResponse(err)
	}

	outBytes, err := json.Marshal(&config)
	if err != nil {
		return makeJSONErrorResponse(err)
	}

	return C.CString(string(outBytes))
}

//export StartNode
func StartNode(configJSON *C.char) *C.char {
	config, err := params.LoadNodeConfig(C.GoString(configJSON))
	if err != nil {
		return makeJSONErrorResponse(err)
	}

	err = geth.CreateAndRunNode(config)
	return makeJSONErrorResponse(err)
}

//export StopNode
func StopNode() *C.char {
	err := geth.NodeManagerInstance().StopNode()
	return makeJSONErrorResponse(err)
}

//export ResumeNode
func ResumeNode() *C.char {
	err := geth.NodeManagerInstance().ResumeNode()
	return makeJSONErrorResponse(err)
}

//export ResetChainData
func ResetChainData() *C.char {
	err := geth.NodeManagerInstance().ResetChainData()
	return makeJSONErrorResponse(err)
}

//export StopNodeRPCServer
func StopNodeRPCServer() *C.char {
	_, err := geth.NodeManagerInstance().StopNodeRPCServer()

	return makeJSONErrorResponse(err)
}

//export StartNodeRPCServer
func StartNodeRPCServer() *C.char {
	_, err := geth.NodeManagerInstance().StartNodeRPCServer()

	return makeJSONErrorResponse(err)
}

//export InitJail
func InitJail(js *C.char) {
	jail.Init(C.GoString(js))
}

//export Parse
func Parse(chatId *C.char, js *C.char) *C.char {
	res := jail.GetInstance().Parse(C.GoString(chatId), C.GoString(js))
	return C.CString(res)
}

//export Call
func Call(chatId *C.char, path *C.char, params *C.char) *C.char {
	res := jail.GetInstance().Call(C.GoString(chatId), C.GoString(path), C.GoString(params))
	return C.CString(res)
}

//export PopulateStaticPeers
func PopulateStaticPeers() {
	geth.NodeManagerInstance().PopulateStaticPeers()
}

//export AddPeer
func AddPeer(url *C.char) *C.char {
	success, err := geth.NodeManagerInstance().AddPeer(C.GoString(url))
	errString := ""
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := geth.AddPeerResult{
		Success: success,
		Error:   errString,
	}
	outBytes, _ := json.Marshal(&out)

	return C.CString(string(outBytes))
}

//export AddWhisperFilter
func AddWhisperFilter(filterJson *C.char) *C.char {

	var id int
	var filter whisper.NewFilterArgs

	err := json.Unmarshal([]byte(C.GoString(filterJson)), &filter)
	if err == nil {
		id = geth.AddWhisperFilter(filter)
	}

	errString := ""
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := geth.AddWhisperFilterResult{
		Id:    id,
		Error: errString,
	}
	outBytes, _ := json.Marshal(&out)

	return C.CString(string(outBytes))

}

//export RemoveWhisperFilter
func RemoveWhisperFilter(idFilter int) {
	geth.RemoveWhisperFilter(idFilter)
}

//export ClearWhisperFilters
func ClearWhisperFilters() {
	geth.ClearWhisperFilters()
}

func makeJSONErrorResponse(err error) *C.char {
	errString := ""
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := geth.JSONError{
		Error: errString,
	}
	outBytes, _ := json.Marshal(&out)

	return C.CString(string(outBytes))
}
