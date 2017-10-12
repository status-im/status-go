package main

import "C"
import (
	"encoding/json"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	gethparams "github.com/ethereum/go-ethereum/params"

	"fmt"

	"github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/signal"
	"github.com/status-im/status-go/geth/txqueue"
	"github.com/status-im/status-go/static"
	. "github.com/status-im/status-go/testing"
	"github.com/stretchr/testify/require"
)

const zeroHash = "0x0000000000000000000000000000000000000000000000000000000000000000"

var nodeConfigJSON = `{
	"NetworkId": ` + strconv.Itoa(params.RopstenNetworkID) + `,
	"DataDir": "` + TestDataDir + `",
	"HTTPPort": ` + strconv.Itoa(TestConfig.Node.HTTPPort) + `,
	"WSPort": ` + strconv.Itoa(TestConfig.Node.WSPort) + `,
	"LogLevel": "INFO"
}`

// nolint: deadcode
func testExportedAPI(t *testing.T, done chan struct{}) {
	<-startTestNode(t)

	// FIXME(tiabc): All of that is done because usage of cgo is not supported in tests.
	// Probably, there should be a cleaner way, for example, test cgo bindings in e2e tests
	// separately from other internal tests.
	tests := []struct {
		name string
		fn   func(t *testing.T) bool
	}{
		{
			"check default configuration",
			testGetDefaultConfig,
		},
		{
			"reset blockchain data",
			testResetChainData,
		},
		{
			"stop/resume node",
			testStopResumeNode,
		},
		{
			"call RPC on in-proc handler",
			testCallRPC,
		},
		{
			"create main and child accounts",
			testCreateChildAccount,
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
			"account logout",
			testAccountLogout,
		},
		{
			"complete single queued transaction",
			testCompleteTransaction,
		},
		{
			"test complete multiple queued transactions",
			testCompleteMultipleQueuedTransactions,
		},
		{
			"discard single queued transaction",
			testDiscardTransaction,
		},
		{
			"test discard multiple queued transactions",
			testDiscardMultipleQueuedTransactions,
		},
		{
			"test jail invalid initialization",
			testJailInitInvalid,
		},
		{
			"test jail invalid parse",
			testJailParseInvalid,
		},
		{
			"test jail initialization",
			testJailInit,
		},
		{
			"test jailed calls",
			testJailFunctionCall,
		},
	}

	for _, test := range tests {
		t.Logf("=== RUN   %s", test.name)
		if ok := test.fn(t); !ok {
			break
		}
	}

	done <- struct{}{}
}

func testVerifyAccountPassword(t *testing.T) bool {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "accounts")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir) // nolint: errcheck

	if err = common.ImportTestAccount(tmpDir, "test-account1.pk"); err != nil {
		t.Fatal(err)
	}
	if err = common.ImportTestAccount(tmpDir, "test-account2.pk"); err != nil {
		t.Fatal(err)
	}

	// rename account file (to see that file's internals reviewed, when locating account key)
	accountFilePathOriginal := filepath.Join(tmpDir, "test-account1.pk")
	accountFilePath := filepath.Join(tmpDir, "foo"+TestConfig.Account1.Address+"bar.pk")
	if err := os.Rename(accountFilePathOriginal, accountFilePath); err != nil {
		t.Fatal(err)
	}

	response := common.APIResponse{}
	rawResponse := VerifyAccountPassword(
		C.CString(tmpDir),
		C.CString(TestConfig.Account1.Address),
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

func testGetDefaultConfig(t *testing.T) bool {
	networks := []struct {
		chainID        int
		refChainConfig *gethparams.ChainConfig
	}{
		{params.MainNetworkID, gethparams.MainnetChainConfig},
		{params.RopstenNetworkID, gethparams.TestnetChainConfig},
		{params.RinkebyNetworkID, gethparams.RinkebyChainConfig},
	}
	for i := range networks {
		network := networks[i]

		t.Run(fmt.Sprintf("networkID=%d", network.chainID), func(t *testing.T) {
			var (
				nodeConfig  = params.NodeConfig{}
				rawResponse = GenerateConfig(C.CString("/tmp/data-folder"), C.int(network.chainID), 1)
			)
			if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &nodeConfig); err != nil {
				t.Errorf("cannot decode response (%s): %v", C.GoString(rawResponse), err)
			}

			genesis := new(core.Genesis)
			if err := json.Unmarshal([]byte(nodeConfig.LightEthConfig.Genesis), genesis); err != nil {
				t.Error(err)
			}

			require.Equal(t, network.refChainConfig, genesis.Config)
		})
	}
	return true
}

// @TODO(adam): quarantined this test until it uses a different directory.
func testResetChainData(t *testing.T) bool {
	t.Skip()

	resetChainDataResponse := common.APIResponse{}
	rawResponse := ResetChainData()

	if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &resetChainDataResponse); err != nil {
		t.Errorf("cannot decode ResetChainData response (%s): %v", C.GoString(rawResponse), err)
		return false
	}
	if resetChainDataResponse.Error != "" {
		t.Errorf("unexpected error: %s", resetChainDataResponse.Error)
		return false
	}

	time.Sleep(TestConfig.Node.SyncSeconds * time.Second) // allow to re-sync blockchain

	testCompleteTransaction(t)

	return true
}

