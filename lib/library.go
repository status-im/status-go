package main

import "C"
import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/NaySoftware/go-fcm"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/profiling"
	"gopkg.in/go-playground/validator.v9"
)

//GenerateConfig for status node
//export GenerateConfig
func GenerateConfig(datadir *C.char, networkID C.int, devMode C.int) *C.char {
	config, err := params.NewNodeConfig(C.GoString(datadir), uint64(networkID), devMode == 1)
	if err != nil {
		return makeJSONResponse(err)
	}

	outBytes, err := json.Marshal(config)
	if err != nil {
		return makeJSONResponse(err)
	}

	return C.CString(string(outBytes))
}

//StartNode - start Status node
//export StartNode
func StartNode(configJSON *C.char) *C.char {
	config, err := params.LoadNodeConfig(C.GoString(configJSON))
	if err != nil {
		return makeJSONResponse(err)
	}

	statusAPI.StartNodeAsync(config)
	return makeJSONResponse(nil)
}

//StopNode - stop status node
//export StopNode
func StopNode() *C.char {
	statusAPI.StopNodeAsync()
	return makeJSONResponse(nil)
}

//ValidateNodeConfig validates config for status node
//export ValidateNodeConfig
func ValidateNodeConfig(configJSON *C.char) *C.char {
	var resp common.APIDetailedResponse

	_, err := params.LoadNodeConfig(C.GoString(configJSON))

	// Convert errors to common.APIDetailedResponse
	switch err := err.(type) {
	case validator.ValidationErrors:
		resp = common.APIDetailedResponse{
			Message:     "validation: validation failed",
			FieldErrors: make([]common.APIFieldError, len(err)),
		}

		for i, ve := range err {
			resp.FieldErrors[i] = common.APIFieldError{
				Parameter: ve.Namespace(),
				Errors: []common.APIError{
					{
						Message: fmt.Sprintf("field validation failed on the '%s' tag", ve.Tag()),
					},
				},
			}
		}
	case error:
		resp = common.APIDetailedResponse{
			Message: fmt.Sprintf("validation: %s", err.Error()),
		}
	case nil:
		resp = common.APIDetailedResponse{
			Status: true,
		}
	}

	respJSON, err := json.Marshal(resp)
	if err != nil {
		return makeJSONResponse(err)
	}

	return C.CString(string(respJSON))
}

//ResetChainData remove chain data from data directory
//export ResetChainData
func ResetChainData() *C.char {
	statusAPI.ResetChainDataAsync()
	return makeJSONResponse(nil)
}

//CallRPC calls status node via rpc
//export CallRPC
func CallRPC(inputJSON *C.char) *C.char {
	outputJSON := statusAPI.CallRPC(C.GoString(inputJSON))
	return C.CString(outputJSON)
}

//CreateAccount is equivalent to creating an account from the command line,
// just modified to handle the function arg passing
//export CreateAccount
func CreateAccount(password *C.char) *C.char {
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
	outBytes, _ := json.Marshal(out)
	return C.CString(string(outBytes))
}

//CreateChildAccount creates sub-account
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
	outBytes, _ := json.Marshal(out)
	return C.CString(string(outBytes))
}

//RecoverAccount re-creates master key using given details
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
	outBytes, _ := json.Marshal(out)
	return C.CString(string(outBytes))
}

//VerifyAccountPassword verifies account password
//export VerifyAccountPassword
func VerifyAccountPassword(keyStoreDir, address, password *C.char) *C.char {
	_, err := statusAPI.VerifyAccountPassword(C.GoString(keyStoreDir), C.GoString(address), C.GoString(password))
	return makeJSONResponse(err)
}

//Login loads a key file (for a given address), tries to decrypt it using the password, to verify ownership
// if verified, purges all the previous identities from Whisper, and injects verified key as shh identity
//export Login
func Login(address, password *C.char) *C.char {
	err := statusAPI.SelectAccount(C.GoString(address), C.GoString(password))
	return makeJSONResponse(err)
}

//Logout is equivalent to clearing whisper identities
//export Logout
func Logout() *C.char {
	err := statusAPI.Logout()
	return makeJSONResponse(err)
}

//CompleteTransaction instructs backend to complete sending of a given transaction
//export CompleteTransaction
func CompleteTransaction(id, password *C.char) *C.char {
	txHash, err := statusAPI.CompleteTransaction(common.QueuedTxID(C.GoString(id)), C.GoString(password))

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
	outBytes, err := json.Marshal(out)
	if err != nil {
		log.Error("failed to marshal CompleteTransaction output", "error", err.Error())
		return makeJSONResponse(err)
	}

	return C.CString(string(outBytes))
}

//CompleteTransactions instructs backend to complete sending of multiple transactions
//export CompleteTransactions
func CompleteTransactions(ids, password *C.char) *C.char {
	out := common.CompleteTransactionsResult{}
	out.Results = make(map[string]common.CompleteTransactionResult)

	parsedIDs, err := common.ParseJSONArray(C.GoString(ids))
	if err != nil {
		out.Results["none"] = common.CompleteTransactionResult{
			Error: err.Error(),
		}
	} else {
		txIDs := make([]common.QueuedTxID, len(parsedIDs))
		for i, id := range parsedIDs {
			txIDs[i] = common.QueuedTxID(id)
		}

		results := statusAPI.CompleteTransactions(txIDs, C.GoString(password))
		for txID, result := range results {
			txResult := common.CompleteTransactionResult{
				ID:   string(txID),
				Hash: result.Hash.Hex(),
			}
			if result.Error != nil {
				txResult.Error = result.Error.Error()
			}
			out.Results[string(txID)] = txResult
		}
	}

	outBytes, err := json.Marshal(out)
	if err != nil {
		log.Error("failed to marshal CompleteTransactions output", "error", err.Error())
		return makeJSONResponse(err)
	}

	return C.CString(string(outBytes))
}

