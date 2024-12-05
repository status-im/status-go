package statusgo

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"unsafe"

	"go.uber.org/zap"
	validator "gopkg.in/go-playground/validator.v9"

	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/status-im/zxcvbn-go"
	"github.com/status-im/zxcvbn-go/scoring"

	abi_spec "github.com/status-im/status-go/abi-spec"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/api/multiformat"
	"github.com/status-im/status-go/centralizedmetrics"
	"github.com/status-im/status-go/centralizedmetrics/providers"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/exportlogs"
	"github.com/status-im/status-go/extkeys"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/logutils/requestlog"
	m_requests "github.com/status-im/status-go/mobile/requests"
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
	"github.com/status-im/status-go/server/pairing/preflight"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/services/typeddata"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/signal"

	"path"
	"time"

	"github.com/status-im/status-go/mobile/callog"
)

func call(fn any, params ...any) any {
	return callog.Call(logutils.ZapLogger(), requestlog.GetRequestLogger(), fn, params...)
}

func callWithResponse(fn any, params ...any) string {
	return callog.CallWithResponse(logutils.ZapLogger(), requestlog.GetRequestLogger(), fn, params...)
}

type InitializeApplicationResponse struct {
	Accounts               []multiaccounts.Account         `json:"accounts"`
	CentralizedMetricsInfo *centralizedmetrics.MetricsInfo `json:"centralizedMetricsInfo"`
}

func InitializeApplication(requestJSON string) string {
	// NOTE: InitializeApplication is logs the call on its own rather than using `callWithResponse`,
	//       because the API logging is enabled during this exact call.
	defer callog.Recover(logutils.ZapLogger())

	startTime := time.Now()
	response := initializeApplication(requestJSON)
	callog.Log(
		requestlog.GetRequestLogger(),
		"InitializeApplication",
		requestJSON,
		response,
		startTime,
	)

	return response
}

func initializeApplication(requestJSON string) string {
	var request requests.InitializeApplication
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}
	err = request.Validate()
	if err != nil {
		return makeJSONResponse(err)
	}

	err = initializeLogging(&request)
	if err != nil {
		return makeJSONResponse(err)
	}

	providers.MixpanelAppID = request.MixpanelAppID
	providers.MixpanelToken = request.MixpanelToken

	statusBackend.StatusNode().SetMediaServerEnableTLS(request.MediaServerEnableTLS)
	statusBackend.UpdateRootDataDir(request.DataDir)

	err = statusBackend.OpenAccounts()
	if err != nil {
		return makeJSONResponse(err)
	}
	accs, err := statusBackend.GetAccounts()
	if err != nil {
		return makeJSONResponse(err)
	}

	metricsInfo, err := statusBackend.CentralizedMetricsInfo()
	if err != nil {
		return makeJSONResponse(err)
	}

	statusBackend.SetSentryDSN(request.SentryDSN)
	if metricsInfo.Enabled {
		err = statusBackend.EnablePanicReporting()
		if err != nil {
			return makeJSONResponse(err)
		}
	}

	response := &InitializeApplicationResponse{
		Accounts:               accs,
		CentralizedMetricsInfo: metricsInfo,
	}
	data, err := json.Marshal(response)
	if err != nil {
		return makeJSONResponse(err)
	}
	return string(data)
}

func initializeLogging(request *requests.InitializeApplication) error {
	if request.LogDir == "" {
		request.LogDir = request.DataDir
	}

	logSettings := logutils.LogSettings{
		Enabled: request.LogEnabled,
		Level:   request.LogLevel,
		File:    path.Join(request.LogDir, api.DefaultLogFile),
	}

	err := logutils.OverrideRootLoggerWithConfig(logSettings)
	if err != nil {
		return err
	}

	logutils.ZapLogger().Info("logging initialised",
		zap.Any("logSettings", logSettings),
		zap.Bool("APILoggingEnabled", request.APILoggingEnabled),
	)

	if request.APILoggingEnabled {
		logRequestsFile := path.Join(request.LogDir, api.DefaultAPILogFile)
		err = requestlog.ConfigureAndEnableRequestLogging(logRequestsFile)
		if err != nil {
			return err
		}
	}

	return nil
}

// Deprecated: Use InitializeApplication instead.
func OpenAccounts(datadir string) string {
	return callWithResponse(openAccounts, datadir)
}

