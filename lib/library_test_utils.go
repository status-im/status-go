// +build e2e_test

// This is a file with e2e tests for C bindings written in library.go.
// As a CGO file, it can't have `_test.go` suffix as it's not allowed by Go.
// At the same time, we don't want this file to be included in the binaries.
// This is why `e2e_test` tag was introduced. Without it, this file is excluded
// from the build. Providing this tag will include this file into the build
// and that's what is done while running e2e tests for `lib/` package.

package main

import "C"
import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-go/account"
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
	zeroHash       = gethcommon.Hash{}
	testChainDir   string
	nodeConfigJSON string
)

func buildLoginParamsJSON(chatAddress, password string) *C.char {
	return C.CString(fmt.Sprintf(`{
		"chatAddress": "%s",
		"password": "%s",
		"mainAccount": "%s"
	}`, chatAddress, password, chatAddress))
}

func buildLoginParams(mainAccountAddress, chatAddress, password string) account.LoginParams {
	return account.LoginParams{
		ChatAddress: gethcommon.HexToAddress(chatAddress),
		Password:    password,
		MainAccount: gethcommon.HexToAddress(mainAccountAddress),
	}
}

func init() {
	testChainDir = filepath.Join(TestDataDir, TestNetworkNames[GetNetworkID()])

	nodeConfigJSON = `{
	"NetworkId": ` + strconv.Itoa(GetNetworkID()) + `,
	"DataDir": "` + testChainDir + `",
	"KeyStoreDir": "` + filepath.Join(testChainDir, "keystore") + `",
	"HTTPPort": ` + strconv.Itoa(TestConfig.Node.HTTPPort) + `,
	"LogLevel": "INFO",
	"NoDiscovery": true,
	"LightEthConfig": {
		"Enabled": true
	},
	"WhisperConfig": {
		"Enabled": true,
		"DataDir": "` + path.Join(testChainDir, "wnode") + `",
		"EnableNTPSync": false
	},
	"ShhextConfig": {
	    "BackupDisabledDataDir": "` + testChainDir + `"
	}
}`
}

// nolint: deadcode
func testExportedAPI(t *testing.T, done chan struct{}) {
	<-startTestNode(t)
	defer func() {
		done <- struct{}{}
	}()

	// prepare accounts
	testKeyDir := filepath.Join(testChainDir, "keystore")
	if err := ImportTestAccount(testKeyDir, GetAccount1PKFile()); err != nil {
		panic(err)
	}
	if err := ImportTestAccount(testKeyDir, GetAccount2PKFile()); err != nil {
		panic(err)
	}
	_ = InitKeystore(C.CString(testKeyDir))

	// FIXME(tiabc): All of that is done because usage of cgo is not supported in tests.
	// Probably, there should be a cleaner way, for example, test cgo bindings in e2e tests
	// separately from other internal tests.
	// FIXME(@jekamas): ATTENTION! this tests depends on each other!
	tests := []struct {
		name string
		fn   func(t *testing.T) bool
	}{
		{
			"stop/resume node",
			testStopResumeNode,
		},
		{
			"call RPC on in-proc handler",
			testCallRPC,
		},
		{
			"call private API using RPC",
			testCallRPCWithPrivateAPI,
		},
		{
			"call private API using private RPC client",
			testCallPrivateRPCWithPrivateAPI,
		},
		{
			"verify account password",
			testVerifyAccountPassword,
		},
		{
			"recover account",
			testRecoverAccount,
		},
		{
			"account select/login",
			testAccountSelect,
		},
		{
			"login with keycard",
			testLoginWithKeycard,
		},
		{
			"account logout",
			testAccountLogout,
		},
		{
			"send transaction",
			testSendTransaction,
		},
		{
			"send transaction with invalid password",
			testSendTransactionInvalidPassword,
		},
		{
			"failed single transaction",
			testFailedTransaction,
		},
		{
			"MultiAccount - Generate/Derive/StoreDerived/Load/Reset",
			testMultiAccountGenerateDeriveStoreLoadReset,
		},
		{
			"MultiAccount - ImportMnemonic/Derive",
			testMultiAccountImportMnemonicAndDerive,
		},
		{
			"MultiAccount - GenerateAndDerive",
			testMultiAccountGenerateAndDerive,
		},
		{
			"MultiAccount - Import/Store",
			testMultiAccountImportStore,
		},
	}

	for _, test := range tests {
		t.Logf("=== RUN   %s", test.name)
		if ok := test.fn(t); !ok {
			t.Logf("=== FAILED   %s", test.name)
			break
		}
	}
}