//DiscardTransaction discards a given transaction from transaction queue
//export DiscardTransaction
func DiscardTransaction(id *C.char) *C.char {
	err := statusAPI.DiscardTransaction(common.QueuedTxID(C.GoString(id)))

	errString := ""
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := common.DiscardTransactionResult{
		ID:    C.GoString(id),
		Error: errString,
	}
	outBytes, err := json.Marshal(out)
	if err != nil {
		log.Error("failed to marshal DiscardTransaction output", "error", err.Error())
		return makeJSONResponse(err)
	}

	return C.CString(string(outBytes))
}

//DiscardTransactions discards given multiple transactions from transaction queue
//export DiscardTransactions
func DiscardTransactions(ids *C.char) *C.char {
	out := common.DiscardTransactionsResult{}
	out.Results = make(map[string]common.DiscardTransactionResult)

	parsedIDs, err := common.ParseJSONArray(C.GoString(ids))
	if err != nil {
		out.Results["none"] = common.DiscardTransactionResult{
			Error: err.Error(),
		}
	} else {
		txIDs := make([]common.QueuedTxID, len(parsedIDs))
		for i, id := range parsedIDs {
			txIDs[i] = common.QueuedTxID(id)
		}

		results := statusAPI.DiscardTransactions(txIDs)
		for txID, result := range results {
			txResult := common.DiscardTransactionResult{
				ID: string(txID),
			}
			if result.Error != nil {
				txResult.Error = result.Error.Error()
			}
			out.Results[string(txID)] = txResult
		}
	}

	outBytes, err := json.Marshal(out)
	if err != nil {
		log.Error("failed to marshal DiscardTransactions output", "error", err.Error())
		return makeJSONResponse(err)
	}

	return C.CString(string(outBytes))
}

//InitJail setup initial JavaScript
//export InitJail
func InitJail(js *C.char) {
	statusAPI.SetJailBaseJS(C.GoString(js))
}

//Parse creates a new jail cell context and executes provided JavaScript code.
//DEPRECATED in favour of CreateAndInitCell.
//export Parse
func Parse(chatID *C.char, js *C.char) *C.char {
	res := statusAPI.CreateAndInitCell(C.GoString(chatID), C.GoString(js))
	return C.CString(res)
}

//CreateAndInitCell creates a new jail cell context and executes provided JavaScript code.
//export CreateAndInitCell
func CreateAndInitCell(chatID *C.char, js *C.char) *C.char {
	res := statusAPI.CreateAndInitCell(C.GoString(chatID), C.GoString(js))
	return C.CString(res)
}

//ExecuteJS allows to run arbitrary JS code within a cell.
//export ExecuteJS
func ExecuteJS(chatID *C.char, code *C.char) *C.char {
	res := statusAPI.JailExecute(C.GoString(chatID), C.GoString(code))
	return C.CString(res)
}

//Call executes given JavaScript function
//export Call
func Call(chatID *C.char, path *C.char, params *C.char) *C.char {
	res := statusAPI.JailCall(C.GoString(chatID), C.GoString(path), C.GoString(params))
	return C.CString(res)
}

//StartCPUProfile runs pprof for cpu
//export StartCPUProfile
func StartCPUProfile(dataDir *C.char) *C.char {
	err := profiling.StartCPUProfile(C.GoString(dataDir))
	return makeJSONResponse(err)
}

//StopCPUProfiling stops pprof for cpu
//export StopCPUProfiling
func StopCPUProfiling() *C.char { //nolint: deadcode
	err := profiling.StopCPUProfile()
	return makeJSONResponse(err)
}

//WriteHeapProfile starts pprof for heap
//export WriteHeapProfile
func WriteHeapProfile(dataDir *C.char) *C.char { //nolint: deadcode
	err := profiling.WriteHeapFile(C.GoString(dataDir))
	return makeJSONResponse(err)
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
	outBytes, _ := json.Marshal(out)

	return C.CString(string(outBytes))
}

// Notify sends push notification by given token
// @deprecated
//export Notify
func Notify(token *C.char) *C.char {
	res := statusAPI.Notify(C.GoString(token))
	return C.CString(res)
}

// NotifyUsers sends push notifications by given tokens.
//export NotifyUsers
func NotifyUsers(message, payloadJSON, tokensArray *C.char) (outCBytes *C.char) {
	var (
		err      error
		outBytes []byte
	)
	errString := ""

	defer func() {
		out := common.NotifyResult{
			Status: err == nil,
			Error:  errString,
		}

		outBytes, err = json.Marshal(out)
		if err != nil {
			log.Error("failed to marshal Notify output", "error", err.Error())
			outCBytes = makeJSONResponse(err)
			return
		}

		outCBytes = C.CString(string(outBytes))
	}()

	tokens, err := common.ParseJSONArray(C.GoString(tokensArray))
	if err != nil {
		errString = err.Error()
		return
	}

	var payload fcm.NotificationPayload
	err = json.Unmarshal([]byte(C.GoString(payloadJSON)), &payload)
	if err != nil {
		errString = err.Error()
		return
	}

	err = statusAPI.NotifyUsers(C.GoString(message), payload, tokens...)
	if err != nil {
		errString = err.Error()
		return
	}

	return
}

// AddPeer adds an enode as a peer.
//export AddPeer
func AddPeer(enode *C.char) *C.char {
	err := statusAPI.NodeManager().AddPeer(C.GoString(enode))
	return makeJSONResponse(err)
}
