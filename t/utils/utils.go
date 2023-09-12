package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/stretchr/testify/suite" // required to register testify flags

	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/static"
	"github.com/status-im/status-go/t"
)

var (
	networkSelected = flag.String("network", "statuschain", "-network=NETWORKID or -network=NETWORKNAME to select network used for tests")
	logLevel        = flag.String("log", "INFO", `Log level, one of: "ERROR", "WARN", "INFO", "DEBUG", and "TRACE"`)
)

var (
	// ErrNoRemoteURL is returned when network id has no associated url.
	ErrNoRemoteURL = errors.New("network id requires a remote URL")

	// ErrTimeout is returned when test times out
	ErrTimeout = errors.New("timeout")

	// TestConfig defines the default config usable at package-level.
	TestConfig *testConfig

	// RootDir is the main application directory
	RootDir string

	// TestDataDir is data directory used for tests
	TestDataDir string

	// TestNetworkNames network ID to name mapping
	TestNetworkNames = map[int]string{
		params.MainNetworkID:        "Mainnet",
		params.StatusChainNetworkID: "StatusChain",
		params.GoerliNetworkID:      "Goerli",
	}

	syncTimeout = 50 * time.Minute
)

func Init() {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	flag.Parse()

	// set up logger
	loggerEnabled := *logLevel != ""
	if err := logutils.OverrideRootLog(loggerEnabled, *logLevel, logutils.FileOptions{}, true); err != nil {
		panic(err)
	}

	// setup root directory
	const pathSeparator = string(os.PathSeparator)
	RootDir = filepath.Dir(pwd)
	pathDirs := strings.Split(RootDir, pathSeparator)
	for i := len(pathDirs) - 1; i >= 0; i-- {
		if pathDirs[i] == "status-go" {
			RootDir = filepath.Join(pathDirs[:i+1]...)
			RootDir = filepath.Join(pathSeparator, RootDir)
			break
		}
	}

	// setup auxiliary directories
	TestDataDir = filepath.Join(RootDir, ".ethereumtest")

	TestConfig, err = loadTestConfig()
	if err != nil {
		panic(err)
	}
}

// LoadFromFile is useful for loading test data, from testdata/filename into a variable
// nolint: errcheck
func LoadFromFile(filename string) string {
	f, err := os.Open(filename)
	if err != nil {
		return ""
	}

	buf := bytes.NewBuffer(nil)
	io.Copy(buf, f) // nolint: gas
	f.Close()       // nolint: gas

	return buf.String()
}

// EnsureSync waits until blockchain synchronization is complete and returns.
type EnsureSync func(context.Context) error

// EnsureNodeSync waits until node synchronzation is done to continue
// with tests afterwards. Panics in case of an error or a timeout.
func EnsureNodeSync(ensureSync EnsureSync) {
	ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	defer cancel()

	if err := ensureSync(ctx); err != nil {
		panic(err)
	}
}

// GetRemoteURLFromNetworkID returns associated network url for giving network id.
func GetRemoteURLFromNetworkID(id int) (url string, err error) {
	switch id {
	case params.MainNetworkID:
		url = params.MainnetEthereumNetworkURL
	case params.GoerliNetworkID:
		url = params.GoerliEthereumNetworkURL
	default:
		err = ErrNoRemoteURL
	}

	return
}

// GetHeadHashFromNetworkID returns the hash associated with a given network id.
func GetHeadHashFromNetworkID(id int) string {
	switch id {
	case params.MainNetworkID:
		return "0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3"
	case params.StatusChainNetworkID:
		return "0xe9d8920a99dc66a9557a87d51f9d14a34ec50aae04298e0f142187427d3c832e"
	case params.GoerliNetworkID:
		return "0xbf7e331f7f7c1dd2e05159666b3bf8bc7a8a3a9eb1d518969eab529dd9b88c1a"
	}
	// Every other ID must break the test.
	panic(fmt.Sprintf("invalid network id: %d", id))
}

// GetRemoteURL returns the url associated with a given network id.
func GetRemoteURL() (string, error) {
	return GetRemoteURLFromNetworkID(GetNetworkID())
}

// GetHeadHash returns the hash associated with a given network id.
func GetHeadHash() string {
	return GetHeadHashFromNetworkID(GetNetworkID())
}

// GetNetworkID returns appropriate network id for test based on
// default or provided -network flag.
func GetNetworkID() int {
	switch strings.ToLower(*networkSelected) {
	case fmt.Sprintf("%d", params.MainNetworkID), "mainnet":
		return params.MainNetworkID
	case fmt.Sprintf("%d", params.StatusChainNetworkID), "statuschain":
		return params.StatusChainNetworkID
	case fmt.Sprintf("%d", params.GoerliNetworkID), "goerli":
		return params.GoerliNetworkID
	}
	// Every other selected network must break the test.
	panic(fmt.Sprintf("invalid selected network: %q", *networkSelected))
}

// CheckTestSkipForNetworks checks if network for test is one of the
// prohibited ones and skips the test in this case.
func CheckTestSkipForNetworks(t *testing.T, networks ...int) {
	id := GetNetworkID()
	for _, network := range networks {
		if network == id {
			t.Skipf("skipping test for network %d", network)
		}
	}
}

// GetAccount1PKFile returns the filename for Account1 keystore based
// on the current network. This allows running the e2e tests on the
// private network w/o access to the ACCOUNT_PASSWORD env variable
func GetAccount1PKFile() string {
	if GetNetworkID() == params.StatusChainNetworkID {
		return "test-account1-status-chain.pk"
	}
	return "test-account1.pk"
}