func testVerifyAccountPassword(t *testing.T) bool {
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

//@TODO(adam): quarantined this test until it uses a different directory.
//nolint: deadcode
func testResetChainData(t *testing.T) bool {
	t.Skip()

	resetChainDataResponse := APIResponse{}
	rawResponse := ResetChainData()

	if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &resetChainDataResponse); err != nil {
		t.Errorf("cannot decode ResetChainData response (%s): %v", C.GoString(rawResponse), err)
		return false
	}
	if resetChainDataResponse.Error != "" {
		t.Errorf("unexpected error: %s", resetChainDataResponse.Error)
		return false
	}

	EnsureNodeSync(statusBackend.StatusNode().EnsureSync)
	testSendTransaction(t)

	return true
}

func testStopResumeNode(t *testing.T) bool { //nolint: gocyclo
	// to make sure that we start with empty account (which might have gotten populated during previous tests)
	if err := statusBackend.Logout(); err != nil {
		t.Fatal(err)
	}

	whisperService, err := statusBackend.StatusNode().WhisperService()
	if err != nil {
		t.Errorf("whisper service not running: %v", err)
	}

	// create an account
	account1, _, err := statusBackend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	if err != nil {
		t.Errorf("could not create account: %v", err)
		return false
	}
	t.Logf("account created: {address: %s, key: %s}", account1.WalletAddress, account1.WalletPubKey)

	// make sure that identity is not (yet injected)
	if whisperService.HasKeyPair(account1.ChatPubKey) {
		t.Error("identity already present in whisper")
	}

	// select account
	loginResponse := APIResponse{}
	rawResponse := Login(buildLoginParamsJSON(account1.WalletAddress, TestConfig.Account1.Password))

	if err = json.Unmarshal([]byte(C.GoString(rawResponse)), &loginResponse); err != nil {
		t.Errorf("cannot decode RecoverAccount response (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if loginResponse.Error != "" {
		t.Errorf("could not select account: %v", err)
		return false
	}
	if !whisperService.HasKeyPair(account1.ChatPubKey) {
		t.Errorf("identity not injected into whisper: %v", err)
	}

	// stop and resume node, then make sure that selected account is still selected
	// nolint: dupl
	stopNodeFn := func() bool {
		response := APIResponse{}
		// FIXME(tiabc): Implement https://github.com/status-im/status-go/issues/254 to avoid
		// 9-sec timeout below after stopping the node.
		rawResponse = StopNode()

		if err = json.Unmarshal([]byte(C.GoString(rawResponse)), &response); err != nil {
			t.Errorf("cannot decode StopNode response (%s): %v", C.GoString(rawResponse), err)
			return false
		}
		if response.Error != "" {
			t.Errorf("unexpected error: %s", response.Error)
			return false
		}

		return true
	}

	// nolint: dupl
	resumeNodeFn := func() bool {
		response := APIResponse{}
		// FIXME(tiabc): Implement https://github.com/status-im/status-go/issues/254 to avoid
		// 10-sec timeout below after resuming the node.
		rawResponse = StartNode(C.CString(nodeConfigJSON))

		if err = json.Unmarshal([]byte(C.GoString(rawResponse)), &response); err != nil {
			t.Errorf("cannot decode StartNode response (%s): %v", C.GoString(rawResponse), err)
			return false
		}
		if response.Error != "" {
			t.Errorf("unexpected error: %s", response.Error)
			return false
		}

		return true
	}

	if !stopNodeFn() {
		return false
	}

	time.Sleep(9 * time.Second) // allow to stop

	if !resumeNodeFn() {
		return false
	}

	time.Sleep(10 * time.Second) // allow to start (instead of using blocking version of start, of filter event)

	// now, verify that we still have account logged in
	whisperService, err = statusBackend.StatusNode().WhisperService()
	if err != nil {
		t.Errorf("whisper service not running: %v", err)
	}
	if !whisperService.HasKeyPair(account1.ChatPubKey) {
		t.Errorf("identity evicted from whisper on node restart: %v", err)
	}

	// additionally, let's complete transaction (just to make sure that node lives through pause/resume w/o issues)
	testSendTransaction(t)

	return true
}

func testCallRPC(t *testing.T) bool {
	expected := `{"jsonrpc":"2.0","id":64,"result":"0x47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad"}`
	rawResponse := CallRPC(C.CString(`{"jsonrpc":"2.0","method":"web3_sha3","params":["0x68656c6c6f20776f726c64"],"id":64}`))
	received := C.GoString(rawResponse)
	if expected != received {
		t.Errorf("unexpected response: expected: %v, got: %v", expected, received)
		return false
	}

	return true
}

func testCallRPCWithPrivateAPI(t *testing.T) bool {
	expected := `{"jsonrpc":"2.0","id":64,"error":{"code":-32601,"message":"The method admin_nodeInfo does not exist/is not available"}}`
	rawResponse := CallRPC(C.CString(`{"jsonrpc":"2.0","method":"admin_nodeInfo","params":[],"id":64}`))
	received := C.GoString(rawResponse)
	if expected != received {
		t.Errorf("unexpected response: expected: %v, got: %v", expected, received)
		return false
	}

	return true
}

func testCallPrivateRPCWithPrivateAPI(t *testing.T) bool {
	rawResponse := CallPrivateRPC(C.CString(`{"jsonrpc":"2.0","method":"admin_nodeInfo","params":[],"id":64}`))
	received := C.GoString(rawResponse)
	if strings.Contains(received, "error") {
		t.Errorf("unexpected response containing error: %v", received)
		return false
	}

	return true
}

func testRecoverAccount(t *testing.T) bool { //nolint: gocyclo
	keyStore := statusBackend.AccountManager().GetKeystore()
	if keyStore == nil {
		t.Errorf("keystore is nil")
		return false
	}

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

	// time to login with recovered data
	whisperService, err := statusBackend.StatusNode().WhisperService()
	if err != nil {
		t.Errorf("whisper service not running: %v", err)
	}

	// make sure that identity is not (yet injected)
	if whisperService.HasKeyPair(chatPubKeyCheck) {
		t.Error("identity already present in whisper")
	}
	err = statusBackend.SelectAccount(buildLoginParams(walletAddressCheck, chatAddressCheck, TestConfig.Account1.Password))
	if err != nil {
		t.Errorf("Test failed: could not select account: %v", err)
		return false
	}
	if !whisperService.HasKeyPair(chatPubKeyCheck) {
		t.Errorf("identity not injected into whisper: %v", err)
	}

	return true
}

func testAccountSelect(t *testing.T) bool { //nolint: gocyclo
	// test to see if the account was injected in whisper
	whisperService, err := statusBackend.StatusNode().WhisperService()
	if err != nil {
		t.Errorf("whisper service not running: %v", err)
	}

	// create an account
	accountInfo1, _, err := statusBackend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	if err != nil {
		t.Errorf("could not create account: %v", err)
		return false
	}
	t.Logf("Account created: {address: %s, key: %s}", accountInfo1.WalletAddress, accountInfo1.WalletPubKey)

	accountInfo2, _, err := statusBackend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	if err != nil {
		t.Error("Test failed: could not create account")
		return false
	}
	t.Logf("Account created: {address: %s, key: %s}", accountInfo2.WalletAddress, accountInfo2.WalletPubKey)

	// make sure that identity is not (yet injected)
	if whisperService.HasKeyPair(accountInfo1.ChatPubKey) {
		t.Error("identity already present in whisper")
	}

	// try selecting with wrong password
	loginResponse := APIResponse{}
	rawResponse := Login(buildLoginParamsJSON(accountInfo1.WalletAddress, "wrongPassword"))

	if err = json.Unmarshal([]byte(C.GoString(rawResponse)), &loginResponse); err != nil {
		t.Errorf("cannot decode RecoverAccount response (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if loginResponse.Error == "" {
		t.Error("select account is expected to throw error: wrong password used")
		return false
	}

	loginResponse = APIResponse{}
	rawResponse = Login(buildLoginParamsJSON(accountInfo1.WalletAddress, TestConfig.Account1.Password))

	if err = json.Unmarshal([]byte(C.GoString(rawResponse)), &loginResponse); err != nil {
		t.Errorf("cannot decode RecoverAccount response (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if loginResponse.Error != "" {
		t.Errorf("Test failed: could not select account: %v", err)
		return false
	}
	if !whisperService.HasKeyPair(accountInfo1.ChatPubKey) {
		t.Errorf("identity not injected into whisper: %v", err)
	}

	// select another account, make sure that previous account is wiped out from Whisper cache
	if whisperService.HasKeyPair(accountInfo2.ChatPubKey) {
		t.Error("identity already present in whisper")
	}

	loginResponse = APIResponse{}
	rawResponse = Login(buildLoginParamsJSON(accountInfo2.WalletAddress, TestConfig.Account1.Password))

	if err = json.Unmarshal([]byte(C.GoString(rawResponse)), &loginResponse); err != nil {
		t.Errorf("cannot decode RecoverAccount response (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if loginResponse.Error != "" {
		t.Errorf("Test failed: could not select account: %v", loginResponse.Error)
		return false
	}
	if !whisperService.HasKeyPair(accountInfo2.ChatPubKey) {
		t.Errorf("identity not injected into whisper: %v", err)
	}
	if whisperService.HasKeyPair(accountInfo1.ChatPubKey) {
		t.Error("identity should be removed, but it is still present in whisper")
	}

	return true
}

func testLoginWithKeycard(t *testing.T) bool { //nolint: gocyclo
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

	chatPubKeyHex := hexutil.Encode(crypto.FromECDSAPub(&chatPrivKey.PublicKey))
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

func testAccountLogout(t *testing.T) bool {
	whisperService, err := statusBackend.StatusNode().WhisperService()
	if err != nil {
		t.Errorf("whisper service not running: %v", err)
		return false
	}

	// create an account
	accountInfo, _, err := statusBackend.AccountManager().CreateAccount(TestConfig.Account1.Password)
	if err != nil {
		t.Errorf("could not create account: %v", err)
		return false
	}

	// make sure that identity doesn't exist (yet) in Whisper
	if whisperService.HasKeyPair(accountInfo.ChatPubKey) {
		t.Error("identity already present in whisper")
		return false
	}

	// select/login
	err = statusBackend.SelectAccount(buildLoginParams(accountInfo.WalletAddress, accountInfo.ChatAddress, TestConfig.Account1.Password))
	if err != nil {
		t.Errorf("Test failed: could not select account: %v", err)
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

func testSendTransaction(t *testing.T) bool {
	EnsureNodeSync(statusBackend.StatusNode().EnsureSync)

	// log into account from which transactions will be sent
	if err := statusBackend.SelectAccount(buildLoginParams(TestConfig.Account1.WalletAddress, TestConfig.Account1.ChatAddress, TestConfig.Account1.Password)); err != nil {
		t.Errorf("cannot select account: %v. Error %q", TestConfig.Account1.WalletAddress, err)
		return false
	}

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
	hash := gethcommon.BytesToHash(result.Result)
	if reflect.DeepEqual(hash, gethcommon.Hash{}) {
		t.Errorf("response hash empty: %s", hash.Hex())
		return false
	}

	return true
}

func testSendTransactionInvalidPassword(t *testing.T) bool {
	EnsureNodeSync(statusBackend.StatusNode().EnsureSync)

	// log into account from which transactions will be sent
	if err := statusBackend.SelectAccount(buildLoginParams(
		TestConfig.Account1.WalletAddress,
		TestConfig.Account1.ChatAddress,
		TestConfig.Account1.Password,
	)); err != nil {
		t.Errorf("cannot select account: %v. Error %q", TestConfig.Account1.WalletAddress, err)
		return false
	}

	args, err := json.Marshal(transactions.SendTxArgs{
		From:  account.FromAddress(TestConfig.Account1.WalletAddress),
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

func testFailedTransaction(t *testing.T) bool {
	EnsureNodeSync(statusBackend.StatusNode().EnsureSync)

	// log into wrong account in order to get selectedAccount error
	if err := statusBackend.SelectAccount(buildLoginParams(TestConfig.Account2.WalletAddress, TestConfig.Account2.ChatAddress, TestConfig.Account2.Password)); err != nil {
		t.Errorf("cannot select account: %v. Error %q", TestConfig.Account2.WalletAddress, err)
		return false
	}

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

	if result.Error.Message != transactions.ErrInvalidTxSender.Error() {
		t.Errorf("expected error to be ErrInvalidTxSender, got %s", result.Error.Message)
		return false
	}

	if result.Result != nil {
		t.Errorf("expected result to be nil")
		return false
	}

	return true

}

func startTestNode(t *testing.T) <-chan struct{} {
	testDir := filepath.Join(TestDataDir, TestNetworkNames[GetNetworkID()])

	syncRequired := false
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		syncRequired = true
	}

	// inject test accounts
	testKeyDir := filepath.Join(testDir, "keystore")
	if err := ImportTestAccount(testKeyDir, GetAccount1PKFile()); err != nil {
		panic(err)
	}
	if err := ImportTestAccount(testKeyDir, GetAccount2PKFile()); err != nil {
		panic(err)
	}
	_ = InitKeystore(C.CString(testKeyDir))

	waitForNodeStart := make(chan struct{}, 1)
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		t.Log(jsonEvent)
		var envelope signal.Envelope
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == signal.EventNodeCrashed {
			signal.TriggerDefaultNodeNotificationHandler(jsonEvent)
			return
		}

		if envelope.Type == signal.EventSignRequestAdded {
		}
		if envelope.Type == signal.EventNodeStarted {
			t.Log("Node started, but we wait till it be ready")
		}
		if envelope.Type == signal.EventNodeReady {
			// sync
			if syncRequired {
				t.Logf("Sync is required")
				EnsureNodeSync(statusBackend.StatusNode().EnsureSync)
			} else {
				time.Sleep(5 * time.Second)
			}

			// now we can proceed with tests
			waitForNodeStart <- struct{}{}
		}
	})

	go func() {
		response := StartNode(C.CString(nodeConfigJSON))
		responseErr := APIResponse{}

		if err := json.Unmarshal([]byte(C.GoString(response)), &responseErr); err != nil {
			panic(err)
		}
		if responseErr.Error != "" {
			panic("cannot start node: " + responseErr.Error)
		}
	}()

	return waitForNodeStart
}

//nolint: deadcode
func testValidateNodeConfig(t *testing.T, config string, fn func(*testing.T, APIDetailedResponse)) {
	result := ValidateNodeConfig(C.CString(config))

	var resp APIDetailedResponse

	err := json.Unmarshal([]byte(C.GoString(result)), &resp)
	require.NoError(t, err)

	fn(t, resp)
}