// DEPRECATED: use InitializeApplication
// openAccounts opens database and returns accounts list.
func openAccounts(datadir string) string {
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

func ExtractGroupMembershipSignatures(signaturePairsStr string) string {
	return callWithResponse(extractGroupMembershipSignatures, signaturePairsStr)
}

// ExtractGroupMembershipSignatures extract public keys from tuples of content/signature.
func extractGroupMembershipSignatures(signaturePairsStr string) string {
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

func SignGroupMembership(content string) string {
	return callWithResponse(signGroupMembership, content)
}

// signGroupMembership signs a string containing group membership information.
func signGroupMembership(content string) string {
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

func GetNodeConfig() string {
	return callWithResponse(getNodeConfig)
}

// getNodeConfig returns the current config of the Status node
func getNodeConfig() string {
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

func ValidateNodeConfig(configJSON string) string {
	return callWithResponse(validateNodeConfig, configJSON)
}

// validateNodeConfig validates config for the Status node.
func validateNodeConfig(configJSON string) string {
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

func ResetChainData() string {
	return callWithResponse(resetChainData)
}

// resetChainData removes chain data from data directory.
func resetChainData() string {
	api.RunAsync(statusBackend.ResetChainData)
	return makeJSONResponse(nil)
}

func CallRPC(inputJSON string) string {
	return callWithResponse(callRPC, inputJSON)
}

// callRPC calls public APIs via RPC.
func callRPC(inputJSON string) string {
	resp, err := statusBackend.CallRPC(inputJSON)
	if err != nil {
		return makeJSONResponse(err)
	}
	return resp
}

func CallPrivateRPC(inputJSON string) string {
	return callWithResponse(callPrivateRPC, inputJSON)
}

// callPrivateRPC calls both public and private APIs via RPC.
func callPrivateRPC(inputJSON string) string {
	resp, err := statusBackend.CallPrivateRPC(inputJSON)
	if err != nil {
		return makeJSONResponse(err)
	}
	return resp
}

// Deprecated: Use VerifyAccountPasswordV2 instead
func VerifyAccountPassword(keyStoreDir, address, password string) string {
	return verifyAccountPassword(keyStoreDir, address, password)
}

// verifyAccountPassword verifies account password.
func verifyAccountPassword(keyStoreDir, address, password string) string {
	_, err := statusBackend.AccountManager().VerifyAccountPassword(keyStoreDir, address, password)
	return makeJSONResponse(err)
}

func VerifyAccountPasswordV2(requestJSON string) string {
	return callWithResponse(verifyAccountPasswordV2, requestJSON)
}

func verifyAccountPasswordV2(requestJSON string) string {
	var request requests.VerifyAccountPassword
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = request.Validate()
	if err != nil {
		return makeJSONResponse(err)
	}

	_, err = statusBackend.AccountManager().VerifyAccountPassword(request.KeyStoreDir, request.Address, request.Password)
	return makeJSONResponse(err)
}

func VerifyDatabasePasswordV2(requestJSON string) string {
	return callWithResponse(verifyDatabasePasswordV2, requestJSON)
}

func verifyDatabasePasswordV2(requestJSON string) string {
	var request requests.VerifyDatabasePassword
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = request.Validate()
	if err != nil {
		return makeJSONResponse(err)
	}
	err = statusBackend.VerifyDatabasePassword(request.KeyUID, request.Password)
	return makeJSONResponse(err)
}

// Deprecated: use VerifyDatabasePasswordV2 instead
func VerifyDatabasePassword(keyUID, password string) string {
	return verifyDatabasePassword(keyUID, password)
}

// verifyDatabasePassword verifies database password.
func verifyDatabasePassword(keyUID, password string) string {
	err := statusBackend.VerifyDatabasePassword(keyUID, password)
	return makeJSONResponse(err)
}

func MigrateKeyStoreDirV2(requestJSON string) string {
	return callWithResponse(migrateKeyStoreDirV2, requestJSON)
}

func migrateKeyStoreDirV2(requestJSON string) string {
	var request requests.MigrateKeystoreDir
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = request.Validate()
	if err != nil {
		return makeJSONResponse(err)
	}

	err = statusBackend.MigrateKeyStoreDir(request.Account, request.Password, request.OldDir, request.NewDir)
	return makeJSONResponse(err)
}

// Deprecated: Use MigrateKeyStoreDirV2 instead
func MigrateKeyStoreDir(accountData, password, oldDir, newDir string) string {
	return migrateKeyStoreDir(accountData, password, oldDir, newDir)
}

// migrateKeyStoreDir migrates key files to a new directory
func migrateKeyStoreDir(accountData, password, oldDir, newDir string) string {
	var account multiaccounts.Account
	err := json.Unmarshal([]byte(accountData), &account)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = statusBackend.MigrateKeyStoreDir(account, password, oldDir, newDir)
	return makeJSONResponse(err)
}

// login deprecated as Login and LoginWithConfig are deprecated
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
		logutils.ZapLogger().Debug("start a node with account", zap.String("key-uid", account.KeyUID))
		err := statusBackend.UpdateNodeConfigFleet(account, password, &conf)
		if err != nil {
			logutils.ZapLogger().Error("failed to update node config fleet", zap.String("key-uid", account.KeyUID), zap.Error(err))
			return statusBackend.LoggedIn(account.KeyUID, err)
		}

		err = statusBackend.StartNodeWithAccount(account, password, &conf, nil)
		if err != nil {
			logutils.ZapLogger().Error("failed to start a node", zap.String("key-uid", account.KeyUID), zap.Error(err))
			return err
		}
		logutils.ZapLogger().Debug("started a node with", zap.String("key-uid", account.KeyUID))
		return nil
	})

	return nil
}

// Login loads a key file (for a given address), tries to decrypt it using the password,
// to verify ownership if verified, purges all the previous identities from Whisper,
// and injects verified key as shh identity.
//
// Deprecated: Use LoginAccount instead.
func Login(accountData, password string) string {
	err := login(accountData, password, "")
	if err != nil {
		return makeJSONResponse(err)
	}
	return makeJSONResponse(nil)
}

// LoginWithConfig loads a key file (for a given address), tries to decrypt it using the password,
// to verify ownership if verified, purges all the previous identities from Whisper,
// and injects verified key as shh identity. It then updates the accounts node db configuration
// mergin the values received in the configJSON parameter
//
// Deprecated: Use LoginAccount instead.
func LoginWithConfig(accountData, password, configJSON string) string {
	err := login(accountData, password, configJSON)
	if err != nil {
		return makeJSONResponse(err)
	}
	return makeJSONResponse(nil)
}

func CreateAccountAndLogin(requestJSON string) string {
	return callWithResponse(createAccountAndLogin, requestJSON)
}

func createAccountAndLogin(requestJSON string) string {
	var request requests.CreateAccount
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = request.Validate(&requests.CreateAccountValidation{
		AllowEmptyDisplayName: false,
	})
	if err != nil {
		return makeJSONResponse(err)
	}

	api.RunAsync(func() error {
		logutils.ZapLogger().Debug("starting a node and creating config")
		_, err := statusBackend.CreateAccountAndLogin(&request)
		if err != nil {
			logutils.ZapLogger().Error("failed to create account", zap.Error(err))
			return err
		}
		logutils.ZapLogger().Debug("started a node, and created account")
		return nil
	})
	return makeJSONResponse(nil)
}

func AcceptTerms() string {
	return callWithResponse(acceptTerms)
}

func acceptTerms() string {
	err := statusBackend.AcceptTerms()
	return makeJSONResponse(err)
}

func LoginAccount(requestJSON string) string {
	return callWithResponse(loginAccount, requestJSON)
}

func loginAccount(requestJSON string) string {
	var request requests.Login
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = request.Validate()
	if err != nil {
		return makeJSONResponse(err)
	}

	api.RunAsync(func() error {
		err := statusBackend.LoginAccount(&request)
		if err != nil {
			logutils.ZapLogger().Error("loginAccount failed", zap.Error(err))
			return err
		}
		logutils.ZapLogger().Debug("loginAccount started node")
		return nil
	})
	return makeJSONResponse(nil)
}

func RestoreAccountAndLogin(requestJSON string) string {
	return callWithResponse(restoreAccountAndLogin, requestJSON)
}

func restoreAccountAndLogin(requestJSON string) string {
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
		logutils.ZapLogger().Debug("starting a node and restoring account")

		if request.Keycard != nil {
			_, err = statusBackend.RestoreKeycardAccountAndLogin(&request)
		} else {
			_, err = statusBackend.RestoreAccountAndLogin(&request)
		}

		if err != nil {
			logutils.ZapLogger().Error("failed to restore account", zap.Error(err))
			return err
		}
		logutils.ZapLogger().Debug("started a node, and restored account")
		return nil
	})

	return makeJSONResponse(nil)
}

// SaveAccountAndLogin saves account in status-go database.
// Deprecated: Use CreateAccountAndLogin instead.
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

	if *settings.Mnemonic != "" {
		settings.MnemonicWasNotShown = true
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
		logutils.ZapLogger().Debug("starting a node, and saving account with configuration", zap.String("key-uid", account.KeyUID))
		err := statusBackend.StartNodeWithAccountAndInitialConfig(account, password, settings, &conf, subaccs, nil)
		if err != nil {
			logutils.ZapLogger().Error("failed to start node and save account", zap.String("key-uid", account.KeyUID), zap.Error(err))
			return err
		}
		logutils.ZapLogger().Debug("started a node, and saved account", zap.String("key-uid", account.KeyUID))
		return nil
	})
	return makeJSONResponse(nil)
}

// Deprecated: Use DeleteMultiaccountV2 instead
func DeleteMultiaccount(keyUID, keyStoreDir string) string {
	return callWithResponse(deleteMultiaccount, keyUID, keyStoreDir)
}

// deleteMultiaccount
func deleteMultiaccount(keyUID, keyStoreDir string) string {
	err := statusBackend.DeleteMultiaccount(keyUID, keyStoreDir)
	return makeJSONResponse(err)
}

func DeleteMultiaccountV2(requestJSON string) string {
	return callWithResponse(deleteMultiaccountV2, requestJSON)
}