func testStopResumeNode(t *testing.T) bool {
	// to make sure that we start with empty account (which might get populated during previous tests)
	if err := statusAPI.Logout(); err != nil {
		t.Fatal(err)
	}

	whisperService, err := statusAPI.NodeManager().WhisperService()
	if err != nil {
		t.Errorf("whisper service not running: %v", err)
	}

	// create an account
	address1, pubKey1, _, err := statusAPI.CreateAccount(TestConfig.Account1.Password)
	if err != nil {
		t.Errorf("could not create account: %v", err)
		return false
	}
	t.Logf("account created: {address: %s, key: %s}", address1, pubKey1)

	// make sure that identity is not (yet injected)
	if whisperService.HasKeyPair(pubKey1) {
		t.Error("identity already present in whisper")
	}

	// select account
	loginResponse := common.APIResponse{}
	rawResponse := Login(C.CString(address1), C.CString(TestConfig.Account1.Password))

	if err = json.Unmarshal([]byte(C.GoString(rawResponse)), &loginResponse); err != nil {
		t.Errorf("cannot decode RecoverAccount response (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if loginResponse.Error != "" {
		t.Errorf("could not select account: %v", err)
		return false
	}
	if !whisperService.HasKeyPair(pubKey1) {
		t.Errorf("identity not injected into whisper: %v", err)
	}

	// stop and resume node, then make sure that selected account is still selected
	// nolint: dupl
	stopNodeFn := func() bool {
		response := common.APIResponse{}
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
		response := common.APIResponse{}
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
	if !resumeNodeFn() {
		return false
	}

	time.Sleep(5 * time.Second) // allow to start (instead of using blocking version of start, of filter event)

	// now, verify that we still have account logged in
	whisperService, err = statusAPI.NodeManager().WhisperService()
	if err != nil {
		t.Errorf("whisper service not running: %v", err)
	}
	if !whisperService.HasKeyPair(pubKey1) {
		t.Errorf("identity evicted from whisper on node restart: %v", err)
	}

	// additionally, let's complete transaction (just to make sure that node lives through pause/resume w/o issues)
	testCompleteTransaction(t)

	return true
}

func testCallRPC(t *testing.T) bool {
	expected := `{"jsonrpc":"2.0","id":64,"result":"0x47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad"}`
	rawResponse := CallRPC(C.CString(`{"jsonrpc":"2.0","method":"web3_sha3","params":["0x68656c6c6f20776f726c64"],"id":64}`))
	received := C.GoString(rawResponse)
	if expected != received {
		t.Errorf("unexpected reponse: expected: %v, got: %v", expected, received)
		return false
	}

	return true
}

func testCreateChildAccount(t *testing.T) bool {
	// to make sure that we start with empty account (which might get populated during previous tests)
	if err := statusAPI.Logout(); err != nil {
		t.Fatal(err)
	}

	keyStore, err := statusAPI.NodeManager().AccountKeyStore()
	if err != nil {
		t.Error(err)
		return false
	}

	// create an account
	createAccountResponse := common.AccountInfo{}
	rawResponse := CreateAccount(C.CString(TestConfig.Account1.Password))

	if err = json.Unmarshal([]byte(C.GoString(rawResponse)), &createAccountResponse); err != nil {
		t.Errorf("cannot decode CreateAccount response (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if createAccountResponse.Error != "" {
		t.Errorf("could not create account: %s", err)
		return false
	}
	address, pubKey, mnemonic := createAccountResponse.Address, createAccountResponse.PubKey, createAccountResponse.Mnemonic
	t.Logf("Account created: {address: %s, key: %s, mnemonic:%s}", address, pubKey, mnemonic)

	acct, err := common.ParseAccountString(address)
	if err != nil {
		t.Errorf("can not get account from address: %v", err)
		return false
	}

	// obtain decrypted key, and make sure that extended key (which will be used as root for sub-accounts) is present
	_, key, err := keyStore.AccountDecryptedKey(acct, TestConfig.Account1.Password)
	if err != nil {
		t.Errorf("can not obtain decrypted account key: %v", err)
		return false
	}

	if key.ExtendedKey == nil {
		t.Error("CKD#2 has not been generated for new account")
		return false
	}

	// try creating sub-account, w/o selecting main account i.e. w/o login to main account
	createSubAccountResponse := common.AccountInfo{}
	rawResponse = CreateChildAccount(C.CString(""), C.CString(TestConfig.Account1.Password))

	if err = json.Unmarshal([]byte(C.GoString(rawResponse)), &createSubAccountResponse); err != nil {
		t.Errorf("cannot decode CreateChildAccount response (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if createSubAccountResponse.Error != account.ErrNoAccountSelected.Error() {
		t.Errorf("expected error is not returned (tried to create sub-account w/o login): %v", createSubAccountResponse.Error)
		return false
	}

	err = statusAPI.SelectAccount(address, TestConfig.Account1.Password)
	if err != nil {
		t.Errorf("Test failed: could not select account: %v", err)
		return false
	}

	// try to create sub-account with wrong password
	createSubAccountResponse = common.AccountInfo{}
	rawResponse = CreateChildAccount(C.CString(""), C.CString("wrong password"))

	if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &createSubAccountResponse); err != nil {
		t.Errorf("cannot decode CreateChildAccount response (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if createSubAccountResponse.Error != "cannot retrieve a valid key for a given account: could not decrypt key with given passphrase" {
		t.Errorf("expected error is not returned (tried to create sub-account with wrong password): %v", createSubAccountResponse.Error)
		return false
	}

	// create sub-account (from implicit parent)
	createSubAccountResponse1 := common.AccountInfo{}
	rawResponse = CreateChildAccount(C.CString(""), C.CString(TestConfig.Account1.Password))

	if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &createSubAccountResponse1); err != nil {
		t.Errorf("cannot decode CreateChildAccount response (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if createSubAccountResponse1.Error != "" {
		t.Errorf("cannot create sub-account: %v", createSubAccountResponse1.Error)
		return false
	}

	// make sure that sub-account index automatically progresses
	createSubAccountResponse2 := common.AccountInfo{}
	rawResponse = CreateChildAccount(C.CString(""), C.CString(TestConfig.Account1.Password))

	if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &createSubAccountResponse2); err != nil {
		t.Errorf("cannot decode CreateChildAccount response (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if createSubAccountResponse2.Error != "" {
		t.Errorf("cannot create sub-account: %v", createSubAccountResponse2.Error)
	}

	if createSubAccountResponse1.Address == createSubAccountResponse2.Address || createSubAccountResponse1.PubKey == createSubAccountResponse2.PubKey {
		t.Error("sub-account index auto-increament failed")
		return false
	}

	// create sub-account (from explicit parent)
	createSubAccountResponse3 := common.AccountInfo{}
	rawResponse = CreateChildAccount(C.CString(createSubAccountResponse2.Address), C.CString(TestConfig.Account1.Password))

	if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &createSubAccountResponse3); err != nil {
		t.Errorf("cannot decode CreateChildAccount response (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if createSubAccountResponse3.Error != "" {
		t.Errorf("cannot create sub-account: %v", createSubAccountResponse3.Error)
	}

	subAccount1, subAccount2, subAccount3 := createSubAccountResponse1.Address, createSubAccountResponse2.Address, createSubAccountResponse3.Address
	subPubKey1, subPubKey2, subPubKey3 := createSubAccountResponse1.PubKey, createSubAccountResponse2.PubKey, createSubAccountResponse3.PubKey

	if subAccount1 == subAccount3 || subPubKey1 == subPubKey3 || subAccount2 == subAccount3 || subPubKey2 == subPubKey3 {
		t.Error("sub-account index auto-increament failed")
		return false
	}

	return true
}

func testRecoverAccount(t *testing.T) bool {
	keyStore, _ := statusAPI.NodeManager().AccountKeyStore()

	// create an account
	address, pubKey, mnemonic, err := statusAPI.CreateAccount(TestConfig.Account1.Password)
	if err != nil {
		t.Errorf("could not create account: %v", err)
		return false
	}
	t.Logf("Account created: {address: %s, key: %s, mnemonic:%s}", address, pubKey, mnemonic)

	// try recovering using password + mnemonic
	recoverAccountResponse := common.AccountInfo{}
	rawResponse := RecoverAccount(C.CString(TestConfig.Account1.Password), C.CString(mnemonic))

	if err = json.Unmarshal([]byte(C.GoString(rawResponse)), &recoverAccountResponse); err != nil {
		t.Errorf("cannot decode RecoverAccount response (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if recoverAccountResponse.Error != "" {
		t.Errorf("recover account failed: %v", recoverAccountResponse.Error)
		return false
	}
	addressCheck, pubKeyCheck := recoverAccountResponse.Address, recoverAccountResponse.PubKey
	if address != addressCheck || pubKey != pubKeyCheck {
		t.Error("recover account details failed to pull the correct details")
	}

	// now test recovering, but make sure that account/key file is removed i.e. simulate recovering on a new device
	account, err := common.ParseAccountString(address)
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

	recoverAccountResponse = common.AccountInfo{}
	rawResponse = RecoverAccount(C.CString(TestConfig.Account1.Password), C.CString(mnemonic))

	if err = json.Unmarshal([]byte(C.GoString(rawResponse)), &recoverAccountResponse); err != nil {
		t.Errorf("cannot decode RecoverAccount response (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if recoverAccountResponse.Error != "" {
		t.Errorf("recover account failed (for non-cached account): %v", recoverAccountResponse.Error)
		return false
	}
	addressCheck, pubKeyCheck = recoverAccountResponse.Address, recoverAccountResponse.PubKey
	if address != addressCheck || pubKey != pubKeyCheck {
		t.Error("recover account details failed to pull the correct details (for non-cached account)")
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
	recoverAccountResponse = common.AccountInfo{}
	rawResponse = RecoverAccount(C.CString(TestConfig.Account1.Password), C.CString(mnemonic))

	if err = json.Unmarshal([]byte(C.GoString(rawResponse)), &recoverAccountResponse); err != nil {
		t.Errorf("cannot decode RecoverAccount response (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if recoverAccountResponse.Error != "" {
		t.Errorf("recover account failed (for non-cached account): %v", recoverAccountResponse.Error)
		return false
	}
	addressCheck, pubKeyCheck = recoverAccountResponse.Address, recoverAccountResponse.PubKey
	if address != addressCheck || pubKey != pubKeyCheck {
		t.Error("recover account details failed to pull the correct details (for non-cached account)")
	}

	// time to login with recovered data
	whisperService, err := statusAPI.NodeManager().WhisperService()
	if err != nil {
		t.Errorf("whisper service not running: %v", err)
	}

	// make sure that identity is not (yet injected)
	if whisperService.HasKeyPair(pubKeyCheck) {
		t.Error("identity already present in whisper")
	}
	err = statusAPI.SelectAccount(addressCheck, TestConfig.Account1.Password)
	if err != nil {
		t.Errorf("Test failed: could not select account: %v", err)
		return false
	}
	if !whisperService.HasKeyPair(pubKeyCheck) {
		t.Errorf("identity not injected into whisper: %v", err)
	}

	return true
}

func testAccountSelect(t *testing.T) bool {
	// test to see if the account was injected in whisper
	whisperService, err := statusAPI.NodeManager().WhisperService()
	if err != nil {
		t.Errorf("whisper service not running: %v", err)
	}

	// create an account
	address1, pubKey1, _, err := statusAPI.CreateAccount(TestConfig.Account1.Password)
	if err != nil {
		t.Errorf("could not create account: %v", err)
		return false
	}
	t.Logf("Account created: {address: %s, key: %s}", address1, pubKey1)

	address2, pubKey2, _, err := statusAPI.CreateAccount(TestConfig.Account1.Password)
	if err != nil {
		t.Error("Test failed: could not create account")
		return false
	}
	t.Logf("Account created: {address: %s, key: %s}", address2, pubKey2)

	// make sure that identity is not (yet injected)
	if whisperService.HasKeyPair(pubKey1) {
		t.Error("identity already present in whisper")
	}

	// try selecting with wrong password
	loginResponse := common.APIResponse{}
	rawResponse := Login(C.CString(address1), C.CString("wrongPassword"))

	if err = json.Unmarshal([]byte(C.GoString(rawResponse)), &loginResponse); err != nil {
		t.Errorf("cannot decode RecoverAccount response (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if loginResponse.Error == "" {
		t.Error("select account is expected to throw error: wrong password used")
		return false
	}

	loginResponse = common.APIResponse{}
	rawResponse = Login(C.CString(address1), C.CString(TestConfig.Account1.Password))

	if err = json.Unmarshal([]byte(C.GoString(rawResponse)), &loginResponse); err != nil {
		t.Errorf("cannot decode RecoverAccount response (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if loginResponse.Error != "" {
		t.Errorf("Test failed: could not select account: %v", err)
		return false
	}
	if !whisperService.HasKeyPair(pubKey1) {
		t.Errorf("identity not injected into whisper: %v", err)
	}

	// select another account, make sure that previous account is wiped out from Whisper cache
	if whisperService.HasKeyPair(pubKey2) {
		t.Error("identity already present in whisper")
	}

	loginResponse = common.APIResponse{}
	rawResponse = Login(C.CString(address2), C.CString(TestConfig.Account1.Password))

	if err = json.Unmarshal([]byte(C.GoString(rawResponse)), &loginResponse); err != nil {
		t.Errorf("cannot decode RecoverAccount response (%s): %v", C.GoString(rawResponse), err)
		return false
	}

	if loginResponse.Error != "" {
		t.Errorf("Test failed: could not select account: %v", loginResponse.Error)
		return false
	}
	if !whisperService.HasKeyPair(pubKey2) {
		t.Errorf("identity not injected into whisper: %v", err)
	}
	if whisperService.HasKeyPair(pubKey1) {
		t.Error("identity should be removed, but it is still present in whisper")
	}

	return true
}

func testAccountLogout(t *testing.T) bool {
	whisperService, err := statusAPI.NodeManager().WhisperService()
	if err != nil {
		t.Errorf("whisper service not running: %v", err)
		return false
	}

	// create an account
	address, pubKey, _, err := statusAPI.CreateAccount(TestConfig.Account1.Password)
	if err != nil {
		t.Errorf("could not create account: %v", err)
		return false
	}

	// make sure that identity doesn't exist (yet) in Whisper
	if whisperService.HasKeyPair(pubKey) {
		t.Error("identity already present in whisper")
		return false
	}

	// select/login
	err = statusAPI.SelectAccount(address, TestConfig.Account1.Password)
	if err != nil {
		t.Errorf("Test failed: could not select account: %v", err)
		return false
	}
	if !whisperService.HasKeyPair(pubKey) {
		t.Error("identity not injected into whisper")
		return false
	}

	logoutResponse := common.APIResponse{}
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
	if whisperService.HasKeyPair(pubKey) {
		t.Error("identity not cleared from whisper")
		return false
	}

	return true
}

func testCompleteTransaction(t *testing.T) bool {
	txQueueManager := statusAPI.TxQueueManager()
	txQueue := txQueueManager.TransactionQueue()

	txQueue.Reset()

	time.Sleep(5 * time.Second) // allow to sync

	// log into account from which transactions will be sent
	if err := statusAPI.SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password); err != nil {
		t.Errorf("cannot select account: %v", TestConfig.Account1.Address)
		return false
	}

	// make sure you panic if transaction complete doesn't return
	queuedTxCompleted := make(chan struct{}, 1)
	abortPanic := make(chan struct{}, 1)
	common.PanicAfter(10*time.Second, abortPanic, "testCompleteTransaction")

	// replace transaction notification handler
	var txHash = ""
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope signal.Envelope
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == txqueue.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			t.Logf("transaction queued (will be completed shortly): {id: %s}\n", event["id"].(string))

			completeTxResponse := common.CompleteTransactionResult{}
			rawResponse := CompleteTransaction(C.CString(event["id"].(string)), C.CString(TestConfig.Account1.Password))

			if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &completeTxResponse); err != nil {
				t.Errorf("cannot decode RecoverAccount response (%s): %v", C.GoString(rawResponse), err)
			}

			if completeTxResponse.Error != "" {
				t.Errorf("cannot complete queued transaction[%v]: %v", event["id"], completeTxResponse.Error)
			}

			txHash = completeTxResponse.Hash

			t.Logf("transaction complete: https://testnet.etherscan.io/tx/%s", txHash)
			abortPanic <- struct{}{} // so that timeout is aborted
			queuedTxCompleted <- struct{}{}
		}
	})

	// this call blocks, up until Complete Transaction is called
	txCheckHash, err := statusAPI.SendTransaction(nil, common.SendTxArgs{
		From:  common.FromAddress(TestConfig.Account1.Address),
		To:    common.ToAddress(TestConfig.Account2.Address),
		Value: (*hexutil.Big)(big.NewInt(1000000000000)),
	})
	if err != nil {
		t.Errorf("Failed to SendTransaction: %s", err)
		return false
	}

	<-queuedTxCompleted // make sure that complete transaction handler completes its magic, before we proceed

	if txHash != txCheckHash.Hex() {
		t.Errorf("Transaction hash returned from SendTransaction is invalid: expected %s, got %s",
			txCheckHash.Hex(), txHash)
		return false
	}

	if reflect.DeepEqual(txCheckHash, gethcommon.Hash{}) {
		t.Error("Test failed: transaction was never queued or completed")
		return false
	}

	if txQueue.Count() != 0 {
		t.Error("tx queue must be empty at this point")
		return false
	}

	return true
}

func testCompleteMultipleQueuedTransactions(t *testing.T) bool {
	txQueue := statusAPI.TxQueueManager().TransactionQueue()
	txQueue.Reset()

	// log into account from which transactions will be sent
	if err := statusAPI.SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password); err != nil {
		t.Errorf("cannot select account: %v", TestConfig.Account1.Address)
		return false
	}

	// make sure you panic if transaction complete doesn't return
	testTxCount := 3
	txIDs := make(chan string, testTxCount)
	allTestTxCompleted := make(chan struct{}, 1)

	// replace transaction notification handler
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var txID string
		var envelope signal.Envelope
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == txqueue.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			txID = event["id"].(string)
			t.Logf("transaction queued (will be completed in a single call, once aggregated): {id: %s}\n", txID)

			txIDs <- txID
		}
	})

	//  this call blocks, and should return when DiscardQueuedTransaction() for a given tx id is called
	sendTx := func() {
		txHashCheck, err := statusAPI.SendTransaction(nil, common.SendTxArgs{
			From:  common.FromAddress(TestConfig.Account1.Address),
			To:    common.ToAddress(TestConfig.Account2.Address),
			Value: (*hexutil.Big)(big.NewInt(1000000000000)),
		})
		if err != nil {
			t.Errorf("unexpected error thrown: %v", err)
			return
		}

		if reflect.DeepEqual(txHashCheck, gethcommon.Hash{}) {
			t.Error("transaction returned empty hash")
			return
		}
	}

	// wait for transactions, and complete them in a single call
	completeTxs := func(txIDStrings string) {
		var parsedIDs []string
		if err := json.Unmarshal([]byte(txIDStrings), &parsedIDs); err != nil {
			t.Error(err)
			return
		}

		parsedIDs = append(parsedIDs, "invalid-tx-id")
		updatedTxIDStrings, _ := json.Marshal(parsedIDs)

		// complete
		resultsString := CompleteTransactions(C.CString(string(updatedTxIDStrings)), C.CString(TestConfig.Account1.Password))
		resultsStruct := common.CompleteTransactionsResult{}
		if err := json.Unmarshal([]byte(C.GoString(resultsString)), &resultsStruct); err != nil {
			t.Error(err)
			return
		}
		results := resultsStruct.Results

		if len(results) != (testTxCount+1) || results["invalid-tx-id"].Error != txqueue.ErrQueuedTxIDNotFound.Error() {
			t.Errorf("cannot complete txs: %v", results)
			return
		}
		for txID, txResult := range results {
			if txID != txResult.ID {
				t.Errorf("tx id not set in result: expected id is %s", txID)
				return
			}
			if txResult.Error != "" && txID != "invalid-tx-id" {
				t.Errorf("invalid error for %s", txID)
				return
			}
			if txResult.Hash == zeroHash && txID != "invalid-tx-id" {
				t.Errorf("invalid hash (expected non empty hash): %s", txID)
				return
			}

			if txResult.Hash != zeroHash {
				t.Logf("transaction complete: https://testnet.etherscan.io/tx/%s", txResult.Hash)
			}
		}

		time.Sleep(1 * time.Second) // make sure that tx complete signal propagates
		for _, txID := range parsedIDs {
			if txQueue.Has(common.QueuedTxID(txID)) {
				t.Errorf("txqueue should not have test tx at this point (it should be completed): %s", txID)
				return
			}
		}
	}
	go func() {
		var txIDStrings []string
		for i := 0; i < testTxCount; i++ {
			txIDStrings = append(txIDStrings, <-txIDs)
		}

		txIDJSON, _ := json.Marshal(txIDStrings)
		completeTxs(string(txIDJSON))
		allTestTxCompleted <- struct{}{}
	}()

	// send multiple transactions
	for i := 0; i < testTxCount; i++ {
		go sendTx()
	}

	select {
	case <-allTestTxCompleted:
		// pass
	case <-time.After(20 * time.Second):
		t.Error("test timed out")
		return false
	}

	if txQueue.Count() != 0 {
		t.Error("tx queue must be empty at this point")
		return false
	}

	return true
}

func testDiscardTransaction(t *testing.T) bool {
	txQueue := statusAPI.TxQueueManager().TransactionQueue()
	txQueue.Reset()

	// log into account from which transactions will be sent
	if err := statusAPI.SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password); err != nil {
		t.Errorf("cannot select account: %v", TestConfig.Account1.Address)
		return false
	}

	// make sure you panic if transaction complete doesn't return
	completeQueuedTransaction := make(chan struct{}, 1)
	common.PanicAfter(20*time.Second, completeQueuedTransaction, "testDiscardTransaction")

	// replace transaction notification handler
	var txID string
	txFailedEventCalled := false
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope signal.Envelope
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == txqueue.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			txID = event["id"].(string)
			t.Logf("transaction queued (will be discarded soon): {id: %s}\n", txID)

			if !txQueue.Has(common.QueuedTxID(txID)) {
				t.Errorf("txqueue should still have test tx: %s", txID)
				return
			}

			// discard
			discardResponse := common.DiscardTransactionResult{}
			rawResponse := DiscardTransaction(C.CString(txID))

			if err := json.Unmarshal([]byte(C.GoString(rawResponse)), &discardResponse); err != nil {
				t.Errorf("cannot decode RecoverAccount response (%s): %v", C.GoString(rawResponse), err)
			}

			if discardResponse.Error != "" {
				t.Errorf("cannot discard tx: %v", discardResponse.Error)
				return
			}

			// try completing discarded transaction
			_, err := statusAPI.CompleteTransaction(common.QueuedTxID(txID), TestConfig.Account1.Password)
			if err != txqueue.ErrQueuedTxIDNotFound {
				t.Error("expects tx not found, but call to CompleteTransaction succeeded")
				return
			}

			time.Sleep(1 * time.Second) // make sure that tx complete signal propagates
			if txQueue.Has(common.QueuedTxID(txID)) {
				t.Errorf("txqueue should not have test tx at this point (it should be discarded): %s", txID)
				return
			}

			completeQueuedTransaction <- struct{}{} // so that timeout is aborted
		}

		if envelope.Type == txqueue.EventTransactionFailed {
			event := envelope.Event.(map[string]interface{})
			t.Logf("transaction return event received: {id: %s}\n", event["id"].(string))

			receivedErrMessage := event["error_message"].(string)
			expectedErrMessage := txqueue.ErrQueuedTxDiscarded.Error()
			if receivedErrMessage != expectedErrMessage {
				t.Errorf("unexpected error message received: got %v", receivedErrMessage)
				return
			}

			receivedErrCode := event["error_code"].(string)
			if receivedErrCode != txqueue.SendTransactionDiscardedErrorCode {
				t.Errorf("unexpected error code received: got %v", receivedErrCode)
				return
			}

			txFailedEventCalled = true
		}
	})

	// this call blocks, and should return when DiscardQueuedTransaction() is called
	txHashCheck, err := statusAPI.SendTransaction(nil, common.SendTxArgs{
		From:  common.FromAddress(TestConfig.Account1.Address),
		To:    common.ToAddress(TestConfig.Account2.Address),
		Value: (*hexutil.Big)(big.NewInt(1000000000000)),
	})
	if err != txqueue.ErrQueuedTxDiscarded {
		t.Errorf("expected error not thrown: %v", err)
		return false
	}

	if !reflect.DeepEqual(txHashCheck, gethcommon.Hash{}) {
		t.Error("transaction returned hash, while it shouldn't")
		return false
	}

	if txQueue.Count() != 0 {
		t.Error("tx queue must be empty at this point")
		return false
	}

	if !txFailedEventCalled {
		t.Error("expected tx failure signal is not received")
		return false
	}

	return true
}

