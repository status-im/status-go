package integration

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
)

var (
	networkSelected = flag.Int("network", "statuschain", "-network=NETWORKID to select network used for tests")

	// TestConfig defines the default config usable at package-level.
	TestConfig *common.TestConfig

	// RootDir is the main application directory
	RootDir string

	// TestDataDir is data directory used for tests
	TestDataDir string

	// TestNetworkNames network ID to name mapping
	TestNetworkNames = map[int]string{
		params.MainNetworkID:        "Mainnet",
		params.RopstenNetworkID:     "Ropsten",
		params.RinkebyNetworkID:     "Rinkeby",
		params.StatusChainNetworkID: "StatusChain",
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

// EnsureNodeSync waits until node synchronzation is done to continue
// with tests afterwards. Returns an error in case of a timeout.
func EnsureNodeSync(nodeManager common.NodeManager) error {
	nc, err := nodeManager.NodeConfig()
	if err != nil {
		return errors.New("can't retrieve NodeConfig")
	}
	// Don't wait for any blockchain sync for the local private chain as blocks are never mined.
	if nc.NetworkID == params.StatusChainNetworkID {
		return nil
	}

	les, err := nodeManager.LightEthereumService()
	if err != nil {
		return err
	}
	if les == nil {
		return errors.New("LightEthereumService is nil")
	}

	timeouter := time.NewTimer(20 * time.Minute)
	defer timeouter.Stop()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeouter.C:
			return errors.New("timout during node synchronization")
		case <-ticker.C:
			downloader := les.Downloader()

			if downloader != nil {
				isSyncing := downloader.Synchronising()
				progress := downloader.Progress()

				if !isSyncing && progress.HighestBlock > 0 && progress.CurrentBlock >= progress.HighestBlock {
					return nil
				}
			}

		}
	}
}

// GetNetworkID returns appropriate network id for test based on
// default or provided -network flag.
func GetNetworkID() int {
	switch strings.ToLower(*networkSelected) {
	case params.MainNetworkID:
		return params.MainNetworkID
	case params.RinkebyNetworkID:
		return params.RinkebyNetworkID
	case params.RopstenNetworkID:
		return params.RopstenNetworkID
	case params.StatusChainNetworkID:
		return params.StatusChainNetworkID
	}

	return params.StatusChainNetworkID
}
