package integration

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
)

var (
	// TestConfig defines the default config usable at package-level.
	TestConfig *common.TestConfig

	// RootDir is the main application directory
	RootDir string

	// TestDataDir is data directory used for tests
	TestDataDir string

	// TestNetworkNames network ID to name mapping
	TestNetworkNames = map[int]string{
		params.MainNetworkID:    "Mainnet",
		params.RopstenNetworkID: "Ropsten",
		params.RinkebyNetworkID: "Rinkeby",
	}
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

	TestConfig, err = common.LoadTestConfig()
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
	io.Copy(buf, f)
	f.Close()

	return string(buf.Bytes())
}