func deleteMultiaccountV2(requestJSON string) string {
	var request requests.DeleteMultiaccount
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = request.Validate()
	if err != nil {
		return makeJSONResponse(err)
	}

	err = statusBackend.DeleteMultiaccount(request.KeyUID, request.KeyStoreDir)
	return makeJSONResponse(err)
}

func DeleteImportedKeyV2(requestJSON string) string {
	return callWithResponse(deleteImportedKeyV2, requestJSON)
}

func deleteImportedKeyV2(requestJSON string) string {
	var request requests.DeleteImportedKey
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = request.Validate()
	if err != nil {
		return makeJSONResponse(err)
	}

	err = statusBackend.DeleteImportedKey(request.Address, request.Password, request.KeyStoreDir)
	return makeJSONResponse(err)
}

// Deprecated: Use DeleteImportedKeyV2 instead
func DeleteImportedKey(address, password, keyStoreDir string) string {
	return deleteImportedKey(address, password, keyStoreDir)
}

// deleteImportedKey
func deleteImportedKey(address, password, keyStoreDir string) string {
	err := statusBackend.DeleteImportedKey(address, password, keyStoreDir)
	return makeJSONResponse(err)
}

func InitKeystore(keydir string) string {
	return callWithResponse(initKeystore, keydir)
}

// initKeystore initialize keystore before doing any operations with keys.
func initKeystore(keydir string) string {
	err := statusBackend.AccountManager().InitKeystore(keydir)
	return makeJSONResponse(err)
}

// SaveAccountAndLoginWithKeycard saves account in status-go database.
// Deprecated: Use CreateAndAccountAndLogin with required keycard properties.
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
		logutils.ZapLogger().Debug("starting a node, and saving account with configuration", zap.String("key-uid", account.KeyUID))
		err := statusBackend.SaveAccountAndStartNodeWithKey(account, password, settings, &conf, subaccs, keyHex)
		if err != nil {
			logutils.ZapLogger().Error("failed to start node and save account", zap.String("key-uid", account.KeyUID), zap.Error(err))
			return err
		}
		logutils.ZapLogger().Debug("started a node, and saved account", zap.String("key-uid", account.KeyUID))
		return nil
	})
	return makeJSONResponse(nil)
}

// LoginWithKeycard initializes an account with a chat key and encryption key used for PFS.
// It purges all the previous identities from Whisper, and injects the key as shh identity.
// Deprecated: Use LoginAccount instead.
func LoginWithKeycard(accountData, password, keyHex string, configJSON string) string {
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
		logutils.ZapLogger().Debug("start a node with account", zap.String("key-uid", account.KeyUID))
		err := statusBackend.StartNodeWithKey(account, password, keyHex, &conf)
		if err != nil {
			logutils.ZapLogger().Error("failed to start a node", zap.String("key-uid", account.KeyUID), zap.Error(err))
			return err
		}
		logutils.ZapLogger().Debug("started a node with", zap.String("key-uid", account.KeyUID))
		return nil
	})
	return makeJSONResponse(nil)
}

func Logout() string {
	return callWithResponse(logout)
}

// logout is equivalent to clearing whisper identities.
func logout() string {
	return makeJSONResponse(statusBackend.Logout())
}

func SignMessage(rpcParams string) string {
	return callWithResponse(signMessage, rpcParams)
}

// signMessage unmarshals rpc params {data, address, password} and
// passes them onto backend.SignMessage.
func signMessage(rpcParams string) string {
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
func SignTypedData(data, address, password string) string {
	return signTypedData(data, address, password)
}

func signTypedData(data, address, password string) string {
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
	return callWithResponse(hashTypedData, data)
}

func hashTypedData(data string) string {
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
	return signTypedDataV4(data, address, password)
}

func signTypedDataV4(data, address, password string) string {
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
	return callWithResponse(hashTypedDataV4, data)
}

func hashTypedDataV4(data string) string {
	var typed apitypes.TypedData
	err := json.Unmarshal([]byte(data), &typed)
	if err != nil {
		return prepareJSONResponseWithCode(nil, err, codeFailedParseParams)
	}
	result, err := statusBackend.HashTypedDataV4(typed)
	return prepareJSONResponse(result.String(), err)
}

func Recover(rpcParams string) string {
	return callWithResponse(recoverWithRPCParams, rpcParams)
}

// recoverWithRPCParams unmarshals rpc params {signDataString, signedData} and passes
// them onto backend.
func recoverWithRPCParams(rpcParams string) string {
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
	return sendTransactionWithChainID(chainID, txArgsJSON, password)
}

// sendTransactionWithChainID converts RPC args and calls backend.SendTransactionWithChainID.
func sendTransactionWithChainID(chainID int, txArgsJSON, password string) string {
	var params wallettypes.SendTxArgs
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

// Deprecated: Use SendTransactionV2 instead.
func SendTransaction(txArgsJSON, password string) string {
	return sendTransaction(txArgsJSON, password)
}

// sendTransaction converts RPC args and calls backend.SendTransaction.
// Deprecated: Use sendTransactionV2 instead.
func sendTransaction(txArgsJSON, password string) string {
	var params wallettypes.SendTxArgs
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

func SendTransactionV2(requestJSON string) string {
	return callWithResponse(sendTransactionV2, requestJSON)
}

func sendTransactionV2(requestJSON string) string {
	var request requests.SendTransaction
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return prepareJSONResponseWithCode(nil, err, codeFailedParseParams)
	}

	err = request.Validate()
	if err != nil {
		return prepareJSONResponseWithCode(nil, err, codeFailedParseParams)
	}
	hash, err := statusBackend.SendTransaction(request.TxArgs, request.Password)
	code := codeUnknown
	if c, ok := errToCodeMap[err]; ok {
		code = c
	}
	return prepareJSONResponseWithCode(hash.String(), err, code)
}

func SendTransactionWithSignature(txArgsJSON, sigString string) string {
	return callWithResponse(sendTransactionWithSignature, txArgsJSON, sigString)
}

// sendTransactionWithSignature converts RPC args and calls backend.SendTransactionWithSignature
func sendTransactionWithSignature(txArgsJSON, sigString string) string {
	var params wallettypes.SendTxArgs
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

func HashTransaction(txArgsJSON string) string {
	return callWithResponse(hashTransaction, txArgsJSON)
}

// hashTransaction validate the transaction and returns new txArgs and the transaction hash.
func hashTransaction(txArgsJSON string) string {
	var params wallettypes.SendTxArgs
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
		Transaction wallettypes.SendTxArgs `json:"transaction"`
		Hash        types.Hash             `json:"hash"`
	}{
		Transaction: newTxArgs,
		Hash:        hash,
	}

	return prepareJSONResponseWithCode(result, err, code)
}

func HashMessage(message string) string {
	return callWithResponse(hashMessage, message)
}

// hashMessage calculates the hash of a message to be safely signed by the keycard
// The hash is calulcated as
//
//	keccak256("\x19Ethereum Signed Message:\n"${message length}${message}).
//
// This gives context to the signed message and prevents signing of transactions.
func hashMessage(message string) string {
	hash, err := api.HashMessage(message)
	code := codeUnknown
	if c, ok := errToCodeMap[err]; ok {
		code = c
	}
	return prepareJSONResponseWithCode(fmt.Sprintf("0x%x", hash), err, code)
}

func StartCPUProfile(dataDir string) string {
	return callWithResponse(startCPUProfile, dataDir)
}

// startCPUProfile runs pprof for CPU.
func startCPUProfile(dataDir string) string {
	err := profiling.StartCPUProfile(dataDir)
	return makeJSONResponse(err)
}

func StopCPUProfiling() string {
	return callWithResponse(stopCPUProfiling)
}

// stopCPUProfiling stops pprof for cpu.
func stopCPUProfiling() string { //nolint: deadcode
	err := profiling.StopCPUProfile()
	return makeJSONResponse(err)
}

func WriteHeapProfile(dataDir string) string {
	return callWithResponse(writeHeapProfile, dataDir)
}

// writeHeapProfile starts pprof for heap
func writeHeapProfile(dataDir string) string { //nolint: deadcode
	err := profiling.WriteHeapFile(dataDir)
	return makeJSONResponse(err)
}

// StartProfiling starts profiling and HTTP server for pprof
func StartProfiling(address string) string {
	return callWithResponse(startProfiling, address)
}

func startProfiling(address string) string {
	runtime.SetMutexProfileFraction(5)
	profiling.NewProfiler(address).Go()
	return makeJSONResponse(nil)
}

func makeJSONResponse(err error) string {
	errString := ""
	if err != nil {
		logutils.ZapLogger().Error("error in makeJSONResponse", zap.Error(err))
		errString = err.Error()
	}

	out := APIResponse{
		Error: errString,
	}
	outBytes, _ := json.Marshal(out)

	return string(outBytes)
}

func AddPeer(enode string) string {
	return callWithResponse(addPeer, enode)
}

// addPeer adds an enode as a peer.
func addPeer(enode string) string {
	err := statusBackend.StatusNode().AddPeer(enode)
	return makeJSONResponse(err)
}

// Deprecated: Use ConnectionChangeV2 instead.
func ConnectionChange(typ string, expensive int) {
	call(connectionChange, typ, expensive)
}

// connectionChange handles network state changes as reported
// by ReactNative (see https://facebook.github.io/react-native/docs/netinfo.html)
func connectionChange(typ string, expensive int) {
	statusBackend.ConnectionChange(typ, expensive == 1)
}

func ConnectionChangeV2(requestJSON string) string {
	return callWithResponse(connectionChangeV2, requestJSON)
}

func connectionChangeV2(requestJSON string) string {
	var request requests.ConnectionChange
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}
	statusBackend.ConnectionChange(request.Type, request.Expensive)
	return makeJSONResponse(nil)
}

// Deprecated: Use AppStateChangeV2 instead.
func AppStateChange(state string) {
	call(appStateChange, state)
}

// appStateChange handles app state changes (background/foreground).
func appStateChange(state string) {
	s, err := api.ParseAppState(state)
	if err != nil {
		logutils.ZapLogger().Error("parse app state failed, ignoring", zap.Error(err))
		return
	}
	statusBackend.AppStateChange(s)
}

func AppStateChangeV2(requestJSON string) string {
	return callWithResponse(appStateChangeV2, requestJSON)
}

func appStateChangeV2(requestJSON string) string {
	var request m_requests.AppStateChange
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}
	err = request.Validate()
	if err != nil {
		return makeJSONResponse(err)
	}
	statusBackend.AppStateChange(request.State)
	return makeJSONResponse(nil)
}

