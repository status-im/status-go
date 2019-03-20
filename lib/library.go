package main

// #include <stdlib.h>
import "C"
import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/exportlogs"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/profiling"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/services/typeddata"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/transactions"
	validator "gopkg.in/go-playground/validator.v9"
)

// All general log messages in this package should be routed through this logger.
var logger = log.New("package", "status-go/lib")

//StartNode - start Status node
//export StartNode
func StartNode(configJSON *C.char) *C.char {
	config, err := params.NewConfigFromJSON(C.GoString(configJSON))
	if err != nil {
		return makeJSONResponse(err)
	}

	if err := logutils.OverrideRootLogWithConfig(config, false); err != nil {
		return makeJSONResponse(err)
	}

	api.RunAsync(func() error { return statusBackend.StartNode(config) })

	return makeJSONResponse(nil)
}

//StopNode - stop status node
//export StopNode
func StopNode() *C.char {
	api.RunAsync(statusBackend.StopNode)
	return makeJSONResponse(nil)
}

// Create an X3DH bundle
//export CreateContactCode
func CreateContactCode() *C.char {
	bundle, err := statusBackend.CreateContactCode()
	if err != nil {
		return makeJSONResponse(err)
	}

	cstr := C.CString(bundle)

	return cstr
}

//export ProcessContactCode
func ProcessContactCode(bundleString *C.char) *C.char {
	err := statusBackend.ProcessContactCode(C.GoString(bundleString))
	if err != nil {
		return makeJSONResponse(err)
	}

	return nil
}

// Get an X3DH bundle
//export GetContactCode
func GetContactCode(identityString *C.char) *C.char {
	bundle, err := statusBackend.GetContactCode(C.GoString(identityString))
	if err != nil {
		return makeJSONResponse(err)
	}

	data, err := json.Marshal(struct {
		ContactCode string `json:"code"`
	}{ContactCode: bundle})
	if err != nil {
		return makeJSONResponse(err)
	}

	return C.CString(string(data))
}

//export ExtractIdentityFromContactCode
func ExtractIdentityFromContactCode(bundleString *C.char) *C.char {
	bundle := C.GoString(bundleString)

	identity, err := statusBackend.ExtractIdentityFromContactCode(bundle)
	if err != nil {
		return makeJSONResponse(err)
	}

	if err := statusBackend.ProcessContactCode(bundle); err != nil {
		return makeJSONResponse(err)
	}

	data, err := json.Marshal(struct {
		Identity string `json:"identity"`
	}{Identity: identity})
	if err != nil {
		return makeJSONResponse(err)
	}

	return C.CString(string(data))
}

// ExtractGroupMembershipSignatures extract public keys from tuples of content/signature
//export ExtractGroupMembershipSignatures
func ExtractGroupMembershipSignatures(signaturePairsStr *C.char) *C.char {
	var signaturePairs [][2]string

	if err := json.Unmarshal([]byte(C.GoString(signaturePairsStr)), &signaturePairs); err != nil {
		return makeJSONResponse(err)
	}

	identities, err := statusBackend.ExtractGroupMembershipSignatures(signaturePairs)
	if err != nil {
		return makeJSONResponse(err)
	}

	data, err := json.Marshal(struct {
		Identities []string `json:"identities"`
	}{Identities: identities})
	if err != nil {
		return makeJSONResponse(err)
	}

	return C.CString(string(data))
}

// Sign signs a string containing group membership information
//export SignGroupMembership
func SignGroupMembership(content *C.char) *C.char {
	signature, err := statusBackend.SignGroupMembership(C.GoString(content))
	if err != nil {
		return makeJSONResponse(err)
	}

	data, err := json.Marshal(struct {
		Signature string `json:"signature"`
	}{Signature: signature})
	if err != nil {
		return makeJSONResponse(err)
	}

	return C.CString(string(data))
}

// EnableInstallation enables an installation for multi-device sync.
//export EnableInstallation
func EnableInstallation(installationID *C.char) *C.char {
	err := statusBackend.EnableInstallation(C.GoString(installationID))
	if err != nil {
		return makeJSONResponse(err)
	}

	data, err := json.Marshal(struct {
		Response string `json:"response"`
	}{Response: "ok"})
	if err != nil {
		return makeJSONResponse(err)
	}

	return C.CString(string(data))
}

