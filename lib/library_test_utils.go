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

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/keystore"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/signal"
	. "github.com/status-im/status-go/t/utils" //nolint: golint
	"github.com/status-im/status-go/transactions"
)
import "github.com/status-im/status-go/params"

var (
	testChainDir   string
	keystoreDir    string
	nodeConfigJSON string
)

func buildAccountData(name, chatAddress string) *C.char {
	return C.CString(fmt.Sprintf(`{
		"name": "%s",
		"key-uid": "%s"
	}`, name, chatAddress))
}

func buildAccountSettings(name string) *C.char {
	return C.CString(fmt.Sprintf(`{
"address": "0xdC540f3745Ff2964AFC1171a5A0DD726d1F6B472",
"current-network": "mainnet_rpc",
"dapps-address": "0xD1300f99fDF7346986CbC766903245087394ecd0",
"eip1581-address": "0xB1DDDE9235a541d1344550d969715CF43982de9f",
"installation-id": "d3efcff6-cffa-560e-a547-21d3858cbc51",
"key-uid": "0x4e8129f3edfc004875be17bf468a784098a9f69b53c095be1f52deff286935ab",
"last-derived-path": 0,
"name": "%s",
"networks/networks": {},
"photo-path": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADIAAAAyCAIAAACRXR/mAAAAjklEQVR4nOzXwQmFMBAAUZXUYh32ZB32ZB02sxYQQSZGsod55/91WFgSS0RM+SyjA56ZRZhFmEWYRRT6h+M6G16zrxv6fdJpmUWYRbxsYr13dKfanpN0WmYRZhGzXz6AWYRZRIfbaX26fT9Jk07LLMIsosPt9I/dTDotswizCG+nhFmEWYRZhFnEHQAA///z1CFkYamgfQAAAABJRU5ErkJggg==",
"preview-privacy": false,
"public-key": "0x04211fe0f69772ecf7eb0b5bfc7678672508a9fb01f2d699096f0d59ef7fe1a0cb1e648a80190db1c0f5f088872444d846f2956d0bd84069f3f9f69335af852ac0",
"signing-phrase": "yurt joey vibe",
"wallet-root-address": "0x3B591fd819F86D0A6a2EF2Bcb94f77807a7De1a6"
}`, name))
}

func buildSubAccountData(chatAddress string) *C.char {
	accs := []accounts.Account{
		{
			Wallet:  true,
			Chat:    true,
			Address: types.HexToAddress(chatAddress),
		},
	}
	data, _ := json.Marshal(accs)
	return C.CString(string(data))
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

	nodeConfig, _ := params.NewConfigFromJSON(nodeConfigJSON)
	nodeConfig.KeyStoreDir = "keystore"
	nodeConfig.DataDir = "/"
	cnf, err := json.Marshal(nodeConfig)
	require.NoError(t, err)

	signalErrC := make(chan error, 1)
	go func() {
		signalErrC <- waitSignal(feed, signal.EventLoggedIn, 10*time.Second)
	}()

	// SaveAccountAndLogin must be called only once when an account is created.
	// If the account already exists, Login should be used.
	rawResponse := SaveAccountAndLogin(
		buildAccountData("test", account1.WalletAddress),
		C.CString(TestConfig.Account1.Password),
		buildAccountSettings("test"),
		C.CString(string(cnf)),
		buildSubAccountData(account1.WalletAddress),
	)
	var loginResponse APIResponse
	require.NoError(t, json.Unmarshal([]byte(C.GoString(rawResponse)), &loginResponse))
	require.Empty(t, loginResponse.Error)
	require.NoError(t, <-signalErrC)
	return account1
}