func StartLocalNotifications() string {
	return callWithResponse(startLocalNotifications)
}

// startLocalNotifications
func startLocalNotifications() string {
	err := statusBackend.StartLocalNotifications()
	return makeJSONResponse(err)
}

func StopLocalNotifications() string {
	return callWithResponse(stopLocalNotifications)
}

// stopLocalNotifications
func stopLocalNotifications() string {
	err := statusBackend.StopLocalNotifications()
	return makeJSONResponse(err)
}

func SetMobileSignalHandler(handler SignalHandler) {
	call(setMobileSignalHandler, handler)
}

// setMobileSignalHandler setup geth callback to notify about new signal
// used for gomobile builds
func setMobileSignalHandler(handler SignalHandler) {
	signal.SetMobileSignalHandler(func(data []byte) {
		if len(data) > 0 {
			handler.HandleSignal(string(data))
		}
	})
}

func SetSignalEventCallback(cb unsafe.Pointer) {
	call(setSignalEventCallback, cb)
}

// setSignalEventCallback setup geth callback to notify about new signal
func setSignalEventCallback(cb unsafe.Pointer) {
	signal.SetSignalEventCallback(cb)
}

// ExportNodeLogs reads current node log and returns content to a caller.
//
//export ExportNodeLogs
func ExportNodeLogs() string {
	return callWithResponse(exportNodeLogs)
}

