package statusgo

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

var statusBackend = api.NewStatusBackend()

// OpenAccounts opens database and returns accounts list.
func OpenAccounts(datadir string) string {
	statusBackend.UpdateRootDataDir(datadir)
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
	return string(data)
}

// GenerateConfig for status node.
func GenerateConfig(datadir string, networkID int) string {
	config, err := params.NewNodeConfig(datadir, uint64(networkID))
	if err != nil {
		return makeJSONResponse(err)
	}

	outBytes, err := json.Marshal(config)
	if err != nil {
		return makeJSONResponse(err)
	}

	return string(outBytes)
}

// ExtractGroupMembershipSignatures extract public keys from tuples of content/signature.
func ExtractGroupMembershipSignatures(signaturePairsStr string) string {
	var signaturePairs [][2]string

	if err := json.Unmarshal([]byte(signaturePairsStr), &signaturePairs); err != nil {
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

	return string(data)
}

// SignGroupMembership signs a string containing group membership information.
func SignGroupMembership(content string) string {
	signature, err := statusBackend.SignGroupMembership(content)
	if err != nil {
		return makeJSONResponse(err)
	}

	data, err := json.Marshal(struct {
		Signature string `json:"signature"`
	}{Signature: signature})
	if err != nil {
		return makeJSONResponse(err)
	}

	return string(data)
}

// EnableInstallation enables an installation for multi-device sync.
func EnableInstallation(installationID string) string {
	err := statusBackend.EnableInstallation(installationID)
	if err != nil {
		return makeJSONResponse(err)
	}

	data, err := json.Marshal(struct {
		Response string `json:"response"`
	}{Response: "ok"})
	if err != nil {
		return makeJSONResponse(err)
	}

	return string(data)
}

// DisableInstallation disables an installation for multi-device sync.
func DisableInstallation(installationID string) string {
	err := statusBackend.DisableInstallation(installationID)
	if err != nil {
		return makeJSONResponse(err)
	}

	data, err := json.Marshal(struct {
		Response string `json:"response"`
	}{Response: "ok"})
	if err != nil {
		return makeJSONResponse(err)
	}

	return string(data)
}

// ValidateNodeConfig validates config for the Status node.
func ValidateNodeConfig(configJSON string) string {
	var resp APIDetailedResponse

	_, err := params.NewConfigFromJSON(configJSON)

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

	return string(respJSON)
}

// ResetChainData removes chain data from data directory.
func ResetChainData() string {
	api.RunAsync(statusBackend.ResetChainData)
	return makeJSONResponse(nil)
}

// CallRPC calls public APIs via RPC.
func CallRPC(inputJSON string) string {
	resp, err := statusBackend.CallRPC(inputJSON)
	if err != nil {
		return makeJSONResponse(err)
	}
	return resp
}

// CallPrivateRPC calls both public and private APIs via RPC.
func CallPrivateRPC(inputJSON string) string {
	resp, err := statusBackend.CallPrivateRPC(inputJSON)
	if err != nil {
		return makeJSONResponse(err)
	}
	return resp
}

// CreateAccount is equivalent to creating an account from the command line,
// just modified to handle the function arg passing.
func CreateAccount(password string) string {
	info, mnemonic, err := statusBackend.AccountManager().CreateAccount(password)
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
	return string(outBytes)
}

// RecoverAccount re-creates master key using given details.
func RecoverAccount(password, mnemonic string) string {
	info, err := statusBackend.AccountManager().RecoverAccount(password, mnemonic)

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
	return string(outBytes)
}

// StartOnboarding initialize the onboarding with n random accounts
func StartOnboarding(n, mnemonicPhraseLength int) string {
	out := struct {
		Accounts []OnboardingAccount `json:"accounts"`
		Error    string              `json:"error"`
	}{
		Accounts: make([]OnboardingAccount, 0),
	}

	accounts, err := statusBackend.AccountManager().StartOnboarding(n, mnemonicPhraseLength)

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
	return string(outBytes)
}

//ImportOnboardingAccount re-creates and imports an account created during onboarding.
func ImportOnboardingAccount(id, password string) string {
	info, mnemonic, err := statusBackend.AccountManager().ImportOnboardingAccount(id, password)

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
	return string(outBytes)
}

// RemoveOnboarding resets the current onboarding removing from memory all the generated keys.
func RemoveOnboarding() {
	statusBackend.AccountManager().RemoveOnboarding()
}

// VerifyAccountPassword verifies account password.
func VerifyAccountPassword(keyStoreDir, address, password string) string {
	_, err := statusBackend.AccountManager().VerifyAccountPassword(keyStoreDir, address, password)
	return makeJSONResponse(err)
}

// Login loads a key file (for a given address), tries to decrypt it using the password,
// to verify ownership if verified, purges all the previous identities from Whisper,
// and injects verified key as shh identity.
func Login(accountData, password string) string {
	var account multiaccounts.Account
	err := json.Unmarshal([]byte(accountData), &account)
	if err != nil {
		return makeJSONResponse(err)
	}
	api.RunAsync(func() error {
		log.Debug("start a node with account", "address", account.Address)
		err := statusBackend.StartNodeWithAccount(account, password)
		if err != nil {
			log.Error("failed to start a node", "address", account.Address, "error", err)
			return err
		}
		log.Debug("started a node with", "address", account.Address)
		return nil
	})
	return makeJSONResponse(nil)
}

// SaveAccountAndLogin saves account in status-go database..
func SaveAccountAndLogin(accountData, password, configJSON, subaccountData string) string {
	var account multiaccounts.Account
	err := json.Unmarshal([]byte(accountData), &account)
	if err != nil {
		return makeJSONResponse(err)
	}
	var conf params.NodeConfig
	err = json.Unmarshal([]byte(configJSON), &conf)
	if err != nil {
		return makeJSONResponse(err)
	}
	var subaccs []accounts.Account
	err = json.Unmarshal([]byte(subaccountData), &subaccs)
	if err != nil {
		return makeJSONResponse(err)
	}
	api.RunAsync(func() error {
		log.Debug("starting a node, and saving account with configuration", "address", account.Address)
		err := statusBackend.StartNodeWithAccountAndConfig(account, password, &conf, subaccs)
		if err != nil {
			log.Error("failed to start node and save account", "address", account.Address, "error", err)
			return err
		}
		log.Debug("started a node, and saved account", "address", account.Address)
		return nil
	})
	return makeJSONResponse(nil)
}

// InitKeystore initialize keystore before doing any operations with keys.
func InitKeystore(keydir string) string {
	err := statusBackend.AccountManager().InitKeystore(keydir)
	return makeJSONResponse(err)
}

// SaveAccountAndLoginWithKeycard saves account in status-go database..
func SaveAccountAndLoginWithKeycard(accountData, password, configJSON, keyHex string) string {
	var account multiaccounts.Account
	err := json.Unmarshal([]byte(accountData), &account)
	if err != nil {
		return makeJSONResponse(err)
	}
	var conf params.NodeConfig
	err = json.Unmarshal([]byte(configJSON), &conf)
	if err != nil {
		return makeJSONResponse(err)
	}
	api.RunAsync(func() error {
		log.Debug("starting a node, and saving account with configuration", "address", account.Address)
		err := statusBackend.SaveAccountAndStartNodeWithKey(account, &conf, password, keyHex)
		if err != nil {
			log.Error("failed to start node and save account", "address", account.Address, "error", err)
			return err
		}
		log.Debug("started a node, and saved account", "address", account.Address)
		return nil
	})
	return makeJSONResponse(nil)
}

// LoginWithKeycard initializes an account with a chat key and encryption key used for PFS.
// It purges all the previous identities from Whisper, and injects the key as shh identity.
func LoginWithKeycard(accountData, password, keyHex string) string {
	var account multiaccounts.Account
	err := json.Unmarshal([]byte(accountData), &account)
	if err != nil {
		return makeJSONResponse(err)
	}
	api.RunAsync(func() error {
		log.Debug("start a node with account", "address", account.Address)
		err := statusBackend.StartNodeWithKey(account, password, keyHex)
		if err != nil {
			log.Error("failed to start a node", "address", account.Address, "error", err)
			return err
		}
		log.Debug("started a node with", "address", account.Address)
		return nil
	})
	return makeJSONResponse(nil)
}

// Logout is equivalent to clearing whisper identities.
func Logout() string {
	err := statusBackend.Logout()
	if err != nil {
		makeJSONResponse(err)
	}
	return makeJSONResponse(statusBackend.StopNode())
}

// SignMessage unmarshals rpc params {data, address, password} and
// passes them onto backend.SignMessage.
func SignMessage(rpcParams string) string {
	var params personal.SignParams
	err := json.Unmarshal([]byte(rpcParams), &params)
	if err != nil {
		return prepareJSONResponseWithCode(nil, err, codeFailedParseParams)
	}
	result, err := statusBackend.SignMessage(params)
	return prepareJSONResponse(result.String(), err)
}

// SignTypedData unmarshall data into TypedData, validate it and signs with selected account,
// if password matches selected account.
//export SignTypedData
func SignTypedData(data, address, password string) string {
	var typed typeddata.TypedData
	err := json.Unmarshal([]byte(data), &typed)
	if err != nil {
		return prepareJSONResponseWithCode(nil, err, codeFailedParseParams)
	}
	if err := typed.Validate(); err != nil {
		return prepareJSONResponseWithCode(nil, err, codeFailedParseParams)
	}
	result, err := statusBackend.SignTypedData(typed, address, password)
	return prepareJSONResponse(result.String(), err)
}

// HashTypedData unmarshalls data into TypedData, validates it and hashes it.
func HashTypedData(data string) string {
	var typed typeddata.TypedData
	err := json.Unmarshal([]byte(data), &typed)
	if err != nil {
		return prepareJSONResponseWithCode(nil, err, codeFailedParseParams)
	}
	if err := typed.Validate(); err != nil {
		return prepareJSONResponseWithCode(nil, err, codeFailedParseParams)
	}
	result, err := statusBackend.HashTypedData(typed)
	return prepareJSONResponse(result.String(), err)
}

// Recover unmarshals rpc params {signDataString, signedData} and passes
// them onto backend.
func Recover(rpcParams string) string {
	var params personal.RecoverParams
	err := json.Unmarshal([]byte(rpcParams), &params)
	if err != nil {
		return prepareJSONResponseWithCode(nil, err, codeFailedParseParams)
	}
	addr, err := statusBackend.Recover(params)
	return prepareJSONResponse(addr.String(), err)
}

// SendTransaction converts RPC args and calls backend.SendTransaction.
func SendTransaction(txArgsJSON, password string) string {
	var params transactions.SendTxArgs
	err := json.Unmarshal([]byte(txArgsJSON), &params)
	if err != nil {
		return prepareJSONResponseWithCode(nil, err, codeFailedParseParams)
	}
	hash, err := statusBackend.SendTransaction(params, password)
	code := codeUnknown
	if c, ok := errToCodeMap[err]; ok {
		code = c
	}
	return prepareJSONResponseWithCode(hash.String(), err, code)
}

// SendTransactionWithSignature converts RPC args and calls backend.SendTransactionWithSignature
func SendTransactionWithSignature(txArgsJSON, sigString string) string {
	var params transactions.SendTxArgs
	err := json.Unmarshal([]byte(txArgsJSON), &params)
	if err != nil {
		return prepareJSONResponseWithCode(nil, err, codeFailedParseParams)
	}

	sig, err := hex.DecodeString(sigString)
	if err != nil {
		return prepareJSONResponseWithCode(nil, err, codeFailedParseParams)
	}

	hash, err := statusBackend.SendTransactionWithSignature(params, sig)
	code := codeUnknown
	if c, ok := errToCodeMap[err]; ok {
		code = c
	}
	return prepareJSONResponseWithCode(hash.String(), err, code)
}

// HashTransaction validate the transaction and returns new txArgs and the transaction hash.
func HashTransaction(txArgsJSON string) string {
	var params transactions.SendTxArgs
	err := json.Unmarshal([]byte(txArgsJSON), &params)
	if err != nil {
		return prepareJSONResponseWithCode(nil, err, codeFailedParseParams)
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

	return prepareJSONResponseWithCode(result, err, code)
}

// HashMessage calculates the hash of a message to be safely signed by the keycard
// The hash is calulcated as
//   keccak256("\x19Ethereum Signed Message:\n"${message length}${message}).
// This gives context to the signed message and prevents signing of transactions.
func HashMessage(message string) string {
	hash, err := api.HashMessage(message)
	code := codeUnknown
	if c, ok := errToCodeMap[err]; ok {
		code = c
	}
	return prepareJSONResponseWithCode(fmt.Sprintf("0x%x", hash), err, code)
}

// StartCPUProfile runs pprof for CPU.
func StartCPUProfile(dataDir string) string {
	err := profiling.StartCPUProfile(dataDir)
	return makeJSONResponse(err)
}

// StopCPUProfiling stops pprof for cpu.
func StopCPUProfiling() string { //nolint: deadcode
	err := profiling.StopCPUProfile()
	return makeJSONResponse(err)
}

//WriteHeapProfile starts pprof for heap
func WriteHeapProfile(dataDir string) string { //nolint: deadcode
	err := profiling.WriteHeapFile(dataDir)
	return makeJSONResponse(err)
}

func makeJSONResponse(err error) string {
	errString := ""
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		errString = err.Error()
	}

	out := APIResponse{
		Error: errString,
	}
	outBytes, _ := json.Marshal(out)

	return string(outBytes)
}

// UpdateMailservers updates mail servers in status backend.
//export UpdateMailservers
func UpdateMailservers(data string) string {
	var enodes []string
	err := json.Unmarshal([]byte(data), &enodes)
	if err != nil {
		return makeJSONResponse(err)
	}
	err = statusBackend.UpdateMailservers(enodes)
	return makeJSONResponse(err)
}

// GetNodesFromContract returns a list of nodes from a given contract
//export GetNodesFromContract
func GetNodesFromContract(rpcEndpoint string, contractAddress string) string {
	nodes, err := statusBackend.GetNodesFromContract(
		rpcEndpoint,
		contractAddress,
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

	return string(data)
}

// AddPeer adds an enode as a peer.
func AddPeer(enode string) string {
	err := statusBackend.StatusNode().AddPeer(enode)
	return makeJSONResponse(err)
}

// ConnectionChange handles network state changes as reported
// by ReactNative (see https://facebook.github.io/react-native/docs/netinfo.html)
func ConnectionChange(typ string, expensive int) {
	statusBackend.ConnectionChange(typ, expensive == 1)
}

// AppStateChange handles app state changes (background/foreground).
func AppStateChange(state string) {
	statusBackend.AppStateChange(state)
}

// SetMobileSignalHandler setup geth callback to notify about new signal
// used for gomobile builds
func SetMobileSignalHandler(handler SignalHandler) {
	signal.SetMobileSignalHandler(func(data []byte) {
		if len(data) > 0 {
			handler.HandleSignal(string(data))
		}
	})
}

// SetSignalEventCallback setup geth callback to notify about new signal
func SetSignalEventCallback(cb unsafe.Pointer) {
	signal.SetSignalEventCallback(cb)
}

// ExportNodeLogs reads current node log and returns content to a caller.
//export ExportNodeLogs
func ExportNodeLogs() string {
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
	return string(data)
}

// ChaosModeUpdate sets the Chaos Mode on or off.
func ChaosModeUpdate(on bool) string {
	node := statusBackend.StatusNode()
	if node == nil {
		return makeJSONResponse(errors.New("node is not running"))
	}

	err := node.ChaosModeCheckRPCClientsUpstreamURL(on)
	return makeJSONResponse(err)
}

// SignHash exposes vanilla ECDSA signing required for Swarm messages
func SignHash(hexEncodedHash string) string {
	hexEncodedSignature, err := statusBackend.SignHash(hexEncodedHash)
	if err != nil {
		return makeJSONResponse(err)
	}

	return hexEncodedSignature
}

func GenerateAlias(pk string) string {
	// We ignore any error, empty string is considered an error
	name, _ := protocol.GenerateAlias(pk)
	return name
}

func Identicon(pk string) string {
	// We ignore any error, empty string is considered an error
	identicon, _ := protocol.Identicon(pk)
	return identicon
}