func testDiscardMultipleQueuedTransactions(t *testing.T) bool {
	txQueue := statusAPI.TxQueueManager().TransactionQueue()
	txQueue.Reset()

	// log into account from which transactions will be sent
	if err := statusAPI.SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password); err != nil {
		t.Errorf("cannot select account: %v", TestConfig.Account1.Address)
		return false
	}

	// make sure you panic if transaction complete doesn't return
	testTxCount := 3
	txIDs := make(chan string, testTxCount)
	allTestTxDiscarded := make(chan struct{}, 1)

	// replace transaction notification handler
	txFailedEventCallCount := 0
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var txID string
		var envelope signal.Envelope
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == txqueue.EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})
			txID = event["id"].(string)
			t.Logf("transaction queued (will be discarded soon): {id: %s}\n", txID)

			if !txQueue.Has(common.QueuedTxID(txID)) {
				t.Errorf("txqueue should still have test tx: %s", txID)
				return
			}

			txIDs <- txID
		}

		if envelope.Type == txqueue.EventTransactionFailed {
			event := envelope.Event.(map[string]interface{})
			t.Logf("transaction return event received: {id: %s}\n", event["id"].(string))

			receivedErrMessage := event["error_message"].(string)
			expectedErrMessage := txqueue.ErrQueuedTxDiscarded.Error()
			if receivedErrMessage != expectedErrMessage {
				t.Errorf("unexpected error message received: got %v", receivedErrMessage)
				return
			}

			receivedErrCode := event["error_code"].(string)
			if receivedErrCode != txqueue.SendTransactionDiscardedErrorCode {
				t.Errorf("unexpected error code received: got %v", receivedErrCode)
				return
			}

			txFailedEventCallCount++
			if txFailedEventCallCount == testTxCount {
				allTestTxDiscarded <- struct{}{}
			}
		}
	})

	// this call blocks, and should return when DiscardQueuedTransaction() for a given tx id is called
	sendTx := func() {
		txHashCheck, err := statusAPI.SendTransaction(nil, common.SendTxArgs{
			From:  common.FromAddress(TestConfig.Account1.Address),
			To:    common.ToAddress(TestConfig.Account2.Address),
			Value: (*hexutil.Big)(big.NewInt(1000000000000)),
		})
		if err != txqueue.ErrQueuedTxDiscarded {
			t.Errorf("expected error not thrown: %v", err)
			return
		}

		if !reflect.DeepEqual(txHashCheck, gethcommon.Hash{}) {
			t.Error("transaction returned hash, while it shouldn't")
			return
		}
	}

	// wait for transactions, and discard immediately
	discardTxs := func(txIDStrings string) {
		var parsedIDs []string
		if err := json.Unmarshal([]byte(txIDStrings), &parsedIDs); err != nil {
			t.Error(err)
			return
		}

		parsedIDs = append(parsedIDs, "invalid-tx-id")
		updatedTxIDStrings, _ := json.Marshal(parsedIDs)

		// discard
		discardResultsString := DiscardTransactions(C.CString(string(updatedTxIDStrings)))
		discardResultsStruct := common.DiscardTransactionsResult{}
		if err := json.Unmarshal([]byte(C.GoString(discardResultsString)), &discardResultsStruct); err != nil {
			t.Error(err)
			return
		}
		discardResults := discardResultsStruct.Results

		if len(discardResults) != 1 || discardResults["invalid-tx-id"].Error != txqueue.ErrQueuedTxIDNotFound.Error() {
			t.Errorf("cannot discard txs: %v", discardResults)
			return
		}

		// try completing discarded transaction
		completeResultsString := CompleteTransactions(C.CString(string(updatedTxIDStrings)), C.CString(TestConfig.Account1.Password))
		completeResultsStruct := common.CompleteTransactionsResult{}
		if err := json.Unmarshal([]byte(C.GoString(completeResultsString)), &completeResultsStruct); err != nil {
			t.Error(err)
			return
		}
		completeResults := completeResultsStruct.Results

		if len(completeResults) != (testTxCount + 1) {
			t.Error("unexpected number of errors (call to CompleteTransaction should not succeed)")
		}
		for txID, txResult := range completeResults {
			if txID != txResult.ID {
				t.Errorf("tx id not set in result: expected id is %s", txID)
				return
			}
			if txResult.Error != txqueue.ErrQueuedTxIDNotFound.Error() {
				t.Errorf("invalid error for %s", txResult.Hash)
				return
			}
			if txResult.Hash != zeroHash {
				t.Errorf("invalid hash (expected zero): %s", txResult.Hash)
				return
			}
		}

		time.Sleep(1 * time.Second) // make sure that tx complete signal propagates
		for _, txID := range parsedIDs {
			if txQueue.Has(common.QueuedTxID(txID)) {
				t.Errorf("txqueue should not have test tx at this point (it should be discarded): %s", txID)
				return
			}
		}
	}
	go func() {
		var txIDStrings []string
		for i := 0; i < testTxCount; i++ {
			txIDStrings = append(txIDStrings, <-txIDs)
		}

		txIDJSON, _ := json.Marshal(txIDStrings)
		discardTxs(string(txIDJSON))
	}()

	// send multiple transactions
	for i := 0; i < testTxCount; i++ {
		go sendTx()
	}

	select {
	case <-allTestTxDiscarded:
		// pass
	case <-time.After(20 * time.Second):
		t.Error("test timed out")
		return false
	}

	if txQueue.Count() != 0 {
		t.Error("tx queue must be empty at this point")
		return false
	}

	return true
}

