package main

import "C"
import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
)

//export GenerateConfig
func GenerateConfig(datadir *C.char, networkID C.int, devMode C.int) *C.char {
	config, err := params.NewNodeConfig(C.GoString(datadir), uint64(networkID), devMode == 1)
	if err != nil {
		return makeJSONResponse(err)
	}

	outBytes, err := json.Marshal(&config)
	if err != nil {
		return makeJSONResponse(err)
	}

	return C.CString(string(outBytes))
}

//export StartNode
func StartNode(configJSON *C.char) *C.char {
	config, err := params.LoadNodeConfig(C.GoString(configJSON))
	if err != nil {
		return makeJSONResponse(err)
	}

	err = statusAPI.StartNodeNonBlocking(config)
	return makeJSONResponse(err)
}

//export StopNode
func StopNode() *C.char {
	return makeJSONResponse(statusAPI.StopNode())
}

//export ResumeNode
func ResumeNode() *C.char {
	err := fmt.Errorf("%v: %v", common.ErrDeprecatedMethod.Error(), "ResumeNode")
	return makeJSONResponse(err)
}

//export ResetChainData
func ResetChainData() *C.char {
	return makeJSONResponse(statusAPI.ResetChainData())
}

//export StopNodeRPCServer
func StopNodeRPCServer() *C.char {
	err := fmt.Errorf("%v: %v", common.ErrDeprecatedMethod.Error(), "StopNodeRPCServer")
	return makeJSONResponse(err)
}

//export StartNodeRPCServer
func StartNodeRPCServer() *C.char {
	err := fmt.Errorf("%v: %v", common.ErrDeprecatedMethod.Error(), "StartNodeRPCServer")
	return makeJSONResponse(err)
}

//export PopulateStaticPeers
func PopulateStaticPeers() *C.char {
	err := fmt.Errorf("%v: %v", common.ErrDeprecatedMethod.Error(), "PopulateStaticPeers")
	return makeJSONResponse(err)
}

//export AddPeer
func AddPeer(url *C.char) *C.char {
	err := fmt.Errorf("%v: %v", common.ErrDeprecatedMethod.Error(), "AddPeer")
	return makeJSONResponse(err)
}

//export CreateAccount
func CreateAccount(password *C.char) *C.char {
	// This is equivalent to creating an account from the command line,
	// just modified to handle the function arg passing
	address, pubKey, mnemonic, err := statusAPI.CreateAccount(C.GoString(password))

	errString := ""
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := common.AccountInfo{
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
	address, pubKey, err := statusAPI.CreateChildAccount(C.GoString(parentAddress), C.GoString(password))

	errString := ""
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := common.AccountInfo{
		Address: address,
		PubKey:  pubKey,
		Error:   errString,
	}
	outBytes, _ := json.Marshal(&out)
	return C.CString(string(outBytes))
}

//export RecoverAccount
func RecoverAccount(password, mnemonic *C.char) *C.char {
	address, pubKey, err := statusAPI.RecoverAccount(C.GoString(password), C.GoString(mnemonic))

	errString := ""
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := common.AccountInfo{
		Address:  address,
		PubKey:   pubKey,
		Mnemonic: C.GoString(mnemonic),
		Error:    errString,
	}
	outBytes, _ := json.Marshal(&out)
	return C.CString(string(outBytes))
}

//export VerifyAccountPassword
func VerifyAccountPassword(keyStoreDir, address, password *C.char) *C.char {
	_, err := statusAPI.VerifyAccountPassword(C.GoString(keyStoreDir), C.GoString(address), C.GoString(password))
	return makeJSONResponse(err)
}

//export Login
func Login(address, password *C.char) *C.char {
	// loads a key file (for a given address), tries to decrypt it using the password, to verify ownership
	// if verified, purges all the previous identities from Whisper, and injects verified key as shh identity
	err := statusAPI.SelectAccount(C.GoString(address), C.GoString(password))
	return makeJSONResponse(err)
}

//export Logout
func Logout() *C.char {
	// This is equivalent to clearing whisper identities
	err := statusAPI.Logout()
	return makeJSONResponse(err)
}

//export CompleteTransaction
func CompleteTransaction(id, password *C.char) *C.char {
	txHash, err := statusAPI.CompleteTransaction(C.GoString(id), C.GoString(password))

	errString := ""
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := common.CompleteTransactionResult{
		ID:    C.GoString(id),
		Hash:  txHash.Hex(),
		Error: errString,
	}
	outBytes, _ := json.Marshal(&out)

	return C.CString(string(outBytes))
}

//export CompleteTransactions
func CompleteTransactions(ids, password *C.char) *C.char {
	out := common.CompleteTransactionsResult{}
	out.Results = make(map[string]common.CompleteTransactionResult)

	results := statusAPI.CompleteTransactions(C.GoString(ids), C.GoString(password))
	for txID, result := range results {
		txResult := common.CompleteTransactionResult{
			ID:   txID,
			Hash: result.Hash.Hex(),
		}
		if result.Error != nil {
			txResult.Error = result.Error.Error()
		}
		out.Results[txID] = txResult
	}
	outBytes, _ := json.Marshal(&out)

	return C.CString(string(outBytes))
}

//export DiscardTransaction
func DiscardTransaction(id *C.char) *C.char {
	err := statusAPI.DiscardTransaction(C.GoString(id))

	errString := ""
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := common.DiscardTransactionResult{
		ID:    C.GoString(id),
		Error: errString,
	}
	outBytes, _ := json.Marshal(&out)

	return C.CString(string(outBytes))
}

//export DiscardTransactions
func DiscardTransactions(ids *C.char) *C.char {
	out := common.DiscardTransactionsResult{}
	out.Results = make(map[string]common.DiscardTransactionResult)

	results := statusAPI.DiscardTransactions(C.GoString(ids))
	for txID, result := range results {
		txResult := common.DiscardTransactionResult{
			ID: txID,
		}
		if result.Error != nil {
			txResult.Error = result.Error.Error()
		}
		out.Results[txID] = txResult
	}
	outBytes, _ := json.Marshal(&out)

	return C.CString(string(outBytes))
}

//export InitJail
func InitJail(js *C.char) {
	statusAPI.JailBaseJS(C.GoString(js))
}

//export Parse
func Parse(chatID *C.char, js *C.char) *C.char {
	res := statusAPI.JailParse(C.GoString(chatID), C.GoString(js))
	return C.CString(res)
}

//export Call
func Call(chatID *C.char, path *C.char, params *C.char) *C.char {
	res := statusAPI.JailCall(C.GoString(chatID), C.GoString(path), C.GoString(params))
	return C.CString(res)
}

func makeJSONResponse(err error) *C.char {
	errString := ""
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := common.APIResponse{
		Error: errString,
	}
	outBytes, _ := json.Marshal(&out)

	return C.CString(string(outBytes))
}