// DisableInstallation disables an installation for multi-device sync.
//export DisableInstallation
func DisableInstallation(installationID *C.char) *C.char {
	err := statusBackend.DisableInstallation(C.GoString(installationID))
	if err != nil {
		return makeJSONResponse(err)
	}

	data, err := json.Marshal(struct {
		Response string `json:"response"`
	}{Response: "ok"})
	if err != nil {
		return makeJSONResponse(err)
	}

	return C.CString(string(data))
}

//ValidateNodeConfig validates config for status node
//export ValidateNodeConfig
func ValidateNodeConfig(configJSON *C.char) *C.char {
	var resp APIDetailedResponse

	_, err := params.NewConfigFromJSON(C.GoString(configJSON))

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
	api.RunAsync(statusBackend.ResetChainData)
	return makeJSONResponse(nil)
}

//CallRPC calls public APIs via RPC
//export CallRPC
func CallRPC(inputJSON *C.char) *C.char {
	outputJSON, err := statusBackend.CallRPC(C.GoString(inputJSON))
	if err != nil {
		return makeJSONResponse(err)
	}
	return C.CString(outputJSON)
}

//CallPrivateRPC calls both public and private APIs via RPC
//export CallPrivateRPC
func CallPrivateRPC(inputJSON *C.char) *C.char {
	outputJSON, err := statusBackend.CallPrivateRPC(C.GoString(inputJSON))
	if err != nil {
		return makeJSONResponse(err)
	}
	return C.CString(outputJSON)
}

//CreateAccount is equivalent to creating an account from the command line,
// just modified to handle the function arg passing
//export CreateAccount
func CreateAccount(password *C.char) *C.char {
	info, mnemonic, err := statusBackend.AccountManager().CreateAccount(C.GoString(password))

	errString := ""
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := AccountInfo{
		Address:       info.WalletAddress,
		PubKey:        info.WalletPubKey,
		WalletAddress: info.WalletAddress,
		WalletPubKey:  info.WalletPubKey,
		ChatAddress:   info.ChatAddress,
		ChatPubKey:    info.ChatPubKey,
		Mnemonic:      mnemonic,
		Error:         errString,
	}
	outBytes, _ := json.Marshal(out)
	return C.CString(string(outBytes))
}

//CreateChildAccount creates sub-account
//export CreateChildAccount
func CreateChildAccount(parentAddress, password *C.char) *C.char {
	address, pubKey, err := statusBackend.AccountManager().CreateChildAccount(C.GoString(parentAddress), C.GoString(password))

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
	info, err := statusBackend.AccountManager().RecoverAccount(C.GoString(password), C.GoString(mnemonic))

	errString := ""
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := AccountInfo{
		Address:       info.WalletAddress,
		PubKey:        info.WalletPubKey,
		WalletAddress: info.WalletAddress,
		WalletPubKey:  info.WalletPubKey,
		ChatAddress:   info.ChatAddress,
		ChatPubKey:    info.ChatPubKey,
		Mnemonic:      C.GoString(mnemonic),
		Error:         errString,
	}
	outBytes, _ := json.Marshal(out)
	return C.CString(string(outBytes))
}

//VerifyAccountPassword verifies account password
//export VerifyAccountPassword
func VerifyAccountPassword(keyStoreDir, address, password *C.char) *C.char {
	_, err := statusBackend.AccountManager().VerifyAccountPassword(C.GoString(keyStoreDir), C.GoString(address), C.GoString(password))
	return makeJSONResponse(err)
}

//Login loads a key file (for a given address), tries to decrypt it using the password, to verify ownership
// if verified, purges all the previous identities from Whisper, and injects verified key as shh identity
//export Login
func Login(address, password *C.char) *C.char {
	err := statusBackend.SelectAccount(C.GoString(address), C.GoString(address), C.GoString(password))
	return makeJSONResponse(err)
}