func testJailInitInvalid(t *testing.T) bool {
	// Arrange.
	initInvalidCode := `
	var _status_catalog = {
		foo: 'bar'
	`

	// Act.
	InitJail(C.CString(initInvalidCode))
	response := C.GoString(Parse(C.CString("CHAT_ID_INIT_TEST"), C.CString(``)))

	// Assert.
	expectedResponse := `{"error":"(anonymous): Line 4:3 Unexpected end of input (and 3 more errors)"}`
	if expectedResponse != response {
		t.Errorf("unexpected response, expected: %v, got: %v", expectedResponse, response)
		return false
	}
	return true
}

func testJailParseInvalid(t *testing.T) bool {
	// Arrange.
	initCode := `
	var _status_catalog = {
		foo: 'bar'
	};
	`

	// Act.
	InitJail(C.CString(initCode))
	extraInvalidCode := `
	var extraFunc = function (x) {
	  return x * x;
	`
	response := C.GoString(Parse(C.CString("CHAT_ID_INIT_TEST"), C.CString(extraInvalidCode)))

	// Assert.
	expectedResponse := `{"error":"(anonymous): Line 16331:50 Unexpected end of input (and 1 more errors)"}`
	if expectedResponse != response {
		t.Errorf("unexpected response, expected: %v, got: %v", expectedResponse, response)
		return false
	}
	return true
}