func exportNodeLogs() string {
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

func SignHash(hexEncodedHash string) string {
	return callWithResponse(signHash, hexEncodedHash)
}

// signHash exposes vanilla ECDSA signing required for Swarm messages
func signHash(hexEncodedHash string) string {
	hexEncodedSignature, err := statusBackend.SignHash(hexEncodedHash)
	if err != nil {
		return makeJSONResponse(err)
	}
	return hexEncodedSignature
}

func GenerateAlias(pk string) string {
	return callWithResponse(generateAlias, pk)
}

func generateAlias(pk string) string {
	// We ignore any error, empty string is considered an error
	name, _ := protocol.GenerateAlias(pk)
	return name
}

func IsAlias(value string) string {
	return callWithResponse(isAlias, value)
}

func isAlias(value string) string {
	return prepareJSONResponse(alias.IsAlias(value), nil)
}

func Identicon(pk string) string {
	return callWithResponse(identicon, pk)
}

func identicon(pk string) string {
	// We ignore any error, empty string is considered an error
	identicon, _ := protocol.Identicon(pk)
	return identicon
}

func EmojiHash(pk string) string {
	return callWithResponse(emojiHash, pk)
}

func emojiHash(pk string) string {
	return prepareJSONResponse(emojihash.GenerateFor(pk))
}

func ColorHash(pk string) string {
	return callWithResponse(colorHash, pk)
}

func colorHash(pk string) string {
	return prepareJSONResponse(colorhash.GenerateFor(pk))
}

func ColorID(pk string) string {
	return callWithResponse(colorID, pk)
}

func colorID(pk string) string {
	return prepareJSONResponse(identityUtils.ToColorID(pk))
}

// Deprecated: Use ValidateMnemonicV2 instead.
func ValidateMnemonic(mnemonic string) string {
	return validateMnemonic(mnemonic)
}

func validateMnemonic(mnemonic string) string {
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

func ValidateMnemonicV2(requestJSON string) string {
	return callWithResponse(validateMnemonicV2, requestJSON)
}

func validateMnemonicV2(requestJSON string) string {
	var request requests.ValidateMnemonic
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}
	return validateMnemonic(request.Mnemonic)
}

func DecompressPublicKey(key string) string {
	return callWithResponse(decompressPublicKey, key)
}

// decompressPublicKey decompresses 33-byte compressed format to uncompressed 65-byte format.
func decompressPublicKey(key string) string {
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

func CompressPublicKey(key string) string {
	return callWithResponse(compressPublicKey, key)
}

// compressPublicKey compresses uncompressed 65-byte format to 33-byte compressed format.
func compressPublicKey(key string) string {
	pubKey, err := common.HexToPubkey(key)
	if err != nil {
		return makeJSONResponse(err)
	}
	return types.EncodeHex(crypto.CompressPubkey(pubKey))
}

func SerializeLegacyKey(key string) string {
	return callWithResponse(serializeLegacyKey, key)
}

// serializeLegacyKey compresses an old format public key (0x04...) to the new one zQ...
func serializeLegacyKey(key string) string {
	cpk, err := multiformat.SerializeLegacyKey(key)
	if err != nil {
		return makeJSONResponse(err)
	}
	return cpk
}

func MultiformatSerializePublicKey(key, outBase string) string {
	return callWithResponse(multiformatSerializePublicKey, key, outBase)
}

// SerializePublicKey compresses an uncompressed multibase encoded multicodec identified EC public key
// For details on usage see specs https://specs.status.im/spec/2#public-key-serialization
func multiformatSerializePublicKey(key, outBase string) string {
	cpk, err := multiformat.SerializePublicKey(key, outBase)
	if err != nil {
		return makeJSONResponse(err)
	}
	return cpk
}

// Deprecated: Use MultiformatDeserializePublicKeyV2 instead.
func MultiformatDeserializePublicKey(key, outBase string) string {
	return callWithResponse(multiformatDeserializePublicKey, key, outBase)
}

// DeserializePublicKey decompresses a compressed multibase encoded multicodec identified EC public key
// For details on usage see specs https://specs.status.im/spec/2#public-key-serialization
func multiformatDeserializePublicKey(key, outBase string) string {
	pk, err := multiformat.DeserializePublicKey(key, outBase)
	if err != nil {
		return makeJSONResponse(err)
	}
	return pk
}

func MultiformatDeserializePublicKeyV2(requestJSON string) string {
	return callWithResponse(multiformatDeserializePublicKeyV2, requestJSON)
}

func multiformatDeserializePublicKeyV2(requestJSON string) string {
	var request requests.MultiformatDeserializePublicKey
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}
	return multiformatDeserializePublicKey(request.Key, request.OutBase)
}

// Deprecated: Use ExportUnencryptedDatabaseV2 instead.
func ExportUnencryptedDatabase(accountData, password, databasePath string) string {
	return exportUnencryptedDatabase(accountData, password, databasePath)
}

// exportUnencryptedDatabase exports the database unencrypted to the given path
func exportUnencryptedDatabase(accountData, password, databasePath string) string {
	var account multiaccounts.Account
	err := json.Unmarshal([]byte(accountData), &account)
	if err != nil {
		return makeJSONResponse(err)
	}
	err = statusBackend.ExportUnencryptedDatabase(account, password, databasePath)
	return makeJSONResponse(err)
}

func ExportUnencryptedDatabaseV2(requestJSON string) string {
	return callWithResponse(exportUnencryptedDatabaseV2, requestJSON)
}

func exportUnencryptedDatabaseV2(requestJSON string) string {
	var request requests.ExportUnencryptedDatabase
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}
	err = statusBackend.ExportUnencryptedDatabase(request.Account, request.Password, request.DatabasePath)
	return makeJSONResponse(err)
}

// Deprecated: Use ImportUnencryptedDatabaseV2 instead.
func ImportUnencryptedDatabase(accountData, password, databasePath string) string {
	return importUnencryptedDatabase(accountData, password, databasePath)
}

// importUnencryptedDatabase imports the database unencrypted to the given directory
func importUnencryptedDatabase(accountData, password, databasePath string) string {
	var account multiaccounts.Account
	err := json.Unmarshal([]byte(accountData), &account)
	if err != nil {
		return makeJSONResponse(err)
	}
	err = statusBackend.ImportUnencryptedDatabase(account, password, databasePath)
	return makeJSONResponse(err)
}

func ImportUnencryptedDatabaseV2(requestJSON string) string {
	return callWithResponse(importUnencryptedDatabaseV2, requestJSON)
}

func importUnencryptedDatabaseV2(requestJSON string) string {
	var request requests.ImportUnencryptedDatabase
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}
	err = statusBackend.ImportUnencryptedDatabase(request.Account, request.Password, request.DatabasePath)
	return makeJSONResponse(err)
}

// Deprecated: Use ChangeDatabasePasswordV2 instead.
func ChangeDatabasePassword(keyUID, password, newPassword string) string {
	return changeDatabasePassword(keyUID, password, newPassword)
}

// changeDatabasePassword changes the password of the database
func changeDatabasePassword(keyUID, password, newPassword string) string {
	err := statusBackend.ChangeDatabasePassword(keyUID, password, newPassword)
	return makeJSONResponse(err)
}

func ChangeDatabasePasswordV2(requestJSON string) string {
	return callWithResponse(changeDatabasePasswordV2, requestJSON)
}

func changeDatabasePasswordV2(requestJSON string) string {
	var request requests.ChangeDatabasePassword
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}
	return changeDatabasePassword(request.KeyUID, request.OldPassword, request.NewPassword)
}

// Deprecated: Use ConvertToKeycardAccountV2 instead.
func ConvertToKeycardAccount(accountData, settingsJSON, keycardUID, password, newPassword string) string {
	return convertToKeycardAccount(accountData, settingsJSON, keycardUID, password, newPassword)
}

// convertToKeycardAccount converts the account to a keycard account
func convertToKeycardAccount(accountData, settingsJSON, keycardUID, password, newPassword string) string {
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
	return makeJSONResponse(err)
}

func ConvertToKeycardAccountV2(requestJSON string) string {
	return callWithResponse(convertToKeycardAccountV2, requestJSON)
}

func convertToKeycardAccountV2(requestJSON string) string {
	var request requests.ConvertToKeycardAccount
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}
	err = statusBackend.ConvertToKeycardAccount(request.Account, request.Settings, request.KeycardUID, request.OldPassword, request.NewPassword)
	return makeJSONResponse(err)
}

// Deprecated: Use ConvertToRegularAccountV2 instead.
func ConvertToRegularAccount(mnemonic, currPassword, newPassword string) string {
	return convertToRegularAccount(mnemonic, currPassword, newPassword)
}

// convertToRegularAccount converts the account to a regular account
func convertToRegularAccount(mnemonic, currPassword, newPassword string) string {
	err := statusBackend.ConvertToRegularAccount(mnemonic, currPassword, newPassword)
	return makeJSONResponse(err)
}

func ConvertToRegularAccountV2(requestJSON string) string {
	return callWithResponse(convertToRegularAccountV2, requestJSON)
}

func convertToRegularAccountV2(requestJSON string) string {
	var request requests.ConvertToRegularAccount
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}
	return convertToRegularAccount(request.Mnemonic, request.CurrPassword, request.NewPassword)
}

