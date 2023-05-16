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
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/status-im/zxcvbn-go"
	"github.com/status-im/zxcvbn-go/scoring"

	abi_spec "github.com/status-im/status-go/abi-spec"
	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/api/multiformat"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/exportlogs"
	"github.com/status-im/status-go/extkeys"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/profiling"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/protocol/common"
	identityUtils "github.com/status-im/status-go/protocol/identity"
	"github.com/status-im/status-go/protocol/identity/alias"
	"github.com/status-im/status-go/protocol/identity/colorhash"
	"github.com/status-im/status-go/protocol/identity/emojihash"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/server"
	"github.com/status-im/status-go/server/pairing"
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

func CreateAccountAndLogin(requestJSON string) string {
	var request requests.CreateAccount
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = request.Validate()
	if err != nil {
		return makeJSONResponse(err)
	}

	api.RunAsync(func() error {
		log.Debug("starting a node and creating config")
		err := statusBackend.CreateAccountAndLogin(&request)
		if err != nil {
			log.Error("failed to create account", "error", err)
			return err
		}
		log.Debug("started a node, and created account")
		return nil
	})
	return makeJSONResponse(nil)
}

func RestoreAccountAndLogin(requestJSON string) string {
	var request requests.RestoreAccount
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = request.Validate()
	if err != nil {
		return makeJSONResponse(err)
	}

	api.RunAsync(func() error {
		log.Debug("starting a node and restoring account")
		err := statusBackend.RestoreAccountAndLogin(&request)
		if err != nil {
			log.Error("failed to restore account", "error", err)
			return err
		}
		log.Debug("started a node, and restored account")
		return nil
	})
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
	err := statusBackend.DeleteMultiaccount(keyUID, keyStoreDir)
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
//
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
//
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
//
//export SignTypedDataV4
func SignTypedDataV4(data, address, password string) string {
	var typed apitypes.TypedData
	err := json.Unmarshal([]byte(data), &typed)
	if err != nil {
		return prepareJSONResponseWithCode(nil, err, codeFailedParseParams)
	}
	result, err := statusBackend.SignTypedDataV4(typed, address, password)
	return prepareJSONResponse(result.String(), err)
}

// HashTypedDataV4 unmarshalls data into TypedData, validates it and hashes it.
//
//export HashTypedDataV4
func HashTypedDataV4(data string) string {
	var typed apitypes.TypedData
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

// SendTransactionWithChainID converts RPC args and calls backend.SendTransactionWithChainID.
func SendTransactionWithChainID(chainID int, txArgsJSON, password string) string {
	var params transactions.SendTxArgs
	err := json.Unmarshal([]byte(txArgsJSON), &params)
	if err != nil {
		return prepareJSONResponseWithCode(nil, err, codeFailedParseParams)
	}
	hash, err := statusBackend.SendTransactionWithChainID(uint64(chainID), params, password)
	code := codeUnknown
	if c, ok := errToCodeMap[err]; ok {
		code = c
	}
	return prepareJSONResponseWithCode(hash.String(), err, code)
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
//
//	keccak256("\x19Ethereum Signed Message:\n"${message length}${message}).
//
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

// WriteHeapProfile starts pprof for heap
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
//
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
	if err != nil {
		return makeJSONResponse(err)
	}

	keyUID, err := statusBackend.GetKeyUIDByMnemonic(mnemonic)

	if err != nil {
		return makeJSONResponse(err)
	}

	response := &APIKeyUIDResponse{KeyUID: keyUID}
	data, err := json.Marshal(response)
	if err != nil {
		return makeJSONResponse(err)
	}
	return string(data)
}

// DecompressPublicKey decompresses 33-byte compressed format to uncompressed 65-byte format.
func DecompressPublicKey(key string) string {
	decoded, err := types.DecodeHex(key)
	if err != nil {
		return makeJSONResponse(err)
	}
	const compressionBytesNumber = 33
	if len(decoded) != compressionBytesNumber {
		return makeJSONResponse(errors.New("key is not 33 bytes long"))
	}
	pubKey, err := crypto.DecompressPubkey(decoded)
	if err != nil {
		return makeJSONResponse(err)
	}
	return types.EncodeHex(crypto.FromECDSAPub(pubKey))
}

// CompressPublicKey compresses uncompressed 65-byte format to 33-byte compressed format.
func CompressPublicKey(key string) string {
	pubKey, err := common.HexToPubkey(key)
	if err != nil {
		return makeJSONResponse(err)
	}
	return types.EncodeHex(crypto.CompressPubkey(pubKey))
}

// SerializeLegacyKey compresses an old format public key (0x04...) to the new one zQ...
func SerializeLegacyKey(key string) string {
	cpk, err := multiformat.SerializeLegacyKey(key)
	if err != nil {
		return makeJSONResponse(err)
	}

	return cpk
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

func ChangeDatabasePassword(KeyUID, password, newPassword string) string {
	err := statusBackend.ChangeDatabasePassword(KeyUID, password, newPassword)
	if err != nil {
		return makeJSONResponse(err)
	}
	return makeJSONResponse(nil)
}

func ConvertToKeycardAccount(accountData, settingsJSON, keycardUID, password, newPassword string) string {
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

	err = statusBackend.ConvertToKeycardAccount(account, settings, keycardUID, password, newPassword)
	if err != nil {
		return makeJSONResponse(err)
	}
	return makeJSONResponse(nil)
}

func ConvertToRegularAccount(mnemonic, currPassword, newPassword string) string {
	err := statusBackend.ConvertToRegularAccount(mnemonic, currPassword, newPassword)
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

// StartSearchForLocalPairingPeers starts a UDP multicast beacon that both listens for and broadcasts to LAN peers
// on discovery the beacon will emit a signal with the details of the discovered peer.
//
// Currently, beacons are configured to search for 2 minutes pinging the network every 500 ms;
//   - If no peer discovery is made before this time elapses the operation will terminate.
//   - If a peer is discovered the pairing.PeerNotifier will terminate operation after 5 seconds, giving the peer
//     reasonable time to discover this device.
//
// Peer details are represented by a json.Marshal peers.LocalPairingPeerHello
func StartSearchForLocalPairingPeers() string {
	pn := pairing.NewPeerNotifier()
	err := pn.Search()
	return makeJSONResponse(err)
}

// GetConnectionStringForBeingBootstrapped starts a pairing.ReceiverServer
// then generates a pairing.ConnectionParams. Used when the device is Logged out or has no Account keys
// and the device has no camera to read a QR code with
//
// Example: A desktop device (device without camera) receiving account data from mobile (device with camera)
func GetConnectionStringForBeingBootstrapped(configJSON string) string {
	if configJSON == "" {
		return makeJSONResponse(fmt.Errorf("no config given, PayloadSourceConfig is expected"))
	}
	cs, err := pairing.StartUpReceiverServer(statusBackend, configJSON)
	if err != nil {
		return makeJSONResponse(err)
	}
	return cs
}

// GetConnectionStringForBootstrappingAnotherDevice starts a pairing.SenderServer
// then generates a pairing.ConnectionParams. Used when the device is Logged in and therefore has Account keys
// and the device might not have a camera
//
// Example: A mobile or desktop device (devices that MAY have a camera but MUST have a screen)
// sending account data to a mobile (device with camera)
func GetConnectionStringForBootstrappingAnotherDevice(configJSON string) string {
	if configJSON == "" {
		return makeJSONResponse(fmt.Errorf("no config given, SendingServerConfig is expected"))
	}
	cs, err := pairing.StartUpSenderServer(statusBackend, configJSON)
	if err != nil {
		return makeJSONResponse(err)
	}
	return cs
}

// InputConnectionStringForBootstrapping starts a pairing.ReceiverClient
// The given server.ConnectionParams string will determine the server.Mode
//
// server.Mode = server.Sending
// Used when the device is Logged out or has no Account keys and has a camera to read a QR code
//
// Example: A mobile device (device with a camera) receiving account data from
// a device with a screen (mobile or desktop devices)
func InputConnectionStringForBootstrapping(cs, configJSON string) string {
	if configJSON == "" {
		return makeJSONResponse(fmt.Errorf("no config given, ReceiverClientConfig is expected"))
	}

	err := pairing.StartUpReceivingClient(statusBackend, cs, configJSON)
	return makeJSONResponse(err)
}

// InputConnectionStringForBootstrappingAnotherDevice starts a pairing.SendingClient
// The given server.ConnectionParams string will determine the server.Mode
//
// server.Mode = server.Receiving
// Used when the device is Logged in and therefore has Account keys and the has a camera to read a QR code
//
// Example: A mobile (device with camera) sending account data to a desktop device (device without camera)
func InputConnectionStringForBootstrappingAnotherDevice(cs, configJSON string) string {
	if configJSON == "" {
		return makeJSONResponse(fmt.Errorf("no config given, SenderClientConfig is expected"))
	}

	err := pairing.StartUpSendingClient(statusBackend, cs, configJSON)
	return makeJSONResponse(err)
}

func ValidateConnectionString(cs string) string {
	err := pairing.ValidateConnectionString(cs)
	if err == nil {
		return ""
	}
	return err.Error()
}

func EncodeTransfer(to string, value string) string {
	result, err := abi_spec.EncodeTransfer(to, value)
	if err != nil {
		log.Error("failed to encode transfer", "to", to, "value", value, "error", err)
		return ""
	}
	return result
}

func EncodeFunctionCall(method string, paramsJSON string) string {
	result, err := abi_spec.Encode(method, paramsJSON)
	if err != nil {
		log.Error("failed to encode function call", "method", method, "paramsJSON", paramsJSON, "error", err)
	}
	return result
}

func DecodeParameters(decodeParamJSON string) string {
	decodeParam := struct {
		BytesString string   `json:"bytesString"`
		Types       []string `json:"types"`
	}{}
	err := json.Unmarshal([]byte(decodeParamJSON), &decodeParam)
	if err != nil {
		log.Error("failed to unmarshal json when decoding parameters", "decodeParamJSON", decodeParamJSON, "error", err)
		return ""
	}
	result, err := abi_spec.Decode(decodeParam.BytesString, decodeParam.Types)
	if err != nil {
		log.Error("failed to decode parameters", "decodeParamJSON", decodeParamJSON, "error", err)
		return ""
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		log.Error("failed to marshal result", "result", result, "decodeParamJSON", decodeParamJSON, "error", err)
		return ""
	}
	return string(bytes)
}

func HexToNumber(hex string) string {
	return abi_spec.HexToNumber(hex)
}

func NumberToHex(numString string) string {
	return abi_spec.NumberToHex(numString)
}

func Sha3(str string) string {
	return "0x" + abi_spec.Sha3(str)
}

func Utf8ToHex(str string) string {
	hexString, err := abi_spec.Utf8ToHex(str)
	if err != nil {
		log.Error("failed to convert utf8 to hex", "str", str, "error", err)
	}
	return hexString
}

func HexToUtf8(hexString string) string {
	str, err := abi_spec.HexToUtf8(hexString)
	if err != nil {
		log.Error("failed to convert hex to utf8", "hexString", hexString, "error", err)
	}
	return str
}

func CheckAddressChecksum(address string) string {
	valid, err := abi_spec.CheckAddressChecksum(address)
	if err != nil {
		log.Error("failed to invoke check address checksum", "address", address, "error", err)
	}
	result, _ := json.Marshal(valid)
	return string(result)
}

func IsAddress(address string) string {
	valid, err := abi_spec.IsAddress(address)
	if err != nil {
		log.Error("failed to invoke IsAddress", "address", address, "error", err)
	}
	result, _ := json.Marshal(valid)
	return string(result)
}

func ToChecksumAddress(address string) string {
	address, err := abi_spec.ToChecksumAddress(address)
	if err != nil {
		log.Error("failed to convert to checksum address", "address", address, "error", err)
	}
	return address
}

func DeserializeAndCompressKey(DesktopKey string) string {
	deserialisedKey := MultiformatDeserializePublicKey(DesktopKey, "f")
	sanitisedKey := "0x" + deserialisedKey[5:]
	return CompressPublicKey(sanitisedKey)
}