// LoginWithKeycard initializes an account with a chat key and encryption key used for PFS.
// It purges all the previous identities from Whisper, and injects the key as shh identity.
//export LoginWithKeycard
func LoginWithKeycard(chatKeyData, encryptionKeyData *C.char) *C.char {
	err := statusBackend.InjectChatAccount(C.GoString(chatKeyData), C.GoString(encryptionKeyData))
	return makeJSONResponse(err)
}

//Logout is equivalent to clearing whisper identities
//export Logout
func Logout() *C.char {
	err := statusBackend.Logout()
	return makeJSONResponse(err)
}

// SignMessage unmarshals rpc params {data, address, password} and passes
// them onto backend.SignMessage
//export SignMessage
func SignMessage(rpcParams *C.char) *C.char {
	var params personal.SignParams
	err := json.Unmarshal([]byte(C.GoString(rpcParams)), &params)
	if err != nil {
		return C.CString(prepareJSONResponseWithCode(nil, err, codeFailedParseParams))
	}
	result, err := statusBackend.SignMessage(params)
	return C.CString(prepareJSONResponse(result.String(), err))
}

// Recover unmarshals rpc params {signDataString, signedData} and passes
// them onto backend.
//export Recover
func Recover(rpcParams *C.char) *C.char {
	var params personal.RecoverParams
	err := json.Unmarshal([]byte(C.GoString(rpcParams)), &params)
	if err != nil {
		return C.CString(prepareJSONResponseWithCode(nil, err, codeFailedParseParams))
	}
	addr, err := statusBackend.Recover(params)
	return C.CString(prepareJSONResponse(addr.String(), err))
}

// SendTransaction converts RPC args and calls backend.SendTransaction
//export SendTransaction
func SendTransaction(txArgsJSON, password *C.char) *C.char {
	var params transactions.SendTxArgs
	err := json.Unmarshal([]byte(C.GoString(txArgsJSON)), &params)
	if err != nil {
		return C.CString(prepareJSONResponseWithCode(nil, err, codeFailedParseParams))
	}
	hash, err := statusBackend.SendTransaction(params, C.GoString(password))
	code := codeUnknown
	if c, ok := errToCodeMap[err]; ok {
		code = c
	}
	return C.CString(prepareJSONResponseWithCode(hash.String(), err, code))
}

// SendTransactionWithSignature converts RPC args and calls backend.SendTransactionWithSignature
//export SendTransactionWithSignature
func SendTransactionWithSignature(txArgsJSON, sigString *C.char) *C.char {
	var params transactions.SendTxArgs
	err := json.Unmarshal([]byte(C.GoString(txArgsJSON)), &params)
	if err != nil {
		return C.CString(prepareJSONResponseWithCode(nil, err, codeFailedParseParams))
	}

	sig, err := hex.DecodeString(C.GoString(sigString))
	if err != nil {
		return C.CString(prepareJSONResponseWithCode(nil, err, codeFailedParseParams))
	}

	hash, err := statusBackend.SendTransactionWithSignature(params, sig)
	code := codeUnknown
	if c, ok := errToCodeMap[err]; ok {
		code = c
	}
	return C.CString(prepareJSONResponseWithCode(hash.String(), err, code))
}

// HashTransaction validate the transaction and returns new txArgs and the transaction hash.
//export HashTransaction
func HashTransaction(txArgsJSON *C.char) *C.char {
	var params transactions.SendTxArgs
	err := json.Unmarshal([]byte(C.GoString(txArgsJSON)), &params)
	if err != nil {
		return C.CString(prepareJSONResponseWithCode(nil, err, codeFailedParseParams))
	}

	newTxArgs, hash, err := statusBackend.HashTransaction(params)
	code := codeUnknown
	if c, ok := errToCodeMap[err]; ok {
		code = c
	}

	result := struct {
		Transaction transactions.SendTxArgs `json:"transaction"`
		Hash        common.Hash             `json:"hash"`
	}{
		Transaction: newTxArgs,
		Hash:        hash,
	}

	return C.CString(prepareJSONResponseWithCode(result, err, code))
}