func loginUsingAccount(t *testing.T, feed *event.Feed, addr string) {
	signalErrC := make(chan error, 1)
	go func() {
		signalErrC <- waitSignal(feed, signal.EventLoggedIn, 5*time.Second)
	}()

	nodeConfig, _ := params.NewConfigFromJSON(nodeConfigJSON)
	nodeConfig.KeyStoreDir = "keystore"
	nodeConfig.DataDir = "/"
	cnf, _ := json.Marshal(nodeConfig)

	// SaveAccountAndLogin must be called only once when an account is created.
	// If the account already exists, Login should be used.
	rawResponse := SaveAccountAndLogin(
		buildAccountData("test", addr),
		C.CString(TestConfig.Account1.Password),
		buildAccountSettings("test"),
		C.CString(string(cnf)),
		buildSubAccountData(addr),
	)
	var loginResponse APIResponse
	require.NoError(t, json.Unmarshal([]byte(C.GoString(rawResponse)), &loginResponse))
	require.Empty(t, loginResponse.Error)
	require.NoError(t, <-signalErrC)
}

// nolint: deadcode
func testExportedAPI(t *testing.T) {
	// All of that is done because usage of cgo is not supported in tests.
	// Probably, there should be a cleaner way, for example, test cgo bindings in e2e tests
	// separately from other internal tests.
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
			"AccountLogout",
			testAccountLogout,
		},
		{
			"SendTransactionWithLogin",
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
			testDir := filepath.Join(TestDataDir, TestNetworkNames[GetNetworkID()])
			defer os.RemoveAll(testDir)

			err := os.MkdirAll(testDir, os.ModePerm)
			require.NoError(t, err)

			testKeyDir := filepath.Join(testDir, "keystore")
			require.NoError(t, ImportTestAccount(testKeyDir, GetAccount1PKFile()))
			require.NoError(t, ImportTestAccount(testKeyDir, GetAccount2PKFile()))

			// Inject test accounts.
			response := InitKeystore(C.CString(testKeyDir))
			if C.GoString(response) != `{"error":""}` {
				t.Fatalf("failed to InitKeystore: %v", C.GoString(response))
			}

			// Initialize the accounts database. It must be called
			// after the test account got injected.
			result := OpenAccounts(C.CString(testDir))
			if C.GoString(response) != `{"error":""}` {
				t.Fatalf("OpenAccounts() failed: %v", C.GoString(result))
			}

			// Create a custom signals handler so that we can examine them here.
			feed := &event.Feed{}
			signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
				var envelope signal.Envelope
				require.NoError(t, json.Unmarshal([]byte(jsonEvent), &envelope))
				feed.Send(envelope)
			})
			defer func() {
				errCh := make(chan error, 1)
				go func() {
					errCh <- waitSignal(feed, signal.EventNodeStopped, 5*time.Second)
				}()
				if n := statusBackend.StatusNode(); n == nil || !n.IsRunning() {
					return
				}
				Logout()
				require.NoError(t, <-errCh)
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

	errC := make(chan error, 1)
	go func() {
		errC <- waitSignal(feed, signal.EventLoggedIn, 5*time.Second)
	}()
	rawResponse = SaveAccountAndLogin(buildAccountData("test", walletAddressCheck), C.CString(TestConfig.Account1.Password), buildAccountSettings("test"), C.CString(nodeConfigJSON), buildSubAccountData(walletAddressCheck))
	loginResponse := APIResponse{}
	require.NoError(t, json.Unmarshal([]byte(C.GoString(rawResponse)), &loginResponse))
	require.Empty(t, loginResponse.Error)
	require.NoError(t, <-errC)

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
		return false
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
	loginUsingAccount(t, feed, TestConfig.Account1.WalletAddress)
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
	hash := types.BytesToHash(result.Result)
	if reflect.DeepEqual(hash, types.Hash{}) {
		t.Errorf("response hash empty: %s", hash.Hex())
		return false
	}

	return true
}

func testSendTransactionInvalidPassword(t *testing.T, feed *event.Feed) bool {
	acc := createAccountAndLogin(t, feed)
	EnsureNodeSync(statusBackend.StatusNode().EnsureSync)

	args, err := json.Marshal(transactions.SendTxArgs{
		From:  types.HexToAddress(acc.WalletAddress),
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
