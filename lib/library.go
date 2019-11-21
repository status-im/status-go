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
	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/exportlogs"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/profiling"
	protocol "github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/services/typeddata"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/transactions"
	validator "gopkg.in/go-playground/validator.v9"
)

// OpenAccounts opens database and returns accounts list.
//export OpenAccounts
func OpenAccounts(datadir *C.char) *C.char {
	statusBackend.UpdateRootDataDir(C.GoString(datadir))
	err := statusBackend.OpenAccounts()
	if err != nil {
		return makeJSONResponse(err)
	}
	accs, err := statusBackend.GetAccounts()
	if err != nil {
		return makeJSONResponse(err)
	}
	data, err := json.Marshal(accs)
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

// StartOnboarding initialize the onboarding with n random accounts
//export StartOnboarding
func StartOnboarding(n, mnemonicPhraseLength C.int) *C.char {
	out := struct {
		Accounts []OnboardingAccount `json:"accounts"`
		Error    string              `json:"error"`
	}{
		Accounts: make([]OnboardingAccount, 0),
	}

	accounts, err := statusBackend.AccountManager().StartOnboarding(int(n), int(mnemonicPhraseLength))

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		out.Error = err.Error()
	}

	if err == nil {
		for _, account := range accounts {
			out.Accounts = append(out.Accounts, OnboardingAccount{
				ID:            account.ID,
				Address:       account.Info.WalletAddress,
				PubKey:        account.Info.WalletPubKey,
				WalletAddress: account.Info.WalletAddress,
				WalletPubKey:  account.Info.WalletPubKey,
				ChatAddress:   account.Info.ChatAddress,
				ChatPubKey:    account.Info.ChatPubKey,
			})
		}
	}

	outBytes, _ := json.Marshal(out)
	return C.CString(string(outBytes))
}

