package geth

/*
#include <stddef.h>
#include <stdbool.h>
extern bool StatusServiceSignalEvent(const char *jsonEvent);
*/
import "C"
import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/static"
)

var (
	muPrepareTestNode sync.Mutex

	// RootDir is the main application directory
	RootDir string

	// TestDataDir is data directory used for tests
	TestDataDir string
)

func init() {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// setup root directory
	RootDir = filepath.Dir(pwd)
	if strings.HasSuffix(RootDir, "geth") || strings.HasSuffix(RootDir, "cmd") { // we need to hop one more level
		RootDir = filepath.Join(RootDir, "..")
	}

	// setup auxiliary directories
	TestDataDir = filepath.Join(RootDir, ".ethereumtest")
}

// NodeNotificationHandler defines a handler able to process incoming node events.
// Events are encoded as JSON strings.
type NodeNotificationHandler func(jsonEvent string)

var notificationHandler NodeNotificationHandler = TriggerDefaultNodeNotificationHandler

// SetDefaultNodeNotificationHandler sets notification handler to invoke on SendSignal
func SetDefaultNodeNotificationHandler(fn NodeNotificationHandler) {
	notificationHandler = fn
}

// TriggerDefaultNodeNotificationHandler triggers default notification handler (helpful in tests)
func TriggerDefaultNodeNotificationHandler(jsonEvent string) {
	log.Info("Notification received (default handler)", "event", jsonEvent)
}

// SendSignal sends application signal (JSON, normally) upwards to application (via default notification handler)
func SendSignal(signal SignalEnvelope) {
	data, _ := json.Marshal(&signal)
	C.StatusServiceSignalEvent(C.CString(string(data)))
}

//export NotifyNode
func NotifyNode(jsonEvent *C.char) { // nolint: golint
	notificationHandler(C.GoString(jsonEvent))
}

//export TriggerTestSignal
func TriggerTestSignal() { // nolint: golint
	C.StatusServiceSignalEvent(C.CString(`{"answer": 42}`))
}

// TestConfig contains shared (among different test packages) parameters
type TestConfig struct {
	Node struct {
		SyncSeconds time.Duration
		HTTPPort    int
		WSPort      int
	}
	Account1 struct {
		Address  string
		Password string
	}
	Account2 struct {
		Address  string
		Password string
	}
}

// LoadTestConfig loads test configuration values from disk
func LoadTestConfig() (*TestConfig, error) {
	var testConfig TestConfig

	configData := string(static.MustAsset("config/test-data.json"))
	if err := json.Unmarshal([]byte(configData), &testConfig); err != nil {
		return nil, err
	}

	return &testConfig, nil
}

// LoadFromFile is useful for loading test data, from testdata/filename into a variable
// nolint: errcheck
func LoadFromFile(filename string) string {
	f, err := os.Open(filename)
	if err != nil {
		return ""
	}

	buf := bytes.NewBuffer(nil)
	io.Copy(buf, f)
	f.Close()

	return string(buf.Bytes())
}

// PrepareTestNode initializes node manager and start a test node (only once!)
func PrepareTestNode() (err error) {
	muPrepareTestNode.Lock()
	defer muPrepareTestNode.Unlock()

	manager := NodeManagerInstance()
	if manager.NodeInited() {
		return nil
	}

	defer HaltOnPanic()

	testConfig, err := LoadTestConfig()
	if err != nil {
		return err
	}

	syncRequired := false
	if _, err = os.Stat(TestDataDir); os.IsNotExist(err) {
		syncRequired = true
	}

	// prepare node directory
	if err = os.MkdirAll(filepath.Join(TestDataDir, "keystore"), os.ModePerm); err != nil {
		log.Warn("make node failed", "error", err)
		return err
	}

	// import test accounts (with test ether on it)
	if err = ImportTestAccount(filepath.Join(TestDataDir, "keystore"), "test-account1.pk"); err != nil {
		panic(err)
	}
	if err = ImportTestAccount(filepath.Join(TestDataDir, "keystore"), "test-account2.pk"); err != nil {
		panic(err)
	}

	// start geth node and wait for it to initialize
	config, err := params.NewNodeConfig(filepath.Join(TestDataDir, "data"), params.RopstenNetworkID, true)
	if err != nil {
		return err
	}
	config.KeyStoreDir = filepath.Join(TestDataDir, "keystore")
	config.HTTPPort = testConfig.Node.HTTPPort // to avoid conflicts with running app, using different port in tests
	config.WSPort = testConfig.Node.WSPort     // ditto
	config.LogEnabled = true

	err = CreateAndRunNode(config)
	if err != nil {
		panic(err)
	}

	manager = NodeManagerInstance()
	if !manager.NodeInited() {
		panic(ErrInvalidGethNode)
	}
	if service, err := manager.RPCClient(); err != nil || service == nil {
		panic(ErrInvalidGethNode)
	}
	if service, err := manager.WhisperService(); err != nil || service == nil {
		panic(ErrInvalidGethNode)
	}
	if service, err := manager.LightEthereumService(); err != nil || service == nil {
		panic(ErrInvalidGethNode)
	}

	if syncRequired {
		log.Warn("Sync is required", "duration", testConfig.Node.SyncSeconds)
		time.Sleep(testConfig.Node.SyncSeconds * time.Second) // LES syncs headers, so that we are up do date when it is done
	}

	return nil
}

