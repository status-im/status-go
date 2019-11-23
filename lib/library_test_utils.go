// +build e2e_test

// This is a file with e2e tests for C bindings written in library.go.
// As a CGO file, it can't have `_test.go` suffix as it's not allowed by Go.
// At the same time, we don't want this file to be included in the binaries.
// This is why `e2e_test` tag was introduced. Without it, this file is excluded
// from the build. Providing this tag will include this file into the build
// and that's what is done while running e2e tests for `lib/` package.

package main

import (
	"C"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/signal"
	. "github.com/status-im/status-go/t/utils" //nolint: golint
	"github.com/status-im/status-go/transactions"
	"github.com/stretchr/testify/require"
)

const initJS = `
	var _status_catalog = {
		foo: 'bar'
	};`

var (
	zeroHash       = common.Hash{}
	testChainDir   string
	keystoreDir    string
	nodeConfigJSON string
)

func buildAccountData(name, chatAddress string) *C.char {
	return C.CString(fmt.Sprintf(`{
		"name": "%s",
		"address": "%s"
	}`, name, chatAddress))
}

func buildSubAccountData(chatAddress string) *C.char {
	accs := []accounts.Account{
		{
			Wallet:  true,
			Chat:    true,
			Address: common.HexToAddress(chatAddress),
		},
	}
	data, _ := json.Marshal(accs)
	return C.CString(string(data))
}

func buildLoginParams(mainAccountAddress, chatAddress, password string) account.LoginParams {
	return account.LoginParams{
		ChatAddress: common.HexToAddress(chatAddress),
		Password:    password,
		MainAccount: common.HexToAddress(mainAccountAddress),
	}
}

func waitSignal(feed *event.Feed, event string, timeout time.Duration) error {
	events := make(chan signal.Envelope)
	sub := feed.Subscribe(events)
	defer sub.Unsubscribe()
	after := time.After(timeout)
	for {
		select {
		case envelope := <-events:
			if envelope.Type == event {
				return nil
			}
		case <-after:
			return fmt.Errorf("signal %v wasn't received in %v", event, timeout)
		}
	}
}

func createAccountAndLogin(t *testing.T, feed *event.Feed) account.Info {
	account1, _, err := statusBackend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	require.NoError(t, err)
	t.Logf("account created: {address: %s, key: %s}", account1.WalletAddress, account1.WalletPubKey)

	// select account
	loginResponse := APIResponse{}
	rawResponse := SaveAccountAndLogin(buildAccountData("test", account1.WalletAddress), C.CString(TestConfig.Account1.Password), C.CString(nodeConfigJSON), buildSubAccountData(account1.WalletAddress))
	require.NoError(t, json.Unmarshal([]byte(C.GoString(rawResponse)), &loginResponse))
	require.Empty(t, loginResponse.Error)
	require.NoError(t, waitSignal(feed, signal.EventLoggedIn, 5*time.Second))
	return account1
}