func ImageServerTLSCert() string {
	cert, err := server.PublicMediaTLSCert()
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

type FleetDescription struct {
	DefaultFleet string                         `json:"defaultFleet"`
	Fleets       map[string]map[string][]string `json:"fleets"`
}

func Fleets() string {
	return callWithResponse(fleets)
}

func fleets() string {
	fleets := FleetDescription{
		DefaultFleet: api.DefaultFleet,
		Fleets:       params.GetSupportedFleets(),
	}

	data, err := json.Marshal(fleets)
	if err != nil {
		return makeJSONResponse(fmt.Errorf("Error marshalling to json: %v", err))
	}
	return string(data)
}

// Deprecated: Use SwitchFleetV2 instead.
func SwitchFleet(fleet string, configJSON string) string {
	return callWithResponse(switchFleet, fleet, configJSON)
}

func switchFleet(fleet string, configJSON string) string {
	var conf params.NodeConfig
	if configJSON != "" {
		err := json.Unmarshal([]byte(configJSON), &conf)
		if err != nil {
			return makeJSONResponse(err)
		}
	}

	clusterConfig, err := params.LoadClusterConfigFromFleet(fleet)
	if err != nil {
		return makeJSONResponse(err)
	}

	conf.ClusterConfig.Fleet = fleet
	conf.ClusterConfig.ClusterID = clusterConfig.ClusterID

	err = statusBackend.SwitchFleet(fleet, &conf)

	return makeJSONResponse(err)
}

func SwitchFleetV2(requestJSON string) string {
	return callWithResponse(switchFleetV2, requestJSON)
}

func switchFleetV2(requestJSON string) string {
	var request requests.SwitchFleet
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}
	return switchFleet(request.Fleet, request.ConfigJSON)
}

// Deprecated: Use GenerateImagesV2 instead.
func GenerateImages(filepath string, aX, aY, bX, bY int) string {
	return generateImages(filepath, aX, aY, bX, bY)
}

func GenerateImagesV2(requestJSON string) string {
	return callWithResponse(generateImagesV2, requestJSON)
}

func generateImagesV2(requestJSON string) string {
	var request requests.GenerateImages
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}
	return generateImages(request.Filepath, request.AX, request.AY, request.BX, request.BY)
}

func generateImages(filepath string, aX, aY, bX, bY int) string {
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

func LocalPairingPreflightOutboundCheck() string {
	return callWithResponse(localPairingPreflightOutboundCheck)
}

// localPairingPreflightOutboundCheck creates a local tls server accessible via an outbound network address.
// The function creates a client and makes an outbound network call to the local server. This function should be
// triggered to ensure that the device has permissions to access its LAN or to make outbound network calls.
//
// In addition, the functionality attempts to address an issue with iOS devices https://stackoverflow.com/a/64242745
func localPairingPreflightOutboundCheck() string {
	err := preflight.CheckOutbound()
	return makeJSONResponse(err)
}

func StartSearchForLocalPairingPeers() string {
	return callWithResponse(startSearchForLocalPairingPeers)
}

// startSearchForLocalPairingPeers starts a UDP multicast beacon that both listens for and broadcasts to LAN peers
// on discovery the beacon will emit a signal with the details of the discovered peer.
//
// Currently, beacons are configured to search for 2 minutes pinging the network every 500 ms;
//   - If no peer discovery is made before this time elapses the operation will terminate.
//   - If a peer is discovered the pairing.PeerNotifier will terminate operation after 5 seconds, giving the peer
//     reasonable time to discover this device.
//
// Peer details are represented by a json.Marshal peers.LocalPairingPeerHello
func startSearchForLocalPairingPeers() string {
	pn := pairing.NewPeerNotifier()
	err := pn.Search()
	return makeJSONResponse(err)
}

func GetConnectionStringForBeingBootstrapped(configJSON string) string {
	return callWithResponse(getConnectionStringForBeingBootstrapped, configJSON)
}

// getConnectionStringForBeingBootstrapped starts a pairing.ReceiverServer
// then generates a pairing.ConnectionParams. Used when the device is Logged out or has no Account keys
// and the device has no camera to read a QR code with
//
// Example: A desktop device (device without camera) receiving account data from mobile (device with camera)
func getConnectionStringForBeingBootstrapped(configJSON string) string {
	if configJSON == "" {
		return makeJSONResponse(fmt.Errorf("no config given, PayloadSourceConfig is expected"))
	}

	statusBackend.LocalPairingStateManager.SetPairing(true)
	defer func() {
		statusBackend.LocalPairingStateManager.SetPairing(false)
	}()

	cs, err := pairing.StartUpReceiverServer(statusBackend, configJSON)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = statusBackend.Logout()
	if err != nil {
		return makeJSONResponse(err)
	}

	return cs
}

func GetConnectionStringForBootstrappingAnotherDevice(configJSON string) string {
	return callWithResponse(getConnectionStringForBootstrappingAnotherDevice, configJSON)
}

// getConnectionStringForBootstrappingAnotherDevice starts a pairing.SenderServer
// then generates a pairing.ConnectionParams. Used when the device is Logged in and therefore has Account keys
// and the device might not have a camera
//
// Example: A mobile or desktop device (devices that MAY have a camera but MUST have a screen)
// sending account data to a mobile (device with camera)
func getConnectionStringForBootstrappingAnotherDevice(configJSON string) string {
	if configJSON == "" {
		return makeJSONResponse(fmt.Errorf("no config given, SendingServerConfig is expected"))
	}

	statusBackend.LocalPairingStateManager.SetPairing(true)
	defer func() {
		statusBackend.LocalPairingStateManager.SetPairing(false)
	}()

	cs, err := pairing.StartUpSenderServer(statusBackend, configJSON)
	if err != nil {
		return makeJSONResponse(err)
	}
	return cs
}

type inputConnectionStringForBootstrappingResponse struct {
	InstallationID string `json:"installationId"`
	KeyUID         string `json:"keyUID"`
	Error          error  `json:"error"`
}

func (i *inputConnectionStringForBootstrappingResponse) toJSON(err error) string {
	i.Error = err
	j, _ := json.Marshal(i)
	return string(j)
}

// Deprecated: Use InputConnectionStringForBootstrappingV2 instead.
func InputConnectionStringForBootstrapping(cs, configJSON string) string {
	return callWithResponse(inputConnectionStringForBootstrapping, cs, configJSON)
}

// inputConnectionStringForBootstrapping starts a pairing.ReceiverClient
// The given server.ConnectionParams string will determine the server.Mode
//
// server.Mode = server.Sending
// Used when the device is Logged out or has no Account keys and has a camera to read a QR code
//
// Example: A mobile device (device with a camera) receiving account data from
// a device with a screen (mobile or desktop devices)
func inputConnectionStringForBootstrapping(cs, configJSON string) string {
	var err error
	if configJSON == "" {
		return makeJSONResponse(fmt.Errorf("no config given, ReceiverClientConfig is expected"))
	}

	params := &pairing.ConnectionParams{}
	err = params.FromString(cs)
	if err != nil {
		response := &inputConnectionStringForBootstrappingResponse{}
		return response.toJSON(fmt.Errorf("could not parse connection string"))
	}
	response := &inputConnectionStringForBootstrappingResponse{
		InstallationID: params.InstallationID(),
		KeyUID:         params.KeyUID(),
	}

	err = statusBackend.LocalPairingStateManager.StartPairing(cs)
	defer func() { statusBackend.LocalPairingStateManager.StopPairing(cs, err) }()
	if err != nil {
		return response.toJSON(err)
	}

	var conf pairing.ReceiverClientConfig
	err = json.Unmarshal([]byte(configJSON), &conf)
	if err != nil {
		return response.toJSON(err)
	}

	err = pairing.StartUpReceivingClient(statusBackend, cs, &conf)
	if err != nil {
		return response.toJSON(err)
	}

	return response.toJSON(statusBackend.Logout())
}

func InputConnectionStringForBootstrappingV2(requestJSON string) string {
	return callWithResponse(inputConnectionStringForBootstrappingV2, requestJSON)
}

func inputConnectionStringForBootstrappingV2(requestJSON string) string {
	var request m_requests.InputConnectionStringForBootstrapping
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}
	if err := request.Validate(); err != nil {
		return makeJSONResponse(err)
	}

	params := &pairing.ConnectionParams{}
	err = params.FromString(request.ConnectionString)
	if err != nil {
		response := &inputConnectionStringForBootstrappingResponse{}
		return response.toJSON(fmt.Errorf("could not parse connection string"))
	}
	response := &inputConnectionStringForBootstrappingResponse{
		InstallationID: params.InstallationID(),
		KeyUID:         params.KeyUID(),
	}

	err = statusBackend.LocalPairingStateManager.StartPairing(request.ConnectionString)
	defer func() { statusBackend.LocalPairingStateManager.StopPairing(request.ConnectionString, err) }()
	if err != nil {
		return response.toJSON(err)
	}

	err = pairing.StartUpReceivingClient(statusBackend, request.ConnectionString, request.ReceiverClientConfig)
	if err != nil {
		return response.toJSON(err)
	}

	return response.toJSON(statusBackend.Logout())
}

