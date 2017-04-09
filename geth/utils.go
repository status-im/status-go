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
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/static"
)

var (
	muPrepareTestNode sync.Mutex
	RootDir           string
	TestDataDir       string
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

type NodeNotificationHandler func(jsonEvent string)

var notificationHandler NodeNotificationHandler = TriggerDefaultNodeNotificationHandler

// SetDefaultNodeNotificationHandler sets notification handler to invoke on SendSignal
func SetDefaultNodeNotificationHandler(fn NodeNotificationHandler) {
	notificationHandler = fn
}

// TriggerDefaultNodeNotificationHandler triggers default notification handler (helpful in tests)
func TriggerDefaultNodeNotificationHandler(jsonEvent string) {
	glog.V(logger.Info).Infof("notification received (default notification handler): %s\n", jsonEvent)
}

// SendSignal sends application signal (JSON, normally) upwards to application (via default notification handler)
func SendSignal(signal SignalEnvelope) {
	data, _ := json.Marshal(&signal)
	C.StatusServiceSignalEvent(C.CString(string(data)))
}

//export NotifyNode
func NotifyNode(jsonEvent *C.char) {
	notificationHandler(C.GoString(jsonEvent))
}

//export TriggerTestSignal
func TriggerTestSignal() {
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

func CopyFile(dst, src string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()

	if _, err := io.Copy(d, s); err != nil {
		return err
	}

	return nil
}

// LoadFromFile is usefull for loading test data, from testdata/filename into a variable
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
	if _, err := os.Stat(TestDataDir); os.IsNotExist(err) {
		syncRequired = true
	}

	// prepare node directory
	if err := os.MkdirAll(filepath.Join(TestDataDir, "keystore"), os.ModePerm); err != nil {
		glog.V(logger.Warn).Infoln("make node failed:", err)
		return err
	}

	// import test accounts (with test ether on it)
	if err := ImportTestAccount(filepath.Join(TestDataDir, "keystore"), "test-account1.pk"); err != nil {
		panic(err)
	}
	if err := ImportTestAccount(filepath.Join(TestDataDir, "keystore"), "test-account2.pk"); err != nil {
		panic(err)
	}

	// start geth node and wait for it to initialize
	config, err := params.NewNodeConfig(TestDataDir, params.TestNetworkId)
	if err != nil {
		return err
	}
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
		glog.V(logger.Warn).Infof("Sync is required, it will take %d seconds", testConfig.Node.SyncSeconds)
		time.Sleep(testConfig.Node.SyncSeconds * time.Second) // LES syncs headers, so that we are up do date when it is done
	}

	return nil
}

func RemoveTestNode() {
	err := os.RemoveAll(TestDataDir)
	if err != nil {
		glog.V(logger.Warn).Infof("could not clean up temporary datadir")
	}
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

func FromAddress(accountAddress string) common.Address {
	from, err := ParseAccountString(accountAddress)
	if err != nil {
		return common.Address{}
	}

	return from.Address
}

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
		os.MkdirAll(keystoreDir, os.ModePerm)
	}

	dst := filepath.Join(keystoreDir, accountFile)
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		err = ioutil.WriteFile(dst, static.MustAsset("keys/"+accountFile), 0644)
		if err != nil {
			glog.V(logger.Warn).Infof("cannot copy test account PK: %v", err)
			return err
		}
	}

	return nil
}