// nolint: deadcode
func testExportedAPI(t *testing.T) {
	testDir := filepath.Join(TestDataDir, TestNetworkNames[GetNetworkID()])
	_ = OpenAccounts(C.CString(testDir))
	// inject test accounts
	testKeyDir := filepath.Join(testDir, "keystore")
	_ = InitKeystore(C.CString(testKeyDir))
	require.NoError(t, ImportTestAccount(testKeyDir, GetAccount1PKFile()))
	require.NoError(t, ImportTestAccount(testKeyDir, GetAccount2PKFile()))

	// FIXME(tiabc): All of that is done because usage of cgo is not supported in tests.
	// Probably, there should be a cleaner way, for example, test cgo bindings in e2e tests
	// separately from other internal tests.
	// NOTE(dshulyak) tests are using same backend with same keystore. but after every test we explicitly logging out.
	tests := []struct {
		name string
		fn   func(t *testing.T, feed *event.Feed) bool
	}{
		{
			"StopResumeNode",
			testStopResumeNode,
		},

		{
			"RPCInProc",
			testCallRPC,
		},

		{
			"RPCPrivateAPI",
			testCallRPCWithPrivateAPI,
		},
		{
			"RPCPrivateClient",
			testCallPrivateRPCWithPrivateAPI,
		},
		{
			"VerifyAccountPassword",
			testVerifyAccountPassword,
		},
		{
			"RecoverAccount",
			testRecoverAccount,
		},
		{
			"LoginKeycard",
			testLoginWithKeycard,
		},
		{
			"AccountLoout",
			testAccountLogout,
		},
		{
			"SendTransaction",
			testSendTransactionWithLogin,
		},
		{
			"SendTransactionInvalidPassword",
			testSendTransactionInvalidPassword,
		},
		{
			"SendTransactionFailed",
			testFailedTransaction,
		},
		{
			"MultiAccount/Generate/Derive/StoreDerived/Load/Reset",
			testMultiAccountGenerateDeriveStoreLoadReset,
		},
		{
			"MultiAccount/ImportMnemonic/Derive",
			testMultiAccountImportMnemonicAndDerive,
		},
		{
			"MultiAccount/GenerateAndDerive",
			testMultiAccountGenerateAndDerive,
		},
		{
			"MultiAccount/Import/Store",
			testMultiAccountImportStore,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			feed := &event.Feed{}
			signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
				var envelope signal.Envelope
				require.NoError(t, json.Unmarshal([]byte(jsonEvent), &envelope))
				feed.Send(envelope)
			})
			defer func() {
				if snode := statusBackend.StatusNode(); snode == nil || !snode.IsRunning() {
					return
				}
				Logout()
				waitSignal(feed, signal.EventNodeStopped, 5*time.Second)
			}()
			require.True(t, tc.fn(t, feed))
		})
	}
}

func testVerifyAccountPassword(t *testing.T, feed *event.Feed) bool {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "accounts")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir) // nolint: errcheck

	if err = ImportTestAccount(tmpDir, GetAccount1PKFile()); err != nil {
		t.Fatal(err)
	}
	if err = ImportTestAccount(tmpDir, GetAccount2PKFile()); err != nil {
		t.Fatal(err)
	}

	// rename account file (to see that file's internals reviewed, when locating account key)
	accountFilePathOriginal := filepath.Join(tmpDir, GetAccount1PKFile())
	accountFilePath := filepath.Join(tmpDir, "foo"+TestConfig.Account1.WalletAddress+"bar.pk")
	if err := os.Rename(accountFilePathOriginal, accountFilePath); err != nil {
		t.Fatal(err)
	}

	response := APIResponse{}
	rawResponse := VerifyAccountPassword(
		C.CString(tmpDir),
		C.CString(TestConfig.Account1.WalletAddress),
		C.CString(TestConfig.Account1.Password))

	if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &response); err != nil {
		t.Errorf("cannot decode response (%s): %v", C.GoString(rawResponse), err)
		return false
	}
	if response.Error != "" {
		t.Errorf("unexpected error: %s", response.Error)
		return false
	}

	return true
}

func testStopResumeNode(t *testing.T, feed *event.Feed) bool { //nolint: gocyclo
	account1 := createAccountAndLogin(t, feed)
	whisperService, err := statusBackend.StatusNode().WhisperService()
	require.NoError(t, err)
	require.True(t, whisperService.HasKeyPair(account1.ChatPubKey), "whisper should have keypair")

	response := APIResponse{}
	rawResponse := StopNode()
	require.NoError(t, json.Unmarshal([]byte(C.GoString(rawResponse)), &response))
	require.Empty(t, response.Error)

	require.NoError(t, waitSignal(feed, signal.EventNodeStopped, 3*time.Second))
	response = APIResponse{}
	rawResponse = StartNode(C.CString(nodeConfigJSON))
	require.NoError(t, json.Unmarshal([]byte(C.GoString(rawResponse)), &response))
	require.Empty(t, response.Error)

	require.NoError(t, waitSignal(feed, signal.EventNodeReady, 5*time.Second))

	// now, verify that we still have account logged in
	whisperService, err = statusBackend.StatusNode().WhisperService()
	require.NoError(t, err)
	require.True(t, whisperService.HasKeyPair(account1.ChatPubKey))
	return true
}