// GetAccount2PKFile returns the filename for Account2 keystore based
// on the current network. This allows running the e2e tests on the
// private network w/o access to the ACCOUNT_PASSWORD env variable
func GetAccount2PKFile() string {
	if GetNetworkID() == params.StatusChainNetworkID {
		return "test-account2-status-chain.pk"
	}
	return "test-account2.pk"
}

// WaitClosed used to wait on a channel in tests
func WaitClosed(c <-chan struct{}, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-c:
		return nil
	case <-timer.C:
		return ErrTimeout
	}
}

// MakeTestNodeConfig defines a function to return a params.NodeConfig
// where specific network addresses are assigned based on provided network id.
func MakeTestNodeConfig(networkID int) (*params.NodeConfig, error) {
	testDir := filepath.Join(TestDataDir, TestNetworkNames[networkID])

	if runtime.GOOS == "windows" {
		testDir = filepath.ToSlash(testDir)
	}

	// run tests with "INFO" log level only
	// when `go test` invoked with `-v` flag
	errorLevel := "ERROR"
	if testing.Verbose() {
		errorLevel = "INFO"
	}

	configJSON := `{
		"Name": "test",
		"NetworkId": ` + strconv.Itoa(networkID) + `,
		"DataDir": "` + testDir + `",
		"KeyStoreDir": "` + path.Join(testDir, "keystore") + `",
		"KeycardPairingDataFile": "` + path.Join(testDir, "keycard/pairings.json") + `",
		"HTTPPort": ` + strconv.Itoa(TestConfig.Node.HTTPPort) + `,
		"WSPort": ` + strconv.Itoa(TestConfig.Node.WSPort) + `,
		"LogLevel": "` + errorLevel + `",
		"NoDiscovery": true,
		"LightEthConfig": {
			"Enabled": true
		},
		"ShhextConfig": {
			"BackupDisabledDataDir": "` + testDir + `"
		}
	}`

	nodeConfig, err := params.NewConfigFromJSON(configJSON)
	if err != nil {
		return nil, err
	}

	return nodeConfig, nil
}

// MakeTestNodeConfigWithDataDir defines a function to return a params.NodeConfig
// where specific network addresses are assigned based on provided network id, and assigns
// a given name and data dir.
func MakeTestNodeConfigWithDataDir(name, dataDir string, networkID uint64) (*params.NodeConfig, error) {
	cfg, err := params.NewNodeConfig(dataDir, networkID)
	if err != nil {
		return nil, err
	}
	if name == "" {
		cfg.Name = "test"
	} else {
		cfg.Name = name
	}
	cfg.NoDiscovery = true
	cfg.LightEthConfig.Enabled = false
	if dataDir != "" {
		cfg.KeyStoreDir = path.Join(dataDir, "keystore")
	}

	// Only attempt to validate if a dataDir is specified, we only support in-memory DB for tests
	if dataDir != "" {
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

type account struct {
	WalletAddress string
	ChatAddress   string
	Password      string
}

// testConfig contains shared (among different test packages) parameters
type testConfig struct {
	Node struct {
		SyncSeconds time.Duration
		HTTPPort    int
		WSPort      int
	}
	Account1 account
	Account2 account
	Account3 account
}

const passphraseEnvName = "ACCOUNT_PASSWORD"

// loadTestConfig loads test configuration values from disk
func loadTestConfig() (*testConfig, error) {
	var config testConfig

	err := parseTestConfigFromFile("config/test-data.json", &config)
	if err != nil {
		return nil, err
	}

	if GetNetworkID() == params.StatusChainNetworkID {
		err := parseTestConfigFromFile("config/status-chain-accounts.json", &config)
		if err != nil {
			return nil, err
		}
	} else {
		err := parseTestConfigFromFile("public-chain-accounts.json", &config)
		if err != nil {
			return nil, err
		}

		pass, ok := os.LookupEnv(passphraseEnvName)
		if !ok {
			err := fmt.Errorf("Missing %s environment variable", passphraseEnvName)
			return nil, err
		}

		config.Account1.Password = pass
		config.Account2.Password = pass
	}

	return &config, nil
}

// ImportTestAccount imports keystore from static resources, see "static/keys" folder
func ImportTestAccount(keystoreDir, accountFile string) error {
	// make sure that keystore folder exists
	if _, err := os.Stat(keystoreDir); os.IsNotExist(err) {
		os.MkdirAll(keystoreDir, os.ModePerm) // nolint: errcheck, gas
	}

	var (
		data []byte
		err  error
	)

	// Allow to read keys from a custom dir.
	// Fallback to embedded data.
	if dir := os.Getenv("TEST_KEYS_DIR"); dir != "" {
		data, err = ioutil.ReadFile(filepath.Join(dir, accountFile))
	} else {
		data, err = static.Asset(filepath.Join("keys", accountFile))
	}

	if err != nil {
		return err
	}

	return createFile(data, filepath.Join(keystoreDir, accountFile))
}

func parseTestConfigFromFile(file string, config *testConfig) error {
	var (
		data []byte
		err  error
	)

	// Allow to read config from a custom dir.
	// Fallback to embedded data.
	if dir := os.Getenv("TEST_CONFIG_DIR"); dir != "" {
		data, err = ioutil.ReadFile(filepath.Join(dir, file))
	} else {
		data, err = t.Asset(file)
	}

	if err != nil {
		return err
	}

	return json.Unmarshal(data, &config)
}

func createFile(data []byte, dst string) error {
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, bytes.NewBuffer(data))
	return err
}

// Eventually will raise error if condition won't be met during the given timeout.
func Eventually(f func() error, timeout, period time.Duration) (err error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		select {
		case <-timer.C:
			return
		case <-ticker.C:
			err = f()
			if err == nil {
				return nil
			}
		}
	}
}