// SignTypedData unmarshall data into TypedData, validate it and signs with selected account,
// if password matches selected account.
//export SignTypedData
func SignTypedData(data, password *C.char) *C.char {
	var typed typeddata.TypedData
	err := json.Unmarshal([]byte(C.GoString(data)), &typed)
	if err != nil {
		return C.CString(prepareJSONResponseWithCode(nil, err, codeFailedParseParams))
	}
	if err := typed.Validate(); err != nil {
		return C.CString(prepareJSONResponseWithCode(nil, err, codeFailedParseParams))
	}
	result, err := statusBackend.SignTypedData(typed, C.GoString(password))
	return C.CString(prepareJSONResponse(result.String(), err))
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

// SendDataNotification sends push notifications by given tokens.
// dataPayloadJSON is a JSON string that looks like this:
// {
//	"data": {
//		"msg-v2": {
//			"from": "0x2cea3bd5", // hash of sender (first 10 characters/4 bytes of sha3 hash)
//			"to": "0xb1f89744", // hash of recipient (first 10 characters/4 bytes of sha3 hash)
//			"id": "0x872653ad", // message ID hash (first 10 characters/4 bytes of sha3 hash)
//		}
//	}
// }
//export SendDataNotification
func SendDataNotification(dataPayloadJSON, tokensArray *C.char) (outCBytes *C.char) {
	var (
		err      error
		outBytes []byte
	)
	errString := ""

	defer func() {
		out := SendDataNotificationResult{
			Status: err == nil,
			Error:  errString,
		}

		outBytes, err = json.Marshal(out)
		if err != nil {
			logger.Error("failed to marshal SendDataNotification output", "error", err)
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

	err = statusBackend.SendDataNotification(C.GoString(dataPayloadJSON), tokens...)
	if err != nil {
		errString = err.Error()
		return
	}

	return
}

// UpdateMailservers updates mail servers in status backend.
//export UpdateMailservers
func UpdateMailservers(data *C.char) *C.char {
	var enodes []string
	err := json.Unmarshal([]byte(C.GoString(data)), &enodes)
	if err != nil {
		return makeJSONResponse(err)
	}
	err = statusBackend.UpdateMailservers(enodes)
	return makeJSONResponse(err)
}

// AddPeer adds an enode as a peer.
//export AddPeer
func AddPeer(enode *C.char) *C.char {
	err := statusBackend.StatusNode().AddPeer(C.GoString(enode))
	return makeJSONResponse(err)
}

// ConnectionChange handles network state changes as reported
// by ReactNative (see https://facebook.github.io/react-native/docs/netinfo.html)
//export ConnectionChange
func ConnectionChange(typ *C.char, expensive C.int) {
	statusBackend.ConnectionChange(C.GoString(typ), expensive == 1)
}

// AppStateChange handles app state changes (background/foreground).
//export AppStateChange
func AppStateChange(state *C.char) {
	statusBackend.AppStateChange(C.GoString(state))
}

// SetSignalEventCallback setup geth callback to notify about new signal
//export SetSignalEventCallback
func SetSignalEventCallback(cb unsafe.Pointer) {
	signal.SetSignalEventCallback(cb)
}

// ExportNodeLogs reads current node log and returns content to a caller.
//export ExportNodeLogs
func ExportNodeLogs() *C.char {
	node := statusBackend.StatusNode()
	if node == nil {
		return makeJSONResponse(errors.New("node is not running"))
	}
	config := node.Config()
	if config == nil {
		return makeJSONResponse(errors.New("config and log file are not available"))
	}
	data, err := json.Marshal(exportlogs.ExportFromBaseFile(config.LogFile))
	if err != nil {
		return makeJSONResponse(fmt.Errorf("error marshalling to json: %v", err))
	}
	return C.CString(string(data))
}

// ChaosModeSetUpstreamURL changes the upstream RPC client URL, if enabled.
// Additionally, if the custom URL is infura, it changes it to https://httpstat.us/500.
//export ChaosModeSetUpstreamURL
func ChaosModeSetUpstreamURL(url *C.char) *C.char {
	node := statusBackend.StatusNode()
	if node == nil {
		return makeJSONResponse(errors.New("node is not running"))
	}

	err := node.ChaosModeChangeRPCClientsUpstreamURL(C.GoString(url))
	return makeJSONResponse(err)
}