func testCallRPC(t *testing.T, feed *event.Feed) bool {
	createAccountAndLogin(t, feed)
	expected := `{"jsonrpc":"2.0","id":64,"result":"0x47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad"}`
	rawResponse := CallRPC(C.CString(`{"jsonrpc":"2.0","method":"web3_sha3","params":["0x68656c6c6f20776f726c64"],"id":64}`))
	received := C.GoString(rawResponse)
	if expected != received {
		t.Errorf("unexpected response: expected: %v, got: %v", expected, received)
		return false
	}

	return true
}

func testCallRPCWithPrivateAPI(t *testing.T, feed *event.Feed) bool {
	createAccountAndLogin(t, feed)
	expected := `{"jsonrpc":"2.0","id":64,"error":{"code":-32601,"message":"the method admin_nodeInfo does not exist/is not available"}}`
	rawResponse := CallRPC(C.CString(`{"jsonrpc":"2.0","method":"admin_nodeInfo","params":[],"id":64}`))
	require.Equal(t, expected, C.GoString(rawResponse))
	return true
}

func testCallPrivateRPCWithPrivateAPI(t *testing.T, feed *event.Feed) bool {
	createAccountAndLogin(t, feed)
	rawResponse := CallPrivateRPC(C.CString(`{"jsonrpc":"2.0","method":"admin_nodeInfo","params":[],"id":64}`))
	received := C.GoString(rawResponse)
	if strings.Contains(received, "error") {
		t.Errorf("unexpected response containing error: %v", received)
		return false
	}

	return true
}