func testJailInit(t *testing.T) bool {
	initCode := `
	var _status_catalog = {
		foo: 'bar'
	};
	`
	InitJail(C.CString(initCode))

	extraCode := `
	var extraFunc = function (x) {
	  return x * x;
	};
	`
	rawResponse := Parse(C.CString("CHAT_ID_INIT_TEST"), C.CString(extraCode))
	parsedResponse := C.GoString(rawResponse)

	expectedResponse := `{"result": {"foo":"bar"}}`

	if !reflect.DeepEqual(expectedResponse, parsedResponse) {
		t.Error("expected output not returned from jail.Parse()")
		return false
	}

	t.Logf("jail inited and parsed: %s", parsedResponse)

	return true
}

func testJailFunctionCall(t *testing.T) bool {
	InitJail(C.CString(""))

	// load Status JS and add test command to it
	statusJS := string(static.MustAsset("testdata/jail/status.js")) + `;
	_status_catalog.commands["testCommand"] = function (params) {
		return params.val * params.val;
	};`
	Parse(C.CString("CHAT_ID_CALL_TEST"), C.CString(statusJS))

	// call with wrong chat id
	rawResponse := Call(C.CString("CHAT_IDNON_EXISTENT"), C.CString(""), C.CString(""))
	parsedResponse := C.GoString(rawResponse)
	expectedError := `{"error":"Cell[CHAT_IDNON_EXISTENT] doesn't exist."}`
	if parsedResponse != expectedError {
		t.Errorf("expected error is not returned: expected %s, got %s", expectedError, parsedResponse)
		return false
	}

	// call extraFunc()
	rawResponse = Call(C.CString("CHAT_ID_CALL_TEST"), C.CString(`["commands", "testCommand"]`), C.CString(`{"val": 12}`))
	parsedResponse = C.GoString(rawResponse)
	expectedResponse := `{"result": 144}`
	if parsedResponse != expectedResponse {
		t.Errorf("expected response is not returned: expected %s, got %s", expectedResponse, parsedResponse)
		return false
	}

	t.Logf("jailed method called: %s", parsedResponse)

	return true
}