// Deprecated: Use InputConnectionStringForBootstrappingAnotherDeviceV2 instead.
func InputConnectionStringForBootstrappingAnotherDevice(cs, configJSON string) string {
	return callWithResponse(inputConnectionStringForBootstrappingAnotherDevice, cs, configJSON)
}

// inputConnectionStringForBootstrappingAnotherDevice starts a pairing.SendingClient
// The given server.ConnectionParams string will determine the server.Mode
//
// server.Mode = server.Receiving
// Used when the device is Logged in and therefore has Account keys and the has a camera to read a QR code
//
// Example: A mobile (device with camera) sending account data to a desktop device (device without camera)
func inputConnectionStringForBootstrappingAnotherDevice(cs, configJSON string) string {
	var err error
	if configJSON == "" {
		return makeJSONResponse(fmt.Errorf("no config given, SenderClientConfig is expected"))
	}

	err = statusBackend.LocalPairingStateManager.StartPairing(cs)
	defer func() { statusBackend.LocalPairingStateManager.StopPairing(cs, err) }()
	if err != nil {
		return makeJSONResponse(err)
	}
	var conf pairing.SenderClientConfig
	err = json.Unmarshal([]byte(configJSON), &conf)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = pairing.StartUpSendingClient(statusBackend, cs, &conf)
	return makeJSONResponse(err)
}

func InputConnectionStringForBootstrappingAnotherDeviceV2(requestJSON string) string {
	return callWithResponse(inputConnectionStringForBootstrappingAnotherDeviceV2, requestJSON)
}

func inputConnectionStringForBootstrappingAnotherDeviceV2(requestJSON string) string {
	var request m_requests.InputConnectionStringForBootstrappingAnotherDevice
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}
	if err := request.Validate(); err != nil {
		return makeJSONResponse(err)
	}

	err = statusBackend.LocalPairingStateManager.StartPairing(request.ConnectionString)
	defer func() { statusBackend.LocalPairingStateManager.StopPairing(request.ConnectionString, err) }()
	if err != nil {
		return makeJSONResponse(err)
	}

	err = pairing.StartUpSendingClient(statusBackend, request.ConnectionString, request.SenderClientConfig)
	return makeJSONResponse(err)
}

func GetConnectionStringForExportingKeypairsKeystores(configJSON string) string {
	return callWithResponse(getConnectionStringForExportingKeypairsKeystores, configJSON)
}

// getConnectionStringForExportingKeypairsKeystores starts a pairing.SenderServer
// then generates a pairing.ConnectionParams. Used when the device is Logged in and therefore has Account keys
// and the device might not have a camera, to transfer kestore files of provided key uids.
func getConnectionStringForExportingKeypairsKeystores(configJSON string) string {
	if configJSON == "" {
		return makeJSONResponse(fmt.Errorf("no config given, SendingServerConfig is expected"))
	}

	cs, err := pairing.StartUpKeystoreFilesSenderServer(statusBackend, configJSON)
	if err != nil {
		return makeJSONResponse(err)
	}
	return cs
}

// Deprecated: Use InputConnectionStringForImportingKeypairsKeystoresV2 instead.
func InputConnectionStringForImportingKeypairsKeystores(cs, configJSON string) string {
	return callWithResponse(inputConnectionStringForImportingKeypairsKeystores, cs, configJSON)
}

// inputConnectionStringForImportingKeypairsKeystores starts a pairing.ReceiverClient
// The given server.ConnectionParams string will determine the server.Mode
// Used when the device is Logged in and has Account keys and has a camera to read a QR code
//
// Example: A mobile device (device with a camera) receiving account data from
// a device with a screen (mobile or desktop devices)
func inputConnectionStringForImportingKeypairsKeystores(cs, configJSON string) string {
	if configJSON == "" {
		return makeJSONResponse(fmt.Errorf("no config given, ReceiverClientConfig is expected"))
	}

	var conf pairing.KeystoreFilesReceiverClientConfig
	err := json.Unmarshal([]byte(configJSON), &conf)
	if err != nil {
		return makeJSONResponse(err)
	}
	err = pairing.StartUpKeystoreFilesReceivingClient(statusBackend, cs, &conf)
	return makeJSONResponse(err)
}

func InputConnectionStringForImportingKeypairsKeystoresV2(requestJSON string) string {
	return callWithResponse(inputConnectionStringForImportingKeypairsKeystoresV2, requestJSON)
}

func inputConnectionStringForImportingKeypairsKeystoresV2(requestJSON string) string {
	var request m_requests.InputConnectionStringForImportingKeypairsKeystores
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}
	if err := request.Validate(); err != nil {
		return makeJSONResponse(err)
	}
	err = pairing.StartUpKeystoreFilesReceivingClient(statusBackend, request.ConnectionString, request.KeystoreFilesReceiverClientConfig)
	return makeJSONResponse(err)
}

func ValidateConnectionString(cs string) string {
	return callWithResponse(validateConnectionString, cs)
}

func validateConnectionString(cs string) string {
	err := pairing.ValidateConnectionString(cs)
	if err == nil {
		return ""
	}
	return err.Error()
}

// Deprecated: Use EncodeTransferV2 instead.
func EncodeTransfer(to string, value string) string {
	return callWithResponse(encodeTransfer, to, value)
}

func encodeTransfer(to string, value string) string {
	result, err := abi_spec.EncodeTransfer(to, value)
	if err != nil {
		logutils.ZapLogger().Error("failed to encode transfer", zap.String("to", to), zap.String("value", value), zap.Error(err))
		return ""
	}
	return result
}

func EncodeTransferV2(requestJSON string) string {
	return callWithResponse(encodeTransferV2, requestJSON)
}

func encodeTransferV2(requestJSON string) string {
	var request requests.EncodeTransfer
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}
	return encodeTransfer(request.To, request.Value)
}

// Deprecated: Use EncodeFunctionCallV2 instead.
func EncodeFunctionCall(method string, paramsJSON string) string {
	return callWithResponse(encodeFunctionCall, method, paramsJSON)
}

func encodeFunctionCall(method string, paramsJSON string) string {
	result, err := abi_spec.Encode(method, paramsJSON)
	if err != nil {
		logutils.ZapLogger().Error("failed to encode function call", zap.String("method", method), zap.String("paramsJSON", paramsJSON), zap.Error(err))
		return ""
	}
	return result
}

