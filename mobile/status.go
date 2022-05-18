package statusgo

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"unsafe"

	validator "gopkg.in/go-playground/validator.v9"

	"github.com/ethereum/go-ethereum/log"
	signercore "github.com/ethereum/go-ethereum/signer/core"

	"github.com/status-im/zxcvbn-go"
	"github.com/status-im/zxcvbn-go/scoring"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/api/multiformat"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/exportlogs"
	"github.com/status-im/status-go/extkeys"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/profiling"
	protocol "github.com/status-im/status-go/protocol"
	identityUtils "github.com/status-im/status-go/protocol/identity"
	"github.com/status-im/status-go/protocol/identity/alias"
	"github.com/status-im/status-go/protocol/identity/colorhash"
	"github.com/status-im/status-go/protocol/identity/emojihash"
	"github.com/status-im/status-go/server"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/services/typeddata"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/transactions"
)

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

// GetNodeConfig returns the current config of the Status node
func GetNodeConfig() string {
	conf, err := statusBackend.GetNodeConfig()
	if err != nil {
		return makeJSONResponse(err)
	}

	respJSON, err := json.Marshal(conf)

	if err != nil {
		return makeJSONResponse(err)
	}

	return string(respJSON)
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

// VerifyAccountPassword verifies account password.
func VerifyAccountPassword(keyStoreDir, address, password string) string {
	_, err := statusBackend.AccountManager().VerifyAccountPassword(keyStoreDir, address, password)
	return makeJSONResponse(err)
}

func VerifyDatabasePassword(keyUID, password string) string {
	return makeJSONResponse(statusBackend.VerifyDatabasePassword(keyUID, password))
}

// MigrateKeyStoreDir migrates key files to a new directory
func MigrateKeyStoreDir(accountData, password, oldDir, newDir string) string {
	var account multiaccounts.Account
	err := json.Unmarshal([]byte(accountData), &account)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = statusBackend.MigrateKeyStoreDir(account, password, oldDir, newDir)
	if err != nil {
		return makeJSONResponse(err)
	}

	return makeJSONResponse(nil)
}

func login(accountData, password, configJSON string) error {
	var account multiaccounts.Account
	err := json.Unmarshal([]byte(accountData), &account)
	if err != nil {
		return err
	}

	var conf params.NodeConfig
	if configJSON != "" {
		err = json.Unmarshal([]byte(configJSON), &conf)
		if err != nil {
			return err
		}
	}

	api.RunAsync(func() error {
		log.Debug("start a node with account", "key-uid", account.KeyUID)
		err := statusBackend.StartNodeWithAccount(account, password, &conf)
		if err != nil {
			log.Error("failed to start a node", "key-uid", account.KeyUID, "error", err)
			return err
		}
		log.Debug("started a node with", "key-uid", account.KeyUID)
		return nil
	})

	return nil
}

// Login loads a key file (for a given address), tries to decrypt it using the password,
// to verify ownership if verified, purges all the previous identities from Whisper,
// and injects verified key as shh identity.
func Login(accountData, password string) string {
	err := login(accountData, password, "")
	if err != nil {
		return makeJSONResponse(err)
	}
	return makeJSONResponse(nil)
}

// Login loads a key file (for a given address), tries to decrypt it using the password,
// to verify ownership if verified, purges all the previous identities from Whisper,
// and injects verified key as shh identity. It then updates the accounts node db configuration
// mergin the values received in the configJSON parameter
func LoginWithConfig(accountData, password, configJSON string) string {
	err := login(accountData, password, configJSON)
	if err != nil {
		return makeJSONResponse(err)
	}
	return makeJSONResponse(nil)
}

// SaveAccountAndLogin saves account in status-go database..
func SaveAccountAndLogin(accountData, password, settingsJSON, configJSON, subaccountData string) string {
	var account multiaccounts.Account
	err := json.Unmarshal([]byte(accountData), &account)
	if err != nil {
		return makeJSONResponse(err)
	}
	var settings settings.Settings
	err = json.Unmarshal([]byte(settingsJSON), &settings)
	if err != nil {
		return makeJSONResponse(err)
	}
	var conf params.NodeConfig
	err = json.Unmarshal([]byte(configJSON), &conf)
	if err != nil {
		return makeJSONResponse(err)
	}
	var subaccs []*accounts.Account
	err = json.Unmarshal([]byte(subaccountData), &subaccs)
	if err != nil {
		return makeJSONResponse(err)
	}

	for _, acc := range subaccs {
		if acc.Chat {
			colorHash, _ := colorhash.GenerateFor(string(acc.PublicKey.Bytes()))
			colorID, _ := identityUtils.ToColorID(string(acc.PublicKey.Bytes()))
			account.ColorHash = colorHash
			account.ColorID = colorID

			break
		}
	}

	api.RunAsync(func() error {
		log.Debug("starting a node, and saving account with configuration", "key-uid", account.KeyUID)
		err := statusBackend.StartNodeWithAccountAndInitialConfig(account, password, settings, &conf, subaccs)
		if err != nil {
			log.Error("failed to start node and save account", "key-uid", account.KeyUID, "error", err)
			return err
		}
		log.Debug("started a node, and saved account", "key-uid", account.KeyUID)
		return nil
	})
	return makeJSONResponse(nil)
}

// DeleteMultiaccount
func DeleteMultiaccount(keyUID, keyStoreDir string) string {
	err := statusBackend.DeleteMulticcount(keyUID, keyStoreDir)
	if err != nil {
		return makeJSONResponse(err)
	}

	return makeJSONResponse(nil)
}

// DeleteImportedKey
func DeleteImportedKey(address, password, keyStoreDir string) string {
	err := statusBackend.DeleteImportedKey(address, password, keyStoreDir)
	if err != nil {
		return makeJSONResponse(err)
	}

	return makeJSONResponse(nil)
}

// InitKeystore initialize keystore before doing any operations with keys.
func InitKeystore(keydir string) string {
	err := statusBackend.AccountManager().InitKeystore(keydir)
	return makeJSONResponse(err)
}

// SaveAccountAndLoginWithKeycard saves account in status-go database..
func SaveAccountAndLoginWithKeycard(accountData, password, settingsJSON, configJSON, subaccountData string, keyHex string) string {
	var account multiaccounts.Account
	err := json.Unmarshal([]byte(accountData), &account)
	if err != nil {
		return makeJSONResponse(err)
	}
	var settings settings.Settings
	err = json.Unmarshal([]byte(settingsJSON), &settings)
	if err != nil {
		return makeJSONResponse(err)
	}
	var conf params.NodeConfig
	err = json.Unmarshal([]byte(configJSON), &conf)
	if err != nil {
		return makeJSONResponse(err)
	}
	var subaccs []*accounts.Account
	err = json.Unmarshal([]byte(subaccountData), &subaccs)
	if err != nil {
		return makeJSONResponse(err)
	}
	api.RunAsync(func() error {
		log.Debug("starting a node, and saving account with configuration", "key-uid", account.KeyUID)
		err := statusBackend.SaveAccountAndStartNodeWithKey(account, password, settings, &conf, subaccs, keyHex)
		if err != nil {
			log.Error("failed to start node and save account", "key-uid", account.KeyUID, "error", err)
			return err
		}
		log.Debug("started a node, and saved account", "key-uid", account.KeyUID)
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
		log.Debug("start a node with account", "key-uid", account.KeyUID)
		err := statusBackend.StartNodeWithKey(account, password, keyHex)
		if err != nil {
			log.Error("failed to start a node", "key-uid", account.KeyUID, "error", err)
			return err
		}
		log.Debug("started a node with", "key-uid", account.KeyUID)
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
//export HashTypedData
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

// SignTypedDataV4 unmarshall data into TypedData, validate it and signs with selected account,
// if password matches selected account.
//export SignTypedDataV4
func SignTypedDataV4(data, address, password string) string {
	var typed signercore.TypedData
	err := json.Unmarshal([]byte(data), &typed)
	if err != nil {
		return prepareJSONResponseWithCode(nil, err, codeFailedParseParams)
	}
	result, err := statusBackend.SignTypedDataV4(typed, address, password)
	return prepareJSONResponse(result.String(), err)
}

// HashTypedDataV4 unmarshalls data into TypedData, validates it and hashes it.
//export HashTypedDataV4
func HashTypedDataV4(data string) string {
	var typed signercore.TypedData
	err := json.Unmarshal([]byte(data), &typed)
	if err != nil {
		return prepareJSONResponseWithCode(nil, err, codeFailedParseParams)
	}
	result, err := statusBackend.HashTypedDataV4(typed)
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
		Hash        types.Hash              `json:"hash"`
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

// StartLocalNotifications
func StartLocalNotifications() string {
	err := statusBackend.StartLocalNotifications()
	return makeJSONResponse(err)
}

// StopLocalNotifications
func StopLocalNotifications() string {
	err := statusBackend.StopLocalNotifications()
	return makeJSONResponse(err)
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

func IsAlias(value string) string {
	return prepareJSONResponse(alias.IsAlias(value), nil)
}

func Identicon(pk string) string {
	// We ignore any error, empty string is considered an error
	identicon, _ := protocol.Identicon(pk)
	return identicon
}

func EmojiHash(pk string) string {
	return prepareJSONResponse(emojihash.GenerateFor(pk))
}

func ColorHash(pk string) string {
	return prepareJSONResponse(colorhash.GenerateFor(pk))
}

func ColorID(pk string) string {
	return prepareJSONResponse(identityUtils.ToColorID(pk))
}

func ValidateMnemonic(mnemonic string) string {
	m := extkeys.NewMnemonic()
	err := m.ValidateMnemonic(mnemonic, extkeys.Language(0))
	return makeJSONResponse(err)
}

// SerializePublicKey compresses an uncompressed multibase encoded multicodec identified EC public key
// For details on usage see specs https://specs.status.im/spec/2#public-key-serialization
func MultiformatSerializePublicKey(key, outBase string) string {
	cpk, err := multiformat.SerializePublicKey(key, outBase)
	if err != nil {
		return makeJSONResponse(err)
	}

	return cpk
}

// DeserializePublicKey decompresses a compressed multibase encoded multicodec identified EC public key
// For details on usage see specs https://specs.status.im/spec/2#public-key-serialization
func MultiformatDeserializePublicKey(key, outBase string) string {
	pk, err := multiformat.DeserializePublicKey(key, outBase)
	if err != nil {
		return makeJSONResponse(err)
	}

	return pk
}

// ExportUnencryptedDatabase exports the database unencrypted to the given path
func ExportUnencryptedDatabase(accountData, password, databasePath string) string {
	var account multiaccounts.Account
	err := json.Unmarshal([]byte(accountData), &account)
	if err != nil {
		return makeJSONResponse(err)
	}
	err = statusBackend.ExportUnencryptedDatabase(account, password, databasePath)
	if err != nil {
		return makeJSONResponse(err)
	}
	return makeJSONResponse(nil)
}

// ImportUnencryptedDatabase imports the database unencrypted to the given directory
func ImportUnencryptedDatabase(accountData, password, databasePath string) string {
	var account multiaccounts.Account
	err := json.Unmarshal([]byte(accountData), &account)
	if err != nil {
		return makeJSONResponse(err)
	}
	err = statusBackend.ImportUnencryptedDatabase(account, password, databasePath)
	if err != nil {
		return makeJSONResponse(err)
	}
	return makeJSONResponse(nil)
}

func ChangeDatabasePassword(keyUID, password, newPassword string) string {
	err := statusBackend.ChangeDatabasePassword(keyUID, password, newPassword)
	if err != nil {
		return makeJSONResponse(err)
	}
	return makeJSONResponse(nil)
}

func ConvertToKeycardAccount(keyStoreDir, accountData, settingsJSON, password, newPassword string) string {
	var account multiaccounts.Account
	err := json.Unmarshal([]byte(accountData), &account)
	if err != nil {
		return makeJSONResponse(err)
	}
	var settings settings.Settings
	err = json.Unmarshal([]byte(settingsJSON), &settings)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = statusBackend.ConvertToKeycardAccount(keyStoreDir, account, settings, password, newPassword)
	if err != nil {
		return makeJSONResponse(err)
	}
	return makeJSONResponse(nil)
}

func ImageServerTLSCert() string {
	cert, err := server.PublicTLSCert()

	if err != nil {
		return makeJSONResponse(err)
	}

	return cert
}

type GetPasswordStrengthRequest struct {
	Password   string   `json:"password"`
	UserInputs []string `json:"userInputs"`
}

type PasswordScoreResponse struct {
	Score int `json:"score"`
}

// GetPasswordStrength uses zxcvbn module and generates a JSON containing information about the quality of the given password
// (Entropy, CrackTime, CrackTimeDisplay, Score, MatchSequence and CalcTime).
// userInputs argument can be whatever list of strings like user's personal info or site-specific vocabulary that zxcvbn will
// make use to determine the result.
// For more details on usage see https://github.com/status-im/zxcvbn-go
func GetPasswordStrength(paramsJSON string) string {
	var requestParams GetPasswordStrengthRequest

	err := json.Unmarshal([]byte(paramsJSON), &requestParams)
	if err != nil {
		return makeJSONResponse(err)
	}

	data, err := json.Marshal(zxcvbn.PasswordStrength(requestParams.Password, requestParams.UserInputs))
	if err != nil {
		return makeJSONResponse(fmt.Errorf("Error marshalling to json: %v", err))
	}
	return string(data)
}

// GetPasswordStrengthScore uses zxcvbn module and gets the score information about the given password.
// userInputs argument can be whatever list of strings like user's personal info or site-specific vocabulary that zxcvbn will
// make use to determine the result.
// For more details on usage see https://github.com/status-im/zxcvbn-go
func GetPasswordStrengthScore(paramsJSON string) string {
	var requestParams GetPasswordStrengthRequest
	var quality scoring.MinEntropyMatch

	err := json.Unmarshal([]byte(paramsJSON), &requestParams)
	if err != nil {
		return makeJSONResponse(err)
	}

	quality = zxcvbn.PasswordStrength(requestParams.Password, requestParams.UserInputs)

	data, err := json.Marshal(PasswordScoreResponse{
		Score: quality.Score,
	})
	if err != nil {
		return makeJSONResponse(fmt.Errorf("Error marshalling to json: %v", err))
	}
	return string(data)
}

func SwitchFleet(fleet string, configJSON string) string {
	var conf params.NodeConfig
	if configJSON != "" {
		err := json.Unmarshal([]byte(configJSON), &conf)
		if err != nil {
			return makeJSONResponse(err)
		}
	}

	conf.ClusterConfig.Fleet = fleet

	err := statusBackend.SwitchFleet(fleet, &conf)

	return makeJSONResponse(err)
}

func GenerateImages(filepath string, aX, aY, bX, bY int) string {
	iis, err := images.GenerateIdentityImages(filepath, aX, aY, bX, bY)
	if err != nil {
		return makeJSONResponse(err)
	}

	data, err := json.Marshal(iis)
	if err != nil {
		return makeJSONResponse(fmt.Errorf("Error marshalling to json: %v", err))
	}
	return string(data)
}