func startTestNode(t *testing.T) <-chan struct{} {
	syncRequired := false
	if _, err := os.Stat(TestDataDir); os.IsNotExist(err) {
		syncRequired = true
	}

	// inject test accounts
	if err := common.ImportTestAccount(filepath.Join(TestDataDir, "keystore"), "test-account1.pk"); err != nil {
		panic(err)
	}
	if err := common.ImportTestAccount(filepath.Join(TestDataDir, "keystore"), "test-account2.pk"); err != nil {
		panic(err)
	}

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

		if envelope.Type == txqueue.EventTransactionQueued {
		}
		if envelope.Type == signal.EventNodeStarted {
			t.Log("Node started, but we wait till it be ready")
		}
		if envelope.Type == signal.EventNodeReady {
			// manually add static nodes (LES auto-discovery is not stable yet)
			PopulateStaticPeers()

			// sync
			if syncRequired {
				t.Logf("Sync is required, it will take %d seconds", TestConfig.Node.SyncSeconds)
				time.Sleep(TestConfig.Node.SyncSeconds * time.Second) // LES syncs headers, so that we are up do date when it is done
			} else {
				time.Sleep(5 * time.Second)
			}

			// now we can proceed with tests
			waitForNodeStart <- struct{}{}
		}
	})

	go func() {
		response := StartNode(C.CString(nodeConfigJSON))
		responseErr := common.APIResponse{}

		if err := json.Unmarshal([]byte(C.GoString(response)), &responseErr); err != nil {
			panic(err)
		}
		if responseErr.Error != "" {
			panic("cannot start node: " + responseErr.Error)
		}
	}()

	return waitForNodeStart
}

func testValidateNodeConfig(t *testing.T, config string, fn func(common.APIDetailedResponse)) {
	result := ValidateNodeConfig(C.CString(config))

	var resp common.APIDetailedResponse

	err := json.Unmarshal([]byte(C.GoString(result)), &resp)
	require.NoError(t, err)

	fn(resp)
}
