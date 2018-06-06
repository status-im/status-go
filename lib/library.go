package main

import "C"
import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/NaySoftware/go-fcm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/profiling"
	"github.com/status-im/status-go/sign"
	"gopkg.in/go-playground/validator.v9"
)

// All general log messages in this package should be routed through this logger.
var logger = log.New("package", "status-go/lib")

//GenerateConfig for status node
//export GenerateConfig
func GenerateConfig(datadir *C.char, networkID C.int) *C.char {
	config, err := params.NewNodeConfig(C.GoString(datadir), "", uint64(networkID))
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

	if err := logutils.OverrideRootLog(config.LogEnabled, config.LogLevel, config.LogFile, false); err != nil {
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
	var resp APIDetailedResponse

	_, err := params.LoadNodeConfig(C.GoString(configJSON))

	// Convert errors to APIDetailedResponse
	switch err := err.(type) {
	case validator.ValidationErrors:
		resp = APIDetailedResponse{
			Message:     "validation: validation failed",
			FieldErrors: make([]APIFieldError, len(err)),
		}

		for i, ve := range err {
			resp.FieldErrors[i] = APIFieldError{
				Parameter: ve.Namespace(),
				Errors: []APIError{
					{
						Message: fmt.Sprintf("field validation failed on the '%s' tag", ve.Tag()),
					},
				},
			}
		}
	case error:
		resp = APIDetailedResponse{
			Message: fmt.Sprintf("validation: %s", err.Error()),
		}
	case nil:
		resp = APIDetailedResponse{
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

//CallRPC calls public APIs via RPC
//export CallRPC
func CallRPC(inputJSON *C.char) *C.char {
	outputJSON := statusAPI.CallRPC(C.GoString(inputJSON))
	return C.CString(outputJSON)
}

//CallPrivateRPC calls both public and private APIs via RPC
//export CallPrivateRPC
func CallPrivateRPC(inputJSON *C.char) *C.char {
	outputJSON := statusAPI.CallPrivateRPC(C.GoString(inputJSON))
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

	out := AccountInfo{
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

	out := AccountInfo{
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

	out := AccountInfo{
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

//ApproveSignRequestWithArgs instructs backend to complete sending of a given transaction.
// gas and gasPrice will be overrided with the given values before signing the
// transaction.
//export ApproveSignRequestWithArgs
func ApproveSignRequestWithArgs(id, password *C.char, gas, gasPrice C.int) *C.char {
	result := statusAPI.ApproveSignRequestWithArgs(C.GoString(id), C.GoString(password), int64(gas), int64(gasPrice))

	return prepareApproveSignRequestResponse(result, id)
}

//ApproveSignRequest instructs backend to complete sending of a given transaction.
//export ApproveSignRequest
func ApproveSignRequest(id, password *C.char) *C.char {
	result := statusAPI.ApproveSignRequest(C.GoString(id), C.GoString(password))

	return prepareApproveSignRequestResponse(result, id)
}

// prepareApproveSignRequestResponse based on a sign.Result prepares the binding
// response.
func prepareApproveSignRequestResponse(result sign.Result, id *C.char) *C.char {
	errString := ""
	if result.Error != nil {
		fmt.Fprintln(os.Stderr, result.Error)
		errString = result.Error.Error()
	}

	out := SignRequestResult{
		ID:    C.GoString(id),
		Hash:  result.Response.Hex(),
		Error: errString,
	}
	outBytes, err := json.Marshal(out)
	if err != nil {
		logger.Error("failed to marshal ApproveSignRequest output", "error", err)
		return makeJSONResponse(err)
	}

	return C.CString(string(outBytes))
}

//ApproveSignRequests instructs backend to complete sending of multiple transactions
//export ApproveSignRequests
func ApproveSignRequests(ids, password *C.char) *C.char {
	out := SignRequestsResult{}
	out.Results = make(map[string]SignRequestResult)

	parsedIDs, err := ParseJSONArray(C.GoString(ids))
	if err != nil {
		out.Results["none"] = SignRequestResult{
			Error: err.Error(),
		}
	} else {
		txIDs := make([]string, len(parsedIDs))
		for i, id := range parsedIDs {
			txIDs[i] = id
		}

		results := statusAPI.ApproveSignRequests(txIDs, C.GoString(password))
		for txID, result := range results {
			txResult := SignRequestResult{
				ID:   txID,
				Hash: result.Response.Hex(),
			}
			if result.Error != nil {
				txResult.Error = result.Error.Error()
			}
			out.Results[txID] = txResult
		}
	}

	outBytes, err := json.Marshal(out)
	if err != nil {
		logger.Error("failed to marshal ApproveSignRequests output", "error", err)
		return makeJSONResponse(err)
	}

	return C.CString(string(outBytes))
}

//DiscardSignRequest discards a given transaction from transaction queue
//export DiscardSignRequest
func DiscardSignRequest(id *C.char) *C.char {
	err := statusAPI.DiscardSignRequest(C.GoString(id))

	errString := ""
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := DiscardSignRequestResult{
		ID:    C.GoString(id),
		Error: errString,
	}
	outBytes, err := json.Marshal(out)
	if err != nil {
		log.Error("failed to marshal DiscardSignRequest output", "error", err)
		return makeJSONResponse(err)
	}

	return C.CString(string(outBytes))
}

//DiscardSignRequests discards given multiple transactions from transaction queue
//export DiscardSignRequests
func DiscardSignRequests(ids *C.char) *C.char {
	out := DiscardSignRequestsResult{}
	out.Results = make(map[string]DiscardSignRequestResult)

	parsedIDs, err := ParseJSONArray(C.GoString(ids))
	if err != nil {
		out.Results["none"] = DiscardSignRequestResult{
			Error: err.Error(),
		}
	} else {
		txIDs := make([]string, len(parsedIDs))
		for i, id := range parsedIDs {
			txIDs[i] = id
		}

		results := statusAPI.DiscardSignRequests(txIDs)
		for txID, err := range results {
			out.Results[txID] = DiscardSignRequestResult{
				ID:    txID,
				Error: err.Error(),
			}
		}
	}

	outBytes, err := json.Marshal(out)
	if err != nil {
		logger.Error("failed to marshal DiscardSignRequests output", "error", err)
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

	out := APIResponse{
		Error: errString,
	}
	outBytes, _ := json.Marshal(out)

	return C.CString(string(outBytes))
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
		out := NotifyResult{
			Status: err == nil,
			Error:  errString,
		}

		outBytes, err = json.Marshal(out)
		if err != nil {
			logger.Error("failed to marshal Notify output", "error", err)
			outCBytes = makeJSONResponse(err)
			return
		}

		outCBytes = C.CString(string(outBytes))
	}()

	tokens, err := ParseJSONArray(C.GoString(tokensArray))
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
	err := statusAPI.StatusNode().AddPeer(C.GoString(enode))
	return makeJSONResponse(err)
}

// ConnectionChange handles network state changes as reported
// by ReactNative (see https://facebook.github.io/react-native/docs/netinfo.html)
//export ConnectionChange
func ConnectionChange(typ *C.char, expensive C.int) {
	statusAPI.ConnectionChange(C.GoString(typ), expensive == 1)
}

// AppStateChange handles app state changes (background/foreground).
//export AppStateChange
func AppStateChange(state *C.char) {
	statusAPI.AppStateChange(C.GoString(state))
}
