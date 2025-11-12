package utils

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/static"
	"github.com/status-im/status-go/t"
)

var (
	// TestConfig defines the default config usable at package-level.
	TestConfig *testConfig

	// RootDir is the main application directory
	RootDir string
)

func Init() {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// set up logger
	err = logutils.OverrideRootLoggerWithConfig(logutils.LogSettings{
		Enabled: true,
		Level:   "INFO",
	})
	if err != nil {
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

	TestConfig, err = loadTestConfig()
	if err != nil {
		panic(err)
	}
}

// GetAccount1PKFile returns the filename for Account1 keystore based
// on the current network. This allows running the e2e tests on the
// private network w/o access to the ACCOUNT_PASSWORD env variable
func GetAccount1PKFile() string {
	return "test-account1-status-chain.pk"
}

// GetAccount2PKFile returns the filename for Account2 keystore based
// on the current network. This allows running the e2e tests on the
// private network w/o access to the ACCOUNT_PASSWORD env variable
func GetAccount2PKFile() string {
	return "test-account2-status-chain.pk"
}

type account struct {
	KeyUID        string
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

// loadTestConfig loads test configuration values from disk
func loadTestConfig() (*testConfig, error) {
	var config testConfig

	err := parseTestConfigFromFile("config/test-data.json", &config)
	if err != nil {
		return nil, err
	}

	err = parseTestConfigFromFile("config/status-chain-accounts.json", &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// ImportTestAccount imports keystore from static resources, see "static/keys" folder
func ImportTestAccount(keystoreDir, accountFile string) error {
	// make sure that keystore folder exists
	if _, err := os.Stat(keystoreDir); os.IsNotExist(err) {
		os.MkdirAll(keystoreDir, os.ModePerm) // nolint: errcheck
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
