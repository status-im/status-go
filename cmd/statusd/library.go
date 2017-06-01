package main

import "C"
import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
)

var statusAPI *api.StatusAPI

//export StartAPI
func StartAPI(host, logLevel *C.char) *C.char {
	defer common.HaltOnPanic()

	var err error
	statusAPI, err = api.NewStatusAPI(C.GoString(host), C.GoString(logLevel))
	log.Info("StatusAPI.StartAPI()", "host", C.GoString(host), "logLevel", C.GoString(logLevel))
	return makeJSONResponse(err)
}

//export GenerateConfig
func GenerateConfig(datadir *C.char, networkID C.int, devMode C.int) *C.char {
	defer common.HaltOnPanic()

	log.Info("StatusAPI.GenerateConfig()", "dataDir", C.GoString(datadir), "networkID", networkID, "devMode", devMode)

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
	defer common.HaltOnPanic()

	log.Info("StatusAPI.StartNode()", "config", C.GoString(configJSON))

	config, err := params.LoadNodeConfig(C.GoString(configJSON))
	if err != nil {
		return makeJSONResponse(err)
	}

	_, err = statusAPI.StartNodeAsync(config)
	return makeJSONResponse(err)
}

//export StopNode
func StopNode() *C.char {
	defer common.HaltOnPanic()

	log.Info("StatusAPI.StopNode()")

	_, err := statusAPI.StopNodeAsync()
	return makeJSONResponse(err)
}

//export ResetChainData
func ResetChainData() *C.char {
	defer common.HaltOnPanic()

	log.Info("StatusAPI.ResetChainData()")

	_, err := statusAPI.ResetChainDataAsync()
	return makeJSONResponse(err)
}

//export CallRPC
func CallRPC(inputJSON *C.char) *C.char {
	defer common.HaltOnPanic()

	log.Info("StatusAPI.CallRPC()", "input", C.GoString(inputJSON))

	outputJSON := statusAPI.CallRPC(C.GoString(inputJSON))
	return C.CString(outputJSON)
}

//export ResumeNode
func ResumeNode() *C.char {
	defer common.HaltOnPanic()

	log.Warn("StatusAPI.ResumeNode()")

	err := fmt.Errorf("%v: %v", common.ErrDeprecatedMethod.Error(), "ResumeNode")
	return makeJSONResponse(err)
}

//export StopNodeRPCServer
func StopNodeRPCServer() *C.char {
	defer common.HaltOnPanic()

	log.Warn("StatusAPI.StopNodeRPCServer()")

	err := fmt.Errorf("%v: %v", common.ErrDeprecatedMethod.Error(), "StopNodeRPCServer")
	return makeJSONResponse(err)
}

//export StartNodeRPCServer
func StartNodeRPCServer() *C.char {
	defer common.HaltOnPanic()

	log.Warn("StatusAPI.StartNodeRPCServer()")

	err := fmt.Errorf("%v: %v", common.ErrDeprecatedMethod.Error(), "StartNodeRPCServer")
	return makeJSONResponse(err)
}

//export PopulateStaticPeers
func PopulateStaticPeers() *C.char {
	defer common.HaltOnPanic()

	log.Warn("StatusAPI.PopulateStaticPeers()")

	err := fmt.Errorf("%v: %v", common.ErrDeprecatedMethod.Error(), "PopulateStaticPeers")
	return makeJSONResponse(err)
}

//export AddPeer
func AddPeer(url *C.char) *C.char {
	defer common.HaltOnPanic()

	log.Warn("StatusAPI.AddPeer()")

	err := fmt.Errorf("%v: %v", common.ErrDeprecatedMethod.Error(), "AddPeer")
	return makeJSONResponse(err)
}

//export CreateAccount
func CreateAccount(password *C.char) *C.char {
	defer common.HaltOnPanic()

	log.Info("StatusAPI.CreateAccount()")

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
	defer common.HaltOnPanic()

	log.Info("StatusAPI.CreateChildAccount()", "parent", C.GoString(parentAddress))

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
	defer common.HaltOnPanic()

	log.Info("StatusAPI.RecoverAccount()")

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
	defer common.HaltOnPanic()

	log.Info("StatusAPI.VerifyAccountPassword()", "keyStoreDir", C.GoString(keyStoreDir), "address", C.GoString(address))

	_, err := statusAPI.VerifyAccountPassword(C.GoString(keyStoreDir), C.GoString(address), C.GoString(password))
	return makeJSONResponse(err)
}

//export Login
func Login(address, password *C.char) *C.char {
	defer common.HaltOnPanic()

	log.Info("StatusAPI.Login()", "address", C.GoString(address))

	// loads a key file (for a given address), tries to decrypt it using the password, to verify ownership
	// if verified, purges all the previous identities from Whisper, and injects verified key as shh identity
	err := statusAPI.SelectAccount(C.GoString(address), C.GoString(password))
	return makeJSONResponse(err)
}

//export Logout
func Logout() *C.char {
	defer common.HaltOnPanic()

	log.Info("StatusAPI.Logout()")

	// This is equivalent to clearing whisper identities
	err := statusAPI.Logout()
	return makeJSONResponse(err)
}

//export CompleteTransaction
func CompleteTransaction(id, password *C.char) *C.char {
	defer common.HaltOnPanic()

	log.Info("StatusAPI.CompleteTransaction()", "id", C.GoString(id))

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
	defer common.HaltOnPanic()

	log.Info("StatusAPI.CompleteTransactions()", "ids", C.GoString(ids))

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
	defer common.HaltOnPanic()

	log.Info("StatusAPI.DiscardTransaction()", "id", C.GoString(id))

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
	defer common.HaltOnPanic()

	log.Info("StatusAPI.DiscardTransactions()", "ids", C.GoString(ids))

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
	defer common.HaltOnPanic()

	log.Info("StatusAPI.InitJail()")

	statusAPI.JailBaseJS(C.GoString(js))
}

//export Parse
func Parse(chatID *C.char, js *C.char) *C.char {
	defer common.HaltOnPanic()

	log.Info("StatusAPI.Parse()")

	res := statusAPI.JailParse(C.GoString(chatID), C.GoString(js))
	return C.CString(res)
}

//export Call
func Call(chatID *C.char, path *C.char, params *C.char) *C.char {
	defer common.HaltOnPanic()

	log.Info("StatusAPI.Call()", "chatID", C.GoString(chatID), "path", C.GoString(path), "params", C.GoString(params))

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