// ImportOnboardingAccount re-creates and imports an account created during onboarding.
//export ImportOnboardingAccount
func ImportOnboardingAccount(id, password *C.char) *C.char {
	info, mnemonic, err := statusBackend.AccountManager().ImportOnboardingAccount(C.GoString(id), C.GoString(password))

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

// RemoveOnboarding resets the current onboarding removing from memory all the generated keys.
//export RemoveOnboarding
func RemoveOnboarding() {
	statusBackend.AccountManager().RemoveOnboarding()
}

//VerifyAccountPassword verifies account password
//export VerifyAccountPassword
func VerifyAccountPassword(keyStoreDir, address, password *C.char) *C.char {
	_, err := statusBackend.AccountManager().VerifyAccountPassword(C.GoString(keyStoreDir), C.GoString(address), C.GoString(password))
	return makeJSONResponse(err)
}

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

//Login loads a key file (for a given address), tries to decrypt it using the password, to verify ownership
// if verified, purges all the previous identities from Whisper, and injects verified key as shh identity
//export Login
func Login(accountData, password *C.char) *C.char {
	data, pass := C.GoString(accountData), C.GoString(password)
	var account multiaccounts.Account
	err := json.Unmarshal([]byte(data), &account)
	if err != nil {
		return makeJSONResponse(err)
	}
	api.RunAsync(func() error { return statusBackend.StartNodeWithAccount(account, pass) })
	return makeJSONResponse(nil)
}

// SaveAccountAndLogin saves account in status-go database..
//export SaveAccountAndLogin
func SaveAccountAndLogin(accountData, password, configJSON, subaccountData *C.char) *C.char {
	data, confJSON, subData := C.GoString(accountData), C.GoString(configJSON), C.GoString(subaccountData)
	var account multiaccounts.Account
	err := json.Unmarshal([]byte(data), &account)
	if err != nil {
		return makeJSONResponse(err)
	}
	conf := params.NodeConfig{}
	err = json.Unmarshal([]byte(confJSON), &conf)
	if err != nil {
		return makeJSONResponse(err)
	}
	var subaccs []accounts.Account
	err = json.Unmarshal([]byte(subData), &subaccs)
	if err != nil {
		return makeJSONResponse(err)
	}
	api.RunAsync(func() error {
		return statusBackend.StartNodeWithAccountAndConfig(account, C.GoString(password), &conf, subaccs)
	})
	return makeJSONResponse(nil)
}

// InitKeystore initialize keystore before doing any operations with keys.
//export InitKeystore
func InitKeystore(keydir *C.char) *C.char {
	err := statusBackend.AccountManager().InitKeystore(C.GoString(keydir))
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
	if err != nil {
		makeJSONResponse(err)
	}
	return makeJSONResponse(statusBackend.StopNode())
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

// HashMessage calculates the hash of a message to be safely signed by the keycard
// The hash is calulcated as
//   keccak256("\x19Ethereum Signed Message:\n"${message length}${message}).
// This gives context to the signed message and prevents signing of transactions.
//export HashMessage
func HashMessage(message *C.char) *C.char {
	hash, err := api.HashMessage(C.GoString(message))
	code := codeUnknown
	if c, ok := errToCodeMap[err]; ok {
		code = c
	}
	return C.CString(prepareJSONResponseWithCode(fmt.Sprintf("0x%x", hash), err, code))
}

// SignTypedData unmarshall data into TypedData, validate it and signs with selected account,
// if password matches selected account.
//export SignTypedData
func SignTypedData(data, address, password *C.char) *C.char {
	var typed typeddata.TypedData
	err := json.Unmarshal([]byte(C.GoString(data)), &typed)
	if err != nil {
		return C.CString(prepareJSONResponseWithCode(nil, err, codeFailedParseParams))
	}
	if err := typed.Validate(); err != nil {
		return C.CString(prepareJSONResponseWithCode(nil, err, codeFailedParseParams))
	}
	result, err := statusBackend.SignTypedData(typed, C.GoString(address), C.GoString(password))
	return C.CString(prepareJSONResponse(result.String(), err))
}

// HashTypedData unmarshalls data into TypedData, validates it and hashes it.
//export HashTypedData
func HashTypedData(data *C.char) *C.char {
	var typed typeddata.TypedData
	err := json.Unmarshal([]byte(C.GoString(data)), &typed)
	if err != nil {
		return C.CString(prepareJSONResponseWithCode(nil, err, codeFailedParseParams))
	}
	if err := typed.Validate(); err != nil {
		return C.CString(prepareJSONResponseWithCode(nil, err, codeFailedParseParams))
	}
	result, err := statusBackend.HashTypedData(typed)
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

// ChaosModeUpdate changes the URL of the upstream RPC client.
//export ChaosModeUpdate
func ChaosModeUpdate(on C.int) *C.char {
	node := statusBackend.StatusNode()
	if node == nil {
		return makeJSONResponse(errors.New("node is not running"))
	}

	err := node.ChaosModeCheckRPCClientsUpstreamURL(on == 1)
	return makeJSONResponse(err)
}

// GetNodesFromContract returns a list of nodes from a contract
//export GetNodesFromContract
func GetNodesFromContract(rpcEndpoint *C.char, contractAddress *C.char) *C.char {
	nodes, err := statusBackend.GetNodesFromContract(
		C.GoString(rpcEndpoint),
		C.GoString(contractAddress),
	)
	if err != nil {
		return makeJSONResponse(err)
	}

	data, err := json.Marshal(struct {
		Nodes []string `json:"result"`
	}{Nodes: nodes})
	if err != nil {
		return makeJSONResponse(err)
	}

	return C.CString(string(data))
}

// SignHash exposes vanilla ECDSA signing required for Swarm messages
//export SignHash
func SignHash(hexEncodedHash *C.char) *C.char {
	hexEncodedSignature, err := statusBackend.SignHash(
		C.GoString(hexEncodedHash),
	)

	if err != nil {
		return makeJSONResponse(err)
	}

	return C.CString(hexEncodedSignature)
}

// GenerateAlias returns a 3 random name words given the pk string 0x prefixed
//export GenerateAlias
func GenerateAlias(pk string) *C.char {
	// We ignore any error, empty string is considered an error
	name, _ := protocol.GenerateAlias(pk)
	return C.CString(name)
}

// Identicon returns the base64 identicon
//export Identicon
func Identicon(pk string) *C.char {
	// We ignore any error, empty string is considered an error
	identicon, _ := protocol.Identicon(pk)
	return C.CString(identicon)
}