func EncodeFunctionCallV2(requestJSON string) string {
	return callWithResponse(encodeFunctionCallV2, requestJSON)
}

func encodeFunctionCallV2(requestJSON string) string {
	var request requests.EncodeFunctionCall
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}
	return encodeFunctionCall(request.Method, request.ParamsJSON)
}

func DecodeParameters(decodeParamJSON string) string {
	return decodeParameters(decodeParamJSON)
}

func decodeParameters(decodeParamJSON string) string {
	decodeParam := struct {
		BytesString string   `json:"bytesString"`
		Types       []string `json:"types"`
	}{}
	err := json.Unmarshal([]byte(decodeParamJSON), &decodeParam)
	if err != nil {
		logutils.ZapLogger().Error("failed to unmarshal json when decoding parameters", zap.String("decodeParamJSON", decodeParamJSON), zap.Error(err))
		return ""
	}
	result, err := abi_spec.Decode(decodeParam.BytesString, decodeParam.Types)
	if err != nil {
		logutils.ZapLogger().Error("failed to decode parameters", zap.String("decodeParamJSON", decodeParamJSON), zap.Error(err))
		return ""
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		logutils.ZapLogger().Error("failed to marshal result", zap.Any("result", result), zap.String("decodeParamJSON", decodeParamJSON), zap.Error(err))
		return ""
	}
	return string(bytes)
}

func HexToNumber(hex string) string {
	return callWithResponse(hexToNumber, hex)
}

func hexToNumber(hex string) string {
	return abi_spec.HexToNumber(hex)
}

func NumberToHex(numString string) string {
	return callWithResponse(numberToHex, numString)
}

func numberToHex(numString string) string {
	return abi_spec.NumberToHex(numString)
}

func Sha3(str string) string {
	return "0x" + abi_spec.Sha3(str)
}

func Utf8ToHex(str string) string {
	return callWithResponse(utf8ToHex, str)
}

func utf8ToHex(str string) string {
	hexString, err := abi_spec.Utf8ToHex(str)
	if err != nil {
		logutils.ZapLogger().Error("failed to convert utf8 to hex", zap.String("str", str), zap.Error(err))
	}
	return hexString
}

func HexToUtf8(hexString string) string {
	return callWithResponse(hexToUtf8, hexString)
}

func hexToUtf8(hexString string) string {
	str, err := abi_spec.HexToUtf8(hexString)
	if err != nil {
		logutils.ZapLogger().Error("failed to convert hex to utf8", zap.String("hexString", hexString), zap.Error(err))
	}
	return str
}

func CheckAddressChecksum(address string) string {
	return callWithResponse(checkAddressChecksum, address)
}

func checkAddressChecksum(address string) string {
	valid, err := abi_spec.CheckAddressChecksum(address)
	if err != nil {
		logutils.ZapLogger().Error("failed to invoke check address checksum", zap.String("address", address), zap.Error(err))
	}
	result, _ := json.Marshal(valid)
	return string(result)
}

func IsAddress(address string) string {
	return callWithResponse(isAddress, address)
}

func isAddress(address string) string {
	valid, err := abi_spec.IsAddress(address)
	if err != nil {
		logutils.ZapLogger().Error("failed to invoke IsAddress", zap.String("address", address), zap.Error(err))
	}
	result, _ := json.Marshal(valid)
	return string(result)
}

func ToChecksumAddress(address string) string {
	return callWithResponse(toChecksumAddress, address)
}

func toChecksumAddress(address string) string {
	address, err := abi_spec.ToChecksumAddress(address)
	if err != nil {
		logutils.ZapLogger().Error("failed to convert to checksum address", zap.String("address", address), zap.Error(err))
	}
	return address
}

func DeserializeAndCompressKey(DesktopKey string) string {
	return callWithResponse(deserializeAndCompressKey, DesktopKey)
}

func deserializeAndCompressKey(DesktopKey string) string {
	deserialisedKey := MultiformatDeserializePublicKey(DesktopKey, "f")
	sanitisedKey := "0x" + deserialisedKey[5:]
	return CompressPublicKey(sanitisedKey)
}

type InitLoggingRequest struct {
	logutils.LogSettings
	LogRequestGo   bool   `json:"LogRequestGo"`
	LogRequestFile string `json:"LogRequestFile"`
}

// InitLogging The InitLogging function should be called when the application starts.
// This ensures that we can capture logs before the user login. Subsequent calls will update the logger settings.
// Before this, we can only capture logs after user login since we will only configure the logging after the login process.
// Deprecated: Use InitializeApplication instead
func InitLogging(logSettingsJSON string) string {
	var logSettings InitLoggingRequest
	var err error
	if err = json.Unmarshal([]byte(logSettingsJSON), &logSettings); err != nil {
		return makeJSONResponse(err)
	}

	if err = logutils.OverrideRootLoggerWithConfig(logSettings.LogSettings); err == nil {
		logutils.ZapLogger().Info("logging initialised", zap.String("logSettings", logSettingsJSON))
	}

	if logSettings.LogRequestGo {
		err = requestlog.ConfigureAndEnableRequestLogging(logSettings.LogRequestFile)
		if err != nil {
			return makeJSONResponse(err)
		}
	}

	return makeJSONResponse(err)
}

func GetRandomMnemonic() string {
	mnemonic, err := account.GetRandomMnemonic()
	if err != nil {
		return makeJSONResponse(err)
	}
	return mnemonic
}

func ToggleCentralizedMetrics(requestJSON string) string {
	return callWithResponse(toggleCentralizedMetrics, requestJSON)
}

func toggleCentralizedMetrics(requestJSON string) string {
	var request requests.ToggleCentralizedMetrics
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = request.Validate()
	if err != nil {
		return makeJSONResponse(err)
	}

	err = statusBackend.ToggleCentralizedMetrics(request.Enabled)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = statusBackend.TogglePanicReporting(request.Enabled)
	if err != nil {
		return makeJSONResponse(err)
	}

	return makeJSONResponse(nil)
}

func CentralizedMetricsInfo() string {
	return callWithResponse(centralizedMetricsInfo)
}

func centralizedMetricsInfo() string {
	metricsInfo, err := statusBackend.CentralizedMetricsInfo()
	if err != nil {
		return makeJSONResponse(err)
	}
	data, err := json.Marshal(metricsInfo)
	if err != nil {
		return makeJSONResponse(err)
	}
	return string(data)
}

func AddCentralizedMetric(requestJSON string) string {
	return callWithResponse(addCentralizedMetric, requestJSON)
}

func addCentralizedMetric(requestJSON string) string {
	var request requests.AddCentralizedMetric
	err := json.Unmarshal([]byte(requestJSON), &request)
	if err != nil {
		return makeJSONResponse(err)
	}

	err = request.Validate()
	if err != nil {
		return makeJSONResponse(err)
	}
	metric := request.Metric

	metric.EnsureID()
	err = statusBackend.AddCentralizedMetric(*metric)
	if err != nil {
		return makeJSONResponse(err)
	}

	return metric.ID
}

func IntendedPanic(message string) string {
	type intendedPanic struct {
		error
	}
	return callWithResponse(func() {
		err := intendedPanic{error: errors.New(message)}
		panic(err)
	})
}