func testRecoverAccount(t *testing.T, feed *event.Feed) bool { //nolint: gocyclo
	keyStore := statusBackend.AccountManager().GetKeystore()
	require.NotNil(t, keyStore)
	// create an account
	accountInfo, mnemonic, err := statusBackend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	if err != nil {
		t.Errorf("could not create account: %v", err)
		return false
	}
	t.Logf("Account created: {address: %s, key: %s, mnemonic:%s}", accountInfo.WalletAddress, accountInfo.WalletPubKey, mnemonic)

	// try recovering using password + mnemonic
	recoverAccountResponse := AccountInfo{}
	rawResponse := RecoverAccount(C.CString(TestConfig.Account1.Password), C.CString(mnemonic))

	if err = json.Unmarshal([]byte(C.GoString(rawResponse)), &recoverAccountResponse); err != nil {
		t.Errorf("cannot decode RecoverAccount response (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if recoverAccountResponse.Error != "" {
		t.Errorf("recover account failed: %v", recoverAccountResponse.Error)
		return false
	}

	if recoverAccountResponse.Address != recoverAccountResponse.WalletAddress ||
		recoverAccountResponse.PubKey != recoverAccountResponse.WalletPubKey {
		t.Error("for backward compatibility pubkey/address should be equal to walletAddress/walletPubKey")
	}

	walletAddressCheck, walletPubKeyCheck := recoverAccountResponse.Address, recoverAccountResponse.PubKey
	chatAddressCheck, chatPubKeyCheck := recoverAccountResponse.ChatAddress, recoverAccountResponse.ChatPubKey

	if accountInfo.WalletAddress != walletAddressCheck || accountInfo.WalletPubKey != walletPubKeyCheck {
		t.Error("recover wallet account details failed to pull the correct details")
	}

	if accountInfo.ChatAddress != chatAddressCheck || accountInfo.ChatPubKey != chatPubKeyCheck {
		t.Error("recover chat account details failed to pull the correct details")
	}

	// now test recovering, but make sure that account/key file is removed i.e. simulate recovering on a new device
	account, err := account.ParseAccountString(accountInfo.WalletAddress)
	if err != nil {
		t.Errorf("can not get account from address: %v", err)
	}

	account, key, err := keyStore.AccountDecryptedKey(account, TestConfig.Account1.Password)
	if err != nil {
		t.Errorf("can not obtain decrypted account key: %v", err)
		return false
	}
	extChild2String := key.ExtendedKey.String()

	if err = keyStore.Delete(account, TestConfig.Account1.Password); err != nil {
		t.Errorf("cannot remove account: %v", err)
	}

	recoverAccountResponse = AccountInfo{}
	rawResponse = RecoverAccount(C.CString(TestConfig.Account1.Password), C.CString(mnemonic))

	if err = json.Unmarshal([]byte(C.GoString(rawResponse)), &recoverAccountResponse); err != nil {
		t.Errorf("cannot decode RecoverAccount response (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if recoverAccountResponse.Error != "" {
		t.Errorf("recover account failed (for non-cached account): %v", recoverAccountResponse.Error)
		return false
	}
	walletAddressCheck, walletPubKeyCheck = recoverAccountResponse.Address, recoverAccountResponse.PubKey
	if accountInfo.WalletAddress != walletAddressCheck || accountInfo.WalletPubKey != walletPubKeyCheck {
		t.Error("recover wallet account details failed to pull the correct details (for non-cached account)")
	}

	chatAddressCheck, chatPubKeyCheck = recoverAccountResponse.ChatAddress, recoverAccountResponse.ChatPubKey
	if accountInfo.ChatAddress != chatAddressCheck || accountInfo.ChatPubKey != chatPubKeyCheck {
		t.Error("recover chat account details failed to pull the correct details (for non-cached account)")
	}

	// make sure that extended key exists and is imported ok too
	_, key, err = keyStore.AccountDecryptedKey(account, TestConfig.Account1.Password)
	if err != nil {
		t.Errorf("can not obtain decrypted account key: %v", err)
		return false
	}
	if extChild2String != key.ExtendedKey.String() {
		t.Errorf("CKD#2 key mismatch, expected: %s, got: %s", extChild2String, key.ExtendedKey.String())
	}

	// make sure that calling import several times, just returns from cache (no error is expected)
	recoverAccountResponse = AccountInfo{}
	rawResponse = RecoverAccount(C.CString(TestConfig.Account1.Password), C.CString(mnemonic))

	if err = json.Unmarshal([]byte(C.GoString(rawResponse)), &recoverAccountResponse); err != nil {
		t.Errorf("cannot decode RecoverAccount response (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if recoverAccountResponse.Error != "" {
		t.Errorf("recover account failed (for non-cached account): %v", recoverAccountResponse.Error)
		return false
	}
	walletAddressCheck, walletPubKeyCheck = recoverAccountResponse.Address, recoverAccountResponse.PubKey
	if accountInfo.WalletAddress != walletAddressCheck || accountInfo.WalletPubKey != walletPubKeyCheck {
		t.Error("recover wallet account details failed to pull the correct details (for non-cached account)")
	}

	chatAddressCheck, chatPubKeyCheck = recoverAccountResponse.ChatAddress, recoverAccountResponse.ChatPubKey
	if accountInfo.ChatAddress != chatAddressCheck || accountInfo.ChatPubKey != chatPubKeyCheck {
		t.Error("recover chat account details failed to pull the correct details (for non-cached account)")
	}

	loginResponse := APIResponse{}
	rawResponse = SaveAccountAndLogin(buildAccountData("test", walletAddressCheck), C.CString(TestConfig.Account1.Password), C.CString(nodeConfigJSON), buildSubAccountData(walletAddressCheck))
	require.NoError(t, json.Unmarshal([]byte(C.GoString(rawResponse)), &loginResponse))
	require.Empty(t, loginResponse.Error)
	require.NoError(t, waitSignal(feed, signal.EventLoggedIn, 5*time.Second))

	// time to login with recovered data
	whisperService, err := statusBackend.StatusNode().WhisperService()
	if err != nil {
		t.Errorf("whisper service not running: %v", err)
	}

	if !whisperService.HasKeyPair(chatPubKeyCheck) {
		t.Errorf("identity not injected into whisper: %v", err)
	}

	return true
}

func testLoginWithKeycard(t *testing.T, feed *event.Feed) bool { //nolint: gocyclo
	createAccountAndLogin(t, feed)
	chatPrivKey, err := crypto.GenerateKey()
	if err != nil {
		t.Errorf("error generating chat key")
		return false
	}
	chatPrivKeyHex := hex.EncodeToString(crypto.FromECDSA(chatPrivKey))

	encryptionPrivKey, err := crypto.GenerateKey()
	if err != nil {
		t.Errorf("error generating encryption key")
		return false
	}
	encryptionPrivKeyHex := hex.EncodeToString(crypto.FromECDSA(encryptionPrivKey))

	whisperService, err := statusBackend.StatusNode().WhisperService()
	if err != nil {
		t.Errorf("whisper service not running: %v", err)
	}

	chatPubKeyHex := types.EncodeHex(crypto.FromECDSAPub(&chatPrivKey.PublicKey))
	if whisperService.HasKeyPair(chatPubKeyHex) {
		t.Error("identity already present in whisper")
		return false
	}

	loginResponse := APIResponse{}
	rawResponse := LoginWithKeycard(C.CString(chatPrivKeyHex), C.CString(encryptionPrivKeyHex))

	if err = json.Unmarshal([]byte(C.GoString(rawResponse)), &loginResponse); err != nil {
		t.Errorf("cannot decode LoginWithKeycard response (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if loginResponse.Error != "" {
		t.Errorf("Test failed: could not login with keycard: %v", err)
		return false
	}

	if !whisperService.HasKeyPair(chatPubKeyHex) {
		t.Error("identity not present in whisper after logging in with keycard")
		return false
	}

	return true
}

func testAccountLogout(t *testing.T, feed *event.Feed) bool {
	accountInfo := createAccountAndLogin(t, feed)
	whisperService, err := statusBackend.StatusNode().WhisperService()
	if err != nil {
		t.Errorf("whisper service not running: %v", err)
		return false
	}

	if !whisperService.HasKeyPair(accountInfo.ChatPubKey) {
		t.Error("identity not injected into whisper")
		return false
	}

	logoutResponse := APIResponse{}
	rawResponse := Logout()

	if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &logoutResponse); err != nil {
		t.Errorf("cannot decode RecoverAccount response (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if logoutResponse.Error != "" {
		t.Errorf("cannot logout: %v", logoutResponse.Error)
		return false
	}

	// now, logout and check if identity is removed indeed
	if whisperService.HasKeyPair(accountInfo.ChatPubKey) {
		t.Error("identity not cleared from whisper")
		return false
	}

	return true
}

type jsonrpcAnyResponse struct {
	Result json.RawMessage `json:"result"`
	jsonrpcErrorResponse
}

func testSendTransactionWithLogin(t *testing.T, feed *event.Feed) bool {
	loginResponse := APIResponse{}
	rawResponse := SaveAccountAndLogin(buildAccountData("test", TestConfig.Account1.WalletAddress), C.CString(TestConfig.Account1.Password), C.CString(nodeConfigJSON), buildSubAccountData(TestConfig.Account1.WalletAddress))
	require.NoError(t, json.Unmarshal([]byte(C.GoString(rawResponse)), &loginResponse))
	require.Empty(t, loginResponse.Error)
	require.NoError(t, waitSignal(feed, signal.EventLoggedIn, 5*time.Second))
	EnsureNodeSync(statusBackend.StatusNode().EnsureSync)

	args, err := json.Marshal(transactions.SendTxArgs{
		From:  account.FromAddress(TestConfig.Account1.WalletAddress),
		To:    account.ToAddress(TestConfig.Account2.WalletAddress),
		Value: (*hexutil.Big)(big.NewInt(1000000000000)),
	})
	if err != nil {
		t.Errorf("failed to marshal errors: %v", err)
		return false
	}
	rawResult := SendTransaction(C.CString(string(args)), C.CString(TestConfig.Account1.Password))

	var result jsonrpcAnyResponse
	if err := json.Unmarshal([]byte(C.GoString(rawResult)), &result); err != nil {
		t.Errorf("failed to unmarshal rawResult '%s': %v", C.GoString(rawResult), err)
		return false
	}
	if result.Error.Message != "" {
		t.Errorf("failed to send transaction: %v", result.Error)
		return false
	}
	hash := common.BytesToHash(result.Result)
	if reflect.DeepEqual(hash, common.Hash{}) {
		t.Errorf("response hash empty: %s", hash.Hex())
		return false
	}

	return true
}

func testSendTransactionInvalidPassword(t *testing.T, feed *event.Feed) bool {
	acc := createAccountAndLogin(t, feed)
	EnsureNodeSync(statusBackend.StatusNode().EnsureSync)

	args, err := json.Marshal(transactions.SendTxArgs{
		From:  common.HexToAddress(acc.WalletAddress),
		To:    account.ToAddress(TestConfig.Account2.WalletAddress),
		Value: (*hexutil.Big)(big.NewInt(1000000000000)),
	})
	if err != nil {
		t.Errorf("failed to marshal errors: %v", err)
		return false
	}
	rawResult := SendTransaction(C.CString(string(args)), C.CString("invalid password"))

	var result jsonrpcAnyResponse
	if err := json.Unmarshal([]byte(C.GoString(rawResult)), &result); err != nil {
		t.Errorf("failed to unmarshal rawResult '%s': %v", C.GoString(rawResult), err)
		return false
	}
	if result.Error.Message != keystore.ErrDecrypt.Error() {
		t.Errorf("invalid result: %q", result)
		return false
	}

	return true
}

func testFailedTransaction(t *testing.T, feed *event.Feed) bool {
	createAccountAndLogin(t, feed)
	EnsureNodeSync(statusBackend.StatusNode().EnsureSync)

	args, err := json.Marshal(transactions.SendTxArgs{
		From:  *account.ToAddress(TestConfig.Account1.WalletAddress),
		To:    account.ToAddress(TestConfig.Account2.WalletAddress),
		Value: (*hexutil.Big)(big.NewInt(1000000000000)),
	})
	if err != nil {
		t.Errorf("failed to marshal errors: %v", err)
		return false
	}
	rawResult := SendTransaction(C.CString(string(args)), C.CString(TestConfig.Account1.Password))

	var result jsonrpcAnyResponse
	if err := json.Unmarshal([]byte(C.GoString(rawResult)), &result); err != nil {
		t.Errorf("failed to unmarshal rawResult '%s': %v", C.GoString(rawResult), err)
		return false
	}

	if result.Error.Message != transactions.ErrAccountDoesntExist.Error() {
		t.Errorf("expected error to be ErrAccountDoesntExist, got %s", result.Error.Message)
		return false
	}

	if result.Result != nil {
		t.Errorf("expected result to be nil")
		return false
	}

	return true

}

//nolint: deadcode
func testValidateNodeConfig(t *testing.T, config string, fn func(*testing.T, APIDetailedResponse)) {
	result := ValidateNodeConfig(C.CString(config))

	var resp APIDetailedResponse

	err := json.Unmarshal([]byte(C.GoString(result)), &resp)
	require.NoError(t, err)

	fn(t, resp)
}