// MakeTestCompleteTxHandler returns node notification handler to be used in test
// basically notification handler completes a transaction (that is enqueued after
// the handler has been installed)
func MakeTestCompleteTxHandler(t *testing.T, txHash *common.Hash, completed chan struct{}) (handler func(jsonEvent string), err error) {
	testConfig, err := LoadTestConfig()
	if err != nil {
		return
	}

	handler = func(jsonEvent string) {
		var envelope SignalEnvelope
		if err := json.Unmarshal([]byte(jsonEvent), &envelope); err != nil {
			t.Errorf("cannot unmarshal event's JSON: %s", jsonEvent)
			return
		}
		if envelope.Type == EventTransactionQueued {
			event := envelope.Event.(map[string]interface{})

			t.Logf("Transaction queued (will be completed shortly): {id: %s}\n", event["id"].(string))

			if err := SelectAccount(testConfig.Account1.Address, testConfig.Account1.Password); err != nil {
				t.Errorf("cannot select account: %v", testConfig.Account1.Address)
				return
			}

			var err error
			if *txHash, err = CompleteTransaction(event["id"].(string), testConfig.Account1.Password); err != nil {
				t.Errorf("cannot complete queued transaction[%v]: %v", event["id"], err)
				return
			}

			t.Logf("Contract created: https://testnet.etherscan.io/tx/%s", txHash.Hex())
			close(completed) // so that timeout is aborted
		}
	}
	return
}

// PanicAfter throws panic() after waitSeconds, unless abort channel receives notification
func PanicAfter(waitSeconds time.Duration, abort chan struct{}, desc string) {
	go func() {
		select {
		case <-abort:
			return
		case <-time.After(waitSeconds):
			panic("whatever you were doing takes toooo long: " + desc)
		}
	}()
}

// FromAddress converts account address from string to common.Address.
// The function is useful to format "From" field of send transaction struct.
func FromAddress(accountAddress string) common.Address {
	from, err := ParseAccountString(accountAddress)
	if err != nil {
		return common.Address{}
	}

	return from.Address
}

// ToAddress converts account address from string to *common.Address.
// The function is useful to format "To" field of send transaction struct.
func ToAddress(accountAddress string) *common.Address {
	to, err := ParseAccountString(accountAddress)
	if err != nil {
		return nil
	}

	return &to.Address
}

// ParseAccountString parses hex encoded string and returns is as accounts.Account.
func ParseAccountString(account string) (accounts.Account, error) {
	// valid address, convert to account
	if common.IsHexAddress(account) {
		return accounts.Account{Address: common.HexToAddress(account)}, nil
	}

	return accounts.Account{}, ErrInvalidAccountAddressOrKey
}

// AddressToDecryptedAccount tries to load and decrypt account with a given password
func AddressToDecryptedAccount(address, password string) (accounts.Account, *keystore.Key, error) {
	nodeManager := NodeManagerInstance()
	keyStore, err := nodeManager.AccountKeyStore()
	if err != nil {
		return accounts.Account{}, nil, err
	}

	account, err := ParseAccountString(address)
	if err != nil {
		return accounts.Account{}, nil, ErrAddressToAccountMappingFailure
	}

	return keyStore.AccountDecryptedKey(account, password)
}

// ImportTestAccount checks if test account exists in keystore, and if not
// tries to import it (from static resources, see "static/keys" folder)
func ImportTestAccount(keystoreDir, accountFile string) error {
	// make sure that keystore folder exists
	if _, err := os.Stat(keystoreDir); os.IsNotExist(err) {
		os.MkdirAll(keystoreDir, os.ModePerm) // nolint: errcheck
	}

	dst := filepath.Join(keystoreDir, accountFile)
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		err = ioutil.WriteFile(dst, static.MustAsset("keys/"+accountFile), 0644)
		if err != nil {
			log.Warn("cannot copy test account PK", "error", err)
			return err
		}
	}

	return nil
}
